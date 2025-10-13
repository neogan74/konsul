package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/template"
)

const version = "0.1.0"

func main() {
	// Command line flags
	var (
		configFile  = flag.String("config", "", "Path to configuration file (HCL or JSON)")
		once        = flag.Bool("once", false, "Run once and exit (don't watch for changes)")
		dryRun      = flag.Bool("dry", false, "Dry run mode (render but don't write files)")
		konsulAddr  = flag.String("konsul", "http://localhost:8500", "Konsul server address")
		templateSrc = flag.String("template", "", "Single template source file")
		dest        = flag.String("dest", "", "Single template destination file")
		showVersion = flag.Bool("version", false, "Show version and exit")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("konsul-template version %s\n", version)
		os.Exit(0)
	}

	// Setup logger
	log := logger.GetDefault()

	// For now, we'll implement a simple mode that works with command-line args
	// Later we can add HCL/JSON config file support
	if *templateSrc == "" && *configFile == "" {
		fmt.Fprintln(os.Stderr, "Error: either -template or -config must be specified")
		flag.Usage()
		os.Exit(1)
	}

	// Create a simple client that connects to Konsul
	client := NewKonsulClient(*konsulAddr, log)

	// Build configuration
	config := template.Config{
		Once:       *once,
		DryRun:     *dryRun,
		KonsulAddr: *konsulAddr,
	}

	// Add template from command line if specified
	if *templateSrc != "" {
		if *dest == "" {
			fmt.Fprintln(os.Stderr, "Error: -dest must be specified when using -template")
			os.Exit(1)
		}

		config.Templates = []template.TemplateConfig{
			{
				Source:      *templateSrc,
				Destination: *dest,
				Perms:       0644,
			},
		}
	}

	// Create template engine
	engine := template.New(config, client.KVStore(), client.ServiceStore(), log)

	if *once {
		// Run once mode
		log.Info("Running in once mode")
		if err := engine.RunOnce(); err != nil {
			log.Error("Failed to run templates", logger.Error(err))
			os.Exit(1)
		}
		log.Info("Templates rendered successfully")
		return
	}

	// Watch mode
	log.Info("Starting watch mode (press Ctrl+C to stop)")

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start engine in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- engine.Run(ctx)
	}()

	// Wait for signal or error
	select {
	case <-sigCh:
		log.Info("Received interrupt signal, shutting down...")
		cancel()
		// Give engine time to clean up
		time.Sleep(1 * time.Second)
	case err := <-errCh:
		if err != nil {
			log.Error("Engine error", logger.Error(err))
			os.Exit(1)
		}
	}

	log.Info("Shutdown complete")
}
