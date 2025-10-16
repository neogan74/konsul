package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/ratelimit"
)

// RateLimitHandler handles rate limit administration endpoints
type RateLimitHandler struct {
	service *ratelimit.Service
	log     logger.Logger
}

// NewRateLimitHandler creates a new rate limit admin handler
func NewRateLimitHandler(service *ratelimit.Service, log logger.Logger) *RateLimitHandler {
	return &RateLimitHandler{
		service: service,
		log:     log,
	}
}

// GetStats returns current rate limiting statistics
// GET /admin/ratelimit/stats
func (h *RateLimitHandler) GetStats(c *fiber.Ctx) error {
	stats := h.service.Stats()

	h.log.Debug("Rate limit stats retrieved",
		logger.Int("ip_limiters", getIntStat(stats, "ip_limiters")),
		logger.Int("apikey_limiters", getIntStat(stats, "apikey_limiters")))

	return c.JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}

// GetConfig returns current rate limit configuration
// GET /admin/ratelimit/config
func (h *RateLimitHandler) GetConfig(c *fiber.Ctx) error {
	config := h.service.GetConfig()

	return c.JSON(fiber.Map{
		"success": true,
		"config": fiber.Map{
			"enabled":          config.Enabled,
			"requests_per_sec": config.RequestsPerSec,
			"burst":            config.Burst,
			"by_ip":            config.ByIP,
			"by_apikey":        config.ByAPIKey,
			"cleanup_interval": config.CleanupInterval.String(),
		},
	})
}

// ResetIP resets rate limit for a specific IP address
// POST /admin/ratelimit/reset/ip/:ip
func (h *RateLimitHandler) ResetIP(c *fiber.Ctx) error {
	ip := c.Params("ip")
	if ip == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "IP address is required",
		})
	}

	h.service.ResetIP(ip)

	h.log.Info("Rate limit reset for IP",
		logger.String("ip", ip),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Rate limit reset successfully",
		"ip":      ip,
	})
}

// ResetAPIKey resets rate limit for a specific API key
// POST /admin/ratelimit/reset/apikey/:key_id
func (h *RateLimitHandler) ResetAPIKey(c *fiber.Ctx) error {
	keyID := c.Params("key_id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "API key ID is required",
		})
	}

	h.service.ResetAPIKey(keyID)

	h.log.Info("Rate limit reset for API key",
		logger.String("key_id", keyID),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Rate limit reset successfully",
		"key_id":  keyID,
	})
}

// ResetAll resets all rate limiters
// POST /admin/ratelimit/reset/all
func (h *RateLimitHandler) ResetAll(c *fiber.Ctx) error {
	limiterType := c.Query("type", "all") // all, ip, apikey

	var message string
	switch limiterType {
	case "ip":
		h.service.ResetAllIP()
		message = "All IP rate limiters reset"
	case "apikey":
		h.service.ResetAllAPIKey()
		message = "All API key rate limiters reset"
	case "all":
		h.service.ResetAllIP()
		h.service.ResetAllAPIKey()
		message = "All rate limiters reset"
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid type parameter. Must be: all, ip, or apikey",
		})
	}

	h.log.Warn("All rate limiters reset",
		logger.String("type", limiterType),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success": true,
		"message": message,
		"type":    limiterType,
	})
}

// GetActiveClients returns list of currently rate-limited clients
// GET /admin/ratelimit/clients
func (h *RateLimitHandler) GetActiveClients(c *fiber.Ctx) error {
	limiterType := c.Query("type", "all") // all, ip, apikey

	clients := h.service.GetActiveClients(limiterType)

	return c.JSON(fiber.Map{
		"success": true,
		"count":   len(clients),
		"clients": clients,
	})
}

// GetClientStatus returns rate limit status for a specific client
// GET /admin/ratelimit/client/:identifier
func (h *RateLimitHandler) GetClientStatus(c *fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Client identifier is required",
		})
	}

	status := h.service.GetClientStatus(identifier)
	if status == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Client not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"client":  status,
	})
}

// UpdateConfig updates rate limit configuration (dynamic reconfiguration)
// PUT /admin/ratelimit/config
func (h *RateLimitHandler) UpdateConfig(c *fiber.Ctx) error {
	type ConfigUpdate struct {
		RequestsPerSec *float64 `json:"requests_per_sec"`
		Burst          *int     `json:"burst"`
	}

	var update ConfigUpdate
	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate inputs
	if update.RequestsPerSec != nil && *update.RequestsPerSec <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "requests_per_sec must be greater than 0",
		})
	}

	if update.Burst != nil && *update.Burst <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "burst must be greater than 0",
		})
	}

	// Apply updates
	changed := h.service.UpdateConfig(update.RequestsPerSec, update.Burst)
	if !changed {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "No changes applied",
		})
	}

	h.log.Info("Rate limit configuration updated",
		logger.String("admin_user", getAdminUser(c)))

	// Get updated config
	config := h.service.GetConfig()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Configuration updated successfully",
		"config": fiber.Map{
			"requests_per_sec": config.RequestsPerSec,
			"burst":            config.Burst,
		},
	})
}

// Helper functions

func getIntStat(stats map[string]interface{}, key string) int {
	if val, ok := stats[key].(int); ok {
		return val
	}
	return 0
}

func getAdminUser(c *fiber.Ctx) string {
	if username, ok := c.Locals("username").(string); ok {
		return username
	}
	if userID, ok := c.Locals("user_id").(string); ok {
		return userID
	}
	return "unknown"
}
