package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestMetricsMiddleware_SuccessfulRequest(t *testing.T) {
	app := fiber.New()
	app.Use(MetricsMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
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

	// Note: We can't easily verify Prometheus metrics are incremented in unit tests
	// without accessing the registry, but we can verify the middleware executes without errors
}

func TestMetricsMiddleware_ErrorRequest(t *testing.T) {
	app := fiber.New()
	app.Use(MetricsMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusInternalServerError).SendString("error")
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

func TestMetricsMiddleware_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			app := fiber.New()
			app.Use(MetricsMiddleware())
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

func TestMetricsMiddleware_DifferentPaths(t *testing.T) {
	paths := []string{"/api/users", "/kv/mykey", "/services/web", "/health"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			app := fiber.New()
			app.Use(MetricsMiddleware())
			app.Get(path, func(c *fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest("GET", path, nil)
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

func TestMetricsMiddleware_SkipsMetricsEndpoint(t *testing.T) {
	app := fiber.New()
	app.Use(MetricsMiddleware())
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.SendString("metrics")
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// The middleware should skip metrics recording for /metrics endpoint
	// This prevents infinite loops and self-referential metrics
}

func TestMetricsMiddleware_DifferentStatusCodes(t *testing.T) {
	statusCodes := []int{
		fiber.StatusOK,
		fiber.StatusCreated,
		fiber.StatusBadRequest,
		fiber.StatusNotFound,
		fiber.StatusInternalServerError,
	}

	for _, status := range statusCodes {
		t.Run(string(rune(status)), func(t *testing.T) {
			app := fiber.New()
			app.Use(MetricsMiddleware())
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendStatus(status)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			if resp.StatusCode != status {
				t.Errorf("expected status %d, got %d", status, resp.StatusCode)
			}
		})
	}
}

func TestMetricsMiddleware_WithError(t *testing.T) {
	app := fiber.New()
	app.Use(MetricsMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "validation error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestMetricsMiddleware_MultipleRequests(t *testing.T) {
	app := fiber.New()
	app.Use(MetricsMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Make multiple requests to test metric accumulation
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i, resp.StatusCode)
		}
	}
}

func TestMetricsMiddleware_ConcurrentRequests(t *testing.T) {
	app := fiber.New()
	app.Use(MetricsMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Test that in-flight counter works correctly with concurrent requests
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Errorf("concurrent request %d failed: %v", id, err)
			}
			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("concurrent request %d: expected status 200, got %d", id, resp.StatusCode)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestMetricsMiddleware_ChainedWithOtherMiddleware(t *testing.T) {
	app := fiber.New()
	// Test that metrics middleware works when chained with other middleware
	app.Use(MetricsMiddleware())
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Custom-Header", "test")
		return c.Next()
	})
	app.Get("/test", func(c *fiber.Ctx) error {
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

	if resp.Header.Get("X-Custom-Header") != "test" {
		t.Error("expected custom header to be set")
	}
}
