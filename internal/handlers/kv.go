package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/middleware"
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
	log := middleware.GetLogger(c)

	log.Debug("Getting key", logger.String("key", key))

	value, ok := h.store.Get(key)
	if !ok {
		log.Warn("Key not found", logger.String("key", key))
		return middleware.NotFound(c, "Key not found")
	}

	log.Info("Key retrieved successfully", logger.String("key", key))
	return c.JSON(fiber.Map{"key": key, "value": value})
}

func (h *KVHandler) Set(c *fiber.Ctx) error {
	key := c.Params("key")
	log := middleware.GetLogger(c)

	body := struct {
		Value string `json:"value"`
	}{}

	if err := c.BodyParser(&body); err != nil {
		log.Error("Failed to parse request body",
			logger.String("key", key),
			logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	log.Debug("Setting key",
		logger.String("key", key),
		logger.String("value_length", fmt.Sprintf("%d", len(body.Value))))

	h.store.Set(key, body.Value)

	log.Info("Key set successfully", logger.String("key", key))
	return c.JSON(fiber.Map{"message": "key set", "key": key})
}

func (h *KVHandler) Delete(c *fiber.Ctx) error {
	key := c.Params("key")
	log := middleware.GetLogger(c)

	log.Debug("Deleting key", logger.String("key", key))

	h.store.Delete(key)

	log.Info("Key deleted successfully", logger.String("key", key))
	return c.JSON(fiber.Map{"message": "key deleted", "key": key})
}