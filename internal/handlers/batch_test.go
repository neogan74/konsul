package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

func setupBatchTestApp() (*fiber.App, *BatchHandler) {
	app := fiber.New()
	kvStore := store.NewKVStore()
	serviceStore := store.NewServiceStore()
	handler := NewBatchHandler(kvStore, serviceStore, nil)
	return app, handler
}

func TestBatchKVGet_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/get", handler.BatchKVGet)

	// Set up test data
	handler.kvStore.Set("key1", "value1")
	handler.kvStore.Set("key2", "value2")
	handler.kvStore.Set("key3", "value3")

	// Test batch get
	reqBody := BatchKVGetRequest{
		Keys: []string{"key1", "key2", "nonexistent"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/get", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchKVGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Found) != 2 {
		t.Errorf("Expected 2 found keys, got %d", len(result.Found))
	}
	if result.Found["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %s", result.Found["key1"])
	}
	if result.Found["key2"] != "value2" {
		t.Errorf("Expected key2=value2, got %s", result.Found["key2"])
	}
	if len(result.NotFound) != 1 {
		t.Errorf("Expected 1 not found key, got %d", len(result.NotFound))
	}
	if result.NotFound[0] != "nonexistent" {
		t.Errorf("Expected 'nonexistent' in not found, got %s", result.NotFound[0])
	}
}

func TestBatchKVGet_EmptyKeys(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/get", handler.BatchKVGet)

	reqBody := BatchKVGetRequest{Keys: []string{}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/get", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchKVSet_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/set", handler.BatchKVSet)

	reqBody := BatchKVSetRequest{
		Items: map[string]string{
			"batch_key1": "batch_value1",
			"batch_key2": "batch_value2",
			"batch_key3": "batch_value3",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/set", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchKVSetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Expected count 3, got %d", result.Count)
	}

	// Verify keys were actually set
	val1, ok1 := handler.kvStore.Get("batch_key1")
	val2, ok2 := handler.kvStore.Get("batch_key2")
	val3, ok3 := handler.kvStore.Get("batch_key3")

	if !ok1 || val1 != "batch_value1" {
		t.Error("batch_key1 not set correctly")
	}
	if !ok2 || val2 != "batch_value2" {
		t.Error("batch_key2 not set correctly")
	}
	if !ok3 || val3 != "batch_value3" {
		t.Error("batch_key3 not set correctly")
	}
}

func TestBatchKVSet_EmptyItems(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/set", handler.BatchKVSet)

	reqBody := BatchKVSetRequest{Items: map[string]string{}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/set", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchKVDelete_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/delete", handler.BatchKVDelete)

	// Set up test data
	handler.kvStore.Set("del_key1", "value1")
	handler.kvStore.Set("del_key2", "value2")
	handler.kvStore.Set("del_key3", "value3")
	handler.kvStore.Set("keep_key", "keep_value")

	reqBody := BatchKVDeleteRequest{
		Keys: []string{"del_key1", "del_key2", "del_key3"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchKVDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Expected count 3, got %d", result.Count)
	}

	// Verify keys were deleted
	_, ok1 := handler.kvStore.Get("del_key1")
	_, ok2 := handler.kvStore.Get("del_key2")
	_, ok3 := handler.kvStore.Get("del_key3")
	_, okKeep := handler.kvStore.Get("keep_key")

	if ok1 {
		t.Error("del_key1 should have been deleted")
	}
	if ok2 {
		t.Error("del_key2 should have been deleted")
	}
	if ok3 {
		t.Error("del_key3 should have been deleted")
	}
	if !okKeep {
		t.Error("keep_key should not have been deleted")
	}
}

func TestBatchKVDelete_EmptyKeys(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/delete", handler.BatchKVDelete)

	reqBody := BatchKVDeleteRequest{Keys: []string{}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchServiceRegister_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/register", handler.BatchServiceRegister)

	reqBody := BatchServiceRegisterRequest{
		Services: []store.Service{
			{
				Name:    "service1",
				Address: "127.0.0.1",
				Port:    8001,
				Tags:    []string{"web", "api"},
			},
			{
				Name:    "service2",
				Address: "127.0.0.2",
				Port:    8002,
				Tags:    []string{"db"},
			},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/services/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchServiceRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Expected count 2, got %d", result.Count)
	}
	if len(result.Failed) != 0 {
		t.Errorf("Expected 0 failed, got %d", len(result.Failed))
	}

	// Verify services were registered
	svc1, ok1 := handler.serviceStore.Get("service1")
	svc2, ok2 := handler.serviceStore.Get("service2")

	if !ok1 {
		t.Error("service1 not registered")
	}
	if !ok2 {
		t.Error("service2 not registered")
	}
	if svc1.Port != 8001 {
		t.Errorf("service1 port incorrect: got %d", svc1.Port)
	}
	if svc2.Port != 8002 {
		t.Errorf("service2 port incorrect: got %d", svc2.Port)
	}
}

func TestBatchServiceRegister_WithFailures(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/register", handler.BatchServiceRegister)

	reqBody := BatchServiceRegisterRequest{
		Services: []store.Service{
			{
				Name:    "valid_service",
				Address: "127.0.0.1",
				Port:    8001,
			},
			{
				Name:    "", // Invalid: no name
				Address: "127.0.0.2",
				Port:    8002,
			},
			{
				Name:    "no_address",
				Address: "", // Invalid: no address
				Port:    8003,
			},
			{
				Name:    "invalid_port",
				Address: "127.0.0.4",
				Port:    -1, // Invalid port
			},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/services/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchServiceRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count != 1 {
		t.Errorf("Expected 1 registered, got %d", result.Count)
	}
	if len(result.Failed) != 3 {
		t.Errorf("Expected 3 failed, got %d", len(result.Failed))
	}

	// Verify only valid service was registered
	_, ok := handler.serviceStore.Get("valid_service")
	if !ok {
		t.Error("valid_service should have been registered")
	}
}

func TestBatchServiceRegister_EmptyServices(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/register", handler.BatchServiceRegister)

	reqBody := BatchServiceRegisterRequest{Services: []store.Service{}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/services/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchServiceDeregister_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/deregister", handler.BatchServiceDeregister)

	// Register some services first
	handler.serviceStore.Register(store.Service{Name: "svc1", Address: "127.0.0.1", Port: 8001})
	handler.serviceStore.Register(store.Service{Name: "svc2", Address: "127.0.0.2", Port: 8002})
	handler.serviceStore.Register(store.Service{Name: "svc3", Address: "127.0.0.3", Port: 8003})
	handler.serviceStore.Register(store.Service{Name: "keep", Address: "127.0.0.4", Port: 8004})

	reqBody := BatchServiceDeregisterRequest{
		Names: []string{"svc1", "svc2", "svc3"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/services/deregister", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchServiceDeregisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Expected count 3, got %d", result.Count)
	}

	// Verify services were deregistered
	_, ok1 := handler.serviceStore.Get("svc1")
	_, ok2 := handler.serviceStore.Get("svc2")
	_, ok3 := handler.serviceStore.Get("svc3")
	_, okKeep := handler.serviceStore.Get("keep")

	if ok1 {
		t.Error("svc1 should have been deregistered")
	}
	if ok2 {
		t.Error("svc2 should have been deregistered")
	}
	if ok3 {
		t.Error("svc3 should have been deregistered")
	}
	if !okKeep {
		t.Error("keep should not have been deregistered")
	}
}

func TestBatchServiceDeregister_EmptyNames(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/deregister", handler.BatchServiceDeregister)

	reqBody := BatchServiceDeregisterRequest{Names: []string{}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/services/deregister", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchServiceGet_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/get", handler.BatchServiceGet)

	// Register some services
	handler.serviceStore.Register(store.Service{Name: "web", Address: "127.0.0.1", Port: 8001})
	handler.serviceStore.Register(store.Service{Name: "api", Address: "127.0.0.2", Port: 8002})
	handler.serviceStore.Register(store.Service{Name: "db", Address: "127.0.0.3", Port: 8003})

	reqBody := BatchServiceGetRequest{
		Names: []string{"web", "api", "nonexistent"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/services/get", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchServiceGetResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Found) != 2 {
		t.Errorf("Expected 2 found, got %d", len(result.Found))
	}
	if len(result.NotFound) != 1 {
		t.Errorf("Expected 1 not found, got %d", len(result.NotFound))
	}
	if result.Found["web"].Port != 8001 {
		t.Errorf("web service port incorrect")
	}
	if result.Found["api"].Port != 8002 {
		t.Errorf("api service port incorrect")
	}
	if result.NotFound[0] != "nonexistent" {
		t.Errorf("Expected 'nonexistent' in not found list")
	}
}

func TestBatchServiceGet_EmptyNames(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/services/get", handler.BatchServiceGet)

	reqBody := BatchServiceGetRequest{Names: []string{}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/services/get", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchKVGet_InvalidJSON(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/get", handler.BatchKVGet)

	req := httptest.NewRequest("POST", "/batch/kv/get", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchKVSetCAS_CreateOnly_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/set-cas", handler.BatchKVSetCAS)

	// Create-only with CAS=0 for all keys
	reqBody := BatchKVSetCASRequest{
		Items: map[string]string{
			"new_key1": "value1",
			"new_key2": "value2",
			"new_key3": "value3",
		},
		ExpectedIndices: map[string]uint64{
			"new_key1": 0,
			"new_key2": 0,
			"new_key3": 0,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/set-cas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchKVSetCASResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Expected count 3, got %d", result.Count)
	}
	if len(result.NewIndices) != 3 {
		t.Errorf("Expected 3 new indices, got %d", len(result.NewIndices))
	}

	// Verify keys were created
	val1, ok1 := handler.kvStore.Get("new_key1")
	val2, ok2 := handler.kvStore.Get("new_key2")
	val3, ok3 := handler.kvStore.Get("new_key3")

	if !ok1 || val1 != "value1" {
		t.Error("new_key1 should exist with value1")
	}
	if !ok2 || val2 != "value2" {
		t.Error("new_key2 should exist with value2")
	}
	if !ok3 || val3 != "value3" {
		t.Error("new_key3 should exist with value3")
	}
}

func TestBatchKVSetCAS_ConditionalUpdate_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/set-cas", handler.BatchKVSetCAS)

	// Create initial keys
	handler.kvStore.Set("key1", "old_value1")
	handler.kvStore.Set("key2", "old_value2")

	// Get current indices
	entry1, _ := handler.kvStore.GetEntry("key1")
	entry2, _ := handler.kvStore.GetEntry("key2")

	// Update with correct indices
	reqBody := BatchKVSetCASRequest{
		Items: map[string]string{
			"key1": "new_value1",
			"key2": "new_value2",
		},
		ExpectedIndices: map[string]uint64{
			"key1": entry1.ModifyIndex,
			"key2": entry2.ModifyIndex,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/set-cas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify values were updated
	val1, _ := handler.kvStore.Get("key1")
	val2, _ := handler.kvStore.Get("key2")

	if val1 != "new_value1" {
		t.Errorf("Expected key1=new_value1, got %s", val1)
	}
	if val2 != "new_value2" {
		t.Errorf("Expected key2=new_value2, got %s", val2)
	}
}

func TestBatchKVSetCAS_Conflict(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/set-cas", handler.BatchKVSetCAS)

	// Create initial key
	handler.kvStore.Set("key1", "value1")
	entry1, _ := handler.kvStore.GetEntry("key1")

	// Modify the key to change its index
	handler.kvStore.Set("key1", "value1_modified")

	// Try to update with old index - should fail
	reqBody := BatchKVSetCASRequest{
		Items: map[string]string{
			"key1": "value1_new",
			"key2": "value2",
		},
		ExpectedIndices: map[string]uint64{
			"key1": entry1.ModifyIndex, // Old index - will conflict
			"key2": 0,                  // Create-only
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/set-cas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 409 {
		t.Errorf("Expected status 409 (conflict), got %d", resp.StatusCode)
	}

	// Verify NO keys were modified (atomicity)
	val1, _ := handler.kvStore.Get("key1")
	_, ok2 := handler.kvStore.Get("key2")

	if val1 != "value1_modified" {
		t.Errorf("key1 should not have been modified due to conflict")
	}
	if ok2 {
		t.Error("key2 should not have been created due to conflict")
	}
}

func TestBatchKVSetCAS_MissingIndices(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/set-cas", handler.BatchKVSetCAS)

	// Missing expected index for key2
	reqBody := BatchKVSetCASRequest{
		Items: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		ExpectedIndices: map[string]uint64{
			"key1": 0,
			// Missing key2
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/set-cas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestBatchKVDeleteCAS_Success(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/delete-cas", handler.BatchKVDeleteCAS)

	// Create some keys
	handler.kvStore.Set("del1", "value1")
	handler.kvStore.Set("del2", "value2")
	handler.kvStore.Set("del3", "value3")
	handler.kvStore.Set("keep", "keep_value")

	// Get current indices
	entry1, _ := handler.kvStore.GetEntry("del1")
	entry2, _ := handler.kvStore.GetEntry("del2")
	entry3, _ := handler.kvStore.GetEntry("del3")

	// Delete with correct indices
	reqBody := BatchKVDeleteCASRequest{
		Keys: []string{"del1", "del2", "del3"},
		ExpectedIndices: map[string]uint64{
			"del1": entry1.ModifyIndex,
			"del2": entry2.ModifyIndex,
			"del3": entry3.ModifyIndex,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/delete-cas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BatchKVDeleteCASResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Expected count 3, got %d", result.Count)
	}

	// Verify keys were deleted
	_, ok1 := handler.kvStore.Get("del1")
	_, ok2 := handler.kvStore.Get("del2")
	_, ok3 := handler.kvStore.Get("del3")
	_, okKeep := handler.kvStore.Get("keep")

	if ok1 {
		t.Error("del1 should have been deleted")
	}
	if ok2 {
		t.Error("del2 should have been deleted")
	}
	if ok3 {
		t.Error("del3 should have been deleted")
	}
	if !okKeep {
		t.Error("keep should not have been deleted")
	}
}

func TestBatchKVDeleteCAS_Conflict(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/delete-cas", handler.BatchKVDeleteCAS)

	// Create keys
	handler.kvStore.Set("key1", "value1")
	handler.kvStore.Set("key2", "value2")

	entry1, _ := handler.kvStore.GetEntry("key1")
	entry2, _ := handler.kvStore.GetEntry("key2")

	// Modify key1 to change its index
	handler.kvStore.Set("key1", "value1_modified")

	// Try to delete with old index for key1
	reqBody := BatchKVDeleteCASRequest{
		Keys: []string{"key1", "key2"},
		ExpectedIndices: map[string]uint64{
			"key1": entry1.ModifyIndex, // Old index - will conflict
			"key2": entry2.ModifyIndex,
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/delete-cas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 409 {
		t.Errorf("Expected status 409 (conflict), got %d", resp.StatusCode)
	}

	// Verify NO keys were deleted (atomicity)
	_, ok1 := handler.kvStore.Get("key1")
	_, ok2 := handler.kvStore.Get("key2")

	if !ok1 {
		t.Error("key1 should not have been deleted due to conflict")
	}
	if !ok2 {
		t.Error("key2 should not have been deleted due to conflict")
	}
}

func TestBatchKVDeleteCAS_MissingIndices(t *testing.T) {
	app, handler := setupBatchTestApp()
	app.Post("/batch/kv/delete-cas", handler.BatchKVDeleteCAS)

	handler.kvStore.Set("key1", "value1")
	handler.kvStore.Set("key2", "value2")

	entry1, _ := handler.kvStore.GetEntry("key1")

	// Missing expected index for key2
	reqBody := BatchKVDeleteCASRequest{
		Keys: []string{"key1", "key2"},
		ExpectedIndices: map[string]uint64{
			"key1": entry1.ModifyIndex,
			// Missing key2
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/batch/kv/delete-cas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
