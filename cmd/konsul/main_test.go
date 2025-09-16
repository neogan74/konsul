package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

func setupApp() *fiber.App {
	app := fiber.New()
	svcStore := store.NewServiceStore()

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
	if len(services) != 1 || services[0] != service {
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
