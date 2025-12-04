package dataloader

import (
	"context"
	"net/http"

	"github.com/neogan74/konsul/internal/store"
)

// ContextKey is the key for storing DataLoaders in context
type ContextKey string

const (
	// KVLoaderKey is the context key for KV DataLoader
	KVLoaderKey ContextKey = "kvLoader"
	// ServiceLoaderKey is the context key for Service DataLoader
	ServiceLoaderKey ContextKey = "serviceLoader"
)

// Loaders holds all DataLoaders
type Loaders struct {
	KVLoader      *KVLoader
	ServiceLoader *ServiceLoader
}

// Middleware creates an HTTP middleware that injects DataLoaders into the request context
func Middleware(kvStore *store.KVStore, serviceStore *store.ServiceStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create new DataLoaders for each request
			// This ensures proper batching per request and prevents data leakage between requests
			loaders := &Loaders{
				KVLoader:      NewKVLoader(kvStore),
				ServiceLoader: NewServiceLoader(serviceStore),
			}

			// Store in context
			ctx := context.WithValue(r.Context(), KVLoaderKey, loaders.KVLoader)
			ctx = context.WithValue(ctx, ServiceLoaderKey, loaders.ServiceLoader)

			// Call next handler with new context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetKVLoader retrieves the KV DataLoader from context
func GetKVLoader(ctx context.Context) *KVLoader {
	loader, _ := ctx.Value(KVLoaderKey).(*KVLoader)
	return loader
}

// GetServiceLoader retrieves the Service DataLoader from context
func GetServiceLoader(ctx context.Context) *ServiceLoader {
	loader, _ := ctx.Value(ServiceLoaderKey).(*ServiceLoader)
	return loader
}
