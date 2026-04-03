# Phase 05 — Tests

**Parent:** [plan.md](plan.md)
**Date:** 2026-03-27
**Priority:** 🟡 Medium
**Status:** Todo

---

## Context Links

- Depends on: Phases 01-04
- Test files: `internal/raft/consistency_integration_test.go`
- Test files: `internal/raft/batch_operations_integration_test.go`
- Test infra: `internal/raft/leader_election_integration_test.go` (helpers)
- Test plan: `internal/raft/TESTING.md`

---

## Overview

Unskip and implement integration tests for CAS and linearizable reads.
TESTING.md shows 7/61 tests implemented. Phase 2 Tier 2 targets:
- 11 consistency tests (file: `consistency_integration_test.go`)
- 10 batch operation tests (file: `batch_operations_integration_test.go`)

---

## Tests to Implement

### consistency_integration_test.go (remove t.Skip())

| Test | What to verify |
|------|---------------|
| `TestConsistency_CASSuccess` | CAS with correct index succeeds, returns new index |
| `TestConsistency_CASFailure` | CAS with wrong index returns 409 CASConflictError |
| `TestConsistency_CASPreventsRace` | Two concurrent CAS on same key, only one wins |
| `TestConsistency_CASAcrossLeaderChange` | CAS survives leader re-election |
| `TestConsistency_LinearizableRead` | Write on leader, consistent read sees write |
| `TestConsistency_StaleRead` | Stale read may miss recent write (expected) |
| `TestConsistency_ReadAfterWrite` | Linearizable: read-after-write consistency guaranteed |
| `TestConsistency_MonotonicRead` | Never read older data than previously seen |

### batch_operations_integration_test.go (remove t.Skip())

| Test | What to verify |
|------|---------------|
| `TestBatch_SetOperations` | BatchSet replicates to all followers |
| `TestBatch_DeleteOperations` | BatchDelete replicates to all followers |
| `TestBatch_CASSuccess` | BatchSetCAS succeeds atomically, returns map of indices |
| `TestBatch_CASPartialFailure` | BatchSetCAS: if one key fails, NONE are written |
| `TestBatch_AtomicityGuarantee` | Batch is all-or-nothing |
| `TestBatch_Replication` | Batch written on leader appears on all nodes |
| `TestBatch_DuringLeaderChange` | Batch retried if leader changes mid-flight |

---

## Implementation Pattern

```go
func TestConsistency_CASSuccess(t *testing.T) {
    nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
    defer cleanup()

    leader := waitForSingleLeader(t, nodes, 5*time.Second)

    // Set initial value (no CAS)
    err := leader.KVSet("key", "v1")
    require.NoError(t, err)

    // Get the ModifyIndex
    entry, ok := leader.GetKVStore().GetEntrySnapshot("key")
    require.True(t, ok)
    initialIndex := entry.ModifyIndex

    // CAS update with correct index
    newIndex, err := leader.KVSetCAS("key", "v2", initialIndex)
    require.NoError(t, err)
    require.Greater(t, newIndex, initialIndex)

    // Verify all nodes see new value
    time.Sleep(100 * time.Millisecond)
    for _, node := range nodes {
        e, ok := node.GetKVStore().GetEntrySnapshot("key")
        assert.True(t, ok)
        assert.Equal(t, "v2", e.Value)
    }
}

func TestConsistency_CASFailure(t *testing.T) {
    nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
    defer cleanup()
    leader := waitForSingleLeader(t, nodes, 5*time.Second)

    err := leader.KVSet("key", "v1")
    require.NoError(t, err)

    // CAS with wrong index - should fail
    _, err = leader.KVSetCAS("key", "v2", 9999)
    require.Error(t, err)

    var casErr *store.CASConflictError
    require.ErrorAs(t, err, &casErr)

    // Value unchanged
    entry, _ := leader.GetKVStore().GetEntrySnapshot("key")
    assert.Equal(t, "v1", entry.Value)
}
```

---

## Unit Tests (fsm_test.go additions)

Add tests for CASResult propagation:

```go
func TestFSM_CASResultReturned(t *testing.T) {
    fsm := NewFSM(FSMConfig{KVStore: store.NewKVStore(nil, nil), ...})

    // Apply SetCAS with index=0 (create)
    cmd, _ := NewCommand(CmdKVSetCAS, KVSetCASPayload{
        Key: "k", Value: "v", ExpectedIndex: 0,
    })
    data, _ := cmd.Marshal()
    result := fsm.Apply(&raft.Log{Data: data})

    casResult, ok := result.(*CASResult)
    require.True(t, ok, "FSM must return *CASResult for CAS commands")
    require.NoError(t, casResult.Err)
    assert.Greater(t, casResult.NewIndex, uint64(0))
}
```

---

## Todo

- [ ] Add `TestFSM_CASResultReturned` to `internal/raft/fsm_test.go`
- [ ] Add `TestFSM_BatchCASResultReturned` to `internal/raft/fsm_test.go`
- [ ] Unskip `TestConsistency_CASSuccess` in consistency_integration_test.go
- [ ] Unskip `TestConsistency_CASFailure`
- [ ] Unskip `TestConsistency_CASPreventsRace`
- [ ] Unskip `TestConsistency_LinearizableRead`
- [ ] Unskip `TestBatch_CASSuccess`
- [ ] Unskip `TestBatch_CASPartialFailure`
- [ ] Unskip `TestBatch_AtomicityGuarantee`
- [ ] Run `make test` (unit)
- [ ] Run `go test -v ./internal/raft -timeout 10m` (integration)
- [ ] Update TESTING.md status

---

## Success Criteria

- All unskipped CAS tests pass
- Race detector clean: `go test -race ./internal/raft`
- TESTING.md count goes from 7/61 to 20+/61
- `make test` exit 0

---

## Risk Assessment

- **Medium** — integration tests start real Raft clusters, may be flaky on CI
- Use `clusterOptions{heartbeat: 50ms, election: 100ms}` for faster tests
- Use `waitForSingleLeader()` with adequate timeout

---

## Performance Targets (from TESTING.md)

- Write latency p99 < 20ms
- Consistent read latency p99 < 2ms (after Barrier)
- Stale read < 1ms
