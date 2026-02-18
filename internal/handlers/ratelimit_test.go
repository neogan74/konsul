package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/ratelimit"
	"github.com/stretchr/testify/assert"
)

func setupRateLimitTestApp() (*fiber.App, *ratelimit.Service, *RateLimitHandler) {
	app := fiber.New()
	log := logger.NewFromConfig("error", "text") // Use error level to reduce test output

	// Create rate limit service
	service := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  100.0,
		Burst:           20,
		ByIP:            true,
		ByAPIKey:        true,
		CleanupInterval: 1 * time.Minute,
	})

	// Create handler
	handler := NewRateLimitHandler(service, log)

	return app, service, handler
}

func TestGetStats(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Get("/admin/ratelimit/stats", handler.GetStats)

	req := httptest.NewRequest("GET", "/admin/ratelimit/stats", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.NotNil(t, result["data"])

	data := result["data"].(map[string]interface{})
	assert.Contains(t, data, "ip_limiters")
	assert.Contains(t, data, "apikey_limiters")
}

func TestGetConfig(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Get("/admin/ratelimit/config", handler.GetConfig)

	req := httptest.NewRequest("GET", "/admin/ratelimit/config", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.NotNil(t, result["config"])

	config := result["config"].(map[string]interface{})
	assert.Equal(t, true, config["enabled"])
	assert.Equal(t, 100.0, config["requests_per_sec"])
	assert.Equal(t, float64(20), config["burst"])
	assert.Equal(t, true, config["by_ip"])
	assert.Equal(t, true, config["by_apikey"])
}

func TestResetIP(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/reset/ip/:ip", handler.ResetIP)

	// Trigger rate limit for an IP
	testIP := "192.168.1.100"
	service.AllowIP(testIP)

	// Reset the IP
	req := httptest.NewRequest("POST", "/admin/ratelimit/reset/ip/"+testIP, nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Rate limit reset successfully", result["message"])
	assert.Equal(t, testIP, result["ip"])
}

func TestResetIPMissingParam(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/reset/ip/:ip", handler.ResetIP)

	req := httptest.NewRequest("POST", "/admin/ratelimit/reset/ip/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Fiber returns 404 for missing params
}

func TestResetAPIKey(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/reset/apikey/:key_id", handler.ResetAPIKey)

	// Trigger rate limit for an API key
	testKeyID := "konsul_abc123"
	service.AllowAPIKey(testKeyID)

	// Reset the API key
	req := httptest.NewRequest("POST", "/admin/ratelimit/reset/apikey/"+testKeyID, nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Rate limit reset successfully", result["message"])
	assert.Equal(t, testKeyID, result["key_id"])
}

func TestResetAll(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/reset/all", handler.ResetAll)

	// Add some limiters
	service.AllowIP("192.168.1.100")
	service.AllowIP("192.168.1.101")
	service.AllowAPIKey("key1")

	tests := []struct {
		name        string
		queryParam  string
		message     string
		expectedMsg string
	}{
		{"reset all", "", "", "All rate limiters reset"},
		{"reset ip only", "?type=ip", "ip", "All IP rate limiters reset"},
		{"reset apikey only", "?type=apikey", "apikey", "All API key rate limiters reset"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/admin/ratelimit/reset/all"+tt.queryParam, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			assert.True(t, result["success"].(bool))
			assert.Equal(t, tt.expectedMsg, result["message"])
		})
	}
}

func TestResetAllInvalidType(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/reset/all", handler.ResetAll)

	req := httptest.NewRequest("POST", "/admin/ratelimit/reset/all?type=invalid", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "Invalid type parameter")
}

func TestGetActiveClients(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Get("/admin/ratelimit/clients", handler.GetActiveClients)

	// Add some clients
	service.AllowIP("192.168.1.100")
	service.AllowIP("192.168.1.101")
	service.AllowAPIKey("key1")

	// Give them time to be tracked
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name       string
		queryParam string
		minCount   int
	}{
		{"get all clients", "", 3},
		{"get ip clients", "?type=ip", 2},
		{"get apikey clients", "?type=apikey", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin/ratelimit/clients"+tt.queryParam, nil)
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			assert.True(t, result["success"].(bool))
			assert.NotNil(t, result["clients"])

			clients := result["clients"].([]interface{})
			assert.GreaterOrEqual(t, len(clients), tt.minCount)
		})
	}
}

func TestGetClientStatus(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Get("/admin/ratelimit/client/:identifier", handler.GetClientStatus)

	// Add a client
	testIP := "192.168.1.100"
	service.AllowIP(testIP)

	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest("GET", "/admin/ratelimit/client/"+testIP, nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.NotNil(t, result["client"])

	client := result["client"].(map[string]interface{})
	assert.Equal(t, testIP, client["identifier"])
	assert.Equal(t, "ip", client["type"])
	assert.NotZero(t, client["tokens"])
}

func TestGetClientStatusNotFound(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Get("/admin/ratelimit/client/:identifier", handler.GetClientStatus)

	req := httptest.NewRequest("GET", "/admin/ratelimit/client/nonexistent", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Equal(t, "Client not found", result["error"])
}

func TestUpdateConfig(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/config", handler.UpdateConfig)

	updateData := map[string]interface{}{
		"requests_per_sec": 200.0,
		"burst":            50,
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Contains(t, result["message"], "Configuration updated successfully")

	config := result["config"].(map[string]interface{})
	assert.Equal(t, 200.0, config["requests_per_sec"])
	assert.Equal(t, float64(50), config["burst"])
}

func TestUpdateConfigInvalidJSON(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/config", handler.UpdateConfig)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/config", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"], "Invalid request body")
}

func TestUpdateConfigInvalidValues(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/config", handler.UpdateConfig)

	tests := []struct {
		name         string
		data         map[string]interface{}
		expectedCode int
		errorMsg     string
	}{
		{
			"negative requests_per_sec",
			map[string]interface{}{"requests_per_sec": -10.0},
			fiber.StatusBadRequest,
			"requests_per_sec must be greater than 0",
		},
		{
			"zero burst",
			map[string]interface{}{"burst": 0},
			fiber.StatusBadRequest,
			"burst must be greater than 0",
		},
		{
			"negative burst",
			map[string]interface{}{"burst": -5},
			fiber.StatusBadRequest,
			"burst must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.data)
			req := httptest.NewRequest("PUT", "/admin/ratelimit/config", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			assert.False(t, result["success"].(bool))
			assert.Contains(t, result["error"], tt.errorMsg)
		})
	}
}

func TestUpdateConfigNoChanges(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/config", handler.UpdateConfig)

	// Update with same values (no change)
	updateData := map[string]interface{}{
		"requests_per_sec": 100.0, // Same as initial config
		"burst":            20,    // Same as initial config
	}
	body, _ := json.Marshal(updateData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "No changes applied", result["message"])
}

func TestGetWhitelist(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Get("/admin/ratelimit/whitelist", handler.GetWhitelist)

	// Add some whitelist entries
	_ = service.GetAccessList().AddToWhitelist(ratelimit.WhitelistEntry{
		Identifier: "192.168.1.100",
		Type:       "ip",
		Reason:     "Trusted IP",
		AddedBy:    "admin",
	})
	_ = service.GetAccessList().AddToWhitelist(ratelimit.WhitelistEntry{
		Identifier: "key123",
		Type:       "apikey",
		Reason:     "Premium customer",
		AddedBy:    "admin",
	})

	req := httptest.NewRequest("GET", "/admin/ratelimit/whitelist", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, float64(2), result["count"])
	assert.NotNil(t, result["entries"])

	entries := result["entries"].([]interface{})
	assert.Len(t, entries, 2)
}

func TestAddToWhitelist(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/whitelist", handler.AddToWhitelist)

	whitelistData := map[string]interface{}{
		"identifier": "192.168.1.200",
		"type":       "ip",
		"reason":     "VIP customer",
	}
	body, _ := json.Marshal(whitelistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/whitelist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Added to whitelist successfully", result["message"])
	assert.NotNil(t, result["entry"])

	entry := result["entry"].(map[string]interface{})
	assert.Equal(t, "192.168.1.200", entry["identifier"])
	assert.Equal(t, "ip", entry["type"])
	assert.Equal(t, "VIP customer", entry["reason"])
}

func TestAddToWhitelistWithDuration(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/whitelist", handler.AddToWhitelist)

	duration := "24h"
	whitelistData := map[string]interface{}{
		"identifier": "key456",
		"type":       "apikey",
		"reason":     "Temporary access",
		"duration":   duration,
	}
	body, _ := json.Marshal(whitelistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/whitelist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))

	entry := result["entry"].(map[string]interface{})
	assert.Equal(t, "key456", entry["identifier"])
	assert.NotNil(t, entry["expires_at"])
}

func TestAddToWhitelistInvalidType(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/whitelist", handler.AddToWhitelist)

	whitelistData := map[string]interface{}{
		"identifier": "test",
		"type":       "invalid",
		"reason":     "Test",
	}
	body, _ := json.Marshal(whitelistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/whitelist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "type must be 'ip' or 'apikey'")
}

func TestAddToWhitelistMissingIdentifier(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/whitelist", handler.AddToWhitelist)

	whitelistData := map[string]interface{}{
		"type":   "ip",
		"reason": "Test",
	}
	body, _ := json.Marshal(whitelistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/whitelist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "identifier is required")
}

func TestAddToWhitelistInvalidDuration(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/whitelist", handler.AddToWhitelist)

	duration := "invalid"
	whitelistData := map[string]interface{}{
		"identifier": "test",
		"type":       "ip",
		"reason":     "Test",
		"duration":   duration,
	}
	body, _ := json.Marshal(whitelistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/whitelist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "Invalid duration format")
}

func TestRemoveFromWhitelist(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Delete("/admin/ratelimit/whitelist/:identifier", handler.RemoveFromWhitelist)

	// Add entry first
	identifier := "192.168.1.100"
	_ = service.GetAccessList().AddToWhitelist(ratelimit.WhitelistEntry{
		Identifier: identifier,
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
	})

	req := httptest.NewRequest("DELETE", "/admin/ratelimit/whitelist/"+identifier, nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Removed from whitelist successfully", result["message"])
	assert.Equal(t, identifier, result["identifier"])

	// Verify it's removed
	assert.False(t, service.IsWhitelisted(identifier))
}

func TestRemoveFromWhitelistNotFound(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Delete("/admin/ratelimit/whitelist/:identifier", handler.RemoveFromWhitelist)

	req := httptest.NewRequest("DELETE", "/admin/ratelimit/whitelist/nonexistent", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Equal(t, "Identifier not found in whitelist", result["error"])
}

func TestGetBlacklist(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Get("/admin/ratelimit/blacklist", handler.GetBlacklist)

	// Add some blacklist entries
	_ = service.GetAccessList().AddToBlacklist(ratelimit.BlacklistEntry{
		Identifier: "192.168.1.50",
		Type:       "ip",
		Reason:     "Abuse detected",
		AddedBy:    "system",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	})
	_ = service.GetAccessList().AddToBlacklist(ratelimit.BlacklistEntry{
		Identifier: "badkey",
		Type:       "apikey",
		Reason:     "Suspicious activity",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})

	req := httptest.NewRequest("GET", "/admin/ratelimit/blacklist", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, float64(2), result["count"])
	assert.NotNil(t, result["entries"])

	entries := result["entries"].([]interface{})
	assert.Len(t, entries, 2)
}

func TestAddToBlacklist(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/blacklist", handler.AddToBlacklist)

	blacklistData := map[string]interface{}{
		"identifier": "192.168.1.99",
		"type":       "ip",
		"reason":     "Attack detected",
		"duration":   "2h",
	}
	body, _ := json.Marshal(blacklistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/blacklist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Added to blacklist successfully", result["message"])
	assert.NotNil(t, result["entry"])

	entry := result["entry"].(map[string]interface{})
	assert.Equal(t, "192.168.1.99", entry["identifier"])
	assert.Equal(t, "ip", entry["type"])
	assert.Equal(t, "Attack detected", entry["reason"])
}

func TestAddToBlacklistMissingDuration(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/blacklist", handler.AddToBlacklist)

	blacklistData := map[string]interface{}{
		"identifier": "test",
		"type":       "ip",
		"reason":     "Test",
	}
	body, _ := json.Marshal(blacklistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/blacklist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "duration is required")
}

func TestAddToBlacklistInvalidType(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Post("/admin/ratelimit/blacklist", handler.AddToBlacklist)

	blacklistData := map[string]interface{}{
		"identifier": "test",
		"type":       "invalid",
		"reason":     "Test",
		"duration":   "1h",
	}
	body, _ := json.Marshal(blacklistData)

	req := httptest.NewRequest("POST", "/admin/ratelimit/blacklist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "type must be 'ip' or 'apikey'")
}

func TestRemoveFromBlacklist(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Delete("/admin/ratelimit/blacklist/:identifier", handler.RemoveFromBlacklist)

	// Add entry first
	identifier := "192.168.1.77"
	_ = service.GetAccessList().AddToBlacklist(ratelimit.BlacklistEntry{
		Identifier: identifier,
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	})

	req := httptest.NewRequest("DELETE", "/admin/ratelimit/blacklist/"+identifier, nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Removed from blacklist successfully", result["message"])
	assert.Equal(t, identifier, result["identifier"])

	// Verify it's removed
	assert.False(t, service.IsBlacklisted(identifier))
}

func TestRemoveFromBlacklistNotFound(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Delete("/admin/ratelimit/blacklist/:identifier", handler.RemoveFromBlacklist)

	req := httptest.NewRequest("DELETE", "/admin/ratelimit/blacklist/nonexistent", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Equal(t, "Identifier not found in blacklist", result["error"])
}

func TestAdjustClientLimit(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/client/:type/:id", handler.AdjustClientLimit)

	// Create a client first
	testIP := "192.168.1.150"
	service.AllowIP(testIP)

	adjustData := map[string]interface{}{
		"rate":     50.0,
		"burst":    10,
		"duration": "1h",
	}
	body, _ := json.Marshal(adjustData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/client/ip/"+testIP, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "Rate limit adjusted successfully", result["message"])
	assert.Equal(t, "ip", result["type"])
	assert.Equal(t, testIP, result["identifier"])

	config := result["config"].(map[string]interface{})
	assert.Equal(t, 50.0, config["rate"])
	assert.Equal(t, float64(10), config["burst"])
	assert.Equal(t, "1h", config["duration"])
}

func TestAdjustClientLimitAPIKey(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/client/:type/:id", handler.AdjustClientLimit)

	// Create an API key client first
	testKey := "key789"
	service.AllowAPIKey(testKey)

	adjustData := map[string]interface{}{
		"rate":     100.0,
		"burst":    25,
		"duration": "30m",
	}
	body, _ := json.Marshal(adjustData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/client/apikey/"+testKey, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.True(t, result["success"].(bool))
	assert.Equal(t, "apikey", result["type"])
	assert.Equal(t, testKey, result["identifier"])
}

func TestAdjustClientLimitInvalidType(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/client/:type/:id", handler.AdjustClientLimit)

	adjustData := map[string]interface{}{
		"rate":     50.0,
		"burst":    10,
		"duration": "1h",
	}
	body, _ := json.Marshal(adjustData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/client/invalid/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "Invalid type")
}

func TestAdjustClientLimitInvalidRate(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/client/:type/:id", handler.AdjustClientLimit)

	testIP := "192.168.1.151"
	service.AllowIP(testIP)

	adjustData := map[string]interface{}{
		"rate":     0.0, // Invalid
		"burst":    10,
		"duration": "1h",
	}
	body, _ := json.Marshal(adjustData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/client/ip/"+testIP, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "rate must be greater than 0")
}

func TestAdjustClientLimitInvalidBurst(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/client/:type/:id", handler.AdjustClientLimit)

	testIP := "192.168.1.152"
	service.AllowIP(testIP)

	adjustData := map[string]interface{}{
		"rate":     50.0,
		"burst":    -5, // Invalid
		"duration": "1h",
	}
	body, _ := json.Marshal(adjustData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/client/ip/"+testIP, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "burst must be greater than 0")
}

func TestAdjustClientLimitInvalidDuration(t *testing.T) {
	app, service, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/client/:type/:id", handler.AdjustClientLimit)

	testIP := "192.168.1.153"
	service.AllowIP(testIP)

	adjustData := map[string]interface{}{
		"rate":     50.0,
		"burst":    10,
		"duration": "invalid", // Invalid
	}
	body, _ := json.Marshal(adjustData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/client/ip/"+testIP, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "Invalid duration format")
}

func TestAdjustClientLimitStoreNotEnabled(t *testing.T) {
	app, _, handler := setupRateLimitTestApp()
	app.Put("/admin/ratelimit/client/:type/:id", handler.AdjustClientLimit)

	// Try to adjust a client for a disabled store type
	// We need a service with only IP enabled
	serviceIPOnly := ratelimit.NewService(ratelimit.Config{
		Enabled:         true,
		RequestsPerSec:  100.0,
		Burst:           20,
		ByIP:            true,
		ByAPIKey:        false, // API key not enabled
		CleanupInterval: 1 * time.Minute,
	})
	handlerIPOnly := NewRateLimitHandler(serviceIPOnly, logger.NewFromConfig("error", "text"))

	appIPOnly := fiber.New()
	appIPOnly.Put("/admin/ratelimit/client/:type/:id", handlerIPOnly.AdjustClientLimit)

	adjustData := map[string]interface{}{
		"rate":     50.0,
		"burst":    10,
		"duration": "1h",
	}
	body, _ := json.Marshal(adjustData)

	req := httptest.NewRequest("PUT", "/admin/ratelimit/client/apikey/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := appIPOnly.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assert.False(t, result["success"].(bool))
	assert.Contains(t, result["error"].(string), "not enabled")
}
