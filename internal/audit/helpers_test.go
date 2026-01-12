package audit

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

func TestExtractActorFromContext(t *testing.T) {
	app := fiber.New()

	tests := []struct {
		name     string
		setup    func(*fiber.Ctx)
		expected Actor
	}{
		{
			name: "anonymous_user",
			setup: func(c *fiber.Ctx) {
				// No locals set
			},
			expected: Actor{Type: "anonymous"},
		},
		{
			name: "jwt_authenticated_user",
			setup: func(c *fiber.Ctx) {
				c.Locals("user_id", "user123")
				c.Locals("username", "john.doe")
				c.Locals("roles", []string{"admin", "developer"})
			},
			expected: Actor{
				ID:    "user123",
				Type:  "user",
				Name:  "john.doe",
				Roles: []string{"admin", "developer"},
			},
		},
		{
			name: "api_key_authenticated",
			setup: func(c *fiber.Ctx) {
				c.Locals("api_key_id", "key789")
			},
			expected: Actor{
				Type:    "api_key",
				TokenID: "key789",
			},
		},
		{
			name: "service_token",
			setup: func(c *fiber.Ctx) {
				c.Locals("service_id", "svc-backend")
			},
			expected: Actor{
				ID:   "svc-backend",
				Type: "service",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := app.AcquireCtx(&fasthttp.RequestCtx{})
			defer app.ReleaseCtx(c)

			tt.setup(c)
			actor := ExtractActorFromContext(c)

			if actor.ID != tt.expected.ID {
				t.Errorf("expected ID %q, got %q", tt.expected.ID, actor.ID)
			}
			if actor.Type != tt.expected.Type {
				t.Errorf("expected Type %q, got %q", tt.expected.Type, actor.Type)
			}
			if actor.Name != tt.expected.Name {
				t.Errorf("expected Name %q, got %q", tt.expected.Name, actor.Name)
			}
		})
	}
}

func TestExtractResourceFromPath(t *testing.T) {
	app := fiber.New()

	app.Get("/kv/:key", func(c *fiber.Ctx) error {
		resource := ExtractResourceFromPath(c, "kv")
		if resource.Type != "kv" {
			t.Errorf("expected resource type 'kv', got %q", resource.Type)
		}
		if resource.ID != "config/app" {
			t.Errorf("expected resource ID 'config/app', got %q", resource.ID)
		}
		return nil
	})

	req := httptest.NewRequest("GET", "/kv/config/app", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close response body: %v", err)
		}
	}()
}

func TestHashRequestBody(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		expected string
	}{
		{
			name:     "empty_body",
			body:     []byte{},
			expected: "",
		},
		{
			name:     "simple_json",
			body:     []byte(`{"key":"value"}`),
			expected: "e43abcf3375244839c012f9633f95862d232a95b00d5bc7348b3098b9fed7f32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashRequestBody(tt.body)
			if hash != tt.expected {
				t.Errorf("expected hash %q, got %q", tt.expected, hash)
			}
		})
	}
}

func TestBuildEvent(t *testing.T) {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", "user123")
		c.Locals("jwt_auth", true)
		return c.Next()
	})

	app.Post("/api/v1/kv/:key", func(c *fiber.Ctx) error {
		event := BuildEvent(c, "kv.set", "kv")

		if event.Action != "kv.set" {
			t.Errorf("expected action 'kv.set', got %q", event.Action)
		}
		if event.Actor.ID != "user123" {
			t.Errorf("expected actor ID 'user123', got %q", event.Actor.ID)
		}
		if event.HTTPMethod != "POST" {
			t.Errorf("expected HTTP method 'POST', got %q", event.HTTPMethod)
		}
		if event.AuthMethod != "jwt" {
			t.Errorf("expected auth method 'jwt', got %q", event.AuthMethod)
		}
		if event.Resource.Type != "kv" {
			t.Errorf("expected resource type 'kv', got %q", event.Resource.Type)
		}

		return c.SendStatus(200)
	})

	req := httptest.NewRequest("POST", "/api/v1/kv/test-key", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("close response body: %v", err)
		}
	}()
}
