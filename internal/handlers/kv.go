package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

type KVHandler struct {
	store *store.KVStore
}

func NewKVHandler(kvStore *store.KVStore) *KVHandler {
	return &KVHandler{store: kvStore}
}

func (h *KVHandler) Get(c *fiber.Ctx) error {
	key := c.Params("key")
	value, ok := h.store.Get(key)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "key not found"})
	}
	return c.JSON(fiber.Map{"key": key, "value": value})
}

func (h *KVHandler) Set(c *fiber.Ctx) error {
	key := c.Params("key")
	body := struct {
		Value string `json:"value"`
	}{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	h.store.Set(key, body.Value)
	return c.JSON(fiber.Map{"message": "key set", "key": key})
}

func (h *KVHandler) Delete(c *fiber.Ctx) error {
	key := c.Params("key")
	h.store.Delete(key)
	return c.JSON(fiber.Map{"message": "key deleted", "key": key})
}