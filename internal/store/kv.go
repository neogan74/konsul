package store

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/persistence"
	"github.com/neogan74/konsul/internal/watch"
)

// KVEntry represents a key-value entry with version tracking for CAS operations
type KVEntry struct {
	Value       string `json:"value"`
	ModifyIndex uint64 `json:"modify_index"`
	CreateIndex uint64 `json:"create_index"`
	Flags       uint64 `json:"flags,omitempty"`
}

type KVStore struct {
	Data         map[string]KVEntry
	Mutex        sync.RWMutex
	globalIndex  uint64 // Monotonically increasing global index
	engine       persistence.Engine
	log          logger.Logger
	watchManager *watch.Manager
}

// NewKVStore creates a KV store with optional persistence
func NewKVStore() *KVStore {
	return &KVStore{
		Data:        make(map[string]KVEntry),
		globalIndex: 0,
		log:         logger.GetDefault(),
	}
}

// NewKVStoreWithPersistence creates a KV store with persistence engine
func NewKVStoreWithPersistence(engine persistence.Engine, log logger.Logger) (*KVStore, error) {
	store := &KVStore{
		Data:        make(map[string]KVEntry),
		globalIndex: 0,
		engine:      engine,
		log:         log,
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

	var maxIndex uint64
	for _, key := range keys {
		value, err := kv.engine.Get(key)
		if err != nil {
			kv.log.Warn("Failed to load key from persistence",
				logger.String("key", key),
				logger.Error(err))
			continue
		}
		// Try to unmarshal as KVEntry first (new format)
		var entry KVEntry
		if err := json.Unmarshal(value, &entry); err != nil {
			// Fallback to old format (plain string value)
			entry = KVEntry{
				Value:       string(value),
				ModifyIndex: 1,
				CreateIndex: 1,
			}
		}
		kv.Data[key] = entry
		if entry.ModifyIndex > maxIndex {
			maxIndex = entry.ModifyIndex
		}
	}

	// Set global index to max found index
	kv.globalIndex = maxIndex

	kv.log.Info("Loaded KV data from persistence",
		logger.Int("keys", len(keys)),
		logger.String("max_index", fmt.Sprintf("%d", maxIndex)))
	return nil
}

func (kv *KVStore) Get(key string) (string, bool) {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()
	entry, ok := kv.Data[key]
	if !ok {
		return "", false
	}
	return entry.Value, true
}

// GetEntry returns the full KVEntry with version information
func (kv *KVStore) GetEntry(key string) (KVEntry, bool) {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()
	entry, ok := kv.Data[key]
	return entry, ok
}

// nextIndex atomically increments and returns the next global index
func (kv *KVStore) nextIndex() uint64 {
	return atomic.AddUint64(&kv.globalIndex, 1)
}

func (kv *KVStore) Set(key, value string) {
	kv.Mutex.Lock()
	oldEntry, existed := kv.Data[key]

	newIndex := kv.nextIndex()
	entry := KVEntry{
		Value:       value,
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
		entry.Flags = oldEntry.Flags
	} else {
		entry.CreateIndex = newIndex
	}
	kv.Data[key] = entry
	kv.Mutex.Unlock()

	// Persist to storage if engine is available
	if kv.engine != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			kv.log.Error("Failed to marshal KV entry",
				logger.String("key", key),
				logger.Error(err))
		} else if err := kv.engine.Set(key, data); err != nil {
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
			event.OldValue = oldEntry.Value
		}
		kv.watchManager.Notify(event)
	}
}

// SetCAS performs a Compare-And-Swap operation
// It will only update the value if the current ModifyIndex matches the expected index
// If expectedIndex is 0, it means "create only if not exists"
// Returns the new ModifyIndex on success, or error on conflict
func (kv *KVStore) SetCAS(key, value string, expectedIndex uint64) (uint64, error) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	oldEntry, existed := kv.Data[key]

	// Check CAS condition
	if expectedIndex == 0 {
		// Create only if not exists
		if existed {
			return 0, &CASConflictError{
				Key:           key,
				ExpectedIndex: 0,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "key",
			}
		}
	} else {
		// Update only if index matches
		if !existed {
			return 0, &NotFoundError{Type: "key", Key: key}
		}
		if oldEntry.ModifyIndex != expectedIndex {
			return 0, &CASConflictError{
				Key:           key,
				ExpectedIndex: expectedIndex,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "key",
			}
		}
	}

	newIndex := kv.nextIndex()
	entry := KVEntry{
		Value:       value,
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
		entry.Flags = oldEntry.Flags
	} else {
		entry.CreateIndex = newIndex
	}
	kv.Data[key] = entry

	// Persist to storage if engine is available
	if kv.engine != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			kv.log.Error("Failed to marshal KV entry",
				logger.String("key", key),
				logger.Error(err))
			return newIndex, err
		}
		if err := kv.engine.Set(key, data); err != nil {
			kv.log.Error("Failed to persist key",
				logger.String("key", key),
				logger.Error(err))
			return newIndex, err
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
			event.OldValue = oldEntry.Value
		}
		kv.watchManager.Notify(event)
	}

	return newIndex, nil
}

// SetWithFlags sets a value with custom flags
func (kv *KVStore) SetWithFlags(key, value string, flags uint64) {
	kv.Mutex.Lock()
	oldEntry, existed := kv.Data[key]

	newIndex := kv.nextIndex()
	entry := KVEntry{
		Value:       value,
		ModifyIndex: newIndex,
		Flags:       flags,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
	} else {
		entry.CreateIndex = newIndex
	}
	kv.Data[key] = entry
	kv.Mutex.Unlock()

	// Persist to storage if engine is available
	if kv.engine != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			kv.log.Error("Failed to marshal KV entry",
				logger.String("key", key),
				logger.Error(err))
		} else if err := kv.engine.Set(key, data); err != nil {
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
			event.OldValue = oldEntry.Value
		}
		kv.watchManager.Notify(event)
	}
}

func (kv *KVStore) Delete(key string) {
	kv.Mutex.Lock()
	oldEntry, existed := kv.Data[key]
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
			OldValue:  oldEntry.Value,
			Timestamp: time.Now().Unix(),
		}
		kv.watchManager.Notify(event)
	}
}

// DeleteCAS performs a Compare-And-Swap delete operation
// It will only delete the value if the current ModifyIndex matches the expected index
// Returns error on conflict
func (kv *KVStore) DeleteCAS(key string, expectedIndex uint64) error {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	oldEntry, existed := kv.Data[key]
	if !existed {
		return &NotFoundError{Type: "key", Key: key}
	}

	if oldEntry.ModifyIndex != expectedIndex {
		return &CASConflictError{
			Key:           key,
			ExpectedIndex: expectedIndex,
			CurrentIndex:  oldEntry.ModifyIndex,
			OperationType: "key",
		}
	}

	delete(kv.Data, key)

	// Delete from persistence if engine is available
	if kv.engine != nil {
		if err := kv.engine.Delete(key); err != nil {
			kv.log.Error("Failed to delete key from persistence",
				logger.String("key", key),
				logger.Error(err))
			return err
		}
	}

	// Notify watchers
	if kv.watchManager != nil {
		event := watch.WatchEvent{
			Type:      watch.EventTypeDelete,
			Key:       key,
			OldValue:  oldEntry.Value,
			Timestamp: time.Now().Unix(),
		}
		kv.watchManager.Notify(event)
	}

	return nil
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

// ListEntries returns all key-value entries with their metadata
func (kv *KVStore) ListEntries() map[string]KVEntry {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()
	result := make(map[string]KVEntry, len(kv.Data))
	for key, entry := range kv.Data {
		result[key] = entry
	}
	return result
}

// BatchGet retrieves multiple keys at once
// Returns a map of key to value, and a slice of keys that were not found
func (kv *KVStore) BatchGet(keys []string) (map[string]string, []string) {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()

	found := make(map[string]string)
	notFound := make([]string, 0)

	for _, key := range keys {
		if entry, ok := kv.Data[key]; ok {
			found[key] = entry.Value
		} else {
			notFound = append(notFound, key)
		}
	}

	return found, notFound
}

// BatchGetEntries retrieves multiple entries with their metadata
func (kv *KVStore) BatchGetEntries(keys []string) (map[string]KVEntry, []string) {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()

	found := make(map[string]KVEntry)
	notFound := make([]string, 0)

	for _, key := range keys {
		if entry, ok := kv.Data[key]; ok {
			found[key] = entry
		} else {
			notFound = append(notFound, key)
		}
	}

	return found, notFound
}

// BatchSet sets multiple key-value pairs atomically
func (kv *KVStore) BatchSet(items map[string]string) error {
	kv.Mutex.Lock()

	// Track old values for watch events
	oldEntries := make(map[string]KVEntry)
	newEntries := make(map[string]KVEntry)

	for key, value := range items {
		oldEntry, existed := kv.Data[key]
		if existed {
			oldEntries[key] = oldEntry
		}

		newIndex := kv.nextIndex()
		entry := KVEntry{
			Value:       value,
			ModifyIndex: newIndex,
		}
		if existed {
			entry.CreateIndex = oldEntry.CreateIndex
			entry.Flags = oldEntry.Flags
		} else {
			entry.CreateIndex = newIndex
		}
		kv.Data[key] = entry
		newEntries[key] = entry
	}
	kv.Mutex.Unlock()

	// Persist if engine is available
	if kv.engine != nil {
		byteItems := make(map[string][]byte)
		for key, entry := range newEntries {
			data, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			byteItems[key] = data
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
			if oldEntry, existed := oldEntries[key]; existed {
				event.OldValue = oldEntry.Value
			}
			kv.watchManager.Notify(event)
		}
	}

	return nil
}

// BatchSetCAS performs atomic batch set with CAS checks
// Each item must have a matching expectedIndex, or 0 for create-only
// Returns map of new indices on success, or error on first conflict
func (kv *KVStore) BatchSetCAS(items map[string]string, expectedIndices map[string]uint64) (map[string]uint64, error) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	// First pass: validate all CAS conditions
	for key := range items {
		expectedIndex := expectedIndices[key]
		oldEntry, existed := kv.Data[key]

		if expectedIndex == 0 {
			if existed {
				return nil, &CASConflictError{
					Key:           key,
					ExpectedIndex: 0,
					CurrentIndex:  oldEntry.ModifyIndex,
					OperationType: "key",
				}
			}
		} else {
			if !existed {
				return nil, &NotFoundError{Type: "key", Key: key}
			}
			if oldEntry.ModifyIndex != expectedIndex {
				return nil, &CASConflictError{
					Key:           key,
					ExpectedIndex: expectedIndex,
					CurrentIndex:  oldEntry.ModifyIndex,
					OperationType: "key",
				}
			}
		}
	}

	// Second pass: apply all updates
	oldEntries := make(map[string]KVEntry)
	newEntries := make(map[string]KVEntry)
	newIndices := make(map[string]uint64)

	for key, value := range items {
		oldEntry, existed := kv.Data[key]
		if existed {
			oldEntries[key] = oldEntry
		}

		newIndex := kv.nextIndex()
		entry := KVEntry{
			Value:       value,
			ModifyIndex: newIndex,
		}
		if existed {
			entry.CreateIndex = oldEntry.CreateIndex
			entry.Flags = oldEntry.Flags
		} else {
			entry.CreateIndex = newIndex
		}
		kv.Data[key] = entry
		newEntries[key] = entry
		newIndices[key] = newIndex
	}

	// Persist if engine is available
	if kv.engine != nil {
		byteItems := make(map[string][]byte)
		for key, entry := range newEntries {
			data, err := json.Marshal(entry)
			if err != nil {
				return newIndices, err
			}
			byteItems[key] = data
		}
		if err := kv.engine.BatchSet(byteItems); err != nil {
			return newIndices, err
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
			if oldEntry, existed := oldEntries[key]; existed {
				event.OldValue = oldEntry.Value
			}
			kv.watchManager.Notify(event)
		}
	}

	return newIndices, nil
}

// BatchDelete deletes multiple keys atomically
func (kv *KVStore) BatchDelete(keys []string) error {
	kv.Mutex.Lock()

	// Track old values for watch events
	oldEntries := make(map[string]KVEntry)
	for _, key := range keys {
		if oldEntry, existed := kv.Data[key]; existed {
			oldEntries[key] = oldEntry
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
		for key, oldEntry := range oldEntries {
			event := watch.WatchEvent{
				Type:      watch.EventTypeDelete,
				Key:       key,
				OldValue:  oldEntry.Value,
				Timestamp: timestamp,
			}
			kv.watchManager.Notify(event)
		}
	}

	return nil
}

// BatchDeleteCAS performs atomic batch delete with CAS checks
// Each key must have a matching expectedIndex
// Returns error on first conflict
func (kv *KVStore) BatchDeleteCAS(keys []string, expectedIndices map[string]uint64) error {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	// First pass: validate all CAS conditions
	for _, key := range keys {
		expectedIndex := expectedIndices[key]
		oldEntry, existed := kv.Data[key]

		if !existed {
			return &NotFoundError{Type: "key", Key: key}
		}
		if oldEntry.ModifyIndex != expectedIndex {
			return &CASConflictError{
				Key:           key,
				ExpectedIndex: expectedIndex,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "key",
			}
		}
	}

	// Second pass: delete all keys
	oldEntries := make(map[string]KVEntry)
	for _, key := range keys {
		oldEntries[key] = kv.Data[key]
		delete(kv.Data, key)
	}

	// Delete from persistence if engine is available
	if kv.engine != nil {
		if err := kv.engine.BatchDelete(keys); err != nil {
			return err
		}
	}

	// Notify watchers for deleted keys
	if kv.watchManager != nil {
		timestamp := time.Now().Unix()
		for key, oldEntry := range oldEntries {
			event := watch.WatchEvent{
				Type:      watch.EventTypeDelete,
				Key:       key,
				OldValue:  oldEntry.Value,
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

// =============================================================================
// Raft Integration Methods
// These methods are used by the Raft FSM to apply changes without persistence.
// Raft handles durability through log replication, so we skip the persistence layer.
// =============================================================================

// SetLocal stores a key-value pair without persisting to the storage engine.
// This is used by Raft FSM when applying committed log entries.
func (kv *KVStore) SetLocal(key, value string) {
	kv.Mutex.Lock()
	oldEntry, existed := kv.Data[key]

	newIndex := kv.nextIndex()
	entry := KVEntry{
		Value:       value,
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
		entry.Flags = oldEntry.Flags
	} else {
		entry.CreateIndex = newIndex
	}
	kv.Data[key] = entry
	kv.Mutex.Unlock()

	// Notify watchers (watchers still work in Raft mode)
	if kv.watchManager != nil {
		event := watch.WatchEvent{
			Type:      watch.EventTypeSet,
			Key:       key,
			Value:     value,
			Timestamp: time.Now().Unix(),
		}
		if existed {
			event.OldValue = oldEntry.Value
		}
		kv.watchManager.Notify(event)
	}
}

// SetWithFlagsLocal stores a key-value pair with flags without persisting.
// This is used by Raft FSM when applying committed log entries.
func (kv *KVStore) SetWithFlagsLocal(key, value string, flags uint64) {
	kv.Mutex.Lock()
	oldEntry, existed := kv.Data[key]

	newIndex := kv.nextIndex()
	entry := KVEntry{
		Value:       value,
		ModifyIndex: newIndex,
		Flags:       flags,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
	} else {
		entry.CreateIndex = newIndex
	}
	kv.Data[key] = entry
	kv.Mutex.Unlock()

	// Notify watchers
	if kv.watchManager != nil {
		event := watch.WatchEvent{
			Type:      watch.EventTypeSet,
			Key:       key,
			Value:     value,
			Timestamp: time.Now().Unix(),
		}
		if existed {
			event.OldValue = oldEntry.Value
		}
		kv.watchManager.Notify(event)
	}
}

// SetCASLocal performs Compare-And-Swap without persistence.
func (kv *KVStore) SetCASLocal(key, value string, expectedIndex uint64) (uint64, error) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	oldEntry, existed := kv.Data[key]

	// Check CAS condition
	if expectedIndex == 0 {
		if existed {
			return 0, &CASConflictError{
				Key:           key,
				ExpectedIndex: 0,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "key",
			}
		}
	} else {
		if !existed {
			return 0, &NotFoundError{Type: "key", Key: key}
		}
		if oldEntry.ModifyIndex != expectedIndex {
			return 0, &CASConflictError{
				Key:           key,
				ExpectedIndex: expectedIndex,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "key",
			}
		}
	}

	newIndex := kv.nextIndex()
	entry := KVEntry{
		Value:       value,
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
		entry.Flags = oldEntry.Flags
	} else {
		entry.CreateIndex = newIndex
	}
	kv.Data[key] = entry

	// Notify watchers
	if kv.watchManager != nil {
		event := watch.WatchEvent{
			Type:      watch.EventTypeSet,
			Key:       key,
			Value:     value,
			Timestamp: time.Now().Unix(),
		}
		if existed {
			event.OldValue = oldEntry.Value
		}
		kv.watchManager.Notify(event)
	}

	return newIndex, nil
}

// DeleteCASLocal performs Compare-And-Swap delete without persistence.
func (kv *KVStore) DeleteCASLocal(key string, expectedIndex uint64) error {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	oldEntry, existed := kv.Data[key]
	if !existed {
		return &NotFoundError{Type: "key", Key: key}
	}

	if oldEntry.ModifyIndex != expectedIndex {
		return &CASConflictError{
			Key:           key,
			ExpectedIndex: expectedIndex,
			CurrentIndex:  oldEntry.ModifyIndex,
			OperationType: "key",
		}
	}

	delete(kv.Data, key)

	// Notify watchers
	if kv.watchManager != nil {
		event := watch.WatchEvent{
			Type:      watch.EventTypeDelete,
			Key:       key,
			OldValue:  oldEntry.Value,
			Timestamp: time.Now().Unix(),
		}
		kv.watchManager.Notify(event)
	}

	return nil
}

// BatchSetCASLocal performs atomic batch set with CAS checks without persistence.
func (kv *KVStore) BatchSetCASLocal(items map[string]string, expectedIndices map[string]uint64) (map[string]uint64, error) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	// First pass: validate all CAS conditions
	for key := range items {
		expectedIndex := expectedIndices[key]
		oldEntry, existed := kv.Data[key]

		if expectedIndex == 0 {
			if existed {
				return nil, &CASConflictError{
					Key:           key,
					ExpectedIndex: 0,
					CurrentIndex:  oldEntry.ModifyIndex,
					OperationType: "key",
				}
			}
		} else {
			if !existed {
				return nil, &NotFoundError{Type: "key", Key: key}
			}
			if oldEntry.ModifyIndex != expectedIndex {
				return nil, &CASConflictError{
					Key:           key,
					ExpectedIndex: expectedIndex,
					CurrentIndex:  oldEntry.ModifyIndex,
					OperationType: "key",
				}
			}
		}
	}

	// Second pass: apply all updates
	oldEntries := make(map[string]KVEntry)
	newIndices := make(map[string]uint64)

	for key, value := range items {
		oldEntry, existed := kv.Data[key]
		if existed {
			oldEntries[key] = oldEntry
		}

		newIndex := kv.nextIndex()
		entry := KVEntry{
			Value:       value,
			ModifyIndex: newIndex,
		}
		if existed {
			entry.CreateIndex = oldEntry.CreateIndex
			entry.Flags = oldEntry.Flags
		} else {
			entry.CreateIndex = newIndex
		}
		kv.Data[key] = entry
		newIndices[key] = newIndex
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
			if oldEntry, existed := oldEntries[key]; existed {
				event.OldValue = oldEntry.Value
			}
			kv.watchManager.Notify(event)
		}
	}

	return newIndices, nil
}

// BatchDeleteCASLocal performs atomic batch delete with CAS checks without persistence.
func (kv *KVStore) BatchDeleteCASLocal(keys []string, expectedIndices map[string]uint64) error {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	// First pass: validate all CAS conditions
	for _, key := range keys {
		expectedIndex := expectedIndices[key]
		oldEntry, existed := kv.Data[key]

		if !existed {
			return &NotFoundError{Type: "key", Key: key}
		}
		if oldEntry.ModifyIndex != expectedIndex {
			return &CASConflictError{
				Key:           key,
				ExpectedIndex: expectedIndex,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "key",
			}
		}
	}

	// Second pass: delete all keys
	oldEntries := make(map[string]KVEntry)
	for _, key := range keys {
		oldEntries[key] = kv.Data[key]
		delete(kv.Data, key)
	}

	// Notify watchers
	if kv.watchManager != nil {
		timestamp := time.Now().Unix()
		for key, oldEntry := range oldEntries {
			event := watch.WatchEvent{
				Type:      watch.EventTypeDelete,
				Key:       key,
				OldValue:  oldEntry.Value,
				Timestamp: timestamp,
			}
			kv.watchManager.Notify(event)
		}
	}

	return nil
}

// GetEntrySnapshot returns a snapshot of a KV entry.
func (kv *KVStore) GetEntrySnapshot(key string) (KVEntrySnapshot, bool) {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()

	entry, ok := kv.Data[key]
	if !ok {
		return KVEntrySnapshot{}, false
	}

	return KVEntrySnapshot{
		Value:       entry.Value,
		Flags:       entry.Flags,
		ModifyIndex: entry.ModifyIndex,
		CreateIndex: entry.CreateIndex,
	}, true
}

// BatchSetLocal sets multiple key-value pairs without persisting.
// This is used by Raft FSM when applying committed log entries.
func (kv *KVStore) BatchSetLocal(items map[string]string) error {
	kv.Mutex.Lock()

	// Track old values for watch events
	oldEntries := make(map[string]KVEntry)

	for key, value := range items {
		oldEntry, existed := kv.Data[key]
		if existed {
			oldEntries[key] = oldEntry
		}

		newIndex := kv.nextIndex()
		entry := KVEntry{
			Value:       value,
			ModifyIndex: newIndex,
		}
		if existed {
			entry.CreateIndex = oldEntry.CreateIndex
			entry.Flags = oldEntry.Flags
		} else {
			entry.CreateIndex = newIndex
		}
		kv.Data[key] = entry
	}
	kv.Mutex.Unlock()

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
			if oldEntry, existed := oldEntries[key]; existed {
				event.OldValue = oldEntry.Value
			}
			kv.watchManager.Notify(event)
		}
	}

	return nil
}

// DeleteLocal removes a key without persisting.
// This is used by Raft FSM when applying committed log entries.
func (kv *KVStore) DeleteLocal(key string) {
	kv.Mutex.Lock()
	oldEntry, existed := kv.Data[key]
	delete(kv.Data, key)
	kv.Mutex.Unlock()

	// Notify watchers if key existed
	if kv.watchManager != nil && existed {
		event := watch.WatchEvent{
			Type:      watch.EventTypeDelete,
			Key:       key,
			OldValue:  oldEntry.Value,
			Timestamp: time.Now().Unix(),
		}
		kv.watchManager.Notify(event)
	}
}

// BatchDeleteLocal deletes multiple keys without persisting.
// This is used by Raft FSM when applying committed log entries.
func (kv *KVStore) BatchDeleteLocal(keys []string) error {
	kv.Mutex.Lock()

	// Track old values for watch events
	oldEntries := make(map[string]KVEntry)
	for _, key := range keys {
		if oldEntry, existed := kv.Data[key]; existed {
			oldEntries[key] = oldEntry
		}
		delete(kv.Data, key)
	}
	kv.Mutex.Unlock()

	// Notify watchers for deleted keys
	if kv.watchManager != nil {
		timestamp := time.Now().Unix()
		for key, oldEntry := range oldEntries {
			event := watch.WatchEvent{
				Type:      watch.EventTypeDelete,
				Key:       key,
				OldValue:  oldEntry.Value,
				Timestamp: timestamp,
			}
			kv.watchManager.Notify(event)
		}
	}

	return nil
}

// KVEntrySnapshot represents KV entry data for Raft snapshots.
type KVEntrySnapshot struct {
	Value       string `json:"value"`
	ModifyIndex uint64 `json:"modify_index"`
	CreateIndex uint64 `json:"create_index"`
	Flags       uint64 `json:"flags,omitempty"`
}

// GetAllData returns all KV data for Raft snapshotting.
// Returns a deep copy to ensure snapshot consistency.
func (kv *KVStore) GetAllData() map[string]KVEntrySnapshot {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()

	result := make(map[string]KVEntrySnapshot, len(kv.Data))
	for key, entry := range kv.Data {
		result[key] = KVEntrySnapshot{
			Value:       entry.Value,
			ModifyIndex: entry.ModifyIndex,
			CreateIndex: entry.CreateIndex,
			Flags:       entry.Flags,
		}
	}
	return result
}

// RestoreFromSnapshot restores KV data from a Raft snapshot.
// This replaces all existing data with the snapshot data.
func (kv *KVStore) RestoreFromSnapshot(data map[string]KVEntrySnapshot) error {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	// Clear existing data
	kv.Data = make(map[string]KVEntry, len(data))

	// Restore from snapshot
	var maxIndex uint64
	for key, snapshot := range data {
		kv.Data[key] = KVEntry{
			Value:       snapshot.Value,
			ModifyIndex: snapshot.ModifyIndex,
			CreateIndex: snapshot.CreateIndex,
			Flags:       snapshot.Flags,
		}
		if snapshot.ModifyIndex > maxIndex {
			maxIndex = snapshot.ModifyIndex
		}
	}

	// Update global index to max found index
	kv.globalIndex = maxIndex

	return nil
}
