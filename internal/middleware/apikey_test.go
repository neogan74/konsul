package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/auth"
)

func TestAPIKeyAuth_PublicPath(t *testing.T) {
	apiKeyService := auth.NewAPIKeyService("test-prefix")

	app := fiber.New()
	app.Use(APIKeyAuth(apiKeyService, []string{"/health", "/public"}))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	apiKeyService := auth.NewAPIKeyService("test-prefix")

	app := fiber.New()
	app.Use(APIKeyAuth(apiKeyService, []string{}))
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
	if !contains(string(body), "missing API key") {
		t.Errorf("expected 'missing API key' in response, got: %s", string(body))
	}
}

func TestAPIKeyAuth_ValidKey_XAPIKeyHeader(t *testing.T) {
	apiKeyService := auth.NewAPIKeyService("test-prefix")

	// Create a test API key
	expiresAt := time.Now().Add(24 * time.Hour)
	keyString, key, err := apiKeyService.GenerateAPIKey("test-key", []string{"read", "write"}, nil, &expiresAt)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	app := fiber.New()
	app.Use(APIKeyAuth(apiKeyService, []string{}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		keyID := GetAPIKeyID(c)
		keyName := GetAPIKeyName(c)
		perms := GetAPIKeyPermissions(c)
		apiKey := GetAPIKey(c)

		if keyID != key.ID {
			t.Errorf("expected keyID %q, got %q", key.ID, keyID)
		}
		if keyName != "test-key" {
			t.Errorf("expected keyName 'test-key', got %q", keyName)
		}
		if len(perms) != 2 || perms[0] != "read" || perms[1] != "write" {
			t.Errorf("expected permissions ['read', 'write'], got %v", perms)
		}
		if apiKey == nil {
			t.Error("expected apiKey to be set")
		}

		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", key.Key)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestAPIKeyAuth_ValidKey_AuthorizationHeader(t *testing.T) {
	apiKeyService := auth.NewAPIKeyService("test-prefix")

	// Create a test API key
	key, err := apiKeyService.CreateAPIKey("test-key", []string{"read"}, nil, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	app := fiber.New()
	app.Use(APIKeyAuth(apiKeyService, []string{}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Authorization", "ApiKey "+key.Key)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	apiKeyService := auth.NewAPIKeyService("test-prefix")

	app := fiber.New()
	app.Use(APIKeyAuth(apiKeyService, []string{}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "invalid API key") {
		t.Errorf("expected 'invalid API key' in response, got: %s", string(body))
	}
}

func TestAPIKeyAuth_ExpiredKey(t *testing.T) {
	apiKeyService := auth.NewAPIKeyService("test-prefix")

	// Create an expired API key
	key, err := apiKeyService.CreateAPIKey("test-key", []string{"read"}, nil, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	app := fiber.New()
	app.Use(APIKeyAuth(apiKeyService, []string{}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", key.Key)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "API key expired") {
		t.Errorf("expected 'API key expired' in response, got: %s", string(body))
	}
}

func TestAPIKeyAuth_DisabledKey(t *testing.T) {
	apiKeyService := auth.NewAPIKeyService("test-prefix")

	// Create and disable API key
	key, err := apiKeyService.CreateAPIKey("test-key", []string{"read"}, nil, 24*time.Hour)
	if err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	err = apiKeyService.DisableAPIKey(key.ID)
	if err != nil {
		t.Fatalf("failed to disable API key: %v", err)
	}

	app := fiber.New()
	app.Use(APIKeyAuth(apiKeyService, []string{}))
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Key", key.Key)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "API key disabled") {
		t.Errorf("expected 'API key disabled' in response, got: %s", string(body))
	}
}

func TestGetAPIKeyID_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		keyID := GetAPIKeyID(c)
		if keyID != "" {
			t.Errorf("expected empty keyID, got %q", keyID)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetAPIKeyName_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		keyName := GetAPIKeyName(c)
		if keyName != "" {
			t.Errorf("expected empty keyName, got %q", keyName)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetAPIKeyPermissions_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		perms := GetAPIKeyPermissions(c)
		if len(perms) != 0 {
			t.Errorf("expected empty permissions, got %v", perms)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetAPIKey_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		key := GetAPIKey(c)
		if key != nil {
			t.Errorf("expected nil key, got %v", key)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestHasPermission(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		// Set permissions in context
		c.Locals("api_key_permissions", []string{"read", "write"})

		if !HasPermission(c, "read") {
			t.Error("expected HasPermission('read') to be true")
		}
		if !HasPermission(c, "write") {
			t.Error("expected HasPermission('write') to be true")
		}
		if HasPermission(c, "delete") {
			t.Error("expected HasPermission('delete') to be false")
		}

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestHasPermission_Wildcard(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		// Set wildcard permission
		c.Locals("api_key_permissions", []string{"*"})

		if !HasPermission(c, "read") {
			t.Error("expected HasPermission('read') to be true with wildcard")
		}
		if !HasPermission(c, "write") {
			t.Error("expected HasPermission('write') to be true with wildcard")
		}
		if !HasPermission(c, "delete") {
			t.Error("expected HasPermission('delete') to be true with wildcard")
		}

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestRequirePermission_Success(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key_permissions", []string{"read", "write"})
		return c.Next()
	})
	app.Use(RequirePermission("read"))
	app.Get("/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/data", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequirePermission_Forbidden(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key_permissions", []string{"read"})
		return c.Next()
	})
	app.Use(RequirePermission("write"))
	app.Get("/data", func(c *fiber.Ctx) error {
		return c.SendString("data")
	})

	req := httptest.NewRequest("GET", "/data", nil)
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
