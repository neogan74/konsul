package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/auth"
)

// APIKeyAuth creates a middleware for API key authentication
func APIKeyAuth(apiKeyService *auth.APIKeyService, publicPaths []string) fiber.Handler {
	// Create a map for faster path lookup
	publicPathMap := make(map[string]bool)
	for _, path := range publicPaths {
		publicPathMap[path] = true
	}

	return func(c *fiber.Ctx) error {
		// Check if path is public
		if publicPathMap[c.Path()] {
			return c.Next()
		}

		// Try to get API key from X-API-Key header first
		apiKey := c.Get("X-API-Key")

		// If not found, try Authorization header with "ApiKey" scheme
		if apiKey == "" {
			authHeader := c.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "ApiKey" {
					apiKey = parts[1]
				}
			}
		}

		// If still no API key found, return unauthorized
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing API key",
			})
		}

		// Validate API key
		key, err := apiKeyService.ValidateAPIKey(apiKey)
		if err != nil {
			switch err {
			case auth.ErrAPIKeyExpired:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "API key expired",
				})
			case auth.ErrAPIKeyDisabled:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "API key disabled",
				})
			case auth.ErrAPIKeyNotFound:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "invalid API key",
				})
			default:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "API key validation failed",
				})
			}
		}

		// Store API key info in context
		c.Locals("api_key_id", key.ID)
		c.Locals("api_key_name", key.Name)
		c.Locals("api_key_permissions", key.Permissions)
		c.Locals("api_key_metadata", key.Metadata)
		c.Locals("api_key", key)

		return c.Next()
	}
}

// GetAPIKeyID returns the API key ID from the context
func GetAPIKeyID(c *fiber.Ctx) string {
	if id, ok := c.Locals("api_key_id").(string); ok {
		return id
	}
	return ""
}

// GetAPIKeyName returns the API key name from the context
func GetAPIKeyName(c *fiber.Ctx) string {
	if name, ok := c.Locals("api_key_name").(string); ok {
		return name
	}
	return ""
}

// GetAPIKeyPermissions returns the API key permissions from the context
func GetAPIKeyPermissions(c *fiber.Ctx) []string {
	if perms, ok := c.Locals("api_key_permissions").([]string); ok {
		return perms
	}
	return []string{}
}

// GetAPIKey returns the full API key from the context
func GetAPIKey(c *fiber.Ctx) *auth.APIKey {
	if key, ok := c.Locals("api_key").(*auth.APIKey); ok {
		return key
	}
	return nil
}

// HasPermission checks if the API key has a specific permission
func HasPermission(c *fiber.Ctx, permission string) bool {
	perms := GetAPIKeyPermissions(c)
	for _, p := range perms {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// RequirePermission creates a middleware that requires a specific permission
func RequirePermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !HasPermission(c, permission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		}
		return c.Next()
	}
}
