package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/auth"
)

func setupAuthHandler() (*AuthHandler, *fiber.App) {
	jwtService := auth.NewJWTService("test-secret-key-for-testing-purposes-only", 15*time.Minute, 60*time.Minute, "konsul-test")
	apiKeyService := auth.NewAPIKeyService("konsul-test")
	handler := NewAuthHandler(jwtService, apiKeyService)

	app := fiber.New()

	// Auth routes
	app.Post("/auth/login", handler.Login)
	app.Post("/auth/refresh", handler.Refresh)
	app.Get("/auth/verify", handler.Verify)

	// API Key routes
	app.Post("/auth/api-keys", handler.CreateAPIKey)
	app.Get("/auth/api-keys", handler.ListAPIKeys)
	app.Get("/auth/api-keys/:id", handler.GetAPIKey)
	app.Put("/auth/api-keys/:id", handler.UpdateAPIKey)
	app.Post("/auth/api-keys/:id/revoke", handler.RevokeAPIKey)
	app.Delete("/auth/api-keys/:id", handler.DeleteAPIKey)

	return handler, app
}

func TestAuthHandler_Login_Success(t *testing.T) {
	_, app := setupAuthHandler()

	body := bytes.NewReader([]byte(`{"user_id": "user1", "username": "testuser", "roles": ["admin"]}`))
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Token == "" {
		t.Error("expected token in response")
	}
	if result.RefreshToken == "" {
		t.Error("expected refresh_token in response")
	}
	if result.ExpiresIn != 900 { // 15 minutes
		t.Errorf("expected expires_in to be 900, got %d", result.ExpiresIn)
	}
}

func TestAuthHandler_Login_MissingFields(t *testing.T) {
	_, app := setupAuthHandler()

	tests := []struct {
		name string
		body string
	}{
		{"missing user_id", `{"username": "testuser"}`},
		{"missing username", `{"user_id": "user1"}`},
		{"empty body", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewReader([]byte(tt.body))
			req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Login request failed: %v", err)
			}

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	_, app := setupAuthHandler()

	body := bytes.NewReader([]byte(`invalid json`))
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	_, app := setupAuthHandler()

	// First, login to get tokens
	loginBody := bytes.NewReader([]byte(`{"user_id": "user1", "username": "testuser", "roles": ["admin"]}`))
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := app.Test(loginReq)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}

	var loginResult LoginResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	// Now refresh the token
	refreshBody := bytes.NewReader([]byte(`{"refresh_token": "` + loginResult.RefreshToken + `", "username": "testuser", "roles": ["admin"]}`))
	refreshReq := httptest.NewRequest(http.MethodPost, "/auth/refresh", refreshBody)
	refreshReq.Header.Set("Content-Type", "application/json")

	refreshResp, err := app.Test(refreshReq)
	if err != nil {
		t.Fatalf("Refresh request failed: %v", err)
	}

	if refreshResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", refreshResp.StatusCode)
	}

	var refreshResult LoginResponse
	if err := json.NewDecoder(refreshResp.Body).Decode(&refreshResult); err != nil {
		t.Fatalf("failed to decode refresh response: %v", err)
	}

	if refreshResult.Token == "" {
		t.Error("expected new token in response")
	}
}

func TestAuthHandler_Refresh_MissingFields(t *testing.T) {
	_, app := setupAuthHandler()

	tests := []struct {
		name string
		body string
	}{
		{"missing refresh_token", `{"username": "testuser"}`},
		{"missing username", `{"refresh_token": "some-token"}`},
		{"empty body", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewReader([]byte(tt.body))
			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", body)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Refresh request failed: %v", err)
			}

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	_, app := setupAuthHandler()

	body := bytes.NewReader([]byte(`{"refresh_token": "invalid-token", "username": "testuser"}`))
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Refresh request failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_CreateAPIKey_Success(t *testing.T) {
	_, app := setupAuthHandler()

	body := bytes.NewReader([]byte(`{"name": "test-key", "permissions": ["read", "write"]}`))
	req := httptest.NewRequest(http.MethodPost, "/auth/api-keys", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result CreateAPIKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Key == "" {
		t.Error("expected key in response")
	}
	if result.APIKey == nil {
		t.Error("expected api_key in response")
	}
	if result.APIKey.Name != "test-key" {
		t.Errorf("expected name 'test-key', got '%s'", result.APIKey.Name)
	}
}

func TestAuthHandler_CreateAPIKey_MissingName(t *testing.T) {
	_, app := setupAuthHandler()

	body := bytes.NewReader([]byte(`{"permissions": ["read"]}`))
	req := httptest.NewRequest(http.MethodPost, "/auth/api-keys", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_CreateAPIKey_WithExpiration(t *testing.T) {
	_, app := setupAuthHandler()

	body := bytes.NewReader([]byte(`{"name": "expiring-key", "permissions": ["read"], "expires_in": 3600}`))
	req := httptest.NewRequest(http.MethodPost, "/auth/api-keys", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result CreateAPIKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.APIKey.ExpiresAt == nil {
		t.Error("expected expiration time to be set")
	}
}

func TestAuthHandler_ListAPIKeys(t *testing.T) {
	_, app := setupAuthHandler()

	// Create a few API keys first
	for i := 0; i < 3; i++ {
		body := bytes.NewReader([]byte(`{"name": "key-` + string(rune('a'+i)) + `"}`))
		req := httptest.NewRequest(http.MethodPost, "/auth/api-keys", body)
		req.Header.Set("Content-Type", "application/json")
		app.Test(req)
	}

	// List all keys
	req := httptest.NewRequest(http.MethodGet, "/auth/api-keys", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ListAPIKeys request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["keys"]; !ok {
		t.Error("expected 'keys' in response")
	}
	if _, ok := result["count"]; !ok {
		t.Error("expected 'count' in response")
	}
}

func TestAuthHandler_GetAPIKey_Success(t *testing.T) {
	_, app := setupAuthHandler()

	// Create an API key first
	createBody := bytes.NewReader([]byte(`{"name": "get-test-key"}`))
	createReq := httptest.NewRequest(http.MethodPost, "/auth/api-keys", createBody)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, _ := app.Test(createReq)
	var createResult CreateAPIKeyResponse
	json.NewDecoder(createResp.Body).Decode(&createResult)

	// Get the API key
	req := httptest.NewRequest(http.MethodGet, "/auth/api-keys/"+createResult.APIKey.ID, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GetAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_GetAPIKey_NotFound(t *testing.T) {
	_, app := setupAuthHandler()

	req := httptest.NewRequest(http.MethodGet, "/auth/api-keys/nonexistent-id", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GetAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_UpdateAPIKey_Success(t *testing.T) {
	_, app := setupAuthHandler()

	// Create an API key first
	createBody := bytes.NewReader([]byte(`{"name": "update-test-key"}`))
	createReq := httptest.NewRequest(http.MethodPost, "/auth/api-keys", createBody)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, _ := app.Test(createReq)
	var createResult CreateAPIKeyResponse
	json.NewDecoder(createResp.Body).Decode(&createResult)

	// Update the API key
	updateBody := bytes.NewReader([]byte(`{"name": "updated-key-name", "permissions": ["admin"]}`))
	req := httptest.NewRequest(http.MethodPut, "/auth/api-keys/"+createResult.APIKey.ID, updateBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("UpdateAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_UpdateAPIKey_NotFound(t *testing.T) {
	_, app := setupAuthHandler()

	body := bytes.NewReader([]byte(`{"name": "new-name"}`))
	req := httptest.NewRequest(http.MethodPut, "/auth/api-keys/nonexistent-id", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("UpdateAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_RevokeAPIKey_Success(t *testing.T) {
	_, app := setupAuthHandler()

	// Create an API key first
	createBody := bytes.NewReader([]byte(`{"name": "revoke-test-key"}`))
	createReq := httptest.NewRequest(http.MethodPost, "/auth/api-keys", createBody)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, _ := app.Test(createReq)
	var createResult CreateAPIKeyResponse
	json.NewDecoder(createResp.Body).Decode(&createResult)

	// Revoke the API key
	req := httptest.NewRequest(http.MethodPost, "/auth/api-keys/"+createResult.APIKey.ID+"/revoke", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("RevokeAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_RevokeAPIKey_NotFound(t *testing.T) {
	_, app := setupAuthHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/api-keys/nonexistent-id/revoke", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("RevokeAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAuthHandler_DeleteAPIKey_Success(t *testing.T) {
	_, app := setupAuthHandler()

	// Create an API key first
	createBody := bytes.NewReader([]byte(`{"name": "delete-test-key"}`))
	createReq := httptest.NewRequest(http.MethodPost, "/auth/api-keys", createBody)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, _ := app.Test(createReq)
	var createResult CreateAPIKeyResponse
	json.NewDecoder(createResp.Body).Decode(&createResult)

	// Delete the API key
	req := httptest.NewRequest(http.MethodDelete, "/auth/api-keys/"+createResult.APIKey.ID, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("DeleteAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify it's deleted
	getReq := httptest.NewRequest(http.MethodGet, "/auth/api-keys/"+createResult.APIKey.ID, nil)
	getResp, _ := app.Test(getReq)
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected key to be deleted, got status %d", getResp.StatusCode)
	}
}

func TestAuthHandler_DeleteAPIKey_NotFound(t *testing.T) {
	_, app := setupAuthHandler()

	req := httptest.NewRequest(http.MethodDelete, "/auth/api-keys/nonexistent-id", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("DeleteAPIKey request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}
