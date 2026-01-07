package raft

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKVStore implements KVStoreInterface for testing.
type mockKVStore struct {
	data map[string]store.KVEntrySnapshot
}

func newMockKVStore() *mockKVStore {
	return &mockKVStore{
		data: make(map[string]store.KVEntrySnapshot),
	}
}

func (m *mockKVStore) SetLocal(key, value string) {
	m.data[key] = store.KVEntrySnapshot{
		Value:       value,
		ModifyIndex: uint64(len(m.data) + 1),
		CreateIndex: uint64(len(m.data) + 1),
	}
}

func (m *mockKVStore) SetWithFlagsLocal(key, value string, flags uint64) {
	m.data[key] = store.KVEntrySnapshot{
		Value:       value,
		ModifyIndex: uint64(len(m.data) + 1),
		CreateIndex: uint64(len(m.data) + 1),
		Flags:       flags,
	}
}

func (m *mockKVStore) DeleteLocal(key string) {
	delete(m.data, key)
}

func (m *mockKVStore) BatchSetLocal(items map[string]string) error {
	for k, v := range items {
		m.SetLocal(k, v)
	}
	return nil
}

func (m *mockKVStore) BatchDeleteLocal(keys []string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func (m *mockKVStore) GetAllData() map[string]store.KVEntrySnapshot {
	result := make(map[string]store.KVEntrySnapshot, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

func (m *mockKVStore) RestoreFromSnapshot(data map[string]store.KVEntrySnapshot) error {
	m.data = make(map[string]store.KVEntrySnapshot, len(data))
	for k, v := range data {
		m.data[k] = v
	}
	return nil
}

// mockServiceStore implements ServiceStoreInterface for testing.
type mockServiceStore struct {
	data map[string]store.ServiceEntrySnapshot
}

func newMockServiceStore() *mockServiceStore {
	return &mockServiceStore{
		data: make(map[string]store.ServiceEntrySnapshot),
	}
}

func (m *mockServiceStore) RegisterLocal(service store.ServiceDataSnapshot) error {
	m.data[service.Name] = store.ServiceEntrySnapshot{
		Service:     service,
		ExpiresAt:   time.Now().Add(30 * time.Second),
		ModifyIndex: uint64(len(m.data) + 1),
		CreateIndex: uint64(len(m.data) + 1),
	}
	return nil
}

func (m *mockServiceStore) DeregisterLocal(name string) {
	delete(m.data, name)
}

func (m *mockServiceStore) HeartbeatLocal(name string) bool {
	if entry, ok := m.data[name]; ok {
		entry.ExpiresAt = time.Now().Add(30 * time.Second)
		m.data[name] = entry
		return true
	}
	return false
}

func (m *mockServiceStore) GetAllData() map[string]store.ServiceEntrySnapshot {
	result := make(map[string]store.ServiceEntrySnapshot, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

func (m *mockServiceStore) RestoreFromSnapshot(data map[string]store.ServiceEntrySnapshot) error {
	m.data = make(map[string]store.ServiceEntrySnapshot, len(data))
	for k, v := range data {
		m.data[k] = v
	}
	return nil
}

// Helper to create a raft.Log from a command.
func makeLog(t *testing.T, cmd *Command) *raft.Log {
	data, err := cmd.Marshal()
	require.NoError(t, err)
	return &raft.Log{
		Index: 1,
		Term:  1,
		Type:  raft.LogCommand,
		Data:  data,
	}
}

func TestFSM_Apply_KVSet(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create KV set command
	cmd, err := NewCommand(CmdKVSet, KVSetPayload{Key: "foo", Value: "bar"})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify the value was set
	entry, ok := kvStore.data["foo"]
	assert.True(t, ok)
	assert.Equal(t, "bar", entry.Value)
}

func TestFSM_Apply_KVSetWithFlags(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create KV set with flags command
	cmd, err := NewCommand(CmdKVSetWithFlags, KVSetWithFlagsPayload{
		Key:   "flagged",
		Value: "value",
		Flags: 42,
	})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify the value and flags were set
	entry, ok := kvStore.data["flagged"]
	assert.True(t, ok)
	assert.Equal(t, "value", entry.Value)
	assert.Equal(t, uint64(42), entry.Flags)
}

func TestFSM_Apply_KVDelete(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	// Pre-populate data
	kvStore.data["todelete"] = store.KVEntrySnapshot{Value: "exists"}

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create KV delete command
	cmd, err := NewCommand(CmdKVDelete, KVDeletePayload{Key: "todelete"})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify the key was deleted
	_, ok := kvStore.data["todelete"]
	assert.False(t, ok)
}

func TestFSM_Apply_KVBatchSet(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create batch set command
	cmd, err := NewCommand(CmdKVBatchSet, KVBatchSetPayload{
		Items: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify all values were set
	assert.Len(t, kvStore.data, 3)
	assert.Equal(t, "value1", kvStore.data["key1"].Value)
	assert.Equal(t, "value2", kvStore.data["key2"].Value)
	assert.Equal(t, "value3", kvStore.data["key3"].Value)
}

func TestFSM_Apply_KVBatchDelete(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	// Pre-populate data
	kvStore.data["keep"] = store.KVEntrySnapshot{Value: "keeper"}
	kvStore.data["del1"] = store.KVEntrySnapshot{Value: "delete1"}
	kvStore.data["del2"] = store.KVEntrySnapshot{Value: "delete2"}

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create batch delete command
	cmd, err := NewCommand(CmdKVBatchDelete, KVBatchDeletePayload{
		Keys: []string{"del1", "del2"},
	})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify correct keys were deleted
	assert.Len(t, kvStore.data, 1)
	_, ok := kvStore.data["keep"]
	assert.True(t, ok)
}

func TestFSM_Apply_ServiceRegister(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create service register command
	cmd, err := NewCommand(CmdServiceRegister, ServiceRegisterPayload{
		Service: store.Service{
			Name:    "web",
			Address: "10.0.0.1",
			Port:    8080,
			Tags:    []string{"primary", "v2"},
			Meta:    map[string]string{"version": "2.0"},
		},
	})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify service was registered
	entry, ok := serviceStore.data["web"]
	assert.True(t, ok)
	assert.Equal(t, "web", entry.Service.Name)
	assert.Equal(t, "10.0.0.1", entry.Service.Address)
	assert.Equal(t, 8080, entry.Service.Port)
	assert.Equal(t, []string{"primary", "v2"}, entry.Service.Tags)
	assert.Equal(t, "2.0", entry.Service.Meta["version"])
}

func TestFSM_Apply_ServiceDeregister(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	// Pre-populate service
	serviceStore.data["web"] = store.ServiceEntrySnapshot{
		Service: store.ServiceDataSnapshot{Name: "web"},
	}

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create service deregister command
	cmd, err := NewCommand(CmdServiceDeregister, ServiceDeregisterPayload{Name: "web"})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify service was deregistered
	_, ok := serviceStore.data["web"]
	assert.False(t, ok)
}

func TestFSM_Apply_ServiceHeartbeat(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	// Pre-populate service with old expiry
	oldExpiry := time.Now().Add(-5 * time.Second)
	serviceStore.data["web"] = store.ServiceEntrySnapshot{
		Service:   store.ServiceDataSnapshot{Name: "web"},
		ExpiresAt: oldExpiry,
	}

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create service heartbeat command
	cmd, err := NewCommand(CmdServiceHeartbeat, ServiceHeartbeatPayload{Name: "web"})
	require.NoError(t, err)

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.Nil(t, result)

	// Verify service expiry was updated
	entry, ok := serviceStore.data["web"]
	assert.True(t, ok)
	assert.True(t, entry.ExpiresAt.After(oldExpiry))
}

func TestFSM_Apply_UnknownCommand(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create command with invalid type
	cmd := &Command{
		Type:    CommandType(255), // Invalid
		Payload: []byte("{}"),
	}

	// Apply the command
	result := fsm.Apply(makeLog(t, cmd))
	assert.NotNil(t, result)
	assert.Error(t, result.(error))
}

func TestFSM_Snapshot(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	// Pre-populate data
	kvStore.data["key1"] = store.KVEntrySnapshot{Value: "value1", ModifyIndex: 1}
	kvStore.data["key2"] = store.KVEntrySnapshot{Value: "value2", ModifyIndex: 2}
	serviceStore.data["web"] = store.ServiceEntrySnapshot{
		Service: store.ServiceDataSnapshot{Name: "web", Address: "10.0.0.1", Port: 8080},
	}

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create snapshot
	snapshot, err := fsm.Snapshot()
	require.NoError(t, err)

	// Persist snapshot to buffer
	var buf bytes.Buffer
	sink := &mockSnapshotSink{buf: &buf}
	err = snapshot.Persist(sink)
	require.NoError(t, err)

	// Verify snapshot contains expected data
	var data SnapshotData
	err = json.Unmarshal(buf.Bytes(), &data)
	require.NoError(t, err)

	assert.Len(t, data.KVData, 2)
	assert.Equal(t, "value1", data.KVData["key1"].Value)
	assert.Equal(t, "value2", data.KVData["key2"].Value)

	assert.Len(t, data.ServiceData, 1)
	assert.Equal(t, "web", data.ServiceData["web"].Service.Name)
}

func TestFSM_Restore(t *testing.T) {
	kvStore := newMockKVStore()
	serviceStore := newMockServiceStore()

	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create snapshot data
	snapshotData := SnapshotData{
		KVData: map[string]store.KVEntrySnapshot{
			"restored1": {Value: "value1", ModifyIndex: 10},
			"restored2": {Value: "value2", ModifyIndex: 11},
		},
		ServiceData: map[string]store.ServiceEntrySnapshot{
			"restored-svc": {
				Service:     store.ServiceDataSnapshot{Name: "restored-svc", Address: "10.0.0.5", Port: 9000},
				ModifyIndex: 5,
			},
		},
	}

	// Serialize snapshot
	data, err := json.Marshal(snapshotData)
	require.NoError(t, err)

	// Restore from snapshot
	reader := &mockReadCloser{buf: bytes.NewBuffer(data)}
	err = fsm.Restore(reader)
	require.NoError(t, err)

	// Verify KV data was restored
	assert.Len(t, kvStore.data, 2)
	assert.Equal(t, "value1", kvStore.data["restored1"].Value)
	assert.Equal(t, "value2", kvStore.data["restored2"].Value)

	// Verify service data was restored
	assert.Len(t, serviceStore.data, 1)
	assert.Equal(t, "restored-svc", serviceStore.data["restored-svc"].Service.Name)
	assert.Equal(t, "10.0.0.5", serviceStore.data["restored-svc"].Service.Address)
}

// mockSnapshotSink implements raft.SnapshotSink for testing.
type mockSnapshotSink struct {
	buf      *bytes.Buffer
	canceled bool
}

func (m *mockSnapshotSink) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

func (m *mockSnapshotSink) Close() error {
	return nil
}

func (m *mockSnapshotSink) ID() string {
	return "test-snapshot"
}

func (m *mockSnapshotSink) Cancel() error {
	m.canceled = true
	return nil
}

// mockReadCloser implements io.ReadCloser for testing.
type mockReadCloser struct {
	buf *bytes.Buffer
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return m.buf.Read(p)
}

func (m *mockReadCloser) Close() error {
	return nil
}
