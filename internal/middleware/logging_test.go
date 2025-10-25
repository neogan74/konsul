package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
)

func TestRequestLogging(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestLogging_WithRequestID(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		requestID := GetRequestID(c)
		if requestID == "" {
			t.Error("expected request ID to be set")
		}

		// Verify it's a valid UUID format (basic check)
		if len(requestID) != 36 {
			t.Errorf("expected UUID length 36, got %d", len(requestID))
		}

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestRequestLogging_WithLogger(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		requestLogger := GetLogger(c)
		if requestLogger == nil {
			t.Error("expected logger to be set")
		}

		// Test that logger is usable
		requestLogger.Info("test message")

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestLogging_StatusCodes(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	testCases := []struct {
		name   string
		status int
	}{
		{"success", fiber.StatusOK},
		{"created", fiber.StatusCreated},
		{"bad request", fiber.StatusBadRequest},
		{"not found", fiber.StatusNotFound},
		{"internal error", fiber.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(RequestLogging(log))
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendStatus(tc.status)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			if resp.StatusCode != tc.status {
				t.Errorf("expected status %d, got %d", tc.status, resp.StatusCode)
			}
		})
	}
}

func TestRequestLogging_WithError(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "test error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestGetRequestID_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		requestID := GetRequestID(c)
		if requestID != "" {
			t.Errorf("expected empty request ID, got %q", requestID)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetLogger_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		// Should return default logger as fallback
		log := GetLogger(c)
		if log == nil {
			t.Error("expected default logger, got nil")
		}

		// Test that default logger is usable
		log.Info("test message")

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestLogging_LargeResponse(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	app := fiber.New()
	app.Use(RequestLogging(log))
	app.Get("/test", func(c *fiber.Ctx) error {
		// Generate a large response
		largeData := make([]byte, 10000)
		for i := range largeData {
			largeData[i] = 'a'
		}
		return c.Send(largeData)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestLogging_DifferentMethods(t *testing.T) {
	log := logger.NewFromConfig("info", "text")

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			app := fiber.New()
			app.Use(RequestLogging(log))
			app.All("/test", func(c *fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest(method, "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}
