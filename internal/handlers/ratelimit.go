package handlers

import (
	"fmt"
	"time"

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

// AdjustClientLimit adjusts rate limit for a specific client temporarily
// PUT /admin/ratelimit/client/:type/:id
func (h *RateLimitHandler) AdjustClientLimit(c *fiber.Ctx) error {
	clientType := c.Params("type") // ip or apikey
	identifier := c.Params("id")

	if clientType != "ip" && clientType != "apikey" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid type. Must be 'ip' or 'apikey'",
		})
	}

	if identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Identifier is required",
		})
	}

	type LimitAdjustment struct {
		Rate     float64 `json:"rate"`     // requests per second
		Burst    int     `json:"burst"`    // max burst size
		Duration string  `json:"duration"` // e.g., "1h", "30m"
	}

	var adj LimitAdjustment
	if err := c.BodyParser(&adj); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate
	if adj.Rate <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "rate must be greater than 0",
		})
	}
	if adj.Burst <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "burst must be greater than 0",
		})
	}

	// Parse duration
	duration, err := parseDuration(adj.Duration)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid duration format",
			"details": err.Error(),
		})
	}

	// Get the limiter and apply custom config
	var store *ratelimit.Store
	if clientType == "ip" {
		store = h.service.GetIPStore()
	} else {
		store = h.service.GetAPIKeyStore()
	}

	if store == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Rate limiting not enabled for this type",
		})
	}

	limiter := store.GetLimiter(identifier)
	limiter.SetCustomConfig(adj.Rate, adj.Burst, duration)

	h.log.Info("Client rate limit adjusted",
		logger.String("type", clientType),
		logger.String("identifier", identifier),
		logger.String("rate", fmt.Sprintf("%.2f", adj.Rate)),
		logger.Int("burst", adj.Burst),
		logger.String("duration", adj.Duration),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Rate limit adjusted successfully",
		"type":       clientType,
		"identifier": identifier,
		"config": fiber.Map{
			"rate":     adj.Rate,
			"burst":    adj.Burst,
			"duration": adj.Duration,
		},
	})
}

// GetWhitelist returns all whitelisted entries
// GET /admin/ratelimit/whitelist
func (h *RateLimitHandler) GetWhitelist(c *fiber.Ctx) error {
	entries := h.service.GetAccessList().GetWhitelist()

	return c.JSON(fiber.Map{
		"success": true,
		"count":   len(entries),
		"entries": entries,
	})
}

// AddToWhitelist adds an identifier to the whitelist
// POST /admin/ratelimit/whitelist
func (h *RateLimitHandler) AddToWhitelist(c *fiber.Ctx) error {
	type WhitelistRequest struct {
		Identifier string  `json:"identifier"`
		Type       string  `json:"type"` // ip or apikey
		Reason     string  `json:"reason"`
		Duration   *string `json:"duration"` // optional, e.g., "24h"
	}

	var req WhitelistRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate
	if req.Identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "identifier is required",
		})
	}
	if req.Type != "ip" && req.Type != "apikey" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "type must be 'ip' or 'apikey'",
		})
	}

	// Parse optional duration
	var expiresAt *time.Time
	if req.Duration != nil {
		duration, err := parseDuration(*req.Duration)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid duration format",
				"details": err.Error(),
			})
		}
		expires := time.Now().Add(duration)
		expiresAt = &expires
	}

	// Add to whitelist
	entry := ratelimit.WhitelistEntry{
		Identifier: req.Identifier,
		Type:       req.Type,
		Reason:     req.Reason,
		AddedBy:    getAdminUser(c),
		ExpiresAt:  expiresAt,
	}

	if err := h.service.GetAccessList().AddToWhitelist(entry); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	h.log.Info("Added to whitelist",
		logger.String("type", req.Type),
		logger.String("identifier", req.Identifier),
		logger.String("reason", req.Reason),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Added to whitelist successfully",
		"entry":   entry,
	})
}

// RemoveFromWhitelist removes an identifier from the whitelist
// DELETE /admin/ratelimit/whitelist/:identifier
func (h *RateLimitHandler) RemoveFromWhitelist(c *fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Identifier is required",
		})
	}

	removed := h.service.GetAccessList().RemoveFromWhitelist(identifier)
	if !removed {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Identifier not found in whitelist",
		})
	}

	h.log.Info("Removed from whitelist",
		logger.String("identifier", identifier),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Removed from whitelist successfully",
		"identifier": identifier,
	})
}

// GetBlacklist returns all blacklisted entries
// GET /admin/ratelimit/blacklist
func (h *RateLimitHandler) GetBlacklist(c *fiber.Ctx) error {
	entries := h.service.GetAccessList().GetBlacklist()

	return c.JSON(fiber.Map{
		"success": true,
		"count":   len(entries),
		"entries": entries,
	})
}

// AddToBlacklist adds an identifier to the blacklist
// POST /admin/ratelimit/blacklist
func (h *RateLimitHandler) AddToBlacklist(c *fiber.Ctx) error {
	type BlacklistRequest struct {
		Identifier string `json:"identifier"`
		Type       string `json:"type"` // ip or apikey
		Reason     string `json:"reason"`
		Duration   string `json:"duration"` // required, e.g., "24h"
	}

	var req BlacklistRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	// Validate
	if req.Identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "identifier is required",
		})
	}
	if req.Type != "ip" && req.Type != "apikey" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "type must be 'ip' or 'apikey'",
		})
	}
	if req.Duration == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "duration is required for blacklist entries",
		})
	}

	// Parse duration
	duration, err := parseDuration(req.Duration)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid duration format",
			"details": err.Error(),
		})
	}

	// Add to blacklist
	entry := ratelimit.BlacklistEntry{
		Identifier: req.Identifier,
		Type:       req.Type,
		Reason:     req.Reason,
		AddedBy:    getAdminUser(c),
		ExpiresAt:  time.Now().Add(duration),
	}

	if err := h.service.GetAccessList().AddToBlacklist(entry); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	h.log.Warn("Added to blacklist",
		logger.String("type", req.Type),
		logger.String("identifier", req.Identifier),
		logger.String("reason", req.Reason),
		logger.String("duration", req.Duration),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Added to blacklist successfully",
		"entry":   entry,
	})
}

// RemoveFromBlacklist removes an identifier from the blacklist
// DELETE /admin/ratelimit/blacklist/:identifier
func (h *RateLimitHandler) RemoveFromBlacklist(c *fiber.Ctx) error {
	identifier := c.Params("identifier")
	if identifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Identifier is required",
		})
	}

	removed := h.service.GetAccessList().RemoveFromBlacklist(identifier)
	if !removed {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Identifier not found in blacklist",
		})
	}

	h.log.Info("Removed from blacklist",
		logger.String("identifier", identifier),
		logger.String("admin_user", getAdminUser(c)))

	return c.JSON(fiber.Map{
		"success":    true,
		"message":    "Removed from blacklist successfully",
		"identifier": identifier,
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

func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
