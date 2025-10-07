package main

import (
	"context"
	"log"

	"github.com/neogan74/konsul/internal/app"
	"github.com/neogan74/konsul/internal/config"
)

const version = "0.1.0"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	builder := app.NewBuilder(cfg, version)

	konsulApp, err := builder.Build(context.Background())
	if err != nil {
		log.Fatalf("Failed to build application: %v", err)
	}

	if err := konsulApp.Run(context.Background()); err != nil {
		log.Fatalf("Application exited with error: %v", err)
	}
}
