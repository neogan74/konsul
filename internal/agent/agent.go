package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

// Agent represents a Konsul agent instance
type Agent struct {
	config *Config
	info   AgentInfo
	log    logger.Logger

	// Components
	cache        *Cache
	syncEngine   *SyncEngine
	serverClient *ServerClient
	api          *API

	// Local state
	localServices map[string]*store.ServiceEntry
	mu            sync.RWMutex

	// Lifecycle
	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time
	wg        sync.WaitGroup
}

// NewAgent creates a new agent with the given configuration
func NewAgent(cfg *Config, log logger.Logger) (*Agent, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Generate agent ID if not provided
	if cfg.ID == "" || cfg.ID == "agent-default" {
		cfg.ID = generateAgentID(cfg.NodeName)
	}

	// Build agent info
	info := AgentInfo{
		ID:         cfg.ID,
		NodeName:   cfg.NodeName,
		NodeIP:     cfg.NodeIP,
		Datacenter: cfg.Datacenter,
		Metadata:   cfg.Metadata,
		StartedAt:  time.Now(),
		Version:    "0.1.0", // TODO: Get from build info
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create server client
	serverClient, err := NewServerClient(cfg.ServerAddress, cfg.TLS, cfg.ID)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create server client: %w", err)
	}

	// Create cache
	cache := NewCache(cfg.Cache)

	// Create sync engine
	syncEngine := NewSyncEngine(cfg.Sync, log)

	agent := &Agent{
		config:        cfg,
		info:          info,
		log:           log,
		cache:         cache,
		syncEngine:    syncEngine,
		serverClient:  serverClient,
		localServices: make(map[string]*store.ServiceEntry),
		ctx:           ctx,
		cancel:        cancel,
		startTime:     time.Now(),
	}

	// Create API server
	agent.api = NewAPI(agent)

	return agent, nil
}

// Start starts the agent and all its components
func (a *Agent) Start() error {
	a.log.Info("Starting Konsul agent",
		logger.String("id", a.info.ID),
		logger.String("node", a.info.NodeName),
		logger.String("server", a.config.ServerAddress))

	// Start API server
	if err := a.api.Start(); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	// Register with server
	if err := a.registerWithServer(); err != nil {
		return fmt.Errorf("failed to register with server: %w", err)
	}

	// Start sync engine
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.syncEngine.Run(a.ctx, a.serverClient, a.cache, a.config.WatchedPrefixes)
	}()

	a.log.Info("Agent started successfully")
	return nil
}

// Stop stops the agent gracefully
func (a *Agent) Stop() error {
	a.log.Info("Stopping Konsul agent")

	// Stop API server first
	if err := a.api.Stop(); err != nil {
		a.log.Error("Failed to stop API server", logger.Error(err))
	}

	// Signal shutdown
	a.cancel()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.log.Info("All goroutines stopped")
	case <-time.After(30 * time.Second):
		a.log.Warn("Timeout waiting for goroutines to stop")
	}

	// Close server client
	a.serverClient.Close()

	a.log.Info("Agent stopped")
	return nil
}

// registerWithServer registers the agent with the Konsul server
func (a *Agent) registerWithServer() error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if err := a.serverClient.RegisterAgent(ctx, a.info); err != nil {
		return err
	}

	a.log.Info("Registered with server", logger.String("agent_id", a.info.ID))
	return nil
}

// Service operations

// RegisterService registers a service locally
func (a *Agent) RegisterService(svc store.Service) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Generate service ID if not provided
	serviceID := fmt.Sprintf("%s:%s:%d", a.info.NodeName, svc.Name, svc.Port)

	entry := &store.ServiceEntry{
		Service:     svc,
		ExpiresAt:   time.Now().Add(a.config.Cache.ServiceTTL),
		ModifyIndex: uint64(time.Now().UnixNano()),
		CreateIndex: uint64(time.Now().UnixNano()),
	}

	a.localServices[serviceID] = entry

	// Queue for sync
	a.syncEngine.QueueServiceUpdate(ServiceUpdate{
		Type:        UpdateTypeAdd,
		ServiceName: svc.Name,
		Service:     &svc,
		Entry:       entry,
	})

	a.log.Info("Service registered locally",
		logger.String("service", svc.Name),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port))

	return nil
}

// DeregisterService deregisters a service
func (a *Agent) DeregisterService(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find and remove local service
	var found bool
	for id, entry := range a.localServices {
		if entry.Service.Name == name {
			delete(a.localServices, id)
			found = true

			// Queue for sync
			a.syncEngine.QueueServiceUpdate(ServiceUpdate{
				Type:        UpdateTypeDelete,
				ServiceName: name,
			})
			break
		}
	}

	if !found {
		return fmt.Errorf("service not found: %s", name)
	}

	a.log.Info("Service deregistered", logger.String("service", name))
	return nil
}

// GetService retrieves service entries (from cache or server)
func (a *Agent) GetService(name string) ([]*store.ServiceEntry, error) {
	// Try cache first
	if entries, ok := a.cache.GetService(name); ok {
		a.log.Debug("Service found in cache", logger.String("service", name))
		return entries, nil
	}

	// Cache miss - fetch from server
	a.log.Debug("Service cache miss, fetching from server", logger.String("service", name))

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	entries, err := a.serverClient.GetService(ctx, name)
	if err != nil {
		return nil, err
	}

	// Update cache
	if entries != nil {
		a.cache.SetService(name, entries)
	}

	return entries, nil
}

// ListLocalServices returns all locally registered services
func (a *Agent) ListLocalServices() []*store.ServiceEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entries := make([]*store.ServiceEntry, 0, len(a.localServices))
	for _, entry := range a.localServices {
		entries = append(entries, entry)
	}

	return entries
}

// KV operations

// GetKV retrieves a KV entry (from cache or server)
func (a *Agent) GetKV(key string) (*store.KVEntry, error) {
	// Try cache first
	if entry, ok := a.cache.GetKV(key); ok {
		a.log.Debug("KV found in cache", logger.String("key", key))
		return entry, nil
	}

	// Cache miss - fetch from server
	a.log.Debug("KV cache miss, fetching from server", logger.String("key", key))

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	entry, err := a.serverClient.GetKV(ctx, key)
	if err != nil {
		return nil, err
	}

	// Update cache
	if entry != nil {
		a.cache.SetKV(key, entry)
	}

	return entry, nil
}

// SetKV sets a KV entry (write-through cache)
func (a *Agent) SetKV(key string, entry *store.KVEntry) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	// Write to server first
	if err := a.serverClient.SetKV(ctx, key, entry); err != nil {
		return err
	}

	// Update cache
	a.cache.SetKV(key, entry)

	a.log.Debug("KV entry set", logger.String("key", key))
	return nil
}

// DeleteKV deletes a KV entry
func (a *Agent) DeleteKV(key string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	// Delete from server
	if err := a.serverClient.DeleteKV(ctx, key); err != nil {
		return err
	}

	// Remove from cache
	a.cache.DeleteKV(key)

	a.log.Debug("KV entry deleted", logger.String("key", key))
	return nil
}

// Agent information

// Info returns agent information
func (a *Agent) Info() AgentInfo {
	return a.info
}

// Stats returns agent statistics
func (a *Agent) Stats() AgentStats {
	a.mu.RLock()
	localServiceCount := len(a.localServices)
	a.mu.RUnlock()

	return AgentStats{
		CacheHitRate:    a.cache.HitRate(),
		CacheEntries:    a.cache.Len(),
		LocalServices:   localServiceCount,
		LastSyncTime:    a.syncEngine.GetLastSyncTime(),
		SyncErrorsTotal: int64(a.syncEngine.GetSyncErrors()),
		Uptime:          time.Since(a.startTime).String(),
	}
}

// Health returns agent health status
func (a *Agent) Health() bool {
	// Agent is healthy if:
	// 1. Context is not cancelled
	// 2. Last sync was recent (within 2x sync interval)
	// 3. Not too many sync errors

	select {
	case <-a.ctx.Done():
		return false
	default:
	}

	lastSync := a.syncEngine.GetLastSyncTime()
	if !lastSync.IsZero() && time.Since(lastSync) > 2*a.config.Sync.Interval {
		return false
	}

	syncErrors := a.syncEngine.GetSyncErrors()
	syncCount := a.syncEngine.GetSyncCount()
	if syncCount > 0 && float64(syncErrors)/float64(syncCount) > 0.5 {
		return false
	}

	return true
}

// Helper functions

// generateAgentID generates a unique agent ID
func generateAgentID(nodeName string) string {
	// Generate random suffix
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("agent-%s-%d", nodeName, time.Now().UnixNano())
	}

	randomHex := hex.EncodeToString(randomBytes)

	// Get hostname if nodeName is empty
	if nodeName == "" {
		hostname, err := os.Hostname()
		if err == nil {
			nodeName = hostname
		} else {
			nodeName = "unknown"
		}
	}

	return fmt.Sprintf("agent-%s-%s", nodeName, randomHex)
}
