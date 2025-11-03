package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/loadbalancer"
	"github.com/neogan74/konsul/internal/logger"
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

	log.Debug("Load balancer: selecting service instance",
		logger.String("service_name", serviceName),
		logger.String("strategy", string(h.balancer.GetStrategy())))

	svc, ok := h.balancer.SelectService(serviceName)
	if !ok {
		log.Warn("Load balancer: no instances available",
			logger.String("service_name", serviceName))
		return middleware.NotFound(c, "No service instances available")
	}

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
		return middleware.BadRequest(c, "At least one tag must be specified")
	}

	log.Debug("Load balancer: selecting service by tags",
		logger.Int("tag_count", len(tagList)),
		logger.String("strategy", string(h.balancer.GetStrategy())))

	svc, ok := h.balancer.SelectServiceByTags(tagList)
	if !ok {
		log.Warn("Load balancer: no instances matching tags",
			logger.Int("tag_count", len(tagList)))
		return middleware.NotFound(c, "No service instances matching tags")
	}

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

	// Parse all query parameters as metadata filters
	filters := make(map[string]string)
	parser := c.Context().QueryArgs()
	parser.VisitAll(func(key, value []byte) {
		filters[string(key)] = string(value)
	})

	if len(filters) == 0 {
		log.Warn("Load balancer: no metadata filters specified")
		return middleware.BadRequest(c, "At least one metadata filter must be specified")
	}

	log.Debug("Load balancer: selecting service by metadata",
		logger.Int("filter_count", len(filters)),
		logger.String("strategy", string(h.balancer.GetStrategy())))

	svc, ok := h.balancer.SelectServiceByMetadata(filters)
	if !ok {
		log.Warn("Load balancer: no instances matching metadata",
			logger.Int("filter_count", len(filters)))
		return middleware.NotFound(c, "No service instances matching metadata")
	}

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
		return middleware.BadRequest(c, "At least one tag or metadata filter must be specified")
	}

	log.Debug("Load balancer: selecting service by combined query",
		logger.Int("tag_count", len(tagList)),
		logger.Int("filter_count", len(filters)),
		logger.String("strategy", string(h.balancer.GetStrategy())))

	svc, ok := h.balancer.SelectServiceByQuery(tagList, filters)
	if !ok {
		log.Warn("Load balancer: no instances matching query",
			logger.Int("tag_count", len(tagList)),
			logger.Int("filter_count", len(filters)))
		return middleware.NotFound(c, "No service instances matching query")
	}

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

	strategy := loadbalancer.Strategy(req.Strategy)

	// Validate strategy
	switch strategy {
	case loadbalancer.StrategyRoundRobin, loadbalancer.StrategyRandom, loadbalancer.StrategyLeastConnections:
		h.balancer.SetStrategy(strategy)
		log.Info("Load balancing strategy updated", logger.String("strategy", string(strategy)))
		return c.JSON(fiber.Map{
			"message":  "strategy updated",
			"strategy": strategy,
		})
	default:
		log.Warn("Invalid load balancing strategy requested", logger.String("strategy", req.Strategy))
		return middleware.BadRequest(c, "Invalid strategy. Must be one of: round-robin, random, least-connections")
	}
}
