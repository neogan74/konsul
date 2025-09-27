package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

func setupServiceHandler() (*ServiceHandler, *fiber.App) {
	serviceStore := store.NewServiceStore()
	handler := NewServiceHandler(serviceStore)
	app := fiber.New()

	app.Put("/register", handler.Register)
	app.Get("/services/", handler.List)
	app.Get("/services/:name", handler.Get)
	app.Delete("/deregister/:name", handler.Deregister)
	app.Put("/heartbeat/:name", handler.Heartbeat)

	return handler, app
}

func TestServiceHandler_Register(t *testing.T) {
	_, app := setupServiceHandler()

	// Test valid registration
	service := store.Service{Name: "test-service", Address: "127.0.0.1", Port: 8080}
	body, _ := json.Marshal(service)
	req := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for valid registration, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "service registered" {
		t.Errorf("unexpected response: %+v", result)
	}

	// Test invalid JSON
	req = httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("register with invalid JSON failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}

	// Test missing fields
	invalidService := store.Service{Name: "", Address: "127.0.0.1", Port: 8080}
	body, _ = json.Marshal(invalidService)
	req = httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("register with missing fields failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", resp.StatusCode)
	}
}

func TestServiceHandler_List(t *testing.T) {
	handler, app := setupServiceHandler()

	// Register some services
	service1 := store.Service{Name: "service1", Address: "127.0.0.1", Port: 8080}
	service2 := store.Service{Name: "service2", Address: "127.0.0.2", Port: 8081}
	handler.store.Register(service1)
	handler.store.Register(service2)

	// Test list
	req := httptest.NewRequest(http.MethodGet, "/services/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for list, got %d", resp.StatusCode)
	}

	var services []store.Service
	json.NewDecoder(resp.Body).Decode(&services)
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}

func TestServiceHandler_Get(t *testing.T) {
	handler, app := setupServiceHandler()

	// Test non-existent service
	req := httptest.NewRequest(http.MethodGet, "/services/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent service, got %d", resp.StatusCode)
	}

	// Register a service
	service := store.Service{Name: "test-service", Address: "127.0.0.1", Port: 8080}
	handler.store.Register(service)

	// Test existing service
	req = httptest.NewRequest(http.MethodGet, "/services/test-service", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for existing service, got %d", resp.StatusCode)
	}

	var result store.Service
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != service.Name || result.Address != service.Address || result.Port != service.Port {
		t.Errorf("expected %+v, got %+v", service, result)
	}
}

func TestServiceHandler_Deregister(t *testing.T) {
	handler, app := setupServiceHandler()

	// Register a service
	service := store.Service{Name: "test-service", Address: "127.0.0.1", Port: 8080}
	handler.store.Register(service)

	// Test deregister
	req := httptest.NewRequest(http.MethodDelete, "/deregister/test-service", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("deregister request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for deregister, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "service deregistered" || result["name"] != "test-service" {
		t.Errorf("unexpected response: %+v", result)
	}

	// Verify service is removed
	_, ok := handler.store.Get("test-service")
	if ok {
		t.Error("service should be removed from store")
	}
}

func TestServiceHandler_Heartbeat(t *testing.T) {
	handler, app := setupServiceHandler()

	// Test heartbeat on non-existent service
	req := httptest.NewRequest(http.MethodPut, "/heartbeat/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("heartbeat request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent service, got %d", resp.StatusCode)
	}

	// Register a service
	service := store.Service{Name: "test-service", Address: "127.0.0.1", Port: 8080}
	handler.store.Register(service)

	// Test successful heartbeat
	req = httptest.NewRequest(http.MethodPut, "/heartbeat/test-service", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("heartbeat request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for successful heartbeat, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "heartbeat updated" || result["service"] != "test-service" {
		t.Errorf("unexpected response: %+v", result)
	}
}

func TestServiceHandler_NewServiceHandler(t *testing.T) {
	serviceStore := store.NewServiceStore()
	handler := NewServiceHandler(serviceStore)

	if handler == nil {
		t.Fatal("expected NewServiceHandler to return non-nil handler")
	}
	if handler.store != serviceStore {
		t.Error("handler should reference the provided store")
	}
}