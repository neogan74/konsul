# ADR-0018: Load Balancing Strategies

**Date**: 2025-10-28

**Status**: Accepted

**Implementation Date**: 2025-11-24

**Deciders**: Konsul Core Team

**Tags**: load-balancing, service-discovery, performance, high-availability

## Context

Currently, Konsul returns service instances in an unspecified order when clients query for services. This creates several limitations:

### Current Limitations

1. **No load distribution**: Clients always get the same order of instances
2. **Uneven traffic distribution**: First instance receives most traffic
3. **No health-aware routing**: Unhealthy instances returned alongside healthy ones
4. **No latency optimization**: No consideration for geographic proximity
5. **No weighted routing**: Cannot favor certain instances (e.g., larger machines)
6. **No sticky sessions**: Cannot maintain session affinity
7. **Poor resource utilization**: Some instances idle while others overloaded
8. **No failure handling**: No automatic failover to backup instances

### Requirements

**Functional Requirements**:
- Support multiple load balancing algorithms
- Return service instances ordered by selected strategy
- Support health-aware filtering
- Support weighted instance selection
- Support client-side and server-side load balancing
- Allow per-service load balancing configuration
- Support sticky sessions (session affinity)
- DNS integration for simple load balancing

**Non-Functional Requirements**:
- Selection latency <5ms for most strategies
- Support 1000+ instances per service
- Minimal memory overhead
- Thread-safe selection
- Support dynamic instance list changes
- Metrics for load balancing decisions

### Load Balancing Strategies Needed

1. **Round Robin** - Distribute requests evenly across instances
2. **Weighted Round Robin** - Favor instances with higher weights
3. **Random** - Random instance selection
4. **Weighted Random** - Random with weight consideration
5. **Least Connections** - Route to instance with fewest connections
6. **IP Hash** - Consistent routing based on client IP (sticky sessions)
7. **Ring Hash (Consistent Hashing)** - Distributed consistent hashing
8. **Latency-Based** - Route to geographically nearest instance
9. **Least Load** - Route to instance with lowest CPU/memory usage
10. **Health-Aware** - Only return healthy instances

### Use Cases

**Use Case 1: Even Distribution**
- Web application with identical backend instances
- Use Round Robin to distribute load evenly

**Use Case 2: Heterogeneous Fleet**
- Mix of t3.medium and t3.xlarge instances
- Use Weighted Round Robin (xlarge gets 4x weight)

**Use Case 3: Shopping Cart (Sticky Sessions)**
- User session stored in instance memory
- Use IP Hash to route user to same instance

**Use Case 4: Microservices (Consistent Hashing)**
- Service mesh with many instances
- Use Ring Hash for consistent shard routing

**Use Case 5: Geographic Optimization**
- Multi-region deployment
- Use Latency-Based to route to nearest datacenter

**Use Case 6: Connection Pooling**
- Database proxy or connection-heavy service
- Use Least Connections to balance connection load

## Decision

We will implement a **pluggable load balancing framework** with support for multiple strategies, configurable per service and per query.

### Architecture

```
┌─────────────────────────────────────────────────────┐
│              Load Balancer Framework                │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │          LoadBalancer Interface              │  │
│  │  - Select(instances, opts) []Instance        │  │
│  │  - SelectOne(instances, opts) Instance       │  │
│  └──────────────────────────────────────────────┘  │
│                       ▲                             │
│                       │                             │
│    ┌──────────────────┴────────────────────┐       │
│    │                                         │       │
│  ┌─┴──────────┐  ┌──────────────┐  ┌───────┴────┐ │
│  │ RoundRobin │  │ WeightedRR   │  │  Random    │ │
│  └────────────┘  └──────────────┘  └────────────┘ │
│                                                     │
│  ┌────────────┐  ┌──────────────┐  ┌────────────┐ │
│  │  IPHash    │  │  RingHash    │  │ Latency    │ │
│  └────────────┘  └──────────────┘  └────────────┘ │
│                                                     │
│  ┌────────────┐  ┌──────────────┐                  │
│  │LeastConn   │  │HealthAware   │                  │
│  └────────────┘  └──────────────┘                  │
│                                                     │
└─────────────────────────────────────────────────────┘
                        │
                        ▼
              ┌──────────────────┐
              │  Service Query   │
              │   (DNS/HTTP)     │
              └──────────────────┘
```

### Load Balancer Interface

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
    StrategyLeastLoad          Strategy = "least-load"
)

// SelectOptions contains parameters for load balancing selection
type SelectOptions struct {
    Strategy      Strategy          // Load balancing strategy
    ClientIP      string           // Client IP for IP hash
    SessionKey    string           // Session key for consistent hashing
    ClientRegion  string           // Client region for latency-based
    MaxInstances  int              // Maximum instances to return
    HealthFilter  bool             // Only return healthy instances
    TagFilter     []string         // Filter by tags
    MetaFilter    map[string]string // Filter by metadata
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

// Manager manages load balancers and selection
type Manager struct {
    strategies map[Strategy]LoadBalancer
    default    Strategy
}
```

### Strategy Implementations

#### 1. Round Robin

**Behavior**: Cycle through instances sequentially

**State**: Maintains counter per service

**Algorithm**:
```go
type RoundRobinBalancer struct {
    counters map[string]*atomic.Uint64 // service -> counter
    mu       sync.RWMutex
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
```

**Pros**:
- Simple, predictable
- Even distribution
- No state needed between requests

**Cons**:
- No weight support
- Doesn't consider instance health/load

---

#### 2. Weighted Round Robin

**Behavior**: More traffic to higher-weighted instances

**Weights**: From service metadata: `{"weight": "10"}`

**Algorithm**:
```go
type WeightedRoundRobinBalancer struct {
    counters map[string]*weightedCounter
    mu       sync.RWMutex
}

type weightedCounter struct {
    current []int // Current weight for each instance
    gcd     int   // GCD of all weights
    max     int   // Max weight
    index   int   // Current index
}

func (wrr *WeightedRoundRobinBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    // Smooth weighted round robin algorithm (NGINX-style)
    // See: https://github.com/phusion/nginx/commit/27e94984486058d73157038f7950a0a36ecc6e35

    weights := extractWeights(instances) // From metadata
    total := sum(weights)

    // Add weight to current weight
    for i := range instances {
        weights[i] += extractWeight(instances[i])
    }

    // Select instance with highest current weight
    maxIdx := indexOfMax(weights)

    // Decrease selected instance's current weight
    weights[maxIdx] -= total

    return &instances[maxIdx], nil
}
```

**Pros**:
- Handles heterogeneous instance sizes
- Smooth distribution over time
- Widely used (NGINX, HAProxy)

**Cons**:
- Requires weight metadata
- More complex state management

---

#### 3. Random

**Behavior**: Random instance selection

**Algorithm**:
```go
type RandomBalancer struct {
    rng *rand.Rand
    mu  sync.Mutex
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
```

**Pros**:
- Simple, stateless
- Good for large instance counts
- No coordination needed

**Cons**:
- Not perfectly even distribution
- No determinism

---

#### 4. IP Hash (Sticky Sessions)

**Behavior**: Consistent routing based on client IP

**Algorithm**:
```go
type IPHashBalancer struct{}

func (ih *IPHashBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    if opts.ClientIP == "" {
        return nil, ErrNoClientIP
    }

    // Hash client IP
    hash := fnv.New64a()
    hash.Write([]byte(opts.ClientIP))
    hashValue := hash.Sum64()

    idx := int(hashValue % uint64(len(instances)))
    return &instances[idx], nil
}
```

**Pros**:
- Simple sticky sessions
- No server-side state
- Works with DNS

**Cons**:
- Uneven distribution with few clients
- IP changes break session

---

#### 5. Ring Hash (Consistent Hashing)

**Behavior**: Consistent hashing for distributed systems

**Algorithm**:
```go
type RingHashBalancer struct {
    rings map[string]*hashRing
    mu    sync.RWMutex
}

type hashRing struct {
    nodes      []uint64           // Sorted hash values
    instances  map[uint64]int     // Hash -> instance index
    replicas   int                // Virtual nodes per instance
}

func (rh *RingHashBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    serviceName := instances[0].Name

    ring := rh.getRing(serviceName, instances)

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
```

**Pros**:
- Minimal remapping on instance changes
- Great for sharding
- Used by Memcached, Cassandra

**Cons**:
- More complex
- Requires session key

---

#### 6. Least Connections

**Behavior**: Route to instance with fewest active connections

**State**: Connection counter per instance

**Algorithm**:
```go
type LeastConnectionsBalancer struct {
    connections map[string]map[string]*atomic.Int64 // service -> address -> count
    mu          sync.RWMutex
}

func (lc *LeastConnectionsBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if len(instances) == 0 {
        return nil, ErrNoInstances
    }

    lc.mu.RLock()
    defer lc.mu.RUnlock()

    minConns := int64(math.MaxInt64)
    var selected *store.Service

    for i := range instances {
        addr := instances[i].Address
        conns := lc.getConnectionCount(instances[i].Name, addr)
        if conns < minConns {
            minConns = conns
            selected = &instances[i]
        }
    }

    return selected, nil
}

func (lc *LeastConnectionsBalancer) IncrementConnections(service, address string) {
    // Called when connection established
}

func (lc *LeastConnectionsBalancer) DecrementConnections(service, address string) {
    // Called when connection closed
}
```

**Pros**:
- Great for long-lived connections
- Adapts to load

**Cons**:
- Requires connection tracking
- Only works with server-side LB

---

#### 7. Latency-Based

**Behavior**: Route to geographically nearest instance

**Uses**: Service tags: `region:us-east-1`, `az:us-east-1a`

**Algorithm**:
```go
type LatencyBasedBalancer struct {
    latencyMap map[string]map[string]time.Duration // clientRegion -> serviceRegion -> latency
}

func (lb *LatencyBasedBalancer) SelectOne(instances []store.Service, opts SelectOptions) (*store.Service, error) {
    if opts.ClientRegion == "" {
        // Fallback to random
        return selectRandom(instances)
    }

    minLatency := time.Duration(math.MaxInt64)
    var selected *store.Service

    for i := range instances {
        region := extractRegionFromTags(instances[i].Tags)
        latency := lb.getLatency(opts.ClientRegion, region)

        if latency < minLatency {
            minLatency = latency
            selected = &instances[i]
        }
    }

    return selected, nil
}
```

**Pros**:
- Optimizes for user experience
- Reduces latency
- Multi-region aware

**Cons**:
- Requires latency data/config
- May not balance load evenly

---

#### 8. Health-Aware Wrapper

**Behavior**: Filter unhealthy instances before selection

**Algorithm**:
```go
type HealthAwareBalancer struct {
    inner         LoadBalancer
    healthManager *healthcheck.Manager
}

func (ha *HealthAwareBalancer) Select(instances []store.Service, opts SelectOptions) []store.Service {
    if !opts.HealthFilter {
        return ha.inner.Select(instances, opts)
    }

    // Filter to only healthy instances
    healthy := make([]store.Service, 0, len(instances))
    for _, instance := range instances {
        if ha.isHealthy(instance) {
            healthy = append(healthy, instance)
        }
    }

    if len(healthy) == 0 {
        // Fallback to all instances if none healthy
        return ha.inner.Select(instances, opts)
    }

    return ha.inner.Select(healthy, opts)
}

func (ha *HealthAwareBalancer) isHealthy(instance store.Service) bool {
    checks := ha.healthManager.GetChecksByService(instance.Name)
    for _, check := range checks {
        if check.Status != healthcheck.StatusPassing {
            return false
        }
    }
    return true
}
```

---

### API Integration

#### HTTP API

**Query with Load Balancing**:
```http
GET /v1/catalog/services?name=api-service&lb=round-robin&limit=1
GET /v1/catalog/services?name=api-service&lb=weighted-round-robin&limit=3
GET /v1/catalog/services?name=api-service&lb=ip-hash&client-ip=10.0.1.50
GET /v1/catalog/services?name=api-service&lb=ring-hash&session-key=user123
GET /v1/catalog/services?name=api-service&lb=latency-based&client-region=us-east-1
```

**Service-Level Configuration**:
```json
{
  "name": "api-service",
  "address": "10.0.1.50",
  "port": 8080,
  "meta": {
    "lb-strategy": "weighted-round-robin",
    "weight": "10"
  }
}
```

#### DNS Integration

**DNS Load Balancing**:
- Default: Round Robin (DNS standard)
- Return multiple A records, let client choose
- Order based on strategy

```bash
# Round robin DNS response
dig @localhost -p 8600 api-service.service.konsul

; ANSWER SECTION:
api-service.service.konsul. 0 IN A 10.0.1.1
api-service.service.konsul. 0 IN A 10.0.1.2
api-service.service.konsul. 0 IN A 10.0.1.3
```

#### Configuration

**Global Default**:
```yaml
load_balancer:
  default_strategy: round-robin
  health_filter_enabled: true
  connection_tracking_enabled: true

  # Strategy-specific settings
  round_robin:
    enabled: true

  weighted_round_robin:
    enabled: true
    default_weight: 1

  ring_hash:
    enabled: true
    virtual_nodes: 150

  latency_based:
    enabled: true
    latency_map_file: /etc/konsul/latency-map.yaml
```

**Per-Service Override**:
```json
{
  "name": "database-proxy",
  "meta": {
    "lb-strategy": "least-connections"
  }
}
```

### Metrics

```
konsul_lb_selections_total{service, strategy, result}
konsul_lb_selection_duration_seconds{service, strategy}
konsul_lb_instance_connections{service, instance}
konsul_lb_instance_weights{service, instance}
konsul_lb_health_filtered_total{service}
```

## Alternatives Considered

### Alternative 1: Client-Side Only Load Balancing

- **Pros**:
  - No server-side state
  - Simpler implementation
  - Lower server load
- **Cons**:
  - Clients must implement logic
  - No connection tracking possible
  - Limited strategies (no least-connections)
- **Reason for rejection**: Server-side provides more features and flexibility

### Alternative 2: External Load Balancer (HAProxy/NGINX)

- **Pros**:
  - Battle-tested
  - Feature-rich
  - High performance
- **Cons**:
  - Extra infrastructure component
  - Additional configuration
  - Not integrated with service discovery
- **Reason for rejection**: Integrated solution provides better UX

### Alternative 3: DNS Round Robin Only

- **Pros**:
  - Simple, standard
  - Works everywhere
  - No implementation needed
- **Cons**:
  - Limited strategies
  - No health awareness
  - Client DNS caching issues
- **Reason for rejection**: Too limited for production needs

### Alternative 4: All Strategies Always Available

- **Pros**:
  - Maximum flexibility
  - Every query can specify strategy
- **Cons**:
  - State management for all strategies
  - Memory overhead
  - Unused strategies waste resources
- **Reason for rejection**: Configurable enables/disables more efficient

### Alternative 5: Load Balancer as Separate Service

- **Pros**:
  - Separation of concerns
  - Can scale independently
  - Multiple load balancer instances
- **Cons**:
  - Extra network hop
  - More complex deployment
  - Added latency
- **Reason for rejection**: Integrated approach simpler for most users

### Alternative 6: Do Nothing (Let Clients Handle It)

- **Pros**:
  - No implementation work
  - No added complexity
- **Cons**:
  - Poor user experience
  - Inconsistent behavior across clients
  - Not competitive with other solutions
- **Reason for rejection**: Feature essential for production deployments

## Consequences

### Positive

- **Better load distribution**: Even traffic across instances
- **Flexible strategies**: Choose algorithm per use case
- **Sticky sessions**: Support for stateful applications
- **Health-aware routing**: Automatic failover from unhealthy instances
- **Weighted routing**: Optimize for heterogeneous fleets
- **Geographic optimization**: Lower latency for users
- **Connection awareness**: Balance long-lived connections
- **Consistent hashing**: Better sharding for distributed systems
- **Metrics and observability**: Track load balancing decisions
- **API and DNS integration**: Works with existing interfaces
- **Configurable**: Per-service and global configuration

### Negative

- **State management**: Some strategies need server state
- **Memory overhead**: Counters, connection tracking, hash rings
- **Complexity**: Multiple strategies to implement and test
- **Configuration burden**: Users must understand strategies
- **Coordination**: State must be consistent across Konsul nodes
- **Performance impact**: Selection adds latency (5-10ms)
- **Raft integration needed**: State replication for HA
- **DNS limitations**: Limited strategies work with DNS
- **Testing complexity**: Each strategy needs thorough testing

### Neutral

- Connection tracking requires instrumentation
- Some strategies require metadata (weights, regions)
- Raft clustering needed for shared state
- Latency-based requires latency configuration

## Implementation Notes

### Phase 1: Core Framework (Week 1-2)

**Tasks**:
1. Define LoadBalancer interface
2. Create Manager for strategy registration
3. Implement Round Robin
4. Implement Random
5. Implement Health-Aware wrapper
6. Add unit tests

### Phase 2: Advanced Strategies (Week 2-3)

**Tasks**:
1. Implement Weighted Round Robin
2. Implement IP Hash
3. Implement Ring Hash (Consistent Hashing)
4. Implement Least Connections
5. Implement Latency-Based
6. Add integration tests

### Phase 3: API Integration (Week 3)

**Tasks**:
1. Add load balancing to catalog handler
2. Add query parameters (lb, limit, client-ip, etc.)
3. Update DNS handler for ordered responses
4. Add per-service configuration support
5. Add metrics

### Phase 4: State Management & HA (Week 4)

**Tasks**:
1. Integrate with Raft for shared state
2. Connection tracking mechanism
3. State synchronization across nodes
4. Implement state persistence

### Phase 5: Documentation & Tools (Week 4-5)

**Tasks**:
1. User guide for load balancing
2. Strategy comparison guide
3. Configuration examples
4. CLI support for testing strategies
5. Grafana dashboard for LB metrics

### Testing Strategy

**Unit Tests**:
- Each strategy in isolation
- State management
- Edge cases (empty instances, single instance)

**Integration Tests**:
- HTTP API with load balancing
- DNS with multiple A records
- Health filtering
- Configuration override

**Performance Tests**:
- Selection latency benchmarks
- Memory usage per strategy
- Concurrent selection performance
- Large instance count (1000+)

**Chaos Tests**:
- Instance failure during selection
- State corruption
- Raft leader change during selection

### Performance Targets

- **Selection latency**: <5ms for most strategies
- **Throughput**: 10,000+ selections/second
- **Memory per service**: <1KB overhead
- **State sync**: <50ms propagation time
- **Concurrent requests**: Support 1000+ concurrent

### Configuration Examples

```yaml
# konsul.yaml
load_balancer:
  default_strategy: round-robin
  health_filter_enabled: true

  strategies:
    round_robin:
      enabled: true

    weighted_round_robin:
      enabled: true
      smooth_weighting: true

    ip_hash:
      enabled: true

    ring_hash:
      enabled: true
      virtual_nodes: 150

    least_connections:
      enabled: true
      connection_timeout: 30s

    latency_based:
      enabled: true
      latency_map:
        us-east-1:
          us-east-1: 5ms
          us-west-2: 60ms
          eu-west-1: 80ms
        us-west-2:
          us-east-1: 60ms
          us-west-2: 5ms
          eu-west-1: 100ms
```

### Migration Path

**Backward Compatibility**:
- Default behavior unchanged (unordered list)
- Opt-in via query parameters or configuration
- Existing clients continue to work

## References

- [NGINX Load Balancing Algorithms](https://www.nginx.com/blog/choosing-nginx-plus-load-balancing-techniques/)
- [HAProxy Load Balancing Algorithms](https://www.haproxy.com/blog/load-balancing-algorithms/)
- [Consistent Hashing Paper](https://en.wikipedia.org/wiki/Consistent_hashing)
- [Envoy Load Balancing](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/load_balancing/overview)
- [AWS ELB Load Balancing](https://docs.aws.amazon.com/elasticloadbalancing/latest/userguide/how-elastic-load-balancing-works.html)
- [Weighted Round Robin Algorithm (smooth)](https://github.com/phusion/nginx/commit/27e94984486058d73157038f7950a0a36ecc6e35)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-28 | Konsul Team | Initial proposal |
