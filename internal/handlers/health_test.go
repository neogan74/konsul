package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

func TestHealthHandler_Check(t *testing.T) {
	app := fiber.New()

	// Create stores
	kvStore := store.NewKVStore()
	serviceStore := store.NewServiceStore()

	// Add some test data
	kvStore.Set("test-key", "test-value")
	if err := serviceStore.Register(store.Service{
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}); err != nil {
		t.Fatalf("register service: %v", err)
	}
	healthHandler := NewHealthHandler(kvStore, serviceStore, "1.0.0-test")
	app.Get("/health", healthHandler.Check)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)

	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.Version != "1.0.0-test" {
		t.Errorf("Expected version '1.0.0-test', got '%s'", health.Version)
	}

	if health.KVStore.Total != 1 {
		t.Errorf("Expected 1 KV key, got %d", health.KVStore.Total)
	}

	if health.Services.Total != 1 {
		t.Errorf("Expected 1 service, got %d", health.Services.Total)
	}

	if health.Services.Active != 1 {
		t.Errorf("Expected 1 active service, got %d", health.Services.Active)
	}
}

func TestHealthHandler_Liveness(t *testing.T) {
	app := fiber.New()

	kvStore := store.NewKVStore()
	serviceStore := store.NewServiceStore()
	healthHandler := NewHealthHandler(kvStore, serviceStore, "1.0.0-test")

	app.Get("/health/live", healthHandler.Liveness)

	req := httptest.NewRequest("GET", "/health/live", nil)
	resp, err := app.Test(req, -1)

	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "alive" {
		t.Errorf("Expected status 'alive', got '%s'", result["status"])
	}
}

func TestHealthHandler_Readiness(t *testing.T) {
	app := fiber.New()

	kvStore := store.NewKVStore()
	serviceStore := store.NewServiceStore()
	healthHandler := NewHealthHandler(kvStore, serviceStore, "1.0.0-test")

	app.Get("/health/ready", healthHandler.Readiness)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	resp, err := app.Test(req, -1)

	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", result["status"])
	}
}
