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

func setupKVHandler() (*KVHandler, *fiber.App) {
	kvStore := store.NewKVStore()
	handler := NewKVHandler(kvStore)
	app := fiber.New()

	app.Get("/kv/:key", handler.Get)
	app.Put("/kv/:key", handler.Set)
	app.Delete("/kv/:key", handler.Delete)

	return handler, app
}

func TestKVHandler_Get(t *testing.T) {
	handler, app := setupKVHandler()

	// Test non-existent key
	req := httptest.NewRequest(http.MethodGet, "/kv/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent key, got %d", resp.StatusCode)
	}

	// Set a key first
	handler.store.Set("test-key", "test-value")

	// Test existing key
	req = httptest.NewRequest(http.MethodGet, "/kv/test-key", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for existing key, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["key"] != "test-key" || result["value"] != "test-value" {
		t.Errorf("unexpected response: %+v", result)
	}
}

func TestKVHandler_Set(t *testing.T) {
	_, app := setupKVHandler()

	// Test valid set request
	body := bytes.NewReader([]byte(`{"value": "new-value"}`))
	req := httptest.NewRequest(http.MethodPut, "/kv/new-key", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for valid PUT, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["message"] != "key set" || result["key"] != "new-key" {
		t.Errorf("unexpected response: %+v", result)
	}

	// Test invalid JSON
	body = bytes.NewReader([]byte(`invalid json`))
	req = httptest.NewRequest(http.MethodPut, "/kv/test", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("PUT request with invalid JSON failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestKVHandler_Delete(t *testing.T) {
	handler, app := setupKVHandler()

	// Set a key first
	handler.store.Set("delete-key", "delete-value")

	// Test delete
	req := httptest.NewRequest(http.MethodDelete, "/kv/delete-key", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("DELETE request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for DELETE, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["message"] != "key deleted" || result["key"] != "delete-key" {
		t.Errorf("unexpected response: %+v", result)
	}

	// Verify key is deleted
	_, ok := handler.store.Get("delete-key")
	if ok {
		t.Error("key should be deleted from store")
	}
}

func TestKVHandler_NewKVHandler(t *testing.T) {
	kvStore := store.NewKVStore()
	handler := NewKVHandler(kvStore)

	if handler == nil {
		t.Fatal("expected NewKVHandler to return non-nil handler")
	}
	if handler.store != kvStore {
		t.Error("handler should reference the provided store")
	}
}