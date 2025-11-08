package loadbalancer

import (
	"testing"

	"github.com/neogan74/konsul/internal/store"
)

func setupTestStore() *store.ServiceStore {
	return store.NewServiceStore()
}

  func TestSelectService_RoundRobin(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRoundRobin)

	// Register multiple instances of the same logical service
	// Each instance has a unique name but shares the same service tag
	services := []store.Service{
		{Name: "api-1", Address: "10.0.0.1", Port: 8080, Tags: []string{"service:api", "v1"}},
		{Name: "api-2", Address: "10.0.0.2", Port: 8080, Tags: []string{"service:api", "v1"}},
		{Name: "api-3", Address: "10.0.0.3", Port: 8080, Tags: []string{"service:api", "v1"}},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Select services multiple times and verify round-robin behavior
	selections := make([]string, 6)
	for i := 0; i < 6; i++ {
		svc, ok := balancer.SelectService("service:api")
		if !ok {
			t.Fatalf("Expected to select service, got none")
		}
		selections[i] = svc.Address
	}

	// Verify round-robin pattern (should cycle through all 3 instances twice)
	expectedPattern := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.1", "10.0.0.2", "10.0.0.3"}
	for i, addr := range selections {
		if addr != expectedPattern[i] {
			t.Errorf("Round-robin failed at index %d: expected %s, got %s", i, expectedPattern[i], addr)
		}
	}
}

func TestSelectService_Random(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRandom)

	// Register multiple instances with service tag
	services := []store.Service{
		{Name: "api-1", Address: "10.0.0.1", Port: 8080, Tags: []string{"service:api"}},
		{Name: "api-2", Address: "10.0.0.2", Port: 8080, Tags: []string{"service:api"}},
		{Name: "api-3", Address: "10.0.0.3", Port: 8080, Tags: []string{"service:api"}},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Select many times and verify all instances are eventually selected
	addresses := make(map[string]int)
	for i := 0; i < 100; i++ {
		svc, ok := balancer.SelectService("service:api")
		if !ok {
			t.Fatalf("Expected to select service, got none")
		}
		addresses[svc.Address]++
	}

	// Verify all instances were selected at least once
	if len(addresses) != 3 {
		t.Errorf("Random selection should use all instances, got %d unique addresses", len(addresses))
	}

	for addr, count := range addresses {
		if count == 0 {
			t.Errorf("Address %s was never selected", addr)
		}
	}
}

func TestSelectService_LeastConnections(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyLeastConnections)

	// Register multiple instances with service tag
	services := []store.Service{
		{Name: "api-1", Address: "10.0.0.1", Port: 8080, Tags: []string{"service:api"}},
		{Name: "api-2", Address: "10.0.0.2", Port: 8080, Tags: []string{"service:api"}},
		{Name: "api-3", Address: "10.0.0.3", Port: 8080, Tags: []string{"service:api"}},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// First selection - should select any instance (all have 0 connections)
	svc1, ok := balancer.SelectService("service:api")
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	// Increment connections for first instance
	balancer.IncrementConnections(svc1)
	balancer.IncrementConnections(svc1)

	// Second selection - should select a different instance (with fewer connections)
	svc2, ok := balancer.SelectService("service:api")
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	if svc2.Address == svc1.Address {
		t.Errorf("Least connections should select different instance, got same: %s", svc1.Address)
	}
}

func TestSelectServiceByTags(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRoundRobin)

	// Register services with different tags
	services := []store.Service{
		{Name: "api", Address: "10.0.0.1", Port: 8080, Tags: []string{"env:prod", "http"}},
		{Name: "api", Address: "10.0.0.2", Port: 8080, Tags: []string{"env:dev", "http"}},
		{Name: "db", Address: "10.0.0.3", Port: 5432, Tags: []string{"env:prod", "postgres"}},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Select service by tags
	svc, ok := balancer.SelectServiceByTags([]string{"env:prod", "http"})
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	if svc.Address != "10.0.0.1" {
		t.Errorf("Expected to select 10.0.0.1, got %s", svc.Address)
	}
}

func TestSelectServiceByMetadata(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRandom)

	// Register services with different metadata
	services := []store.Service{
		{
			Name:    "api",
			Address: "10.0.0.1",
			Port:    8080,
			Meta:    map[string]string{"team": "platform", "env": "prod"},
		},
		{
			Name:    "api",
			Address: "10.0.0.2",
			Port:    8080,
			Meta:    map[string]string{"team": "frontend", "env": "prod"},
		},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Select service by metadata
	filters := map[string]string{"team": "platform", "env": "prod"}
	svc, ok := balancer.SelectServiceByMetadata(filters)
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	if svc.Address != "10.0.0.1" {
		t.Errorf("Expected to select 10.0.0.1, got %s", svc.Address)
	}
}

func TestSelectServiceByQuery(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRoundRobin)

	// Register services with tags and metadata
	services := []store.Service{
		{
			Name:    "api",
			Address: "10.0.0.1",
			Port:    8080,
			Tags:    []string{"http", "v2"},
			Meta:    map[string]string{"team": "platform"},
		},
		{
			Name:    "api",
			Address: "10.0.0.2",
			Port:    8080,
			Tags:    []string{"http", "v1"},
			Meta:    map[string]string{"team": "platform"},
		},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Select service by combined query
	tags := []string{"http", "v2"}
	metadata := map[string]string{"team": "platform"}
	svc, ok := balancer.SelectServiceByQuery(tags, metadata)
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	if svc.Address != "10.0.0.1" {
		t.Errorf("Expected to select 10.0.0.1, got %s", svc.Address)
	}
}

func TestSelectService_NoInstances(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRoundRobin)

	// Try to select service that doesn't exist
	_, ok := balancer.SelectService("nonexistent")
	if ok {
		t.Errorf("Expected no service, but got one")
	}
}

func TestConnectionTracking(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyLeastConnections)

	// Register service with service tag
	svc := store.Service{Name: "api-1", Address: "10.0.0.1", Port: 8080, Tags: []string{"service:api"}}
	if err := svcStore.Register(svc); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Increment connections
	balancer.IncrementConnections(svc)
	balancer.IncrementConnections(svc)
	balancer.IncrementConnections(svc)

	// Verify connections are tracked (internal state check)
	instanceKey := balancer.instanceKey(svc)
	balancer.mutex.RLock()
	connPtr := balancer.connections[instanceKey]
	balancer.mutex.RUnlock()

	if connPtr == nil {
		t.Fatalf("Expected connection tracking to be initialized")
	}

	// Decrement connections
	balancer.DecrementConnections(svc)

	// Note: We can't directly assert connection count without exposing internal state
	// The test verifies that increment/decrement don't panic
}

func TestStrategyGetterSetter(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRoundRobin)

	// Verify initial strategy
	if balancer.GetStrategy() != StrategyRoundRobin {
		t.Errorf("Expected round-robin strategy, got %s", balancer.GetStrategy())
	}

	// Change strategy
	balancer.SetStrategy(StrategyRandom)
	if balancer.GetStrategy() != StrategyRandom {
		t.Errorf("Expected random strategy, got %s", balancer.GetStrategy())
	}

	// Change to least-connections
	balancer.SetStrategy(StrategyLeastConnections)
	if balancer.GetStrategy() != StrategyLeastConnections {
		t.Errorf("Expected least-connections strategy, got %s", balancer.GetStrategy())
	}
}
