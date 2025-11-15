package middleware

import (
	"context"
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

func TestAuditMiddleware_DisabledManager(t *testing.T) {
	app := fiber.New()

	// Disabled audit manager
	mgr, _ := audit.NewManager(audit.Config{Enabled: false}, logger.GetDefault())

	app.Use(AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "kv",
	}))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestAuditMiddleware_RecordsEvent(t *testing.T) {
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
	defer mgr.Shutdown(context.Background())

	app := fiber.New()

	app.Use(AuditMiddleware(AuditConfig{
		Manager:      mgr,
		ResourceType: "kv",
		ActionMapper: KVActionMapper,
	}))

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "test-user")
		c.Locals("jwt_auth", true)
		return c.Next()
	})

	app.Put("/kv/:key", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("PUT", "/kv/config/app", strings.NewReader(`{"value":"test"}`))
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Give time for async flush
	time.Sleep(50 * time.Millisecond)

	// Shutdown to flush remaining events
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	// Read audit log
	data, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	content := string(data)

	// Verify event was recorded
	if !strings.Contains(content, "\"action\":\"kv.set\"") {
		t.Errorf("audit log missing action, got: %s", content)
	}
	if !strings.Contains(content, "\"result\":\"success\"") {
		t.Errorf("audit log missing result, got: %s", content)
	}
	if !strings.Contains(content, "\"http_method\":\"PUT\"") {
		t.Errorf("audit log missing HTTP method, got: %s", content)
	}
}

func TestDeriveAction_DefaultMapping(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		route        string
		resourceType string
		expected     string
	}{
		{"create", "POST", "/service", "/service", "service", "service.create"},
		{"update", "PUT", "/kv", "/kv", "kv", "kv.update"},
		{"delete", "DELETE", "/acl", "/acl", "acl", "acl.delete"},
		{"read", "GET", "/kv/test-id", "/kv/:key", "kv", "kv.read"},
		{"list", "GET", "/service", "/service", "service", "service.list"},
		{"modify", "PATCH", "/config", "/config", "config", "config.modify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Add(tt.method, tt.route, func(c *fiber.Ctx) error {
				action := deriveAction(c, tt.resourceType, nil)
				if action != tt.expected {
					t.Errorf("expected action %q, got %q", tt.expected, action)
				}
				return c.SendStatus(200)
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			app.Test(req)
		})
	}
}

func TestKVActionMapper(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		uri      string
		expected string
	}{
		{"set", "PUT", "/kv/test", "kv.set"},
		{"delete", "DELETE", "/kv/test", "kv.delete"},
		{"get", "GET", "/kv/test", "kv.get"},
		{"list", "GET", "/kv", "kv.list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Add(tt.method, "/kv/:key?", func(c *fiber.Ctx) error {
				action := KVActionMapper(c)
				if action != tt.expected {
					t.Errorf("expected action %q, got %q", tt.expected, action)
				}
				return c.SendStatus(200)
			})

			req := httptest.NewRequest(tt.method, tt.uri, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestServiceActionMapper(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		uri      string
		query    string
		expected string
	}{
		{"register", "POST", "/service", "", "service.register"},
		{"deregister", "DELETE", "/service/web", "", "service.deregister"},
		{"heartbeat", "PUT", "/service/web", "heartbeat=true", "service.heartbeat"},
		{"get", "GET", "/service/web", "", "service.get"},
		{"list", "GET", "/service", "", "service.list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Add(tt.method, "/service/:name?", func(c *fiber.Ctx) error {
				action := ServiceActionMapper(c)
				if action != tt.expected {
					t.Errorf("expected action %q, got %q", tt.expected, action)
				}
				return c.SendStatus(200)
			})

			uri := tt.uri
			if tt.query != "" {
				uri += "?" + tt.query
			}

			req := httptest.NewRequest(tt.method, uri, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestACLActionMapper(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		uri      string
		expected string
	}{
		{"token_create", "POST", "/acl/token", "acl.token.create"},
		{"token_revoke", "DELETE", "/acl/token/abc123", "acl.token.revoke"},
		{"token_read", "GET", "/acl/token/abc123", "acl.token.read"},
		{"policy_create", "POST", "/acl/policy", "acl.policy.create"},
		{"policy_update", "PUT", "/acl/policy/admin", "acl.policy.update"},
		{"policy_delete", "DELETE", "/acl/policy/admin", "acl.policy.delete"},
		{"policy_read", "GET", "/acl/policy/admin", "acl.policy.read"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Add(tt.method, "/acl/*", func(c *fiber.Ctx) error {
				action := ACLActionMapper(c)
				if action != tt.expected {
					t.Errorf("expected action %q, got %q", tt.expected, action)
				}
				return c.SendStatus(200)
			})

			req := httptest.NewRequest(tt.method, tt.uri, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

func TestBackupActionMapper(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		uri      string
		expected string
	}{
		{"create", "POST", "/backup", "backup.create"},
		{"restore", "PUT", "/backup/123", "backup.restore"},
		{"delete", "DELETE", "/backup/123", "backup.delete"},
		{"download", "GET", "/backup/123", "backup.download"},
		{"list", "GET", "/backup", "backup.list"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Add(tt.method, "/backup/:id?", func(c *fiber.Ctx) error {
				action := BackupActionMapper(c)
				if action != tt.expected {
					t.Errorf("expected action %q, got %q", tt.expected, action)
				}
				return c.SendStatus(200)
			})

			req := httptest.NewRequest(tt.method, tt.uri, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}
