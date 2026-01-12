package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/store"
)

// KonsulFSM implements the raft.FSM interface.
// It applies commands from the Raft log to the KV and Service stores.
type KonsulFSM struct {
	mu           sync.RWMutex
	kvStore      KVStoreInterface
	serviceStore ServiceStoreInterface

	// Metrics callbacks (optional)
	onApply func(cmdType CommandType, duration float64, err error)
}

// FSMConfig contains configuration for the FSM.
type FSMConfig struct {
	KVStore      KVStoreInterface
	ServiceStore ServiceStoreInterface
	OnApply      func(cmdType CommandType, duration float64, err error)
}

// NewFSM creates a new KonsulFSM instance.
func NewFSM(cfg FSMConfig) *KonsulFSM {
	return &KonsulFSM{
		kvStore:      cfg.KVStore,
		serviceStore: cfg.ServiceStore,
		onApply:      cfg.OnApply,
	}
}

// Apply implements raft.FSM.Apply.
// It applies a Raft log entry to the local state.
// This is called by Raft after a log entry is committed.
func (f *KonsulFSM) Apply(log *raft.Log) interface{} {
	// Parse the command
	cmd, err := UnmarshalCommand(log.Data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal command: %w", err)
	}

	// Apply the command based on its type
	var applyErr error
	switch cmd.Type {
	case CmdKVSet:
		applyErr = f.applyKVSet(cmd.Payload)
	case CmdKVSetWithFlags:
		applyErr = f.applyKVSetWithFlags(cmd.Payload)
	case CmdKVSetCAS:
		applyErr = f.applyKVSetCAS(cmd.Payload)
	case CmdKVDelete:
		applyErr = f.applyKVDelete(cmd.Payload)
	case CmdKVDeleteCAS:
		applyErr = f.applyKVDeleteCAS(cmd.Payload)
	case CmdKVBatchSet:
		applyErr = f.applyKVBatchSet(cmd.Payload)
	case CmdKVBatchSetCAS:
		applyErr = f.applyKVBatchSetCAS(cmd.Payload)
	case CmdKVBatchDelete:
		applyErr = f.applyKVBatchDelete(cmd.Payload)
	case CmdKVBatchDeleteCAS:
		applyErr = f.applyKVBatchDeleteCAS(cmd.Payload)
	case CmdServiceRegister:
		applyErr = f.applyServiceRegister(cmd.Payload)
	case CmdServiceRegisterCAS:
		applyErr = f.applyServiceRegisterCAS(cmd.Payload)
	case CmdServiceDeregister:
		applyErr = f.applyServiceDeregister(cmd.Payload)
	case CmdServiceDeregisterCAS:
		applyErr = f.applyServiceDeregisterCAS(cmd.Payload)
	case CmdServiceHeartbeat:
		applyErr = f.applyServiceHeartbeat(cmd.Payload)
	case CmdHealthTTLUpdate:
		applyErr = f.applyHealthTTLUpdate(cmd.Payload)
	default:
		applyErr = fmt.Errorf("unknown command type: %d", cmd.Type)
	}

	return applyErr
}

// --- KV Apply Methods ---

func (f *KonsulFSM) applyKVSet(payload []byte) error {
	var p KVSetPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVSetPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.kvStore.SetLocal(p.Key, p.Value)
	return nil
}

func (f *KonsulFSM) applyKVSetWithFlags(payload []byte) error {
	var p KVSetWithFlagsPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVSetWithFlagsPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.kvStore.SetWithFlagsLocal(p.Key, p.Value, p.Flags)
	return nil
}

func (f *KonsulFSM) applyKVDelete(payload []byte) error {
	var p KVDeletePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVDeletePayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.kvStore.DeleteLocal(p.Key)
	return nil
}

func (f *KonsulFSM) applyKVBatchSet(payload []byte) error {
	var p KVBatchSetPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVBatchSetPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.kvStore.BatchSetLocal(p.Items)
}

func (f *KonsulFSM) applyKVBatchDelete(payload []byte) error {
	var p KVBatchDeletePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVBatchDeletePayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.kvStore.BatchDeleteLocal(p.Keys)
}

func (f *KonsulFSM) applyKVSetCAS(payload []byte) error {
	var p KVSetCASPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVSetCASPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	_, err := f.kvStore.SetCASLocal(p.Key, p.Value, p.ExpectedIndex)
	return err
}

func (f *KonsulFSM) applyKVDeleteCAS(payload []byte) error {
	var p KVDeleteCASPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVDeleteCASPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.kvStore.DeleteCASLocal(p.Key, p.ExpectedIndex)
}

func (f *KonsulFSM) applyKVBatchSetCAS(payload []byte) error {
	var p KVBatchSetCASPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVBatchSetCASPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	_, err := f.kvStore.BatchSetCASLocal(p.Items, p.ExpectedIndices)
	return err
}

func (f *KonsulFSM) applyKVBatchDeleteCAS(payload []byte) error {
	var p KVBatchDeleteCASPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal KVBatchDeleteCASPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.kvStore.BatchDeleteCASLocal(p.Keys, p.ExpectedIndices)
}

// --- Service Apply Methods ---

func (f *KonsulFSM) applyServiceRegister(payload []byte) error {
	var p ServiceRegisterPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal ServiceRegisterPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	service := store.ServiceDataSnapshot{
		Name:    p.Service.Name,
		Address: p.Service.Address,
		Port:    p.Service.Port,
		Tags:    p.Service.Tags,
		Meta:    p.Service.Meta,
	}

	return f.serviceStore.RegisterLocal(service)
}

func (f *KonsulFSM) applyServiceDeregister(payload []byte) error {
	var p ServiceDeregisterPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal ServiceDeregisterPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.serviceStore.DeregisterLocal(p.Name)
	return nil
}

func (f *KonsulFSM) applyServiceHeartbeat(payload []byte) error {
	var p ServiceHeartbeatPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal ServiceHeartbeatPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.serviceStore.HeartbeatLocal(p.Name)
	return nil
}

func (f *KonsulFSM) applyServiceRegisterCAS(payload []byte) error {
	var p ServiceRegisterCASPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal ServiceRegisterCASPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	service := store.ServiceDataSnapshot{
		Name:    p.Service.Name,
		Address: p.Service.Address,
		Port:    p.Service.Port,
		Tags:    p.Service.Tags,
		Meta:    p.Service.Meta,
	}

	_, err := f.serviceStore.RegisterCASLocal(service, p.ExpectedIndex)
	return err
}

func (f *KonsulFSM) applyServiceDeregisterCAS(payload []byte) error {
	var p ServiceDeregisterCASPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal ServiceDeregisterCASPayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.serviceStore.DeregisterCASLocal(p.Name, p.ExpectedIndex)
}

func (f *KonsulFSM) applyHealthTTLUpdate(payload []byte) error {
	var p HealthTTLUpdatePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("failed to unmarshal HealthTTLUpdatePayload: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.serviceStore.UpdateTTLCheck(p.CheckID)
}

// Snapshot implements raft.FSM.Snapshot.
// It returns a snapshot of the current state for persistence.
// Raft calls this periodically to compact the log.
func (f *KonsulFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create deep copies of the data
	kvData := f.kvStore.GetAllData()
	serviceData := f.serviceStore.GetAllData()

	return &KonsulSnapshot{
		KVData:      kvData,
		ServiceData: serviceData,
	}, nil
}

// Restore implements raft.FSM.Restore.
// It restores the FSM state from a snapshot.
// This is called when a node joins the cluster or recovers from a crash.
func (f *KonsulFSM) Restore(rc io.ReadCloser) error {
	defer func() { _ = rc.Close() }()

	var snapshot SnapshotData
	if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Restore KV store
	if err := f.kvStore.RestoreFromSnapshot(snapshot.KVData); err != nil {
		return fmt.Errorf("failed to restore KV store: %w", err)
	}

	// Restore service store
	if err := f.serviceStore.RestoreFromSnapshot(snapshot.ServiceData); err != nil {
		return fmt.Errorf("failed to restore service store: %w", err)
	}

	return nil
}

// SnapshotData represents the data structure stored in a snapshot.
type SnapshotData struct {
	KVData      map[string]store.KVEntrySnapshot      `json:"kv_data"`
	ServiceData map[string]store.ServiceEntrySnapshot `json:"service_data"`
}

// KonsulSnapshot implements raft.FSMSnapshot.
// It holds a point-in-time snapshot of the FSM state.
type KonsulSnapshot struct {
	KVData      map[string]store.KVEntrySnapshot
	ServiceData map[string]store.ServiceEntrySnapshot
}

// Persist implements raft.FSMSnapshot.Persist.
// It writes the snapshot to the given sink.
func (s *KonsulSnapshot) Persist(sink raft.SnapshotSink) error {
	data := SnapshotData{
		KVData:      s.KVData,
		ServiceData: s.ServiceData,
	}

	// Encode the snapshot as JSON
	if err := json.NewEncoder(sink).Encode(data); err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return sink.Close()
}

// Release implements raft.FSMSnapshot.Release.
// It is called when Raft is finished with the snapshot.
func (s *KonsulSnapshot) Release() {
	// Nothing to release - we use plain Go maps
}
