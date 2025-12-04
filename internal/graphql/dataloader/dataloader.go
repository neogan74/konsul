package dataloader

import (
	"context"
	"sync"
	"time"
)

// Loader is a generic batching and caching layer
type Loader[K comparable, V any] struct {
	// BatchFn is the function that fetches multiple items at once
	BatchFn func(ctx context.Context, keys []K) ([]V, []error)

	// Wait is how long to wait before sending a batch
	Wait time.Duration

	// MaxBatch is the maximum batch size (0 = unlimited)
	MaxBatch int

	// Cache stores loaded results
	cache map[K]*result[V]
	mu    sync.RWMutex

	// Batch is the current pending batch
	batch *batch[K, V]
}

type result[V any] struct {
	data V
	err  error
}

type batch[K comparable, V any] struct {
	keys    []K
	data    map[K]*result[V]
	done    chan struct{}
	closing bool
	mu      sync.Mutex
}

// NewLoader creates a new DataLoader
func NewLoader[K comparable, V any](batchFn func(context.Context, []K) ([]V, []error)) *Loader[K, V] {
	return &Loader[K, V]{
		BatchFn:  batchFn,
		Wait:     16 * time.Millisecond, // Default: 16ms batching window
		MaxBatch: 100,                   // Default: max 100 items per batch
		cache:    make(map[K]*result[V]),
	}
}

// Load loads a single item by key
func (l *Loader[K, V]) Load(ctx context.Context, key K) (V, error) {
	return l.LoadThunk(ctx, key)()
}

// LoadThunk returns a function that will load the item
// This allows delaying the load until the result is actually needed
func (l *Loader[K, V]) LoadThunk(ctx context.Context, key K) func() (V, error) {
	l.mu.RLock()
	if cached, ok := l.cache[key]; ok {
		l.mu.RUnlock()
		return func() (V, error) {
			return cached.data, cached.err
		}
	}
	l.mu.RUnlock()

	// Get or create batch
	l.mu.Lock()
	if l.batch == nil {
		l.batch = &batch[K, V]{
			keys: []K{},
			data: make(map[K]*result[V]),
			done: make(chan struct{}),
		}
	}
	batch := l.batch
	l.mu.Unlock()

	// Add key to batch
	batch.mu.Lock()

	// Check if already in this batch
	if _, exists := batch.data[key]; exists {
		batch.mu.Unlock()
		return func() (V, error) {
			<-batch.done
			res := batch.data[key]
			return res.data, res.err
		}
	}

	// Add to batch
	batch.keys = append(batch.keys, key)
	batch.data[key] = &result[V]{} // Placeholder

	// Check if we should dispatch immediately
	shouldDispatch := len(batch.keys) >= l.MaxBatch && l.MaxBatch > 0
	batch.mu.Unlock()

	// Schedule batch dispatch
	if shouldDispatch {
		l.dispatchBatch(ctx, batch)
	} else {
		go l.scheduleBatch(ctx, batch)
	}

	return func() (V, error) {
		<-batch.done
		res := batch.data[key]
		return res.data, res.err
	}
}

// scheduleBatch schedules a batch to be dispatched after Wait duration
func (l *Loader[K, V]) scheduleBatch(ctx context.Context, batch *batch[K, V]) {
	batch.mu.Lock()
	if batch.closing {
		batch.mu.Unlock()
		return
	}
	batch.closing = true
	batch.mu.Unlock()

	time.Sleep(l.Wait)
	l.dispatchBatch(ctx, batch)
}

// dispatchBatch executes the batch function
func (l *Loader[K, V]) dispatchBatch(ctx context.Context, batch *batch[K, V]) {
	batch.mu.Lock()
	keys := batch.keys
	batch.mu.Unlock()

	// Execute batch function
	results, errors := l.BatchFn(ctx, keys)

	// Store results
	batch.mu.Lock()
	for i, key := range keys {
		var res result[V]
		if errors != nil && i < len(errors) && errors[i] != nil {
			res.err = errors[i]
		} else if i < len(results) {
			res.data = results[i]
		}
		batch.data[key] = &res

		// Cache the result
		l.mu.Lock()
		l.cache[key] = &res
		l.mu.Unlock()
	}
	batch.mu.Unlock()

	// Mark batch as done
	close(batch.done)

	// Clear current batch
	l.mu.Lock()
	if l.batch == batch {
		l.batch = nil
	}
	l.mu.Unlock()
}

// LoadMany loads multiple items at once
func (l *Loader[K, V]) LoadMany(ctx context.Context, keys []K) ([]V, []error) {
	thunks := make([]func() (V, error), len(keys))
	for i, key := range keys {
		thunks[i] = l.LoadThunk(ctx, key)
	}

	results := make([]V, len(keys))
	errors := make([]error, len(keys))
	for i, thunk := range thunks {
		results[i], errors[i] = thunk()
	}

	return results, errors
}

// Clear clears the cache
func (l *Loader[K, V]) Clear(key K) {
	l.mu.Lock()
	delete(l.cache, key)
	l.mu.Unlock()
}

// ClearAll clears all cached items
func (l *Loader[K, V]) ClearAll() {
	l.mu.Lock()
	l.cache = make(map[K]*result[V])
	l.mu.Unlock()
}
