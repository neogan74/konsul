# ADR-0027: Multi-Datacenter Federation

**Date**: 2025-12-06

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: multi-datacenter, federation, wan, global, replication, disaster-recovery

## Context

Modern enterprises require global deployment with services spanning multiple geographic regions and data centers. Current Konsul architecture supports only single-datacenter deployments, which creates limitations:

### Current Limitations (Single Datacenter)

1. **No geographic distribution**: Cannot deploy across multiple regions
2. **Single region failure**: Entire system unavailable if DC goes down
3. **High latency**: Global users experience high latency to single DC
4. **No disaster recovery**: No automatic failover to backup DC
5. **Compliance issues**: Data residency requirements (GDPR, CCPA)
6. **Limited scalability**: Single DC capacity ceiling

### Real-World Requirements

**Global E-Commerce**:
- Services in US-East, US-West, EU, Asia
- Users routed to nearest DC (low latency)
- Cross-DC service discovery (payment, inventory)
- Data residency (EU user data stays in EU)

**Financial Services**:
- Active-active across multiple DCs (no downtime)
- Cross-DC replication for DR
- Regional isolation for compliance
- <100ms latency for transactions

**SaaS Platform**:
- Multi-tenant with region preferences
- Cross-region service communication
- Global service catalog
- Regional failover

## Decision

We will implement **Multi-Datacenter Federation** using a hub-and-spoke model with WAN gossip protocol for service discovery across datacenters.

### Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│              Multi-Datacenter Federation Architecture             │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────┐         WAN Gossip          ┌─────────────┐    │
│  │   DC1:      │◄───────────────────────────►│   DC2:      │    │
│  │   US-East   │         (Mesh)              │   US-West   │    │
│  │  (Primary)  │                              │ (Secondary) │    │
│  ├─────────────┤                              ├─────────────┤    │
│  │ 5 Servers   │                              │ 5 Servers   │    │
│  │ + Agents    │                              │ + Agents    │    │
│  │ + Gateway   │                              │ + Gateway   │    │
│  └──────┬──────┘                              └──────┬──────┘    │
│         │                                            │           │
│         │          ┌──────────────────┐             │           │
│         │          │                  │             │           │
│         └─────────►│  Mesh Gateway    │◄────────────┘           │
│                    │   (Cross-DC      │                         │
│                    │    Routing)      │                         │
│                    └────────┬─────────┘                         │
│                             │                                   │
│  ┌──────────────────────────┼──────────────────────────┐       │
│  │                          │                          │       │
│  ▼                          ▼                          ▼       │
│  DC3: EU-Central      DC4: Asia-Pacific      DC5: AU-East     │
│  (Secondary)          (Secondary)            (Secondary)      │
│  ├─────────────┐      ├─────────────┐       ├─────────────┐  │
│  │ 3 Servers   │      │ 3 Servers   │       │ 3 Servers   │  │
│  │ + Agents    │      │ + Agents    │       │ + Agents    │  │
│  │ + Gateway   │      │ + Gateway   │       │ + Gateway   │  │
│  └─────────────┘      └─────────────┘       └─────────────┘  │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

### Key Components

**1. Datacenter (DC)**:
- Independent Raft cluster (3-5 servers)
- Own service registry and KV store
- Local agents for service management
- Mesh gateway for cross-DC communication

**2. Primary Datacenter**:
- Authoritative for global configuration
- Manages ACL policies and roles
- Certificate authority for mTLS
- Source of truth for replication

**3. Secondary Datacenters**:
- Replicate from primary
- Independent service registries
- Read-only global config
- Read-write local services

**4. Mesh Gateway**:
- Proxy for cross-DC traffic
- TLS encryption for WAN
- Traffic routing and load balancing
- Firewall-friendly (single port)

**5. WAN Gossip**:
- Datacenter discovery
- Health monitoring across DCs
- Membership changes
- Serf protocol (same as Consul)

### Federation Models

**Model 1: Hub-and-Spoke (Recommended)**

```
Primary DC (Hub) ◄──► Secondary DC1 (Spoke)
     ▲
     ├───────────────► Secondary DC2 (Spoke)
     │
     └───────────────► Secondary DC3 (Spoke)
```

**Characteristics**:
- Primary is source of truth
- Secondaries replicate from primary
- Simplifies configuration management
- Single point for policy updates

**Model 2: Mesh (Full Federation)**

```
DC1 ◄──► DC2
 ▲ ╲     ╱ ▲
 │   ╲ ╱   │
 │   ╱ ╲   │
 ▼ ╱     ╲ ▼
DC3 ◄──► DC4
```

**Characteristics**:
- All DCs equal peers
- No primary/secondary distinction
- Higher network complexity
- Better fault tolerance

**Decision**: Start with **Hub-and-Spoke**, add Mesh option later

### Replication Strategy

**Global Data (Replicated)**:
- ACL policies and tokens
- CA certificates and roots
- Global KV prefix (e.g., `global/*`)
- Intentions (service-to-service policies)

**Local Data (Not Replicated)**:
- Service registrations
- Local KV prefix (e.g., `local/*`)
- Health check results
- Agent state

**Replication Protocol**:

```protobuf
syntax = "proto3";

package konsul.federation;

// Replication Request
message ReplicationRequest {
  string source_dc = 1;
  string target_dc = 2;
  int64 last_replicated_index = 3;
  repeated string prefixes = 4;  // KV prefixes to replicate
}

// Replication Response
message ReplicationResponse {
  int64 current_index = 1;
  repeated ReplicatedEntry entries = 2;
  bool full_sync_required = 3;
}

// Replicated Entry
message ReplicatedEntry {
  enum EntryType {
    KV = 0;
    ACL_POLICY = 1;
    ACL_TOKEN = 2;
    CA_CERT = 3;
    INTENTION = 4;
  }
  EntryType type = 1;
  string key = 2;
  bytes value = 3;
  int64 modify_index = 4;
  bool deleted = 5;
}
```

**Replication Frequency**:
- **ACL changes**: Immediate (via watch)
- **Global KV**: Every 30 seconds
- **CA certificates**: Every 5 minutes
- **Full sync**: Every 30 minutes (safety net)

### Service Discovery Across DCs

**Local Discovery (Default)**:
```bash
# Query services in same DC (fast)
konsulctl service get api

# Returns services from local DC only
```

**Cross-DC Discovery**:
```bash
# Query services in specific DC
konsulctl service get api --datacenter us-west

# Query services in all DCs
konsulctl service get api --all-datacenters

# Query with failover
konsulctl service get api --failover us-west,eu-central
```

**Geo-Aware Routing**:
```go
// Application code
func discoverService(name string) (*Service, error) {
    // Try local DC first (lowest latency)
    local, err := konsul.Service.Get(name, &QueryOptions{
        Datacenter: "local",
    })
    if err == nil && len(local) > 0 {
        return selectNearest(local), nil
    }

    // Failover to nearest DC
    global, err := konsul.Service.Get(name, &QueryOptions{
        AllDatacenters: true,
        PreferLocal: true,
        SortByLatency: true,
    })
    if err != nil {
        return nil, err
    }

    return selectNearest(global), nil
}
```

### Mesh Gateway Configuration

**Gateway Deployment**:

```yaml
# mesh-gateway.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: konsul-mesh-gateway
  namespace: konsul-system
spec:
  replicas: 3  # HA for cross-DC traffic
  template:
    spec:
      containers:
      - name: gateway
        image: konsul/mesh-gateway:latest
        env:
        - name: DATACENTER
          value: "us-east"
        - name: WAN_ADDRESS
          value: "mesh-gateway.us-east.example.com:8443"
        - name: TLS_ENABLED
          value: "true"
        ports:
        - containerPort: 8443
          name: wan
        - containerPort: 8500
          name: http
---
apiVersion: v1
kind: Service
metadata:
  name: konsul-mesh-gateway
spec:
  type: LoadBalancer
  ports:
  - name: wan
    port: 8443
    targetPort: 8443
  selector:
    app: konsul-mesh-gateway
```

**Gateway Routing**:
- Routes traffic between DCs
- Handles mTLS termination
- Load balances across DC servers
- Firewall-friendly (single port 8443)

### WAN Gossip Configuration

**Server Configuration**:

```yaml
# Primary DC (US-East)
server:
  datacenter: us-east
  primary_datacenter: us-east
  wan:
    enabled: true
    bind_addr: "0.0.0.0:8302"
    advertise_addr: "konsul-us-east.example.com:8302"
    members:
      - "konsul-us-west.example.com:8302"
      - "konsul-eu-central.example.com:8302"
      - "konsul-asia-pacific.example.com:8302"

# Secondary DC (US-West)
server:
  datacenter: us-west
  primary_datacenter: us-east
  wan:
    enabled: true
    bind_addr: "0.0.0.0:8302"
    advertise_addr: "konsul-us-west.example.com:8302"
    join:
      - "konsul-us-east.example.com:8302"  # Join primary
```

**Gossip Protocol** (Serf):
- Membership management
- Failure detection (swim protocol)
- Event propagation
- Network topology awareness

### Deployment Topology

**Recommended Setups**:

**2 DCs (Active-Passive DR)**:
```
Primary DC (Active)     Secondary DC (Passive)
US-East                 US-West
- 5 servers             - 3 servers
- Production traffic    - Standby for DR
- Read-write           - Read-only (replicated)
```

**3 DCs (Active-Active-Active)**:
```
DC1: US-East      DC2: EU-Central    DC3: Asia-Pacific
- 5 servers       - 5 servers        - 5 servers
- Americas        - Europe           - Asia/Pacific
- Active traffic  - Active traffic   - Active traffic
```

**5 DCs (Global)**:
```
DC1: US-East (Primary)
DC2: US-West
DC3: EU-Central
DC4: Asia-Pacific
DC5: AU-East
```

### Cross-DC Configuration Management

**Global Configuration** (replicated):

```bash
# Set in primary DC, replicated to all
konsulctl kv set --global config/app/version "v2.0"
konsulctl kv set --global config/feature_flags/new_ui "true"

# Reads from any DC get replicated value
konsulctl kv get --datacenter us-east config/app/version  # v2.0
konsulctl kv get --datacenter eu-central config/app/version  # v2.0
```

**DC-Specific Configuration** (not replicated):

```bash
# Set in specific DC
konsulctl kv set --datacenter us-east config/local/db_url "postgresql://us-db:5432/app"
konsulctl kv set --datacenter eu-central config/local/db_url "postgresql://eu-db:5432/app"

# Each DC has its own value
```

### Failover Strategies

**1. Automatic Failover**:

```yaml
# Service with failover configuration
service:
  name: payment-api
  datacenter: us-east
  failover:
    - datacenter: us-west
      targets: 3  # Failover to 3 instances
    - datacenter: eu-central
      targets: 2
```

**2. Circuit Breaker**:

```go
// Automatic failover with circuit breaker
func callPaymentAPI() error {
    // Try local DC
    err := callService("us-east", "payment-api")
    if err != nil {
        // Failover to us-west
        err = callService("us-west", "payment-api")
    }
    if err != nil {
        // Failover to eu-central
        err = callService("eu-central", "payment-api")
    }
    return err
}
```

**3. Health-Based Failover**:
- Monitor DC health
- Automatically route to healthy DC
- Gradual traffic shift (no thundering herd)

### Network Requirements

**Connectivity**:
- **Mesh Gateway Port**: 8443 (TCP, bidirectional)
- **WAN Gossip Port**: 8302 (TCP + UDP, bidirectional)
- **Bandwidth**: ~1-10 Mbps per DC pair (depends on replication)

**Latency Tolerance**:
- **<50ms**: Optimal (same region)
- **50-100ms**: Good (cross-region)
- **100-200ms**: Acceptable (cross-continent)
- **>200ms**: May impact replication lag

**Firewall Rules**:
```
# Allow mesh gateway traffic
ALLOW TCP 8443 FROM any-dc-mesh-gateway TO any-dc-mesh-gateway

# Allow WAN gossip
ALLOW TCP 8302 FROM any-dc-server TO any-dc-server
ALLOW UDP 8302 FROM any-dc-server TO any-dc-server
```

### Monitoring & Observability

**Metrics**:

```
# Federation health
konsul_federation_connected_datacenters
konsul_federation_replication_lag_seconds{source_dc, target_dc}
konsul_federation_wan_gossip_health

# Cross-DC queries
konsul_cross_dc_queries_total{source_dc, target_dc, service}
konsul_cross_dc_query_duration_seconds

# Mesh gateway
konsul_mesh_gateway_connections{datacenter}
konsul_mesh_gateway_bytes_sent{source_dc, target_dc}
konsul_mesh_gateway_bytes_received{source_dc, target_dc}
```

**Alerts**:
- Replication lag >60 seconds
- DC disconnected >5 minutes
- Mesh gateway down
- WAN gossip failures

### Disaster Recovery

**Scenario 1: DC Failure (Partial)**

```bash
# DC1 (primary) goes down
# Promote DC2 to primary
konsulctl datacenter promote --datacenter us-west

# Update all clients to use new primary
# Automatic via failover configuration
```

**Scenario 2: DC Failure (Total)**

```bash
# Restore from backup in new DC
konsulctl backup restore --file backup-us-east.tar.gz --datacenter us-east-new

# Re-establish federation
konsulctl datacenter federate --primary us-east-new --secondary us-west
```

**Scenario 3: Split-Brain**

```bash
# If network partition splits federation
# Manual intervention required

# Check cluster state
konsulctl cluster status --all-datacenters

# Force reconciliation
konsulctl cluster reconcile --force
```

### Implementation Phases

**Phase 1: Basic Federation (6 weeks)**
1. WAN gossip protocol (Serf integration)
2. Mesh gateway implementation
3. Cross-DC service discovery
4. Basic replication (global KV)
5. Datacenter configuration

**Phase 2: Advanced Replication (4 weeks)**
1. ACL policy replication
2. CA certificate replication
3. Intention replication
4. Delta replication optimization
5. Conflict resolution

**Phase 3: Failover & DR (3 weeks)**
1. Automatic failover
2. Health-based routing
3. DC promotion/demotion
4. Backup/restore across DCs
5. Split-brain detection

**Phase 4: Production Hardening (3 weeks)**
1. Performance optimization (latency, bandwidth)
2. Security hardening (mTLS, network policies)
3. Monitoring dashboards
4. Runbooks and documentation
5. Chaos testing

**Total**: 16 weeks (~4 months)

## Alternatives Considered

### Alternative 1: Multi-Primary (All DCs Equal)
- **Pros**: No single point of failure, symmetric architecture
- **Cons**: Complex conflict resolution, harder to reason about, eventual consistency challenges
- **Reason for rejection**: Hub-and-spoke simpler for v1, can add later

### Alternative 2: Database Replication (PostgreSQL)
- **Pros**: Leverage existing database replication
- **Cons**: Tight coupling to database, complex setup, not service-registry native
- **Reason for rejection**: Want built-in solution, database-agnostic

### Alternative 3: Kubernetes Federation
- **Pros**: Kubernetes-native, multi-cluster support
- **Cons**: K8s-only, doesn't help non-K8s deployments, limited to K8s constructs
- **Reason for rejection**: Need solution that works with and without K8s

### Alternative 4: DNS-Based Federation
- **Pros**: Simple, works with any client, no code changes
- **Cons**: No health checking, no smart routing, DNS caching issues
- **Reason for rejection**: Too limited for enterprise needs

### Alternative 5: API Gateway Federation
- **Pros**: Centralized routing, traffic management
- **Cons**: Single point of failure, doesn't help with service registry
- **Reason for rejection**: Need distributed solution

## Consequences

### Positive
- **Global deployment** capability
- **Disaster recovery** with automatic failover
- **Low latency** for global users (nearest DC)
- **Data residency** compliance (regional isolation)
- **Horizontal scalability** (add DCs as needed)
- **Fault isolation** (DC failures don't affect others)
- **Geo-aware routing** for performance

### Negative
- **Increased complexity** (multi-DC operations)
- **Higher costs** (multiple DCs to run)
- **Eventual consistency** (replication lag)
- **Network requirements** (cross-DC connectivity)
- **Operational overhead** (more infrastructure to manage)
- **Debugging complexity** (distributed traces needed)

### Neutral
- Federation model choice (hub-and-spoke vs mesh)
- Replication strategy (what to replicate)
- Failover policies (automatic vs manual)

## References

- [Consul Multi-Datacenter](https://www.consul.io/docs/architecture/multi-datacenter)
- [Serf Protocol](https://www.serf.io/docs/internals/gossip.html)
- [ARCHITECTURE_USE_CASES.md](../ARCHITECTURE_USE_CASES.md#scenario-4-large-enterprise-1000-servers-multi-cluster)
- [ADR-0026: Agent Mode Architecture](./0026-agent-mode-architecture.md)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-06 | Konsul Team | Initial version |