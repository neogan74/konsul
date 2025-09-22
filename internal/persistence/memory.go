package persistence

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

// MemoryEngine is an in-memory implementation of Engine
type MemoryEngine struct {
	mu       sync.RWMutex
	kvData   map[string][]byte
	svcData  map[string]serviceEntry
}

type serviceEntry struct {
	Data      []byte
	ExpiresAt time.Time
}

// NewMemoryEngine creates a new in-memory persistence engine
func NewMemoryEngine() *MemoryEngine {
	return &MemoryEngine{
		kvData:  make(map[string][]byte),
		svcData: make(map[string]serviceEntry),
	}
}

func (m *MemoryEngine) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if val, ok := m.kvData[key]; ok {
		return val, nil
	}
	return nil, errors.New("key not found")
}

func (m *MemoryEngine) Set(key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.kvData[key] = value
	return nil
}

func (m *MemoryEngine) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.kvData, key)
	return nil
}

func (m *MemoryEngine) List(prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for key := range m.kvData {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (m *MemoryEngine) GetService(name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.svcData[name]; ok {
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Data, nil
		}
	}
	return nil, errors.New("service not found")
}

func (m *MemoryEngine) SetService(name string, data []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.svcData[name] = serviceEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (m *MemoryEngine) DeleteService(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.svcData, name)
	return nil
}

func (m *MemoryEngine) ListServices() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	now := time.Now()
	for name, entry := range m.svcData {
		if now.Before(entry.ExpiresAt) {
			names = append(names, name)
		}
	}
	return names, nil
}

func (m *MemoryEngine) BatchSet(items map[string][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, value := range items {
		m.kvData[key] = value
	}
	return nil
}

func (m *MemoryEngine) BatchDelete(keys []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.kvData, key)
	}
	return nil
}

func (m *MemoryEngine) Close() error {
	return nil
}

func (m *MemoryEngine) Backup(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := map[string]interface{}{
		"kv":       m.kvData,
		"services": m.svcData,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonData, 0644)
}

func (m *MemoryEngine) Restore(path string) error {
	jsonData, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing data
	m.kvData = make(map[string][]byte)
	m.svcData = make(map[string]serviceEntry)

	// Restore KV data
	if kvData, ok := data["kv"].(map[string]interface{}); ok {
		for key, value := range kvData {
			if strVal, ok := value.(string); ok {
				m.kvData[key] = []byte(strVal)
			}
		}
	}

	// Restore service data would need proper unmarshaling
	// Simplified for now

	return nil
}

func (m *MemoryEngine) BeginTx() (Transaction, error) {
	return &memoryTx{engine: m, operations: make(map[string]interface{})}, nil
}

// memoryTx implements Transaction for memory engine
type memoryTx struct {
	engine     *MemoryEngine
	operations map[string]interface{}
}

func (tx *memoryTx) Set(key string, value []byte) error {
	tx.operations[key] = value
	return nil
}

func (tx *memoryTx) Delete(key string) error {
	tx.operations[key] = nil
	return nil
}

func (tx *memoryTx) Commit() error {
	tx.engine.mu.Lock()
	defer tx.engine.mu.Unlock()

	for key, value := range tx.operations {
		if value == nil {
			delete(tx.engine.kvData, key)
		} else {
			tx.engine.kvData[key] = value.([]byte)
		}
	}
	return nil
}

func (tx *memoryTx) Rollback() error {
	tx.operations = make(map[string]interface{})
	return nil
}