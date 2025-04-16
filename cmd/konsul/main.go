package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
	"log"
)

func main() {
	app := fiber.New()
	kv := store.NewKVStore()
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
	log.Println("Server started at http://localhost:8888")
	log.Fatal(app.Listen(":8888"))
}
