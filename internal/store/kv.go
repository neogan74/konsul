package store

import (
	"sync"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/persistence"
)

type KVStore struct {
	Data   map[string]string
	Mutex  sync.RWMutex
	engine persistence.Engine
	log    logger.Logger
}

// NewKVStore creates a KV store with optional persistence
func NewKVStore() *KVStore {
	return &KVStore{
		Data: make(map[string]string),
		log:  logger.GetDefault(),
	}
}

// NewKVStoreWithPersistence creates a KV store with persistence engine
func NewKVStoreWithPersistence(engine persistence.Engine, log logger.Logger) (*KVStore, error) {
	store := &KVStore{
		Data:   make(map[string]string),
		engine: engine,
		log:    log,
	}

	// Load existing data from persistence if available
	if engine != nil {
		if err := store.loadFromPersistence(); err != nil {
			log.Warn("Failed to load KV data from persistence", logger.Error(err))
		}
	}

	return store, nil
}

func (kv *KVStore) loadFromPersistence() error {
	if kv.engine == nil {
		return nil
	}

	keys, err := kv.engine.List("")
	if err != nil {
		return err
	}

	for _, key := range keys {
		value, err := kv.engine.Get(key)
		if err != nil {
			kv.log.Warn("Failed to load key from persistence",
				logger.String("key", key),
				logger.Error(err))
			continue
		}
		kv.Data[key] = string(value)
	}

	kv.log.Info("Loaded KV data from persistence",
		logger.Int("keys", len(keys)))
	return nil
}

func (kv *KVStore) Get(key string) (string, bool) {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()
	value, ok := kv.Data[key]
	return value, ok
}

func (kv *KVStore) Set(key, value string) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	kv.Data[key] = value

	// Persist to storage if engine is available
	if kv.engine != nil {
		if err := kv.engine.Set(key, []byte(value)); err != nil {
			kv.log.Error("Failed to persist key",
				logger.String("key", key),
				logger.Error(err))
		}
	}
}

func (kv *KVStore) Delete(key string) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	delete(kv.Data, key)

	// Delete from persistence if engine is available
	if kv.engine != nil {
		if err := kv.engine.Delete(key); err != nil {
			kv.log.Error("Failed to delete key from persistence",
				logger.String("key", key),
				logger.Error(err))
		}
	}
}

func (kv *KVStore) List() []string {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()
	keys := make([]string, 0, len(kv.Data))
	for key := range kv.Data {
		keys = append(keys, key)
	}
	return keys
}

// BatchSet sets multiple key-value pairs atomically
func (kv *KVStore) BatchSet(items map[string]string) error {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	// Update in-memory store
	for key, value := range items {
		kv.Data[key] = value
	}

	// Persist if engine is available
	if kv.engine != nil {
		byteItems := make(map[string][]byte)
		for key, value := range items {
			byteItems[key] = []byte(value)
		}
		return kv.engine.BatchSet(byteItems)
	}

	return nil
}

// BatchDelete deletes multiple keys atomically
func (kv *KVStore) BatchDelete(keys []string) error {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	// Delete from in-memory store
	for _, key := range keys {
		delete(kv.Data, key)
	}

	// Delete from persistence if engine is available
	if kv.engine != nil {
		return kv.engine.BatchDelete(keys)
	}

	return nil
}

// Close closes the persistence engine
func (kv *KVStore) Close() error {
	if kv.engine != nil {
		return kv.engine.Close()
	}
	return nil
}