package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neogan74/konsul/internal/logger"
)

// SyncEngine handles periodic synchronization with the server
type SyncEngine struct {
	config       SyncConfig
	lastIndex    int64
	pendingQueue chan ServiceUpdate
	batchBuffer  []ServiceUpdate
	mu           sync.Mutex
	log          logger.Logger

	// Metrics
	syncCount    uint64
	syncErrors   uint64
	lastSyncTime time.Time
	lastSyncMu   sync.RWMutex
}

// NewSyncEngine creates a new sync engine
func NewSyncEngine(cfg SyncConfig, log logger.Logger) *SyncEngine {
	return &SyncEngine{
		config:       cfg,
		lastIndex:    0,
		pendingQueue: make(chan ServiceUpdate, 1000),
		batchBuffer:  make([]ServiceUpdate, 0, cfg.BatchSize),
		log:          log,
	}
}

// Run starts the sync engine
func (s *SyncEngine) Run(ctx context.Context, client *ServerClient, cache *Cache, watchedPrefixes []string) {
	syncTicker := time.NewTicker(s.config.Interval)
	defer syncTicker.Stop()

	fullSyncTicker := time.NewTicker(s.config.FullSyncInterval)
	defer fullSyncTicker.Stop()

	// Perform initial sync
	if err := s.performSync(ctx, client, cache, watchedPrefixes, false); err != nil {
		s.log.Error("Initial sync failed", logger.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			s.log.Info("Sync engine stopping")
			// Flush any pending updates before stopping
			if len(s.batchBuffer) > 0 {
				_ = s.flushBatch(ctx, client)
			}
			return

		case <-syncTicker.C:
			// Periodic delta sync
			if err := s.performSync(ctx, client, cache, watchedPrefixes, false); err != nil {
				s.log.Error("Periodic sync failed", logger.Error(err))
				atomic.AddUint64(&s.syncErrors, 1)
			}

		case <-fullSyncTicker.C:
			// Periodic full sync (safety net)
			s.log.Debug("Performing full sync")
			if err := s.performSync(ctx, client, cache, watchedPrefixes, true); err != nil {
				s.log.Error("Full sync failed", logger.Error(err))
				atomic.AddUint64(&s.syncErrors, 1)
			}

		case update := <-s.pendingQueue:
			// Buffer updates
			s.mu.Lock()
			s.batchBuffer = append(s.batchBuffer, update)
			shouldFlush := len(s.batchBuffer) >= s.config.BatchSize
			s.mu.Unlock()

			// Flush if batch is full
			if shouldFlush {
				if err := s.flushBatch(ctx, client); err != nil {
					s.log.Error("Failed to flush batch", logger.Error(err))
				}
			}
		}
	}
}

// performSync performs a sync operation with the server
func (s *SyncEngine) performSync(ctx context.Context, client *ServerClient, cache *Cache, watchedPrefixes []string, fullSync bool) error {
	startTime := time.Now()

	// Build sync request
	req := SyncRequest{
		AgentID:         client.agentID,
		LastSyncIndex:   atomic.LoadInt64(&s.lastIndex),
		WatchedPrefixes: watchedPrefixes,
		FullSync:        fullSync,
	}

	// Send sync request
	resp, err := client.Sync(ctx, req)
	if err != nil {
		return err
	}

	// Apply updates to cache
	s.applyUpdates(cache, resp)

	// Update last sync index
	atomic.StoreInt64(&s.lastIndex, resp.CurrentIndex)

	// Update metrics
	atomic.AddUint64(&s.syncCount, 1)
	s.lastSyncMu.Lock()
	s.lastSyncTime = time.Now()
	s.lastSyncMu.Unlock()

	duration := time.Since(startTime)
	s.log.Debug("Sync completed",
		logger.String("duration", duration.String()),
		logger.String("index", string(rune(resp.CurrentIndex))),
		logger.Int("service_updates", len(resp.ServiceUpdates)),
		logger.Int("kv_updates", len(resp.KVUpdates)),
		logger.Int("health_updates", len(resp.HealthUpdates)))

	return nil
}

// applyUpdates applies sync response updates to the cache
func (s *SyncEngine) applyUpdates(cache *Cache, resp *SyncResponse) {
	// Apply service updates
	for _, update := range resp.ServiceUpdates {
		cache.ApplyServiceUpdate(update)
	}

	// Apply KV updates
	for _, update := range resp.KVUpdates {
		cache.ApplyKVUpdate(update)
	}

	// Apply health updates
	for _, update := range resp.HealthUpdates {
		cache.ApplyHealthUpdate(update)
	}
}

// flushBatch sends batched updates to the server
func (s *SyncEngine) flushBatch(ctx context.Context, client *ServerClient) error {
	s.mu.Lock()
	if len(s.batchBuffer) == 0 {
		s.mu.Unlock()
		return nil
	}

	// Copy buffer to avoid holding lock during network call
	updates := make([]ServiceUpdate, len(s.batchBuffer))
	copy(updates, s.batchBuffer)
	s.batchBuffer = s.batchBuffer[:0] // Clear buffer
	s.mu.Unlock()

	// Send batch to server with retry
	var err error
	for attempt := 0; attempt <= s.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			s.log.Debug("Retrying batch update",
				logger.Int("attempt", attempt),
				logger.Int("max_attempts", s.config.RetryAttempts))
			time.Sleep(s.config.RetryDelay)
		}

		err = client.BatchUpdate(ctx, updates)
		if err == nil {
			s.log.Debug("Batch update sent",
				logger.Int("count", len(updates)))
			return nil
		}

		s.log.Warn("Batch update failed",
			logger.Int("attempt", attempt),
			logger.Error(err))
	}

	return err
}

// QueueServiceUpdate queues a service update for batching
func (s *SyncEngine) QueueServiceUpdate(update ServiceUpdate) {
	select {
	case s.pendingQueue <- update:
		// Successfully queued
	default:
		// Queue full, log warning
		s.log.Warn("Sync queue full, dropping update",
			logger.String("service", update.ServiceName),
			logger.String("type", string(update.Type)))
	}
}

// GetLastSyncTime returns the time of the last successful sync
func (s *SyncEngine) GetLastSyncTime() time.Time {
	s.lastSyncMu.RLock()
	defer s.lastSyncMu.RUnlock()
	return s.lastSyncTime
}

// GetSyncCount returns the total number of syncs performed
func (s *SyncEngine) GetSyncCount() uint64 {
	return atomic.LoadUint64(&s.syncCount)
}

// GetSyncErrors returns the total number of sync errors
func (s *SyncEngine) GetSyncErrors() uint64 {
	return atomic.LoadUint64(&s.syncErrors)
}

// GetLastIndex returns the last sync index
func (s *SyncEngine) GetLastIndex() int64 {
	return atomic.LoadInt64(&s.lastIndex)
}

// GetPendingCount returns the number of pending updates in the queue
func (s *SyncEngine) GetPendingCount() int {
	return len(s.pendingQueue)
}
