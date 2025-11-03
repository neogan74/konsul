package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/loadbalancer"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/middleware"
)

type LoadBalancerHandler struct {
	balancer *loadbalancer.Balancer
}

func NewLoadBalancerHandler(balancer *loadbalancer.Balancer) *LoadBalancerHandler {
	return &LoadBalancerHandler{balancer: balancer}
}

// SelectService handles GET /lb/service/:name
// Selects a service instance using the configured load balancing strategy
func (h *LoadBalancerHandler) SelectService(c *fiber.Ctx) error {
	serviceName := c.Params("name")
	log := middleware.GetLogger(c)
	startTime := time.Now()
	strategy := string(h.balancer.GetStrategy())

	log.Debug("Load balancer: selecting service instance",
		logger.String("service_name", serviceName),
		logger.String("strategy", strategy))

	svc, ok := h.balancer.SelectService(serviceName)
	duration := time.Since(startTime).Seconds()

	if !ok {
		log.Warn("Load balancer: no instances available",
			logger.String("service_name", serviceName))
		metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "service", "not_found").Inc()
		metrics.LoadBalancerInstancePoolSize.WithLabelValues("service").Observe(0)
		return middleware.NotFound(c, "No service instances available")
	}

	// Record metrics
	metrics.LoadBalancerSelectionDuration.WithLabelValues(strategy, "service").Observe(duration)
	metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "service", "success").Inc()
	metrics.LoadBalancerInstancePoolSize.WithLabelValues("service").Observe(1)

	log.Info("Load balancer: service instance selected",
		logger.String("service_name", serviceName),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port))

	return c.JSON(fiber.Map{
		"service":  svc,
		"strategy": h.balancer.GetStrategy(),
	})
}

// SelectServiceByTags handles GET /lb/tags?tags=tag1&tags=tag2
// Selects a service instance matching all specified tags
func (h *LoadBalancerHandler) SelectServiceByTags(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	startTime := time.Now()
	strategy := string(h.balancer.GetStrategy())

	// Parse tags from query parameters
	var tagList []string
	parser := c.Context().QueryArgs()
	parser.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if keyStr == "tags" || keyStr == "tag" {
			tagList = append(tagList, string(value))
		}
	})

	if len(tagList) == 0 {
		log.Warn("Load balancer: no tags specified")
		metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "tags", "error").Inc()
		return middleware.BadRequest(c, "At least one tag must be specified")
	}

	log.Debug("Load balancer: selecting service by tags",
		logger.Int("tag_count", len(tagList)),
		logger.String("strategy", strategy))

	svc, ok := h.balancer.SelectServiceByTags(tagList)
	duration := time.Since(startTime).Seconds()

	if !ok {
		log.Warn("Load balancer: no instances matching tags",
			logger.Int("tag_count", len(tagList)))
		metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "tags", "not_found").Inc()
		metrics.LoadBalancerInstancePoolSize.WithLabelValues("tags").Observe(0)
		return middleware.NotFound(c, "No service instances matching tags")
	}

	// Record metrics
	metrics.LoadBalancerSelectionDuration.WithLabelValues(strategy, "tags").Observe(duration)
	metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "tags", "success").Inc()
	metrics.LoadBalancerInstancePoolSize.WithLabelValues("tags").Observe(1)

	log.Info("Load balancer: service instance selected by tags",
		logger.String("service_name", svc.Name),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port))

	return c.JSON(fiber.Map{
		"service":  svc,
		"strategy": h.balancer.GetStrategy(),
		"query":    fiber.Map{"tags": tagList},
	})
}

// SelectServiceByMetadata handles GET /lb/metadata?key1=value1&key2=value2
// Selects a service instance matching all specified metadata
func (h *LoadBalancerHandler) SelectServiceByMetadata(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	startTime := time.Now()
	strategy := string(h.balancer.GetStrategy())

	// Parse all query parameters as metadata filters
	filters := make(map[string]string)
	parser := c.Context().QueryArgs()
	parser.VisitAll(func(key, value []byte) {
		filters[string(key)] = string(value)
	})

	if len(filters) == 0 {
		log.Warn("Load balancer: no metadata filters specified")
		metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "metadata", "error").Inc()
		return middleware.BadRequest(c, "At least one metadata filter must be specified")
	}

	log.Debug("Load balancer: selecting service by metadata",
		logger.Int("filter_count", len(filters)),
		logger.String("strategy", strategy))

	svc, ok := h.balancer.SelectServiceByMetadata(filters)
	duration := time.Since(startTime).Seconds()

	if !ok {
		log.Warn("Load balancer: no instances matching metadata",
			logger.Int("filter_count", len(filters)))
		metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "metadata", "not_found").Inc()
		metrics.LoadBalancerInstancePoolSize.WithLabelValues("metadata").Observe(0)
		return middleware.NotFound(c, "No service instances matching metadata")
	}

	// Record metrics
	metrics.LoadBalancerSelectionDuration.WithLabelValues(strategy, "metadata").Observe(duration)
	metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "metadata", "success").Inc()
	metrics.LoadBalancerInstancePoolSize.WithLabelValues("metadata").Observe(1)

	log.Info("Load balancer: service instance selected by metadata",
		logger.String("service_name", svc.Name),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port))

	return c.JSON(fiber.Map{
		"service":  svc,
		"strategy": h.balancer.GetStrategy(),
		"query":    fiber.Map{"metadata": filters},
	})
}

// SelectServiceByQuery handles GET /lb/query?tags=tag1&tags=tag2&meta.key1=value1
// Selects a service instance matching both tags and metadata
func (h *LoadBalancerHandler) SelectServiceByQuery(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	startTime := time.Now()
	strategy := string(h.balancer.GetStrategy())

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
		log.Warn("Load balancer: no tags or metadata specified")
		metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "combined", "error").Inc()
		return middleware.BadRequest(c, "At least one tag or metadata filter must be specified")
	}

	log.Debug("Load balancer: selecting service by combined query",
		logger.Int("tag_count", len(tagList)),
		logger.Int("filter_count", len(filters)),
		logger.String("strategy", strategy))

	svc, ok := h.balancer.SelectServiceByQuery(tagList, filters)
	duration := time.Since(startTime).Seconds()

	if !ok {
		log.Warn("Load balancer: no instances matching query",
			logger.Int("tag_count", len(tagList)),
			logger.Int("filter_count", len(filters)))
		metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "combined", "not_found").Inc()
		metrics.LoadBalancerInstancePoolSize.WithLabelValues("combined").Observe(0)
		return middleware.NotFound(c, "No service instances matching query")
	}

	// Record metrics
	metrics.LoadBalancerSelectionDuration.WithLabelValues(strategy, "combined").Observe(duration)
	metrics.LoadBalancerSelectionsTotal.WithLabelValues(strategy, "combined", "success").Inc()
	metrics.LoadBalancerInstancePoolSize.WithLabelValues("combined").Observe(1)

	log.Info("Load balancer: service instance selected by combined query",
		logger.String("service_name", svc.Name),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port))

	return c.JSON(fiber.Map{
		"service": svc,
		"strategy": h.balancer.GetStrategy(),
		"query": fiber.Map{
			"tags":     tagList,
			"metadata": filters,
		},
	})
}

// GetStrategy handles GET /lb/strategy
// Returns the current load balancing strategy
func (h *LoadBalancerHandler) GetStrategy(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"strategy": h.balancer.GetStrategy(),
	})
}

// UpdateStrategy handles PUT /lb/strategy
// Updates the load balancing strategy
func (h *LoadBalancerHandler) UpdateStrategy(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req struct {
		Strategy string `json:"strategy"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse strategy update request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	oldStrategy := string(h.balancer.GetStrategy())
	newStrategy := loadbalancer.Strategy(req.Strategy)

	// Validate strategy
	switch newStrategy {
	case loadbalancer.StrategyRoundRobin, loadbalancer.StrategyRandom, loadbalancer.StrategyLeastConnections:
		h.balancer.SetStrategy(newStrategy)

		// Record metrics
		metrics.LoadBalancerStrategyChanges.WithLabelValues(oldStrategy, string(newStrategy)).Inc()

		// Update current strategy gauge
		metrics.LoadBalancerCurrentStrategy.WithLabelValues("round-robin").Set(0)
		metrics.LoadBalancerCurrentStrategy.WithLabelValues("random").Set(0)
		metrics.LoadBalancerCurrentStrategy.WithLabelValues("least-connections").Set(0)
		metrics.LoadBalancerCurrentStrategy.WithLabelValues(string(newStrategy)).Set(1)

		log.Info("Load balancing strategy updated",
			logger.String("old_strategy", oldStrategy),
			logger.String("new_strategy", string(newStrategy)))

		return c.JSON(fiber.Map{
			"message":  "strategy updated",
			"strategy": newStrategy,
		})
	default:
		log.Warn("Invalid load balancing strategy requested", logger.String("strategy", req.Strategy))
		return middleware.BadRequest(c, "Invalid strategy. Must be one of: round-robin, random, least-connections")
	}
}
