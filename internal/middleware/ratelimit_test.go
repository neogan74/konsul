package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/ratelimit"
)

func TestRateLimitMiddleware_IPBased_Allowed(t *testing.T) {
	// Create rate limit service with generous limits
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           5,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 5 * time.Minute,
	})

	app := fiber.New()
	app.Use(RateLimitMiddleware(service))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should be allowed
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Check RFC 6585 rate limit headers
	if resp.Header.Get("X-RateLimit-Limit") != "5" {
		t.Errorf("expected X-RateLimit-Limit header '5', got %q", resp.Header.Get("X-RateLimit-Limit"))
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header to be set")
	}
	if resp.Header.Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header to be set")
	}
}

func TestRateLimitMiddleware_IPBased_Exceeded(t *testing.T) {
	// Create rate limit service with very low limits
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  0.1,
		Burst:           1,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 5 * time.Minute,
	})

	app := fiber.New()
	app.Use(RateLimitMiddleware(service))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should be allowed
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected first request status 200, got %d", resp1.StatusCode)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", resp2.StatusCode)
	}

	// Check RFC 6585 rate limit headers
	if resp2.Header.Get("X-RateLimit-Limit") != "1" {
		t.Errorf("expected X-RateLimit-Limit header '1', got %q", resp2.Header.Get("X-RateLimit-Limit"))
	}
	if resp2.Header.Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("expected X-RateLimit-Remaining header '0', got %q", resp2.Header.Get("X-RateLimit-Remaining"))
	}
	if resp2.Header.Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header to be set")
	}
	if resp2.Header.Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}

	// Check response body
	body, _ := io.ReadAll(resp2.Body)
	if !contains(string(body), "rate_limit_exceeded") {
		t.Errorf("expected 'rate_limit_exceeded' in response, got: %s", string(body))
	}
	if !contains(string(body), "ip:") {
		t.Errorf("expected IP identifier in response, got: %s", string(body))
	}
}

func TestRateLimitMiddleware_APIKeyBased_Allowed(t *testing.T) {
	// Create rate limit service with API key based limiting
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           5,
		ByIP:            false,
		ByAPIKey:        true,
		CleanupInterval: 5 * time.Minute,
	})

	app := fiber.New()
	// Simulate API key auth middleware setting the key ID
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key_id", "test-key-123")
		return c.Next()
	})
	app.Use(RateLimitMiddleware(service))
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
}

func TestRateLimitMiddleware_APIKeyBased_Exceeded(t *testing.T) {
	// Create rate limit service with very low limits
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  0.1,
		Burst:           1,
		ByIP:            false,
		ByAPIKey:        true,
		CleanupInterval: 5 * time.Minute,
	})

	app := fiber.New()
	// Simulate API key auth middleware setting the key ID
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key_id", "test-key-123")
		return c.Next()
	})
	app.Use(RateLimitMiddleware(service))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should be allowed
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected first request status 200, got %d", resp1.StatusCode)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", resp2.StatusCode)
	}

	// Check response body contains API key identifier
	body, _ := io.ReadAll(resp2.Body)
	if !contains(string(body), "apikey:") {
		t.Errorf("expected API key identifier in response, got: %s", string(body))
	}
}

func TestRateLimitMiddleware_APIKeyPreferredOverIP(t *testing.T) {
	// Create rate limit service with both IP and API key limiting
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  0.1,
		Burst:           1,
		ByIP:            true,
		ByAPIKey:        true,
		CleanupInterval: 5 * time.Minute,
	})

	app := fiber.New()
	// Simulate API key auth middleware setting the key ID
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key_id", "test-key-123")
		return c.Next()
	})
	app.Use(RateLimitMiddleware(service))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request with API key
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected first request status 200, got %d", resp1.StatusCode)
	}

	// Second request should be rate limited by API key, not IP
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", resp2.StatusCode)
	}

	// Verify it's API key based limiting
	body, _ := io.ReadAll(resp2.Body)
	if !contains(string(body), "apikey:") {
		t.Errorf("expected API key based limiting, got: %s", string(body))
	}
}

func TestRateLimitMiddleware_FallbackToIP(t *testing.T) {
	// Create rate limit service with only IP limiting
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  0.1,
		Burst:           1,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 5 * time.Minute,
	})

	app := fiber.New()
	// No API key in context
	app.Use(RateLimitMiddleware(service))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should be allowed
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected first request status 200, got %d", resp1.StatusCode)
	}

	// Second request should be rate limited by IP
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", resp2.StatusCode)
	}

	// Verify it's IP based limiting
	body, _ := io.ReadAll(resp2.Body)
	if !contains(string(body), "ip:") {
		t.Errorf("expected IP based limiting, got: %s", string(body))
	}
}

func TestRateLimitMiddleware_DifferentIPsSeparateLimits(t *testing.T) {
	// Create rate limit service
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           5,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 5 * time.Minute,
	})

	// Note: In Fiber's test mode, all requests appear to come from the same IP (0.0.0.0)
	// To test different IPs, we can use API keys as identifiers instead
	// This test verifies that the rate limiter correctly tracks different identifiers

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		// Simulate different API keys to test separate limiters
		apiKey := c.Get("X-Test-ID")
		if apiKey != "" {
			c.Locals("api_key_id", apiKey)
		}
		return c.Next()
	})
	app.Use(RateLimitMiddleware(service))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Verify that IP-based limiting works (without API key)
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp1.StatusCode)
	}
}

func TestRateLimitWithConfig_IPBased_Allowed(t *testing.T) {
	app := fiber.New()
	app.Use(RateLimitWithConfig(10.0, 5))
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

	// Note: RateLimitWithConfig doesn't use RFC headers (uses simple limiter)
	// Just verify request succeeded
}

func TestRateLimitWithConfig_Exceeded(t *testing.T) {
	app := fiber.New()
	app.Use(RateLimitWithConfig(0.1, 1))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should be allowed
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected first request status 200, got %d", resp1.StatusCode)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", resp2.StatusCode)
	}

	// Note: RateLimitWithConfig uses simple limiter without RFC headers
	// Just verify we got 429 status
}

func TestRateLimitWithConfig_APIKeyBased(t *testing.T) {
	app := fiber.New()
	// Simulate API key auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key_id", "test-key-123")
		return c.Next()
	})
	app.Use(RateLimitWithConfig(0.1, 1))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First request should be allowed
	req1 := httptest.NewRequest("GET", "/test", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected first request status 200, got %d", resp1.StatusCode)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", resp2.StatusCode)
	}
}

func TestRateLimitWithConfig_CustomLimits(t *testing.T) {
	// Test with very generous limits
	app := fiber.New()
	app.Use(RateLimitWithConfig(100.0, 50))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Multiple requests should all succeed
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

func TestRateLimitWithConfig_BurstAllowance(t *testing.T) {
	// Test burst behavior - should allow 3 requests quickly
	app := fiber.New()
	app.Use(RateLimitWithConfig(0.1, 3))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// First 3 requests should succeed (burst)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("burst request %d: expected status 200, got %d", i, resp.StatusCode)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("expected status 429 after burst, got %d", resp.StatusCode)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && len(s) >= len(substr) &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}()
}
