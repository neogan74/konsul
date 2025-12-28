package agent

import (
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/store"
)

func TestHealthStatus_Constants(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusPassing, "passing"},
		{HealthStatusWarning, "warning"},
		{HealthStatusCritical, "critical"},
		{HealthStatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.status))
			}
		})
	}
}

func TestUpdateType_Constants(t *testing.T) {
	tests := []struct {
		updateType UpdateType
		expected   string
	}{
		{UpdateTypeAdd, "add"},
		{UpdateTypeUpdate, "update"},
		{UpdateTypeDelete, "delete"},
	}

	for _, tt := range tests {
		t.Run(string(tt.updateType), func(t *testing.T) {
			if string(tt.updateType) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(tt.updateType))
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	svc := &store.Service{
		Name:    "test-service",
		Address: "10.0.0.1",
		Port:    8080,
	}

	entry := &store.ServiceEntry{
		Service:     *svc,
		ModifyIndex: 1,
		CreateIndex: 1,
	}

	update := ServiceUpdate{
		Type:        UpdateTypeAdd,
		ServiceName: "test-service",
		Service:     svc,
		Entry:       entry,
	}

	if update.Type != UpdateTypeAdd {
		t.Errorf("Expected type 'add', got '%s'", update.Type)
	}

	if update.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", update.ServiceName)
	}

	if update.Service == nil {
		t.Error("Expected service to be set")
	}

	if update.Entry == nil {
		t.Error("Expected entry to be set")
	}
}

func TestKVUpdate(t *testing.T) {
	entry := &store.KVEntry{
		Value:       "test-value",
		ModifyIndex: 1,
		CreateIndex: 1,
	}

	update := KVUpdate{
		Type:  UpdateTypeUpdate,
		Key:   "test-key",
		Entry: entry,
	}

	if update.Type != UpdateTypeUpdate {
		t.Errorf("Expected type 'update', got '%s'", update.Type)
	}

	if update.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got '%s'", update.Key)
	}

	if update.Entry == nil {
		t.Error("Expected entry to be set")
	}
}

func TestHealthUpdate(t *testing.T) {
	update := HealthUpdate{
		ServiceID: "svc-1",
		CheckID:   "check-1",
		Status:    HealthStatusPassing,
		Output:    "All systems operational",
	}

	if update.ServiceID != "svc-1" {
		t.Errorf("Expected service ID 'svc-1', got '%s'", update.ServiceID)
	}

	if update.CheckID != "check-1" {
		t.Errorf("Expected check ID 'check-1', got '%s'", update.CheckID)
	}

	if update.Status != HealthStatusPassing {
		t.Errorf("Expected status 'passing', got '%s'", update.Status)
	}

	if update.Output != "All systems operational" {
		t.Errorf("Expected output 'All systems operational', got '%s'", update.Output)
	}
}

func TestSyncRequest(t *testing.T) {
	req := SyncRequest{
		AgentID:         "agent-1",
		LastSyncIndex:   100,
		WatchedPrefixes: []string{"config/", "secrets/"},
		FullSync:        false,
	}

	if req.AgentID != "agent-1" {
		t.Errorf("Expected agent ID 'agent-1', got '%s'", req.AgentID)
	}

	if req.LastSyncIndex != 100 {
		t.Errorf("Expected last sync index 100, got %d", req.LastSyncIndex)
	}

	if len(req.WatchedPrefixes) != 2 {
		t.Errorf("Expected 2 watched prefixes, got %d", len(req.WatchedPrefixes))
	}

	if req.FullSync {
		t.Error("Expected full sync to be false")
	}
}

func TestSyncResponse(t *testing.T) {
	resp := SyncResponse{
		CurrentIndex:   150,
		ServiceUpdates: []ServiceUpdate{},
		KVUpdates:      []KVUpdate{},
		HealthUpdates:  []HealthUpdate{},
	}

	if resp.CurrentIndex != 150 {
		t.Errorf("Expected current index 150, got %d", resp.CurrentIndex)
	}

	if resp.ServiceUpdates == nil {
		t.Error("Expected service updates to be initialized")
	}

	if resp.KVUpdates == nil {
		t.Error("Expected KV updates to be initialized")
	}

	if resp.HealthUpdates == nil {
		t.Error("Expected health updates to be initialized")
	}
}

func TestBatchRegisterRequest(t *testing.T) {
	services := []store.Service{
		{Name: "svc1", Address: "10.0.0.1", Port: 80},
		{Name: "svc2", Address: "10.0.0.2", Port: 8080},
	}

	req := BatchRegisterRequest{
		AgentID:        "agent-1",
		Services:       services,
		SequenceNumber: 42,
	}

	if req.AgentID != "agent-1" {
		t.Errorf("Expected agent ID 'agent-1', got '%s'", req.AgentID)
	}

	if len(req.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(req.Services))
	}

	if req.SequenceNumber != 42 {
		t.Errorf("Expected sequence number 42, got %d", req.SequenceNumber)
	}
}

func TestAgentInfo(t *testing.T) {
	now := time.Now()

	info := AgentInfo{
		ID:         "agent-test-123",
		NodeName:   "node-1",
		NodeIP:     "10.0.0.1",
		Datacenter: "us-east-1",
		Metadata: map[string]string{
			"region": "us-east",
			"zone":   "1a",
		},
		StartedAt: now,
		Version:   "0.1.0",
	}

	if info.ID != "agent-test-123" {
		t.Errorf("Expected ID 'agent-test-123', got '%s'", info.ID)
	}

	if info.NodeName != "node-1" {
		t.Errorf("Expected node name 'node-1', got '%s'", info.NodeName)
	}

	if info.NodeIP != "10.0.0.1" {
		t.Errorf("Expected node IP '10.0.0.1', got '%s'", info.NodeIP)
	}

	if info.Datacenter != "us-east-1" {
		t.Errorf("Expected datacenter 'us-east-1', got '%s'", info.Datacenter)
	}

	if len(info.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(info.Metadata))
	}

	if info.Metadata["region"] != "us-east" {
		t.Errorf("Expected region 'us-east', got '%s'", info.Metadata["region"])
	}

	if !info.StartedAt.Equal(now) {
		t.Error("Expected started_at to match")
	}

	if info.Version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got '%s'", info.Version)
	}
}

func TestAgentStats(t *testing.T) {
	stats := AgentStats{
		CacheHitRate:    0.95,
		CacheEntries:    1000,
		LocalServices:   5,
		LastSyncTime:    time.Now(),
		SyncErrorsTotal: 2,
		Uptime:          "1h30m",
	}

	if stats.CacheHitRate != 0.95 {
		t.Errorf("Expected cache hit rate 0.95, got %f", stats.CacheHitRate)
	}

	if stats.CacheEntries != 1000 {
		t.Errorf("Expected 1000 cache entries, got %d", stats.CacheEntries)
	}

	if stats.LocalServices != 5 {
		t.Errorf("Expected 5 local services, got %d", stats.LocalServices)
	}

	if stats.SyncErrorsTotal != 2 {
		t.Errorf("Expected 2 sync errors, got %d", stats.SyncErrorsTotal)
	}

	if stats.Uptime != "1h30m" {
		t.Errorf("Expected uptime '1h30m', got '%s'", stats.Uptime)
	}
}
