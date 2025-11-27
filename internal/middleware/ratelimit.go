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
		var limiter *ratelimit.Limiter
		var store *ratelimit.Store

		// Determine identifier type
		var limiterType string
		var rawIdentifier string
		if apiKeyID != "" {
			rawIdentifier = apiKeyID
			limiterType = "apikey"
		} else {
			rawIdentifier = clientIP
			limiterType = "ip"
		}
		identifier = fmt.Sprintf("%s:%s", limiterType, rawIdentifier)

		// Check blacklist first - highest priority
		if service.IsBlacklisted(rawIdentifier) {
			metrics.RateLimitExceeded.WithLabelValues(limiterType).Inc()
			metrics.RateLimitRequestsTotal.WithLabelValues(limiterType, "blacklisted").Inc()

			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":      "access_denied",
				"message":    "Your access has been blocked",
				"identifier": identifier,
			})
		}

		// Check whitelist - bypasses rate limiting
		if service.IsWhitelisted(rawIdentifier) {
			metrics.RateLimitRequestsTotal.WithLabelValues(limiterType, "whitelisted").Inc()
			return c.Next()
		}

		// Apply normal rate limiting
		if apiKeyID != "" {
			store = service.GetAPIKeyStore()
			if store != nil {
				limiter = store.GetLimiter(apiKeyID)
				allowed = limiter.AllowWithEndpoint(c.Path())
			} else {
				allowed = true
			}
		} else {
			// Fall back to IP-based rate limiting
			store = service.GetIPStore()
			if store != nil {
				limiter = store.GetLimiter(clientIP)
				allowed = limiter.AllowWithEndpoint(c.Path())
			} else {
				allowed = true
			}
		}

		// Get RFC 6585 compliant headers
		if limiter != nil {
			limit, remaining, resetAt := limiter.GetHeaders()
			c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))
		}

		if !allowed {
			// Record rate limit exceeded
			metrics.RateLimitExceeded.WithLabelValues(limiterType).Inc()
			metrics.RateLimitRequestsTotal.WithLabelValues(limiterType, "exceeded").Inc()

			// Calculate Retry-After in seconds
			if limiter != nil {
				_, _, resetAt := limiter.GetHeaders()
				retryAfter := int(time.Unix(resetAt, 0).Sub(time.Now()).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))

				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error":       "rate_limit_exceeded",
					"message":     fmt.Sprintf("Rate limit exceeded. Please retry after %d seconds.", retryAfter),
					"identifier":  identifier,
					"retry_after": retryAfter,
					"reset_at":    time.Unix(resetAt, 0).Format(time.RFC3339),
				})
			}

			// Fallback if limiter is nil (shouldn't happen)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":      "rate_limit_exceeded",
				"message":    "Too many requests. Please try again later.",
				"identifier": identifier,
			})
		}

		// Record successful rate limit check
		metrics.RateLimitRequestsTotal.WithLabelValues(limiterType, "allowed").Inc()

		return c.Next()
	}
}

// RateLimitWithConfig creates a middleware with custom configuration for specific endpoints
func RateLimitWithConfig(requestsPerSec float64, burst int) fiber.Handler {
	store := ratelimit.NewStore(requestsPerSec, burst, 5*time.Minute)

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

		// Get the limiter for this client
		limiter := store.GetLimiter(identifier)

		// Get RFC 6585 compliant headers
		limit, remaining, resetAt := limiter.GetHeaders()
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

		// Check if request is allowed
		if !limiter.Allow() {
			// Calculate Retry-After in seconds
			retryAfter := int(time.Unix(resetAt, 0).Sub(time.Now()).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate_limit_exceeded",
				"message":     fmt.Sprintf("Rate limit exceeded. Please retry after %d seconds.", retryAfter),
				"retry_after": retryAfter,
				"reset_at":    time.Unix(resetAt, 0).Format(time.RFC3339),
			})
		}

		return c.Next()
	}
}
