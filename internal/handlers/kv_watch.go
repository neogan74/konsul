package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/auth"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/store"
	"github.com/neogan74/konsul/internal/watch"
)

// KVWatchHandler handles watch/subscribe requests for KV store
type KVWatchHandler struct {
	store        *store.KVStore
	watchManager *watch.Manager
	aclEval      *acl.Evaluator
	log          logger.Logger
}

// NewKVWatchHandler creates a new KV watch handler
func NewKVWatchHandler(kvStore *store.KVStore, watchManager *watch.Manager, aclEval *acl.Evaluator, log logger.Logger) *KVWatchHandler {
	return &KVWatchHandler{
		store:        kvStore,
		watchManager: watchManager,
		aclEval:      aclEval,
		log:          log,
	}
}

// WatchWebSocket handles WebSocket watch connections
func (h *KVWatchHandler) WatchWebSocket(c *websocket.Conn) {
	// Extract key pattern from query param or URL
	pattern := c.Params("key")
	if pattern == "" {
		pattern = c.Query("key", "*")
	}

	// Get claims from locals (set before WebSocket upgrade)
	claimsVal := c.Locals("claims")
	var claims *auth.Claims
	var userID string
	var policies []string

	if claimsVal != nil {
		var ok bool
		claims, ok = claimsVal.(*auth.Claims)
		if ok && claims != nil {
			userID = claims.UserID
			if userID == "" {
				userID = claims.Username
			}
			policies = claims.Policies
		}
	}

	// If no user ID, use a default identifier
	if userID == "" {
		userID = "anonymous"
	}

	h.log.Info("WebSocket watch connection established",
		logger.String("user_id", userID),
		logger.String("pattern", pattern),
		logger.String("transport", "websocket"))

	// Check ACL permission - user must have read permission for the pattern
	resource := acl.NewKVResource(pattern)
	if h.aclEval != nil && !h.aclEval.Evaluate(policies, resource, acl.CapabilityRead) {
		h.log.Warn("WebSocket watch: ACL check failed")
		c.WriteJSON(fiber.Map{
			"error":   "forbidden",
			"message": "insufficient permissions to watch this key pattern",
		})
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "forbidden"))
		return
	}

	// Add watcher
	watcher, err := h.watchManager.AddWatcher(pattern, policies, watch.TransportWebSocket, userID)
	if err != nil {
		h.log.Error("Failed to add watcher", logger.Error(err))
		c.WriteJSON(fiber.Map{
			"error":   "failed to add watcher",
			"message": err.Error(),
		})
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
		return
	}
	defer h.watchManager.RemoveWatcher(watcher.ID)

	h.log.Info("Watcher added", logger.String("watcher_id", watcher.ID))

	// Send initial value if exact key match (not a wildcard)
	if pattern != "*" && pattern != "**" && !containsWildcard(pattern) {
		if value, ok := h.store.Get(pattern); ok {
			initialEvent := watch.WatchEvent{
				Type:      watch.EventTypeSet,
				Key:       pattern,
				Value:     value,
				Timestamp: time.Now().Unix(),
			}
			if err := c.WriteJSON(initialEvent); err != nil {
				h.log.Error("Failed to send initial value", logger.Error(err))
				return
			}
			h.log.Debug("Sent initial value")
		}
	}

	// Setup ping/pong for connection health
	c.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// Read loop (to detect client disconnect)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				h.log.Debug("WebSocket read error (client disconnected)", logger.Error(err))
				return
			}
		}
	}()

	// Event streaming loop
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				h.log.Info("Watcher channel closed")
				return
			}

			// Send event to client
			if err := c.WriteJSON(event); err != nil {
				h.log.Error("Failed to write event", logger.Error(err))
				return
			}

			h.log.Debug("Sent watch event",
				logger.String("key", event.Key),
				logger.String("type", string(event.Type)))

		case <-pingTicker.C:
			// Send ping to keep connection alive
			if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				h.log.Debug("Failed to send ping", logger.Error(err))
				return
			}

		case <-done:
			h.log.Info("Client disconnected")
			return
		}
	}
}

// WatchSSE handles Server-Sent Events watch connections
func (h *KVWatchHandler) WatchSSE(c *fiber.Ctx) error {
	pattern := c.Params("key")
	if pattern == "" {
		pattern = c.Query("key", "*")
	}

	// Get claims from context (set by JWT middleware)
	claims := middleware.GetClaims(c)
	if claims == nil {
		h.log.Warn("SSE watch: no claims found")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "unauthorized",
			"message": "authentication required",
		})
	}

	// Get user ID
	userID := claims.UserID
	if userID == "" {
		userID = claims.Username
	}

	h.log.Info("SSE watch connection established",
		logger.String("user_id", userID),
		logger.String("pattern", pattern),
		logger.String("transport", "sse"))

	// Check ACL permission
	resource := acl.NewKVResource(pattern)
	if h.aclEval != nil && !h.aclEval.Evaluate(claims.Policies, resource, acl.CapabilityRead) {
		h.log.Warn("SSE watch: ACL check failed")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "insufficient permissions to watch this key pattern",
		})
	}

	// Add watcher
	watcher, err := h.watchManager.AddWatcher(pattern, claims.Policies, watch.TransportSSE, userID)
	if err != nil {
		h.log.Error("Failed to add watcher", logger.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to add watcher",
			"message": err.Error(),
		})
	}
	defer h.watchManager.RemoveWatcher(watcher.ID)

	h.log.Info("Watcher added", logger.String("watcher_id", watcher.ID))

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering
	c.Set("Access-Control-Allow-Origin", "*")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Send initial value if exact key match
		if pattern != "*" && pattern != "**" && !containsWildcard(pattern) {
			if value, ok := h.store.Get(pattern); ok {
				initialEvent := watch.WatchEvent{
					Type:      watch.EventTypeSet,
					Key:       pattern,
					Value:     value,
					Timestamp: time.Now().Unix(),
				}
				sendSSEEvent(w, initialEvent)
				h.log.Debug("Sent initial value")
			}
		}

		// Keep-alive ticker
		keepAliveTicker := time.NewTicker(30 * time.Second)
		defer keepAliveTicker.Stop()

		// Event streaming loop
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					h.log.Info("Watcher channel closed")
					return
				}

				// Send event
				if err := sendSSEEvent(w, event); err != nil {
					h.log.Error("Failed to send SSE event", logger.Error(err))
					return
				}

				h.log.Debug("Sent SSE event",
					logger.String("key", event.Key),
					logger.String("type", string(event.Type)))

			case <-keepAliveTicker.C:
				// Send keep-alive comment
				fmt.Fprintf(w, ": keep-alive\n\n")
				if err := w.Flush(); err != nil {
					h.log.Debug("Failed to send keep-alive", logger.Error(err))
					return
				}

			case <-c.Context().Done():
				h.log.Info("Client disconnected")
				return
			}
		}
	})

	return nil
}

// sendSSEEvent sends a watch event in SSE format
func sendSSEEvent(w *bufio.Writer, event watch.WatchEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "event: kv-change\n")
	fmt.Fprintf(w, "data: %s\n\n", string(data))
	return w.Flush()
}

// containsWildcard checks if a pattern contains wildcards
func containsWildcard(pattern string) bool {
	return strings.Contains(pattern, "*")
}
