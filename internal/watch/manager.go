package watch

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
)

// Manager manages all active watchers
type Manager struct {
	watchers     map[string]*Watcher // ID -> Watcher
	patterns     map[string][]string // Pattern -> []WatcherID
	mu           sync.RWMutex
	aclEval      *acl.Evaluator
	log          logger.Logger
	bufferSize   int
	maxPerClient int
	clientCounts map[string]int // UserID -> count
}

// NewManager creates a new watch manager
func NewManager(aclEval *acl.Evaluator, log logger.Logger, bufferSize, maxPerClient int) *Manager {
	return &Manager{
		watchers:     make(map[string]*Watcher),
		patterns:     make(map[string][]string),
		aclEval:      aclEval,
		log:          log,
		bufferSize:   bufferSize,
		maxPerClient: maxPerClient,
		clientCounts: make(map[string]int),
	}
}

// AddWatcher adds a new watcher for the given pattern
func (wm *Manager) AddWatcher(pattern string, policies []string, transport TransportType, userID string) (*Watcher, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Check per-client limit
	if wm.maxPerClient > 0 && userID != "" {
		if wm.clientCounts[userID] >= wm.maxPerClient {
			wm.log.Warn("Client exceeded max watchers",
				logger.String("user_id", userID),
				logger.Int("current", wm.clientCounts[userID]),
				logger.Int("max", wm.maxPerClient))
			return nil, ErrTooManyWatchers
		}
	}

	// Create watcher
	watcher := NewWatcher(
		uuid.New().String(),
		pattern,
		policies,
		transport,
		wm.bufferSize,
	)
	watcher.UserID = userID

	// Add to maps
	wm.watchers[watcher.ID] = watcher
	wm.patterns[pattern] = append(wm.patterns[pattern], watcher.ID)

	// Increment client count
	if userID != "" {
		wm.clientCounts[userID]++
	}

	wm.log.Info("Watcher added",
		logger.String("id", watcher.ID),
		logger.String("pattern", pattern),
		logger.String("transport", string(transport)),
		logger.String("user_id", userID))

	// Update metrics
	metrics.WatchersActive.WithLabelValues(string(transport)).Inc()
	metrics.WatchConnectionsTotal.WithLabelValues(string(transport), "opened").Inc()

	return watcher, nil
}

// RemoveWatcher removes a watcher by ID
func (wm *Manager) RemoveWatcher(id string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	watcher, exists := wm.watchers[id]
	if !exists {
		return
	}

	// Remove from patterns map
	ids := wm.patterns[watcher.Pattern]
	for i, wid := range ids {
		if wid == id {
			wm.patterns[watcher.Pattern] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	// Clean up empty pattern entry
	if len(wm.patterns[watcher.Pattern]) == 0 {
		delete(wm.patterns, watcher.Pattern)
	}

	// Decrement client count
	if watcher.UserID != "" {
		wm.clientCounts[watcher.UserID]--
		if wm.clientCounts[watcher.UserID] == 0 {
			delete(wm.clientCounts, watcher.UserID)
		}
	}

	// Close channel and remove
	close(watcher.Events)
	delete(wm.watchers, id)

	// Update metrics
	metrics.WatchersActive.WithLabelValues(string(watcher.Transport)).Dec()
	metrics.WatchConnectionsTotal.WithLabelValues(string(watcher.Transport), "closed").Inc()

	wm.log.Info("Watcher removed",
		logger.String("id", id),
		logger.String("pattern", watcher.Pattern))
}

// Notify sends an event to all matching watchers
func (wm *Manager) Notify(event WatchEvent) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	notified := 0
	dropped := 0

	// Find all watchers matching this key
	for pattern, watcherIDs := range wm.patterns {
		if wm.matchesPattern(event.Key, pattern) {
			for _, id := range watcherIDs {
				watcher, exists := wm.watchers[id]
				if !exists {
					continue
				}

				// Check ACL permissions
				if !wm.canWatch(watcher, event.Key) {
					wm.log.Debug("Watcher ACL check failed",
						logger.String("watcher_id", id),
						logger.String("key", event.Key))
					continue
				}

				// Send event (non-blocking)
				select {
				case watcher.Events <- event:
					notified++
					metrics.WatchEventsTotal.WithLabelValues(string(event.Type)).Inc()
				default:
					// Channel full, log warning and drop event
					dropped++
					metrics.WatchEventsDropped.WithLabelValues("channel_full").Inc()
					wm.log.Warn("Watcher channel full, dropping event",
						logger.String("watcher_id", id),
						logger.String("pattern", pattern),
						logger.String("key", event.Key),
						logger.String("event_type", string(event.Type)))
				}
			}
		}
	}

	if notified > 0 || dropped > 0 {
		wm.log.Debug("Watch event notified",
			logger.String("key", event.Key),
			logger.String("event_type", string(event.Type)),
			logger.Int("notified", notified),
			logger.Int("dropped", dropped))
	}
}

// matchesPattern checks if a key matches a watch pattern
func (wm *Manager) matchesPattern(key, pattern string) bool {
	// Exact match
	if key == pattern {
		return true
	}

	// No wildcards - only exact match works
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Handle ** (multi-level wildcard)
	if strings.Contains(pattern, "**") {
		prefix := strings.TrimSuffix(pattern, "**")
		return strings.HasPrefix(key, prefix)
	}

	// Handle * (single-level wildcard) - use filepath.Match
	matched, err := filepath.Match(pattern, key)
	if err != nil {
		wm.log.Warn("Invalid pattern",
			logger.String("pattern", pattern),
			logger.Error(err))
		return false
	}

	return matched
}

// canWatch checks if a watcher has permission to watch a key
func (wm *Manager) canWatch(watcher *Watcher, key string) bool {
	if wm.aclEval == nil {
		// ACL not enabled, allow all
		return true
	}

	// Check if watcher's policies allow reading this key
	resource := acl.NewKVResource(key)
	return wm.aclEval.Evaluate(watcher.ACLPolicies, resource, acl.CapabilityRead)
}

// GetActiveWatcherCount returns the number of active watchers
func (wm *Manager) GetActiveWatcherCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.watchers)
}

// GetWatcherCountByTransport returns the number of watchers by transport type
func (wm *Manager) GetWatcherCountByTransport(transport TransportType) int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	count := 0
	for _, watcher := range wm.watchers {
		if watcher.Transport == transport {
			count++
		}
	}
	return count
}

// GetWatchersByUser returns the number of watchers for a user
func (wm *Manager) GetWatchersByUser(userID string) int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.clientCounts[userID]
}

// Close closes all watchers and cleans up resources
func (wm *Manager) Close() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	for id, watcher := range wm.watchers {
		close(watcher.Events)
		delete(wm.watchers, id)
	}

	wm.patterns = make(map[string][]string)
	wm.clientCounts = make(map[string]int)

	wm.log.Info("Watch manager closed")
}
