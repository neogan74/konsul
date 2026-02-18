package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/auth"
	"github.com/neogan74/konsul/internal/middleware"
)

type AuthHandler struct {
	jwtService    *auth.JWTService
	apiKeyService *auth.APIKeyService
}

func NewAuthHandler(jwtService *auth.JWTService, apiKeyService *auth.APIKeyService) *AuthHandler {
	return &AuthHandler{
		jwtService:    jwtService,
		apiKeyService: apiKeyService,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

// LoginResponse represents the login response body
type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshRequest represents the refresh token request body
type RefreshRequest struct {
	RefreshToken string   `json:"refresh_token"`
	Username     string   `json:"username"`
	Roles        []string `json:"roles"`
}

// Login handles user login and returns JWT tokens
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.UserID == "" || req.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id and username are required",
		})
	}

	// TODO: Add actual password validation against a user store
	// For now, this is a simple implementation that generates tokens

	// Generate access token
	token, err := h.jwtService.GenerateToken(req.UserID, req.Username, req.Roles)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate token",
		})
	}

	// Generate refresh token
	refreshToken, err := h.jwtService.GenerateRefreshToken(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate refresh token",
		})
	}

	return c.JSON(LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(15 * 60), // 15 minutes in seconds
	})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.RefreshToken == "" || req.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "refresh_token and username are required",
		})
	}

	// Refresh tokens
	newToken, newRefreshToken, err := h.jwtService.RefreshToken(req.RefreshToken, req.Username, req.Roles)
	if err != nil {
		switch err {
		case auth.ErrTokenExpired:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "refresh token expired",
			})
		case auth.ErrTokenInvalid:
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid refresh token",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to refresh token",
			})
		}
	}

	return c.JSON(LoginResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(15 * 60), // 15 minutes in seconds
	})
}

// Verify verifies the current JWT token
func (h *AuthHandler) Verify(c *fiber.Ctx) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "no valid token found",
		})
	}

	return c.JSON(fiber.Map{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"roles":    claims.Roles,
		"issuer":   claims.Issuer,
		"expires":  claims.ExpiresAt.Unix(),
	})
}

// CreateAPIKeyRequest represents the API key creation request
type CreateAPIKeyRequest struct {
	Name        string            `json:"name"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	ExpiresIn   *int64            `json:"expires_in"` // seconds
}

// CreateAPIKeyResponse represents the API key creation response
type CreateAPIKeyResponse struct {
	Key    string       `json:"key"`
	APIKey *auth.APIKey `json:"api_key"`
}

// CreateAPIKey creates a new API key
func (h *AuthHandler) CreateAPIKey(c *fiber.Ctx) error {
	var req CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	// Calculate expiration
	var expiresAt *time.Time
	if req.ExpiresIn != nil {
		exp := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Second)
		expiresAt = &exp
	}

	// Generate API key
	keyString, apiKey, err := h.apiKeyService.GenerateAPIKey(req.Name, req.Permissions, req.Metadata, expiresAt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate API key",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(CreateAPIKeyResponse{
		Key:    keyString,
		APIKey: apiKey,
	})
}

// ListAPIKeys lists all API keys
func (h *AuthHandler) ListAPIKeys(c *fiber.Ctx) error {
	keys := h.apiKeyService.ListAPIKeys()
	return c.JSON(fiber.Map{
		"keys":  keys,
		"count": len(keys),
	})
}

// GetAPIKey gets a specific API key by ID
func (h *AuthHandler) GetAPIKey(c *fiber.Ctx) error {
	keyID := c.Params("id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "key ID is required",
		})
	}

	apiKey, err := h.apiKeyService.GetAPIKey(keyID)
	if err != nil {
		if err == auth.ErrAPIKeyNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "API key not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to get API key",
		})
	}

	return c.JSON(apiKey)
}

// RevokeAPIKey revokes an API key
func (h *AuthHandler) RevokeAPIKey(c *fiber.Ctx) error {
	keyID := c.Params("id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "key ID is required",
		})
	}

	if err := h.apiKeyService.RevokeAPIKey(keyID); err != nil {
		if err == auth.ErrAPIKeyNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "API key not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to revoke API key",
		})
	}

	return c.JSON(fiber.Map{
		"message": "API key revoked successfully",
	})
}

// DeleteAPIKey deletes an API key
func (h *AuthHandler) DeleteAPIKey(c *fiber.Ctx) error {
	keyID := c.Params("id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "key ID is required",
		})
	}

	if err := h.apiKeyService.DeleteAPIKey(keyID); err != nil {
		if err == auth.ErrAPIKeyNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "API key not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete API key",
		})
	}

	return c.JSON(fiber.Map{
		"message": "API key deleted successfully",
	})
}

// UpdateAPIKeyRequest represents the API key update request
type UpdateAPIKeyRequest struct {
	Name        string            `json:"name"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	Enabled     *bool             `json:"enabled"`
}

// UpdateAPIKey updates an API key
func (h *AuthHandler) UpdateAPIKey(c *fiber.Ctx) error {
	keyID := c.Params("id")
	if keyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "key ID is required",
		})
	}

	var req UpdateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.apiKeyService.UpdateAPIKey(keyID, req.Name, req.Permissions, req.Metadata, req.Enabled); err != nil {
		if err == auth.ErrAPIKeyNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "API key not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update API key",
		})
	}

	return c.JSON(fiber.Map{
		"message": "API key updated successfully",
	})
}
