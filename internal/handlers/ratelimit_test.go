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
	json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
			json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
			json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

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
			json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

	assert.True(t, result["success"].(bool))
	assert.Equal(t, "No changes applied", result["message"])
}
