# Phase 01 — FSM CAS Result Type

**Parent:** [plan.md](plan.md)
**Date:** 2026-03-27
**Priority:** 🔴 Critical — blocks all CAS correctness
**Status:** Todo

---

## Context Links

- Scout: `scout/scout-01-codebase.md` §2, §9
- Research: `research/researcher-01-cas-batch-raft.md`
- Depends on: nothing (first phase)
- Blocks: Phase 02, Phase 03

---

## Overview

FSM.Apply() returns `interface{}`. Currently all handlers return `error`.
CAS handlers call `SetCASLocal()` which returns `(uint64, error)` — the uint64 is discarded.
Fix: introduce `CASResult` struct, return it from CAS handlers, thread it back to callers.

---

## Key Insights

- `fsm.go:164` — `_, err := f.kvStore.SetCASLocal(...)` discards new ModifyIndex
- `fsm.go:189` — `_, err := f.kvStore.BatchSetCASLocal(...)` discards `map[string]uint64`
- `fsm.go:270` — `_, err := f.serviceStore.RegisterCASLocal(...)` discards uint64
- `node.go:354` — `return future.Response()` returns whatever FSM returns
- Non-CAS handlers return `error` — this is correct and unchanged

---

## Architecture

```go
// New type in internal/raft/cas_result.go (or commands.go)
type CASResult struct {
    NewIndex   uint64            // for single-key CAS
    NewIndices map[string]uint64 // for batch CAS
    Err        error
}

func (r *CASResult) Error() error { return r.Err }
```

FSM Apply returns:
- `*CASResult` for CmdKVSetCAS, CmdKVDeleteCAS, CmdKVBatchSetCAS, CmdKVBatchDeleteCAS
- `*CASResult` for CmdServiceRegisterCAS, CmdServiceDeregisterCAS
- `error` (unchanged) for all other commands

Callers extract via type assertion: `if res, ok := resp.(*CASResult); ok { ... }`

---

## Related Code Files

| File | Lines | Change |
|------|-------|--------|
| `internal/raft/commands.go` | new | Add `CASResult` struct |
| `internal/raft/fsm.go` | 155-166 | `applyKVSetCAS` — capture + return CASResult |
| `internal/raft/fsm.go` | 180-191 | `applyKVBatchSetCAS` — capture + return CASResult |
| `internal/raft/fsm.go` | 193-204 | `applyKVBatchDeleteCAS` — return CASResult |
| `internal/raft/fsm.go` | 253-285 | service CAS handlers — return CASResult |

---

## Implementation Steps

1. **Add CASResult to `internal/raft/commands.go`** (after line 200):
```go
// CASResult carries both the new index and any error from a CAS operation.
// Returned by FSM.Apply() for all CAS command types.
type CASResult struct {
    NewIndex   uint64
    NewIndices map[string]uint64
    Err        error
}
```

2. **Fix `applyKVSetCAS` in fsm.go** (lines 155-166):
```go
func (f *KonsulFSM) applyKVSetCAS(payload []byte) interface{} {
    var p KVSetCASPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return &CASResult{Err: fmt.Errorf("unmarshal KVSetCASPayload: %w", err)}
    }
    f.mu.Lock()
    defer f.mu.Unlock()
    newIndex, err := f.kvStore.SetCASLocal(p.Key, p.Value, p.ExpectedIndex)
    return &CASResult{NewIndex: newIndex, Err: err}
}
```

3. **Fix `applyKVBatchSetCAS` in fsm.go** (lines 180-191):
```go
func (f *KonsulFSM) applyKVBatchSetCAS(payload []byte) interface{} {
    var p KVBatchSetCASPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return &CASResult{Err: fmt.Errorf("unmarshal KVBatchSetCASPayload: %w", err)}
    }
    f.mu.Lock()
    defer f.mu.Unlock()
    indices, err := f.kvStore.BatchSetCASLocal(p.Items, p.ExpectedIndices)
    return &CASResult{NewIndices: indices, Err: err}
}
```

4. **Fix `applyKVBatchDeleteCAS` in fsm.go** (lines 193-204):
```go
func (f *KonsulFSM) applyKVBatchDeleteCAS(payload []byte) interface{} {
    var p KVBatchDeleteCASPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return &CASResult{Err: fmt.Errorf("unmarshal KVBatchDeleteCASPayload: %w", err)}
    }
    f.mu.Lock()
    defer f.mu.Unlock()
    err := f.kvStore.BatchDeleteCASLocal(p.Keys, p.ExpectedIndices)
    return &CASResult{Err: err}
}
```

5. **Fix `applyKVDeleteCAS` in fsm.go** (lines 168-178):
```go
func (f *KonsulFSM) applyKVDeleteCAS(payload []byte) interface{} {
    var p KVDeleteCASPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return &CASResult{Err: fmt.Errorf("unmarshal KVDeleteCASPayload: %w", err)}
    }
    f.mu.Lock()
    defer f.mu.Unlock()
    err := f.kvStore.DeleteCASLocal(p.Key, p.ExpectedIndex)
    return &CASResult{Err: err}
}
```

6. **Update FSM Apply() switch** — change `applyErr = f.applyKVSetCAS(...)` pattern:
   - CAS handlers no longer assign to `applyErr`, they return `interface{}` directly
   - Non-CAS handlers still return `error` (unchanged)
   - Apply() returns the raw result from the handler

---

## Todo

- [ ] Add `CASResult` struct to `internal/raft/commands.go`
- [ ] Fix `applyKVSetCAS` signature and return
- [ ] Fix `applyKVDeleteCAS` signature and return
- [ ] Fix `applyKVBatchSetCAS` signature and return
- [ ] Fix `applyKVBatchDeleteCAS` signature and return
- [ ] Fix `applyServiceRegisterCAS` signature and return
- [ ] Fix `applyServiceDeregisterCAS` signature and return
- [ ] Update `Apply()` switch to handle mixed return types
- [ ] Run `make test` to confirm no regressions

---

## Success Criteria

- FSM.Apply() for CmdKVSetCAS returns `*CASResult{NewIndex: N, Err: nil}`
- FSM.Apply() for CmdKVBatchSetCAS returns `*CASResult{NewIndices: map, Err: nil}`
- CAS conflict returns `*CASResult{Err: &CASConflictError{...}}`
- Unit test: `fsm_test.go` confirms CASResult is returned with correct index

---

## Risk Assessment

- **Low risk** — change is isolated to fsm.go + new type
- Non-CAS handlers are unchanged
- All callers currently discard resp, so no breakage until Phase 02/03 wires them up

---

## Next Steps → Phase 02
