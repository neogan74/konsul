package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/auth"
)

// ACLMiddleware creates a middleware that enforces ACL policies
func ACLMiddleware(evaluator *acl.Evaluator, resourceType acl.ResourceType, capability acl.Capability) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get policies from JWT claims
		claims := GetClaims(c)
		if claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// If no policies are attached, deny access
		if len(claims.Policies) == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "no policies attached to token",
			})
		}

		// Determine the resource being accessed
		var resource acl.Resource
		switch resourceType {
		case acl.ResourceTypeKV:
			key := c.Params("key")
			if key == "" {
				key = "*" // List all keys
			}
			resource = acl.NewKVResource(key)
		case acl.ResourceTypeService:
			name := c.Params("name")
			if name == "" {
				name = "*" // List all services
			}
			resource = acl.NewServiceResource(name)
		case acl.ResourceTypeHealth:
			resource = acl.NewHealthResource()
		case acl.ResourceTypeBackup:
			resource = acl.NewBackupResource()
		case acl.ResourceTypeAdmin:
			resource = acl.NewAdminResource()
		}

		// Evaluate ACL policies
		allowed := evaluator.Evaluate(claims.Policies, resource, capability)
		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":      "forbidden",
				"message":    "insufficient permissions",
				"resource":   string(resourceType),
				"capability": string(capability),
			})
		}

		// Store resource in context for logging
		c.Locals("acl_resource", resource)
		c.Locals("acl_capability", capability)

		return c.Next()
	}
}

// DynamicACLMiddleware creates a middleware that dynamically determines resource and capability
func DynamicACLMiddleware(evaluator *acl.Evaluator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get policies from JWT claims
		claims := GetClaims(c)
		if claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// If no policies are attached, deny access
		if len(claims.Policies) == 0 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "no policies attached to token",
			})
		}

		// Determine resource and capability from the request
		resource, capability := inferResourceAndCapability(c)

		// Evaluate ACL policies
		allowed := evaluator.Evaluate(claims.Policies, resource, capability)
		if !allowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":      "forbidden",
				"message":    "insufficient permissions",
				"resource":   string(resource.Type),
				"capability": string(capability),
				"path":       resource.Path,
			})
		}

		// Store resource in context for logging
		c.Locals("acl_resource", resource)
		c.Locals("acl_capability", capability)

		return c.Next()
	}
}

// inferResourceAndCapability determines the resource and capability from the request
func inferResourceAndCapability(c *fiber.Ctx) (acl.Resource, acl.Capability) {
	path := c.Path()
	method := c.Method()

	// KV store endpoints
	if strings.HasPrefix(path, "/kv/") || path == "/kv" {
		key := c.Params("key")
		if key == "" {
			key = "*"
		}
		resource := acl.NewKVResource(key)

		switch method {
		case "GET":
			if key == "*" || path == "/kv/" || path == "/kv" {
				return resource, acl.CapabilityList
			}
			return resource, acl.CapabilityRead
		case "PUT", "POST":
			return resource, acl.CapabilityWrite
		case "DELETE":
			return resource, acl.CapabilityDelete
		}
	}

	// Service endpoints
	if strings.HasPrefix(path, "/services/") || strings.HasPrefix(path, "/register") ||
		strings.HasPrefix(path, "/deregister/") || strings.HasPrefix(path, "/heartbeat/") {

		serviceName := c.Params("name")
		if serviceName == "" {
			serviceName = "*"
		}
		resource := acl.NewServiceResource(serviceName)

		switch {
		case strings.HasPrefix(path, "/register"):
			return resource, acl.CapabilityRegister
		case strings.HasPrefix(path, "/deregister/"):
			return resource, acl.CapabilityDeregister
		case strings.HasPrefix(path, "/heartbeat/"):
			return resource, acl.CapabilityWrite
		case method == "GET":
			if serviceName == "*" || path == "/services/" {
				return resource, acl.CapabilityList
			}
			return resource, acl.CapabilityRead
		}
	}

	// Health endpoints
	if strings.HasPrefix(path, "/health") {
		resource := acl.NewHealthResource()
		switch method {
		case "GET":
			return resource, acl.CapabilityRead
		case "PUT", "POST":
			return resource, acl.CapabilityWrite
		}
	}

	// Backup endpoints
	if strings.HasPrefix(path, "/backup") || strings.HasPrefix(path, "/restore") ||
		strings.HasPrefix(path, "/export") || strings.HasPrefix(path, "/import") {

		resource := acl.NewBackupResource()
		switch {
		case strings.HasPrefix(path, "/backup"):
			return resource, acl.CapabilityCreate
		case strings.HasPrefix(path, "/restore"):
			return resource, acl.CapabilityRestore
		case strings.HasPrefix(path, "/export"):
			return resource, acl.CapabilityExport
		case strings.HasPrefix(path, "/import"):
			return resource, acl.CapabilityImport
		}
	}

	// Admin endpoints (ACL management, metrics, etc.)
	if strings.HasPrefix(path, "/acl/") || strings.HasPrefix(path, "/metrics") {
		resource := acl.NewAdminResource()
		switch method {
		case "GET":
			return resource, acl.CapabilityRead
		case "PUT", "POST", "DELETE":
			return resource, acl.CapabilityWrite
		}
	}

	// Default: deny
	return acl.Resource{Type: acl.ResourceTypeAdmin}, acl.CapabilityDeny
}

// GetACLResource returns the ACL resource from context
func GetACLResource(c *fiber.Ctx) *acl.Resource {
	if resource, ok := c.Locals("acl_resource").(acl.Resource); ok {
		return &resource
	}
	return nil
}

// GetACLCapability returns the ACL capability from context
func GetACLCapability(c *fiber.Ctx) acl.Capability {
	if cap, ok := c.Locals("acl_capability").(acl.Capability); ok {
		return cap
	}
	return ""
}
