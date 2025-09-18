package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path,omitempty"`
}

// BadRequest returns a 400 Bad Request error response
func BadRequest(c *fiber.Ctx, message string) error {
	return errorResponse(c, fiber.StatusBadRequest, "Bad Request", message)
}

// NotFound returns a 404 Not Found error response
func NotFound(c *fiber.Ctx, message string) error {
	return errorResponse(c, fiber.StatusNotFound, "Not Found", message)
}

// InternalServerError returns a 500 Internal Server Error response
func InternalServerError(c *fiber.Ctx, message string) error {
	return errorResponse(c, fiber.StatusInternalServerError, "Internal Server Error", message)
}

// UnprocessableEntity returns a 422 Unprocessable Entity error response
func UnprocessableEntity(c *fiber.Ctx, message string) error {
	return errorResponse(c, fiber.StatusUnprocessableEntity, "Unprocessable Entity", message)
}

// errorResponse creates a structured error response
func errorResponse(c *fiber.Ctx, status int, error string, message string) error {
	response := ErrorResponse{
		Error:     error,
		Message:   message,
		RequestID: GetRequestID(c),
		Timestamp: time.Now(),
		Path:      c.Path(),
	}

	// Log the error with context
	log := GetLogger(c)
	log.Error("HTTP error response",
		logger.String("error", error),
		logger.String("message", message),
		logger.String("method", c.Method()),
		logger.String("path", c.Path()),
		logger.Int("status", status),
		logger.String("user_ip", c.IP()),
	)

	return c.Status(status).JSON(response)
}

