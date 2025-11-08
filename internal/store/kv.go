package store

import (
	"sync"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/persistence"
	"github.com/neogan74/konsul/internal/watch"
)

type KVStore struct {
	Data         map[string]string
	Mutex        sync.RWMutex
	engine       persistence.Engine
	log          logger.Logger
	watchManager *watch.Manager
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

// SetWatchManager sets the watch manager for this KV store
func (kv *KVStore) SetWatchManager(wm *watch.Manager) {
	kv.watchManager = wm
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
	oldValue, existed := kv.Data[key]
	kv.Data[key] = value
	kv.Mutex.Unlock()

	// Persist to storage if engine is available
	if kv.engine != nil {
		if err := kv.engine.Set(key, []byte(value)); err != nil {
			kv.log.Error("Failed to persist key",
				logger.String("key", key),
				logger.Error(err))
		}
	}

	// Notify watchers
	if kv.watchManager != nil {
		event := watch.WatchEvent{
			Type:      watch.EventTypeSet,
			Key:       key,
			Value:     value,
			Timestamp: time.Now().Unix(),
		}
		if existed {
			event.OldValue = oldValue
		}
		kv.watchManager.Notify(event)
	}
}

func (kv *KVStore) Delete(key string) {
	kv.Mutex.Lock()
	oldValue, existed := kv.Data[key]
	delete(kv.Data, key)
	kv.Mutex.Unlock()

	// Delete from persistence if engine is available
	if kv.engine != nil {
		if err := kv.engine.Delete(key); err != nil {
			kv.log.Error("Failed to delete key from persistence",
				logger.String("key", key),
				logger.Error(err))
		}
	}

	// Notify watchers if key existed
	if kv.watchManager != nil && existed {
		event := watch.WatchEvent{
			Type:      watch.EventTypeDelete,
			Key:       key,
			OldValue:  oldValue,
			Timestamp: time.Now().Unix(),
		}
		kv.watchManager.Notify(event)
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

	// Track old values for watch events
	oldValues := make(map[string]string)
	for key, value := range items {
		if oldVal, existed := kv.Data[key]; existed {
			oldValues[key] = oldVal
		}
		kv.Data[key] = value
	}
	kv.Mutex.Unlock()

	// Persist if engine is available
	if kv.engine != nil {
		byteItems := make(map[string][]byte)
		for key, value := range items {
			byteItems[key] = []byte(value)
		}
		if err := kv.engine.BatchSet(byteItems); err != nil {
			return err
		}
	}

	// Notify watchers
	if kv.watchManager != nil {
		timestamp := time.Now().Unix()
		for key, value := range items {
			event := watch.WatchEvent{
				Type:      watch.EventTypeSet,
				Key:       key,
				Value:     value,
				Timestamp: timestamp,
			}
			if oldVal, existed := oldValues[key]; existed {
				event.OldValue = oldVal
			}
			kv.watchManager.Notify(event)
		}
	}

	return nil
}

// BatchDelete deletes multiple keys atomically
func (kv *KVStore) BatchDelete(keys []string) error {
	kv.Mutex.Lock()

	// Track old values for watch events
	oldValues := make(map[string]string)
	for _, key := range keys {
		if oldVal, existed := kv.Data[key]; existed {
			oldValues[key] = oldVal
		}
		delete(kv.Data, key)
	}
	kv.Mutex.Unlock()

	// Delete from persistence if engine is available
	if kv.engine != nil {
		if err := kv.engine.BatchDelete(keys); err != nil {
			return err
		}
	}

	// Notify watchers for deleted keys
	if kv.watchManager != nil {
		timestamp := time.Now().Unix()
		for key, oldVal := range oldValues {
			event := watch.WatchEvent{
				Type:      watch.EventTypeDelete,
				Key:       key,
				OldValue:  oldVal,
				Timestamp: timestamp,
			}
			kv.watchManager.Notify(event)
		}
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
