package handlers

import (
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status      string            `json:"status"`
	Version     string            `json:"version"`
	Uptime      string            `json:"uptime"`
	Timestamp   time.Time         `json:"timestamp"`
	Services    ServiceHealth     `json:"services"`
	KVStore     KVHealth          `json:"kv_store"`
	System      SystemHealth      `json:"system"`
}

type ServiceHealth struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Expired int `json:"expired"`
}

type KVHealth struct {
	Total int `json:"total_keys"`
}

type SystemHealth struct {
	Goroutines   int    `json:"goroutines"`
	MemoryAlloc  uint64 `json:"memory_alloc_bytes"`
	MemorySys    uint64 `json:"memory_sys_bytes"`
	NumGC        uint32 `json:"num_gc"`
}

// HealthHandler handles health check operations
type HealthHandler struct {
	kvStore      *store.KVStore
	serviceStore *store.ServiceStore
	startTime    time.Time
	version      string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(kvStore *store.KVStore, serviceStore *store.ServiceStore, version string) *HealthHandler {
	return &HealthHandler{
		kvStore:      kvStore,
		serviceStore: serviceStore,
		startTime:    time.Now(),
		version:      version,
	}
}

// Check returns the health status of the service
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get service stats
	serviceEntries := h.serviceStore.ListAll()
	activeCount := 0
	expiredCount := 0
	now := time.Now()

	for _, entry := range serviceEntries {
		if entry.ExpiresAt.After(now) {
			activeCount++
		} else {
			expiredCount++
		}
	}

	// Get KV store stats
	kvKeys := h.kvStore.List()

	status := HealthStatus{
		Status:    "healthy",
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now(),
		Services: ServiceHealth{
			Total:   len(serviceEntries),
			Active:  activeCount,
			Expired: expiredCount,
		},
		KVStore: KVHealth{
			Total: len(kvKeys),
		},
		System: SystemHealth{
			Goroutines:  runtime.NumGoroutine(),
			MemoryAlloc: m.Alloc,
			MemorySys:   m.Sys,
			NumGC:       m.NumGC,
		},
	}

	return c.JSON(status)
}

// Liveness is a simple liveness probe
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "alive",
		"timestamp": time.Now(),
	})
}

// Readiness checks if the service is ready to accept traffic
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	// Here you could add checks for external dependencies
	// For now, we'll just check if the stores are accessible

	// Try to access KV store
	_ = h.kvStore.List()

	// Try to access service store
	_ = h.serviceStore.List()

	return c.JSON(fiber.Map{
		"status": "ready",
		"timestamp": time.Now(),
	})
}