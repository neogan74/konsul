package raft

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neogan74/konsul/internal/store"
)

// MockKVStore is a thread-safe in-memory KV store with ModifyIndex tracking.
type MockKVStore struct {
	mu        sync.Mutex
	data      map[string]store.KVEntrySnapshot
	nextIndex uint64
}

func NewMockKVStore() *MockKVStore {
	return &MockKVStore{data: make(map[string]store.KVEntrySnapshot)}
}

// setLocked sets key=value, increments nextIndex. Caller must hold mu.
func (m *MockKVStore) setLocked(key, value string) uint64 {
	m.nextIndex++
	existing := m.data[key]
	createIdx := existing.CreateIndex
	if createIdx == 0 {
		createIdx = m.nextIndex
	}
	m.data[key] = store.KVEntrySnapshot{Value: value, ModifyIndex: m.nextIndex, CreateIndex: createIdx}
	return m.nextIndex
}

// Extra methods used directly by integration tests (not part of KVStoreInterface)

func (m *MockKVStore) Set(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setLocked(key, value)
	return nil
}

func (m *MockKVStore) SetWithFlags(key, value string, flags uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextIndex++
	existing := m.data[key]
	createIdx := existing.CreateIndex
	if createIdx == 0 {
		createIdx = m.nextIndex
	}
	m.data[key] = store.KVEntrySnapshot{Value: value, ModifyIndex: m.nextIndex, CreateIndex: createIdx, Flags: flags}
	return nil
}

func (m *MockKVStore) Get(key string) (value string, ok bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[key]
	return entry.Value, ok, nil
}

func (m *MockKVStore) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MockKVStore) List(_ string) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]string, len(m.data))
	for k, v := range m.data {
		result[k] = v.Value
	}
	return result, nil
}

// KVStoreInterface methods

func (m *MockKVStore) SetLocal(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setLocked(key, value)
}

func (m *MockKVStore) SetWithFlagsLocal(key, value string, flags uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextIndex++
	existing := m.data[key]
	createIdx := existing.CreateIndex
	if createIdx == 0 {
		createIdx = m.nextIndex
	}
	m.data[key] = store.KVEntrySnapshot{Value: value, ModifyIndex: m.nextIndex, CreateIndex: createIdx, Flags: flags}
}

func (m *MockKVStore) DeleteLocal(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *MockKVStore) BatchSetLocal(items map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range items {
		m.setLocked(k, v)
	}
	return nil
}

func (m *MockKVStore) BatchDeleteLocal(keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *MockKVStore) SetCASLocal(key, value string, expectedIndex uint64) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[key]
	if expectedIndex == 0 {
		if ok {
			return 0, fmt.Errorf("CAS conflict: key %q already exists", key)
		}
	} else {
		if !ok || entry.ModifyIndex != expectedIndex {
			return 0, fmt.Errorf("CAS conflict: key %q expected index %d, got %d", key, expectedIndex, entry.ModifyIndex)
		}
	}
	return m.setLocked(key, value), nil
}

func (m *MockKVStore) DeleteCASLocal(key string, expectedIndex uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[key]
	if !ok || entry.ModifyIndex != expectedIndex {
		return fmt.Errorf("CAS conflict: delete key %q expected index %d", key, expectedIndex)
	}
	delete(m.data, key)
	return nil
}

func (m *MockKVStore) BatchSetCASLocal(items map[string]string, expectedIndices map[string]uint64) (map[string]uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Validate all keys first (atomicity guarantee)
	for k := range items {
		expected := expectedIndices[k]
		entry, ok := m.data[k]
		if expected == 0 {
			if ok {
				return nil, fmt.Errorf("CAS conflict: key %q already exists", k)
			}
		} else {
			if !ok || entry.ModifyIndex != expected {
				return nil, fmt.Errorf("CAS conflict: key %q index mismatch", k)
			}
		}
	}
	// Apply all (no validation errors means proceed)
	results := make(map[string]uint64, len(items))
	for k, v := range items {
		results[k] = m.setLocked(k, v)
	}
	return results, nil
}

func (m *MockKVStore) BatchDeleteCASLocal(keys []string, expectedIndices map[string]uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Validate all keys first (atomicity guarantee)
	for _, k := range keys {
		entry, ok := m.data[k]
		if !ok || entry.ModifyIndex != expectedIndices[k] {
			return fmt.Errorf("CAS conflict: batch delete key %q index mismatch", k)
		}
	}
	// Apply all
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *MockKVStore) GetAllData() map[string]store.KVEntrySnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]store.KVEntrySnapshot, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

func (m *MockKVStore) RestoreFromSnapshot(data map[string]store.KVEntrySnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]store.KVEntrySnapshot, len(data))
	for k, v := range data {
		m.data[k] = v
		if v.ModifyIndex > m.nextIndex {
			m.nextIndex = v.ModifyIndex
		}
	}
	return nil
}

func (m *MockKVStore) GetEntrySnapshot(key string) (store.KVEntrySnapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[key]
	return entry, ok
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

func (m *MockServiceStore) Register(name, _ string, _ int, _ []string, _ map[string]string) error {
	m.services[name] = struct{}{}
	return nil
}

func (m *MockServiceStore) Deregister(name string) error {
	delete(m.services, name)
	return nil
}

func (m *MockServiceStore) Get(name string) (svc interface{}, ok bool, err error) {
	svc, ok := m.services[name]
	return svc, ok, nil
}

func (m *MockServiceStore) List() (map[string]interface{}, error) {
	return m.services, nil
}

func (m *MockServiceStore) Heartbeat(_ string) error {
	return nil
}

func (m *MockServiceStore) RegisterLocal(service store.ServiceDataSnapshot) error {
	m.services[service.Name] = service
	return nil
}

func (m *MockServiceStore) DeregisterLocal(name string) {
	delete(m.services, name)
}

func (m *MockServiceStore) HeartbeatLocal(_ string) bool {
	return true
}

func (m *MockServiceStore) RegisterCASLocal(_ store.ServiceDataSnapshot, _ uint64) (uint64, error) {
	return 0, nil
}

func (m *MockServiceStore) DeregisterCASLocal(_ string, _ uint64) error {
	return nil
}

func (m *MockServiceStore) UpdateTTLCheck(_ string) error {
	return nil
}

func (m *MockServiceStore) GetEntrySnapshot(name string) (store.ServiceEntrySnapshot, bool) {
	val, ok := m.services[name]
	if !ok {
		return store.ServiceEntrySnapshot{}, false
	}

	if svc, ok := val.(store.ServiceDataSnapshot); ok {
		return store.ServiceEntrySnapshot{
			Service:     svc,
			ModifyIndex: 1,
			CreateIndex: 1,
		}, true
	}

	return store.ServiceEntrySnapshot{
		Service:     store.ServiceDataSnapshot{Name: name},
		ModifyIndex: 1,
		CreateIndex: 1,
	}, true
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
	defer func() { _ = node.Shutdown() }()

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
	defer func() { _ = node.Shutdown() }()

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
	defer func() { _ = node.Shutdown() }()

	// Wait for leader with short timeout - should timeout
	err = node.WaitForLeader(500 * time.Millisecond)
	assert.Error(t, err)
	assert.Equal(t, ErrNoLeader, err)
}
