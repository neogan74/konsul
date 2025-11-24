package loadbalancer

import (
	"testing"

	"github.com/neogan74/konsul/internal/store"
)

func TestSelectService_WeightedRoundRobin(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyWeightedRoundRobin)

	// Register instances with different weights
	services := []store.Service{
		{
			Name:    "api-1",
			Address: "10.0.0.1",
			Port:    8080,
			Tags:    []string{"service:api"},
			Meta:    map[string]string{"weight": "1"},
		},
		{
			Name:    "api-2",
			Address: "10.0.0.2",
			Port:    8080,
			Tags:    []string{"service:api"},
			Meta:    map[string]string{"weight": "3"},
		},
		{
			Name:    "api-3",
			Address: "10.0.0.3",
			Port:    8080,
			Tags:    []string{"service:api"},
			Meta:    map[string]string{"weight": "2"},
		},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Select services multiple times and verify weighted distribution
	selections := make(map[string]int)
	for i := 0; i < 60; i++ {
		svc, ok := balancer.SelectService("service:api")
		if !ok {
			t.Fatalf("Expected to select service, got none")
		}
		selections[svc.Address]++
	}

	// api-2 (weight 3) should be selected ~50% of the time (30/60)
	// api-3 (weight 2) should be selected ~33% of the time (20/60)
	// api-1 (weight 1) should be selected ~17% of the time (10/60)

	// Allow some variance (±20%)
	if selections["10.0.0.2"] < 24 || selections["10.0.0.2"] > 36 {
		t.Errorf("Weight 3 instance: expected ~30 selections, got %d", selections["10.0.0.2"])
	}
	if selections["10.0.0.3"] < 16 || selections["10.0.0.3"] > 24 {
		t.Errorf("Weight 2 instance: expected ~20 selections, got %d", selections["10.0.0.3"])
	}
	if selections["10.0.0.1"] < 8 || selections["10.0.0.1"] > 12 {
		t.Errorf("Weight 1 instance: expected ~10 selections, got %d", selections["10.0.0.1"])
	}
}

func TestSelectService_WeightedRandom(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyWeightedRandom)

	// Register instances with different weights
	services := []store.Service{
		{
			Name:    "api-1",
			Address: "10.0.0.1",
			Port:    8080,
			Tags:    []string{"service:api"},
			Meta:    map[string]string{"weight": "1"},
		},
		{
			Name:    "api-2",
			Address: "10.0.0.2",
			Port:    8080,
			Tags:    []string{"service:api"},
			Meta:    map[string]string{"weight": "4"},
		},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Select many times and verify weighted distribution
	selections := make(map[string]int)
	for i := 0; i < 500; i++ {
		svc, ok := balancer.SelectService("service:api")
		if !ok {
			t.Fatalf("Expected to select service, got none")
		}
		selections[svc.Address]++
	}

	// api-2 (weight 4) should be selected ~80% of the time (400/500)
	// api-1 (weight 1) should be selected ~20% of the time (100/500)

	// Allow variance (±15%)
	if selections["10.0.0.2"] < 340 || selections["10.0.0.2"] > 460 {
		t.Errorf("Weight 4 instance: expected ~400 selections, got %d", selections["10.0.0.2"])
	}
	if selections["10.0.0.1"] < 40 || selections["10.0.0.1"] > 160 {
		t.Errorf("Weight 1 instance: expected ~100 selections, got %d", selections["10.0.0.1"])
	}
}

func TestSelectService_IPHash(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyIPHash)

	// Register multiple instances
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

	// Test sticky sessions - same client IP should always get same instance
	clientIPs := []string{"192.168.1.100", "192.168.1.101", "192.168.1.102"}
	selections := make(map[string]string) // clientIP -> selected address

	for _, clientIP := range clientIPs {
		opts := SelectOptions{ClientIP: clientIP}
		for i := 0; i < 10; i++ {
			svc, ok := balancer.SelectServiceWithOptions("service:api", opts)
			if !ok {
				t.Fatalf("Expected to select service, got none")
			}

			if selections[clientIP] == "" {
				selections[clientIP] = svc.Address
			} else if selections[clientIP] != svc.Address {
				t.Errorf("IP hash failed: client %s got different instances: %s vs %s",
					clientIP, selections[clientIP], svc.Address)
			}
		}
	}

	// Verify all clients got assigned
	if len(selections) != 3 {
		t.Errorf("Expected 3 client assignments, got %d", len(selections))
	}
}

func TestSelectService_RingHash(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyRingHash)

	// Register multiple instances
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

	// Test consistent hashing - same session key should always get same instance
	sessionKeys := []string{"user-123", "user-456", "user-789"}
	selections := make(map[string]string) // sessionKey -> selected address

	for _, sessionKey := range sessionKeys {
		opts := SelectOptions{SessionKey: sessionKey}
		for i := 0; i < 10; i++ {
			svc, ok := balancer.SelectServiceWithOptions("service:api", opts)
			if !ok {
				t.Fatalf("Expected to select service, got none")
			}

			if selections[sessionKey] == "" {
				selections[sessionKey] = svc.Address
			} else if selections[sessionKey] != svc.Address {
				t.Errorf("Ring hash failed: session %s got different instances: %s vs %s",
					sessionKey, selections[sessionKey], svc.Address)
			}
		}
	}

	// Verify all sessions got assigned
	if len(selections) != 3 {
		t.Errorf("Expected 3 session assignments, got %d", len(selections))
	}
}

func TestSelectService_LatencyBased(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyLatencyBased)

	// Register instances in different regions
	services := []store.Service{
		{Name: "api-east", Address: "10.0.0.1", Port: 8080, Tags: []string{"service:api", "region:us-east-1"}},
		{Name: "api-west", Address: "10.0.0.2", Port: 8080, Tags: []string{"service:api", "region:us-west-2"}},
		{Name: "api-eu", Address: "10.0.0.3", Port: 8080, Tags: []string{"service:api", "region:eu-west-1"}},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Test latency-based selection - should prefer same region
	opts := SelectOptions{ClientRegion: "us-east-1"}
	svc, ok := balancer.SelectServiceWithOptions("service:api", opts)
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	if svc.Address != "10.0.0.1" {
		t.Errorf("Expected to select us-east-1 instance (10.0.0.1), got %s", svc.Address)
	}

	// Test with different client region
	opts = SelectOptions{ClientRegion: "us-west-2"}
	svc, ok = balancer.SelectServiceWithOptions("service:api", opts)
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	if svc.Address != "10.0.0.2" {
		t.Errorf("Expected to select us-west-2 instance (10.0.0.2), got %s", svc.Address)
	}
}

func TestSelectService_WeightedRoundRobin_NoWeights(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyWeightedRoundRobin)

	// Register instances without weights (should fall back to regular round robin)
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

	// Should behave like regular round robin
	selections := make([]string, 6)
	for i := 0; i < 6; i++ {
		svc, ok := balancer.SelectService("service:api")
		if !ok {
			t.Fatalf("Expected to select service, got none")
		}
		selections[i] = svc.Address
	}

	expectedPattern := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.1", "10.0.0.2", "10.0.0.3"}
	for i, addr := range selections {
		if addr != expectedPattern[i] {
			t.Errorf("Round-robin fallback failed at index %d: expected %s, got %s", i, expectedPattern[i], addr)
		}
	}
}

func TestSelectService_IPHash_NoClientIP(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyIPHash)

	// Register instances
	services := []store.Service{
		{Name: "api-1", Address: "10.0.0.1", Port: 8080, Tags: []string{"service:api"}},
		{Name: "api-2", Address: "10.0.0.2", Port: 8080, Tags: []string{"service:api"}},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Without client IP, should fall back to random
	opts := SelectOptions{} // No ClientIP
	svc, ok := balancer.SelectServiceWithOptions("service:api", opts)
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	// Just verify we got a service (can't predict random)
	if svc.Address != "10.0.0.1" && svc.Address != "10.0.0.2" {
		t.Errorf("Unexpected service address: %s", svc.Address)
	}
}

func TestSelectService_LatencyBased_NoRegion(t *testing.T) {
	svcStore := setupTestStore()
	balancer := New(svcStore, StrategyLatencyBased)

	// Register instances with regions
	services := []store.Service{
		{Name: "api-east", Address: "10.0.0.1", Port: 8080, Tags: []string{"service:api", "region:us-east-1"}},
		{Name: "api-west", Address: "10.0.0.2", Port: 8080, Tags: []string{"service:api", "region:us-west-2"}},
	}

	for _, svc := range services {
		if err := svcStore.Register(svc); err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}
	}

	// Without client region, should fall back to random
	opts := SelectOptions{} // No ClientRegion
	svc, ok := balancer.SelectServiceWithOptions("service:api", opts)
	if !ok {
		t.Fatalf("Expected to select service, got none")
	}

	// Just verify we got a service (can't predict random)
	if svc.Address != "10.0.0.1" && svc.Address != "10.0.0.2" {
		t.Errorf("Unexpected service address: %s", svc.Address)
	}
}

func TestExtractWeight(t *testing.T) {
	tests := []struct {
		name     string
		service  store.Service
		expected int
	}{
		{
			name:     "with valid weight",
			service:  store.Service{Meta: map[string]string{"weight": "5"}},
			expected: 5,
		},
		{
			name:     "without weight",
			service:  store.Service{Meta: map[string]string{}},
			expected: 1,
		},
		{
			name:     "with invalid weight",
			service:  store.Service{Meta: map[string]string{"weight": "invalid"}},
			expected: 1,
		},
		{
			name:     "with zero weight",
			service:  store.Service{Meta: map[string]string{"weight": "0"}},
			expected: 1,
		},
		{
			name:     "with negative weight",
			service:  store.Service{Meta: map[string]string{"weight": "-5"}},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWeight(tt.service)
			if result != tt.expected {
				t.Errorf("extractWeight() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestExtractRegionFromTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "with region tag",
			tags:     []string{"env:prod", "region:us-east-1", "http"},
			expected: "us-east-1",
		},
		{
			name:     "without region tag",
			tags:     []string{"env:prod", "http"},
			expected: "",
		},
		{
			name:     "empty tags",
			tags:     []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRegionFromTags(tt.tags)
			if result != tt.expected {
				t.Errorf("extractRegionFromTags() = %q, want %q", result, tt.expected)
			}
		})
	}
}
