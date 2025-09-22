package persistence

import (
	"time"
)

// Engine represents a persistence backend
type Engine interface {
	// KV operations
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	List(prefix string) ([]string, error)

	// Service operations
	GetService(name string) ([]byte, error)
	SetService(name string, data []byte, ttl time.Duration) error
	DeleteService(name string) error
	ListServices() ([]string, error)

	// Batch operations
	BatchSet(items map[string][]byte) error
	BatchDelete(keys []string) error

	// Management
	Close() error
	Backup(path string) error
	Restore(path string) error

	// Transaction support
	BeginTx() (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	Set(key string, value []byte) error
	Delete(key string) error
	Commit() error
	Rollback() error
}

// Config holds persistence configuration
type Config struct {
	Enabled    bool
	Type       string // "memory", "badger", "bolt"
	DataDir    string
	BackupDir  string
	SyncWrites bool
	WALEnabled bool
}