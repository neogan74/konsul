package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/store"
)

type ServiceHandler struct {
	store *store.ServiceStore
}

func NewServiceHandler(serviceStore *store.ServiceStore) *ServiceHandler {
	return &ServiceHandler{store: serviceStore}
}

func (h *ServiceHandler) Register(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	var svc store.Service

	if err := c.BodyParser(&svc); err != nil {
		log.Error("Failed to parse service registration body", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	log.Info("Registering service",
		logger.String("service_name", svc.Name),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port),
		logger.Int("tags", len(svc.Tags)),
		logger.Int("metadata_keys", len(svc.Meta)))

	// Register service (validation happens inside)
	if err := h.store.Register(svc); err != nil {
		log.Error("Failed to register service",
			logger.String("service", svc.Name),
			logger.Error(err))
		metrics.ServiceOperationsTotal.WithLabelValues("register", "error").Inc()
		return middleware.BadRequest(c, err.Error())
	}

	log.Info("Service registered successfully",
		logger.String("service_name", svc.Name))

	metrics.ServiceOperationsTotal.WithLabelValues("register", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))

	return c.JSON(fiber.Map{"message": "service registered", "service": svc})
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

	log.Info("Deregistering service", logger.String("service_name", name))

	h.store.Deregister(name)

	log.Info("Service deregistered successfully", logger.String("service_name", name))
	metrics.ServiceOperationsTotal.WithLabelValues("deregister", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))
	return c.JSON(fiber.Map{"message": "service deregistered", "name": name})
}

func (h *ServiceHandler) Heartbeat(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	log.Debug("Processing heartbeat", logger.String("service_name", name))

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
		return middleware.BadRequest(c, "At least one tag must be specified")
	}

	log.Info("Querying services by tags",
		logger.Int("tag_count", len(tagList)),
		logger.String("tags", tags))

	services := h.store.QueryByTags(tagList)

	log.Info("Query by tags completed",
		logger.Int("result_count", len(services)))

	metrics.ServiceOperationsTotal.WithLabelValues("query_tags", "success").Inc()
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

	// Parse all query parameters as metadata filters
	filters := make(map[string]string)
	parser := c.Context().QueryArgs()
	parser.VisitAll(func(key, value []byte) {
		filters[string(key)] = string(value)
	})

	if len(filters) == 0 {
		log.Warn("Query by metadata called with no filters")
		return middleware.BadRequest(c, "At least one metadata filter must be specified")
	}

	log.Info("Querying services by metadata",
		logger.Int("filter_count", len(filters)))

	services := h.store.QueryByMetadata(filters)

	log.Info("Query by metadata completed",
		logger.Int("result_count", len(services)))

	metrics.ServiceOperationsTotal.WithLabelValues("query_metadata", "success").Inc()
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
		return middleware.BadRequest(c, "At least one tag or metadata filter must be specified")
	}

	log.Info("Querying services by tags and metadata",
		logger.Int("tag_count", len(tagList)),
		logger.Int("filter_count", len(filters)))

	services := h.store.QueryByTagsAndMetadata(tagList, filters)

	log.Info("Combined query completed",
		logger.Int("result_count", len(services)))

	metrics.ServiceOperationsTotal.WithLabelValues("query_combined", "success").Inc()
	return c.JSON(fiber.Map{
		"count":    len(services),
		"services": services,
		"query": fiber.Map{
			"tags":     tagList,
			"metadata": filters,
		},
	})
}
