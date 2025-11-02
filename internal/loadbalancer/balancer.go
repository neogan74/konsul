package loadbalancer

import (
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/neogan74/konsul/internal/store"
)

// Strategy defines the load balancing strategy
type Strategy string

const (
	// StrategyRoundRobin distributes requests evenly across all services
	StrategyRoundRobin Strategy = "round-robin"

	// StrategyRandom selects a random service for each request
	StrategyRandom Strategy = "random"

	// StrategyLeastConnections selects the service with the fewest active connections
	StrategyLeastConnections Strategy = "least-connections"
)

// Balancer provides load balancing capabilities for service discovery
type Balancer struct {
	store       *store.ServiceStore
	strategy    Strategy
	counters    map[string]*uint64      // Round-robin counters per service name
	connections map[string]*int32       // Active connection counters per service instance
	mutex       sync.RWMutex
}

// New creates a new load balancer with the specified strategy
func New(serviceStore *store.ServiceStore, strategy Strategy) *Balancer {
	return &Balancer{
		store:       serviceStore,
		strategy:    strategy,
		counters:    make(map[string]*uint64),
		connections: make(map[string]*int32),
		mutex:       sync.RWMutex{},
	}
}

// SelectService selects a service instance using the configured strategy
// Returns the selected service and true if successful, or an empty Service and false if no instances available
func (b *Balancer) SelectService(serviceName string) (store.Service, bool) {
	// Get all instances of the service
	services := b.store.List()
	var instances []store.Service

	for _, svc := range services {
		if svc.Name == serviceName {
			instances = append(instances, svc)
		}
	}

	if len(instances) == 0 {
		return store.Service{}, false
	}

	// Select based on strategy
	switch b.strategy {
	case StrategyRandom:
		return b.selectRandom(instances), true
	case StrategyLeastConnections:
		return b.selectLeastConnections(instances), true
	case StrategyRoundRobin:
		fallthrough
	default:
		return b.selectRoundRobin(serviceName, instances), true
	}
}

// SelectServiceByTags selects a service instance that matches all specified tags
func (b *Balancer) SelectServiceByTags(tags []string) (store.Service, bool) {
	services := b.store.QueryByTags(tags)
	if len(services) == 0 {
		return store.Service{}, false
	}

	// Apply strategy to filtered services
	switch b.strategy {
	case StrategyRandom:
		return b.selectRandom(services), true
	case StrategyLeastConnections:
		return b.selectLeastConnections(services), true
	case StrategyRoundRobin:
		fallthrough
	default:
		// For tag-based queries, use first service name for counter
		return b.selectRoundRobin(services[0].Name, services), true
	}
}

// SelectServiceByMetadata selects a service instance that matches all specified metadata
func (b *Balancer) SelectServiceByMetadata(filters map[string]string) (store.Service, bool) {
	services := b.store.QueryByMetadata(filters)
	if len(services) == 0 {
		return store.Service{}, false
	}

	// Apply strategy to filtered services
	switch b.strategy {
	case StrategyRandom:
		return b.selectRandom(services), true
	case StrategyLeastConnections:
		return b.selectLeastConnections(services), true
	case StrategyRoundRobin:
		fallthrough
	default:
		// For metadata-based queries, use first service name for counter
		return b.selectRoundRobin(services[0].Name, services), true
	}
}

// SelectServiceByQuery selects a service instance matching both tags and metadata
func (b *Balancer) SelectServiceByQuery(tags []string, metadata map[string]string) (store.Service, bool) {
	services := b.store.QueryByTagsAndMetadata(tags, metadata)
	if len(services) == 0 {
		return store.Service{}, false
	}

	// Apply strategy to filtered services
	switch b.strategy {
	case StrategyRandom:
		return b.selectRandom(services), true
	case StrategyLeastConnections:
		return b.selectLeastConnections(services), true
	case StrategyRoundRobin:
		fallthrough
	default:
		// For combined queries, use first service name for counter
		return b.selectRoundRobin(services[0].Name, services), true
	}
}

// selectRoundRobin implements round-robin selection
func (b *Balancer) selectRoundRobin(serviceName string, instances []store.Service) store.Service {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Initialize counter if needed
	if b.counters[serviceName] == nil {
		var counter uint64
		b.counters[serviceName] = &counter
	}

	// Get and increment counter atomically
	counter := atomic.AddUint64(b.counters[serviceName], 1)

	// Select instance using modulo
	index := (counter - 1) % uint64(len(instances))
	return instances[index]
}

// selectRandom implements random selection
func (b *Balancer) selectRandom(instances []store.Service) store.Service {
	index := rand.Intn(len(instances))
	return instances[index]
}

// selectLeastConnections implements least-connections selection
func (b *Balancer) selectLeastConnections(instances []store.Service) store.Service {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Find instance with minimum connections
	minIdx := 0
	minConns := int32(0)

	for i, svc := range instances {
		instanceKey := b.instanceKey(svc)

		// Initialize connection counter if needed
		if b.connections[instanceKey] == nil {
			var conns int32
			b.connections[instanceKey] = &conns
		}

		conns := atomic.LoadInt32(b.connections[instanceKey])

		if i == 0 || conns < minConns {
			minIdx = i
			minConns = conns
		}
	}

	return instances[minIdx]
}

// instanceKey generates a unique key for a service instance
func (b *Balancer) instanceKey(svc store.Service) string {
	return svc.Name + ":" + svc.Address + ":" + string(rune(svc.Port))
}

// IncrementConnections increments the connection count for a service instance
// Used for least-connections strategy tracking
func (b *Balancer) IncrementConnections(svc store.Service) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	instanceKey := b.instanceKey(svc)

	if b.connections[instanceKey] == nil {
		var conns int32
		b.connections[instanceKey] = &conns
	}

	atomic.AddInt32(b.connections[instanceKey], 1)
}

// DecrementConnections decrements the connection count for a service instance
// Used for least-connections strategy tracking
func (b *Balancer) DecrementConnections(svc store.Service) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	instanceKey := b.instanceKey(svc)

	if b.connections[instanceKey] == nil {
		return
	}

	conns := atomic.LoadInt32(b.connections[instanceKey])
	if conns > 0 {
		atomic.AddInt32(b.connections[instanceKey], -1)
	}
}

// GetStrategy returns the current load balancing strategy
func (b *Balancer) GetStrategy() Strategy {
	return b.strategy
}

// SetStrategy updates the load balancing strategy
func (b *Balancer) SetStrategy(strategy Strategy) {
	b.strategy = strategy
}
