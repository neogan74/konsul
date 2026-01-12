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

type ServiceHandler struct {
	store    *store.ServiceStore
	raftNode *konsulraft.Node
}

func NewServiceHandler(serviceStore *store.ServiceStore, raftNode *konsulraft.Node) *ServiceHandler {
	return &ServiceHandler{store: serviceStore, raftNode: raftNode}
}

// NewServiceHandlerWithRaft creates a Service handler with Raft support for replicated writes.
func NewServiceHandlerWithRaft(serviceStore *store.ServiceStore, raftNode *konsulraft.Node) *ServiceHandler {
	return &ServiceHandler{
		store:    serviceStore,
		raftNode: raftNode,
	}
}

// isRaftEnabled returns true if Raft clustering is enabled.
func (h *ServiceHandler) isRaftEnabled() bool {
	return h.raftNode != nil
}

// checkLeaderForWrite checks if this node can handle writes.
// Returns nil if writes are allowed, or an error response if not leader.
func (h *ServiceHandler) checkLeaderForWrite(c *fiber.Ctx) error {
	if !h.isRaftEnabled() {
		return nil // Standalone mode, writes allowed
	}

	if h.raftNode.IsLeader() {
		return nil // Leader, writes allowed
	}

	// Not leader, return redirect response
	leaderAddr := h.raftNode.LeaderAddr()
	if leaderAddr == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "no leader",
			"message": "No leader is currently elected. The cluster may be initializing or partitioned.",
		})
	}

	return c.Status(fiber.StatusTemporaryRedirect).JSON(fiber.Map{
		"error":       "not leader",
		"message":     "This node is not the leader. Redirect to leader for write operations.",
		"leader_addr": leaderAddr,
	})
}

func (h *ServiceHandler) Register(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	// Check if this node can handle writes (leader check for Raft mode)
	if err := h.checkLeaderForWrite(c); err != nil {
		return err
	}

	body := struct {
		store.Service
		CAS *uint64 `json:"cas,omitempty"` // Optional CAS index
	}{}

	if err := c.BodyParser(&body); err != nil {
		log.Error("Failed to parse service registration body", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	svc := body.Service

	log.Info("Registering service",
		logger.String("service_name", svc.Name),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port),
		logger.Int("tags", len(svc.Tags)),
		logger.Int("metadata_keys", len(svc.Meta)))

	// Use CAS if provided (CAS operations not replicated via Raft)
	if body.CAS != nil {
		var newIndex uint64
		var err error
		if h.raftNode != nil {
			cmd, marshalErr := konsulraft.NewCommand(konsulraft.CmdServiceRegisterCAS, konsulraft.ServiceRegisterCASPayload{
				Service:       svc,
				ExpectedIndex: *body.CAS,
			})
			if marshalErr != nil {
				log.Error("Failed to build raft log entry", logger.Error(marshalErr))
				return middleware.InternalError(c, "Failed to register service")
			}
			resp, applyErr := h.raftNode.ApplyEntry(cmd, 5*time.Second)
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
			newIndex, err = h.store.RegisterCAS(svc, *body.CAS)
		}
		if err != nil {
			if store.IsCASConflict(err) {
				log.Warn("CAS conflict", logger.String("service", svc.Name), logger.Error(err))
				metrics.ServiceOperationsTotal.WithLabelValues("register", "cas_conflict").Inc()
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error":   "CAS conflict",
					"message": err.Error(),
				})
			}
			if store.IsNotFound(err) {
				log.Warn("Service not found for CAS update", logger.String("service", svc.Name))
				metrics.ServiceOperationsTotal.WithLabelValues("register", "not_found").Inc()
				return middleware.NotFound(c, "Service not found")
			}
			log.Error("Failed to register service with CAS",
				logger.String("service", svc.Name),
				logger.Error(err))
			metrics.ServiceOperationsTotal.WithLabelValues("register", "error").Inc()
			return middleware.BadRequest(c, err.Error())
		}

		log.Info("Service registered successfully with CAS",
			logger.String("service_name", svc.Name))

		// Record service registration metrics
		metrics.ServiceOperationsTotal.WithLabelValues("register", "success").Inc()
		metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))
		metrics.ServiceTagsPerService.Observe(float64(len(svc.Tags)))
		metrics.ServiceMetadataKeysPerService.Observe(float64(len(svc.Meta)))

		return c.JSON(fiber.Map{
			"message":      "service registered",
			"service":      svc,
			"modify_index": newIndex,
		})
	}

	// Use Raft for replicated registration if enabled
	if h.isRaftEnabled() {
		if err := h.raftNode.ServiceRegister(svc.Name, svc.Address, svc.Port, svc.Tags, svc.Meta); err != nil {
			log.Error("Raft service registration failed",
				logger.String("service", svc.Name),
				logger.Error(err))
			metrics.ServiceOperationsTotal.WithLabelValues("register", "error").Inc()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "replication failed",
				"message": err.Error(),
			})
		}
	} else {
		// Standalone mode: direct store registration
		if err := h.store.Register(svc); err != nil {
			log.Error("Failed to register service",
				logger.String("service", svc.Name),
				logger.Error(err))
			metrics.ServiceOperationsTotal.WithLabelValues("register", "error").Inc()
			return middleware.BadRequest(c, err.Error())
		}
	}

	log.Info("Service registered successfully",
		logger.String("service_name", svc.Name))

	// Record service registration metrics
	metrics.ServiceOperationsTotal.WithLabelValues("register", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))

	// Record tags and metadata metrics
	metrics.ServiceTagsPerService.Observe(float64(len(svc.Tags)))
	metrics.ServiceMetadataKeysPerService.Observe(float64(len(svc.Meta)))

	// Return with index
	entry, _ := h.store.GetEntry(svc.Name)
	return c.JSON(fiber.Map{
		"message":      "service registered",
		"service":      svc,
		"modify_index": entry.ModifyIndex,
	})
}

func (h *ServiceHandler) List(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	services := h.store.List()

	log.Debug("Listing services", logger.Int("count", len(services)))
	return c.JSON(services)
}

func (h *ServiceHandler) Get(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	log.Debug("Getting service", logger.String("service_name", name))

	// Check if client wants full entry with indices
	includeMetadata := c.Query("metadata", "false") == "true"

	if includeMetadata {
		entry, ok := h.store.GetEntry(name)
		if !ok {
			log.Warn("Service not found", logger.String("service_name", name))
			metrics.ServiceOperationsTotal.WithLabelValues("get", "not_found").Inc()
			return middleware.NotFound(c, "Service not found")
		}
		log.Info("Service retrieved successfully with metadata", logger.String("service_name", name))
		metrics.ServiceOperationsTotal.WithLabelValues("get", "success").Inc()
		return c.JSON(fiber.Map{
			"service":      entry.Service,
			"expires_at":   entry.ExpiresAt,
			"modify_index": entry.ModifyIndex,
			"create_index": entry.CreateIndex,
		})
	}

	svc, ok := h.store.Get(name)
	if !ok {
		log.Warn("Service not found", logger.String("service_name", name))
		metrics.ServiceOperationsTotal.WithLabelValues("get", "not_found").Inc()
		return middleware.NotFound(c, "Service not found")
	}

	log.Info("Service retrieved successfully", logger.String("service_name", name))
	metrics.ServiceOperationsTotal.WithLabelValues("get", "success").Inc()
	return c.JSON(svc)
}

func (h *ServiceHandler) Deregister(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	// Check if this node can handle writes (leader check for Raft mode)
	if err := h.checkLeaderForWrite(c); err != nil {
		return err
	}

	log.Info("Deregistering service", logger.String("service_name", name))

	// Check if CAS is requested via query parameter (CAS operations not replicated via Raft)
	casParam := c.Query("cas")
	if casParam != "" {
		var expectedIndex uint64
		if _, err := fmt.Sscanf(casParam, "%d", &expectedIndex); err != nil {
			log.Error("Invalid CAS parameter", logger.String("cas", casParam), logger.Error(err))
			return middleware.BadRequest(c, "Invalid CAS parameter")
		}

		var err error
		if h.raftNode != nil {
			cmd, marshalErr := konsulraft.NewCommand(konsulraft.CmdServiceDeregisterCAS, konsulraft.ServiceDeregisterCASPayload{
				Name:          name,
				ExpectedIndex: expectedIndex,
			})
			if marshalErr != nil {
				log.Error("Failed to build raft log entry", logger.Error(marshalErr))
				return middleware.InternalError(c, "Failed to deregister service")
			}
			if _, applyErr := h.raftNode.ApplyEntry(cmd, 5*time.Second); applyErr != nil {
				if errors.Is(applyErr, hashiraft.ErrNotLeader) {
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":  "not leader",
						"leader": h.raftNode.Leader(),
					})
				}
				err = applyErr
			}
		} else {
			err = h.store.DeregisterCAS(name, expectedIndex)
		}
		if err != nil {
			if store.IsCASConflict(err) {
				log.Warn("CAS conflict on deregister", logger.String("service", name), logger.Error(err))
				metrics.ServiceOperationsTotal.WithLabelValues("deregister", "cas_conflict").Inc()
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error":   "CAS conflict",
					"message": err.Error(),
				})
			}
			if store.IsNotFound(err) {
				log.Warn("Service not found for CAS deregister", logger.String("service", name))
				metrics.ServiceOperationsTotal.WithLabelValues("deregister", "not_found").Inc()
				return middleware.NotFound(c, "Service not found")
			}
			log.Error("CAS deregister failed", logger.String("service", name), logger.Error(err))
			metrics.ServiceOperationsTotal.WithLabelValues("deregister", "error").Inc()
			return middleware.InternalError(c, "Failed to deregister service")
		}

		log.Info("Service deregistered successfully with CAS", logger.String("service_name", name))
		metrics.ServiceOperationsTotal.WithLabelValues("deregister", "success").Inc()
		metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))
		return c.JSON(fiber.Map{"message": "service deregistered", "name": name})
	}

	// Use Raft for replicated deregistration if enabled
	if h.isRaftEnabled() {
		if err := h.raftNode.ServiceDeregister(name); err != nil {
			log.Error("Raft service deregistration failed",
				logger.String("service", name),
				logger.Error(err))
			metrics.ServiceOperationsTotal.WithLabelValues("deregister", "error").Inc()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "replication failed",
				"message": err.Error(),
			})
		}
	} else {
		// Standalone mode: direct store deregistration
		h.store.Deregister(name)
	}

	log.Info("Service deregistered successfully", logger.String("service_name", name))
	metrics.ServiceOperationsTotal.WithLabelValues("deregister", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))
	return c.JSON(fiber.Map{"message": "service deregistered", "name": name})
}

func (h *ServiceHandler) Heartbeat(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	// Check if this node can handle writes (leader check for Raft mode)
	if err := h.checkLeaderForWrite(c); err != nil {
		return err
	}

	log.Debug("Processing heartbeat", logger.String("service_name", name))

	// Use Raft for replicated heartbeat if enabled
	if h.isRaftEnabled() {
		if err := h.raftNode.ServiceHeartbeat(name); err != nil {
			log.Error("Raft service heartbeat failed",
				logger.String("service", name),
				logger.Error(err))
			metrics.ServiceHeartbeatsTotal.WithLabelValues(name, "error").Inc()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "replication failed",
				"message": err.Error(),
			})
		}
		log.Info("Heartbeat updated successfully via Raft", logger.String("service_name", name))
		metrics.ServiceHeartbeatsTotal.WithLabelValues(name, "success").Inc()
		return c.JSON(fiber.Map{"message": "heartbeat updated", "service": name})
	}

	// Standalone mode: direct store heartbeat
	if h.store.Heartbeat(name) {
		log.Info("Heartbeat updated successfully", logger.String("service_name", name))
		metrics.ServiceHeartbeatsTotal.WithLabelValues(name, "success").Inc()
		return c.JSON(fiber.Map{"message": "heartbeat updated", "service": name})
	}

	log.Warn("Heartbeat failed - service not found", logger.String("service_name", name))
	metrics.ServiceHeartbeatsTotal.WithLabelValues(name, "not_found").Inc()
	return middleware.NotFound(c, "Service not found")
}

// QueryByTags handles GET /services/query/tags?tags=tag1&tags=tag2
// Returns services that have ALL specified tags (AND logic)
func (h *ServiceHandler) QueryByTags(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	startTime := c.Context().Time()

	// Parse tags from query parameters (can appear multiple times)
	tags := c.Query("tags", "")
	if tags == "" {
		tags = c.Query("tag", "")
	}

	var tagList []string
	if tags != "" {
		tagList = append(tagList, tags)
	}

	// Support multiple tag parameters: ?tags=tag1&tags=tag2
	parser := c.Context().QueryArgs()
	parser.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if (keyStr == "tags" || keyStr == "tag") && string(value) != tags {
			tagList = append(tagList, string(value))
		}
	})

	if len(tagList) == 0 {
		log.Warn("Query by tags called with no tags")
		metrics.ServiceQueryTotal.WithLabelValues("tags", "error").Inc()
		return middleware.BadRequest(c, "At least one tag must be specified")
	}

	log.Info("Querying services by tags",
		logger.Int("tag_count", len(tagList)),
		logger.String("tags", tags))

	services := h.store.QueryByTags(tagList)

	// Record metrics
	duration := c.Context().Time().Sub(startTime).Seconds()
	metrics.ServiceQueryDuration.WithLabelValues("tags").Observe(duration)
	metrics.ServiceQueryResultsCount.WithLabelValues("tags").Observe(float64(len(services)))
	metrics.ServiceQueryTotal.WithLabelValues("tags", "success").Inc()

	log.Info("Query by tags completed",
		logger.Int("result_count", len(services)))

	return c.JSON(fiber.Map{
		"count":    len(services),
		"services": services,
		"query":    fiber.Map{"tags": tagList},
	})
}

// QueryByMetadata handles GET /services/query/metadata?key1=value1&key2=value2
// Returns services that have ALL specified metadata key-value pairs (AND logic)
func (h *ServiceHandler) QueryByMetadata(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	startTime := c.Context().Time()

	// Parse all query parameters as metadata filters
	filters := make(map[string]string)
	parser := c.Context().QueryArgs()
	parser.VisitAll(func(key, value []byte) {
		filters[string(key)] = string(value)
	})

	if len(filters) == 0 {
		log.Warn("Query by metadata called with no filters")
		metrics.ServiceQueryTotal.WithLabelValues("metadata", "error").Inc()
		return middleware.BadRequest(c, "At least one metadata filter must be specified")
	}

	log.Info("Querying services by metadata",
		logger.Int("filter_count", len(filters)))

	services := h.store.QueryByMetadata(filters)

	// Record metrics
	duration := c.Context().Time().Sub(startTime).Seconds()
	metrics.ServiceQueryDuration.WithLabelValues("metadata").Observe(duration)
	metrics.ServiceQueryResultsCount.WithLabelValues("metadata").Observe(float64(len(services)))
	metrics.ServiceQueryTotal.WithLabelValues("metadata", "success").Inc()

	log.Info("Query by metadata completed",
		logger.Int("result_count", len(services)))

	return c.JSON(fiber.Map{
		"count":    len(services),
		"services": services,
		"query":    fiber.Map{"metadata": filters},
	})
}

// QueryByTagsAndMetadata handles combined queries with both tags and metadata
// GET /services/query?tags=tag1&tags=tag2&meta.key1=value1&meta.key2=value2
// Returns services matching ALL specified criteria (AND logic)
func (h *ServiceHandler) QueryByTagsAndMetadata(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	startTime := c.Context().Time()

	// Parse tags
	var tagList []string
	parser := c.Context().QueryArgs()
	parser.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if keyStr == "tags" || keyStr == "tag" {
			tagList = append(tagList, string(value))
		}
	})

	// Parse metadata filters (prefixed with "meta.")
	filters := make(map[string]string)
	parser.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if len(keyStr) > 5 && keyStr[:5] == "meta." {
			metaKey := keyStr[5:]
			filters[metaKey] = string(value)
		}
	})

	if len(tagList) == 0 && len(filters) == 0 {
		log.Warn("Combined query called with no tags or metadata")
		metrics.ServiceQueryTotal.WithLabelValues("combined", "error").Inc()
		return middleware.BadRequest(c, "At least one tag or metadata filter must be specified")
	}

	log.Info("Querying services by tags and metadata",
		logger.Int("tag_count", len(tagList)),
		logger.Int("filter_count", len(filters)))

	services := h.store.QueryByTagsAndMetadata(tagList, filters)

	// Record metrics
	duration := c.Context().Time().Sub(startTime).Seconds()
	metrics.ServiceQueryDuration.WithLabelValues("combined").Observe(duration)
	metrics.ServiceQueryResultsCount.WithLabelValues("combined").Observe(float64(len(services)))
	metrics.ServiceQueryTotal.WithLabelValues("combined", "success").Inc()

	log.Info("Combined query completed",
		logger.Int("result_count", len(services)))

	return c.JSON(fiber.Map{
		"count":    len(services),
		"services": services,
		"query": fiber.Map{
			"tags":     tagList,
			"metadata": filters,
		},
	})
}
