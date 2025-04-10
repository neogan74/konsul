package api

import (
	"net/http"
	"strings"
	"sync"

	"github.com/neogan74/konsul/internal/models"
)

var (
	kv      = models.KVStore{Data: make(map[string]string)}
	kvMutex sync.RWMutex
)

func KvPut(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")
	value := r.URL.Query().Get("value")

	kvMutex.Lock()
	kv.Data[key] = value
	kvMutex.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func KvGet(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")

	kvMutex.RLock()
	value, ok := kv.Data[key]
	kvMutex.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	if _, err := w.Write([]byte(value)); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func KvDel(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/kv/")

	kvMutex.Lock()
	delete(kv.Data, key)
	kvMutex.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
