package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// CLI represents the command-line interface with dependencies
type CLI struct {
	Output io.Writer
	Error  io.Writer
	Exit   func(int)
}

// NewCLI creates a new CLI instance with default dependencies
func NewCLI() *CLI {
	return &CLI{
		Output: os.Stdout,
		Error:  os.Stderr,
		Exit:   os.Exit,
	}
}

// GlobalConfig holds common configuration for all commands
type GlobalConfig struct {
	ServerURL     string
	TLSSkipVerify bool
	TLSCACert     string
	TLSClientCert string
	TLSClientKey  string
}

// ParseGlobalFlags parses common flags and returns GlobalConfig and remaining args
func (cli *CLI) ParseGlobalFlags(args []string, commandName string) (*GlobalConfig, []string, error) {
	config := &GlobalConfig{}

	flagSet := flag.NewFlagSet(commandName, flag.ContinueOnError)
	flagSet.SetOutput(cli.Error)
	flagSet.StringVar(&config.ServerURL, "server", "http://localhost:8888", "Konsul server URL")
	flagSet.BoolVar(&config.TLSSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	flagSet.StringVar(&config.TLSCACert, "ca-cert", "", "Path to CA certificate file")
	flagSet.StringVar(&config.TLSClientCert, "client-cert", "", "Path to client certificate file")
	flagSet.StringVar(&config.TLSClientKey, "client-key", "", "Path to client key file")

	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		return nil, nil, flag.ErrHelp
	}

	err := flagSet.Parse(args)
	if err != nil {
		return nil, nil, err
	}

	return config, flagSet.Args(), nil
}

// CreateClient creates a KonsulClient from GlobalConfig
func (cli *CLI) CreateClient(config *GlobalConfig) *KonsulClient {
	tlsConfig := &TLSConfig{
		Enabled:        strings.HasPrefix(config.ServerURL, "https://"),
		SkipVerify:     config.TLSSkipVerify,
		CACertFile:     config.TLSCACert,
		ClientCertFile: config.TLSClientCert,
		ClientKeyFile:  config.TLSClientKey,
	}

	return NewKonsulClientWithTLS(config.ServerURL, tlsConfig)
}

// Printf writes formatted output to the output writer
func (cli *CLI) Printf(format string, args ...interface{}) {
	fmt.Fprintf(cli.Output, format, args...)
}

// Println writes a line to the output writer
func (cli *CLI) Println(args ...interface{}) {
	fmt.Fprintln(cli.Output, args...)
}

// Errorf writes formatted error to the error writer
func (cli *CLI) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(cli.Error, format, args...)
}

// Errorln writes an error line to the error writer
func (cli *CLI) Errorln(args ...interface{}) {
	fmt.Fprintln(cli.Error, args...)
}

// ExitError prints an error message and exits
func (cli *CLI) ExitError(format string, args ...interface{}) {
	cli.Errorf(format, args...)
	cli.Exit(1)
}

// HandleError checks if error exists, prints it and exits
func (cli *CLI) HandleError(err error, context string) {
	if err != nil {
		cli.ExitError("Error %s: %v\n", context, err)
	}
}

// ValidateArgs checks if the number of arguments is within the expected range
func (cli *CLI) ValidateArgs(args []string, min, max int, usage string) {
	if len(args) < min || len(args) > max {
		cli.Errorln(usage)
		cli.Exit(1)
	}
}

// ValidateExactArgs checks if exactly n arguments are provided
func (cli *CLI) ValidateExactArgs(args []string, n int, usage string) {
	cli.ValidateArgs(args, n, n, usage)
}

// ValidateMinArgs checks if at least n arguments are provided
func (cli *CLI) ValidateMinArgs(args []string, n int, usage string) {
	if len(args) < n {
		cli.Errorln(usage)
		cli.Exit(1)
	}
}

// ValidateMaxArgs checks if at most n arguments are provided
func (cli *CLI) ValidateMaxArgs(args []string, n int, usage string) {
	if len(args) > n {
		cli.Errorln(usage)
		cli.Exit(1)
	}
}
