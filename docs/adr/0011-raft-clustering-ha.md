# ADR-0011: Raft Consensus for Clustering and High Availability

**Date**: 2025-10-09

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: clustering, high-availability, raft, distributed-systems, production

## Context

Konsul currently operates as a **single-node system**. This creates several critical limitations for production deployments:

### Current Limitations

1. **Single Point of Failure (SPOF)**: If the node crashes, the entire system is unavailable
2. **No data redundancy**: Data loss if disk fails or node is destroyed
3. **Downtime during upgrades**: Must stop service to upgrade
4. **Limited scalability**: Single node capacity ceiling
5. **No geographic distribution**: Cannot span multiple datacenters
6. **Manual failover**: Requires human intervention to restore service
7. **Not production-grade**: Unacceptable for critical infrastructure

### Requirements

**High Availability**:
- Survive node failures (N-1 fault tolerance)
- Automatic leader election
- No manual intervention for failover
- Sub-second failure detection
- Split-brain prevention

**Data Consistency**:
- Strong consistency guarantees
- Linearizable reads and writes
- No data loss on leader failure
- Replicated state machine

**Operational**:
- Dynamic cluster membership (add/remove nodes)
- Rolling upgrades without downtime
- Health monitoring and diagnostics
- Configurable cluster sizes (3, 5, 7 nodes typical)
- Support for odd number of nodes (quorum)

**Performance**:
- <10ms write latency (within datacenter)
- High read throughput (from followers)
- Minimal replication overhead

## Decision

We will implement **Raft consensus algorithm** for clustering and high availability using HashiCorp's `hashicorp/raft` library.

### Why Raft?

**Raft Advantages**:
- Understandable consensus algorithm
- Strong leader model (simplifies design)
- Battle-tested in production (Consul, etcd, Nomad)
- Excellent Go implementation available
- Log-based replication
- Membership changes supported
- Well-documented and researched

### Architecture

```
┌─────────────────────────────────────────────────────┐
│                 Konsul Cluster (3 nodes)            │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────┐      ┌──────────┐      ┌──────────┐ │
│  │  Node 1  │      │  Node 2  │      │  Node 3  │ │
│  │ (Leader) │◄────►│(Follower)│◄────►│(Follower)│ │
│  └──────────┘      └──────────┘      └──────────┘ │
│       │                 │                 │        │
│       ├─────────────────┴─────────────────┤        │
│       │         Raft Consensus            │        │
│       └────────────────────────────────────┘       │
│                                                     │
│  Client ──► Any Node ──► Leader ──► Replicate      │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Raft Components

**1. Leader Election**
- Nodes start as followers
- If no heartbeat from leader → become candidate
- Request votes from peers
- Majority votes → become leader
- Leader sends periodic heartbeats

**2. Log Replication**
- All writes go through leader
- Leader appends to log, replicates to followers
- Follower acknowledges receipt
- Majority acknowledgment → commit
- Apply committed entries to state machine

**3. State Machine**
- KV store operations
- Service registrations/deregistrations
- Health check updates
- Configuration changes

**4. Log Storage**
- Persistent log on disk (BadgerDB)
- In-memory log cache
- Snapshot + compaction
- Log replay on restart

### Cluster Configuration

**Quorum Sizes**:
- **3 nodes**: Tolerates 1 failure (recommended minimum)
- **5 nodes**: Tolerates 2 failures (production standard)
- **7 nodes**: Tolerates 3 failures (large deployments)

**Node Roles**:
- **Leader**: Handles all writes, coordinates replication
- **Follower**: Replicates log, can serve reads
- **Candidate**: Transitional state during election

### Write Path

```
1. Client sends write request to any node
2. If follower, redirect to leader
3. Leader appends to local log
4. Leader replicates to followers (parallel)
5. Followers persist log entry
6. Followers acknowledge to leader
7. Once majority acknowledges → commit
8. Leader applies to state machine
9. Leader responds to client
10. Followers apply committed entries asynchronously
```

**Write Latency**: 1 RTT + log persist (typically 5-10ms)

### Read Path

**Options**:

**1. Leader Reads (Linearizable)**
```
- Client reads from leader
- Leader checks it's still leader (heartbeat quorum)
- Serve from leader's state machine
- Consistency: Linearizable
- Latency: Low (no network round trip)
```

**2. Follower Reads (Stale)**
```
- Client reads from any node
- Serve from local state machine
- Consistency: Eventually consistent
- Latency: Very low (local read)
- Use case: Monitoring, dashboards
```

**3. Consistent Reads (ReadIndex)**
```
- Client sends read to leader
- Leader confirms leadership with quorum
- Return data from state machine
- Consistency: Linearizable
- Latency: 1 RTT for quorum check
```

### Membership Changes

**Add Node**:
```
1. Start new node with cluster address
2. New node joins as non-voter
3. Catches up with log
4. Once caught up → promote to voter
5. Update cluster configuration (Raft operation)
```

**Remove Node**:
```
1. Graceful shutdown signals intent
2. Leader commits configuration change
3. Node removed from cluster
4. Remaining nodes form new quorum
```

**Safety**: Raft ensures only one configuration change at a time

## Alternatives Considered

### Alternative 1: Multi-Raft (Like CockroachDB)
- **Pros**:
  - Better scalability (shard data across Raft groups)
  - Higher throughput
  - Parallel replication
- **Cons**:
  - Much more complex
  - Harder to reason about
  - Overkill for service discovery
  - Complex failure scenarios
- **Reason for rejection**: Complexity not justified; single Raft sufficient

### Alternative 2: Paxos Consensus
- **Pros**:
  - Proven algorithm (academic gold standard)
  - More flexible than Raft
  - No leader required
- **Cons**:
  - Harder to understand and implement
  - No production Go implementations
  - More complex failure recovery
  - Weaker leadership model
- **Reason for rejection**: Raft simpler and well-supported in Go

### Alternative 3: etcd Raft (etcd/raft)
- **Pros**:
  - High-quality implementation
  - Used by Kubernetes
  - Well-documented
- **Cons**:
  - Lower-level API (more integration work)
  - Less documentation than hashicorp/raft
  - Different design philosophy
- **Reason for rejection**: hashicorp/raft better documented, higher-level API

### Alternative 4: Active-Passive Replication
- **Pros**:
  - Simpler than Raft
  - Easy to implement
  - Lower overhead
- **Cons**:
  - Manual failover required
  - Split-brain risk
  - No automatic leader election
  - Weaker consistency
- **Reason for rejection**: Not truly HA; manual intervention needed

### Alternative 5: Master-Slave with ZooKeeper
- **Pros**:
  - Proven pattern
  - ZooKeeper handles coordination
- **Cons**:
  - External dependency (ZooKeeper cluster)
  - More operational complexity
  - Another system to monitor
  - Network latency to ZooKeeper
- **Reason for rejection**: Prefer embedded consensus; no external deps

### Alternative 6: No Clustering (Current State)
- **Pros**:
  - Simple
  - No implementation needed
  - No distributed systems complexity
- **Cons**:
  - Not production-ready
  - Single point of failure
  - No data redundancy
  - Unacceptable for critical systems
- **Reason for rejection**: Production requirements demand HA

## Consequences

### Positive
- **High availability**: Survives node failures automatically
- **Data safety**: Replicated to multiple nodes
- **Automatic failover**: Sub-second leader election
- **Strong consistency**: Linearizable reads/writes
- **Zero downtime upgrades**: Rolling restart possible
- **Geographic distribution**: Nodes in multiple AZs
- **Production ready**: Battle-tested consensus algorithm
- **Scalability**: Add nodes for redundancy
- **Well-understood**: Raft is widely studied
- **Great library**: hashicorp/raft mature and maintained

### Negative
- **Complexity increase**: Distributed systems are hard
- **Write latency**: +5-10ms for replication
- **Network dependency**: Cluster requires network connectivity
- **Operational overhead**: More nodes to manage
- **Split-brain possible**: If network partitions
- **Quorum required**: Need majority for writes
- **Configuration complexity**: Cluster setup more involved
- **Debugging harder**: Distributed traces needed
- **Storage overhead**: Log replication increases disk usage

### Neutral
- Need monitoring for cluster health
- Odd number of nodes required (3, 5, 7)
- Leader election takes 100-300ms
- Read consistency configurable

## Implementation Notes

### Phase 1: Core Raft Integration (3-4 weeks)

**Dependencies**:
```go
import (
    "github.com/hashicorp/raft"
    raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)
```

**Raft Setup**:
```go
type RaftNode struct {
    raft      *raft.Raft
    fsm       *KonsulFSM
    transport *raft.NetworkTransport
    config    *raft.Config
}

func NewRaftNode(cfg Config) (*RaftNode, error) {
    // Configure Raft
    raftConfig := raft.DefaultConfig()
    raftConfig.LocalID = raft.ServerID(cfg.NodeID)
    raftConfig.HeartbeatTimeout = 1 * time.Second
    raftConfig.ElectionTimeout = 1 * time.Second
    raftConfig.LeaderLeaseTimeout = 500 * time.Millisecond

    // Create FSM (Finite State Machine)
    fsm := &KonsulFSM{
        kvStore:      kvStore,
        serviceStore: serviceStore,
    }

    // Log store (persistent)
    logStore, err := raftboltdb.NewBoltStore(cfg.LogPath)

    // Stable store (persistent)
    stableStore, err := raftboltdb.NewBoltStore(cfg.StablePath)

    // Snapshot store
    snapshotStore, err := raft.NewFileSnapshotStore(
        cfg.SnapshotPath, 3, os.Stderr,
    )

    // Transport (network)
    transport, err := raft.NewTCPTransport(
        cfg.BindAddr, nil, 3, 10*time.Second, os.Stderr,
    )

    // Create Raft
    r, err := raft.NewRaft(
        raftConfig, fsm, logStore, stableStore,
        snapshotStore, transport,
    )

    return &RaftNode{raft: r, fsm: fsm, transport: transport}, nil
}
```

**Finite State Machine (FSM)**:
```go
type KonsulFSM struct {
    kvStore      *store.KVStore
    serviceStore *store.ServiceStore
    mu           sync.RWMutex
}

type LogEntry struct {
    Type      string          // "kv_set", "kv_delete", "service_register", etc.
    Key       string
    Value     []byte
    Timestamp time.Time
}

func (f *KonsulFSM) Apply(log *raft.Log) interface{} {
    var entry LogEntry
    if err := json.Unmarshal(log.Data, &entry); err != nil {
        return err
    }

    f.mu.Lock()
    defer f.mu.Unlock()

    switch entry.Type {
    case "kv_set":
        return f.kvStore.Set(entry.Key, entry.Value)
    case "kv_delete":
        return f.kvStore.Delete(entry.Key)
    case "service_register":
        var svc Service
        json.Unmarshal(entry.Value, &svc)
        return f.serviceStore.Register(svc)
    case "service_deregister":
        return f.serviceStore.Deregister(entry.Key)
    default:
        return fmt.Errorf("unknown log type: %s", entry.Type)
    }
}

func (f *KonsulFSM) Snapshot() (raft.FSMSnapshot, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()

    // Create snapshot of current state
    snapshot := &KonsulSnapshot{
        kvData:      f.kvStore.Dump(),
        serviceData: f.serviceStore.Dump(),
    }
    return snapshot, nil
}

func (f *KonsulFSM) Restore(rc io.ReadCloser) error {
    defer rc.Close()

    var snapshot KonsulSnapshot
    if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
        return err
    }

    f.mu.Lock()
    defer f.mu.Unlock()

    // Restore state from snapshot
    f.kvStore.Restore(snapshot.kvData)
    f.serviceStore.Restore(snapshot.serviceData)
    return nil
}
```

**Handler Integration**:
```go
func (h *KVHandler) Set(c *fiber.Ctx) error {
    key := c.Params("key")
    value := c.Body()

    // Check if leader
    if !h.raft.IsLeader() {
        // Redirect to leader
        leader := h.raft.Leader()
        return c.Redirect(fmt.Sprintf("https://%s/kv/%s", leader, key))
    }

    // Create log entry
    entry := LogEntry{
        Type:      "kv_set",
        Key:       key,
        Value:     value,
        Timestamp: time.Now(),
    }

    data, _ := json.Marshal(entry)

    // Apply to Raft (replicate + commit)
    future := h.raft.Apply(data, 5*time.Second)
    if err := future.Error(); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.SendStatus(200)
}
```

### Phase 2: Cluster Management API (1-2 weeks)

**Endpoints**:
```
GET    /cluster/status       - Cluster health and members
GET    /cluster/leader       - Current leader info
POST   /cluster/join         - Join cluster
DELETE /cluster/leave/:id    - Remove node
GET    /cluster/peers        - List all peers
GET    /cluster/config       - Raft configuration
POST   /cluster/snapshot     - Trigger snapshot
GET    /cluster/stats        - Raft statistics
```

**CLI Commands**:
```bash
# Cluster management
konsulctl cluster status
konsulctl cluster leader
konsulctl cluster join --address node2:7000
konsulctl cluster leave --node node-3
konsulctl cluster peers

# Diagnostics
konsulctl cluster stats
konsulctl cluster health
```

### Phase 3: Rolling Upgrades & Autopilot (2 weeks)

**Autopilot Features** (from hashicorp/raft-autopilot):
- Automated dead server cleanup
- Server health monitoring
- Safe node addition/removal
- Redundancy zone awareness

### Phase 4: Monitoring & Observability (1 week)

**Metrics**:
```
konsul_raft_state{state="leader|follower|candidate"}
konsul_raft_leader_changes_total
konsul_raft_commit_time_seconds
konsul_raft_apply_time_seconds
konsul_raft_log_entries_total
konsul_raft_log_size_bytes
konsul_raft_snapshot_duration_seconds
konsul_raft_peers
konsul_raft_last_contact_seconds
```

**Health Checks**:
- Leader election time
- Replication lag
- Peer connectivity
- Quorum status

### Configuration

```bash
# Enable clustering
KONSUL_CLUSTER_ENABLED=true

# Node identity
KONSUL_NODE_ID=node-1
KONSUL_NODE_NAME=konsul-1

# Raft configuration
KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000
KONSUL_RAFT_ADVERTISE_ADDR=10.0.1.10:7000
KONSUL_RAFT_DATA_DIR=./raft-data
KONSUL_RAFT_BOOTSTRAP=true  # Only for first node

# Join existing cluster
KONSUL_RAFT_JOIN=node1:7000,node2:7000

# Timeouts
KONSUL_RAFT_ELECTION_TIMEOUT=1s
KONSUL_RAFT_HEARTBEAT_TIMEOUT=1s
KONSUL_RAFT_LEADER_LEASE_TIMEOUT=500ms

# Snapshots
KONSUL_RAFT_SNAPSHOT_INTERVAL=120s
KONSUL_RAFT_SNAPSHOT_THRESHOLD=8192
```

### Deployment Patterns

**3-Node Cluster (Minimum)**:
```
Node 1: DC1-AZ1 (Leader)
Node 2: DC1-AZ2 (Follower)
Node 3: DC1-AZ3 (Follower)
```

**5-Node Cluster (Recommended)**:
```
Node 1: DC1-AZ1 (Leader)
Node 2: DC1-AZ1 (Follower)
Node 3: DC1-AZ2 (Follower)
Node 4: DC1-AZ2 (Follower)
Node 5: DC1-AZ3 (Follower)
```

### Testing Strategy

**Unit Tests**:
- FSM apply logic
- Snapshot/restore
- Log entry serialization

**Integration Tests**:
- 3-node cluster bootstrap
- Leader election
- Write replication
- Node failure scenarios
- Network partition handling

**Chaos Tests**:
- Kill leader during write
- Network partition (split-brain)
- Disk full scenarios
- Clock skew

### Performance Targets

- **Write latency**: <10ms (p99)
- **Leader election**: <300ms
- **Replication lag**: <50ms
- **Read latency (leader)**: <2ms
- **Read latency (follower)**: <1ms

### Security Considerations

- TLS for Raft transport
- Mutual TLS between nodes
- Cluster join tokens
- Network ACLs between nodes
- Encrypted snapshots

### Migration Path

**From Single Node → Cluster**:
1. Deploy 3 new nodes with clustering enabled
2. Bootstrap first node
3. Join second and third nodes
4. Export data from old single node
5. Import into cluster leader
6. Switch traffic to cluster
7. Decommission old node

## References

- [Raft Paper (In Search of an Understandable Consensus Algorithm)](https://raft.github.io/raft.pdf)
- [HashiCorp Raft Library](https://github.com/hashicorp/raft)
- [Consul Architecture](https://www.consul.io/docs/architecture)
- [etcd Raft Implementation](https://etcd.io/docs/latest/learning/design-learner/)
- [Designing Data-Intensive Applications (Chapter 9)](https://dataintensive.net/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-09 | Konsul Team | Initial proposal |
