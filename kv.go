package main

import (
	"net/http"
	"strings"
	"sync"
)

var (
	kv      = KVStore{data: make(map[string]string)}
	kvMutex sync.RWMutex
)

func kvPut(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")
	value := r.URL.Query().Get("value")

	kvMutex.Lock()
	kv.data[key] = value
	kvMutex.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func kvGet(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")

	kvMutex.RLock()
	value, ok := kv.data[key]
	kvMutex.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	if _, err := w.Write([]byte(value)); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func kvDel(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")

	kvMutex.Lock()
	delete(kv.data, key)
	kvMutex.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
