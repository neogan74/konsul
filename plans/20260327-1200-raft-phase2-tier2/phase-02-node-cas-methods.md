# Phase 02 — Node-Level CAS Methods

**Parent:** [plan.md](plan.md)
**Date:** 2026-03-27
**Priority:** 🔴 High
**Status:** Todo

---

## Context Links

- Depends on: [Phase 01](phase-01-fsm-cas-result.md) (CASResult must exist)
- Blocks: [Phase 03](phase-03-handlers-cas.md)
- Scout: `scout/scout-01-codebase.md` §3

---

## Overview

`node.go` has convenience methods like `KVSet()`, `KVDelete()`, `KVBatchSet()` etc.
CAS operations currently bypass Node and are built manually in HTTP handlers.
Add CAS methods to Node that: build command → apply via Raft → extract CASResult → return typed values.

---

## Key Insights

- `node.go:363` — `KVSet()` pattern to follow: create payload, NewCommand, applyCommand, return error
- `node.go:354` — `applyCommand()` returns `(interface{}, error)` from future
- After Phase 01, future response for CAS is `*CASResult`
- Adding typed methods simplifies handlers and makes CAS testable in isolation

---

## Related Code Files

| File | Lines | Change |
|------|-------|--------|
| `internal/raft/node.go` | after ~440 | Add 4 new CAS methods |

---

## Implementation Steps

Add to `internal/raft/node.go` after existing KVBatchDelete method:

```go
// KVSetCAS atomically sets a key only if its ModifyIndex matches expectedIndex.
// expectedIndex=0 means "create only if not exists".
// Returns the new ModifyIndex on success.
func (n *Node) KVSetCAS(key, value string, expectedIndex uint64) (uint64, error) {
    cmd, err := NewCommand(CmdKVSetCAS, KVSetCASPayload{
        Key:           key,
        Value:         value,
        ExpectedIndex: expectedIndex,
    })
    if err != nil {
        return 0, err
    }
    resp, err := n.applyCommand(cmd, 5*time.Second)
    if err != nil {
        return 0, err
    }
    res, ok := resp.(*CASResult)
    if !ok {
        return 0, fmt.Errorf("unexpected FSM response type: %T", resp)
    }
    return res.NewIndex, res.Err
}

// KVDeleteCAS atomically deletes a key only if its ModifyIndex matches expectedIndex.
func (n *Node) KVDeleteCAS(key string, expectedIndex uint64) error {
    cmd, err := NewCommand(CmdKVDeleteCAS, KVDeleteCASPayload{
        Key:           key,
        ExpectedIndex: expectedIndex,
    })
    if err != nil {
        return err
    }
    resp, err := n.applyCommand(cmd, 5*time.Second)
    if err != nil {
        return err
    }
    res, ok := resp.(*CASResult)
    if !ok {
        return fmt.Errorf("unexpected FSM response type: %T", resp)
    }
    return res.Err
}

// KVBatchSetCAS atomically sets multiple keys with per-key CAS checks.
// All keys succeed or none (atomic). Returns map of key→new ModifyIndex.
func (n *Node) KVBatchSetCAS(items map[string]string, expectedIndices map[string]uint64) (map[string]uint64, error) {
    cmd, err := NewCommand(CmdKVBatchSetCAS, KVBatchSetCASPayload{
        Items:           items,
        ExpectedIndices: expectedIndices,
    })
    if err != nil {
        return nil, err
    }
    resp, err := n.applyCommand(cmd, 10*time.Second)
    if err != nil {
        return nil, err
    }
    res, ok := resp.(*CASResult)
    if !ok {
        return nil, fmt.Errorf("unexpected FSM response type: %T", resp)
    }
    return res.NewIndices, res.Err
}

// KVBatchDeleteCAS atomically deletes multiple keys with per-key CAS checks.
func (n *Node) KVBatchDeleteCAS(keys []string, expectedIndices map[string]uint64) error {
    cmd, err := NewCommand(CmdKVBatchDeleteCAS, KVBatchDeleteCASPayload{
        Keys:            keys,
        ExpectedIndices: expectedIndices,
    })
    if err != nil {
        return err
    }
    resp, err := n.applyCommand(cmd, 10*time.Second)
    if err != nil {
        return err
    }
    res, ok := resp.(*CASResult)
    if !ok {
        return fmt.Errorf("unexpected FSM response type: %T", resp)
    }
    return res.Err
}
```

---

## Todo

- [ ] Add `KVSetCAS(key, value, expectedIndex) (uint64, error)` to node.go
- [ ] Add `KVDeleteCAS(key, expectedIndex) error` to node.go
- [ ] Add `KVBatchSetCAS(items, expectedIndices) (map[string]uint64, error)` to node.go
- [ ] Add `KVBatchDeleteCAS(keys, expectedIndices) error` to node.go
- [ ] Run `make build` to confirm compilation

---

## Success Criteria

- All 4 methods compile and return correct types
- `node.KVSetCAS("key", "val", 0)` on a fresh key returns new index
- `node.KVSetCAS("key", "val", 999)` on non-matching index returns CASConflictError

---

## Risk Assessment

- **Very low** — additive only, no existing code changed
- Methods follow exact same pattern as existing KVSet/KVDelete

---

## Next Steps → Phase 03
