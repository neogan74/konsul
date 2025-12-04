package dataloader

import (
	"context"

	"github.com/neogan74/konsul/internal/store"
)

// KVLoader is a DataLoader for KV store
type KVLoader struct {
	*Loader[string, string]
	kvStore *store.KVStore
}

// NewKVLoader creates a new KV DataLoader
func NewKVLoader(kvStore *store.KVStore) *KVLoader {
	loader := &KVLoader{
		kvStore: kvStore,
	}

	loader.Loader = NewLoader(func(ctx context.Context, keys []string) ([]string, []error) {
		return loader.batchFetch(ctx, keys)
	})

	return loader
}

// batchFetch fetches multiple keys at once
func (l *KVLoader) batchFetch(ctx context.Context, keys []string) ([]string, []error) {
	results := make([]string, len(keys))
	errors := make([]error, len(keys))

	for i, key := range keys {
		value, exists := l.kvStore.Get(key)
		if !exists {
			errors[i] = ErrNotFound
		} else {
			results[i] = value
		}
	}

	return results, errors
}

// ErrNotFound is returned when a key is not found
var ErrNotFound = &NotFoundError{}

// NotFoundError represents a key not found error
type NotFoundError struct{}

func (e *NotFoundError) Error() string {
	return "key not found"
}
