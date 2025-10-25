package middleware

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/auth"
	"github.com/neogan74/konsul/internal/logger"
)

// setupTestApp creates a Fiber app with JWT and ACL middleware for testing
func setupTestApp(jwtService *auth.JWTService, evaluator *acl.Evaluator, resourceType acl.ResourceType, capability acl.Capability) *fiber.App {
	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Use(ACLMiddleware(evaluator, resourceType, capability))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})
	app.Get("/kv/:key", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})
	app.Get("/services/:name", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})
	return app
}

func TestACLMiddleware_NoClaims(t *testing.T) {
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	app := fiber.New()
	// No JWT middleware, so no claims will be set
	app.Use(ACLMiddleware(evaluator, acl.ResourceTypeKV, acl.CapabilityRead))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "unauthorized") {
		t.Errorf("expected 'unauthorized' in response, got: %s", string(body))
	}
}

func TestACLMiddleware_NoPolicies(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	// Generate token with no policies
	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := setupTestApp(jwtService, evaluator, acl.ResourceTypeKV, acl.CapabilityRead)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !contains(string(body), "no policies attached") {
		t.Errorf("expected 'no policies attached' in response, got: %s", string(body))
	}
}

func TestACLMiddleware_Denied(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	// Create a policy that only allows read on different path
	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "app/other/*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	// Generate token with the policy
	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"test-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := setupTestApp(jwtService, evaluator, acl.ResourceTypeKV, acl.CapabilityRead)

	// Request to /test which doesn't match the policy
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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

func TestACLMiddleware_Allowed(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	// Create a policy that allows read on all KV
	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	// Generate token with the policy
	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"test-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := setupTestApp(jwtService, evaluator, acl.ResourceTypeKV, acl.CapabilityRead)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestACLMiddleware_KVResource(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	// Create a policy with KV rules - use single segment key
	policy := &acl.Policy{
		Name:        "kv-policy",
		Description: "KV policy",
		KV: []acl.KVRule{
			{
				Path:         "mykey",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"kv-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Create a custom app with route parameter for KV
	// Note: ACL middleware must be attached to the route, not globally, so params are available
	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Get("/kv/:key",
		ACLMiddleware(evaluator, acl.ResourceTypeKV, acl.CapabilityRead),
		func(c *fiber.Ctx) error {
			return c.SendString("success")
		},
	)

	// Should succeed for matching path
	req := httptest.NewRequest("GET", "/kv/mykey", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestACLMiddleware_ServiceResource(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	// Create a policy with service rules
	policy := &acl.Policy{
		Name:        "service-policy",
		Description: "Service policy",
		Service: []acl.ServiceRule{
			{
				Name:         "myservice",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"service-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Create a custom app with route parameter for service
	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Use(ACLMiddleware(evaluator, acl.ResourceTypeService, acl.CapabilityRead))
	app.Get("/services/:name", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	// Should succeed for matching service
	req := httptest.NewRequest("GET", "/services/myservice", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestACLMiddleware_HealthResource(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	policy := &acl.Policy{
		Name:        "health-policy",
		Description: "Health policy",
		Health: []acl.HealthRule{
			{
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"health-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := setupTestApp(jwtService, evaluator, acl.ResourceTypeHealth, acl.CapabilityRead)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestACLMiddleware_BackupResource(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	policy := &acl.Policy{
		Name:        "backup-policy",
		Description: "Backup policy",
		Backup: []acl.BackupRule{
			{
				Capabilities: []acl.Capability{acl.CapabilityCreate},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"backup-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := setupTestApp(jwtService, evaluator, acl.ResourceTypeBackup, acl.CapabilityCreate)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestACLMiddleware_AdminResource(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	policy := &acl.Policy{
		Name:        "admin-policy",
		Description: "Admin policy",
		Admin: []acl.AdminRule{
			{
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"admin-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := setupTestApp(jwtService, evaluator, acl.ResourceTypeAdmin, acl.CapabilityRead)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestACLMiddleware_ContextStorage(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"test-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Use(ACLMiddleware(evaluator, acl.ResourceTypeKV, acl.CapabilityRead))
	app.Get("/test", func(c *fiber.Ctx) error {
		// Check that resource and capability are stored in context
		resource := GetACLResource(c)
		if resource == nil {
			t.Error("expected resource to be set in context")
		} else if resource.Type != acl.ResourceTypeKV {
			t.Errorf("expected resource type KV, got %s", resource.Type)
		}

		capability := GetACLCapability(c)
		if capability != acl.CapabilityRead {
			t.Errorf("expected capability read, got %s", capability)
		}

		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDynamicACLMiddleware_NoClaims(t *testing.T) {
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	app := fiber.New()
	app.Use(DynamicACLMiddleware(evaluator))
	app.Get("/kv/test", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/kv/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestDynamicACLMiddleware_NoPolicies(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Use(DynamicACLMiddleware(evaluator))
	app.Get("/kv/test", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/kv/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestDynamicACLMiddleware_Denied(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "app/other/*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"test-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Use(DynamicACLMiddleware(evaluator))
	app.Get("/kv/:key", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/kv/app/config/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestDynamicACLMiddleware_Allowed(t *testing.T) {
	jwtService := auth.NewJWTService("test-secret", 15*time.Minute, 24*time.Hour, "konsul")
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "app/**",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("failed to add policy: %v", err)
	}

	token, err := jwtService.GenerateTokenWithPolicies("user123", "testuser", []string{"user"}, []string{"test-policy"})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	app := fiber.New()
	app.Use(JWTAuth(jwtService, []string{}))
	app.Use(DynamicACLMiddleware(evaluator))
	app.Get("/kv/:key", func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/kv/app/config/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestInferResourceAndCapability_KV(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		method           string
		expectedResource acl.ResourceType
		expectedCap      acl.Capability
		expectedPath     string
	}{
		{
			name:             "GET /kv/mykey - read",
			path:             "/kv/mykey",
			method:           "GET",
			expectedResource: acl.ResourceTypeKV,
			expectedCap:      acl.CapabilityRead,
			expectedPath:     "mykey",
		},
		{
			name:             "GET /kv/ - list",
			path:             "/kv/",
			method:           "GET",
			expectedResource: acl.ResourceTypeKV,
			expectedCap:      acl.CapabilityList,
			expectedPath:     "*",
		},
		{
			name:             "GET /kv - list",
			path:             "/kv",
			method:           "GET",
			expectedResource: acl.ResourceTypeKV,
			expectedCap:      acl.CapabilityList,
			expectedPath:     "*",
		},
		{
			name:             "PUT /kv/mykey - write",
			path:             "/kv/mykey",
			method:           "PUT",
			expectedResource: acl.ResourceTypeKV,
			expectedCap:      acl.CapabilityWrite,
			expectedPath:     "mykey",
		},
		{
			name:             "POST /kv/mykey - write",
			path:             "/kv/mykey",
			method:           "POST",
			expectedResource: acl.ResourceTypeKV,
			expectedCap:      acl.CapabilityWrite,
			expectedPath:     "mykey",
		},
		{
			name:             "DELETE /kv/mykey - delete",
			path:             "/kv/mykey",
			method:           "DELETE",
			expectedResource: acl.ResourceTypeKV,
			expectedCap:      acl.CapabilityDelete,
			expectedPath:     "mykey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Add(tt.method, tt.path, func(c *fiber.Ctx) error {
				resource, capability := inferResourceAndCapability(c)
				if resource.Type != tt.expectedResource {
					t.Errorf("expected resource type %s, got %s", tt.expectedResource, resource.Type)
				}
				if capability != tt.expectedCap {
					t.Errorf("expected capability %s, got %s", tt.expectedCap, capability)
				}
				if resource.Path != tt.expectedPath {
					t.Errorf("expected path %s, got %s", tt.expectedPath, resource.Path)
				}
				return c.SendString("ok")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestInferResourceAndCapability_Service(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		method           string
		expectedResource acl.ResourceType
		expectedCap      acl.Capability
		expectedName     string
	}{
		{
			name:             "GET /services/web - read",
			path:             "/services/web",
			method:           "GET",
			expectedResource: acl.ResourceTypeService,
			expectedCap:      acl.CapabilityRead,
			expectedName:     "web",
		},
		{
			name:             "GET /services/ - list",
			path:             "/services/",
			method:           "GET",
			expectedResource: acl.ResourceTypeService,
			expectedCap:      acl.CapabilityList,
			expectedName:     "*",
		},
		{
			name:             "POST /register - register",
			path:             "/register",
			method:           "POST",
			expectedResource: acl.ResourceTypeService,
			expectedCap:      acl.CapabilityRegister,
			expectedName:     "*",
		},
		{
			name:             "DELETE /deregister/web - deregister",
			path:             "/deregister/web",
			method:           "DELETE",
			expectedResource: acl.ResourceTypeService,
			expectedCap:      acl.CapabilityDeregister,
			expectedName:     "web",
		},
		{
			name:             "PUT /heartbeat/web - write",
			path:             "/heartbeat/web",
			method:           "PUT",
			expectedResource: acl.ResourceTypeService,
			expectedCap:      acl.CapabilityWrite,
			expectedName:     "web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Add(tt.method, tt.path, func(c *fiber.Ctx) error {
				resource, capability := inferResourceAndCapability(c)
				if resource.Type != tt.expectedResource {
					t.Errorf("expected resource type %s, got %s", tt.expectedResource, resource.Type)
				}
				if capability != tt.expectedCap {
					t.Errorf("expected capability %s, got %s", tt.expectedCap, capability)
				}
				if resource.Path != tt.expectedName {
					t.Errorf("expected service name %s, got %s", tt.expectedName, resource.Path)
				}
				return c.SendString("ok")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestInferResourceAndCapability_Health(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		method           string
		expectedResource acl.ResourceType
		expectedCap      acl.Capability
	}{
		{
			name:             "GET /health - read",
			path:             "/health",
			method:           "GET",
			expectedResource: acl.ResourceTypeHealth,
			expectedCap:      acl.CapabilityRead,
		},
		{
			name:             "PUT /health - write",
			path:             "/health",
			method:           "PUT",
			expectedResource: acl.ResourceTypeHealth,
			expectedCap:      acl.CapabilityWrite,
		},
		{
			name:             "POST /health - write",
			path:             "/health",
			method:           "POST",
			expectedResource: acl.ResourceTypeHealth,
			expectedCap:      acl.CapabilityWrite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Add(tt.method, tt.path, func(c *fiber.Ctx) error {
				resource, capability := inferResourceAndCapability(c)
				if resource.Type != tt.expectedResource {
					t.Errorf("expected resource type %s, got %s", tt.expectedResource, resource.Type)
				}
				if capability != tt.expectedCap {
					t.Errorf("expected capability %s, got %s", tt.expectedCap, capability)
				}
				return c.SendString("ok")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestInferResourceAndCapability_Backup(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		method           string
		expectedResource acl.ResourceType
		expectedCap      acl.Capability
	}{
		{
			name:             "POST /backup - create",
			path:             "/backup",
			method:           "POST",
			expectedResource: acl.ResourceTypeBackup,
			expectedCap:      acl.CapabilityCreate,
		},
		{
			name:             "POST /restore - restore",
			path:             "/restore",
			method:           "POST",
			expectedResource: acl.ResourceTypeBackup,
			expectedCap:      acl.CapabilityRestore,
		},
		{
			name:             "GET /export - export",
			path:             "/export",
			method:           "GET",
			expectedResource: acl.ResourceTypeBackup,
			expectedCap:      acl.CapabilityExport,
		},
		{
			name:             "POST /import - import",
			path:             "/import",
			method:           "POST",
			expectedResource: acl.ResourceTypeBackup,
			expectedCap:      acl.CapabilityImport,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Add(tt.method, tt.path, func(c *fiber.Ctx) error {
				resource, capability := inferResourceAndCapability(c)
				if resource.Type != tt.expectedResource {
					t.Errorf("expected resource type %s, got %s", tt.expectedResource, resource.Type)
				}
				if capability != tt.expectedCap {
					t.Errorf("expected capability %s, got %s", tt.expectedCap, capability)
				}
				return c.SendString("ok")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestInferResourceAndCapability_Admin(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		method           string
		expectedResource acl.ResourceType
		expectedCap      acl.Capability
	}{
		{
			name:             "GET /acl/policies - read",
			path:             "/acl/policies",
			method:           "GET",
			expectedResource: acl.ResourceTypeAdmin,
			expectedCap:      acl.CapabilityRead,
		},
		{
			name:             "POST /acl/policies - write",
			path:             "/acl/policies",
			method:           "POST",
			expectedResource: acl.ResourceTypeAdmin,
			expectedCap:      acl.CapabilityWrite,
		},
		{
			name:             "GET /metrics - read",
			path:             "/metrics",
			method:           "GET",
			expectedResource: acl.ResourceTypeAdmin,
			expectedCap:      acl.CapabilityRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Add(tt.method, tt.path, func(c *fiber.Ctx) error {
				resource, capability := inferResourceAndCapability(c)
				if resource.Type != tt.expectedResource {
					t.Errorf("expected resource type %s, got %s", tt.expectedResource, resource.Type)
				}
				if capability != tt.expectedCap {
					t.Errorf("expected capability %s, got %s", tt.expectedCap, capability)
				}
				return c.SendString("ok")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestInferResourceAndCapability_Default(t *testing.T) {
	app := fiber.New()
	app.Get("/unknown/path", func(c *fiber.Ctx) error {
		resource, capability := inferResourceAndCapability(c)
		if resource.Type != acl.ResourceTypeAdmin {
			t.Errorf("expected default resource type admin, got %s", resource.Type)
		}
		if capability != acl.CapabilityDeny {
			t.Errorf("expected default capability deny, got %s", capability)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/unknown/path", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetACLResource_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		resource := GetACLResource(c)
		if resource != nil {
			t.Errorf("expected nil resource, got %v", resource)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}

func TestGetACLCapability_NoContext(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		capability := GetACLCapability(c)
		if capability != "" {
			t.Errorf("expected empty capability, got %s", capability)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
}
