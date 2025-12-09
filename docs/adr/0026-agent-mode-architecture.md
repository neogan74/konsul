# ADR-0026: Agent Mode Architecture

**Date**: 2025-12-06

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: architecture, scalability, agent, performance, distributed-systems

## Context

As documented in [ARCHITECTURE_USE_CASES.md](../ARCHITECTURE_USE_CASES.md), Konsul's direct client-to-server architecture works well for small to medium deployments (<100 services), but faces scalability challenges as the system grows:

### Current Limitations (Client-Server Only)

**At 100+ services**:
- **Server load**: Each service queries the server directly for discovery
- **Network overhead**: High volume of redundant queries (services query same data)
- **Registration latency**: ~50ms for service registration (network RTT + processing)
- **Cache invalidation**: Clients don't cache, resulting in repeated queries
- **Health check coordination**: Server performs all health checks (CPU bottleneck)

**Performance degradation**:
- 100 services × 10 queries/sec = 1000 req/sec to server
- 500 services × 10 queries/sec = 5000 req/sec (server becomes bottleneck)
- 1000 services × 10 queries/sec = 10K req/sec (requires expensive vertical scaling)

### Requirements for Scale

**100-500 Services**:
- Reduce server load by 80-90%
- Sub-millisecond discovery latency (local cache)
- Efficient health check execution
- Batch updates to reduce network traffic

**1000+ Services (Multi-Cluster)**:
- Agent as sidecar proxy (service mesh foundation)
- mTLS termination at agent
- Local policy enforcement
- Metrics aggregation

## Decision

We will implement **Agent Mode** - a distributed architecture where lightweight agents run on each node (or as sidecars) to handle local service operations and cache remote state.

### Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                   Agent Mode Architecture                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────── Control Plane ──────────────┐              │
│  │                                             │              │
│  │  ┌───────┐  ┌───────┐  ┌───────┐          │              │
│  │  │Server1│◄─┤Server2│◄─┤Server3│          │              │
│  │  │Leader │  │Follower│  │Follower│          │              │
│  │  └───┬───┘  └───┬───┘  └───┬───┘          │              │
│  │      │          │          │              │              │
│  │      └──────────┴──────────┘              │              │
│  │              Raft Cluster                 │              │
│  │       (Consensus, Replication)            │              │
│  └─────────────────┬──────────────────────────┘              │
│                    │                                         │
│                    │ Sync Protocol                           │
│                    │ (Delta updates, Batch, Compression)     │
│                    │                                         │
│  ┌─────────────────▼──── Data Plane ──────────────────┐     │
│  │                                                     │     │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐        │     │
│  │  │ Agent-1  │  │ Agent-2  │  │ Agent-N  │        │     │
│  │  │ (Node/   │  │ (Node/   │  │ (Node/   │        │     │
│  │  │ Sidecar) │  │ Sidecar) │  │ Sidecar) │        │     │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘        │     │
│  │       │             │             │              │     │
│  │       │             │             │              │     │
│  │  ┌────▼─────┐  ┌────▼─────┐  ┌────▼─────┐       │     │
│  │  │Services  │  │Services  │  │Services  │       │     │
│  │  │(Local)   │  │(Local)   │  │(Local)   │       │     │
│  │  └──────────┘  └──────────┘  └──────────┘       │     │
│  │                                                     │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### Agent Responsibilities

**1. Service Registration & Deregistration**
- Local services register with agent (not server)
- Agent batches registrations to server (every 10s or 100 changes)
- Immediate acknowledgment to local services (<1ms)

**2. Local Cache Management**
- Cache service registry (TTL: 60s)
- Cache KV store subset (prefix-based)
- Cache health check results
- LRU eviction policy

**3. Health Check Execution**
- Execute HTTP/TCP/gRPC health checks locally
- Report only status changes to server (not every check)
- Support TTL-based health checks
- Aggregate health status

**4. Service Discovery**
- Serve discovery queries from local cache
- Fall back to server if cache miss
- Pre-fetch popular services

**5. KV Store Operations**
- Cache frequently accessed keys
- Write-through cache for updates
- Watch key changes (prefix-based)

**6. Service Mesh (Future)**
- Act as sidecar proxy (Envoy integration)
- mTLS termination
- Traffic shaping
- Circuit breaking

### Communication Protocol

**Agent ↔ Server Protocol**:

```protobuf
// agent_protocol.proto
syntax = "proto3";

package konsul.agent;

// Agent Registration
message AgentInfo {
  string agent_id = 1;
  string node_name = 2;
  string node_ip = 3;
  string datacenter = 4;
  map<string, string> metadata = 5;
}

// Sync Request (Delta updates)
message SyncRequest {
  string agent_id = 1;
  int64 last_sync_index = 2;  // For delta sync
  repeated string watched_prefixes = 3;  // KV prefixes
  bool full_sync = 4;  // Force full sync
}

// Sync Response (Compressed)
message SyncResponse {
  int64 current_index = 1;
  repeated ServiceUpdate service_updates = 2;
  repeated KVUpdate kv_updates = 3;
  repeated HealthUpdate health_updates = 4;
  bytes compressed_data = 5;  // Snappy compression
}

// Service Update (Delta)
message ServiceUpdate {
  enum UpdateType {
    ADD = 0;
    UPDATE = 1;
    DELETE = 2;
  }
  UpdateType type = 1;
  string service_name = 2;
  Service service_data = 3;  // Only for ADD/UPDATE
}

// Batch Registration
message BatchRegisterRequest {
  string agent_id = 1;
  repeated Service services = 2;
  int64 sequence_number = 3;  // For idempotency
}
```

**Sync Frequency**:
- **Service Updates**: Every 10 seconds or on change
- **KV Updates**: Every 5 seconds or on change
- **Health Updates**: Every 30 seconds or on status change
- **Full Sync**: Every 5 minutes (safety net)

**Network Optimizations**:
- Delta sync (only changes since last sync)
- Snappy compression (80% reduction)
- Batch updates (reduce request overhead)
- Persistent connections (HTTP/2)

### Agent Configuration

```yaml
# agent-config.yaml
agent:
  # Agent identity
  id: "agent-node1-abc123"
  node_name: "k8s-worker-1"
  datacenter: "us-east"

  # Server connection
  server_address: "https://konsul-server.konsul-system.svc.cluster.local:8500"
  tls:
    enabled: true
    ca_cert: /etc/konsul/ca.crt
    client_cert: /etc/konsul/agent.crt
    client_key: /etc/konsul/agent.key

  # Cache configuration
  cache:
    service_ttl: 60s
    kv_ttl: 300s
    max_entries: 10000
    eviction_policy: lru

  # Health checks
  health_checks:
    enable_local_execution: true
    check_interval: 10s
    report_only_changes: true  # Don't spam server

  # Sync configuration
  sync:
    interval: 10s
    full_sync_interval: 300s
    batch_size: 100
    compression: true

  # Performance
  resources:
    memory_limit: 128Mi
    cpu_limit: 100m

  # Watched prefixes (KV)
  watched_prefixes:
    - "config/"
    - "feature_flags/"

  # Service mesh (future)
  mesh:
    enabled: false
    proxy_type: "envoy"
```

### API Endpoints

**Agent API (Port 8502)**:

```
Local Service Management:
  POST   /agent/service/register       - Register local service
  DELETE /agent/service/deregister/:id - Deregister service
  GET    /agent/services               - List local services

Service Discovery (Cached):
  GET    /agent/catalog/service/:name  - Get service (from cache)
  GET    /agent/catalog/services       - List all services

KV Store (Cached):
  GET    /agent/kv/:key                - Get key (from cache)
  PUT    /agent/kv/:key                - Set key (write-through)
  DELETE /agent/kv/:key                - Delete key

Health Checks:
  POST   /agent/check/register         - Register health check
  PUT    /agent/check/update/:id       - Update check result
  GET    /agent/checks                 - List local checks

Agent Management:
  GET    /agent/self                   - Agent info
  GET    /agent/metrics                - Agent metrics
  POST   /agent/reload                 - Reload config
  GET    /agent/health                 - Agent health
```

### Implementation Components

**1. Agent Core** (`internal/agent/agent.go`):

```go
package agent

import (
    "context"
    "sync"
    "time"
)

type Agent struct {
    config        *Config
    id            string
    serverClient  *ServerClient
    cache         *Cache
    healthChecker *HealthChecker
    syncEngine    *SyncEngine
    api           *API

    // State
    localServices map[string]*Service
    mu            sync.RWMutex

    // Lifecycle
    ctx    context.Context
    cancel context.CancelFunc
}

func NewAgent(cfg *Config) (*Agent, error) {
    ctx, cancel := context.WithCancel(context.Background())

    agent := &Agent{
        config:        cfg,
        id:            generateAgentID(),
        serverClient:  NewServerClient(cfg.ServerAddress),
        cache:         NewCache(cfg.Cache),
        healthChecker: NewHealthChecker(),
        syncEngine:    NewSyncEngine(cfg.Sync),
        localServices: make(map[string]*Service),
        ctx:           ctx,
        cancel:        cancel,
    }

    // Initialize components
    agent.api = NewAPI(agent)

    return agent, nil
}

func (a *Agent) Start() error {
    // Register with server
    if err := a.registerAgent(); err != nil {
        return err
    }

    // Start sync loop
    go a.syncEngine.Run(a.ctx, a.serverClient, a.cache)

    // Start health check loop
    go a.healthChecker.Run(a.ctx, a.localServices)

    // Start API server
    go a.api.Start(a.ctx)

    return nil
}

func (a *Agent) RegisterService(svc *Service) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    // Store locally
    a.localServices[svc.ID] = svc

    // Queue for batch sync (not blocking)
    a.syncEngine.QueueServiceUpdate(svc)

    return nil
}

func (a *Agent) GetService(name string) ([]*Service, error) {
    // Try cache first
    if cached, ok := a.cache.GetService(name); ok {
        return cached, nil
    }

    // Cache miss - fetch from server
    services, err := a.serverClient.GetService(name)
    if err != nil {
        return nil, err
    }

    // Update cache
    a.cache.SetService(name, services)

    return services, nil
}
```

**2. Cache** (`internal/agent/cache.go`):

```go
package agent

import (
    "sync"
    "time"
    "github.com/hashicorp/golang-lru/v2/expirable"
)

type Cache struct {
    services *expirable.LRU[string, []*Service]
    kv       *expirable.LRU[string, string]
    health   *expirable.LRU[string, HealthStatus]
    mu       sync.RWMutex

    // Metrics
    hits   uint64
    misses uint64
}

func NewCache(cfg CacheConfig) *Cache {
    return &Cache{
        services: expirable.NewLRU[string, []*Service](
            cfg.MaxEntries,
            nil,
            cfg.ServiceTTL,
        ),
        kv: expirable.NewLRU[string, string](
            cfg.MaxEntries,
            nil,
            cfg.KVTTL,
        ),
        health: expirable.NewLRU[string, HealthStatus](
            cfg.MaxEntries,
            nil,
            30*time.Second,
        ),
    }
}

func (c *Cache) GetService(name string) ([]*Service, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    svc, ok := c.services.Get(name)
    if ok {
        atomic.AddUint64(&c.hits, 1)
    } else {
        atomic.AddUint64(&c.misses, 1)
    }
    return svc, ok
}

func (c *Cache) HitRate() float64 {
    hits := atomic.LoadUint64(&c.hits)
    misses := atomic.LoadUint64(&c.misses)
    total := hits + misses
    if total == 0 {
        return 0
    }
    return float64(hits) / float64(total)
}
```

**3. Sync Engine** (`internal/agent/sync.go`):

```go
package agent

import (
    "context"
    "time"
)

type SyncEngine struct {
    config       SyncConfig
    lastIndex    int64
    pendingQueue chan ServiceUpdate
    batchBuffer  []ServiceUpdate
}

func (s *SyncEngine) Run(ctx context.Context, client *ServerClient, cache *Cache) {
    ticker := time.NewTicker(s.config.Interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return

        case <-ticker.C:
            // Periodic sync
            s.performSync(client, cache)

        case update := <-s.pendingQueue:
            // Buffer updates
            s.batchBuffer = append(s.batchBuffer, update)

            // Flush if batch full
            if len(s.batchBuffer) >= s.config.BatchSize {
                s.flushBatch(client)
            }
        }
    }
}

func (s *SyncEngine) performSync(client *ServerClient, cache *Cache) error {
    // Request delta updates
    req := &SyncRequest{
        LastSyncIndex:    s.lastIndex,
        WatchedPrefixes:  s.config.WatchedPrefixes,
        FullSync:         false,
    }

    resp, err := client.Sync(req)
    if err != nil {
        return err
    }

    // Update cache with deltas
    for _, update := range resp.ServiceUpdates {
        cache.ApplyServiceUpdate(update)
    }

    for _, update := range resp.KVUpdates {
        cache.ApplyKVUpdate(update)
    }

    // Update last index
    s.lastIndex = resp.CurrentIndex

    return nil
}

func (s *SyncEngine) flushBatch(client *ServerClient) error {
    if len(s.batchBuffer) == 0 {
        return nil
    }

    // Send batch to server
    err := client.BatchUpdate(s.batchBuffer)
    if err != nil {
        return err
    }

    // Clear buffer
    s.batchBuffer = s.batchBuffer[:0]

    metrics.AgentBatchesTotal.Inc()
    metrics.AgentBatchSize.Observe(float64(len(s.batchBuffer)))

    return nil
}
```

**4. Health Checker** (`internal/agent/health.go`):

```go
package agent

import (
    "context"
    "net/http"
    "time"
)

type HealthChecker struct {
    checks map[string]*HealthCheck
    client *http.Client
}

func (h *HealthChecker) Run(ctx context.Context, services map[string]*Service) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return

        case <-ticker.C:
            h.executeChecks(services)
        }
    }
}

func (h *HealthChecker) executeChecks(services map[string]*Service) {
    for _, svc := range services {
        for _, check := range svc.Checks {
            status := h.performCheck(check)

            // Only report if status changed
            if status != check.LastStatus {
                h.reportStatusChange(svc.ID, check.ID, status)
                check.LastStatus = status
            }
        }
    }
}

func (h *HealthChecker) performCheck(check *HealthCheck) HealthStatus {
    switch check.Type {
    case "http":
        return h.httpCheck(check)
    case "tcp":
        return h.tcpCheck(check)
    case "grpc":
        return h.grpcCheck(check)
    default:
        return HealthStatusUnknown
    }
}
```

### Deployment Modes

**1. DaemonSet Mode (Node-level)**:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: konsul-agent
spec:
  template:
    spec:
      hostNetwork: true
      containers:
      - name: agent
        image: konsul/agent:latest
        env:
        - name: KONSUL_AGENT_MODE
          value: "node"
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
```

**2. Sidecar Mode (Pod-level)**:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: app
    image: myapp:latest

  - name: konsul-agent
    image: konsul/agent:latest
    env:
    - name: KONSUL_AGENT_MODE
      value: "sidecar"
    - name: POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
```

### Performance Characteristics

**Latency**:
- Service registration: <1ms (local agent) vs ~50ms (direct to server)
- Service discovery: <1ms (cache hit) vs ~10ms (server query)
- KV read: <1ms (cache hit) vs ~5ms (server query)
- Health check: Local execution (no network hop)

**Throughput**:
- Agent handles: 1000+ local operations/sec
- Server sees: 10-100 sync requests/sec (vs 10,000+ without agents)
- **90% server load reduction**

**Resource Usage**:
- Memory: 64-128MB per agent
- CPU: 50-100m per agent
- Network: 1-10 KB/sec per agent (vs 100KB/sec without caching)

### Metrics

```
# Agent metrics
konsul_agent_cache_hit_ratio
konsul_agent_cache_entries{type="service|kv|health"}
konsul_agent_local_services_total
konsul_agent_sync_duration_seconds
konsul_agent_sync_errors_total
konsul_agent_batch_size
konsul_agent_health_checks_total{status="passing|warning|critical"}

# Server-side metrics
konsul_server_connected_agents
konsul_server_agent_sync_requests_total
konsul_server_agent_sync_duration_seconds
```

## Alternatives Considered

### Alternative 1: Client-Side SDK with Caching
- **Pros**: No agent process needed, language-specific optimizations
- **Cons**: Duplicate caching logic per SDK, no health check delegation, no service mesh foundation
- **Reason for rejection**: Doesn't solve server load problem, limited capabilities

### Alternative 2: Service Mesh Proxy (Envoy-only)
- **Pros**: Production-ready proxy, rich features
- **Cons**: Heavy weight (~50MB), complex configuration, overkill for simple discovery
- **Reason for rejection**: Too complex for basic use cases, agents can integrate Envoy later

### Alternative 3: gossip Protocol (Memberlist)
- **Pros**: Peer-to-peer, eventual consistency, battle-tested (Consul uses it)
- **Cons**: Complex to implement, harder to reason about, difficult to secure
- **Reason for rejection**: Client-server with agents simpler and sufficient

### Alternative 4: No Agents (Keep Direct Client-Server)
- **Pros**: Simpler architecture, fewer moving parts
- **Cons**: Doesn't scale beyond 100-200 services, high server load
- **Reason for rejection**: Blocks enterprise adoption

## Consequences

### Positive
- **90% server load reduction** at scale
- **Sub-millisecond latency** for local operations
- **Foundation for service mesh** (future)
- **Better fault isolation** (agent failures don't affect other nodes)
- **Flexible deployment** (DaemonSet or sidecar)
- **Network efficiency** (80% traffic reduction)
- **Health check distribution** (no server bottleneck)
- **Cache hit rates** >95% in production

### Negative
- **Increased operational complexity** (more processes to manage)
- **Additional resource usage** (64-128MB per agent)
- **Cache consistency** (eventual consistency with 10s lag)
- **More failure modes** (agent failures to consider)
- **Learning curve** for operators

### Neutral
- Agent deployment strategy (DaemonSet vs sidecar)
- Cache invalidation strategy
- Monitoring requirements (agent-specific metrics)

## Implementation Notes

### Phase 1: Core Agent (4 weeks)
1. Agent core structure
2. Cache implementation
3. Sync engine
4. Agent API
5. Unit tests

### Phase 2: Health Checks (2 weeks)
1. Health check engine
2. HTTP/TCP/gRPC checks
3. Status change detection
4. Health check metrics

### Phase 3: Integration (2 weeks)
1. Server-side agent protocol
2. DaemonSet deployment
3. Sidecar deployment
4. Kubernetes operator integration

### Phase 4: Production Readiness (2 weeks)
1. Performance testing (1000+ services)
2. Failure scenarios
3. Monitoring dashboards
4. Documentation

**Total**: 10 weeks

### Migration Path

**Existing Deployments**:
1. Deploy agents alongside existing setup
2. Gradually move services to use agents
3. Monitor cache hit rates and server load
4. Once >80% traffic through agents, considered migrated

**New Deployments**:
- Start with agents from day one (DaemonSet or sidecar)

## References

- [ARCHITECTURE_USE_CASES.md](../ARCHITECTURE_USE_CASES.md)
- [Consul Agent Architecture](https://www.consul.io/docs/agent)
- [Linkerd Proxy](https://linkerd.io/2/reference/architecture/)
- [Envoy Proxy Architecture](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/arch_overview)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-06 | Konsul Team | Initial version |