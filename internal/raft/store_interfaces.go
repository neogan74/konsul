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

	// DeleteLocal removes a key (without persistence)
	DeleteLocal(key string)

	// BatchSetLocal sets multiple key-value pairs (without persistence)
	BatchSetLocal(items map[string]string) error

	// BatchDeleteLocal deletes multiple keys (without persistence)
	BatchDeleteLocal(keys []string) error

	// GetAllData returns all KV data for snapshotting
	GetAllData() map[string]store.KVEntrySnapshot

	// RestoreFromSnapshot restores KV data from a snapshot
	RestoreFromSnapshot(data map[string]store.KVEntrySnapshot) error
}

// ServiceStoreInterface defines the interface for service store operations used by FSM.
type ServiceStoreInterface interface {
	// RegisterLocal registers a service (without persistence - FSM handles durability via Raft)
	RegisterLocal(service store.ServiceDataSnapshot) error

	// DeregisterLocal removes a service (without persistence)
	DeregisterLocal(name string)

	// HeartbeatLocal updates service TTL (without persistence)
	HeartbeatLocal(name string) bool

	// GetAllData returns all service data for snapshotting
	GetAllData() map[string]store.ServiceEntrySnapshot

	// RestoreFromSnapshot restores service data from a snapshot
	RestoreFromSnapshot(data map[string]store.ServiceEntrySnapshot) error
}
