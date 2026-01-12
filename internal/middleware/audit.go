package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/audit"
)

// AuditConfig holds configuration for the audit middleware.
type AuditConfig struct {
	Manager      *audit.Manager
	ResourceType string                  // The type of resource being accessed (kv, service, acl, etc.)
	ActionMapper func(*fiber.Ctx) string // Optional function to derive action from request
}

// AuditMiddleware creates middleware that records audit events for HTTP requests.
// It captures the request details before execution and records the outcome after.
func AuditMiddleware(cfg AuditConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if audit is disabled
		if cfg.Manager == nil || !cfg.Manager.Enabled() {
			return c.Next()
		}

		// Determine the action
		action := deriveAction(c, cfg.ResourceType, cfg.ActionMapper)

		// Process the request
		err := c.Next()

		// Record audit event with the final status
		statusCode := c.Response().StatusCode()
		_ = audit.RecordHTTPEvent(c.Context(), cfg.Manager, c, action, cfg.ResourceType, statusCode)

		return err
	}
}

// deriveAction determines the audit action string from the HTTP request.
func deriveAction(c *fiber.Ctx, resourceType string, mapper func(*fiber.Ctx) string) string {
	// Use custom mapper if provided
	if mapper != nil {
		return mapper(c)
	}

	// Default action mapping based on HTTP method and resource type
	method := c.Method()
	switch method {
	case "POST":
		return resourceType + ".create"
	case "PUT":
		return resourceType + ".update"
	case "DELETE":
		return resourceType + ".delete"
	case "GET":
		// Distinguish between list and read
		if c.Params("key") != "" || c.Params("id") != "" || c.Params("name") != "" {
			return resourceType + ".read"
		}
		return resourceType + ".list"
	case "PATCH":
		return resourceType + ".modify"
	default:
		return resourceType + "." + method
	}
}

// KVActionMapper provides specific action mapping for KV operations.
func KVActionMapper(c *fiber.Ctx) string {
	method := c.Method()
	key := c.Params("key")
	if key == "" {
		key = c.Params("*")
	}

	switch method {
	case "PUT":
		return "kv.set"
	case "DELETE":
		return "kv.delete"
	case "GET":
		if key != "" {
			return "kv.get"
		}
		return "kv.list"
	default:
		return "kv." + method
	}
}

// ServiceActionMapper provides specific action mapping for service operations.
func ServiceActionMapper(c *fiber.Ctx) string {
	method := c.Method()

	// Check for deregistration endpoint
	if method == "DELETE" {
		return "service.deregister"
	}

	// Check for heartbeat/health update endpoint
	if method == "PUT" && (c.Query("heartbeat") == "true" || c.Params("health") != "") {
		return "service.heartbeat"
	}

	// Check for registration
	if method == "POST" {
		return "service.register"
	}

	// Read operations
	if method == "GET" {
		if c.Params("name") != "" || c.Params("id") != "" {
			return "service.get"
		}
		return "service.list"
	}

	return "service." + method
}

// ACLActionMapper provides specific action mapping for ACL operations.
func ACLActionMapper(c *fiber.Ctx) string {
	method := c.Method()
	path := c.Path()

	// Token operations
	if auditContains(path, "/token") {
		switch method {
		case "POST":
			return "acl.token.create"
		case "DELETE":
			return "acl.token.revoke"
		case "GET":
			return "acl.token.read"
		default:
			return "acl.token." + method
		}
	}

	// Policy operations
	if auditContains(path, "/policy") {
		switch method {
		case "POST":
			return "acl.policy.create"
		case "PUT":
			return "acl.policy.update"
		case "DELETE":
			return "acl.policy.delete"
		case "GET":
			return "acl.policy.read"
		default:
			return "acl.policy." + method
		}
	}

	return "acl." + method
}

// BackupActionMapper provides specific action mapping for backup operations.
func BackupActionMapper(c *fiber.Ctx) string {
	method := c.Method()
	switch method {
	case "POST":
		return "backup.create"
	case "PUT":
		return "backup.restore"
	case "DELETE":
		return "backup.delete"
	case "GET":
		if c.Params("id") != "" {
			return "backup.download"
		}
		return "backup.list"
	default:
		return "backup." + method
	}
}

// AdminActionMapper provides specific action mapping for admin operations.
func AdminActionMapper(c *fiber.Ctx) string {
	path := c.Path()
	method := c.Method()

	if auditContains(path, "/ratelimit") {
		switch method {
		case "POST":
			return "admin.ratelimit.create"
		case "PUT":
			return "admin.ratelimit.update"
		case "DELETE":
			return "admin.ratelimit.delete"
		case "GET":
			return "admin.ratelimit.get"
		default:
			return "admin.ratelimit." + method
		}
	}

	if auditContains(path, "/config") {
		return "admin.config." + method
	}

	return "admin." + method
}

// auditContains checks if a string contains a substring (helper for action mappers).
func auditContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || hasSubstringAudit(s, substr)))
}

func hasSubstringAudit(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
