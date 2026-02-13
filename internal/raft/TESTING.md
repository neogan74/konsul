# Raft Integration Test Suite

This directory contains comprehensive integration tests for the Raft consensus implementation in Konsul.

## Overview

The test suite is organized into **6 test files** covering **50+ test scenarios** across different aspects of Raft behavior:

1. **Leader Election** (`leader_election_integration_test.go`) - ✅ **IMPLEMENTED**
2. **Snapshot & Recovery** (`snapshot_recovery_integration_test.go`) - 🚧 TODO
3. **Data Replication** (`data_replication_integration_test.go`) - 🚧 TODO
4. **Consistency Guarantees** (`consistency_integration_test.go`) - 🚧 TODO
5. **Batch Operations** (`batch_operations_integration_test.go`) - 🚧 TODO
6. **Failure Scenarios** (`failure_scenarios_integration_test.go`) - 🚧 TODO

## Test Categories

### 1. Leader Election Tests ✅ (7 tests - IMPLEMENTED)

**File**: `leader_election_integration_test.go`

- ✅ `TestLeaderElection_ThreeNodeCluster` - Basic 3-node leader election
- ✅ `TestClusterJoinLeave` - Node join and leave operations
- ✅ `TestClusterJoinNonLeader` - Error handling for non-leader joins
- ✅ `TestLinearizableRead_LeaderOnly` - Linearizable reads on leader
- ✅ `TestLeaderElection_LeaderFailureReelection` - Re-election after leader failure
- ✅ `TestLeaderElection_PartitionMinorityNoLeader` - Minority partition behavior
- ✅ `TestLeaderElection_PerfP99` - Leader election performance (p99 < 300ms)

**Status**: Production-ready

### 2. Snapshot & Recovery Tests 🚧 (10 tests - TODO)

**File**: `snapshot_recovery_integration_test.go`

- ⏳ Automatic snapshot creation when threshold reached
- ⏳ Manual snapshot creation via API
- ⏳ Node recovery from snapshot on startup
- ⏳ Snapshot recovery followed by log replay
- ⏳ Log compaction after snapshot
- ⏳ Concurrent writes during snapshot creation
- ⏳ Snapshot retention policy (keep N most recent)
- ⏳ Corrupted snapshot handling
- ⏳ Large dataset snapshot (10,000+ entries)
- ⏳ Snapshot installation on new follower

**Why Important**: Phase 2 Tier 1 requirement for production readiness

### 3. Data Replication Tests 🚧 (11 tests - TODO)

**File**: `data_replication_integration_test.go`

- ⏳ KV write replication to followers
- ⏳ Service registration replication
- ⏳ Multiple concurrent writes replication
- ⏳ Replication lag monitoring
- ⏳ Follower catch-up after disconnect
- ⏳ High throughput replication (10,000+ ops)
- ⏳ AppendEntries retry logic
- ⏳ Parallel replication to multiple followers
- ⏳ Write order preservation
- ⏳ Replication after partition healing
- ⏳ Conflict resolution during catch-up

**Why Important**: Core Raft guarantee - all followers must receive data

### 4. Consistency Tests 🚧 (11 tests - TODO)

**File**: `consistency_integration_test.go`

- ⏳ Linearizable read guarantees
- ⏳ Stale read behavior
- ⏳ CAS operation success
- ⏳ CAS operation failure
- ⏳ CAS prevents race conditions
- ⏳ CAS across leader changes
- ⏳ Read-after-write consistency
- ⏳ Monotonic read guarantee
- ⏳ Causal consistency
- ⏳ Serializable snapshot isolation
- ⏳ Quorum reads

**Why Important**: Phase 2 Tier 2 requirement - correctness guarantees

### 5. Batch Operations Tests 🚧 (10 tests - TODO)

**File**: `batch_operations_integration_test.go`

- ⏳ Batch set operations
- ⏳ Batch delete operations
- ⏳ Batch CAS success
- ⏳ Batch CAS partial failure (atomicity)
- ⏳ Batch atomicity guarantee
- ⏳ Large batch handling (10,000+ ops)
- ⏳ Concurrent batch operations
- ⏳ Mixed operation batches
- ⏳ Batch replication to followers
- ⏳ Batch during leader change

**Why Important**: Phase 2 Tier 2 requirement - atomic batch operations

### 6. Failure Scenario Tests 🚧 (12 tests - TODO)

**File**: `failure_scenarios_integration_test.go`

- ⏳ Single node failure
- ⏳ Leader failure
- ⏳ Minority partition
- ⏳ Majority partition (cluster stops)
- ⏳ Cascading failures
- ⏳ Network flapping
- ⏳ Slow follower handling
- ⏳ Disk failure
- ⏳ Memory pressure
- ⏳ Restart all nodes
- ⏳ Restart followers
- ⏳ Split-brain prevention
- ⏳ Byzantine fault tolerance

**Why Important**: Phase 2 Tier 1 requirement - production resilience

## Running Tests

### Run All Raft Tests

```bash
go test -v ./internal/raft -timeout 10m
```

### Run Specific Test File

```bash
go test -v ./internal/raft -run TestLeaderElection
go test -v ./internal/raft -run TestSnapshot
go test -v ./internal/raft -run TestReplication
```

### Run Specific Test Case

```bash
go test -v ./internal/raft -run TestLeaderElection_ThreeNodeCluster
```

### Run Performance Tests

Performance tests are skipped by default. Enable with:

```bash
KONSUL_PERF_TEST=1 go test -v ./internal/raft -run TestPerf
```

### Run with Race Detector

```bash
go test -v -race ./internal/raft -timeout 15m
```

## Test Infrastructure

### Helper Functions

All test files use common helper functions from `leader_election_integration_test.go`:

- `getFreeAddr(t)` - Get free TCP port for node binding
- `newClusterConfig(t, nodeID, addr, bootstrap, opts)` - Create test cluster config
- `startTestNode(t, cfg)` - Start a Raft node for testing
- `newThreeNodeCluster(t, opts)` - Create and bootstrap 3-node cluster
- `waitForSingleLeader(t, nodes, timeout)` - Wait for leader election
- `waitForConfigSize(t, node, expected, timeout)` - Wait for cluster size

### Cluster Options

Control Raft timing for faster tests:

```go
opts := clusterOptions{
    heartbeat:   50 * time.Millisecond,  // Faster heartbeats
    election:    100 * time.Millisecond, // Faster elections
    leaderLease: 50 * time.Millisecond,  // Faster lease
}
```

## Implementation Roadmap

### Phase 1: Foundation ✅ (Week 1)
- ✅ Leader election tests
- ✅ Basic cluster join/leave
- ✅ Test infrastructure setup

### Phase 2: Data & Snapshots 🚧 (Week 2-3)
- ⏳ Implement snapshot tests (10 tests)
- ⏳ Implement replication tests (11 tests)
- ⏳ Add helper functions for data verification

### Phase 3: Consistency & Batches 🚧 (Week 4-5)
- ⏳ Implement consistency tests (11 tests)
- ⏳ Implement batch operation tests (10 tests)
- ⏳ Add CAS and atomic operation helpers

### Phase 4: Failure Scenarios 🚧 (Week 6-7)
- ⏳ Implement failure scenario tests (12 tests)
- ⏳ Add network partition simulation
- ⏳ Add fault injection helpers

### Phase 5: Integration & CI 🚧 (Week 8)
- ⏳ Full integration test run
- ⏳ CI/CD pipeline integration
- ⏳ Performance benchmarking
- ⏳ Test coverage report

## Coverage Goals

- **Unit Tests**: 80%+ code coverage
- **Integration Tests**: 50+ comprehensive scenarios
- **Performance Tests**: p99 latency benchmarks
- **Failure Tests**: 10+ failure modes covered

## Contributing

When adding new tests:

1. **Follow naming convention**: `Test<Category>_<Scenario>`
2. **Add test plan comment**: Explain what test does
3. **Use helper functions**: Reuse existing test infrastructure
4. **Add to this README**: Document new test in appropriate section
5. **Mark as TODO initially**: Use `t.Skip()` until implemented
6. **Measure performance**: Add timing for critical paths

## Performance Targets

Based on Phase 2 requirements:

- **Leader Election**: p99 < 300ms
- **Write Latency**: p99 < 20ms
- **Read Latency**: p99 < 2ms (linearizable), < 1ms (stale)
- **Throughput**: 100,000+ ops/sec per node
- **Replication Lag**: < 100ms for followers

## Test Maintenance

### Unskipping Tests

As tests are implemented, remove the `t.Skip()` line:

```go
// Before:
func TestSnapshot_AutomaticCreation(t *testing.T) {
    t.Skip("TODO: Implement automatic snapshot creation test")
    // ...
}

// After:
func TestSnapshot_AutomaticCreation(t *testing.T) {
    // Test implementation
    nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
    defer cleanup()
    // ...
}
```

### Updating Test Plans

Keep test plan comments up to date as implementation evolves.

## Questions or Issues?

- See `CLAUDE.md` for project context
- See `docs/adr/0030-raft-integration-implementation.md` for Raft architecture
- See `docs/adr/0031-raft-production-readiness.md` for Phase 2 requirements

---

**Test Suite Status**: 7/61 tests implemented (11.5%)
**Target**: 50+ tests for Phase 2 completion
**Last Updated**: 2026-02-12
