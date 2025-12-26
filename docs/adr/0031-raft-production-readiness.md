# ADR-0031: Raft Production Readiness and Phase 2 Implementation

**Date**: 2025-12-18

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: clustering, raft, production, security, testing, high-availability

**Related**: [ADR-0011: Raft Consensus for Clustering and High Availability](0011-raft-clustering-ha.md), [ADR-0030: Raft Integration Implementation Status](0030-raft-integration-implementation.md)

## Context

Phase 1 of Raft clustering (ADR-0030) successfully implemented:
- Core Raft infrastructure with FSM and commands
- Configuration system with environment variables
- Cluster management API endpoints
- KV and Service handler integration with leader redirection
- Snapshot support and Prometheus metrics
- Unit tests for core components

However, the current implementation is **not production-ready**. Critical gaps exist in:

### Security
- **No encryption**: Raft peer communication is plaintext
- **No authentication**: Any node can join the cluster
- **No authorization**: No join tokens or certificates

### Reliability
- **Split-brain vulnerability**: Insufficient safeguards against network partitions
- **No recovery**: Snapshot restoration not integrated into startup
- **Incomplete testing**: Missing integration and chaos tests
- **CAS operations broken**: Compare-and-swap bypasses Raft in cluster mode

### Operational Maturity
- **Manual operations**: No automatic discovery or dead server cleanup
- **Limited observability**: No Grafana dashboards for Raft metrics
- **Incomplete CLI**: konsulctl lacks cluster commands
- **Inconsistent consistency**: Only eventual reads available

### Production Requirements

For Konsul to be deployed in production clusters, we need:

1. **Zero-trust security**: mTLS between all nodes
2. **Automatic recovery**: From crashes, restarts, and partitions
3. **Comprehensive testing**: Integration, chaos, and failure scenario tests
4. **Operational automation**: Auto-discovery, autopilot, rolling upgrades
5. **Observability**: Dashboards, alerts, and diagnostics
6. **Data integrity**: CAS operations, linearizable reads, batch atomicity

## Decision

We will implement Phase 2 of Raft clustering in **three priority tiers** over multiple iterations.

### Priority Tier 1: Production Safety (Critical - Weeks 1-4)

Must be completed before production deployment. These features prevent data loss, security breaches, and operational failures.

#### 1.1 TLS/mTLS for Raft Transport

**Goal**: Encrypt and authenticate all Raft peer communication.

**Implementation**:
```go
type TLSConfig struct {
    Enabled            bool
    CertFile           string  // Server certificate
    KeyFile            string  // Private key
    CAFile             string  // CA certificate for peer verification
    VerifyPeerCert     bool    // Require client cert verification
    MinTLSVersion      string  // Default: "1.2"
    ServerName         string  // Expected server name in cert
}
```

**Environment Variables**:
```bash
KONSUL_RAFT_TLS_ENABLED=true
KONSUL_RAFT_TLS_CERT_FILE=/etc/konsul/certs/server.crt
KONSUL_RAFT_TLS_KEY_FILE=/etc/konsul/certs/server.key
KONSUL_RAFT_TLS_CA_FILE=/etc/konsul/certs/ca.crt
KONSUL_RAFT_TLS_VERIFY_PEER=true
KONSUL_RAFT_TLS_MIN_VERSION=1.2
```

**Implementation Steps**:
- Add TLS configuration to `internal/config/config.go`
- Create TLS transport wrapper in `internal/raft/transport.go`
- Implement certificate validation and rotation support
- Add certificate generation helper for development
- Document certificate management in operations guide

**Success Criteria**:
- All Raft traffic encrypted with TLS 1.2+
- Mutual authentication enforced
- Certificate validation prevents unauthorized nodes
- Graceful handling of cert rotation

#### 1.2 Join Token Authentication

**Goal**: Prevent unauthorized nodes from joining the cluster.

**Implementation**:
```go
type SecurityConfig struct {
    JoinToken         string  // Secret token required to join
    TokenHash         string  // Bcrypt hash of join token
    RequireToken      bool    // Enforce token on join
    TokenRotationDays int     // Auto-rotation period
}
```

**Flow**:
```
1. Admin generates join token via API/CLI
2. New node includes token in join request
3. Leader validates token before accepting
4. Token can be rotated without cluster restart
```

**API Changes**:
```bash
# Generate join token
POST /cluster/join-token
Response: {"token": "abc123...xyz", "expires_at": "2025-12-25T00:00:00Z"}

# Join with token
POST /cluster/join
Body: {
  "node_id": "node2",
  "address": "10.0.1.11:7000",
  "join_token": "abc123...xyz"
}
```

**Success Criteria**:
- Unauthenticated join requests rejected
- Token validation works across leader changes
- Token rotation doesn't disrupt cluster
- Audit log captures join attempts

#### 1.3 Split-Brain Protection

**Goal**: Prevent multiple leaders and data divergence during partitions.

**Implementation**:
```go
// Quorum checks before writes
func (n *RaftNode) applyWithQuorumCheck(cmd Command) error {
    // Ensure we still have quorum
    peers := n.raft.GetConfiguration().Latest().Servers
    if len(peers) < n.quorumSize() {
        return ErrNoQuorum
    }

    // Verify leadership with lease check
    if !n.raft.VerifyLeader().Error() {
        return ErrNotLeader
    }

    return n.raft.Apply(cmd, timeout)
}

// Leadership lease enforcement
type LeadershipLease struct {
    AcquiredAt    time.Time
    Duration      time.Duration
    HeartbeatOK   bool
    QuorumPeers   int
}
```

**Protection Mechanisms**:
- **Pre-write quorum checks**: Verify majority before accepting writes
- **Leadership lease**: Leader steps down if can't maintain quorum
- **Heartbeat monitoring**: Detect lost connectivity to majority
- **Automatic stepdown**: Leader demotes if partition detected
- **Read-index verification**: Ensure reads reflect committed state

**Configuration**:
```bash
KONSUL_RAFT_QUORUM_CHECK_ENABLED=true
KONSUL_RAFT_LEADERSHIP_LEASE_ENFORCEMENT=true
KONSUL_RAFT_STEPDOWN_ON_PARTITION=true
```

**Success Criteria**:
- No dual leaders during network partition
- Minority partition rejects writes
- Automatic recovery when partition heals
- Metrics track quorum status

#### 1.4 Snapshot Recovery on Startup

**Goal**: Automatically restore from latest snapshot when node restarts.

**Implementation**:
```go
func (n *RaftNode) Start() error {
    // 1. Check for existing snapshots
    snapshots, err := n.snapshotStore.List()
    if err != nil {
        return fmt.Errorf("failed to list snapshots: %w", err)
    }

    if len(snapshots) > 0 {
        latest := snapshots[0]

        // 2. Validate snapshot integrity
        if err := n.validateSnapshot(latest); err != nil {
            log.Warn("snapshot validation failed, using fallback", "err", err)
        } else {
            // 3. Restore from snapshot
            log.Info("restoring from snapshot", "id", latest.ID)
            if err := n.restoreSnapshot(latest); err != nil {
                return fmt.Errorf("snapshot restore failed: %w", err)
            }
        }
    }

    // 4. Cleanup old snapshots beyond retention
    n.cleanupOldSnapshots()

    // 5. Start Raft
    return n.raft.Start()
}

func (n *RaftNode) validateSnapshot(snap *raft.SnapshotMeta) error {
    // Verify checksum
    // Check size is reasonable
    // Validate JSON structure
    // Ensure index is valid
}
```

**Recovery Flow**:
```
Node Restart → List Snapshots → Validate Latest → Restore FSM →
Cleanup Old → Start Raft → Catch Up with Leader
```

**Configuration**:
```bash
KONSUL_RAFT_SNAPSHOT_RESTORE_ON_STARTUP=true
KONSUL_RAFT_SNAPSHOT_VALIDATION_ENABLED=true
KONSUL_RAFT_SNAPSHOT_CHECKSUM_ALGORITHM=sha256
```

**Success Criteria**:
- Node restarts restore latest valid snapshot
- Corrupt snapshots detected and skipped
- Old snapshots cleaned up automatically
- Metrics track restoration success/failure

#### 1.5 Integration Testing Suite

**Goal**: Comprehensive tests covering real-world cluster scenarios.

**Test Categories**:

**1. Cluster Formation Tests** (`internal/raft/integration/cluster_test.go`):
```go
func TestThreeNodeClusterBootstrap(t *testing.T)
func TestFiveNodeClusterBootstrap(t *testing.T)
func TestNodeJoinExistingCluster(t *testing.T)
func TestNodeLeaveGracefully(t *testing.T)
func TestBootstrapWithTLSEnabled(t *testing.T)
```

**2. Leader Election Tests** (`internal/raft/integration/election_test.go`):
```go
func TestLeaderElectionOnBootstrap(t *testing.T)
func TestLeaderFailoverWithinTimeout(t *testing.T)
func TestNoLeaderWhenMajorityDown(t *testing.T)
func TestLeaderStepDownOnQuorumLoss(t *testing.T)
func TestPreventSplitBrainDuringPartition(t *testing.T)
```

**3. Data Replication Tests** (`internal/raft/integration/replication_test.go`):
```go
func TestKVWriteReplicatesToFollowers(t *testing.T)
func TestServiceRegisterReplicatesToFollowers(t *testing.T)
func TestBatchOperationsAtomic(t *testing.T)
func TestReadYourWritesConsistency(t *testing.T)
func TestNoDataLossOnLeaderCrash(t *testing.T)
```

**4. Snapshot Tests** (`internal/raft/integration/snapshot_test.go`):
```go
func TestSnapshotCreatedAtThreshold(t *testing.T)
func TestSnapshotRestoreOnRestart(t *testing.T)
func TestSnapshotTransferToNewNode(t *testing.T)
func TestCorruptSnapshotHandling(t *testing.T)
```

**5. Non-Leader Redirect Tests** (`internal/raft/integration/redirect_test.go`):
```go
func TestFollowerReturns307OnWrite(t *testing.T)
func TestLeaderAddrInRedirectResponse(t *testing.T)
func TestClientFollowsRedirect(t *testing.T)
func TestReadFromFollowerSucceeds(t *testing.T)
```

**6. Chaos/Failure Tests** (`internal/raft/integration/chaos_test.go`):
```go
func TestKillLeaderDuringWrite(t *testing.T)
func TestNetworkPartitionIsolatesLeader(t *testing.T)
func TestSlowDiskOnLeader(t *testing.T)
func TestClockSkewBetweenNodes(t *testing.T)
func TestRollingRestartWithoutDowntime(t *testing.T)
func TestMultipleSimultaneousFailures(t *testing.T)
```

**Test Infrastructure**:
```go
// Helper to create test cluster
type TestCluster struct {
    Nodes    []*RaftNode
    Leader   *RaftNode
    DataDirs []string
}

func NewTestCluster(size int, opts ...Option) (*TestCluster, error)
func (tc *TestCluster) WaitForLeader(timeout time.Duration) error
func (tc *TestCluster) KillNode(id string) error
func (tc *TestCluster) PartitionNetwork(group1, group2 []string) error
func (tc *TestCluster) HealPartition() error
func (tc *TestCluster) Cleanup() error
```

**Success Criteria**:
- 100% pass rate on all integration tests
- Tests complete in <5 minutes total
- Cover all critical failure scenarios
- Tests run in CI/CD pipeline

---

### Priority Tier 2: Correctness & Consistency (Important - Weeks 5-8)

Ensures data operations work correctly in cluster mode.

#### 2.1 CAS Operations via Raft

**Goal**: Make compare-and-swap operations work correctly in cluster mode.

**Current Problem**:
```go
// Current implementation - WRONG in cluster mode
func (h *KVHandler) CompareAndSwap(c *fiber.Ctx) error {
    // Reads and writes local store, bypasses Raft
    // Race conditions across nodes!
    return h.store.CompareAndSwap(key, expected, new)
}
```

**New Implementation**:
```go
// Add CAS commands
const (
    CmdKVSetCAS       CommandType = "kv_set_cas"
    CmdKVDeleteCAS    CommandType = "kv_delete_cas"
)

type KVSetCASPayload struct {
    Key           string
    Value         []byte
    ExpectedIndex uint64  // Must match current index
    Flags         uint64
}

// FSM applies CAS atomically
func (f *FSM) applyKVSetCAS(payload KVSetCASPayload) error {
    current := f.kvStore.Get(payload.Key)

    // Check index matches
    if current.ModifyIndex != payload.ExpectedIndex {
        return ErrCASConflict  // Index mismatch
    }

    // Apply update atomically
    return f.kvStore.Set(payload.Key, payload.Value)
}

// Handler routes through Raft
func (h *KVHandler) CompareAndSwap(c *fiber.Ctx) error {
    if h.raftNode == nil {
        // Standalone mode - use local store
        return h.store.CompareAndSwap(key, expected, new)
    }

    // Cluster mode - apply via Raft
    return h.raftNode.KVSetCAS(key, value, expectedIndex)
}
```

**Index Synchronization**:
- Each KV entry has `ModifyIndex` (Raft log index)
- CAS checks index matches before update
- Atomic check-and-set in FSM prevents races
- Linearizable semantics guaranteed

**Success Criteria**:
- CAS operations work correctly in cluster mode
- No race conditions across nodes
- Index conflicts detected reliably
- Performance acceptable (<20ms p99)

#### 2.2 Batch Operations via Raft

**Goal**: Make batch operations atomic via Raft.

**Commands**:
```go
const (
    CmdKVBatchSetWithRaft    CommandType = "kv_batch_set_raft"
    CmdKVBatchDeleteWithRaft CommandType = "kv_batch_delete_raft"
)

type KVBatchSetPayload struct {
    Operations []KVSetOperation
    Atomic     bool  // All-or-nothing
}

// FSM applies batch atomically
func (f *FSM) applyKVBatchSet(payload KVBatchSetPayload) error {
    if payload.Atomic {
        // Start transaction
        txn := f.kvStore.BeginTxn()
        defer txn.Rollback()

        for _, op := range payload.Operations {
            if err := txn.Set(op.Key, op.Value); err != nil {
                return err  // Rollback all
            }
        }

        return txn.Commit()  // Commit all
    }

    // Best-effort mode
    for _, op := range payload.Operations {
        f.kvStore.Set(op.Key, op.Value)  // Continue on error
    }
    return nil
}
```

**Handler Integration**:
```go
func (h *BatchHandler) SetMultiple(c *fiber.Ctx) error {
    if h.raftNode != nil {
        // Cluster mode - use Raft
        return h.raftNode.KVBatchSet(operations, atomic)
    }
    // Standalone mode
    return h.batchStore.SetMultiple(operations)
}
```

**Success Criteria**:
- Batch operations are atomic
- Partial failures handled correctly
- Performance better than N individual operations

#### 2.3 Linearizable Reads (ReadIndex)

**Goal**: Provide strongly consistent reads via Raft ReadIndex.

**Implementation**:
```go
// Add consistency parameter
func (h *KVHandler) Get(c *fiber.Ctx) error {
    consistency := c.Query("consistency", "stale")  // stale|strong

    if consistency == "strong" && h.raftNode != nil {
        // Wait for ReadIndex
        if err := h.raftNode.BarrierRead(); err != nil {
            return err
        }
    }

    // Read from local FSM (now guaranteed consistent)
    return h.store.Get(key)
}

// RaftNode implements ReadIndex
func (n *RaftNode) BarrierRead() error {
    // Verify leadership with ReadIndex
    future := n.raft.VerifyLeader()
    if err := future.Error(); err != nil {
        return ErrNotLeader
    }

    // Wait for FSM to apply all committed entries
    return n.raft.Barrier(0).Error()
}
```

**Query Modes**:
- `?consistency=stale`: Fast, eventual consistency (default)
- `?consistency=strong`: Slow, linearizable reads via ReadIndex
- `?consistency=leader`: Only read from leader

**Trade-offs**:
- Strong reads: +5-10ms latency, guaranteed up-to-date
- Stale reads: <1ms, may be slightly behind

**Success Criteria**:
- Strong reads reflect all committed writes
- Stale reads document staleness bounds
- Performance matches expectations

---

### Priority Tier 3: Operational Excellence (Nice-to-Have - Weeks 9-12)

Improves operator experience but not critical for launch.

#### 3.1 Automatic Cluster Discovery

**Goal**: Nodes automatically find and join cluster without manual API calls.

**Strategies**:

**DNS-Based Discovery**:
```bash
KONSUL_RAFT_DISCOVERY_METHOD=dns
KONSUL_RAFT_DISCOVERY_DNS_DOMAIN=konsul.service.consul
KONSUL_RAFT_DISCOVERY_DNS_PORT=7000

# Looks up SRV records:
# _konsul._tcp.konsul.service.consul
```

**Static Seed List**:
```bash
KONSUL_RAFT_DISCOVERY_METHOD=static
KONSUL_RAFT_DISCOVERY_SEEDS=10.0.1.10:7000,10.0.1.11:7000,10.0.1.12:7000
```

**Cloud Provider APIs**:
```bash
KONSUL_RAFT_DISCOVERY_METHOD=aws
KONSUL_RAFT_DISCOVERY_AWS_REGION=us-east-1
KONSUL_RAFT_DISCOVERY_AWS_TAG_KEY=konsul-cluster
KONSUL_RAFT_DISCOVERY_AWS_TAG_VALUE=prod
```

**Auto-Join Flow**:
```
1. Node starts with discovery config
2. Query discovery service for peers
3. Attempt join to each peer until success
4. Retry with exponential backoff on failure
5. Log discovery attempts for debugging
```

**Success Criteria**:
- Nodes join automatically on startup
- Discovery failures don't prevent startup
- Retries work across discovery methods

#### 3.2 Autopilot (Dead Server Cleanup)

**Goal**: Automatically remove failed nodes from cluster.

**Integration**:
```bash
go get github.com/hashicorp/raft-autopilot
```

**Configuration**:
```go
type AutopilotConfig struct {
    Enabled                  bool
    CleanupDeadServers       bool
    LastContactThreshold     time.Duration  // 10s
    MaxTrailingLogs          uint64         // 1000
    ServerStabilizationTime  time.Duration  // 10s
}
```

**Features**:
- **Dead server cleanup**: Auto-remove nodes that fail health checks
- **Stable server tracking**: Wait for stabilization before promoting
- **Health monitoring**: Track last contact, log lag, version
- **Safe removal**: Only remove if quorum maintained

**Success Criteria**:
- Dead nodes removed within 60s
- Cluster maintains quorum during cleanup
- Metrics track autopilot actions

#### 3.3 CLI Cluster Commands

**Goal**: Add cluster management to konsulctl.

**Commands**:
```bash
# View cluster status
konsulctl cluster status
konsulctl cluster leader
konsulctl cluster peers

# Manage membership
konsulctl cluster join --node-id node2 --address 10.0.1.11:7000
konsulctl cluster leave --node-id node2

# Operations
konsulctl cluster snapshot
konsulctl cluster transfer-leadership --to node3

# Generate join tokens
konsulctl cluster generate-token --ttl 24h
```

**Implementation** (`cmd/konsulctl/cluster.go`):
```go
var clusterCmd = &cobra.Command{
    Use:   "cluster",
    Short: "Cluster management commands",
}

var clusterStatusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show cluster status",
    Run: func(cmd *cobra.Command, args []string) {
        // GET /cluster/status
    },
}
```

**Success Criteria**:
- All cluster operations available via CLI
- Output formatted as table or JSON
- Error messages are helpful

#### 3.4 Grafana Dashboards for Raft

**Goal**: Visualize Raft cluster health and operations.

**Dashboards**:

**1. Raft Cluster Overview** (`docs/grafana/raft-overview.json`):
- Current leader and term
- Cluster size and voter count
- Leader election frequency
- Overall cluster health gauge

**2. Raft Operations** (`docs/grafana/raft-operations.json`):
- Apply operations by command type
- Apply latency percentiles (p50, p95, p99)
- Commit latency over time
- Apply errors by type

**3. Raft Replication** (`docs/grafana/raft-replication.json`):
- Last/Commit/Applied index by node
- Replication lag by node
- Snapshot frequency and duration
- FSM pending operations

**Alerts**:
```yaml
- alert: RaftNoLeader
  expr: sum(konsul_raft_is_leader) == 0
  for: 30s

- alert: RaftFrequentElections
  expr: increase(konsul_raft_leader_changes_total[5m]) > 3

- alert: RaftHighReplicationLag
  expr: konsul_raft_replication_lag > 1000
```

**Success Criteria**:
- Dashboards load in Grafana 9+
- All metrics display correctly
- Alerts fire on actual issues

---

## Implementation Roadmap

### Phase 2A: Security & Reliability (Weeks 1-4)
**Deliverables**:
- TLS/mTLS transport with certificate validation
- Join token authentication system
- Split-brain protection mechanisms
- Snapshot recovery on startup
- Integration test suite (50+ tests)

**Exit Criteria**:
- All Tier 1 features complete and tested
- Integration tests achieve 100% pass rate
- Security audit passes
- Documentation updated

### Phase 2B: Correctness (Weeks 5-8)
**Deliverables**:
- CAS operations via Raft
- Batch operations atomicity
- Linearizable reads with ReadIndex
- Performance benchmarks

**Exit Criteria**:
- All data operations correct in cluster mode
- Performance within 2x of standalone
- Benchmark suite established

### Phase 2C: Operations (Weeks 9-12)
**Deliverables**:
- Automatic cluster discovery (3 methods)
- Autopilot integration
- CLI cluster commands
- Grafana dashboards and alerts

**Exit Criteria**:
- Zero-touch cluster formation works
- Dead nodes auto-removed
- Complete observability

---

## Alternatives Considered

### Alternative 1: etcd Instead of Raft

**Pros**:
- Mature, battle-tested
- Strong consistency by default
- Rich client libraries

**Cons**:
- External dependency (not embedded)
- Higher operational complexity
- Different API paradigm
- Harder to customize

**Reason for rejection**: We want embedded clustering, not external coordination service.

### Alternative 2: Multi-Paxos

**Pros**:
- Proven algorithm
- Flexible configuration

**Cons**:
- More complex than Raft
- Fewer production-ready implementations
- Harder to understand and debug

**Reason for rejection**: Raft is simpler and better supported.

### Alternative 3: Eventually Consistent (Gossip)

**Pros**:
- High availability
- Simple to implement
- No leader bottleneck

**Cons**:
- No strong consistency
- Conflict resolution required
- Not suitable for KV store semantics

**Reason for rejection**: Need strong consistency for KV operations.

### Alternative 4: Skip TLS, Use VPN/Firewall

**Pros**:
- Simpler configuration
- Rely on network security

**Cons**:
- Defense-in-depth violation
- Insider threat risk
- Compliance issues

**Reason for rejection**: Security best practices require encryption.

---

## Consequences

### Positive

**Security**:
- Zero-trust architecture with mTLS
- Authenticated cluster membership
- Encrypted data in transit
- Audit trail of cluster changes

**Reliability**:
- Automatic recovery from failures
- No split-brain scenarios
- Data loss prevention
- Comprehensive test coverage

**Correctness**:
- CAS operations work correctly
- Batch operations are atomic
- Read consistency options available
- Linearizable semantics where needed

**Operations**:
- Zero-touch cluster formation
- Automatic dead server cleanup
- Rich observability with dashboards
- Complete CLI tooling

**Production Readiness**:
- Meets enterprise security requirements
- Suitable for mission-critical workloads
- SLA-compliant failover times
- Professional-grade monitoring

### Negative

**Complexity**:
- TLS certificate management overhead
- More configuration options to learn
- Harder to debug distributed issues
- Steeper learning curve for operators

**Performance**:
- TLS adds ~1-2ms latency
- Linearizable reads slower than stale
- Quorum checks add overhead
- Snapshots consume disk I/O

**Dependencies**:
- Requires raft-autopilot library
- Certificate infrastructure needed
- Discovery service (DNS/cloud API)
- More external components

**Testing**:
- Integration tests take longer to run
- Chaos tests require special infrastructure
- More CI/CD resources needed
- Harder to reproduce bugs

### Neutral

**Migration Path**:
- Existing clusters must upgrade carefully
- Certificate rollout process required
- May need cluster rebuild for TLS
- Documentation updates extensive

**Timeline**:
- 12 weeks for complete implementation
- Incremental rollout possible
- Can prioritize critical features
- Some features optional (Tier 3)

---

## Testing Strategy

### Unit Tests
- All new components (TLS, CAS, batch)
- Mock Raft for handler tests
- Configuration validation
- Error handling paths

### Integration Tests
- Multi-node cluster scenarios
- Leader election and failover
- Data replication verification
- Snapshot and recovery
- TLS handshake and auth

### Chaos Tests
- Network partitions
- Node crashes during operations
- Slow disk simulation
- Clock skew
- Simultaneous failures

### Performance Tests
- Baseline: standalone mode
- Cluster write latency (p50, p95, p99)
- Cluster read latency by consistency mode
- Replication throughput
- Snapshot performance

### Security Tests
- TLS version enforcement
- Certificate validation
- Join token authentication
- Unauthorized access prevention
- Certificate rotation

---

## Migration Guide

### From Phase 1 to Phase 2A (TLS)

**1. Generate Certificates**:
```bash
# Use provided script
./scripts/generate-certs.sh --cluster-name prod --nodes 3

# Outputs:
# certs/ca.crt
# certs/node1.crt, certs/node1.key
# certs/node2.crt, certs/node2.key
# certs/node3.crt, certs/node3.key
```

**2. Distribute Certificates**:
```bash
# Copy to each node
scp certs/ca.crt node1:/etc/konsul/certs/
scp certs/node1.* node1:/etc/konsul/certs/

scp certs/ca.crt node2:/etc/konsul/certs/
scp certs/node2.* node2:/etc/konsul/certs/
```

**3. Update Configuration**:
```bash
# Add to environment on each node
KONSUL_RAFT_TLS_ENABLED=true
KONSUL_RAFT_TLS_CERT_FILE=/etc/konsul/certs/node1.crt
KONSUL_RAFT_TLS_KEY_FILE=/etc/konsul/certs/node1.key
KONSUL_RAFT_TLS_CA_FILE=/etc/konsul/certs/ca.crt
KONSUL_RAFT_TLS_VERIFY_PEER=true
```

**4. Rolling Restart**:
```bash
# Restart followers first
systemctl restart konsul  # on node3
sleep 30
systemctl restart konsul  # on node2
sleep 30
systemctl restart konsul  # on node1 (leader)
```

**5. Verify TLS**:
```bash
konsulctl cluster status | grep -i tls
# Should show: "tls_enabled": true
```

---

## Rollback Plan

If Phase 2 causes issues:

**1. Immediate Rollback**:
```bash
# Revert to Phase 1 binary
git checkout raft-1
make build
systemctl restart konsul
```

**2. TLS Issues**:
```bash
# Disable TLS temporarily
KONSUL_RAFT_TLS_ENABLED=false
systemctl restart konsul
```

**3. Data Recovery**:
```bash
# Restore from snapshot
konsulctl cluster snapshot restore --file latest.snap
```

**4. Complete Rebuild**:
```bash
# Export data
konsulctl kv export > backup.json

# Rebuild cluster
# Re-import data
konsulctl kv import < backup.json
```

---

## Success Metrics

### Security Metrics
- **TLS Coverage**: 100% of Raft traffic encrypted
- **Join Attempts**: 0 unauthorized successful joins
- **Certificate Validation**: 100% of peers verified

### Reliability Metrics
- **MTTR**: Mean time to recovery <30s
- **Split-Brain Events**: 0 dual leaders
- **Data Loss Events**: 0 on single node failure
- **Test Pass Rate**: 100% integration tests

### Performance Metrics
- **Write Latency**: p99 <20ms (within datacenter)
- **Read Latency (Stale)**: p99 <2ms
- **Read Latency (Strong)**: p99 <15ms
- **Replication Lag**: p99 <100 entries

### Operational Metrics
- **Auto-Discovery Success**: >95%
- **Dead Server Cleanup**: <60s
- **Dashboard Coverage**: 100% of Raft metrics
- **CLI Coverage**: 100% of API operations

---

## Dependencies

### External Libraries
- `github.com/hashicorp/raft` (existing)
- `github.com/hashicorp/raft-autopilot` (new)
- Standard library `crypto/tls` (new)

### Infrastructure
- Certificate authority (CA) for TLS
- DNS server for discovery (optional)
- Cloud provider API access (optional)
- Grafana for dashboards (existing)

### Documentation
- TLS setup guide
- Certificate management guide
- Cluster operations runbook
- Troubleshooting guide
- Security best practices

---

## References

- [ADR-0011: Raft Consensus for Clustering and High Availability](0011-raft-clustering-ha.md)
- [ADR-0030: Raft Integration Implementation Status](0030-raft-integration-implementation.md)
- [HashiCorp Raft Library](https://github.com/hashicorp/raft)
- [Raft Paper](https://raft.github.io/raft.pdf)
- [Raft Autopilot](https://github.com/hashicorp/raft-autopilot)
- [Consul Raft Implementation](https://www.consul.io/docs/architecture/consensus)
- [etcd Raft Implementation](https://etcd.io/docs/latest/learning/design-learner/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-18 | Konsul Core Team | Initial Phase 2 planning document |