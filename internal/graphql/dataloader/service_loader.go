package dataloader

import (
	"context"

	"github.com/neogan74/konsul/internal/store"
)

// ServiceLoader is a DataLoader for Service store
type ServiceLoader struct {
	*Loader[string, *store.ServiceEntry]
	serviceStore *store.ServiceStore
}

// NewServiceLoader creates a new Service DataLoader
func NewServiceLoader(serviceStore *store.ServiceStore) *ServiceLoader {
	loader := &ServiceLoader{
		serviceStore: serviceStore,
	}

	loader.Loader = NewLoader(func(ctx context.Context, keys []string) ([]*store.ServiceEntry, []error) {
		return loader.batchFetch(ctx, keys)
	})

	return loader
}

// batchFetch fetches multiple services at once
func (l *ServiceLoader) batchFetch(ctx context.Context, keys []string) ([]*store.ServiceEntry, []error) {
	results := make([]*store.ServiceEntry, len(keys))
	errors := make([]error, len(keys))

	// Get all service entries at once
	allEntries := l.serviceStore.ListAll()

	// Create a map for quick lookup
	entryMap := make(map[string]*store.ServiceEntry)
	for i := range allEntries {
		entryMap[allEntries[i].Service.Name] = &allEntries[i]
	}

	// Populate results
	for i, key := range keys {
		if entry, exists := entryMap[key]; exists {
			results[i] = entry
		} else {
			errors[i] = ErrNotFound
		}
	}

	return results, errors
}
