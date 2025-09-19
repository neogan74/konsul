package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/neogan74/konsul/internal/logger"
)

// RequestIDKey is the context key for request ID
const RequestIDKey = "request_id"

// LoggerKey is the context key for logger instance
const LoggerKey = "logger"

// RequestLogging creates a middleware for request/response logging with correlation IDs
func RequestLogging(log logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Generate request ID
		requestID := uuid.New().String()

		// Store request ID in context
		c.Locals(RequestIDKey, requestID)

		// Create request-scoped logger
		requestLogger := log.WithRequest(requestID)
		c.Locals(LoggerKey, requestLogger)

		// Log request
		start := time.Now()
		requestLogger.Info("Request started",
			logger.String("method", c.Method()),
			logger.String("path", c.Path()),
			logger.String("ip", c.IP()),
			logger.String("user_agent", c.Get("User-Agent")),
		)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log response
		status := c.Response().StatusCode()
		logFields := []logger.Field{
			logger.String("method", c.Method()),
			logger.String("path", c.Path()),
			logger.Int("status", status),
			logger.Duration("duration", duration),
			logger.Int("response_size", len(c.Response().Body())),
		}

		// Log level based on status code
		switch {
		case status >= 500:
			requestLogger.Error("Request completed", logFields...)
		case status >= 400:
			requestLogger.Warn("Request completed", logFields...)
		default:
			requestLogger.Info("Request completed", logFields...)
		}

		// Log error if present
		if err != nil {
			requestLogger.Error("Request error",
				logger.Error(err),
				logger.String("method", c.Method()),
				logger.String("path", c.Path()),
			)
		}

		return err
	}
}

// GetRequestID returns the request ID from the context
func GetRequestID(c *fiber.Ctx) string {
	if requestID, ok := c.Locals(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetLogger returns the request-scoped logger from the context
func GetLogger(c *fiber.Ctx) logger.Logger {
	if log, ok := c.Locals(LoggerKey).(logger.Logger); ok {
		return log
	}
	// Return default logger as fallback
	return logger.NewFromConfig("info", "text")
}