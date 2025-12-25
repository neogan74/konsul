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

// BatchHandler handles batch operations for KV and Service stores
type BatchHandler struct {
	kvStore      *store.KVStore
	serviceStore *store.ServiceStore
	raftNode     *konsulraft.Node
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(kvStore *store.KVStore, serviceStore *store.ServiceStore, raftNode *konsulraft.Node) *BatchHandler {
	return &BatchHandler{
		kvStore:      kvStore,
		serviceStore: serviceStore,
		raftNode:     raftNode,
	}
}

// BatchKVGetRequest represents a request to get multiple keys
type BatchKVGetRequest struct {
	Keys []string `json:"keys"`
}

// BatchKVGetResponse represents the response for batch get
type BatchKVGetResponse struct {
	Found    map[string]string `json:"found"`
	NotFound []string          `json:"not_found"`
}

// BatchKVSetRequest represents a request to set multiple key-value pairs
type BatchKVSetRequest struct {
	Items map[string]string `json:"items"`
}

// BatchKVSetResponse represents the response for batch set
type BatchKVSetResponse struct {
	Message string   `json:"message"`
	Keys    []string `json:"keys"`
	Count   int      `json:"count"`
}

// BatchKVDeleteRequest represents a request to delete multiple keys
type BatchKVDeleteRequest struct {
	Keys []string `json:"keys"`
}

// BatchKVDeleteResponse represents the response for batch delete
type BatchKVDeleteResponse struct {
	Message string   `json:"message"`
	Keys    []string `json:"keys"`
	Count   int      `json:"count"`
}

// BatchKVSetCASRequest represents a request to set multiple keys with CAS
type BatchKVSetCASRequest struct {
	Items           map[string]string `json:"items"`
	ExpectedIndices map[string]uint64 `json:"expected_indices"`
}

// BatchKVSetCASResponse represents the response for batch CAS set
type BatchKVSetCASResponse struct {
	Message    string            `json:"message"`
	NewIndices map[string]uint64 `json:"new_indices"`
	Count      int               `json:"count"`
}

// BatchKVDeleteCASRequest represents a request to delete multiple keys with CAS
type BatchKVDeleteCASRequest struct {
	Keys            []string          `json:"keys"`
	ExpectedIndices map[string]uint64 `json:"expected_indices"`
}

// BatchKVDeleteCASResponse represents the response for batch CAS delete
type BatchKVDeleteCASResponse struct {
	Message string   `json:"message"`
	Keys    []string `json:"keys"`
	Count   int      `json:"count"`
}

// BatchKVGet retrieves multiple keys at once
// POST /batch/kv/get
func (h *BatchHandler) BatchKVGet(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchKVGetRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch KV get request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Keys) == 0 {
		return middleware.BadRequest(c, "Keys array cannot be empty")
	}

	if len(req.Keys) > 1000 {
		return middleware.BadRequest(c, "Maximum 1000 keys per batch request")
	}

	log.Debug("Batch getting keys", logger.Int("count", len(req.Keys)))

	found, notFound := h.kvStore.BatchGet(req.Keys)

	log.Info("Batch get completed",
		logger.Int("found", len(found)),
		logger.Int("not_found", len(notFound)))
	metrics.KVOperationsTotal.WithLabelValues("batch_get", "success").Inc()

	return c.JSON(BatchKVGetResponse{
		Found:    found,
		NotFound: notFound,
	})
}

// BatchKVSet sets multiple key-value pairs at once
// POST /batch/kv/set
func (h *BatchHandler) BatchKVSet(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchKVSetRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch KV set request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Items) == 0 {
		return middleware.BadRequest(c, "Items map cannot be empty")
	}

	if len(req.Items) > 1000 {
		return middleware.BadRequest(c, "Maximum 1000 items per batch request")
	}

	log.Debug("Batch setting keys", logger.Int("count", len(req.Items)))

	if h.raftNode != nil {
		entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryKVBatchSet, konsulraft.KVBatchSetPayload{
			Items: req.Items,
		})
		if marshalErr != nil {
			log.Error("Failed to build raft log entry", logger.Error(marshalErr))
			return middleware.InternalError(c, "Failed to set keys")
		}
		if _, err := h.raftNode.ApplyEntry(entry, 10*time.Second); err != nil {
			if errors.Is(err, hashiraft.ErrNotLeader) {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":  "not leader",
					"leader": h.raftNode.Leader(),
				})
			}
			log.Error("Failed to batch set keys", logger.Error(err))
			return middleware.InternalError(c, "Failed to set keys")
		}
	} else if err := h.kvStore.BatchSet(req.Items); err != nil {
		log.Error("Failed to batch set keys", logger.Error(err))
		return middleware.InternalError(c, "Failed to set keys")
	}

	keys := make([]string, 0, len(req.Items))
	for key := range req.Items {
		keys = append(keys, key)
	}

	log.Info("Batch set completed", logger.Int("count", len(req.Items)))
	metrics.KVOperationsTotal.WithLabelValues("batch_set", "success").Inc()
	metrics.KVStoreSize.Set(float64(len(h.kvStore.List())))

	return c.JSON(BatchKVSetResponse{
		Message: fmt.Sprintf("Successfully set %d keys", len(req.Items)),
		Keys:    keys,
		Count:   len(req.Items),
	})
}

// BatchKVDelete deletes multiple keys at once
// POST /batch/kv/delete
func (h *BatchHandler) BatchKVDelete(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchKVDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch KV delete request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Keys) == 0 {
		return middleware.BadRequest(c, "Keys array cannot be empty")
	}

	if len(req.Keys) > 1000 {
		return middleware.BadRequest(c, "Maximum 1000 keys per batch request")
	}

	log.Debug("Batch deleting keys", logger.Int("count", len(req.Keys)))

	if h.raftNode != nil {
		entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryKVBatchDelete, konsulraft.KVBatchDeletePayload{
			Keys: req.Keys,
		})
		if marshalErr != nil {
			log.Error("Failed to build raft log entry", logger.Error(marshalErr))
			return middleware.InternalError(c, "Failed to delete keys")
		}
		if _, err := h.raftNode.ApplyEntry(entry, 10*time.Second); err != nil {
			if errors.Is(err, hashiraft.ErrNotLeader) {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":  "not leader",
					"leader": h.raftNode.Leader(),
				})
			}
			log.Error("Failed to batch delete keys", logger.Error(err))
			return middleware.InternalError(c, "Failed to delete keys")
		}
	} else if err := h.kvStore.BatchDelete(req.Keys); err != nil {
		log.Error("Failed to batch delete keys", logger.Error(err))
		return middleware.InternalError(c, "Failed to delete keys")
	}

	log.Info("Batch delete completed", logger.Int("count", len(req.Keys)))
	metrics.KVOperationsTotal.WithLabelValues("batch_delete", "success").Inc()
	metrics.KVStoreSize.Set(float64(len(h.kvStore.List())))

	return c.JSON(BatchKVDeleteResponse{
		Message: fmt.Sprintf("Successfully deleted %d keys", len(req.Keys)),
		Keys:    req.Keys,
		Count:   len(req.Keys),
	})
}

// BatchKVSetCAS sets multiple key-value pairs with CAS checks
// POST /batch/kv/set-cas
func (h *BatchHandler) BatchKVSetCAS(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchKVSetCASRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch KV set CAS request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Items) == 0 {
		return middleware.BadRequest(c, "Items map cannot be empty")
	}

	if len(req.Items) > 1000 {
		return middleware.BadRequest(c, "Maximum 1000 items per batch request")
	}

	// Validate that expected indices are provided for all items
	for key := range req.Items {
		if _, hasIndex := req.ExpectedIndices[key]; !hasIndex {
			return middleware.BadRequest(c, fmt.Sprintf("Missing expected index for key: %s", key))
		}
	}

	log.Debug("Batch setting keys with CAS", logger.Int("count", len(req.Items)))

	var newIndices map[string]uint64
	var err error
	if h.raftNode != nil {
		entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryKVBatchSetCAS, konsulraft.KVBatchSetCASPayload{
			Items:           req.Items,
			ExpectedIndices: req.ExpectedIndices,
		})
		if marshalErr != nil {
			log.Error("Failed to build raft log entry", logger.Error(marshalErr))
			return middleware.InternalError(c, "Failed to set keys")
		}
		resp, applyErr := h.raftNode.ApplyEntry(entry, 10*time.Second)
		if applyErr != nil {
			if errors.Is(applyErr, hashiraft.ErrNotLeader) {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":  "not leader",
					"leader": h.raftNode.Leader(),
				})
			}
			err = applyErr
		} else if cast, ok := resp.(map[string]uint64); ok {
			newIndices = cast
		}
	} else {
		newIndices, err = h.kvStore.BatchSetCAS(req.Items, req.ExpectedIndices)
	}
	if err != nil {
		if store.IsCASConflict(err) {
			log.Warn("CAS conflict in batch set", logger.Error(err))
			metrics.KVOperationsTotal.WithLabelValues("batch_set_cas", "cas_conflict").Inc()
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "CAS conflict",
				"message": err.Error(),
			})
		}
		if store.IsNotFound(err) {
			log.Warn("Key not found in batch CAS set", logger.Error(err))
			metrics.KVOperationsTotal.WithLabelValues("batch_set_cas", "not_found").Inc()
			return middleware.NotFound(c, err.Error())
		}
		log.Error("Failed to batch set keys with CAS", logger.Error(err))
		metrics.KVOperationsTotal.WithLabelValues("batch_set_cas", "error").Inc()
		return middleware.InternalError(c, "Failed to set keys")
	}

	log.Info("Batch set with CAS completed", logger.Int("count", len(req.Items)))
	metrics.KVOperationsTotal.WithLabelValues("batch_set_cas", "success").Inc()
	metrics.KVStoreSize.Set(float64(len(h.kvStore.List())))

	return c.JSON(BatchKVSetCASResponse{
		Message:    fmt.Sprintf("Successfully set %d keys with CAS", len(req.Items)),
		NewIndices: newIndices,
		Count:      len(req.Items),
	})
}

// BatchKVDeleteCAS deletes multiple keys with CAS checks
// POST /batch/kv/delete-cas
func (h *BatchHandler) BatchKVDeleteCAS(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchKVDeleteCASRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch KV delete CAS request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Keys) == 0 {
		return middleware.BadRequest(c, "Keys array cannot be empty")
	}

	if len(req.Keys) > 1000 {
		return middleware.BadRequest(c, "Maximum 1000 keys per batch request")
	}

	// Validate that expected indices are provided for all keys
	for _, key := range req.Keys {
		if _, hasIndex := req.ExpectedIndices[key]; !hasIndex {
			return middleware.BadRequest(c, fmt.Sprintf("Missing expected index for key: %s", key))
		}
	}

	log.Debug("Batch deleting keys with CAS", logger.Int("count", len(req.Keys)))

	var err error
	if h.raftNode != nil {
		entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryKVBatchDeleteCAS, konsulraft.KVBatchDeleteCASPayload{
			Keys:            req.Keys,
			ExpectedIndices: req.ExpectedIndices,
		})
		if marshalErr != nil {
			log.Error("Failed to build raft log entry", logger.Error(marshalErr))
			return middleware.InternalError(c, "Failed to delete keys")
		}
		if _, applyErr := h.raftNode.ApplyEntry(entry, 10*time.Second); applyErr != nil {
			if errors.Is(applyErr, hashiraft.ErrNotLeader) {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":  "not leader",
					"leader": h.raftNode.Leader(),
				})
			}
			err = applyErr
		}
	} else {
		err = h.kvStore.BatchDeleteCAS(req.Keys, req.ExpectedIndices)
	}
	if err != nil {
		if store.IsCASConflict(err) {
			log.Warn("CAS conflict in batch delete", logger.Error(err))
			metrics.KVOperationsTotal.WithLabelValues("batch_delete_cas", "cas_conflict").Inc()
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "CAS conflict",
				"message": err.Error(),
			})
		}
		if store.IsNotFound(err) {
			log.Warn("Key not found in batch CAS delete", logger.Error(err))
			metrics.KVOperationsTotal.WithLabelValues("batch_delete_cas", "not_found").Inc()
			return middleware.NotFound(c, err.Error())
		}
		log.Error("Failed to batch delete keys with CAS", logger.Error(err))
		metrics.KVOperationsTotal.WithLabelValues("batch_delete_cas", "error").Inc()
		return middleware.InternalError(c, "Failed to delete keys")
	}

	log.Info("Batch delete with CAS completed", logger.Int("count", len(req.Keys)))
	metrics.KVOperationsTotal.WithLabelValues("batch_delete_cas", "success").Inc()
	metrics.KVStoreSize.Set(float64(len(h.kvStore.List())))

	return c.JSON(BatchKVDeleteCASResponse{
		Message: fmt.Sprintf("Successfully deleted %d keys with CAS", len(req.Keys)),
		Keys:    req.Keys,
		Count:   len(req.Keys),
	})
}

// BatchServiceRegisterRequest represents a request to register multiple services
type BatchServiceRegisterRequest struct {
	Services []store.Service `json:"services"`
}

// BatchServiceRegisterResponse represents the response for batch register
type BatchServiceRegisterResponse struct {
	Message    string   `json:"message"`
	Registered []string `json:"registered"`
	Failed     []string `json:"failed,omitempty"`
	Count      int      `json:"count"`
}

// BatchServiceDeregisterRequest represents a request to deregister multiple services
type BatchServiceDeregisterRequest struct {
	Names []string `json:"names"`
}

// BatchServiceDeregisterResponse represents the response for batch deregister
type BatchServiceDeregisterResponse struct {
	Message      string   `json:"message"`
	Deregistered []string `json:"deregistered"`
	Count        int      `json:"count"`
}

// BatchServiceRegister registers multiple services at once
// POST /batch/services/register
func (h *BatchHandler) BatchServiceRegister(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchServiceRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch service register request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Services) == 0 {
		return middleware.BadRequest(c, "Services array cannot be empty")
	}

	if len(req.Services) > 100 {
		return middleware.BadRequest(c, "Maximum 100 services per batch request")
	}

	log.Debug("Batch registering services", logger.Int("count", len(req.Services)))

	registered := make([]string, 0, len(req.Services))
	failed := make([]string, 0)

	for _, svc := range req.Services {
		// Validate service
		if svc.Name == "" {
			failed = append(failed, "unnamed-service")
			continue
		}
		if svc.Address == "" {
			failed = append(failed, svc.Name)
			continue
		}
		if svc.Port <= 0 || svc.Port > 65535 {
			failed = append(failed, svc.Name)
			continue
		}

		// Register the service
		if h.raftNode != nil {
			entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryServiceRegister, konsulraft.ServiceRegisterPayload{
				Service: svc,
			})
			if marshalErr != nil {
				log.Error("Failed to build raft log entry", logger.Error(marshalErr))
				failed = append(failed, svc.Name)
				continue
			}
			if _, err := h.raftNode.ApplyEntry(entry, 10*time.Second); err != nil {
				if errors.Is(err, hashiraft.ErrNotLeader) {
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":  "not leader",
						"leader": h.raftNode.Leader(),
					})
				}
				log.Error("Failed to register service in batch",
					logger.String("service", svc.Name),
					logger.Error(err))
				failed = append(failed, svc.Name)
				continue
			}
		} else if err := h.serviceStore.Register(svc); err != nil {
			log.Error("Failed to register service in batch",
				logger.String("service", svc.Name),
				logger.Error(err))
			failed = append(failed, svc.Name)
			continue
		}

		registered = append(registered, svc.Name)
	}

	log.Info("Batch service registration completed",
		logger.Int("registered", len(registered)),
		logger.Int("failed", len(failed)))
	metrics.ServiceOperationsTotal.WithLabelValues("batch_register", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.serviceStore.List())))

	response := BatchServiceRegisterResponse{
		Message:    fmt.Sprintf("Registered %d services", len(registered)),
		Registered: registered,
		Count:      len(registered),
	}

	if len(failed) > 0 {
		response.Failed = failed
		response.Message = fmt.Sprintf("Registered %d services, %d failed", len(registered), len(failed))
	}

	return c.JSON(response)
}

// BatchServiceDeregister deregisters multiple services at once
// POST /batch/services/deregister
func (h *BatchHandler) BatchServiceDeregister(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchServiceDeregisterRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch service deregister request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Names) == 0 {
		return middleware.BadRequest(c, "Names array cannot be empty")
	}

	if len(req.Names) > 100 {
		return middleware.BadRequest(c, "Maximum 100 services per batch request")
	}

	log.Debug("Batch deregistering services", logger.Int("count", len(req.Names)))

	deregistered := make([]string, 0, len(req.Names))

	for _, name := range req.Names {
		if h.raftNode != nil {
			entry, marshalErr := konsulraft.NewLogEntry(konsulraft.EntryServiceDeregister, konsulraft.ServiceDeregisterPayload{
				Name: name,
			})
			if marshalErr != nil {
				log.Error("Failed to build raft log entry", logger.Error(marshalErr))
				return middleware.InternalError(c, "Failed to deregister services")
			}
			if _, err := h.raftNode.ApplyEntry(entry, 10*time.Second); err != nil {
				if errors.Is(err, hashiraft.ErrNotLeader) {
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":  "not leader",
						"leader": h.raftNode.Leader(),
					})
				}
				log.Error("Failed to deregister service in batch",
					logger.String("service", name),
					logger.Error(err))
				return middleware.InternalError(c, "Failed to deregister services")
			}
		} else {
			h.serviceStore.Deregister(name)
		}
		deregistered = append(deregistered, name)
	}

	log.Info("Batch service deregistration completed", logger.Int("count", len(deregistered)))
	metrics.ServiceOperationsTotal.WithLabelValues("batch_deregister", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.serviceStore.List())))

	return c.JSON(BatchServiceDeregisterResponse{
		Message:      fmt.Sprintf("Deregistered %d services", len(deregistered)),
		Deregistered: deregistered,
		Count:        len(deregistered),
	})
}

// BatchServiceGetRequest represents a request to get multiple services
type BatchServiceGetRequest struct {
	Names []string `json:"names"`
}

// BatchServiceGetResponse represents the response for batch service get
type BatchServiceGetResponse struct {
	Found    map[string]store.Service `json:"found"`
	NotFound []string                 `json:"not_found"`
}

// BatchServiceGet retrieves multiple services at once
// POST /batch/services/get
func (h *BatchHandler) BatchServiceGet(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req BatchServiceGetRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse batch service get request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if len(req.Names) == 0 {
		return middleware.BadRequest(c, "Names array cannot be empty")
	}

	if len(req.Names) > 100 {
		return middleware.BadRequest(c, "Maximum 100 services per batch request")
	}

	log.Debug("Batch getting services", logger.Int("count", len(req.Names)))

	found := make(map[string]store.Service)
	notFound := make([]string, 0)

	for _, name := range req.Names {
		if svc, ok := h.serviceStore.Get(name); ok {
			found[name] = svc
		} else {
			notFound = append(notFound, name)
		}
	}

	log.Info("Batch service get completed",
		logger.Int("found", len(found)),
		logger.Int("not_found", len(notFound)))
	metrics.ServiceOperationsTotal.WithLabelValues("batch_get", "success").Inc()

	return c.JSON(BatchServiceGetResponse{
		Found:    found,
		NotFound: notFound,
	})
}
