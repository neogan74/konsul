package agent

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/neogan74/konsul/internal/store"
)

// Cache manages local caching of services, KV entries, and health check results
type Cache struct {
	// LRU caches with expiration
	services *expirable.LRU[string, []*store.ServiceEntry]
	kv       *expirable.LRU[string, *store.KVEntry]
	health   *expirable.LRU[string, *HealthCheckResult]

	mu sync.RWMutex

	// Metrics
	hits   uint64
	misses uint64
}

// HealthCheckResult represents a cached health check result
type HealthCheckResult struct {
	Status    HealthStatus
	Output    string
	Timestamp time.Time
}

// NewCache creates a new cache with the given configuration
func NewCache(cfg CacheConfig) *Cache {
	return &Cache{
		services: expirable.NewLRU[string, []*store.ServiceEntry](
			cfg.MaxEntries,
			nil, // onEvict callback
			cfg.ServiceTTL,
		),
		kv: expirable.NewLRU[string, *store.KVEntry](
			cfg.MaxEntries,
			nil,
			cfg.KVTTL,
		),
		health: expirable.NewLRU[string, *HealthCheckResult](
			cfg.MaxEntries,
			nil,
			cfg.HealthTTL,
		),
	}
}

// Service cache operations

// GetService retrieves services by name from cache
func (c *Cache) GetService(name string) ([]*store.ServiceEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries, ok := c.services.Get(name)
	if ok {
		atomic.AddUint64(&c.hits, 1)
		return entries, true
	}

	atomic.AddUint64(&c.misses, 1)
	return nil, false
}

// SetService stores service entries in cache
func (c *Cache) SetService(name string, entries []*store.ServiceEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services.Add(name, entries)
}

// DeleteService removes a service from cache
func (c *Cache) DeleteService(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services.Remove(name)
}

// ApplyServiceUpdate applies a service update to the cache
func (c *Cache) ApplyServiceUpdate(update ServiceUpdate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch update.Type {
	case UpdateTypeAdd, UpdateTypeUpdate:
		if update.Entry != nil {
			// Get existing entries for this service name
			existing, _ := c.services.Get(update.ServiceName)

			// Append or update the entry
			found := false
			for i, entry := range existing {
				if entry.Service.Name == update.ServiceName &&
					entry.Service.Address == update.Entry.Service.Address &&
					entry.Service.Port == update.Entry.Service.Port {
					existing[i] = update.Entry
					found = true
					break
				}
			}

			if !found {
				existing = append(existing, update.Entry)
			}

			c.services.Add(update.ServiceName, existing)
		}
	case UpdateTypeDelete:
		c.services.Remove(update.ServiceName)
	}
}

// KV cache operations

// GetKV retrieves a KV entry from cache
func (c *Cache) GetKV(key string) (*store.KVEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.kv.Get(key)
	if ok {
		atomic.AddUint64(&c.hits, 1)
		return entry, true
	}

	atomic.AddUint64(&c.misses, 1)
	return nil, false
}

// SetKV stores a KV entry in cache
func (c *Cache) SetKV(key string, entry *store.KVEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.kv.Add(key, entry)
}

// DeleteKV removes a KV entry from cache
func (c *Cache) DeleteKV(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.kv.Remove(key)
}

// ApplyKVUpdate applies a KV update to the cache
func (c *Cache) ApplyKVUpdate(update KVUpdate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch update.Type {
	case UpdateTypeAdd, UpdateTypeUpdate:
		if update.Entry != nil {
			c.kv.Add(update.Key, update.Entry)
		}
	case UpdateTypeDelete:
		c.kv.Remove(update.Key)
	}
}

// Health cache operations

// GetHealth retrieves a health check result from cache
func (c *Cache) GetHealth(checkID string) (*HealthCheckResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result, ok := c.health.Get(checkID)
	if ok {
		atomic.AddUint64(&c.hits, 1)
		return result, true
	}

	atomic.AddUint64(&c.misses, 1)
	return nil, false
}

// SetHealth stores a health check result in cache
func (c *Cache) SetHealth(checkID string, result *HealthCheckResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.health.Add(checkID, result)
}

// ApplyHealthUpdate applies a health update to the cache
func (c *Cache) ApplyHealthUpdate(update HealthUpdate) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := &HealthCheckResult{
		Status:    update.Status,
		Output:    update.Output,
		Timestamp: time.Now(),
	}
	c.health.Add(update.CheckID, result)
}

// Cache statistics

// HitRate returns the cache hit rate
func (c *Cache) HitRate() float64 {
	hits := atomic.LoadUint64(&c.hits)
	misses := atomic.LoadUint64(&c.misses)
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total)
}

// Hits returns total cache hits
func (c *Cache) Hits() uint64 {
	return atomic.LoadUint64(&c.hits)
}

// Misses returns total cache misses
func (c *Cache) Misses() uint64 {
	return atomic.LoadUint64(&c.misses)
}

// Len returns the total number of cached entries
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.services.Len() + c.kv.Len() + c.health.Len()
}

// ServiceCount returns the number of cached service entries
func (c *Cache) ServiceCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.services.Len()
}

// KVCount returns the number of cached KV entries
func (c *Cache) KVCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.kv.Len()
}

// HealthCount returns the number of cached health check results
func (c *Cache) HealthCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.health.Len()
}

// Clear removes all entries from all caches
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.services.Purge()
	c.kv.Purge()
	c.health.Purge()

	atomic.StoreUint64(&c.hits, 0)
	atomic.StoreUint64(&c.misses, 0)
}
