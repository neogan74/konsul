package agent

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

func TestNewAgent(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent-1"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Expected agent to be created")
	}

	if agent.config.ID != "test-agent-1" {
		t.Errorf("Expected agent ID 'test-agent-1', got '%s'", agent.config.ID)
	}

	if agent.info.NodeName != "test-node" {
		t.Errorf("Expected node name 'test-node', got '%s'", agent.info.NodeName)
	}

	if agent.cache == nil {
		t.Error("Expected cache to be initialized")
	}

	if agent.syncEngine == nil {
		t.Error("Expected sync engine to be initialized")
	}

	if agent.serverClient == nil {
		t.Error("Expected server client to be initialized")
	}

	if agent.api == nil {
		t.Error("Expected API to be initialized")
	}
}

func TestNewAgent_InvalidConfig(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.NodeName = "" // Invalid

	_, err := NewAgent(cfg, log)
	if err == nil {
		t.Error("Expected error for invalid config")
	}
}

func TestAgent_GenerateAgentID(t *testing.T) {
	tests := []struct {
		name       string
		nodeName   string
		wantPrefix string
	}{
		{
			name:       "with node name",
			nodeName:   "test-node",
			wantPrefix: "agent-test-node-",
		},
		{
			name:       "empty node name",
			nodeName:   "",
			wantPrefix: "agent-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := generateAgentID(tt.nodeName)

			if !strings.HasPrefix(id, tt.wantPrefix) {
				t.Errorf("Expected ID to start with '%s', got '%s'", tt.wantPrefix, id)
			}

			// Verify ID format (agent-<node>-<hex>)
			parts := strings.Split(id, "-")
			if len(parts) < 3 {
				t.Errorf("Expected at least 3 parts in ID, got %d", len(parts))
			}
		})
	}
}

func TestAgent_GenerateAgentID_Hostname(t *testing.T) {
	// Test with empty node name should use hostname
	id := generateAgentID("")

	hostname, _ := os.Hostname()
	if hostname != "" {
		expectedPrefix := "agent-" + hostname + "-"
		if !strings.HasPrefix(id, expectedPrefix) {
			t.Errorf("Expected ID to start with '%s', got '%s'", expectedPrefix, id)
		}
	}
}

func TestAgent_RegisterService(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	svc := store.Service{
		Name:    "web-service",
		Address: "10.0.0.1",
		Port:    8080,
		Tags:    []string{"http", "api"},
	}

	err = agent.RegisterService(svc)
	if err != nil {
		t.Errorf("Failed to register service: %v", err)
	}

	// Verify service was stored locally
	services := agent.ListLocalServices()
	if len(services) != 1 {
		t.Errorf("Expected 1 local service, got %d", len(services))
	}

	if services[0].Service.Name != "web-service" {
		t.Errorf("Expected service name 'web-service', got '%s'", services[0].Service.Name)
	}
}

func TestAgent_DeregisterService(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Register a service
	svc := store.Service{
		Name:    "web-service",
		Address: "10.0.0.1",
		Port:    8080,
	}

	_ = agent.RegisterService(svc)

	// Deregister it
	err = agent.DeregisterService("web-service")
	if err != nil {
		t.Errorf("Failed to deregister service: %v", err)
	}

	// Verify service was removed
	services := agent.ListLocalServices()
	if len(services) != 0 {
		t.Errorf("Expected 0 local services after deregister, got %d", len(services))
	}
}

func TestAgent_DeregisterService_NotFound(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	err = agent.DeregisterService("nonexistent")
	if err == nil {
		t.Error("Expected error when deregistering nonexistent service")
	}
}

func TestAgent_Stats(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Register a service
	svc := store.Service{
		Name:    "web-service",
		Address: "10.0.0.1",
		Port:    8080,
	}
	_ = agent.RegisterService(svc)

	stats := agent.Stats()

	if stats.LocalServices != 1 {
		t.Errorf("Expected 1 local service, got %d", stats.LocalServices)
	}

	if stats.CacheHitRate != 0 {
		t.Errorf("Expected cache hit rate 0, got %f", stats.CacheHitRate)
	}

	if stats.CacheEntries != 0 {
		t.Errorf("Expected 0 cache entries, got %d", stats.CacheEntries)
	}

	if stats.Uptime == "" {
		t.Error("Expected uptime to be populated")
	}
}

func TestAgent_Health(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Initially should be healthy (no syncs yet)
	if !agent.Health() {
		t.Error("Expected agent to be healthy initially")
	}

	// Cancel context
	agent.cancel()

	// Should be unhealthy after cancel
	if agent.Health() {
		t.Error("Expected agent to be unhealthy after cancel")
	}
}

func TestAgent_Info(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"
	cfg.NodeIP = "10.0.0.1"
	cfg.Datacenter = "us-east-1"
	cfg.Metadata = map[string]string{
		"env": "test",
	}

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	info := agent.Info()

	if info.ID != "test-agent" {
		t.Errorf("Expected ID 'test-agent', got '%s'", info.ID)
	}

	if info.NodeName != "test-node" {
		t.Errorf("Expected node name 'test-node', got '%s'", info.NodeName)
	}

	if info.NodeIP != "10.0.0.1" {
		t.Errorf("Expected node IP '10.0.0.1', got '%s'", info.NodeIP)
	}

	if info.Datacenter != "us-east-1" {
		t.Errorf("Expected datacenter 'us-east-1', got '%s'", info.Datacenter)
	}

	if info.Metadata["env"] != "test" {
		t.Errorf("Expected metadata env 'test', got '%s'", info.Metadata["env"])
	}

	if info.StartedAt.IsZero() {
		t.Error("Expected started_at to be set")
	}
}

func TestAgent_LocalServices(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Register multiple services
	services := []store.Service{
		{Name: "web", Address: "10.0.0.1", Port: 80},
		{Name: "api", Address: "10.0.0.2", Port: 8080},
		{Name: "db", Address: "10.0.0.3", Port: 5432},
	}

	for _, svc := range services {
		_ = agent.RegisterService(svc)
	}

	local := agent.ListLocalServices()

	if len(local) != 3 {
		t.Errorf("Expected 3 local services, got %d", len(local))
	}

	// Verify all services are present
	found := make(map[string]bool)
	for _, entry := range local {
		found[entry.Service.Name] = true
	}

	for _, svc := range services {
		if !found[svc.Name] {
			t.Errorf("Service '%s' not found in local services", svc.Name)
		}
	}
}

func TestAgent_CacheIntegration(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Add entries to cache directly
	entries := []*store.ServiceEntry{
		{
			Service: store.Service{
				Name:    "cached-service",
				Address: "10.0.0.1",
				Port:    8080,
			},
		},
	}

	agent.cache.SetService("cached-service", entries)

	// Verify cache hit count
	if agent.cache.ServiceCount() != 1 {
		t.Errorf("Expected 1 cached service, got %d", agent.cache.ServiceCount())
	}
}

func TestAgent_Uptime(t *testing.T) {
	log := logger.GetDefault()
	cfg := DefaultConfig()
	cfg.ID = "test-agent"
	cfg.NodeName = "test-node"

	agent, err := NewAgent(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Sleep briefly
	time.Sleep(10 * time.Millisecond)

	stats := agent.Stats()

	// Uptime should be non-empty and parseable
	if stats.Uptime == "" {
		t.Error("Expected uptime to be populated")
	}

	// Verify it's a valid duration string
	_, err = time.ParseDuration(stats.Uptime)
	if err != nil {
		t.Errorf("Invalid uptime format: %v", err)
	}
}
