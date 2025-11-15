package audit

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// ExtractActorFromContext extracts actor information from the request context.
// It looks for authentication information set by JWT or API key middleware.
func ExtractActorFromContext(c *fiber.Ctx) Actor {
	actor := Actor{
		Type: "anonymous",
	}

	// Try JWT claims first
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			actor.ID = uid
			actor.Type = "user"
		}
	}

	if username := c.Locals("username"); username != nil {
		if name, ok := username.(string); ok {
			actor.Name = name
		}
	}

	if roles := c.Locals("roles"); roles != nil {
		if roleList, ok := roles.([]string); ok {
			actor.Roles = roleList
		}
	}

	// Check for API key authentication
	if keyID := c.Locals("api_key_id"); keyID != nil {
		if kid, ok := keyID.(string); ok {
			actor.TokenID = kid
			actor.Type = "api_key"
		}
	}

	// Check for service token
	if serviceID := c.Locals("service_id"); serviceID != nil {
		if sid, ok := serviceID.(string); ok {
			actor.ID = sid
			actor.Type = "service"
		}
	}

	return actor
}

// ExtractResourceFromPath extracts resource information from the request path.
// Examples:
//   - /api/v1/kv/config/app -> Resource{Type: "kv", ID: "config/app"}
//   - /api/v1/service/web -> Resource{Type: "service", ID: "web"}
func ExtractResourceFromPath(c *fiber.Ctx, resourceType string) Resource {
	resource := Resource{
		Type: resourceType,
	}

	// Extract resource ID from path parameters
	if id := c.Params("key"); id != "" {
		resource.ID = id
	} else if id := c.Params("id"); id != "" {
		resource.ID = id
	} else if id := c.Params("name"); id != "" {
		resource.ID = id
	}

	// Extract namespace if present
	if ns := c.Query("namespace"); ns != "" {
		resource.Namespace = ns
	}

	return resource
}

// HashRequestBody creates a SHA-256 hash of the request body for audit trails.
// This ensures we can verify integrity without storing sensitive payloads.
func HashRequestBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	hash := sha256.Sum256(body)
	return fmt.Sprintf("%x", hash)
}

// BuildEvent creates a new audit event from HTTP context.
// It automatically populates timestamp, actor, source IP, and HTTP metadata.
func BuildEvent(c *fiber.Ctx, action string, resourceType string) *Event {
	event := &Event{
		Timestamp:  c.Context().Time(),
		Action:     action,
		Actor:      ExtractActorFromContext(c),
		Resource:   ExtractResourceFromPath(c, resourceType),
		SourceIP:   c.IP(),
		HTTPMethod: c.Method(),
		HTTPPath:   c.Path(),
		Metadata:   make(map[string]string),
	}

	// Add trace context if available
	if traceID := c.Locals("trace_id"); traceID != nil {
		if tid, ok := traceID.(string); ok {
			event.TraceID = tid
		}
	}
	if spanID := c.Locals("span_id"); spanID != nil {
		if sid, ok := spanID.(string); ok {
			event.SpanID = sid
		}
	}

	// Determine auth method from context
	if c.Locals("jwt_auth") != nil {
		event.AuthMethod = "jwt"
	} else if c.Locals("api_key_auth") != nil {
		event.AuthMethod = "api_key"
	}

	// Hash request body for non-GET requests
	if c.Method() != "GET" && c.Method() != "HEAD" {
		event.RequestHash = HashRequestBody(c.Body())
	}

	return event
}

// RecordHTTPEvent is a convenience function that builds and records an audit event.
// It handles the full lifecycle: build event, set result based on status, and record.
func RecordHTTPEvent(ctx context.Context, mgr *Manager, c *fiber.Ctx, action string, resourceType string, statusCode int) error {
	if mgr == nil || !mgr.Enabled() {
		return nil
	}

	event := BuildEvent(c, action, resourceType)
	event.HTTPStatus = statusCode

	// Set result based on HTTP status code
	if statusCode >= 200 && statusCode < 300 {
		event.Result = "success"
	} else if statusCode >= 400 && statusCode < 500 {
		event.Result = "denied"
	} else if statusCode >= 500 {
		event.Result = "error"
	} else {
		event.Result = "unknown"
	}

	_, err := mgr.Record(ctx, event)
	return err
}
