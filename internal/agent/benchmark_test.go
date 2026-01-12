package agent

import (
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

// BenchmarkCacheServiceHit benchmarks service cache hit performance
func BenchmarkCacheServiceHit(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Pre-populate cache
	entries := []*store.ServiceEntry{
		{
			Service: store.Service{
				Name:    "test-service",
				Address: "10.0.0.1",
				Port:    8080,
			},
		},
	}
	cache.SetService("test-service", entries)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.GetService("test-service")
	}
}

// BenchmarkCacheServiceMiss benchmarks service cache miss performance
func BenchmarkCacheServiceMiss(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.GetService("nonexistent-service")
	}
}

// BenchmarkCacheKVHit benchmarks KV cache hit performance
func BenchmarkCacheKVHit(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Pre-populate cache
	entry := &store.KVEntry{
		Value: "test-value",
	}
	cache.SetKV("config/app", entry)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.GetKV("config/app")
	}
}

// BenchmarkCacheServiceSetMany benchmarks setting many services
func BenchmarkCacheServiceSetMany(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			entries := []*store.ServiceEntry{
				{
					Service: store.Service{
						Name:    "test-service",
						Address: "10.0.0.1",
						Port:    8080 + j,
					},
				},
			}
			cache.SetService("test-service", entries)
		}
	}
}

// BenchmarkAgentRegisterService benchmarks service registration
func BenchmarkAgentRegisterService(b *testing.B) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "bench-agent"
	cfg.NodeName = "bench-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	svc := store.Service{
		Name:    "bench-service",
		Address: "10.0.0.1",
		Port:    8080,
		Tags:    []string{"benchmark"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = agent.RegisterService(svc)
	}
}

// BenchmarkAgentDeregisterService benchmarks service deregistration
func BenchmarkAgentDeregisterService(b *testing.B) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "bench-agent"
	cfg.NodeName = "bench-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		b.Fatalf("Failed to create agent: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		svc := store.Service{
			Name:    "bench-service",
			Address: "10.0.0.1",
			Port:    8080,
		}
		_ = agent.RegisterService(svc)
		b.StartTimer()

		_ = agent.DeregisterService("bench-service")
	}
}

// BenchmarkCacheConcurrentReads benchmarks concurrent cache reads
func BenchmarkCacheConcurrentReads(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Pre-populate cache with 1000 services
	for i := 0; i < 1000; i++ {
		entries := []*store.ServiceEntry{
			{
				Service: store.Service{
					Name:    "test-service",
					Address: "10.0.0.1",
					Port:    8080 + i,
				},
			},
		}
		cache.SetService("test-service", entries)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.GetService("test-service")
		}
	})
}

// BenchmarkCacheConcurrentWrites benchmarks concurrent cache writes
func BenchmarkCacheConcurrentWrites(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			entries := []*store.ServiceEntry{
				{
					Service: store.Service{
						Name:    "test-service",
						Address: "10.0.0.1",
						Port:    8080 + i,
					},
				},
			}
			cache.SetService("test-service", entries)
			i++
		}
	})
}

// BenchmarkCacheMixedOperations benchmarks mixed read/write operations
func BenchmarkCacheMixedOperations(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

	// Pre-populate cache
	entries := []*store.ServiceEntry{
		{
			Service: store.Service{
				Name:    "test-service",
				Address: "10.0.0.1",
				Port:    8080,
			},
		},
	}
	cache.SetService("test-service", entries)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// 80% reads, 20% writes (typical workload)
			if i%5 == 0 {
				cache.SetService("test-service", entries)
			} else {
				cache.GetService("test-service")
			}
			i++
		}
	})
}

// BenchmarkServiceUpdate benchmarks applying service updates to cache
func BenchmarkServiceUpdate(b *testing.B) {
	cfg := CacheConfig{
		ServiceTTL:     60 * time.Second,
		KVTTL:          300 * time.Second,
		HealthTTL:      30 * time.Second,
		MaxEntries:     10000,
		EvictionPolicy: "lru",
	}

	cache := NewCache(cfg)

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

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.ApplyServiceUpdate(update)
	}
}
