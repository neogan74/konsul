package raft

import (
	"github.com/neogan74/konsul/internal/store"
)

// KVStoreInterface defines the interface for KV store operations used by FSM.
// This allows FSM to work with different KV store implementations.
type KVStoreInterface interface {
	// SetLocal stores a key-value pair (without persistence - FSM handles durability via Raft)
	SetLocal(key, value string)

	// SetWithFlagsLocal stores a key-value pair with flags (without persistence)
	SetWithFlagsLocal(key, value string, flags uint64)

	// SetCASLocal performs Compare-And-Swap operation (without persistence)
	SetCASLocal(key, value string, expectedIndex uint64) (uint64, error)

	// DeleteLocal removes a key (without persistence)
	DeleteLocal(key string)

	// DeleteCASLocal performs Compare-And-Swap delete operation (without persistence)
	DeleteCASLocal(key string, expectedIndex uint64) error

	// BatchSetLocal sets multiple key-value pairs (without persistence)
	BatchSetLocal(items map[string]string) error

	// BatchSetCASLocal performs atomic batch set with CAS checks (without persistence)
	BatchSetCASLocal(items map[string]string, expectedIndices map[string]uint64) (map[string]uint64, error)

	// BatchDeleteLocal deletes multiple keys (without persistence)
	BatchDeleteLocal(keys []string) error

	// BatchDeleteCASLocal performs atomic batch delete with CAS checks (without persistence)
	BatchDeleteCASLocal(keys []string, expectedIndices map[string]uint64) error

	// GetEntrySnapshot returns a snapshot of the KVEntry with version information
	GetEntrySnapshot(key string) (store.KVEntrySnapshot, bool)

	// GetAllData returns all KV data for snapshotting
	GetAllData() map[string]store.KVEntrySnapshot

	// RestoreFromSnapshot restores KV data from a snapshot
	RestoreFromSnapshot(data map[string]store.KVEntrySnapshot) error
}

// ServiceStoreInterface defines the interface for service store operations used by FSM.
type ServiceStoreInterface interface {
	// RegisterLocal registers a service (without persistence - FSM handles durability via Raft)
	RegisterLocal(service store.ServiceDataSnapshot) error

	// RegisterCASLocal performs Compare-And-Swap registration (without persistence)
	RegisterCASLocal(service store.ServiceDataSnapshot, expectedIndex uint64) (uint64, error)

	// DeregisterLocal removes a service (without persistence)
	DeregisterLocal(name string)

	// DeregisterCASLocal performs Compare-And-Swap deregistration (without persistence)
	DeregisterCASLocal(name string, expectedIndex uint64) error

	// HeartbeatLocal updates service TTL (without persistence)
	HeartbeatLocal(name string) bool

	// UpdateTTLCheck updates a TTL-based health check (without persistence)
	UpdateTTLCheck(checkID string) error

	// GetEntrySnapshot returns a snapshot of the ServiceEntry with version information
	GetEntrySnapshot(name string) (store.ServiceEntrySnapshot, bool)

	// GetAllData returns all service data for snapshotting
	GetAllData() map[string]store.ServiceEntrySnapshot

	// RestoreFromSnapshot restores service data from a snapshot
	RestoreFromSnapshot(data map[string]store.ServiceEntrySnapshot) error
}
