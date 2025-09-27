package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

func setupApp() *fiber.App {
	app := fiber.New()
	kv := store.NewKVStore()
	svcStore := store.NewServiceStore()

	// KV endpoints
	app.Get("/kv/:key", func(c *fiber.Ctx) error {
		key := c.Params("key")
		value, ok := kv.Get(key)
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "key not found"})
		}
		return c.JSON(fiber.Map{"key": key, "value": value})
	})

	app.Put("/kv/:key", func(c *fiber.Ctx) error {
		key := c.Params("key")
		body := struct {
			Value string `json:"value"`
		}{}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}
		kv.Set(key, body.Value)
		return c.JSON(fiber.Map{"message": "key set", "key": key})
	})

	app.Delete("/kv/:key", func(c *fiber.Ctx) error {
		key := c.Params("key")
		kv.Delete(key)
		return c.JSON(fiber.Map{"message": "key deleted", "key": key})
	})

	app.Put("/register", func(c *fiber.Ctx) error {
		var svc store.Service
		if err := c.BodyParser(&svc); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
		}
		if svc.Name == "" || svc.Address == "" || svc.Port == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing fields"})
		}
		svcStore.Register(svc)
		return c.JSON(fiber.Map{"message": "service registered", "service": svc})
	})
	app.Get("/services/", func(c *fiber.Ctx) error {
		return c.JSON(svcStore.List())
	})
	app.Get("/services/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		svc, ok := svcStore.Get(name)
		if !ok {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "service not found"})
		}
		return c.JSON(svc)
	})
	app.Delete("/deregister/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		svcStore.Deregister(name)
		return c.JSON(fiber.Map{"message": "service deregistered", "name": name})
	})
	app.Put("/heartbeat/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		if svcStore.Heartbeat(name) {
			return c.JSON(fiber.Map{"message": "heartbeat updated", "service": name})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "service not found"})
	})
	return app
}

func TestServiceDiscoveryIntegration(t *testing.T) {
	app := setupApp()

	// Register a service
	service := store.Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	body, _ := json.Marshal(service)
	req := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("register failed: %v, status: %d", err, resp.StatusCode)
	}

	// List services
	req = httptest.NewRequest(http.MethodGet, "/services/", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list failed: %v, status: %d", err, resp.StatusCode)
	}
	var services []store.Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(services) != 1 || services[0].Name != service.Name || services[0].Address != service.Address || services[0].Port != service.Port {
		t.Errorf("expected 1 service, got %+v", services)
	}

	// Get service by name
	req = httptest.NewRequest(http.MethodGet, "/services/auth", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("get by name failed: %v, status: %d", err, resp.StatusCode)
	}
	var got store.Service
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got != service {
		t.Errorf("got %+v, want %+v", got, service)
	}

	// Deregister service
	req = httptest.NewRequest(http.MethodDelete, "/deregister/auth", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("deregister failed: %v, status: %d", err, resp.StatusCode)
	}

	// Confirm service is gone
	req = httptest.NewRequest(http.MethodGet, "/services/auth", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("get after delete failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestHeartbeatIntegration(t *testing.T) {
	app := setupApp()

	// Register a service
	service := store.Service{Name: "web", Address: "10.0.0.3", Port: 80}
	body, _ := json.Marshal(service)
	req := httptest.NewRequest(http.MethodPut, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("register failed: %v, status: %d", err, resp.StatusCode)
	}

	// Send heartbeat for existing service
	req = httptest.NewRequest(http.MethodPut, "/heartbeat/web", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat failed: %v, status: %d", err, resp.StatusCode)
	}

	// Verify response
	var heartbeatResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		t.Fatalf("decode heartbeat response: %v", err)
	}
	if heartbeatResp["message"] != "heartbeat updated" || heartbeatResp["service"] != "web" {
		t.Errorf("unexpected heartbeat response: %+v", heartbeatResp)
	}

	// Send heartbeat for non-existent service
	req = httptest.NewRequest(http.MethodPut, "/heartbeat/nonexistent", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("heartbeat request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent service, got %d", resp.StatusCode)
	}
}

func TestKVStoreIntegration(t *testing.T) {
	app := setupApp()

	// Test GET on non-existent key
	req := httptest.NewRequest(http.MethodGet, "/kv/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent key, got %d", resp.StatusCode)
	}

	// Test PUT to set a key
	key := "test-key"
	value := "test-value"
	body := fmt.Sprintf(`{"value": "%s"}`, value)
	req = httptest.NewRequest(http.MethodPut, "/kv/"+key, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT request failed: %v, status: %d", err, resp.StatusCode)
	}

	// Verify PUT response
	var putResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&putResp); err != nil {
		t.Fatalf("decode PUT response: %v", err)
	}
	if putResp["message"] != "key set" || putResp["key"] != key {
		t.Errorf("unexpected PUT response: %+v", putResp)
	}

	// Test GET on existing key
	req = httptest.NewRequest(http.MethodGet, "/kv/"+key, nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET request failed: %v, status: %d", err, resp.StatusCode)
	}

	// Verify GET response
	var getResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if getResp["key"] != key || getResp["value"] != value {
		t.Errorf("expected key=%q, value=%q, got %+v", key, value, getResp)
	}

	// Test PUT to update existing key
	newValue := "updated-value"
	body = fmt.Sprintf(`{"value": "%s"}`, newValue)
	req = httptest.NewRequest(http.MethodPut, "/kv/"+key, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT update failed: %v, status: %d", err, resp.StatusCode)
	}

	// Verify updated value
	req = httptest.NewRequest(http.MethodGet, "/kv/"+key, nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET after update failed: %v, status: %d", err, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode GET response after update: %v", err)
	}
	if getResp["value"] != newValue {
		t.Errorf("expected updated value %q, got %q", newValue, getResp["value"])
	}

	// Test DELETE
	req = httptest.NewRequest(http.MethodDelete, "/kv/"+key, nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE request failed: %v, status: %d", err, resp.StatusCode)
	}

	// Verify DELETE response
	var deleteResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&deleteResp); err != nil {
		t.Fatalf("decode DELETE response: %v", err)
	}
	if deleteResp["message"] != "key deleted" || deleteResp["key"] != key {
		t.Errorf("unexpected DELETE response: %+v", deleteResp)
	}

	// Verify key is gone
	req = httptest.NewRequest(http.MethodGet, "/kv/"+key, nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("GET after delete failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestKVStoreEdgeCases(t *testing.T) {
	app := setupApp()

	// Test empty value
	req := httptest.NewRequest(http.MethodPut, "/kv/empty-key", bytes.NewReader([]byte(`{"value": ""}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT with empty value failed: %v, status: %d", err, resp.StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/kv/empty-key", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET empty value failed: %v, status: %d", err, resp.StatusCode)
	}
	var getResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&getResp)
	if getResp["value"] != "" {
		t.Errorf("expected empty value, got %q", getResp["value"])
	}

	// Test special characters in key (URL encoded)
	specialKey := "key-with-dashes_and_underscores"
	req = httptest.NewRequest(http.MethodPut, "/kv/"+url.PathEscape(specialKey), bytes.NewReader([]byte(`{"value": "special-value"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT with special key failed: %v, status: %d", err, resp.StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/kv/"+url.PathEscape(specialKey), nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET special key failed: %v, status: %d", err, resp.StatusCode)
	}

	// Test invalid JSON body
	req = httptest.NewRequest(http.MethodPut, "/kv/test", bytes.NewReader([]byte(`invalid json`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("PUT with invalid JSON failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}

	// Test missing value field
	req = httptest.NewRequest(http.MethodPut, "/kv/test", bytes.NewReader([]byte(`{"not_value": "test"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT with missing value field failed: %v, status: %d", err, resp.StatusCode)
	}
	// Missing value field should result in empty string value
	req = httptest.NewRequest(http.MethodGet, "/kv/test", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET after missing value field failed: %v, status: %d", err, resp.StatusCode)
	}

	// Test DELETE on non-existent key (should succeed)
	req = httptest.NewRequest(http.MethodDelete, "/kv/nonexistent", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE non-existent key failed: %v, status: %d", err, resp.StatusCode)
	}
}
