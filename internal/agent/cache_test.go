package agent

import (
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/store"
)

func TestCache_ServiceOperations(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     100 * time.Millisecond,
		KVTTL:          100 * time.Millisecond,
		HealthTTL:      100 * time.Millisecond,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Test cache miss
	_, ok := cache.GetService("test-service")
	if ok {
		t.Error("Expected cache miss, got hit")
	}

	// Verify miss count
	if cache.Misses() != 1 {
		t.Errorf("Expected 1 miss, got %d", cache.Misses())
	}

	// Test cache set and hit
	entries := []*store.ServiceEntry{
		{
			Service: store.Service{
				Name:    "test-service",
				Address: "127.0.0.1",
				Port:    8080,
			},
			ModifyIndex: 1,
			CreateIndex: 1,
		},
	}

	cache.SetService("test-service", entries)

	retrieved, ok := cache.GetService("test-service")
	if !ok {
		t.Error("Expected cache hit, got miss")
	}

	if len(retrieved) != 1 || retrieved[0].Service.Name != "test-service" {
		t.Error("Retrieved service does not match")
	}

	// Verify hit count
	if cache.Hits() != 1 {
		t.Errorf("Expected 1 hit, got %d", cache.Hits())
	}

	// Test cache deletion
	cache.DeleteService("test-service")
	_, ok = cache.GetService("test-service")
	if ok {
		t.Error("Expected cache miss after delete, got hit")
	}
}

func TestCache_ServiceTTL(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     50 * time.Millisecond,
		KVTTL:          100 * time.Millisecond,
		HealthTTL:      100 * time.Millisecond,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	entries := []*store.ServiceEntry{
		{
			Service: store.Service{
				Name:    "test-service",
				Address: "127.0.0.1",
				Port:    8080,
			},
		},
	}

	cache.SetService("test-service", entries)

	// Verify entry exists
	_, ok := cache.GetService("test-service")
	if !ok {
		t.Error("Expected cache hit")
	}

	// Wait for TTL expiration
	time.Sleep(100 * time.Millisecond)

	// Verify entry expired
	_, ok = cache.GetService("test-service")
	if ok {
		t.Error("Expected cache miss after TTL expiration, got hit")
	}
}

func TestCache_KVOperations(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     100 * time.Millisecond,
		KVTTL:          100 * time.Millisecond,
		HealthTTL:      100 * time.Millisecond,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Test cache miss
	_, ok := cache.GetKV("test-key")
	if ok {
		t.Error("Expected cache miss, got hit")
	}

	// Test cache set and hit
	entry := &store.KVEntry{
		Value:       "test-value",
		ModifyIndex: 1,
		CreateIndex: 1,
	}

	cache.SetKV("test-key", entry)

	retrieved, ok := cache.GetKV("test-key")
	if !ok {
		t.Error("Expected cache hit, got miss")
	}

	if retrieved.Value != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", retrieved.Value)
	}

	// Test cache deletion
	cache.DeleteKV("test-key")
	_, ok = cache.GetKV("test-key")
	if ok {
		t.Error("Expected cache miss after delete, got hit")
	}
}

func TestCache_HealthOperations(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     100 * time.Millisecond,
		KVTTL:          100 * time.Millisecond,
		HealthTTL:      100 * time.Millisecond,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Test cache miss
	_, ok := cache.GetHealth("check-1")
	if ok {
		t.Error("Expected cache miss, got hit")
	}

	// Test cache set and hit
	result := &HealthCheckResult{
		Status:    HealthStatusPassing,
		Output:    "All good",
		Timestamp: time.Now(),
	}

	cache.SetHealth("check-1", result)

	retrieved, ok := cache.GetHealth("check-1")
	if !ok {
		t.Error("Expected cache hit, got miss")
	}

	if retrieved.Status != HealthStatusPassing {
		t.Errorf("Expected status 'passing', got '%s'", retrieved.Status)
	}
}

func TestCache_ApplyServiceUpdate(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     100 * time.Millisecond,
		KVTTL:          100 * time.Millisecond,
		HealthTTL:      100 * time.Millisecond,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Test ADD update
	update := ServiceUpdate{
		Type:        UpdateTypeAdd,
		ServiceName: "web-service",
		Entry: &store.ServiceEntry{
			Service: store.Service{
				Name:    "web-service",
				Address: "10.0.0.1",
				Port:    80,
			},
		},
	}

	cache.ApplyServiceUpdate(update)

	entries, ok := cache.GetService("web-service")
	if !ok {
		t.Error("Expected service to be cached after ADD update")
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// Test UPDATE
	update.Type = UpdateTypeUpdate
	update.Entry.Service.Port = 8080
	cache.ApplyServiceUpdate(update)

	entries, ok = cache.GetService("web-service")
	if !ok {
		t.Error("Expected service to be cached after UPDATE")
	}
	if entries[0].Service.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", entries[0].Service.Port)
	}

	// Test DELETE
	update.Type = UpdateTypeDelete
	cache.ApplyServiceUpdate(update)

	_, ok = cache.GetService("web-service")
	if ok {
		t.Error("Expected service to be removed after DELETE update")
	}
}

func TestCache_ApplyKVUpdate(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     100 * time.Millisecond,
		KVTTL:          100 * time.Millisecond,
		HealthTTL:      100 * time.Millisecond,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Test ADD update
	update := KVUpdate{
		Type: UpdateTypeAdd,
		Key:  "config/app",
		Entry: &store.KVEntry{
			Value: "production",
		},
	}

	cache.ApplyKVUpdate(update)

	entry, ok := cache.GetKV("config/app")
	if !ok {
		t.Error("Expected KV to be cached after ADD update")
	}
	if entry.Value != "production" {
		t.Errorf("Expected value 'production', got '%s'", entry.Value)
	}

	// Test DELETE
	update.Type = UpdateTypeDelete
	cache.ApplyKVUpdate(update)

	_, ok = cache.GetKV("config/app")
	if ok {
		t.Error("Expected KV to be removed after DELETE update")
	}
}

func TestCache_HitRate(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     1 * time.Second,
		KVTTL:          1 * time.Second,
		HealthTTL:      1 * time.Second,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Initial hit rate should be 0
	if cache.HitRate() != 0 {
		t.Errorf("Expected initial hit rate 0, got %f", cache.HitRate())
	}

	// Add an entry
	cache.SetService("test", []*store.ServiceEntry{{Service: store.Service{Name: "test"}}})

	// 1 hit, 0 misses = 100% hit rate
	cache.GetService("test")
	if cache.HitRate() != 1.0 {
		t.Errorf("Expected hit rate 1.0, got %f", cache.HitRate())
	}

	// 1 hit, 1 miss = 50% hit rate
	cache.GetService("nonexistent")
	if cache.HitRate() != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", cache.HitRate())
	}
}

func TestCache_Counts(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     1 * time.Second,
		KVTTL:          1 * time.Second,
		HealthTTL:      1 * time.Second,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Add entries
	cache.SetService("svc1", []*store.ServiceEntry{{Service: store.Service{Name: "svc1"}}})
	cache.SetService("svc2", []*store.ServiceEntry{{Service: store.Service{Name: "svc2"}}})
	cache.SetKV("key1", &store.KVEntry{Value: "val1"})
	cache.SetHealth("check1", &HealthCheckResult{Status: HealthStatusPassing})

	if cache.ServiceCount() != 2 {
		t.Errorf("Expected 2 services, got %d", cache.ServiceCount())
	}

	if cache.KVCount() != 1 {
		t.Errorf("Expected 1 KV entry, got %d", cache.KVCount())
	}

	if cache.HealthCount() != 1 {
		t.Errorf("Expected 1 health entry, got %d", cache.HealthCount())
	}

	if cache.Len() != 4 {
		t.Errorf("Expected total 4 entries, got %d", cache.Len())
	}
}

func TestCache_Clear(t *testing.T) {
	cfg := CacheConfig{
		ServiceTTL:     1 * time.Second,
		KVTTL:          1 * time.Second,
		HealthTTL:      1 * time.Second,
		MaxEntries:     100,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Add entries
	cache.SetService("svc1", []*store.ServiceEntry{{Service: store.Service{Name: "svc1"}}})
	cache.SetKV("key1", &store.KVEntry{Value: "val1"})
	cache.GetService("svc1") // Generate a hit

	// Clear cache
	cache.Clear()

	if cache.Len() != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", cache.Len())
	}

	if cache.Hits() != 0 {
		t.Errorf("Expected 0 hits after clear, got %d", cache.Hits())
	}

	if cache.Misses() != 0 {
		t.Errorf("Expected 0 misses after clear, got %d", cache.Misses())
	}
}
