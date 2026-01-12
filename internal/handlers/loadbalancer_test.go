package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/loadbalancer"
	"github.com/neogan74/konsul/internal/store"
)

func setupLoadBalancerHandler(t *testing.T) (*LoadBalancerHandler, *fiber.App) {
	t.Helper()
	// Create a service store
	serviceStore := store.NewServiceStore()

	// Register some test services
	if err := serviceStore.Register(store.Service{
		Name:    "test-service",
		Address: "10.0.1.1",
		Port:    8080,
		Tags:    []string{"test-service", "web", "api"},
		Meta:    map[string]string{"env": "test", "version": "1.0"},
	}); err != nil {
		t.Fatalf("register service: %v", err)
	}
	if err := serviceStore.Register(store.Service{
		Name:    "test-service",
		Address: "10.0.1.2",
		Port:    8080,
		Tags:    []string{"test-service", "web", "api"},
		Meta:    map[string]string{"env": "test", "version": "1.0"},
	}); err != nil {
		t.Fatalf("register service: %v", err)
	}
	if err := serviceStore.Register(store.Service{
		Name:    "db-service",
		Address: "10.0.2.1",
		Port:    5432,
		Tags:    []string{"db-service", "database", "postgres"},
		Meta:    map[string]string{"env": "prod", "version": "14"},
	}); err != nil {
		t.Fatalf("register service: %v", err)
	}
	balancer := loadbalancer.New(serviceStore, loadbalancer.StrategyRoundRobin)
	handler := NewLoadBalancerHandler(balancer)

	app := fiber.New()

	app.Get("/lb/service/:name", handler.SelectService)
	app.Get("/lb/tags", handler.SelectServiceByTags)
	app.Get("/lb/metadata", handler.SelectServiceByMetadata)
	app.Get("/lb/query", handler.SelectServiceByQuery)
	app.Get("/lb/strategy", handler.GetStrategy)
	app.Put("/lb/strategy", handler.UpdateStrategy)

	return handler, app
}

func TestLoadBalancerHandler_SelectService_Success(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/service/test-service", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectService request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["service"]; !ok {
		t.Error("expected service in response")
	}
	if _, ok := result["strategy"]; !ok {
		t.Error("expected strategy in response")
	}
}

func TestLoadBalancerHandler_SelectService_NotFound(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/service/nonexistent-service", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectService request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByTags_Success(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/tags?tags=web&tags=api", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByTags request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["service"]; !ok {
		t.Error("expected service in response")
	}
	if _, ok := result["strategy"]; !ok {
		t.Error("expected strategy in response")
	}
	if _, ok := result["query"]; !ok {
		t.Error("expected query in response")
	}
}

func TestLoadBalancerHandler_SelectServiceByTags_NoTags(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/tags", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByTags request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByTags_NotFound(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/tags?tags=nonexistent-tag", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByTags request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByMetadata_Success(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/metadata?env=test&version=1.0", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByMetadata request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["service"]; !ok {
		t.Error("expected service in response")
	}
	if _, ok := result["query"]; !ok {
		t.Error("expected query in response")
	}
}

func TestLoadBalancerHandler_SelectServiceByMetadata_NoFilters(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/metadata", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByMetadata request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByMetadata_NotFound(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/metadata?env=nonexistent", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByMetadata request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByQuery_Success(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/query?tags=web&meta.env=test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByQuery request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["service"]; !ok {
		t.Error("expected service in response")
	}
	if _, ok := result["strategy"]; !ok {
		t.Error("expected strategy in response")
	}

	// Check query structure
	query, ok := result["query"].(map[string]interface{})
	if !ok {
		t.Fatal("expected query to be a map")
	}
	if _, ok := query["tags"]; !ok {
		t.Error("expected tags in query")
	}
	if _, ok := query["metadata"]; !ok {
		t.Error("expected metadata in query")
	}
}

func TestLoadBalancerHandler_SelectServiceByQuery_TagsOnly(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/query?tags=web", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByQuery request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByQuery_MetadataOnly(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/query?meta.env=test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByQuery request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByQuery_NoFilters(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/query", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByQuery request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_SelectServiceByQuery_NotFound(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/query?tags=nonexistent&meta.env=invalid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("SelectServiceByQuery request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_GetStrategy(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/lb/strategy", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("GetStrategy request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["strategy"]; !ok {
		t.Error("expected strategy in response")
	}
}

func TestLoadBalancerHandler_UpdateStrategy_Success(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	strategies := []string{"round-robin", "random", "least-connections"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			body := bytes.NewReader([]byte(`{"strategy": "` + strategy + `"}`))
			req := httptest.NewRequest(http.MethodPut, "/lb/strategy", body)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("UpdateStrategy request failed: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if result["strategy"] != strategy {
				t.Errorf("expected strategy '%s', got %v", strategy, result["strategy"])
			}
		})
	}
}

func TestLoadBalancerHandler_UpdateStrategy_InvalidStrategy(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	body := bytes.NewReader([]byte(`{"strategy": "invalid-strategy"}`))
	req := httptest.NewRequest(http.MethodPut, "/lb/strategy", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("UpdateStrategy request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_UpdateStrategy_InvalidJSON(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	body := bytes.NewReader([]byte(`invalid json`))
	req := httptest.NewRequest(http.MethodPut, "/lb/strategy", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("UpdateStrategy request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoadBalancerHandler_StrategyPersistence(t *testing.T) {
	_, app := setupLoadBalancerHandler(t)

	// Update strategy
	updateBody := bytes.NewReader([]byte(`{"strategy": "random"}`))
	updateReq := httptest.NewRequest(http.MethodPut, "/lb/strategy", updateBody)
	updateReq.Header.Set("Content-Type", "application/json")

	updateResp, err := app.Test(updateReq)
	if err != nil {
		t.Fatalf("UpdateStrategy request failed: %v", err)
	}

	if updateResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 on update, got %d", updateResp.StatusCode)
	}

	// Verify strategy was updated
	getReq := httptest.NewRequest(http.MethodGet, "/lb/strategy", nil)
	getResp, err := app.Test(getReq)
	if err != nil {
		t.Fatalf("GetStrategy request failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["strategy"] != "random" {
		t.Errorf("expected strategy 'random', got %v", result["strategy"])
	}
}
