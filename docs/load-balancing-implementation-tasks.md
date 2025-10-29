# Load Balancing Strategies - Implementation Tasks

**Based on**: ADR-0018 (Load Balancing Strategies)
**Created**: 2025-10-28
**Status**: Planning

This document breaks down the Load Balancing implementation into actionable tasks with clear acceptance criteria, dependencies, and time estimates.

---

## Phase 1: Core Framework (Week 1-2)

### 1.1 Package Structure and Interfaces

#### Task 1.1.1: Create Load Balancer Package Structure
**Priority**: P0 (Critical Path)
**Estimated Time**: 2 hours
**Dependencies**: None

**Description**:
Create the package structure for load balancing implementation.

**Acceptance Criteria**:
- [ ] Create `internal/loadbalancer/` directory
- [ ] Create `internal/loadbalancer/strategy.go` - Strategy interface
- [ ] Create `internal/loadbalancer/manager.go` - Manager implementation
- [ ] Create `internal/loadbalancer/types.go` - Common types
- [ ] Create `internal/loadbalancer/errors.go` - Error definitions
- [ ] Add package documentation

**Files to Create**:
- `internal/loadbalancer/strategy.go`
- `internal/loadbalancer/manager.go`
- `internal/loadbalancer/types.go`
- `internal/loadbalancer/errors.go`

---

#### Task 1.1.2: Define LoadBalancer Interface
**Priority**: P0
**Estimated Time**: 3 hours
**Dependencies**: 1.1.1

**Description**:
Define the core LoadBalancer interface and types.

**Acceptance Criteria**:
- [ ] Define `Strategy` type (string enum)
- [ ] Define `SelectOptions` struct
- [ ] Define `LoadBalancer` interface with Select, SelectOne, Name methods
- [ ] Add comprehensive documentation
- [ ] Define error types

**File**: `internal/loadbalancer/strategy.go`

**Code**:
```go
package loadbalancer

import (
    "github.com/neogan74/konsul/internal/store"
)

// Strategy represents a load balancing algorithm
type Strategy string

const (
    StrategyRoundRobin         Strategy = "round-robin"
    StrategyWeightedRoundRobin Strategy = "weighted-round-robin"
    StrategyRandom             Strategy = "random"
    StrategyWeightedRandom     Strategy = "weighted-random"
    StrategyLeastConnections   Strategy = "least-connections"
    StrategyIPHash             Strategy = "ip-hash"
    StrategyRingHash           Strategy = "ring-hash"
    StrategyLatencyBased       Strategy = "latency-based"
)

// SelectOptions contains parameters for load balancing selection
type SelectOptions struct {
    Strategy      Strategy
    ClientIP      string
    SessionKey    string
    ClientRegion  string
    MaxInstances  int
    HealthFilter  bool
    TagFilter     []string
    MetaFilter    map[string]string
}

// LoadBalancer interface for all load balancing strategies
type LoadBalancer interface {
    // Select returns ordered list of instances based on strategy
    Select(instances []store.Service, opts SelectOptions) []store.Service

    // SelectOne returns single best instance
    SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error)

    // Name returns the strategy name
    Name() Strategy
}
```

---

#### Task 1.1.3: Implement Manager
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.1.2

**Description**:
Create the Manager that registers and coordinates load balancers.

**Acceptance Criteria**:
- [ ] Implement Manager struct
- [ ] Implement RegisterStrategy method
- [ ] Implement GetStrategy method
- [ ] Implement Select and SelectOne methods (dispatch to strategies)
- [ ] Support default strategy
- [ ] Thread-safe registration and access
- [ ] Add logging
- [ ] Write unit tests

**File**: `internal/loadbalancer/manager.go`

**Code**:
```go
package loadbalancer

import (
    "fmt"
    "sync"

    "github.com/neogan74/konsul/internal/logger"
    "github.com/neogan74/konsul/internal/store"
)

type Manager struct {
    strategies      map[Strategy]LoadBalancer
    defaultStrategy Strategy
    log             logger.Logger
    mu              sync.RWMutex
}

func NewManager(defaultStrategy Strategy, log logger.Logger) *Manager {
    return &Manager{
        strategies:      make(map[Strategy]LoadBalancer),
        defaultStrategy: defaultStrategy,
        log:             log,
    }
}

func (m *Manager) RegisterStrategy(lb LoadBalancer) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.strategies[lb.Name()] = lb
    m.log.Info("Registered load balancing strategy",
        logger.String("strategy", string(lb.Name())))
}

func (m *Manager) GetStrategy(strategy Strategy) (LoadBalancer, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if strategy == "" {
        strategy = m.defaultStrategy
    }

    lb, ok := m.strategies[strategy]
    if !ok {
        return nil, fmt.Errorf("unknown load balancing strategy: %s", strategy)
    }

    return lb, nil
}

func (m *Manager) Select(instances []store.Service, opts SelectOptions) ([]store.Service, error) {
    lb, err := m.GetStrategy(opts.Strategy)
    if err != nil {
        return nil, err
    }

    return lb.Select(instances, opts), nil
}

func (m *Manager) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    lb, err := m.GetStrategy(opts.Strategy)
    if err != nil {
        return nil, err
    }

    return lb.SelectOne(instances, opts)
}
```

---

### 1.2 Basic Strategies

#### Task 1.2.1: Implement Round Robin
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.1.3

**Description**:
Implement Round Robin load balancing strategy.

**Acceptance Criteria**:
- [ ] Create RoundRobinBalancer struct
- [ ] Maintain per-service counter (atomic)
- [ ] Implement Select method (return all, ordered)
- [ ] Implement SelectOne method (cycle through instances)
- [ ] Thread-safe counter access
- [ ] Handle empty instance list
- [ ] Write unit tests (verify cycling behavior)

**File**: `internal/loadbalancer/round_robin.go`

**Code**:
```go
package loadbalancer

import (
    "sync"
    "sync/atomic"

    "github.com/neogan74/konsul/internal/store"
)

type RoundRobinBalancer struct {
    counters map[string]*atomic.Uint64
    mu       sync.RWMutex
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
    return &RoundRobinBalancer{
        counters: make(map[string]*atomic.Uint64),
    }
}

func (rb *RoundRobinBalancer) Name() Strategy {
    return StrategyRoundRobin
}

func (rb *RoundRobinBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    serviceName := instances[0].Name

    rb.mu.Lock()
    if rb.counters[serviceName] == nil {
        rb.counters[serviceName] = &atomic.Uint64{}
    }
    counter := rb.counters[serviceName]
    rb.mu.Unlock()

    idx := int(counter.Add(1) % uint64(len(instances)))
    return &instances[idx], nil
}

func (rb *RoundRobinBalancer) Select(instances []store.Service, opts SelectOptions) []store.Service {
    if len(instances) == 0 {
        return instances
    }

    if opts.MaxInstances > 0 && opts.MaxInstances < len(instances) {
        // Return subset starting from current position
        selected := make([]store.Service, opts.MaxInstances)
        for i := 0; i < opts.MaxInstances; i++ {
            svc, _ := rb.SelectOne(instances, opts)
            selected[i] = *svc
        }
        return selected
    }

    return instances
}
```

---

#### Task 1.2.2: Implement Random
**Priority**: P0
**Estimated Time**: 3 hours
**Dependencies**: 1.1.3

**Description**:
Implement Random load balancing strategy.

**Acceptance Criteria**:
- [ ] Create RandomBalancer struct
- [ ] Use crypto/rand or math/rand with seed
- [ ] Implement SelectOne method (random selection)
- [ ] Implement Select method (shuffle)
- [ ] Thread-safe random number generation
- [ ] Write unit tests (verify randomness distribution)

**File**: `internal/loadbalancer/random.go`

**Code**:
```go
package loadbalancer

import (
    "math/rand"
    "sync"
    "time"

    "github.com/neogan74/konsul/internal/store"
)

type RandomBalancer struct {
    rng *rand.Rand
    mu  sync.Mutex
}

func NewRandomBalancer() *RandomBalancer {
    return &RandomBalancer{
        rng: rand.New(rand.NewSource(time.Now().UnixNano())),
    }
}

func (r *RandomBalancer) Name() Strategy {
    return StrategyRandom
}

func (r *RandomBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    r.mu.Lock()
    idx := r.rng.Intn(len(instances))
    r.mu.Unlock()

    return &instances[idx], nil
}

func (r *RandomBalancer) Select(instances []store.Service, opts SelectOptions) []store.Service {
    if len(instances) == 0 {
        return instances
    }

    // Shuffle instances
    r.mu.Lock()
    shuffled := make([]store.Service, len(instances))
    copy(shuffled, instances)
    r.rng.Shuffle(len(shuffled), func(i, j int) {
        shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
    })
    r.mu.Unlock()

    if opts.MaxInstances > 0 && opts.MaxInstances < len(shuffled) {
        return shuffled[:opts.MaxInstances]
    }

    return shuffled
}
```

---

#### Task 1.2.3: Write Tests for Basic Strategies
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.2.1, 1.2.2

**Description**:
Comprehensive unit tests for Round Robin and Random strategies.

**Acceptance Criteria**:
- [ ] Test Round Robin cycling behavior
- [ ] Test Round Robin with multiple services
- [ ] Test Round Robin thread safety (concurrent access)
- [ ] Test Random distribution (Chi-square test)
- [ ] Test Random thread safety
- [ ] Test edge cases (empty, single instance)
- [ ] Benchmark selection performance
- [ ] Achieve >90% code coverage

**Files**:
- `internal/loadbalancer/round_robin_test.go`
- `internal/loadbalancer/random_test.go`

---

### 1.3 Health-Aware Wrapper

#### Task 1.3.1: Implement Health-Aware Wrapper
**Priority**: P0
**Estimated Time**: 5 hours
**Dependencies**: 1.2.3

**Description**:
Create a wrapper that filters unhealthy instances before selection.

**Acceptance Criteria**:
- [ ] Create HealthAwareBalancer struct
- [ ] Wrap any LoadBalancer
- [ ] Integrate with health check manager
- [ ] Filter instances based on health status
- [ ] Fallback to all instances if none healthy
- [ ] Add logging for health filtering
- [ ] Add metrics for filtered instances
- [ ] Write unit tests

**File**: `internal/loadbalancer/health_aware.go`

**Code**:
```go
package loadbalancer

import (
    "github.com/neogan74/konsul/internal/healthcheck"
    "github.com/neogan74/konsul/internal/logger"
    "github.com/neogan74/konsul/internal/store"
)

type HealthAwareBalancer struct {
    inner         LoadBalancer
    healthManager *healthcheck.Manager
    log           logger.Logger
}

func NewHealthAwareBalancer(inner LoadBalancer, healthManager *healthcheck.Manager, log logger.Logger) *HealthAwareBalancer {
    return &HealthAwareBalancer{
        inner:         inner,
        healthManager: healthManager,
        log:           log,
    }
}

func (ha *HealthAwareBalancer) Name() Strategy {
    return ha.inner.Name()
}

func (ha *HealthAwareBalancer) Select(instances []store.Service, opts SelectOptions) []store.Service {
    if !opts.HealthFilter {
        return ha.inner.Select(instances, opts)
    }

    healthy := ha.filterHealthy(instances)

    if len(healthy) == 0 {
        ha.log.Warn("No healthy instances found, returning all",
            logger.Int("total_instances", len(instances)))
        return ha.inner.Select(instances, opts)
    }

    ha.log.Debug("Filtered instances by health",
        logger.Int("healthy", len(healthy)),
        logger.Int("total", len(instances)))

    return ha.inner.Select(healthy, opts)
}

func (ha *HealthAwareBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if !opts.HealthFilter {
        return ha.inner.SelectOne(instances, opts)
    }

    healthy := ha.filterHealthy(instances)

    if len(healthy) == 0 {
        ha.log.Warn("No healthy instances found, selecting from all")
        return ha.inner.SelectOne(instances, opts)
    }

    return ha.inner.SelectOne(healthy, opts)
}

func (ha *HealthAwareBalancer) filterHealthy(instances []store.Service) []store.Service {
    healthy := make([]store.Service, 0, len(instances))

    for _, instance := range instances {
        if ha.isHealthy(instance) {
            healthy = append(healthy, instance)
        }
    }

    return healthy
}

func (ha *HealthAwareBalancer) isHealthy(instance store.Service) bool {
    checks := ha.healthManager.GetChecksByService(instance.Name)

    if len(checks) == 0 {
        // No checks = assume healthy
        return true
    }

    for _, check := range checks {
        if check.Status != healthcheck.StatusPassing {
            return false
        }
    }

    return true
}
```

---

## Phase 2: Advanced Strategies (Week 2-3)

### 2.1 Weighted Strategies

#### Task 2.1.1: Implement Weighted Round Robin
**Priority**: P1
**Estimated Time**: 8 hours
**Dependencies**: 1.3.1

**Description**:
Implement Weighted Round Robin using smooth weighting algorithm.

**Acceptance Criteria**:
- [ ] Create WeightedRoundRobinBalancer struct
- [ ] Extract weights from service metadata
- [ ] Implement smooth weighted round robin (NGINX algorithm)
- [ ] Handle missing/invalid weights (default to 1)
- [ ] Maintain per-service state
- [ ] Thread-safe state access
- [ ] Write unit tests (verify weight distribution)
- [ ] Add performance benchmarks

**File**: `internal/loadbalancer/weighted_round_robin.go`

**Code**:
```go
package loadbalancer

import (
    "strconv"
    "sync"

    "github.com/neogan74/konsul/internal/store"
)

type WeightedRoundRobinBalancer struct {
    states map[string]*wrrState
    mu     sync.RWMutex
}

type wrrState struct {
    currentWeights []int
    effectiveWeights []int
    mu sync.Mutex
}

func NewWeightedRoundRobinBalancer() *WeightedRoundRobinBalancer {
    return &WeightedRoundRobinBalancer{
        states: make(map[string]*wrrState),
    }
}

func (wrr *WeightedRoundRobinBalancer) Name() Strategy {
    return StrategyWeightedRoundRobin
}

func (wrr *WeightedRoundRobinBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    serviceName := instances[0].Name
    state := wrr.getOrCreateState(serviceName, instances)

    state.mu.Lock()
    defer state.mu.Unlock()

    // Smooth weighted round robin algorithm
    total := 0
    for i := range instances {
        state.currentWeights[i] += state.effectiveWeights[i]
        total += state.effectiveWeights[i]
    }

    // Select instance with highest current weight
    maxIdx := 0
    maxWeight := state.currentWeights[0]
    for i := 1; i < len(instances); i++ {
        if state.currentWeights[i] > maxWeight {
            maxIdx = i
            maxWeight = state.currentWeights[i]
        }
    }

    // Decrease selected instance's current weight
    state.currentWeights[maxIdx] -= total

    return &instances[maxIdx], nil
}

func (wrr *WeightedRoundRobinBalancer) getOrCreateState(serviceName string, instances []store.Service) *wrrState {
    wrr.mu.Lock()
    defer wrr.mu.Unlock()

    if state, ok := wrr.states[serviceName]; ok {
        return state
    }

    weights := extractWeights(instances)
    state := &wrrState{
        currentWeights:   make([]int, len(instances)),
        effectiveWeights: weights,
    }
    wrr.states[serviceName] = state

    return state
}

func extractWeights(instances []store.Service) []int {
    weights := make([]int, len(instances))
    for i, instance := range instances {
        if weightStr, ok := instance.Meta["weight"]; ok {
            if weight, err := strconv.Atoi(weightStr); err == nil && weight > 0 {
                weights[i] = weight
                continue
            }
        }
        weights[i] = 1 // Default weight
    }
    return weights
}
```

---

#### Task 2.1.2: Implement Weighted Random
**Priority**: P2
**Estimated Time**: 4 hours
**Dependencies**: 2.1.1

**Description**:
Implement Weighted Random selection.

**Acceptance Criteria**:
- [ ] Create WeightedRandomBalancer struct
- [ ] Extract weights from metadata
- [ ] Implement weighted random selection
- [ ] Handle missing/invalid weights
- [ ] Write unit tests (verify weight distribution)

**File**: `internal/loadbalancer/weighted_random.go`

---

### 2.2 Sticky Session Strategies

#### Task 2.2.1: Implement IP Hash
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 1.3.1

**Description**:
Implement IP Hash for sticky sessions based on client IP.

**Acceptance Criteria**:
- [ ] Create IPHashBalancer struct
- [ ] Hash client IP using consistent hash function
- [ ] Map hash to instance index
- [ ] Handle empty client IP
- [ ] Return error if ClientIP not provided
- [ ] Write unit tests (verify same IP → same instance)

**File**: `internal/loadbalancer/ip_hash.go`

**Code**:
```go
package loadbalancer

import (
    "hash/fnv"

    "github.com/neogan74/konsul/internal/store"
)

type IPHashBalancer struct{}

func NewIPHashBalancer() *IPHashBalancer {
    return &IPHashBalancer{}
}

func (ih *IPHashBalancer) Name() Strategy {
    return StrategyIPHash
}

func (ih *IPHashBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    if opts.ClientIP == "" {
        return nil, ErrNoClientIP
    }

    hash := fnv.New64a()
    hash.Write([]byte(opts.ClientIP))
    hashValue := hash.Sum64()

    idx := int(hashValue % uint64(len(instances)))
    return &instances[idx], nil
}

func (ih *IPHashBalancer) Select(instances []store.Service, opts SelectOptions) []store.Service {
    // Return single instance or all if no client IP
    if opts.ClientIP == "" {
        return instances
    }

    selected, err := ih.SelectOne(instances, opts)
    if err != nil {
        return instances
    }

    return []store.Service{*selected}
}
```

---

#### Task 2.2.2: Implement Ring Hash (Consistent Hashing)
**Priority**: P1
**Estimated Time**: 10 hours
**Dependencies**: 2.2.1

**Description**:
Implement Ring Hash for consistent hashing with minimal remapping.

**Acceptance Criteria**:
- [ ] Create RingHashBalancer struct
- [ ] Implement hash ring with virtual nodes
- [ ] Support configurable virtual node count (default 150)
- [ ] Binary search for hash lookup
- [ ] Handle instance addition/removal (rebuild ring)
- [ ] Require SessionKey in opts
- [ ] Write unit tests (verify consistency)
- [ ] Write tests for instance changes (minimal remapping)
- [ ] Add benchmarks

**File**: `internal/loadbalancer/ring_hash.go`

**Code**:
```go
package loadbalancer

import (
    "fmt"
    "hash/fnv"
    "sort"
    "sync"

    "github.com/neogan74/konsul/internal/store"
)

type RingHashBalancer struct {
    rings        map[string]*hashRing
    virtualNodes int
    mu           sync.RWMutex
}

type hashRing struct {
    nodes     []uint64
    instances map[uint64]int
}

func NewRingHashBalancer(virtualNodes int) *RingHashBalancer {
    if virtualNodes <= 0 {
        virtualNodes = 150
    }

    return &RingHashBalancer{
        rings:        make(map[string]*hashRing),
        virtualNodes: virtualNodes,
    }
}

func (rh *RingHashBalancer) Name() Strategy {
    return StrategyRingHash
}

func (rh *RingHashBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    if opts.SessionKey == "" {
        return nil, ErrNoSessionKey
    }

    serviceName := instances[0].Name
    ring := rh.getOrCreateRing(serviceName, instances)

    // Hash the session key
    hash := fnv.New64a()
    hash.Write([]byte(opts.SessionKey))
    hashValue := hash.Sum64()

    // Binary search for closest node
    idx := sort.Search(len(ring.nodes), func(i int) bool {
        return ring.nodes[i] >= hashValue
    })

    if idx == len(ring.nodes) {
        idx = 0 // Wrap around
    }

    instanceIdx := ring.instances[ring.nodes[idx]]
    return &instances[instanceIdx], nil
}

func (rh *RingHashBalancer) getOrCreateRing(serviceName string, instances []store.Service) *hashRing {
    rh.mu.Lock()
    defer rh.mu.Unlock()

    if ring, ok := rh.rings[serviceName]; ok {
        return ring
    }

    ring := rh.buildRing(instances)
    rh.rings[serviceName] = ring
    return ring
}

func (rh *RingHashBalancer) buildRing(instances []store.Service) *hashRing {
    ring := &hashRing{
        nodes:     make([]uint64, 0, len(instances)*rh.virtualNodes),
        instances: make(map[uint64]int),
    }

    for i, instance := range instances {
        for v := 0; v < rh.virtualNodes; v++ {
            key := fmt.Sprintf("%s-%s:%d-v%d", instance.Name, instance.Address, instance.Port, v)
            hash := fnv.New64a()
            hash.Write([]byte(key))
            hashValue := hash.Sum64()

            ring.nodes = append(ring.nodes, hashValue)
            ring.instances[hashValue] = i
        }
    }

    sort.Slice(ring.nodes, func(i, j int) bool {
        return ring.nodes[i] < ring.nodes[j]
    })

    return ring
}
```

---

### 2.3 Connection-Aware Strategies

#### Task 2.3.1: Implement Least Connections
**Priority**: P1
**Estimated Time**: 8 hours
**Dependencies**: 1.3.1

**Description**:
Implement Least Connections strategy with connection tracking.

**Acceptance Criteria**:
- [ ] Create LeastConnectionsBalancer struct
- [ ] Track active connections per instance
- [ ] Implement IncrementConnections method
- [ ] Implement DecrementConnections method
- [ ] Select instance with fewest connections
- [ ] Thread-safe connection counting
- [ ] Add connection timeout/cleanup
- [ ] Write unit tests
- [ ] Add integration tests

**File**: `internal/loadbalancer/least_connections.go`

**Code**:
```go
package loadbalancer

import (
    "math"
    "sync"
    "sync/atomic"

    "github.com/neogan74/konsul/internal/store"
)

type LeastConnectionsBalancer struct {
    connections map[string]map[string]*atomic.Int64
    mu          sync.RWMutex
}

func NewLeastConnectionsBalancer() *LeastConnectionsBalancer {
    return &LeastConnectionsBalancer{
        connections: make(map[string]map[string]*atomic.Int64),
    }
}

func (lc *LeastConnectionsBalancer) Name() Strategy {
    return StrategyLeastConnections
}

func (lc *LeastConnectionsBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    minConns := int64(math.MaxInt64)
    var selected *store.Service

    for i := range instances {
        conns := lc.getConnectionCount(instances[i].Name, instances[i].Address)
        if conns < minConns {
            minConns = conns
            selected = &instances[i]
        }
    }

    return selected, nil
}

func (lc *LeastConnectionsBalancer) getConnectionCount(service, address string) int64 {
    lc.mu.RLock()
    defer lc.mu.RUnlock()

    if serviceConns, ok := lc.connections[service]; ok {
        if counter, ok := serviceConns[address]; ok {
            return counter.Load()
        }
    }

    return 0
}

func (lc *LeastConnectionsBalancer) IncrementConnections(service, address string) {
    lc.mu.Lock()
    if lc.connections[service] == nil {
        lc.connections[service] = make(map[string]*atomic.Int64)
    }
    if lc.connections[service][address] == nil {
        lc.connections[service][address] = &atomic.Int64{}
    }
    counter := lc.connections[service][address]
    lc.mu.Unlock()

    counter.Add(1)
}

func (lc *LeastConnectionsBalancer) DecrementConnections(service, address string) {
    lc.mu.RLock()
    defer lc.mu.RUnlock()

    if serviceConns, ok := lc.connections[service]; ok {
        if counter, ok := serviceConns[address]; ok {
            counter.Add(-1)
        }
    }
}
```

---

### 2.4 Geographic Strategies

#### Task 2.4.1: Implement Latency-Based Strategy
**Priority**: P2
**Estimated Time**: 8 hours
**Dependencies**: 1.3.1

**Description**:
Implement Latency-Based routing for geographic optimization.

**Acceptance Criteria**:
- [ ] Create LatencyBasedBalancer struct
- [ ] Load latency map from configuration
- [ ] Extract region from service tags
- [ ] Calculate latency from client region to service region
- [ ] Select instance with lowest latency
- [ ] Fallback to random if no region info
- [ ] Support custom latency function
- [ ] Write unit tests

**File**: `internal/loadbalancer/latency_based.go`

---

## Phase 3: API Integration (Week 3)

### 3.1 HTTP API Integration

#### Task 3.1.1: Update Catalog Handler
**Priority**: P0
**Estimated Time**: 6 hours
**Dependencies**: Phase 2 complete

**Description**:
Integrate load balancing into catalog service queries.

**Acceptance Criteria**:
- [ ] Add LoadBalancer Manager to CatalogHandler
- [ ] Parse `lb` query parameter (strategy)
- [ ] Parse `limit` query parameter (max instances)
- [ ] Parse `client-ip` query parameter
- [ ] Parse `session-key` query parameter
- [ ] Parse `client-region` query parameter
- [ ] Parse `health-filter` query parameter (boolean)
- [ ] Call load balancer with options
- [ ] Return ordered instances
- [ ] Add error handling
- [ ] Update handler tests

**File**: `internal/handlers/catalog.go`

**Code**:
```go
func (h *CatalogHandler) QueryServices(c *fiber.Ctx) error {
    log := middleware.GetLogger(c)

    // Parse filters
    serviceName := c.Query("name")
    tags := c.Queries()["tag"]
    // ... metadata filters

    // Parse load balancing options
    lbStrategy := loadbalancer.Strategy(c.Query("lb", string(loadbalancer.StrategyRoundRobin)))
    maxInstances, _ := strconv.Atoi(c.Query("limit", "0"))
    clientIP := c.Query("client-ip")
    sessionKey := c.Query("session-key")
    clientRegion := c.Query("client-region")
    healthFilter, _ := strconv.ParseBool(c.Query("health-filter", "true"))

    // Get instances
    var instances []store.Service
    if serviceName != "" {
        if svc, ok := h.store.Get(serviceName); ok {
            instances = []store.Service{svc}
        }
    } else if len(tags) > 0 {
        instances = h.store.QueryByTags(tags)
    } else {
        instances = h.store.List()
    }

    if len(instances) == 0 {
        return c.JSON([]store.Service{})
    }

    // Apply load balancing
    opts := loadbalancer.SelectOptions{
        Strategy:     lbStrategy,
        ClientIP:     clientIP,
        SessionKey:   sessionKey,
        ClientRegion: clientRegion,
        MaxInstances: maxInstances,
        HealthFilter: healthFilter,
    }

    selected, err := h.lbManager.Select(instances, opts)
    if err != nil {
        log.Error("Load balancing failed",
            logger.String("strategy", string(lbStrategy)),
            logger.Error(err))
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    log.Info("Load balanced service query",
        logger.String("strategy", string(lbStrategy)),
        logger.Int("total_instances", len(instances)),
        logger.Int("selected_instances", len(selected)))

    return c.JSON(selected)
}
```

---

#### Task 3.1.2: Add Load Balancer to Main
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 3.1.1

**Description**:
Initialize load balancer manager in main application.

**Acceptance Criteria**:
- [ ] Create LoadBalancer Manager in main
- [ ] Register all enabled strategies
- [ ] Pass manager to catalog handler
- [ ] Add configuration for default strategy
- [ ] Add configuration for enabled strategies
- [ ] Add logging

**File**: `cmd/konsul/main.go`

**Code**:
```go
// Initialize load balancer
lbManager := loadbalancer.NewManager(
    loadbalancer.Strategy(cfg.LoadBalancer.DefaultStrategy),
    logger,
)

// Register strategies
if cfg.LoadBalancer.RoundRobin.Enabled {
    lbManager.RegisterStrategy(loadbalancer.NewRoundRobinBalancer())
}

if cfg.LoadBalancer.Random.Enabled {
    lbManager.RegisterStrategy(loadbalancer.NewRandomBalancer())
}

if cfg.LoadBalancer.WeightedRoundRobin.Enabled {
    lbManager.RegisterStrategy(loadbalancer.NewWeightedRoundRobinBalancer())
}

if cfg.LoadBalancer.IPHash.Enabled {
    lbManager.RegisterStrategy(loadbalancer.NewIPHashBalancer())
}

if cfg.LoadBalancer.RingHash.Enabled {
    lbManager.RegisterStrategy(
        loadbalancer.NewRingHashBalancer(cfg.LoadBalancer.RingHash.VirtualNodes),
    )
}

if cfg.LoadBalancer.LeastConnections.Enabled {
    lbManager.RegisterStrategy(loadbalancer.NewLeastConnectionsBalancer())
}

// Wrap with health awareness if enabled
if cfg.LoadBalancer.HealthFilterEnabled {
    // Wrap each strategy with health-aware wrapper
    // ...
}

// Create catalog handler with load balancer
catalogHandler := handlers.NewCatalogHandler(serviceStore, lbManager)
```

---

### 3.2 DNS Integration

#### Task 3.2.1: Update DNS Handler for Load Balancing
**Priority**: P1
**Estimated Time**: 6 hours
**Dependencies**: 3.1.2

**Description**:
Apply load balancing to DNS responses (order A records).

**Acceptance Criteria**:
- [ ] Add LoadBalancer Manager to DNS handler
- [ ] Use default strategy (Round Robin) for DNS
- [ ] Order A/AAAA records based on selection
- [ ] Support health filtering in DNS
- [ ] Add configuration for DNS load balancing
- [ ] Write integration tests

**File**: `internal/dns/handler.go`

---

### 3.3 Configuration

#### Task 3.3.1: Add Load Balancer Configuration
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 3.1.1

**Description**:
Add configuration structure for load balancing.

**Acceptance Criteria**:
- [ ] Add LoadBalancerConfig to config
- [ ] Add DefaultStrategy field
- [ ] Add HealthFilterEnabled field
- [ ] Add per-strategy enable/disable
- [ ] Add strategy-specific settings (virtual nodes, etc.)
- [ ] Add validation
- [ ] Add environment variable mappings
- [ ] Update config tests

**File**: `internal/config/config.go`

**Code**:
```go
type LoadBalancerConfig struct {
    DefaultStrategy      string `mapstructure:"default_strategy"`
    HealthFilterEnabled  bool   `mapstructure:"health_filter_enabled"`

    RoundRobin struct {
        Enabled bool `mapstructure:"enabled"`
    } `mapstructure:"round_robin"`

    WeightedRoundRobin struct {
        Enabled bool `mapstructure:"enabled"`
    } `mapstructure:"weighted_round_robin"`

    Random struct {
        Enabled bool `mapstructure:"enabled"`
    } `mapstructure:"random"`

    IPHash struct {
        Enabled bool `mapstructure:"enabled"`
    } `mapstructure:"ip_hash"`

    RingHash struct {
        Enabled      bool `mapstructure:"enabled"`
        VirtualNodes int  `mapstructure:"virtual_nodes"`
    } `mapstructure:"ring_hash"`

    LeastConnections struct {
        Enabled bool `mapstructure:"enabled"`
    } `mapstructure:"least_connections"`

    LatencyBased struct {
        Enabled      bool              `mapstructure:"enabled"`
        LatencyMap   map[string]map[string]string `mapstructure:"latency_map"`
    } `mapstructure:"latency_based"`
}
```

---

## Phase 4: Metrics & Observability (Week 4)

### 4.1 Metrics

#### Task 4.1.1: Add Load Balancing Metrics
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: Phase 3 complete

**Description**:
Add Prometheus metrics for load balancing.

**Acceptance Criteria**:
- [ ] Add `konsul_lb_selections_total{service, strategy, result}` counter
- [ ] Add `konsul_lb_selection_duration_seconds{service, strategy}` histogram
- [ ] Add `konsul_lb_instance_connections{service, instance}` gauge
- [ ] Add `konsul_lb_health_filtered_total{service}` counter
- [ ] Instrument all strategies
- [ ] Document metrics

**File**: `internal/metrics/metrics.go`

---

#### Task 4.1.2: Add Metrics Middleware
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 4.1.1

**Description**:
Add middleware to track load balancing metrics.

**Acceptance Criteria**:
- [ ] Create LoadBalancerMetrics wrapper
- [ ] Track selection duration
- [ ] Track selection results (success/error)
- [ ] Track filtered instances
- [ ] Write tests

**File**: `internal/loadbalancer/metrics.go`

---

### 4.2 Documentation

#### Task 4.2.1: Write Load Balancing User Guide
**Priority**: P1
**Estimated Time**: 8 hours
**Dependencies**: Phase 3 complete

**Description**:
Comprehensive guide for load balancing strategies.

**Acceptance Criteria**:
- [ ] Create `docs/load-balancing.md`
- [ ] Section: Overview and concepts
- [ ] Section: Strategy descriptions and use cases
- [ ] Section: Configuration guide
- [ ] Section: API examples (HTTP and DNS)
- [ ] Section: Performance considerations
- [ ] Section: Best practices
- [ ] Section: Troubleshooting

**File**: `docs/load-balancing.md`

---

#### Task 4.2.2: Create Strategy Comparison Guide
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 4.2.1

**Description**:
Create a comparison matrix for strategies.

**Acceptance Criteria**:
- [ ] Create comparison table
- [ ] Include: Use case, pros/cons, performance, state requirements
- [ ] Decision tree for selecting strategy
- [ ] Add examples for each use case

**File**: `docs/load-balancing-strategy-comparison.md`

---

## Phase 5: State Management & HA (Week 4-5)

### 5.1 Raft Integration

#### Task 5.1.1: Replicate Load Balancer State via Raft
**Priority**: P2
**Estimated Time**: 12 hours
**Dependencies**: Phase 4 complete, Raft implementation (ADR-0011)

**Description**:
Integrate load balancer state with Raft for HA.

**Acceptance Criteria**:
- [ ] Replicate round-robin counters
- [ ] Replicate connection counts
- [ ] Replicate weighted RR state
- [ ] Replicate hash ring state
- [ ] Handle state synchronization on follower
- [ ] Handle state recovery after leader election
- [ ] Write integration tests

**File**: `internal/loadbalancer/raft_integration.go`

---

## Summary

### Total Time Estimate

- **Phase 1**: Core Framework - 30 hours (~1 week)
- **Phase 2**: Advanced Strategies - 42 hours (~1.5 weeks)
- **Phase 3**: API Integration - 20 hours (~3 days)
- **Phase 4**: Metrics & Docs - 19 hours (~2-3 days)
- **Phase 5**: State Management - 12 hours (~1-2 days)

**Total**: ~123 hours (~3-4 weeks for a single developer)

### Critical Path

```
Phase 1 (Core) → Phase 2 (Advanced) → Phase 3 (API) → Phase 4 (Observability) → Phase 5 (HA)
```

### Priorities

- **P0** (Must Have): Phase 1, Phase 3 core tasks
- **P1** (Should Have): Phase 2 (except Weighted Random), Phase 4
- **P2** (Nice to Have): Weighted Random, Latency-Based, Phase 5

### MVP Quick Start

For minimum viable product, focus on:
1. Phase 1 - Core framework + Round Robin + Random + Health-Aware
2. Phase 3 - API integration (HTTP only)
3. Basic documentation

**MVP Time**: ~40 hours (~1 week)

### Dependencies Graph

```
1.1.1 → 1.1.2 → 1.1.3 → 1.2.1, 1.2.2 → 1.2.3 → 1.3.1
                              ↓
                          Phase 2
                              ↓
                          Phase 3
                              ↓
                          Phase 4
                              ↓
                          Phase 5
```

### Testing Checklist

- [ ] Unit tests for each strategy
- [ ] Thread safety tests (concurrent access)
- [ ] Integration tests with HTTP API
- [ ] Integration tests with DNS
- [ ] Performance benchmarks
- [ ] Load tests (1000+ instances)
- [ ] Distribution tests (verify even/weighted distribution)
- [ ] Health filtering tests
- [ ] State persistence tests (with Raft)

### Performance Targets

- Selection latency: <5ms for most strategies
- Throughput: 10,000+ selections/second
- Memory per service: <1KB overhead
- Support 1000+ instances per service
- Concurrent requests: 1000+

### Next Steps

1. **Review this plan** with the team
2. **Create GitHub issues** for each task
3. **Set up project board** to track progress
4. **Assign Phase 1 tasks** to developers
5. **Start with Task 1.1.1** - Create package structure
