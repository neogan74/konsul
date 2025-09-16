package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

func main() {
	app := fiber.New()
	kv := store.NewKVStore()
	svcStore := store.NewServiceStore()

	app.Get("/kv/:key", func(c *fiber.Ctx) error {
		key := c.Params("key")
		value, ok := kv.Get(key)
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "key not found"})
		}
		return c.JSON(fiber.Map{"key": key, "value": value})
	})

	app.Put("/kv/:key", func(c *fiber.Ctx) error {
		key := c.Params("key")
		body := struct {
			Value string `json:"value"`
		}{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}
		kv.Set(key, body.Value)
		return c.JSON(fiber.Map{"message": "key set", "key": key})
	})

	app.Delete("/kv/:key", func(c *fiber.Ctx) error {
		key := c.Params("key")
		kv.Delete(key)
		return c.JSON(fiber.Map{"message": "key deleted", "key": key})
	})

	// Service registration
	app.Put("/register", func(c *fiber.Ctx) error {
		var svc store.Service
		if err := c.BodyParser(&svc); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}
		if svc.Name == "" || svc.Address == "" || svc.Port == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing fields"})
		}
		svcStore.Register(svc)
		return c.JSON(fiber.Map{"message": "service registered", "service": svc})
	})

	// List all services
	app.Get("/services/", func(c *fiber.Ctx) error {
		return c.JSON(svcStore.List())
	})

	// Get service by name
	app.Get("/services/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		svc, ok := svcStore.Get(name)
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "service not found"})
		}
		return c.JSON(svc)
	})

	// Deregister service
	app.Delete("/deregister/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		svcStore.Deregister(name)
		return c.JSON(fiber.Map{"message": "service deregistered", "name": name})
	})

	// Heartbeat endpoint
	app.Put("/heartbeat/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		if svcStore.Heartbeat(name) {
			return c.JSON(fiber.Map{"message": "heartbeat updated", "service": name})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "service not found"})
	})

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
