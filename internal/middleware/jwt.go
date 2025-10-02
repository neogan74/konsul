package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/auth"
)

// JWTAuth creates a middleware for JWT authentication
func JWTAuth(jwtService *auth.JWTService, publicPaths []string) fiber.Handler {
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

		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		token := parts[1]

		// Validate token
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			switch err {
			case auth.ErrTokenExpired:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "token expired",
				})
			case auth.ErrTokenMissing:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "token missing",
				})
			default:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "invalid token",
				})
			}
		}

		// Store claims in context
		c.Locals("user_id", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("roles", claims.Roles)
		c.Locals("claims", claims)

		return c.Next()
	}
}

// GetUserID returns the user ID from the context
func GetUserID(c *fiber.Ctx) string {
	if userID, ok := c.Locals("user_id").(string); ok {
		return userID
	}
	return ""
}

// GetUsername returns the username from the context
func GetUsername(c *fiber.Ctx) string {
	if username, ok := c.Locals("username").(string); ok {
		return username
	}
	return ""
}

// GetRoles returns the roles from the context
func GetRoles(c *fiber.Ctx) []string {
	if roles, ok := c.Locals("roles").([]string); ok {
		return roles
	}
	return []string{}
}

// GetClaims returns the JWT claims from the context
func GetClaims(c *fiber.Ctx) *auth.Claims {
	if claims, ok := c.Locals("claims").(*auth.Claims); ok {
		return claims
	}
	return nil
}

// HasRole checks if the user has a specific role
func HasRole(c *fiber.Ctx, role string) bool {
	roles := GetRoles(c)
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !HasRole(c, role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		}
		return c.Next()
	}
}
