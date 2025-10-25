package middleware

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
)

func TestBadRequest(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest(c, "invalid input data")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	// Parse response body
	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify error response structure
	if errResp.Error != "Bad Request" {
		t.Errorf("expected error 'Bad Request', got %q", errResp.Error)
	}
	if errResp.Message != "invalid input data" {
		t.Errorf("expected message 'invalid input data', got %q", errResp.Message)
	}
	if errResp.RequestID == "" {
		t.Error("expected request ID to be set")
	}
	if errResp.Path != "/test" {
		t.Errorf("expected path '/test', got %q", errResp.Path)
	}
	if errResp.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestNotFound(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return NotFound(c, "resource not found")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if errResp.Error != "Not Found" {
		t.Errorf("expected error 'Not Found', got %q", errResp.Error)
	}
	if errResp.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", errResp.Message)
	}
}

func TestInternalServerError(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return InternalServerError(c, "database connection failed")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if errResp.Error != "Internal Server Error" {
		t.Errorf("expected error 'Internal Server Error', got %q", errResp.Error)
	}
	if errResp.Message != "database connection failed" {
		t.Errorf("expected message 'database connection failed', got %q", errResp.Message)
	}
}

func TestUnprocessableEntity(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Post("/test", func(c *fiber.Ctx) error {
		return UnprocessableEntity(c, "validation failed")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if errResp.Error != "Unprocessable Entity" {
		t.Errorf("expected error 'Unprocessable Entity', got %q", errResp.Error)
	}
	if errResp.Message != "validation failed" {
		t.Errorf("expected message 'validation failed', got %q", errResp.Message)
	}
}

func TestConflict(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Post("/test", func(c *fiber.Ctx) error {
		return Conflict(c, "resource already exists")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusConflict {
		t.Errorf("expected status 409, got %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if errResp.Error != "Conflict" {
		t.Errorf("expected error 'Conflict', got %q", errResp.Error)
	}
	if errResp.Message != "resource already exists" {
		t.Errorf("expected message 'resource already exists', got %q", errResp.Message)
	}
}

func TestInternalError_Alias(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return InternalError(c, "something went wrong")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// InternalError is an alias for InternalServerError, should return 500
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if errResp.Error != "Internal Server Error" {
		t.Errorf("expected error 'Internal Server Error', got %q", errResp.Error)
	}
	if errResp.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %q", errResp.Message)
	}
}

func TestErrorResponse_WithoutRequestID(t *testing.T) {
	app := fiber.New()
	// No logging middleware, so no request ID in context
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest(c, "test error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Request ID should be empty when logging middleware is not used
	if errResp.RequestID != "" {
		t.Errorf("expected empty request ID without logging middleware, got %q", errResp.RequestID)
	}

	// But other fields should still be set
	if errResp.Error != "Bad Request" {
		t.Errorf("expected error 'Bad Request', got %q", errResp.Error)
	}
	if errResp.Path != "/test" {
		t.Errorf("expected path '/test', got %q", errResp.Path)
	}
	if errResp.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestErrorResponse_TimestampRecent(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return NotFound(c, "not found")
	})

	before := time.Now()
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	after := time.Now()

	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Timestamp should be within the test execution window
	if errResp.Timestamp.Before(before) || errResp.Timestamp.After(after) {
		t.Errorf("timestamp %v is outside expected range [%v, %v]", errResp.Timestamp, before, after)
	}
}

func TestErrorResponse_DifferentPaths(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	testCases := []struct {
		path     string
		expected string
	}{
		{"/api/users", "/api/users"},
		{"/kv/mykey", "/kv/mykey"},
		{"/services/web", "/services/web"},
		{"/", "/"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			app := fiber.New()
			app.Use(RequestLogging(log))
			app.Get(tc.path, func(c *fiber.Ctx) error {
				return NotFound(c, "test")
			})

			req := httptest.NewRequest("GET", tc.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			var errResp ErrorResponse
			body, _ := io.ReadAll(resp.Body)
			if err := json.Unmarshal(body, &errResp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if errResp.Path != tc.expected {
				t.Errorf("expected path %q, got %q", tc.expected, errResp.Path)
			}
		})
	}
}

func TestErrorResponse_EmptyMessage(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return BadRequest(c, "")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	var errResp ErrorResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Empty message should be allowed
	if errResp.Message != "" {
		t.Errorf("expected empty message, got %q", errResp.Message)
	}

	// But error field should still be set
	if errResp.Error != "Bad Request" {
		t.Errorf("expected error 'Bad Request', got %q", errResp.Error)
	}
}

func TestErrorResponse_JSONStructure(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return Conflict(c, "duplicate entry")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify response is valid JSON
	body, _ := io.ReadAll(resp.Body)

	var rawJSON map[string]interface{}
	if err := json.Unmarshal(body, &rawJSON); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Verify expected fields exist
	expectedFields := []string{"error", "message", "request_id", "timestamp", "path"}
	for _, field := range expectedFields {
		if _, exists := rawJSON[field]; !exists {
			t.Errorf("expected field %q to exist in JSON response", field)
		}
	}
}

func TestErrorResponse_ContentType(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return InternalServerError(c, "error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	// Verify Content-Type is application/json
	contentType := resp.Header.Get("Content-Type")
	if !contains(contentType, "application/json") {
		t.Errorf("expected Content-Type to contain 'application/json', got %q", contentType)
	}
}
