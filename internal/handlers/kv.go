package handlers

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	hashiraft "github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/middleware"
	konsulraft "github.com/neogan74/konsul/internal/raft"
	"github.com/neogan74/konsul/internal/store"
)

type KVHandler struct {
	store    *store.KVStore
	raftNode *konsulraft.Node
}

func NewKVHandler(kvStore *store.KVStore, raftNode *konsulraft.Node) *KVHandler {
	return &KVHandler{store: kvStore, raftNode: raftNode}
}

func (h *KVHandler) Get(c *fiber.Ctx) error {
	key := c.Params("key")
	log := middleware.GetLogger(c)

	log.Debug("Getting key", logger.String("key", key))

	// Check if client wants full entry with indices
	includeMetadata := c.Query("metadata", "false") == "true"

	if includeMetadata {
		entry, ok := h.store.GetEntry(key)
		if !ok {
			log.Warn("Key not found", logger.String("key", key))
			metrics.KVOperationsTotal.WithLabelValues("get", "not_found").Inc()
			return middleware.NotFound(c, "Key not found")
		}
		log.Info("Key retrieved successfully with metadata", logger.String("key", key))
		metrics.KVOperationsTotal.WithLabelValues("get", "success").Inc()
		return c.JSON(fiber.Map{
			"key":          key,
			"value":        entry.Value,
			"modify_index": entry.ModifyIndex,
			"create_index": entry.CreateIndex,
			"flags":        entry.Flags,
		})
	}

	value, ok := h.store.Get(key)
	if !ok {
		log.Warn("Key not found", logger.String("key", key))
		metrics.KVOperationsTotal.WithLabelValues("get", "not_found").Inc()
		return middleware.NotFound(c, "Key not found")
	}

	log.Info("Key retrieved successfully", logger.String("key", key))
	metrics.KVOperationsTotal.WithLabelValues("get", "success").Inc()
	return c.JSON(fiber.Map{"key": key, "value": value})
}

func (h *KVHandler) Set(c *fiber.Ctx) error {
	key := c.Params("key")
	log := middleware.GetLogger(c)

	body := struct {
		Value string  `json:"value"`
		CAS   *uint64 `json:"cas,omitempty"` // Optional CAS index
		Flags uint64  `json:"flags,omitempty"`
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

	// Use CAS if provided
	if body.CAS != nil {
		var newIndex uint64
		var err error
		if h.raftNode != nil {
			entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryKVSetCAS, konsulraft.KVSetCASPayload{
				Key:           key,
				Value:         body.Value,
				ExpectedIndex: *body.CAS,
			})
			if marshalErr != nil {
				log.Error("Failed to build raft log entry", logger.Error(marshalErr))
				return middleware.InternalError(c, "Failed to set key")
			}
			resp, applyErr := h.raftNode.ApplyEntry(entry, 5*time.Second)
			if applyErr != nil {
				if errors.Is(applyErr, hashiraft.ErrNotLeader) {
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":  "not leader",
						"leader": h.raftNode.Leader(),
					})
				}
				err = applyErr
			} else if index, ok := resp.(uint64); ok {
				newIndex = index
			}
		} else {
			newIndex, err = h.store.SetCAS(key, body.Value, *body.CAS)
		}
		if err != nil {
			if store.IsCASConflict(err) {
				log.Warn("CAS conflict", logger.String("key", key), logger.Error(err))
				metrics.KVOperationsTotal.WithLabelValues("set", "cas_conflict").Inc()
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error":   "CAS conflict",
					"message": err.Error(),
				})
			}
			if store.IsNotFound(err) {
				log.Warn("Key not found for CAS update", logger.String("key", key))
				metrics.KVOperationsTotal.WithLabelValues("set", "not_found").Inc()
				return middleware.NotFound(c, "Key not found")
			}
			log.Error("CAS operation failed", logger.String("key", key), logger.Error(err))
			metrics.KVOperationsTotal.WithLabelValues("set", "error").Inc()
			return middleware.InternalError(c, "Failed to set key")
		}
		log.Info("Key set successfully with CAS", logger.String("key", key))
		metrics.KVOperationsTotal.WithLabelValues("set", "success").Inc()
		metrics.KVStoreSize.Set(float64(len(h.store.List())))
		return c.JSON(fiber.Map{
			"message":      "key set",
			"key":          key,
			"modify_index": newIndex,
		})
	}

	// Regular set (with or without flags)
	if h.raftNode != nil {
		entryType := konsulraft.EntryKVSet
		var payload any = konsulraft.KVSetPayload{Key: key, Value: body.Value}
		if body.Flags > 0 {
			entryType = konsulraft.EntryKVSetWithFlags
			payload = konsulraft.KVSetWithFlagsPayload{Key: key, Value: body.Value, Flags: body.Flags}
		}
		entry, marshalErr := konsulraft.NewLogEntry(entryType, payload)
		if marshalErr != nil {
			log.Error("Failed to build raft log entry", logger.Error(marshalErr))
			return middleware.InternalError(c, "Failed to set key")
		}
		if _, err := h.raftNode.ApplyEntry(entry, 5*time.Second); err != nil {
			if errors.Is(err, hashiraft.ErrNotLeader) {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":  "not leader",
					"leader": h.raftNode.Leader(),
				})
			}
			return middleware.InternalError(c, "Failed to set key")
		}
	} else {
		if body.Flags > 0 {
			h.store.SetWithFlags(key, body.Value, body.Flags)
		} else {
			h.store.Set(key, body.Value)
		}
	}

	log.Info("Key set successfully", logger.String("key", key))
	metrics.KVOperationsTotal.WithLabelValues("set", "success").Inc()
	metrics.KVStoreSize.Set(float64(len(h.store.List())))

	// Return the new index
	entry, _ := h.store.GetEntry(key)
	return c.JSON(fiber.Map{
		"message":      "key set",
		"key":          key,
		"modify_index": entry.ModifyIndex,
	})
}

func (h *KVHandler) Delete(c *fiber.Ctx) error {
	key := c.Params("key")
	log := middleware.GetLogger(c)

	log.Debug("Deleting key", logger.String("key", key))

	// Check if CAS is requested via query parameter
	casParam := c.Query("cas")
	if casParam != "" {
		var expectedIndex uint64
		if _, err := fmt.Sscanf(casParam, "%d", &expectedIndex); err != nil {
			log.Error("Invalid CAS parameter", logger.String("cas", casParam), logger.Error(err))
			return middleware.BadRequest(c, "Invalid CAS parameter")
		}

		var err error
		if h.raftNode != nil {
			entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryKVDeleteCAS, konsulraft.KVDeleteCASPayload{
				Key:           key,
				ExpectedIndex: expectedIndex,
			})
			if marshalErr != nil {
				log.Error("Failed to build raft log entry", logger.Error(marshalErr))
				return middleware.InternalError(c, "Failed to delete key")
			}
			if _, applyErr := h.raftNode.ApplyEntry(entry, 5*time.Second); applyErr != nil {
				if errors.Is(applyErr, hashiraft.ErrNotLeader) {
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":  "not leader",
						"leader": h.raftNode.Leader(),
					})
				}
				err = applyErr
			}
		} else {
			err = h.store.DeleteCAS(key, expectedIndex)
		}
		if err != nil {
			if store.IsCASConflict(err) {
				log.Warn("CAS conflict on delete", logger.String("key", key), logger.Error(err))
				metrics.KVOperationsTotal.WithLabelValues("delete", "cas_conflict").Inc()
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error":   "CAS conflict",
					"message": err.Error(),
				})
			}
			if store.IsNotFound(err) {
				log.Warn("Key not found for CAS delete", logger.String("key", key))
				metrics.KVOperationsTotal.WithLabelValues("delete", "not_found").Inc()
				return middleware.NotFound(c, "Key not found")
			}
			log.Error("CAS delete failed", logger.String("key", key), logger.Error(err))
			metrics.KVOperationsTotal.WithLabelValues("delete", "error").Inc()
			return middleware.InternalError(c, "Failed to delete key")
		}

		log.Info("Key deleted successfully with CAS", logger.String("key", key))
		metrics.KVOperationsTotal.WithLabelValues("delete", "success").Inc()
		metrics.KVStoreSize.Set(float64(len(h.store.List())))
		return c.JSON(fiber.Map{"message": "key deleted", "key": key})
	}

	if h.raftNode != nil {
		entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryKVDelete, konsulraft.KVDeletePayload{
			Key: key,
		})
		if marshalErr != nil {
			log.Error("Failed to build raft log entry", logger.Error(marshalErr))
			return middleware.InternalError(c, "Failed to delete key")
		}
		if _, err := h.raftNode.ApplyEntry(entry, 5*time.Second); err != nil {
			if errors.Is(err, hashiraft.ErrNotLeader) {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":  "not leader",
					"leader": h.raftNode.Leader(),
				})
			}
			return middleware.InternalError(c, "Failed to delete key")
		}
	} else {
		h.store.Delete(key)
	}

	log.Info("Key deleted successfully", logger.String("key", key))
	metrics.KVOperationsTotal.WithLabelValues("delete", "success").Inc()
	metrics.KVStoreSize.Set(float64(len(h.store.List())))
	return c.JSON(fiber.Map{"message": "key deleted", "key": key})
}

func (h *KVHandler) List(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	log.Debug("Listing all keys")

	keys := h.store.List()

	log.Info("Keys listed successfully", logger.Int("count", len(keys)))
	metrics.KVOperationsTotal.WithLabelValues("list", "success").Inc()
	return c.JSON(keys)
}
