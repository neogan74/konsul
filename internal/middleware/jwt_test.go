package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/auth"
)

func TestJWTAuth_PublicPath(t *testing.T) {
	// Create JWT service
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")

	// Create app
	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{"/health", "/public"}))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Test public path without token
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestJWTAuth_PublicPathPrefix(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{"/admin/"}))
	app.Get("/admin/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/admin/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{"/health"}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "missing authorization header") {
		t.Errorf("expected 'missing authorization header' in response, got: %s", string(body))
	}
}

func TestJWTAuth_InvalidHeaderFormat(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")

	testCases := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "token123"},
		{"wrong prefix", "Basic token123"},
		{"missing token", "Bearer"},
		{"extra parts", "Bearer token extra"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(JWTAuth(jwtService, []string{}))
			app.Get("/api/data", func(c *fiber.Ctx) error {
				return c.SendString("data")
			})

			req := httptest.NewRequest("GET", "/api/data", nil)
			req.Header.Set("Authorization", tc.header)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Errorf("expected status 401, got %d", resp.StatusCode)
			}
		})
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")

	// Generate valid token
	token, err := jwtService.GenerateToken("user123", "testuser", []string{"admin", "user"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		username := GetUsername(c)
		roles := GetRoles(c)
		claims := GetClaims(c)

		if userID != "user123" {
			t.Errorf("expected userID 'user123', got %q", userID)
		}
		if username != "testuser" {
			t.Errorf("expected username 'testuser', got %q", username)
		}
		if len(roles) != 2 || roles[0] != "admin" || roles[1] != "user" {
			t.Errorf("expected roles ['admin', 'user'], got %v", roles)
		}
		if claims == nil {
			t.Error("expected claims to be set")
		}

		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "invalid token") {
		t.Errorf("expected 'invalid token' in response, got: %s", string(body))
	}
}

func TestGetUserID_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		userID := GetUserID(c)
		if userID != "" {
			t.Errorf("expected empty userID, got %q", userID)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetUsername_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		username := GetUsername(c)
		if username != "" {
			t.Errorf("expected empty username, got %q", username)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetRoles_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		roles := GetRoles(c)
		if len(roles) != 0 {
			t.Errorf("expected empty roles, got %v", roles)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetClaims_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		claims := GetClaims(c)
		if claims != nil {
			t.Errorf("expected nil claims, got %v", claims)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestHasRole(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		// Set roles in context
		c.Locals("roles", []string{"admin", "user"})

		if !HasRole(c, "admin") {
			t.Error("expected HasRole('admin') to be true")
		}
		if !HasRole(c, "user") {
			t.Error("expected HasRole('user') to be true")
		}
		if HasRole(c, "superuser") {
			t.Error("expected HasRole('superuser') to be false")
		}

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestRequireRole_Success(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		// Set roles in context
		c.Locals("roles", []string{"admin", "user"})
		return c.Next()
	})
	app.Use(RequireRole("admin"))
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("admin area")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		// Set roles in context
		c.Locals("roles", []string{"user"})
		return c.Next()
	})
	app.Use(RequireRole("admin"))
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendString("admin area")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "insufficient permissions") {
		t.Errorf("expected 'insufficient permissions' in response, got: %s", string(body))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
