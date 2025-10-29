# Raft Consensus Implementation Tasks

**Based on**: ADR-0011 (Raft Consensus for Clustering and High Availability)
**Created**: 2025-10-28
**Status**: Planning

This document breaks down the Raft consensus implementation into actionable tasks with clear acceptance criteria, dependencies, and time estimates.

---

## Phase 1: Core Raft Integration (3-4 weeks)

### 1.1 Setup & Dependencies

#### Task 1.1.1: Add Raft Dependencies
**Priority**: P0 (Critical Path)
**Estimated Time**: 2 hours
**Dependencies**: None

**Description**:
Add HashiCorp Raft and related dependencies to the project.

**Acceptance Criteria**:
- [ ] Add `github.com/hashicorp/raft` to go.mod
- [ ] Add `github.com/hashicorp/raft-boltdb/v2` to go.mod
- [ ] Run `go mod tidy` successfully
- [ ] Update go.mod and go.sum committed

**Commands**:
```bash
go get github.com/hashicorp/raft
go get github.com/hashicorp/raft-boltdb/v2
go mod tidy
```

---

#### Task 1.1.2: Create Raft Configuration Structure
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.1.1

**Description**:
Define configuration structures for Raft in the existing config package.

**Acceptance Criteria**:
- [ ] Add `RaftConfig` struct to `internal/config/config.go`
- [ ] Include fields: NodeID, BindAddr, AdvertiseAddr, DataDir, Bootstrap, JoinAddresses
- [ ] Include timeout configs: ElectionTimeout, HeartbeatTimeout, LeaderLeaseTimeout
- [ ] Include snapshot configs: SnapshotInterval, SnapshotThreshold
- [ ] Add validation logic for Raft config
- [ ] Add environment variable mappings (KONSUL_RAFT_*)
- [ ] Update config tests

**File**: `internal/config/config.go`

**Example Structure**:
```go
type RaftConfig struct {
    Enabled             bool          `mapstructure:"enabled"`
    NodeID              string        `mapstructure:"node_id"`
    NodeName            string        `mapstructure:"node_name"`
    BindAddr            string        `mapstructure:"bind_addr"`
    AdvertiseAddr       string        `mapstructure:"advertise_addr"`
    DataDir             string        `mapstructure:"data_dir"`
    Bootstrap           bool          `mapstructure:"bootstrap"`
    JoinAddresses       []string      `mapstructure:"join_addresses"`
    ElectionTimeout     time.Duration `mapstructure:"election_timeout"`
    HeartbeatTimeout    time.Duration `mapstructure:"heartbeat_timeout"`
    LeaderLeaseTimeout  time.Duration `mapstructure:"leader_lease_timeout"`
    SnapshotInterval    time.Duration `mapstructure:"snapshot_interval"`
    SnapshotThreshold   uint64        `mapstructure:"snapshot_threshold"`
}
```

---

### 1.2 Raft Core Implementation

#### Task 1.2.1: Create Raft Package Structure
**Priority**: P0
**Estimated Time**: 2 hours
**Dependencies**: 1.1.2

**Description**:
Create the package structure for Raft implementation.

**Acceptance Criteria**:
- [ ] Create `internal/raft/` directory
- [ ] Create `internal/raft/node.go` - Raft node wrapper
- [ ] Create `internal/raft/fsm.go` - Finite State Machine
- [ ] Create `internal/raft/snapshot.go` - Snapshot implementation
- [ ] Create `internal/raft/transport.go` - Network transport setup
- [ ] Create `internal/raft/errors.go` - Raft-specific errors

**Files to Create**:
- `internal/raft/node.go`
- `internal/raft/fsm.go`
- `internal/raft/snapshot.go`
- `internal/raft/transport.go`
- `internal/raft/errors.go`

---

#### Task 1.2.2: Implement Finite State Machine (FSM)
**Priority**: P0
**Estimated Time**: 8 hours
**Dependencies**: 1.2.1

**Description**:
Implement the Raft FSM that applies log entries to the KV and Service stores.

**Acceptance Criteria**:
- [ ] Implement `KonsulFSM` struct
- [ ] Implement `Apply()` method for log entry application
- [ ] Support log entry types: kv_set, kv_delete, service_register, service_deregister
- [ ] Implement `Snapshot()` method for state snapshot
- [ ] Implement `Restore()` method for snapshot restoration
- [ ] Add proper error handling
- [ ] Thread-safe with proper locking
- [ ] Add logging for FSM operations
- [ ] Write unit tests for FSM

**File**: `internal/raft/fsm.go`

**Key Methods**:
```go
type KonsulFSM struct {
    kvStore      *store.KVStore
    serviceStore *store.ServiceStore
    healthStore  *store.HealthStore
    logger       *slog.Logger
    mu           sync.RWMutex
}

func (f *KonsulFSM) Apply(log *raft.Log) interface{}
func (f *KonsulFSM) Snapshot() (raft.FSMSnapshot, error)
func (f *KonsulFSM) Restore(rc io.ReadCloser) error
```

---

#### Task 1.2.3: Implement Log Entry Types
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.2.2

**Description**:
Define log entry structures and serialization.

**Acceptance Criteria**:
- [ ] Create `LogEntry` struct with Type, Key, Value, Timestamp fields
- [ ] Implement JSON marshaling/unmarshaling
- [ ] Support entry types: kv_set, kv_delete, service_register, service_deregister, health_update
- [ ] Add validation for log entries
- [ ] Write unit tests for serialization

**File**: `internal/raft/log_entry.go`

---

#### Task 1.2.4: Implement Snapshot Logic
**Priority**: P0
**Estimated Time**: 6 hours
**Dependencies**: 1.2.3

**Description**:
Implement snapshot creation and restoration for efficient log compaction.

**Acceptance Criteria**:
- [ ] Implement `KonsulSnapshot` struct
- [ ] Implement `Persist()` method to write snapshot
- [ ] Implement `Release()` method for cleanup
- [ ] Snapshot includes: KV store data, Service registry, Health checks
- [ ] Use efficient serialization (JSON or MessagePack)
- [ ] Add compression for large snapshots
- [ ] Write unit tests for snapshot/restore

**File**: `internal/raft/snapshot.go`

---

#### Task 1.2.5: Implement Raft Node Wrapper
**Priority**: P0
**Estimated Time**: 12 hours
**Dependencies**: 1.2.4

**Description**:
Create the main Raft node wrapper that initializes and manages the Raft instance.

**Acceptance Criteria**:
- [ ] Implement `RaftNode` struct
- [ ] Implement `NewRaftNode()` constructor
- [ ] Setup log store (BoltDB)
- [ ] Setup stable store (BoltDB)
- [ ] Setup snapshot store (File-based)
- [ ] Setup network transport (TCP)
- [ ] Configure Raft timeouts
- [ ] Implement `Bootstrap()` for first node
- [ ] Implement `Join()` for joining existing cluster
- [ ] Implement `IsLeader()` helper
- [ ] Implement `Leader()` to get current leader
- [ ] Implement `Apply()` wrapper for log application
- [ ] Add proper cleanup/shutdown
- [ ] Write integration tests

**File**: `internal/raft/node.go`

**Key Methods**:
```go
type RaftNode struct {
    raft         *raft.Raft
    fsm          *KonsulFSM
    transport    *raft.NetworkTransport
    config       *raft.Config
    logStore     raft.LogStore
    stableStore  raft.StableStore
    snapshotStore raft.SnapshotStore
}

func NewRaftNode(cfg *config.RaftConfig, kvStore, serviceStore, healthStore) (*RaftNode, error)
func (n *RaftNode) Bootstrap() error
func (n *RaftNode) Join(nodeID, addr string) error
func (n *RaftNode) IsLeader() bool
func (n *RaftNode) Leader() string
func (n *RaftNode) Apply(data []byte, timeout time.Duration) error
func (n *RaftNode) Shutdown() error
```

---

### 1.3 Handler Integration

#### Task 1.3.1: Update KV Handler for Raft
**Priority**: P0
**Estimated Time**: 6 hours
**Dependencies**: 1.2.5

**Description**:
Modify KV handler to use Raft for writes when clustering is enabled.

**Acceptance Criteria**:
- [ ] Add `raftNode` field to `KVHandler`
- [ ] Modify `Set()` to check if leader
- [ ] Redirect to leader if not leader (HTTP 307)
- [ ] Use `raftNode.Apply()` for writes
- [ ] Keep direct reads for performance
- [ ] Add leader forwarding logic
- [ ] Handle Raft errors gracefully
- [ ] Update handler tests
- [ ] Add integration tests with 3-node cluster

**File**: `internal/handlers/kv.go`

**Modified Methods**:
- `Set()` - Apply through Raft
- `Delete()` - Apply through Raft
- `Get()` - Direct read (with eventual consistency note)
- `List()` - Direct read

---

#### Task 1.3.2: Update Service Handler for Raft
**Priority**: P0
**Estimated Time**: 6 hours
**Dependencies**: 1.2.5

**Description**:
Modify Service handler to use Raft for registration/deregistration.

**Acceptance Criteria**:
- [ ] Add `raftNode` field to `ServiceHandler`
- [ ] Modify `Register()` to use Raft
- [ ] Modify `Deregister()` to use Raft
- [ ] Redirect to leader if not leader
- [ ] Keep direct reads for queries
- [ ] Handle Raft errors
- [ ] Update tests

**File**: `internal/handlers/service.go`

---

#### Task 1.3.3: Update Health Handler for Raft
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 1.2.5

**Description**:
Modify Health handler to use Raft for health check updates.

**Acceptance Criteria**:
- [ ] Add `raftNode` field to `HealthHandler`
- [ ] Modify health check updates to use Raft
- [ ] Redirect to leader if not leader
- [ ] Keep direct reads
- [ ] Update tests

**File**: `internal/handlers/health.go`

---

### 1.4 Main Application Integration

#### Task 1.4.1: Initialize Raft in Main
**Priority**: P0
**Estimated Time**: 6 hours
**Dependencies**: 1.3.1, 1.3.2, 1.3.3

**Description**:
Integrate Raft initialization into the main application startup.

**Acceptance Criteria**:
- [ ] Add Raft initialization logic in `cmd/konsul/main.go`
- [ ] Check if clustering is enabled in config
- [ ] Initialize Raft node
- [ ] Bootstrap or join cluster based on config
- [ ] Pass Raft node to handlers
- [ ] Add graceful shutdown for Raft
- [ ] Handle Raft startup errors
- [ ] Add startup logging

**File**: `cmd/konsul/main.go`

**Logic**:
```go
if cfg.Raft.Enabled {
    raftNode, err := raft.NewRaftNode(cfg.Raft, kvStore, serviceStore, healthStore)
    if err != nil {
        log.Fatal("Failed to create Raft node", "error", err)
    }

    if cfg.Raft.Bootstrap {
        if err := raftNode.Bootstrap(); err != nil {
            log.Fatal("Failed to bootstrap Raft", "error", err)
        }
    } else if len(cfg.Raft.JoinAddresses) > 0 {
        // Join existing cluster
    }

    // Pass raftNode to handlers
    kvHandler := handlers.NewKVHandler(kvStore, raftNode, logger)
}
```

---

### 1.5 Testing

#### Task 1.5.1: Unit Tests for FSM
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.2.2

**Description**:
Write comprehensive unit tests for the Finite State Machine.

**Acceptance Criteria**:
- [ ] Test Apply() for all log entry types
- [ ] Test Snapshot() creates valid snapshots
- [ ] Test Restore() recovers state correctly
- [ ] Test concurrent operations
- [ ] Test error handling
- [ ] Achieve >80% code coverage

**File**: `internal/raft/fsm_test.go`

---

#### Task 1.5.2: Integration Tests for 3-Node Cluster
**Priority**: P0
**Estimated Time**: 8 hours
**Dependencies**: 1.4.1

**Description**:
Create integration tests that spin up a 3-node Raft cluster.

**Acceptance Criteria**:
- [ ] Test cluster bootstrap
- [ ] Test leader election
- [ ] Test write replication
- [ ] Test leader failure and re-election
- [ ] Test follower reads
- [ ] Test cluster with all operations: KV, Service, Health
- [ ] Use in-memory stores for speed
- [ ] Cleanup properly after tests

**File**: `internal/raft/integration_test.go`

---

#### Task 1.5.3: End-to-End Tests with HTTP API
**Priority**: P1
**Estimated Time**: 6 hours
**Dependencies**: 1.4.1

**Description**:
Test the full stack with HTTP requests against a clustered setup.

**Acceptance Criteria**:
- [ ] Test KV operations through HTTP API
- [ ] Test service registration through HTTP API
- [ ] Test leader redirection
- [ ] Test reads from followers
- [ ] Test client behavior during leader election

**File**: `test/e2e/raft_cluster_test.go`

---

## Phase 2: Cluster Management API (1-2 weeks)

### 2.1 Cluster Status & Info

#### Task 2.1.1: Create Cluster Handler
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 1.4.1

**Description**:
Create a new handler for cluster management operations.

**Acceptance Criteria**:
- [ ] Create `internal/handlers/cluster.go`
- [ ] Add `ClusterHandler` struct
- [ ] Add constructor with dependencies
- [ ] Add logging

**File**: `internal/handlers/cluster.go`

---

#### Task 2.1.2: Implement GET /cluster/status
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 2.1.1

**Description**:
Endpoint to get cluster health and member information.

**Acceptance Criteria**:
- [ ] Return cluster state (healthy, degraded, unavailable)
- [ ] Return current leader info
- [ ] Return list of all nodes with status
- [ ] Return quorum size and status
- [ ] Include Raft statistics
- [ ] Add ACL protection (admin capability)
- [ ] Write tests

**Response Example**:
```json
{
  "state": "healthy",
  "leader": {
    "id": "node-1",
    "address": "10.0.1.10:7000"
  },
  "nodes": [
    {"id": "node-1", "address": "10.0.1.10:7000", "state": "leader"},
    {"id": "node-2", "address": "10.0.1.11:7000", "state": "follower"},
    {"id": "node-3", "address": "10.0.1.12:7000", "state": "follower"}
  ],
  "quorum": 2,
  "stats": {
    "commit_index": 1234,
    "last_log_index": 1234,
    "last_snapshot_index": 1000
  }
}
```

---

#### Task 2.1.3: Implement GET /cluster/leader
**Priority**: P1
**Estimated Time**: 2 hours
**Dependencies**: 2.1.1

**Description**:
Endpoint to get current leader information.

**Acceptance Criteria**:
- [ ] Return leader ID and address
- [ ] Return 404 if no leader
- [ ] Add tests

---

#### Task 2.1.4: Implement GET /cluster/peers
**Priority**: P1
**Estimated Time**: 2 hours
**Dependencies**: 2.1.1

**Description**:
List all cluster peers.

**Acceptance Criteria**:
- [ ] Return list of all peer nodes
- [ ] Include ID, address, and voter status
- [ ] Add ACL protection
- [ ] Write tests

---

### 2.2 Cluster Membership

#### Task 2.2.1: Implement POST /cluster/join
**Priority**: P1
**Estimated Time**: 6 hours
**Dependencies**: 2.1.1

**Description**:
Endpoint to join a new node to the cluster.

**Acceptance Criteria**:
- [ ] Accept node ID and address in request body
- [ ] Only leader can process join requests
- [ ] Add node as non-voter initially
- [ ] Promote to voter after catch-up
- [ ] Return join status
- [ ] Add ACL protection (admin capability)
- [ ] Handle errors (duplicate node, invalid address)
- [ ] Write tests

**Request Example**:
```json
{
  "node_id": "node-4",
  "address": "10.0.1.13:7000"
}
```

---

#### Task 2.2.2: Implement DELETE /cluster/leave/:id
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 2.1.1

**Description**:
Endpoint to remove a node from the cluster.

**Acceptance Criteria**:
- [ ] Only leader can process leave requests
- [ ] Remove node from Raft configuration
- [ ] Update quorum
- [ ] Add ACL protection (admin capability)
- [ ] Handle errors (node not found, last node)
- [ ] Write tests

---

#### Task 2.2.3: Implement POST /cluster/snapshot
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 2.1.1

**Description**:
Manually trigger a Raft snapshot.

**Acceptance Criteria**:
- [ ] Only leader can trigger snapshot
- [ ] Return snapshot metadata
- [ ] Add ACL protection (admin capability)
- [ ] Write tests

---

### 2.3 Cluster Metrics

#### Task 2.3.1: Add Raft Prometheus Metrics
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 1.4.1

**Description**:
Export Raft-specific metrics for monitoring.

**Acceptance Criteria**:
- [ ] Add metrics to `internal/metrics/metrics.go`
- [ ] Metric: `konsul_raft_state` (gauge: leader=1, follower=0, candidate=2)
- [ ] Metric: `konsul_raft_leader_changes_total` (counter)
- [ ] Metric: `konsul_raft_commit_time_seconds` (histogram)
- [ ] Metric: `konsul_raft_apply_time_seconds` (histogram)
- [ ] Metric: `konsul_raft_log_entries_total` (counter)
- [ ] Metric: `konsul_raft_log_size_bytes` (gauge)
- [ ] Metric: `konsul_raft_snapshot_duration_seconds` (histogram)
- [ ] Metric: `konsul_raft_peers` (gauge)
- [ ] Metric: `konsul_raft_last_contact_seconds` (gauge)
- [ ] Update metrics in Raft node
- [ ] Document metrics

**File**: `internal/metrics/metrics.go`

---

### 2.4 CLI Commands

#### Task 2.4.1: Add Cluster Commands to konsulctl
**Priority**: P1
**Estimated Time**: 8 hours
**Dependencies**: 2.1.4, 2.2.3

**Description**:
Add cluster management commands to konsulctl CLI.

**Acceptance Criteria**:
- [ ] Add `cluster` command group
- [ ] Command: `konsulctl cluster status`
- [ ] Command: `konsulctl cluster leader`
- [ ] Command: `konsulctl cluster peers`
- [ ] Command: `konsulctl cluster join --address <addr>`
- [ ] Command: `konsulctl cluster leave --node <id>`
- [ ] Command: `konsulctl cluster snapshot`
- [ ] Command: `konsulctl cluster health`
- [ ] Add color output for status
- [ ] Add table formatting for peers
- [ ] Add documentation

**File**: `cmd/konsulctl/cluster.go`

---

## Phase 3: Rolling Upgrades & Autopilot (2 weeks)

### 3.1 Autopilot Integration

#### Task 3.1.1: Add Autopilot Dependency
**Priority**: P2
**Estimated Time**: 2 hours
**Dependencies**: Phase 2 complete

**Description**:
Add HashiCorp Raft Autopilot library.

**Acceptance Criteria**:
- [ ] Add `github.com/hashicorp/raft-autopilot` to go.mod
- [ ] Run `go mod tidy`

---

#### Task 3.1.2: Implement Autopilot Features
**Priority**: P2
**Estimated Time**: 12 hours
**Dependencies**: 3.1.1

**Description**:
Integrate Autopilot for automated cluster management.

**Acceptance Criteria**:
- [ ] Configure Autopilot in Raft node
- [ ] Enable dead server cleanup
- [ ] Enable server health monitoring
- [ ] Configure server stabilization time
- [ ] Add redundancy zone awareness
- [ ] Add Autopilot state to cluster status endpoint
- [ ] Write tests

**File**: `internal/raft/autopilot.go`

---

### 3.2 Rolling Upgrades

#### Task 3.2.1: Document Rolling Upgrade Process
**Priority**: P2
**Estimated Time**: 4 hours
**Dependencies**: 3.1.2

**Description**:
Create detailed documentation for zero-downtime rolling upgrades.

**Acceptance Criteria**:
- [ ] Create `docs/rolling-upgrades.md`
- [ ] Document upgrade process step-by-step
- [ ] Include pre-upgrade checklist
- [ ] Include rollback procedure
- [ ] Include version compatibility matrix
- [ ] Add examples for 3-node and 5-node clusters
- [ ] Document common issues and solutions

---

#### Task 3.2.2: Create Upgrade Scripts
**Priority**: P2
**Estimated Time**: 6 hours
**Dependencies**: 3.2.1

**Description**:
Create helper scripts for rolling upgrades.

**Acceptance Criteria**:
- [ ] Create `scripts/rolling-upgrade.sh`
- [ ] Script validates cluster health before starting
- [ ] Script upgrades one node at a time
- [ ] Script waits for node to rejoin before continuing
- [ ] Script validates cluster health between steps
- [ ] Add rollback capability
- [ ] Add dry-run mode
- [ ] Document script usage

---

## Phase 4: Monitoring & Observability (1 week)

### 4.1 Health Checks

#### Task 4.1.1: Add Raft Health Checks
**Priority**: P2
**Estimated Time**: 4 hours
**Dependencies**: 2.3.1

**Description**:
Add comprehensive health checks for Raft cluster.

**Acceptance Criteria**:
- [ ] Check: Leader exists
- [ ] Check: Quorum is healthy
- [ ] Check: Replication lag < threshold
- [ ] Check: No recent leader elections
- [ ] Check: All peers reachable
- [ ] Add to `/health` endpoint
- [ ] Add detailed status for each check
- [ ] Write tests

**File**: `internal/health/raft.go`

---

#### Task 4.1.2: Add Raft to Readiness/Liveness Probes
**Priority**: P2
**Estimated Time**: 3 hours
**Dependencies**: 4.1.1

**Description**:
Integrate Raft health into Kubernetes probes.

**Acceptance Criteria**:
- [ ] Liveness: Process is running, Raft initialized
- [ ] Readiness: Raft connected to cluster, can serve traffic
- [ ] Update `/health/ready` endpoint
- [ ] Update `/health/live` endpoint
- [ ] Document probe configuration

---

### 4.2 Alerting Rules

#### Task 4.2.1: Create Prometheus Alerting Rules
**Priority**: P2
**Estimated Time**: 4 hours
**Dependencies**: 2.3.1

**Description**:
Create Prometheus alerting rules for Raft cluster issues.

**Acceptance Criteria**:
- [ ] Create `monitoring/prometheus/raft-alerts.yaml`
- [ ] Alert: No leader (critical)
- [ ] Alert: Frequent leader elections (warning)
- [ ] Alert: High replication lag (warning)
- [ ] Alert: Node down (warning)
- [ ] Alert: Quorum at risk (critical)
- [ ] Alert: Slow commit time (warning)
- [ ] Alert: Large log size (info)
- [ ] Document alert meanings and resolutions

---

### 4.3 Dashboards

#### Task 4.3.1: Create Grafana Dashboard for Raft
**Priority**: P2
**Estimated Time**: 6 hours
**Dependencies**: 2.3.1

**Description**:
Create a Grafana dashboard for Raft cluster monitoring.

**Acceptance Criteria**:
- [ ] Create `monitoring/grafana/raft-dashboard.json`
- [ ] Panel: Cluster state overview
- [ ] Panel: Leader election timeline
- [ ] Panel: Commit latency (p50, p95, p99)
- [ ] Panel: Apply latency
- [ ] Panel: Log size over time
- [ ] Panel: Replication lag per follower
- [ ] Panel: Number of peers
- [ ] Panel: Snapshot frequency
- [ ] Add annotations for leader changes
- [ ] Document dashboard import

---

### 4.4 Distributed Tracing

#### Task 4.4.1: Add Tracing to Raft Operations
**Priority**: P2
**Estimated Time**: 4 hours
**Dependencies**: Phase 1 complete

**Description**:
Integrate OpenTelemetry tracing for Raft operations.

**Acceptance Criteria**:
- [ ] Trace: Write operations (client → leader → commit)
- [ ] Trace: Read operations
- [ ] Trace: Leader election
- [ ] Trace: Snapshot creation
- [ ] Include relevant attributes (node ID, log index, etc.)
- [ ] Propagate context across nodes
- [ ] Write tests

---

## Additional Tasks

### A.1 Documentation

#### Task A.1.1: Write Raft User Guide
**Priority**: P1
**Estimated Time**: 6 hours
**Dependencies**: Phase 1 complete

**Description**:
Create comprehensive user documentation for Raft clustering.

**Acceptance Criteria**:
- [ ] Create `docs/clustering.md`
- [ ] Section: Overview and concepts
- [ ] Section: Configuration guide
- [ ] Section: Deployment patterns (3-node, 5-node)
- [ ] Section: Cluster operations (join, leave, upgrade)
- [ ] Section: Monitoring and alerting
- [ ] Section: Troubleshooting guide
- [ ] Section: Performance tuning
- [ ] Add diagrams and examples

---

#### Task A.1.2: Update Main README
**Priority**: P1
**Estimated Time**: 2 hours
**Dependencies**: A.1.1

**Description**:
Update README with clustering information.

**Acceptance Criteria**:
- [ ] Add "High Availability" section
- [ ] Mention Raft consensus
- [ ] Link to clustering.md
- [ ] Update architecture diagram
- [ ] Add quick start for 3-node cluster

---

### A.2 Configuration & Deployment

#### Task A.2.1: Create Helm Chart for Clustered Deployment
**Priority**: P2
**Estimated Time**: 8 hours
**Dependencies**: Phase 1 complete

**Description**:
Create Kubernetes Helm chart for deploying a Konsul cluster.

**Acceptance Criteria**:
- [ ] Create `deploy/helm/konsul/` directory
- [ ] StatefulSet for Raft nodes
- [ ] Headless service for peer discovery
- [ ] LoadBalancer service for client access
- [ ] ConfigMap for configuration
- [ ] PersistentVolumeClaim for Raft data
- [ ] Init container for bootstrapping
- [ ] Support for 3, 5, 7 node clusters
- [ ] Readiness/liveness probes configured
- [ ] Document Helm values

---

#### Task A.2.2: Create Docker Compose for Local Cluster
**Priority**: P2
**Estimated Time**: 4 hours
**Dependencies**: Phase 1 complete

**Description**:
Create docker-compose.yml for local 3-node cluster testing.

**Acceptance Criteria**:
- [ ] Create `docker-compose-cluster.yml`
- [ ] Define 3 Konsul nodes
- [ ] Configure networking
- [ ] Mount volumes for persistence
- [ ] Bootstrap first node
- [ ] Auto-join cluster on startup
- [ ] Expose ports for testing
- [ ] Document usage

---

### A.3 Security

#### Task A.3.1: Add TLS Support for Raft Transport
**Priority**: P2
**Estimated Time**: 8 hours
**Dependencies**: Phase 1 complete

**Description**:
Secure Raft communication with TLS encryption.

**Acceptance Criteria**:
- [ ] Add TLS configuration to RaftConfig
- [ ] Support mTLS between nodes
- [ ] Certificate validation
- [ ] Support for cert rotation
- [ ] Add TLS to TCP transport
- [ ] Document certificate setup
- [ ] Write tests

---

#### Task A.3.2: Add Cluster Join Token Authentication
**Priority**: P2
**Estimated Time**: 4 hours
**Dependencies**: Phase 2 complete

**Description**:
Require authentication token for nodes joining the cluster.

**Acceptance Criteria**:
- [ ] Add join token to configuration
- [ ] Validate token on join requests
- [ ] Support token rotation
- [ ] Document token management
- [ ] Write tests

---

## Summary

### Total Time Estimate
- **Phase 1**: 78 hours (3-4 weeks)
- **Phase 2**: 54 hours (1-2 weeks)
- **Phase 3**: 24 hours (2 weeks with testing)
- **Phase 4**: 21 hours (1 week)
- **Additional**: 32 hours

**Total**: ~209 hours (~5-6 weeks for a single developer)

### Critical Path
1. Dependencies & Config (1.1.1 → 1.1.2)
2. Raft Core (1.2.1 → 1.2.5)
3. Handler Integration (1.3.1 → 1.3.3)
4. Main Integration (1.4.1)
5. Testing (1.5.1 → 1.5.3)
6. Cluster Management API (Phase 2)
7. Autopilot & Rolling Upgrades (Phase 3)
8. Monitoring (Phase 4)

### Priorities
- **P0** (Must Have): Phase 1 tasks - Core functionality
- **P1** (Should Have): Phase 2 + Documentation - Management & usability
- **P2** (Nice to Have): Phases 3 & 4 - Production hardening

### Dependencies Graph
```
1.1.1 → 1.1.2 → 1.2.1 → 1.2.2 → 1.2.3 → 1.2.4 → 1.2.5
                              ↓           ↓
                            1.2.3      1.5.1
                              ↓
                         1.3.1, 1.3.2, 1.3.3
                              ↓
                            1.4.1
                              ↓
                        1.5.2, 1.5.3
                              ↓
                          Phase 2
                              ↓
                          Phase 3
                              ↓
                          Phase 4
```

---

## Next Steps

1. **Review this plan** with the team
2. **Update ADR-0011** status to "Accepted" if approved
3. **Create GitHub issues** for each task
4. **Set up project board** to track progress
5. **Assign tasks** to team members
6. **Start with Phase 1** tasks in order
