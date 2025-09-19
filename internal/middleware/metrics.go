package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/metrics"
)

// MetricsMiddleware tracks HTTP request metrics
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip metrics endpoint to avoid infinite loop
		if c.Path() == "/metrics" {
			return c.Next()
		}

		// Increment in-flight requests
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// Start timer
		start := time.Now()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Get status code
		status := strconv.Itoa(c.Response().StatusCode())

		// Record metrics
		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Method(),
			c.Path(),
			status,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			c.Method(),
			c.Path(),
			status,
		).Observe(duration)

		return err
	}
}