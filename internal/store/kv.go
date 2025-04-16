package store

import "sync"

type KVStore struct {
	Data  map[string]string
	Mutex sync.RWMutex
}

func NewKVStore() *KVStore {
	return &KVStore{Data: make(map[string]string)}
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
}
func (kv *KVStore) Delete(key string) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()
	delete(kv.Data, key)
}
