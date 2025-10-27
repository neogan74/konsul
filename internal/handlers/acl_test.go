package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/logger"
)

func setupACLHandler(policyDir string) (*ACLHandler, *fiber.App) {
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)
	handler := NewACLHandler(evaluator, policyDir, log)
	app := fiber.New()

	// Add logging middleware to set logger in context
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("logger", log)
		return c.Next()
	})

	app.Post("/acl/policies", handler.CreatePolicy)
	app.Get("/acl/policies/:name", handler.GetPolicy)
	app.Get("/acl/policies", handler.ListPolicies)
	app.Put("/acl/policies/:name", handler.UpdatePolicy)
	app.Delete("/acl/policies/:name", handler.DeletePolicy)
	app.Post("/acl/test", handler.TestPolicy)

	return handler, app
}

func TestACLHandler_CreatePolicy(t *testing.T) {
	tmpDir := t.TempDir()
	_, app := setupACLHandler(tmpDir)

	// Test valid policy creation
	policy := acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "app/*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}

	body, _ := json.Marshal(policy)
	req := httptest.NewRequest(http.MethodPost, "/acl/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("create policy request failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "policy created" {
		t.Errorf("unexpected response: %+v", result)
	}

	// Verify policy file was created
	policyFile := filepath.Join(tmpDir, "test-policy.json")
	if _, err := os.Stat(policyFile); os.IsNotExist(err) {
		t.Error("policy file was not created")
	}
}

func TestACLHandler_CreatePolicy_InvalidJSON(t *testing.T) {
	_, app := setupACLHandler("")

	req := httptest.NewRequest(http.MethodPost, "/acl/policies", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestACLHandler_CreatePolicy_InvalidPolicy(t *testing.T) {
	_, app := setupACLHandler("")

	// Policy without name (invalid)
	policy := acl.Policy{
		Description: "Invalid policy",
	}

	body, _ := json.Marshal(policy)
	req := httptest.NewRequest(http.MethodPost, "/acl/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestACLHandler_CreatePolicy_Duplicate(t *testing.T) {
	_, app := setupACLHandler("")

	policy := acl.Policy{
		Name:        "duplicate-policy",
		Description: "Test policy",
	}

	body, _ := json.Marshal(policy)

	// Create first time
	req := httptest.NewRequest(http.MethodPost, "/acl/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first create failed with status %d", resp.StatusCode)
	}

	// Create duplicate
	req = httptest.NewRequest(http.MethodPost, "/acl/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestACLHandler_GetPolicy(t *testing.T) {
	handler, app := setupACLHandler("")

	// Create a policy first
	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
	}
	handler.evaluator.AddPolicy(policy)

	// Get the policy
	req := httptest.NewRequest(http.MethodGet, "/acl/policies/test-policy", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result acl.Policy
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Name != "test-policy" {
		t.Errorf("expected policy name 'test-policy', got '%s'", result.Name)
	}
}

func TestACLHandler_GetPolicy_NotFound(t *testing.T) {
	_, app := setupACLHandler("")

	req := httptest.NewRequest(http.MethodGet, "/acl/policies/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestACLHandler_ListPolicies(t *testing.T) {
	handler, app := setupACLHandler("")

	// Create some policies
	policy1 := &acl.Policy{Name: "policy1", Description: "Test 1"}
	policy2 := &acl.Policy{Name: "policy2", Description: "Test 2"}
	handler.evaluator.AddPolicy(policy1)
	handler.evaluator.AddPolicy(policy2)

	req := httptest.NewRequest(http.MethodGet, "/acl/policies", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	count := int(result["count"].(float64))
	if count != 2 {
		t.Errorf("expected 2 policies, got %d", count)
	}

	policies := result["policies"].([]interface{})
	if len(policies) != 2 {
		t.Errorf("expected 2 policy names, got %d", len(policies))
	}
}

func TestACLHandler_ListPolicies_Empty(t *testing.T) {
	_, app := setupACLHandler("")

	req := httptest.NewRequest(http.MethodGet, "/acl/policies", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	count := int(result["count"].(float64))
	if count != 0 {
		t.Errorf("expected 0 policies, got %d", count)
	}
}

func TestACLHandler_UpdatePolicy(t *testing.T) {
	tmpDir := t.TempDir()
	handler, app := setupACLHandler(tmpDir)

	// Create a policy first
	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Original description",
	}
	handler.evaluator.AddPolicy(policy)

	// Update the policy
	updatedPolicy := acl.Policy{
		Name:        "test-policy",
		Description: "Updated description",
	}

	body, _ := json.Marshal(updatedPolicy)
	req := httptest.NewRequest(http.MethodPut, "/acl/policies/test-policy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["message"] != "policy updated" {
		t.Errorf("unexpected response: %+v", result)
	}

	// Verify the policy was updated
	retrieved, _ := handler.evaluator.GetPolicy("test-policy")
	if retrieved.Description != "Updated description" {
		t.Errorf("policy was not updated")
	}
}

func TestACLHandler_UpdatePolicy_NotFound(t *testing.T) {
	_, app := setupACLHandler("")

	policy := acl.Policy{
		Name:        "nonexistent",
		Description: "Test",
	}

	body, _ := json.Marshal(policy)
	req := httptest.NewRequest(http.MethodPut, "/acl/policies/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestACLHandler_UpdatePolicy_NameMismatch(t *testing.T) {
	handler, app := setupACLHandler("")

	// Create a policy first
	policy := &acl.Policy{Name: "test-policy", Description: "Test"}
	handler.evaluator.AddPolicy(policy)

	// Try to update with different name in body
	updatedPolicy := acl.Policy{
		Name:        "different-name",
		Description: "Test",
	}

	body, _ := json.Marshal(updatedPolicy)
	req := httptest.NewRequest(http.MethodPut, "/acl/policies/test-policy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestACLHandler_DeletePolicy(t *testing.T) {
	tmpDir := t.TempDir()
	handler, app := setupACLHandler(tmpDir)

	// Create a policy and save it to file
	policy := &acl.Policy{Name: "test-policy", Description: "Test"}
	handler.evaluator.AddPolicy(policy)
	handler.savePolicyToFile(policy)

	// Verify file exists
	policyFile := filepath.Join(tmpDir, "test-policy.json")
	if _, err := os.Stat(policyFile); os.IsNotExist(err) {
		t.Fatal("policy file was not created")
	}

	// Delete the policy
	req := httptest.NewRequest(http.MethodDelete, "/acl/policies/test-policy", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify policy was removed from evaluator
	_, err = handler.evaluator.GetPolicy("test-policy")
	if err != acl.ErrPolicyNotFound {
		t.Error("policy was not removed from evaluator")
	}

	// Verify file was deleted
	if _, err := os.Stat(policyFile); !os.IsNotExist(err) {
		t.Error("policy file was not deleted")
	}
}

func TestACLHandler_DeletePolicy_NotFound(t *testing.T) {
	_, app := setupACLHandler("")

	req := httptest.NewRequest(http.MethodDelete, "/acl/policies/nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestACLHandler_TestPolicy_KV(t *testing.T) {
	handler, app := setupACLHandler("")

	// Create a policy
	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "app/*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	handler.evaluator.AddPolicy(policy)

	// Test allowed access
	testReq := map[string]interface{}{
		"policies":   []string{"test-policy"},
		"resource":   "kv",
		"path":       "app/config",
		"capability": "read",
	}

	body, _ := json.Marshal(testReq)
	req := httptest.NewRequest(http.MethodPost, "/acl/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if !result["allowed"].(bool) {
		t.Error("expected access to be allowed")
	}
}

func TestACLHandler_TestPolicy_Denied(t *testing.T) {
	handler, app := setupACLHandler("")

	// Create a policy
	policy := &acl.Policy{
		Name:        "test-policy",
		Description: "Test policy",
		KV: []acl.KVRule{
			{
				Path:         "app/*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
		},
	}
	handler.evaluator.AddPolicy(policy)

	// Test denied access (wrong path)
	testReq := map[string]interface{}{
		"policies":   []string{"test-policy"},
		"resource":   "kv",
		"path":       "other/config",
		"capability": "read",
	}

	body, _ := json.Marshal(testReq)
	req := httptest.NewRequest(http.MethodPost, "/acl/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["allowed"].(bool) {
		t.Error("expected access to be denied")
	}
}

func TestACLHandler_TestPolicy_InvalidResource(t *testing.T) {
	_, app := setupACLHandler("")

	testReq := map[string]interface{}{
		"policies":   []string{"test-policy"},
		"resource":   "invalid",
		"path":       "test",
		"capability": "read",
	}

	body, _ := json.Marshal(testReq)
	req := httptest.NewRequest(http.MethodPost, "/acl/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestACLHandler_TestPolicy_AllResourceTypes(t *testing.T) {
	handler, app := setupACLHandler("")

	// Create policies for each resource type
	kvPolicy := &acl.Policy{
		Name: "kv-policy",
		KV:   []acl.KVRule{{Path: "*", Capabilities: []acl.Capability{acl.CapabilityRead}}},
	}
	servicePolicy := &acl.Policy{
		Name:    "service-policy",
		Service: []acl.ServiceRule{{Name: "*", Capabilities: []acl.Capability{acl.CapabilityRead}}},
	}
	healthPolicy := &acl.Policy{
		Name:   "health-policy",
		Health: []acl.HealthRule{{Capabilities: []acl.Capability{acl.CapabilityRead}}},
	}
	backupPolicy := &acl.Policy{
		Name:   "backup-policy",
		Backup: []acl.BackupRule{{Capabilities: []acl.Capability{acl.CapabilityCreate}}},
	}
	adminPolicy := &acl.Policy{
		Name:  "admin-policy",
		Admin: []acl.AdminRule{{Capabilities: []acl.Capability{acl.CapabilityRead}}},
	}

	handler.evaluator.AddPolicy(kvPolicy)
	handler.evaluator.AddPolicy(servicePolicy)
	handler.evaluator.AddPolicy(healthPolicy)
	handler.evaluator.AddPolicy(backupPolicy)
	handler.evaluator.AddPolicy(adminPolicy)

	tests := []struct {
		name         string
		resourceType string
		policy       string
		path         string
		capability   string
		shouldAllow  bool
	}{
		{"kv allowed", "kv", "kv-policy", "test", "read", true},
		{"service allowed", "service", "service-policy", "web", "read", true},
		{"health allowed", "health", "health-policy", "", "read", true},
		{"backup allowed", "backup", "backup-policy", "", "create", true},
		{"admin allowed", "admin", "admin-policy", "", "read", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testReq := map[string]interface{}{
				"policies":   []string{tt.policy},
				"resource":   tt.resourceType,
				"path":       tt.path,
				"capability": tt.capability,
			}

			body, _ := json.Marshal(testReq)
			req := httptest.NewRequest(http.MethodPost, "/acl/test", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			if result["allowed"].(bool) != tt.shouldAllow {
				t.Errorf("expected allowed=%v, got %v", tt.shouldAllow, result["allowed"])
			}
		})
	}
}

func TestACLHandler_LoadPolicies(t *testing.T) {
	tmpDir := t.TempDir()
	handler, _ := setupACLHandler(tmpDir)

	// Create some policy files
	policy1 := &acl.Policy{
		Name:        "policy1",
		Description: "Test policy 1",
		KV:          []acl.KVRule{{Path: "app/*", Capabilities: []acl.Capability{acl.CapabilityRead}}},
	}
	policy2 := &acl.Policy{
		Name:        "policy2",
		Description: "Test policy 2",
		Service:     []acl.ServiceRule{{Name: "web-*", Capabilities: []acl.Capability{acl.CapabilityRead}}},
	}

	// Save policies to files
	data1, _ := json.MarshalIndent(policy1, "", "  ")
	data2, _ := json.MarshalIndent(policy2, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "policy1.json"), data1, 0644)
	os.WriteFile(filepath.Join(tmpDir, "policy2.json"), data2, 0644)

	// Load policies
	err := handler.LoadPolicies()
	if err != nil {
		t.Fatalf("LoadPolicies failed: %v", err)
	}

	// Verify policies were loaded
	if handler.evaluator.Count() != 2 {
		t.Errorf("expected 2 policies, got %d", handler.evaluator.Count())
	}

	// Verify specific policies
	loaded1, err := handler.evaluator.GetPolicy("policy1")
	if err != nil {
		t.Error("policy1 was not loaded")
	}
	if loaded1.Description != "Test policy 1" {
		t.Error("policy1 has incorrect data")
	}

	loaded2, err := handler.evaluator.GetPolicy("policy2")
	if err != nil {
		t.Error("policy2 was not loaded")
	}
	if loaded2.Description != "Test policy 2" {
		t.Error("policy2 has incorrect data")
	}
}

func TestACLHandler_LoadPolicies_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	handler, _ := setupACLHandler(tmpDir)

	// Create an invalid policy file
	os.WriteFile(filepath.Join(tmpDir, "invalid.json"), []byte("invalid json"), 0644)

	// Load policies - should not fail, just log error
	err := handler.LoadPolicies()
	if err != nil {
		t.Fatalf("LoadPolicies should not fail on invalid files: %v", err)
	}

	// Should have 0 policies
	if handler.evaluator.Count() != 0 {
		t.Errorf("expected 0 policies, got %d", handler.evaluator.Count())
	}
}

func TestACLHandler_LoadPolicies_NoPolicyDir(t *testing.T) {
	handler, _ := setupACLHandler("")

	// Load policies with empty policy dir - should return nil
	err := handler.LoadPolicies()
	if err != nil {
		t.Errorf("LoadPolicies with empty dir should return nil, got: %v", err)
	}
}

func TestACLHandler_SavePolicyToFile(t *testing.T) {
	tmpDir := t.TempDir()
	handler, _ := setupACLHandler(tmpDir)

	policy := &acl.Policy{
		Name:        "test-save",
		Description: "Test save",
		KV:          []acl.KVRule{{Path: "test/*", Capabilities: []acl.Capability{acl.CapabilityRead}}},
	}

	err := handler.savePolicyToFile(policy)
	if err != nil {
		t.Fatalf("savePolicyToFile failed: %v", err)
	}

	// Verify file exists and contains correct data
	policyFile := filepath.Join(tmpDir, "test-save.json")
	data, err := os.ReadFile(policyFile)
	if err != nil {
		t.Fatalf("failed to read policy file: %v", err)
	}

	var loaded acl.Policy
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse policy file: %v", err)
	}

	if loaded.Name != "test-save" {
		t.Errorf("expected name 'test-save', got '%s'", loaded.Name)
	}
	if loaded.Description != "Test save" {
		t.Errorf("expected description 'Test save', got '%s'", loaded.Description)
	}
}
