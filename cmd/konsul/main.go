package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/handlers"
	"github.com/neogan74/konsul/internal/store"
)

func main() {
	app := fiber.New()

	// Initialize stores
	kv := store.NewKVStore()
	svcStore := store.NewServiceStore()

	// Initialize handlers
	kvHandler := handlers.NewKVHandler(kv)
	serviceHandler := handlers.NewServiceHandler(svcStore)

	// KV endpoints
	app.Get("/kv/:key", kvHandler.Get)
	app.Put("/kv/:key", kvHandler.Set)
	app.Delete("/kv/:key", kvHandler.Delete)

	// Service discovery endpoints
	app.Put("/register", serviceHandler.Register)
	app.Get("/services/", serviceHandler.List)
	app.Get("/services/:name", serviceHandler.Get)
	app.Delete("/deregister/:name", serviceHandler.Deregister)
	app.Put("/heartbeat/:name", serviceHandler.Heartbeat)

	// Start background cleanup process
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count := svcStore.CleanupExpired()
				if count > 0 {
					log.Printf("Cleaned up %d expired services", count)
				}
			}
		}
	}()

	log.Println("Server started at http://localhost:8888")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		if err := app.Listen(":8888"); err != nil {
			log.Fatalf("Listen error: %v", err)
		}
	}()
	<-quit
	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited gracefully")
}
