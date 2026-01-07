package raft

import (
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockKVStore is a simple in-memory implementation for testing
type MockKVStore struct {
	data map[string]string
}

func NewMockKVStore() *MockKVStore {
	return &MockKVStore{
		data: make(map[string]string),
	}
}

func (m *MockKVStore) Set(key, value string) error {
	m.data[key] = value
	return nil
}

func (m *MockKVStore) SetWithFlags(key, value string, flags uint64) error {
	m.data[key] = value
	return nil
}

func (m *MockKVStore) Get(key string) (string, bool, error) {
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *MockKVStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockKVStore) List(prefix string) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range m.data {
		result[k] = v
	}
	return result, nil
}

func (m *MockKVStore) SetLocal(key, value string) {
	m.data[key] = value
}

func (m *MockKVStore) SetWithFlagsLocal(key, value string, flags uint64) {
	m.data[key] = value
}

func (m *MockKVStore) DeleteLocal(key string) {
	delete(m.data, key)
}

func (m *MockKVStore) BatchSetLocal(items map[string]string) error {
	for k, v := range items {
		m.data[k] = v
	}
	return nil
}

func (m *MockKVStore) BatchDeleteLocal(keys []string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *MockKVStore) GetAllData() map[string]store.KVEntrySnapshot {
	result := make(map[string]store.KVEntrySnapshot, len(m.data))
	for k, v := range m.data {
		result[k] = store.KVEntrySnapshot{Value: v}
	}
	return result
}

func (m *MockKVStore) RestoreFromSnapshot(data map[string]store.KVEntrySnapshot) error {
	m.data = make(map[string]string, len(data))
	for k, v := range data {
		m.data[k] = v.Value
	}
	return nil
}

// MockServiceStore is a simple in-memory implementation for testing
type MockServiceStore struct {
	services map[string]interface{}
}

func NewMockServiceStore() *MockServiceStore {
	return &MockServiceStore{
		services: make(map[string]interface{}),
	}
}

func (m *MockServiceStore) Register(name, address string, port int, tags []string, meta map[string]string) error {
	m.services[name] = struct{}{}
	return nil
}

func (m *MockServiceStore) Deregister(name string) error {
	delete(m.services, name)
	return nil
}

func (m *MockServiceStore) Get(name string) (interface{}, bool, error) {
	svc, ok := m.services[name]
	return svc, ok, nil
}

func (m *MockServiceStore) List() (map[string]interface{}, error) {
	return m.services, nil
}

func (m *MockServiceStore) Heartbeat(name string) error {
	return nil
}

func (m *MockServiceStore) RegisterLocal(service store.ServiceDataSnapshot) error {
	m.services[service.Name] = service
	return nil
}

func (m *MockServiceStore) DeregisterLocal(name string) {
	delete(m.services, name)
}

func (m *MockServiceStore) HeartbeatLocal(name string) bool {
	return true
}

func (m *MockServiceStore) GetAllData() map[string]store.ServiceEntrySnapshot {
	result := make(map[string]store.ServiceEntrySnapshot, len(m.services))
	for name := range m.services {
		result[name] = store.ServiceEntrySnapshot{
			Service: store.ServiceDataSnapshot{Name: name},
		}
	}
	return result
}

func (m *MockServiceStore) RestoreFromSnapshot(data map[string]store.ServiceEntrySnapshot) error {
	m.services = make(map[string]interface{}, len(data))
	for name, entry := range data {
		m.services[name] = entry.Service
	}
	return nil
}

func newTestConfig(t *testing.T, nodeID string, bootstrap bool) *Config {
	t.Helper()

	resetPrometheusRegistry()
	cfg := DefaultConfig()
	cfg.NodeID = nodeID
	cfg.BindAddr = "127.0.0.1:0"
	cfg.DataDir = t.TempDir()
	cfg.Bootstrap = bootstrap
	return cfg
}

func resetPrometheusRegistry() {
	registry := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry
	prometheus.DefaultGatherer = registry
}

// TestNodeCreation tests that a Node can be created successfully
func TestNodeCreation(t *testing.T) {
	cfg := newTestConfig(t, "test-node", true)

	kvStore := NewMockKVStore()
	serviceStore := NewMockServiceStore()

	node, err := NewNode(cfg, kvStore, serviceStore)
	require.NoError(t, err)
	require.NotNil(t, node)
	defer node.Shutdown()

	// Check that metrics are initialized
	assert.NotNil(t, node.metrics)

	// Wait a bit for bootstrap
	time.Sleep(500 * time.Millisecond)

	// Check that node becomes leader (since it's bootstrapped)
	err = node.WaitForLeader(5 * time.Second)
	require.NoError(t, err)

	assert.True(t, node.IsLeader())
}

// TestMetricsMonitoring tests that state monitoring goroutine updates metrics
func TestMetricsMonitoring(t *testing.T) {
	cfg := newTestConfig(t, "test-metrics-node", true)

	kvStore := NewMockKVStore()
	serviceStore := NewMockServiceStore()

	node, err := NewNode(cfg, kvStore, serviceStore)
	require.NoError(t, err)
	require.NotNil(t, node)
	defer node.Shutdown()

	// Wait for leader election
	err = node.WaitForLeader(5 * time.Second)
	require.NoError(t, err)

	// Give metrics monitoring time to update
	time.Sleep(2 * time.Second)

	// Verify that state metric is set to Leader
	assert.Equal(t, raft.Leader, node.State())

	// The metrics should be updated by monitorState goroutine
	// We can't directly check Prometheus metrics without exposing them,
	// but we can verify the node is in the correct state
	assert.True(t, node.IsLeader())
}

// TestNodeShutdown tests graceful shutdown
func TestNodeShutdown(t *testing.T) {
	cfg := newTestConfig(t, "test-shutdown-node", true)

	kvStore := NewMockKVStore()
	serviceStore := NewMockServiceStore()

	node, err := NewNode(cfg, kvStore, serviceStore)
	require.NoError(t, err)
	require.NotNil(t, node)

	// Shutdown the node
	err = node.Shutdown()
	assert.NoError(t, err)

	// Verify state is Shutdown
	assert.Equal(t, raft.Shutdown, node.State())
}

// TestWaitForLeaderTimeout tests that WaitForLeader times out correctly
func TestWaitForLeaderTimeout(t *testing.T) {
	cfg := newTestConfig(t, "test-timeout-node", false)

	kvStore := NewMockKVStore()
	serviceStore := NewMockServiceStore()

	node, err := NewNode(cfg, kvStore, serviceStore)
	require.NoError(t, err)
	require.NotNil(t, node)
	defer node.Shutdown()

	// Wait for leader with short timeout - should timeout
	err = node.WaitForLeader(500 * time.Millisecond)
	assert.Error(t, err)
	assert.Equal(t, ErrNoLeader, err)
}
