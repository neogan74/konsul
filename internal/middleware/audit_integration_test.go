package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/audit"
	"github.com/neogan74/konsul/internal/logger"
)

// TestAuditIntegration_KVOperations verifies audit events for KV operations
func TestAuditIntegration_KVOperations(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")

	mgr, err := audit.NewManager(audit.Config{
		Enabled:       true,
		Sink:          "file",
		FilePath:      auditPath,
		BufferSize:    10,
		FlushInterval: 10 * time.Millisecond,
		DropPolicy:    audit.DropPolicyBlock,
	}, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create audit manager: %v", err)
	}
	defer func() {
		if err := mgr.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown audit manager: %v", err)
		}
	}()

	app := fiber.New()

	// Setup KV routes with audit middleware
	kvRoutes := app.Group("/kv")

	// Set up mock auth middleware to populate locals before audit middleware
	kvRoutes.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "test-user-123")
		c.Locals("jwt_auth", true)
		return c.Next()
	})

	kvRoutes.Use(AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "kv",
		ActionMapper: KVActionMapper,
	}))

	// Mock handlers
	kvRoutes.Put("/:key", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	kvRoutes.Get("/:key", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"value": "test"})
	})
	kvRoutes.Delete("/:key", func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedAction string
		expectedStatus int
	}{
		{
			name:           "kv_set",
			method:         "PUT",
			path:           "/kv/testkey",
			body:           `{"value":"test"}`,
			expectedAction: "kv.set",
			expectedStatus: 200,
		},
		{
			name:           "kv_get",
			method:         "GET",
			path:           "/kv/testkey",
			body:           "",
			expectedAction: "kv.get",
			expectedStatus: 200,
		},
		{
			name:           "kv_delete",
			method:         "DELETE",
			path:           "/kv/testkey",
			body:           "",
			expectedAction: "kv.delete",
			expectedStatus: 204,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Fatalf("close response body: %v", err)
				}
			}()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}

	// Give time for async flush
	time.Sleep(50 * time.Millisecond)

	// Shutdown to flush remaining events
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown audit manager: %v", err)
	}

	// Read and verify audit log
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	content := string(data)

	// Verify each operation was logged
	if !strings.Contains(content, `"action":"kv.set"`) {
		t.Errorf("audit log missing action 'kv.set', got: %s", content)
	}
	// Note: GET without explicit list operation may be classified as kv.get or kv.list depending on implementation
	if !strings.Contains(content, `"action":"kv.get"`) && !strings.Contains(content, `"action":"kv.list"`) {
		t.Errorf("audit log missing action 'kv.get' or 'kv.list', got: %s", content)
	}
	if !strings.Contains(content, `"action":"kv.delete"`) {
		t.Errorf("audit log missing action 'kv.delete', got: %s", content)
	}

	// Verify common fields
	if !strings.Contains(content, `"resource":{"type":"kv"`) {
		t.Errorf("audit log missing resource type")
	}
	if !strings.Contains(content, `"result":"success"`) {
		t.Errorf("audit log missing success result")
	}
}

// TestAuditIntegration_ServiceOperations verifies audit events for service operations
func TestAuditIntegration_ServiceOperations(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")

	mgr, err := audit.NewManager(audit.Config{
		Enabled:       true,
		Sink:          "file",
		FilePath:      auditPath,
		BufferSize:    10,
		FlushInterval: 10 * time.Millisecond,
		DropPolicy:    audit.DropPolicyBlock,
	}, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create audit manager: %v", err)
	}
	defer func() {
		if err := mgr.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown audit manager: %v", err)
		}
	}()

	app := fiber.New()

	// Setup service routes with audit middleware
	app.Post("/register", AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "service",
		ActionMapper: ServiceActionMapper,
	}), func(c *fiber.Ctx) error {
		return c.SendStatus(201)
	})

	app.Delete("/deregister/:name", AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "service",
		ActionMapper: ServiceActionMapper,
	}), func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})

	app.Put("/heartbeat/:name", AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "service",
		ActionMapper: ServiceActionMapper,
	}), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedAction string
		expectedStatus int
	}{
		{
			name:           "service_register",
			method:         "POST",
			path:           "/register",
			expectedAction: "service.register",
			expectedStatus: 201,
		},
		{
			name:           "service_deregister",
			method:         "DELETE",
			path:           "/deregister/web",
			expectedAction: "service.deregister",
			expectedStatus: 204,
		},
		{
			name:           "service_heartbeat",
			method:         "PUT",
			path:           "/heartbeat/web?heartbeat=true",
			expectedAction: "service.heartbeat",
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Fatalf("close response body: %v", err)
				}
			}()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}

	// Flush and shutdown
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown audit manager: %v", err)
	}

	// Verify audit log
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	content := string(data)

	// Verify actions
	expectedActions := []string{"service.register", "service.deregister", "service.heartbeat"}
	for _, action := range expectedActions {
		if !strings.Contains(content, `"action":"`+action+`"`) {
			t.Errorf("audit log missing action %q", action)
		}
	}
}

// TestAuditIntegration_ACLOperations verifies audit events for ACL operations
func TestAuditIntegration_ACLOperations(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")

	mgr, err := audit.NewManager(audit.Config{
		Enabled:       true,
		Sink:          "file",
		FilePath:      auditPath,
		BufferSize:    10,
		FlushInterval: 10 * time.Millisecond,
		DropPolicy:    audit.DropPolicyBlock,
	}, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create audit manager: %v", err)
	}
	defer func() {
		if err := mgr.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown audit manager: %v", err)
		}
	}()

	app := fiber.New()

	aclRoutes := app.Group("/acl")
	aclRoutes.Use(AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "acl",
		ActionMapper: ACLActionMapper,
	}))

	aclRoutes.Post("/token", func(c *fiber.Ctx) error {
		return c.SendStatus(201)
	})
	aclRoutes.Delete("/token/:id", func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})
	aclRoutes.Post("/policy", func(c *fiber.Ctx) error {
		return c.SendStatus(201)
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedAction string
	}{
		{
			name:           "token_create",
			method:         "POST",
			path:           "/acl/token",
			expectedAction: "acl.token.create",
		},
		{
			name:           "token_revoke",
			method:         "DELETE",
			path:           "/acl/token/abc123",
			expectedAction: "acl.token.revoke",
		},
		{
			name:           "policy_create",
			method:         "POST",
			path:           "/acl/policy",
			expectedAction: "acl.policy.create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Fatalf("close response body: %v", err)
				}
			}()
		})
	}

	// Flush and shutdown
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown audit manager: %v", err)
	}

	// Verify audit log
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	content := string(data)

	expectedActions := []string{"acl.token.create", "acl.token.revoke", "acl.policy.create"}
	for _, action := range expectedActions {
		if !strings.Contains(content, `"action":"`+action+`"`) {
			t.Errorf("audit log missing action %q", action)
		}
	}
}

// TestAuditIntegration_BackupOperations verifies audit events for backup operations
func TestAuditIntegration_BackupOperations(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")

	mgr, err := audit.NewManager(audit.Config{
		Enabled:       true,
		Sink:          "file",
		FilePath:      auditPath,
		BufferSize:    10,
		FlushInterval: 10 * time.Millisecond,
		DropPolicy:    audit.DropPolicyBlock,
	}, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create audit manager: %v", err)
	}
	defer func() {
		if err := mgr.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown audit manager: %v", err)
		}
	}()

	app := fiber.New()

	backupRoutes := app.Group("")
	backupRoutes.Use(AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "backup",
		ActionMapper: BackupActionMapper,
	}))

	backupRoutes.Post("/backup", func(c *fiber.Ctx) error {
		return c.SendStatus(201)
	})
	backupRoutes.Put("/restore", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	backupRoutes.Get("/export", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedAction string
	}{
		{
			name:           "backup_create",
			method:         "POST",
			path:           "/backup",
			expectedAction: "backup.create",
		},
		{
			name:           "backup_restore",
			method:         "PUT",
			path:           "/restore",
			expectedAction: "backup.restore",
		},
		{
			name:           "backup_list",
			method:         "GET",
			path:           "/export",
			expectedAction: "backup.list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Fatalf("close response body: %v", err)
				}
			}()
		})
	}

	// Flush and shutdown
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown audit manager: %v", err)
	}

	// Verify audit log
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	content := string(data)

	expectedActions := []string{"backup.create", "backup.restore", "backup.list"}
	for _, action := range expectedActions {
		if !strings.Contains(content, `"action":"`+action+`"`) {
			t.Errorf("audit log missing action %q", action)
		}
	}
}

// TestAuditIntegration_DisabledManager verifies no events when audit is disabled
func TestAuditIntegration_DisabledManager(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")

	// Create disabled manager
	mgr, err := audit.NewManager(audit.Config{
		Enabled: false,
	}, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create audit manager: %v", err)
	}
	defer func() {
		if err := mgr.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown audit manager: %v", err)
		}
	}()

	app := fiber.New()

	app.Use(AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "kv",
		ActionMapper: KVActionMapper,
	}))

	app.Put("/kv/:key", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("PUT", "/kv/test", strings.NewReader(`{"value":"test"}`))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close response body: %v", err)
		}
	}()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Wait a bit to ensure no async writes
	time.Sleep(50 * time.Millisecond)

	// Verify audit file was not created
	if _, err := os.Stat(auditPath); !os.IsNotExist(err) {
		t.Error("audit file should not exist when manager is disabled")
	}
}

// TestAuditIntegration_EventFields verifies all expected fields are captured
func TestAuditIntegration_EventFields(t *testing.T) {
	dir := t.TempDir()
	auditPath := filepath.Join(dir, "audit.log")

	mgr, err := audit.NewManager(audit.Config{
		Enabled:       true,
		Sink:          "file",
		FilePath:      auditPath,
		BufferSize:    10,
		FlushInterval: 10 * time.Millisecond,
		DropPolicy:    audit.DropPolicyBlock,
	}, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create audit manager: %v", err)
	}
	defer func() {
		if err := mgr.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown audit manager: %v", err)
		}
	}()

	app := fiber.New()

	// Mock auth middleware to populate locals before audit middleware
	app.Use(func(c *fiber.Ctx) error {
		// Simulate authenticated user
		c.Locals("user_id", "user-456")
		c.Locals("username", "john.doe")
		c.Locals("roles", []string{"admin", "developer"})
		c.Locals("jwt_auth", true)
		c.Locals("trace_id", "trace-123")
		c.Locals("span_id", "span-456")
		return c.Next()
	})

	app.Use(AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "kv",
		ActionMapper: KVActionMapper,
	}))

	app.Put("/kv/:key", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("PUT", "/kv/testdatabase", strings.NewReader(`{"url":"postgres://..."}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close response body: %v", err)
		}
	}()

	// Flush and shutdown
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown audit manager: %v", err)
	}

	// Read audit log
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	// Parse JSON event
	var event audit.Event
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		t.Fatal("no audit events found")
	}

	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatalf("failed to parse audit event: %v", err)
	}

	// Verify all fields
	if event.Action != "kv.set" {
		t.Errorf("expected action 'kv.set', got %q", event.Action)
	}
	if event.Result != "success" {
		t.Errorf("expected result 'success', got %q", event.Result)
	}
	if event.Resource.Type != "kv" {
		t.Errorf("expected resource type 'kv', got %q", event.Resource.Type)
	}
	if event.Resource.ID != "testdatabase" {
		t.Errorf("expected resource ID 'testdatabase', got %q", event.Resource.ID)
	}
	if event.Actor.ID != "user-456" {
		t.Errorf("expected actor ID 'user-456', got %q", event.Actor.ID)
	}
	if event.Actor.Name != "john.doe" {
		t.Errorf("expected actor name 'john.doe', got %q", event.Actor.Name)
	}
	if event.Actor.Type != "user" {
		t.Errorf("expected actor type 'user', got %q", event.Actor.Type)
	}
	if event.AuthMethod != "jwt" {
		t.Errorf("expected auth method 'jwt', got %q", event.AuthMethod)
	}
	if event.HTTPMethod != "PUT" {
		t.Errorf("expected HTTP method 'PUT', got %q", event.HTTPMethod)
	}
	if event.HTTPPath != "/kv/testdatabase" {
		t.Errorf("expected HTTP path '/kv/testdatabase', got %q", event.HTTPPath)
	}
	if event.HTTPStatus != 200 {
		t.Errorf("expected HTTP status 200, got %d", event.HTTPStatus)
	}
	if event.TraceID != "trace-123" {
		t.Errorf("expected trace ID 'trace-123', got %q", event.TraceID)
	}
	if event.SpanID != "span-456" {
		t.Errorf("expected span ID 'span-456', got %q", event.SpanID)
	}
	if event.RequestHash == "" {
		t.Error("expected request hash to be set")
	}
	if event.ID == "" {
		t.Error("expected event ID to be set")
	}
	if event.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}
