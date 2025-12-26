# ADR-0030: Raft Clustering Implementation Status

**Date**: 2025-12-17

**Status**: In Progress

**Deciders**: Konsul Core Team

**Tags**: clustering, raft, implementation, high-availability

**Related**: [ADR-0011: Raft Consensus for Clustering and High Availability](0011-raft-clustering-ha.md)

## Context

This ADR documents the implementation progress of Raft clustering as proposed in ADR-0011. It serves as a living document tracking what has been implemented, what remains, and any deviations from the original design.

## Implementation Status

### Completed Components

#### 1. Core Raft Infrastructure ✅

**Location**: `/internal/raft/`

| File | Purpose | Status |
|------|---------|--------|
| `node.go` | Central Raft node wrapper (553 lines) | ✅ Complete |
| `fsm.go` | Finite State Machine implementation (268 lines) | ✅ Complete |
| `config.go` | Configuration management (121 lines) | ✅ Complete |
| `commands.go` | Raft command types and payloads (114 lines) | ✅ Complete |
| `errors.go` | Error definitions (29 lines) | ✅ Complete |
| `store_interfaces.go` | FSM store interfaces (49 lines) | ✅ Complete |
| `metrics.go` | Prometheus metrics (288 lines) | ✅ Complete |

**FSM Commands Implemented**:
- `CmdKVSet` - Set KV pair
- `CmdKVDelete` - Delete KV key
- `CmdKVSetWithFlags` - Set KV with flags
- `CmdKVBatchSet` - Batch set KV pairs
- `CmdKVBatchDelete` - Batch delete KV keys
- `CmdServiceRegister` - Register service
- `CmdServiceDeregister` - Deregister service
- `CmdServiceHeartbeat` - Service heartbeat

#### 2. Configuration System ✅

**Location**: `/internal/config/config.go`

**RaftConfig Fields**:
```go
type RaftConfig struct {
    Enabled            bool
    NodeID             string
    BindAddr           string
    AdvertiseAddr      string
    DataDir            string
    Bootstrap          bool
    HeartbeatTimeout   time.Duration
    ElectionTimeout    time.Duration
    LeaderLeaseTimeout time.Duration
    CommitTimeout      time.Duration
    SnapshotInterval   time.Duration
    SnapshotThreshold  uint64
    SnapshotRetention  int
    MaxAppendEntries   int
    TrailingLogs       uint64
    LogLevel           string
}
```

**Environment Variables**:
```bash
KONSUL_RAFT_ENABLED=true
KONSUL_RAFT_NODE_ID=node1
KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000
KONSUL_RAFT_ADVERTISE_ADDR=192.168.1.10:7000
KONSUL_RAFT_DATA_DIR=./data/raft
KONSUL_RAFT_BOOTSTRAP=true
KONSUL_RAFT_HEARTBEAT_TIMEOUT=1s
KONSUL_RAFT_ELECTION_TIMEOUT=1s
KONSUL_RAFT_LEADER_LEASE_TIMEOUT=500ms
KONSUL_RAFT_COMMIT_TIMEOUT=50ms
KONSUL_RAFT_SNAPSHOT_INTERVAL=120s
KONSUL_RAFT_SNAPSHOT_THRESHOLD=8192
KONSUL_RAFT_SNAPSHOT_RETENTION=2
KONSUL_RAFT_MAX_APPEND_ENTRIES=64
KONSUL_RAFT_TRAILING_LOGS=10240
KONSUL_RAFT_LOG_LEVEL=info
```

#### 3. Cluster Management API ✅

**Location**: `/internal/handlers/cluster.go`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/cluster/status` | GET | Full cluster diagnostics |
| `/cluster/leader` | GET | Current leader info |
| `/cluster/peers` | GET | List cluster members |
| `/cluster/join` | POST | Add new node |
| `/cluster/leave/:id` | DELETE | Remove node |
| `/cluster/snapshot` | POST | Trigger manual snapshot |

#### 4. Main Server Integration ✅

**Location**: `/cmd/konsul/main.go`

- Raft node initialization when `cfg.Raft.Enabled` is true
- Cluster handler routes registration
- Graceful Raft shutdown on server stop
- Leader election wait with timeout (30s)

#### 5. Handler Integration ✅

**KV Handler** (`/internal/handlers/kv.go`):
- `NewKVHandlerWithRaft()` constructor
- Leader check on write operations (Set, Delete)
- Raft-based writes via `raftNode.KVSet()`, `raftNode.KVDelete()`
- Non-leader returns 307 Temporary Redirect with `leader_addr`

**Service Handler** (`/internal/handlers/service.go`):
- `NewServiceHandlerWithRaft()` constructor
- Leader check on write operations (Register, Deregister, Heartbeat)
- Raft-based writes via `raftNode.ServiceRegister()`, etc.
- Non-leader returns 307 Temporary Redirect with `leader_addr`

#### 6. Snapshot Support ✅

- `Snapshot()` method on FSM
- `Restore()` method on FSM
- JSON-based snapshot format
- File-based snapshot store

#### 7. Metrics ✅

**State Metrics**:
- `konsul_raft_state` (gauge)
- `konsul_raft_is_leader` (0/1)
- `konsul_raft_peers_total`
- `konsul_raft_last_index`
- `konsul_raft_commit_index`
- `konsul_raft_applied_index`
- `konsul_raft_fsm_pending`
- `konsul_raft_replication_lag`

**Operation Counters**:
- `konsul_raft_apply_total` (by command_type)
- `konsul_raft_apply_errors_total` (by command_type, error_type)
- `konsul_raft_leader_changes_total`
- `konsul_raft_snapshot_total`
- `konsul_raft_restore_total`

**Latency Histograms**:
- `konsul_raft_apply_duration_seconds`
- `konsul_raft_commit_duration_seconds`
- `konsul_raft_snapshot_duration_seconds`

#### 8. Unit Tests ✅

**Location**: `/internal/raft/`
- `config_test.go` - Configuration validation
- `commands_test.go` - Command serialization/deserialization
- `fsm_test.go` - FSM operations with mocks

---

### Remaining Work

#### 1. TLS/Security for Raft Transport ❌

**Priority**: High

**Description**: Raft peer communication is currently unencrypted.

**Tasks**:
- [ ] Implement mTLS for Raft transport
- [ ] Add certificate validation
- [ ] Implement join token authentication
- [ ] Add network ACLs between nodes
- [ ] Encrypted snapshots

**Implementation Notes**:
```go
// Example: TLS transport configuration
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    RootCAs:      caCertPool,
    ClientAuth:   tls.RequireAndVerifyClientCert,
    MinVersion:   tls.VersionTLS12,
}
transport := raft.NewNetworkTransportWithConfig(...)
```

#### 2. CAS Operations via Raft ❌

**Priority**: Medium

**Description**: Compare-and-swap operations currently bypass Raft and operate on local store only.

**Tasks**:
- [ ] Add `CmdKVSetCAS` command type
- [ ] Add `CmdKVDeleteCAS` command type
- [ ] Add `CmdServiceRegisterCAS` command type
- [ ] Implement linearizable CAS semantics
- [ ] Update handlers to use Raft for CAS

**Complexity**: High - requires careful handling of index synchronization across nodes.

#### 3. Automatic Cluster Discovery ❌

**Priority**: Medium

**Description**: Nodes must currently be manually joined via HTTP API.

**Tasks**:
- [ ] Implement DNS-based discovery
- [ ] Implement cloud provider integration (AWS, GCP, Azure)
- [ ] Add `KONSUL_RAFT_JOIN` environment variable for auto-join
- [ ] Retry logic for join failures

#### 4. Autopilot Features ❌

**Priority**: Medium

**Description**: Advanced cluster management automation.

**Tasks**:
- [ ] Integrate `hashicorp/raft-autopilot`
- [ ] Automated dead server cleanup
- [ ] Server health monitoring
- [ ] Safe node addition/removal
- [ ] Redundancy zone awareness

#### 5. Snapshot Recovery on Startup ❌

**Priority**: Medium

**Description**: Snapshot restoration not fully integrated into startup.

**Tasks**:
- [ ] Restore from latest snapshot on node restart
- [ ] Validate snapshot integrity
- [ ] Handle corrupt snapshot gracefully
- [ ] Cleanup of old snapshots beyond retention

#### 6. Split-Brain Protection ❌

**Priority**: High

**Description**: No safeguards against split-brain scenarios.

**Tasks**:
- [ ] Implement leadership lease enforcement
- [ ] Add quorum checks before writes
- [ ] Network partition detection
- [ ] Automatic recovery from partition

#### 7. Read Consistency Options ❌

**Priority**: Low

**Description**: Only eventual consistency reads available.

**Tasks**:
- [ ] Implement linearizable reads (via ReadIndex)
- [ ] Add `?consistency=strong|stale` query parameter
- [ ] Document trade-offs

#### 8. CLI Cluster Commands ❌

**Priority**: Low

**Description**: `konsulctl` lacks cluster management commands.

**Tasks**:
- [ ] `konsulctl cluster status`
- [ ] `konsulctl cluster join --address <addr>`
- [ ] `konsulctl cluster leave --node <id>`
- [ ] `konsulctl cluster peers`
- [ ] `konsulctl cluster snapshot`

#### 9. Batch Operations via Raft ❌

**Priority**: Low

**Description**: Batch handler doesn't use Raft for writes.

**Tasks**:
- [ ] Update `BatchKVSet` to use Raft
- [ ] Update `BatchKVDelete` to use Raft
- [ ] Update `BatchServiceRegister` to use Raft
- [ ] Update `BatchServiceDeregister` to use Raft

#### 10. Grafana Dashboards for Raft ❌

**Priority**: Low

**Description**: No visualization for Raft metrics.

**Tasks**:
- [ ] Create Raft cluster overview dashboard
- [ ] Create Raft operations dashboard
- [ ] Add alerting rules for cluster health

---

## Architecture Decisions Made During Implementation

### 1. Leader Forwarding vs HTTP Redirect

**Decision**: Return HTTP 307 (Temporary Redirect) with `leader_addr` in response body.

**Rationale**:
- Allows client to handle redirect logic
- Works with any HTTP client
- Provides visibility into cluster topology
- Avoids server-side HTTP forwarding complexity

**Alternative Considered**: Server-side forwarding to leader.
- Rejected due to added latency and complexity.

### 2. CAS Operations Not Replicated

**Decision**: CAS operations bypass Raft and operate on local store.

**Rationale**:
- CAS requires additional coordination complexity
- Index synchronization across nodes is non-trivial
- Can be added in future iteration

**Impact**: CAS operations only work correctly in standalone mode.

### 3. Snapshot Format

**Decision**: JSON-based snapshot format.

**Rationale**:
- Human-readable for debugging
- Easy to implement
- Compatible with existing store structures

**Trade-off**: Less efficient than binary format; acceptable for expected data sizes.

---

## Testing Recommendations

### Integration Tests Needed

1. **3-Node Cluster Bootstrap**
   - Start node1 with bootstrap=true
   - Join node2 and node3
   - Verify all nodes see same data

2. **Leader Failover**
   - Write data to leader
   - Kill leader
   - Verify new leader elected
   - Verify data accessible

3. **Write Replication**
   - Write via leader
   - Read from followers
   - Verify consistency

4. **Non-Leader Write Rejection**
   - Send write to follower
   - Verify 307 redirect response
   - Verify leader_addr in response

### Chaos Tests Needed

1. Kill leader during write operation
2. Network partition (isolate leader)
3. Slow disk on leader
4. Clock skew between nodes

---

## Migration Guide

### From Standalone to Cluster

1. **Prepare**:
   ```bash
   # Export existing data
   curl http://localhost:8500/export > backup.json
   ```

2. **Deploy First Node**:
   ```bash
   KONSUL_RAFT_ENABLED=true \
   KONSUL_RAFT_NODE_ID=node1 \
   KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000 \
   KONSUL_RAFT_ADVERTISE_ADDR=10.0.1.10:7000 \
   KONSUL_RAFT_BOOTSTRAP=true \
   ./konsul
   ```

3. **Join Additional Nodes**:
   ```bash
   # On node2
   KONSUL_RAFT_ENABLED=true \
   KONSUL_RAFT_NODE_ID=node2 \
   KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000 \
   KONSUL_RAFT_ADVERTISE_ADDR=10.0.1.11:7000 \
   ./konsul

   # Join to cluster
   curl -X POST http://10.0.1.11:8500/cluster/join \
     -H "Content-Type: application/json" \
     -d '{"node_id": "node2", "address": "10.0.1.11:7000"}'
   ```

4. **Import Data**:
   ```bash
   curl -X POST http://10.0.1.10:8500/import \
     -H "Content-Type: application/json" \
     -d @backup.json
   ```

---

## Performance Considerations

### Expected Latencies

| Operation | Standalone | Cluster (Leader) | Cluster (Follower) |
|-----------|------------|------------------|-------------------|
| KV Read | <1ms | <1ms | <1ms |
| KV Write | <1ms | 5-10ms | Redirect to leader |
| Service Register | <1ms | 5-10ms | Redirect to leader |

### Resource Requirements

- **Memory**: +50MB per node for Raft state
- **Disk**: Log and snapshot storage (~10x write amplification)
- **Network**: Heartbeats every 1s, replication on writes

---

## References

- [ADR-0011: Raft Consensus for Clustering and High Availability](0011-raft-clustering-ha.md)
- [HashiCorp Raft Library](https://github.com/hashicorp/raft)
- [Raft Paper](https://raft.github.io/raft.pdf)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-17 | Konsul Team | Initial implementation status document |