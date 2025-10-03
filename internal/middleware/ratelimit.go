package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/ratelimit"
)

// RateLimitMiddleware creates a middleware for rate limiting
func RateLimitMiddleware(service *ratelimit.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get client identifier (IP or API key from context)
		clientIP := c.IP()
		apiKeyID := ""

		// Try to get API key ID from context (set by API key auth middleware)
		if id, ok := c.Locals("api_key_id").(string); ok && id != "" {
			apiKeyID = id
		}

		var allowed bool
		var identifier string

		// Check API key rate limit first if available
		if apiKeyID != "" {
			allowed = service.AllowAPIKey(apiKeyID)
			identifier = fmt.Sprintf("apikey:%s", apiKeyID)
		} else {
			// Fall back to IP-based rate limiting
			allowed = service.AllowIP(clientIP)
			identifier = fmt.Sprintf("ip:%s", clientIP)
		}

		if !allowed {
			// Rate limit exceeded
			c.Set("X-RateLimit-Limit", "exceeded")
			c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":      "rate limit exceeded",
				"message":    "Too many requests. Please try again later.",
				"identifier": identifier,
			})
		}

		// Set rate limit headers (informational)
		c.Set("X-RateLimit-Limit", "ok")

		return c.Next()
	}
}

// RateLimitWithConfig creates a middleware with custom configuration for specific endpoints
func RateLimitWithConfig(requestsPerSec float64, burst int) fiber.Handler {
	limiter := ratelimit.NewStore(requestsPerSec, burst, 5*time.Minute)

	return func(c *fiber.Ctx) error {
		clientIP := c.IP()
		apiKeyID := ""

		// Try to get API key ID from context (set by API key auth middleware)
		if id, ok := c.Locals("api_key_id").(string); ok && id != "" {
			apiKeyID = id
		}

		identifier := clientIP
		if apiKeyID != "" {
			identifier = apiKeyID
		}

		if !limiter.Allow(identifier) {
			c.Set("X-RateLimit-Limit", "exceeded")
			c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "rate limit exceeded",
				"message": "Too many requests. Please try again later.",
			})
		}

		c.Set("X-RateLimit-Limit", "ok")
		return c.Next()
	}
}
