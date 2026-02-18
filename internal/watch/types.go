package watch

import (
	"time"
)

// EventType represents the type of watch event
type EventType string

// EventTypeSet represents a set event
const (
	EventTypeSet    EventType = "set"
	EventTypeDelete EventType = "delete"
)

// Event represents a change to a key
type Event struct {
	Type      EventType `json:"type"`
	Key       string    `json:"key"`
	Value     string    `json:"value,omitempty"`
	OldValue  string    `json:"old_value,omitempty"`
	Timestamp int64     `json:"timestamp"`
}

// TransportType represents the transport protocol for watch connections
type TransportType string

// TransportWebSocket represents a WebSocket transport type
const (
	TransportWebSocket TransportType = "websocket"
	TransportSSE       TransportType = "sse"
)

// Watcher represents a single watch subscription
type Watcher struct {
	ID          string
	Pattern     string // Key or prefix to watch (supports * and **)
	Events      chan Event
	ACLPolicies []string // Policies for ACL checks
	CreatedAt   time.Time
	Transport   TransportType
	UserID      string // Optional user identifier
}

// NewWatcher creates a new watcher with a buffered event channel
func NewWatcher(id, pattern string, policies []string, transport TransportType, bufferSize int) *Watcher {
	return &Watcher{
		ID:          id,
		Pattern:     pattern,
		Events:      make(chan Event, bufferSize),
		ACLPolicies: policies,
		CreatedAt:   time.Now(),
		Transport:   transport,
	}
}
