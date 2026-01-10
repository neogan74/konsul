package watch

import (
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/logger"
)

func TestManager_AddWatcher(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher, err := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")
	if err != nil {
		t.Fatalf("Failed to add watcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Expected watcher to be created")
	}

	if watcher.Pattern != "app/config" {
		t.Errorf("Expected pattern 'app/config', got '%s'", watcher.Pattern)
	}

	if watcher.Transport != TransportWebSocket {
		t.Errorf("Expected WebSocket transport, got %s", watcher.Transport)
	}

	if manager.GetActiveWatcherCount() != 1 {
		t.Errorf("Expected 1 active watcher, got %d", manager.GetActiveWatcherCount())
	}
}

func TestManager_RemoveWatcher(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher, _ := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")

	manager.RemoveWatcher(watcher.ID)

	if manager.GetActiveWatcherCount() != 0 {
		t.Errorf("Expected 0 active watchers, got %d", manager.GetActiveWatcherCount())
	}

	// Channel should be closed
	_, ok := <-watcher.Events
	if ok {
		t.Error("Expected watcher channel to be closed")
	}
}

func TestManager_MaxWatchersPerClient(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 2) // Max 2 watchers per client

	// Add first watcher
	_, err := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")
	if err != nil {
		t.Fatalf("Failed to add first watcher: %v", err)
	}

	// Add second watcher
	_, err = manager.AddWatcher("app/data", []string{}, TransportWebSocket, "user1")
	if err != nil {
		t.Fatalf("Failed to add second watcher: %v", err)
	}

	// Third watcher should fail
	_, err = manager.AddWatcher("app/cache", []string{}, TransportWebSocket, "user1")
	if err != ErrTooManyWatchers {
		t.Errorf("Expected ErrTooManyWatchers, got %v", err)
	}

	// Different user should be allowed
	_, err = manager.AddWatcher("app/cache", []string{}, TransportWebSocket, "user2")
	if err != nil {
		t.Fatalf("Failed to add watcher for different user: %v", err)
	}
}

func TestManager_Notify_ExactMatch(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher, _ := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")

	// Send matching event
	event := WatchEvent{
		Type:      EventTypeSet,
		Key:       "app/config",
		Value:     "test-value",
		Timestamp: time.Now().Unix(),
	}

	manager.Notify(event)

	// Receive event
	select {
	case received := <-watcher.Events:
		if received.Key != "app/config" {
			t.Errorf("Expected key 'app/config', got '%s'", received.Key)
		}
		if received.Value != "test-value" {
			t.Errorf("Expected value 'test-value', got '%s'", received.Value)
		}
		if received.Type != EventTypeSet {
			t.Errorf("Expected EventTypeSet, got %s", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive event")
	}
}

func TestManager_Notify_NoMatch(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher, _ := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")

	// Send non-matching event
	event := WatchEvent{
		Type:      EventTypeSet,
		Key:       "other/key",
		Value:     "test-value",
		Timestamp: time.Now().Unix(),
	}

	manager.Notify(event)

	// Should not receive event
	select {
	case <-watcher.Events:
		t.Error("Should not receive event for non-matching key")
	case <-time.After(50 * time.Millisecond):
		// Expected - no event received
	}
}

func TestManager_Notify_SingleLevelWildcard(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher, _ := manager.AddWatcher("app/*", []string{}, TransportWebSocket, "user1")

	tests := []struct {
		name      string
		key       string
		shouldGet bool
	}{
		{"match single level", "app/config", true},
		{"match single level 2", "app/data", true},
		{"no match nested", "app/config/nested", false},
		{"no match different prefix", "other/config", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := WatchEvent{
				Type:      EventTypeSet,
				Key:       tt.key,
				Value:     "test",
				Timestamp: time.Now().Unix(),
			}

			manager.Notify(event)

			if tt.shouldGet {
				select {
				case received := <-watcher.Events:
					if received.Key != tt.key {
						t.Errorf("Expected key '%s', got '%s'", tt.key, received.Key)
					}
				case <-time.After(50 * time.Millisecond):
					t.Error("Expected to receive event")
				}
			} else {
				select {
				case <-watcher.Events:
					t.Error("Should not receive event for non-matching key")
				case <-time.After(50 * time.Millisecond):
					// Expected - no event
				}
			}
		})
	}
}

func TestManager_Notify_MultiLevelWildcard(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher, _ := manager.AddWatcher("app/**", []string{}, TransportWebSocket, "user1")

	tests := []struct {
		name      string
		key       string
		shouldGet bool
	}{
		{"match single level", "app/config", true},
		{"match nested", "app/config/nested", true},
		{"match deeply nested", "app/config/nested/deep", true},
		{"no match different prefix", "other/config", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := WatchEvent{
				Type:      EventTypeSet,
				Key:       tt.key,
				Value:     "test",
				Timestamp: time.Now().Unix(),
			}

			manager.Notify(event)

			if tt.shouldGet {
				select {
				case received := <-watcher.Events:
					if received.Key != tt.key {
						t.Errorf("Expected key '%s', got '%s'", tt.key, received.Key)
					}
				case <-time.After(50 * time.Millisecond):
					t.Error("Expected to receive event")
				}
			} else {
				select {
				case <-watcher.Events:
					t.Error("Should not receive event for non-matching key")
				case <-time.After(50 * time.Millisecond):
					// Expected
				}
			}
		})
	}
}

func TestManager_Notify_ACLFiltering(t *testing.T) {
	log := logger.GetDefault()
	evaluator := acl.NewEvaluator(log)

	// Create policy: can read app/config/*, deny app/secrets/*
	policy := &acl.Policy{
		Name: "test-policy",
		KV: []acl.KVRule{
			{
				Path:         "app/config/*",
				Capabilities: []acl.Capability{acl.CapabilityRead},
			},
			{
				Path:         "app/secrets/*",
				Capabilities: []acl.Capability{acl.CapabilityDeny},
			},
		},
	}
	if err := evaluator.AddPolicy(policy); err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	manager := NewManager(evaluator, log, 10, 0)

	// Watcher with policy attached, watching all keys
	watcher, _ := manager.AddWatcher("app/**", []string{"test-policy"}, TransportWebSocket, "user1")

	tests := []struct {
		name      string
		key       string
		shouldGet bool
	}{
		{"allowed - config", "app/config/database", true},
		{"denied - secrets", "app/secrets/password", false},
		{"no match - other", "app/data/cache", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := WatchEvent{
				Type:      EventTypeSet,
				Key:       tt.key,
				Value:     "test",
				Timestamp: time.Now().Unix(),
			}

			manager.Notify(event)

			if tt.shouldGet {
				select {
				case received := <-watcher.Events:
					if received.Key != tt.key {
						t.Errorf("Expected key '%s', got '%s'", tt.key, received.Key)
					}
				case <-time.After(50 * time.Millisecond):
					t.Error("Expected to receive event")
				}
			} else {
				select {
				case <-watcher.Events:
					t.Error("Should not receive event due to ACL or pattern")
				case <-time.After(50 * time.Millisecond):
					// Expected
				}
			}
		})
	}
}

func TestManager_Notify_MultipleWatchers(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher1, _ := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")
	watcher2, _ := manager.AddWatcher("app/*", []string{}, TransportWebSocket, "user2")
	watcher3, _ := manager.AddWatcher("other/key", []string{}, TransportWebSocket, "user3")

	event := WatchEvent{
		Type:      EventTypeSet,
		Key:       "app/config",
		Value:     "test",
		Timestamp: time.Now().Unix(),
	}

	manager.Notify(event)

	// watcher1 should receive (exact match)
	select {
	case <-watcher1.Events:
		// Expected
	case <-time.After(50 * time.Millisecond):
		t.Error("watcher1 should receive event")
	}

	// watcher2 should receive (wildcard match)
	select {
	case <-watcher2.Events:
		// Expected
	case <-time.After(50 * time.Millisecond):
		t.Error("watcher2 should receive event")
	}

	// watcher3 should NOT receive (no match)
	select {
	case <-watcher3.Events:
		t.Error("watcher3 should not receive event")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}
}

func TestManager_Notify_BufferFull(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 2, 0) // Small buffer size

	watcher, _ := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")

	// Fill buffer
	for i := 0; i < 2; i++ {
		event := WatchEvent{
			Type:      EventTypeSet,
			Key:       "app/config",
			Value:     "test",
			Timestamp: time.Now().Unix(),
		}
		manager.Notify(event)
	}

	// Send one more (should be dropped)
	event := WatchEvent{
		Type:      EventTypeSet,
		Key:       "app/config",
		Value:     "dropped",
		Timestamp: time.Now().Unix(),
	}
	manager.Notify(event)

	// Drain first two events
	<-watcher.Events
	<-watcher.Events

	// Third event should not be in channel
	select {
	case <-watcher.Events:
		t.Error("Expected channel to be empty (event dropped)")
	case <-time.After(50 * time.Millisecond):
		// Expected - event was dropped
	}
}

func TestManager_GetWatcherCountByTransport(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	if _, err := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1"); err != nil {
		t.Fatalf("Failed to add watcher 1: %v", err)
	}
	if _, err := manager.AddWatcher("app/data", []string{}, TransportWebSocket, "user2"); err != nil {
		t.Fatalf("Failed to add watcher 2: %v", err)
	}
	if _, err := manager.AddWatcher("app/cache", []string{}, TransportSSE, "user3"); err != nil {
		t.Fatalf("Failed to add watcher 3: %v", err)
	}

	wsCount := manager.GetWatcherCountByTransport(TransportWebSocket)
	if wsCount != 2 {
		t.Errorf("Expected 2 WebSocket watchers, got %d", wsCount)
	}

	sseCount := manager.GetWatcherCountByTransport(TransportSSE)
	if sseCount != 1 {
		t.Errorf("Expected 1 SSE watcher, got %d", sseCount)
	}
}

func TestManager_GetWatchersByUser(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	if _, err := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1"); err != nil {
		t.Fatalf("Failed to add watcher 1: %v", err)
	}
	if _, err := manager.AddWatcher("app/data", []string{}, TransportWebSocket, "user1"); err != nil {
		t.Fatalf("Failed to add watcher 2: %v", err)
	}
	if _, err := manager.AddWatcher("app/cache", []string{}, TransportSSE, "user2"); err != nil {
		t.Fatalf("Failed to add watcher 3: %v", err)
	}

	user1Count := manager.GetWatchersByUser("user1")
	if user1Count != 2 {
		t.Errorf("Expected 2 watchers for user1, got %d", user1Count)
	}

	user2Count := manager.GetWatchersByUser("user2")
	if user2Count != 1 {
		t.Errorf("Expected 1 watcher for user2, got %d", user2Count)
	}

	user3Count := manager.GetWatchersByUser("user3")
	if user3Count != 0 {
		t.Errorf("Expected 0 watchers for user3, got %d", user3Count)
	}
}

func TestManager_Close(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	watcher1, _ := manager.AddWatcher("app/config", []string{}, TransportWebSocket, "user1")
	watcher2, _ := manager.AddWatcher("app/data", []string{}, TransportWebSocket, "user2")

	manager.Close()

	if manager.GetActiveWatcherCount() != 0 {
		t.Errorf("Expected 0 active watchers after close, got %d", manager.GetActiveWatcherCount())
	}

	// Channels should be closed
	_, ok1 := <-watcher1.Events
	_, ok2 := <-watcher2.Events

	if ok1 || ok2 {
		t.Error("Expected all watcher channels to be closed")
	}
}

func TestManager_matchesPattern(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(nil, log, 10, 0)

	tests := []struct {
		name     string
		key      string
		pattern  string
		expected bool
	}{
		// Exact matches
		{"exact match", "app/config", "app/config", true},
		{"exact no match", "app/config", "app/data", false},

		// Single-level wildcard
		{"single wildcard match", "app/config", "app/*", true},
		{"single wildcard no match nested", "app/config/nested", "app/*", false},

		// Multi-level wildcard
		{"multi wildcard match single", "app/config", "app/**", true},
		{"multi wildcard match nested", "app/config/nested", "app/**", true},
		{"multi wildcard match deep", "app/config/nested/deep", "app/**", true},
		{"multi wildcard no match", "other/config", "app/**", false},

		// No wildcards
		{"no wildcard match", "exact", "exact", true},
		{"no wildcard no match", "exact", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.matchesPattern(tt.key, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%s, %s) = %v, expected %v",
					tt.key, tt.pattern, result, tt.expected)
			}
		})
	}
}
