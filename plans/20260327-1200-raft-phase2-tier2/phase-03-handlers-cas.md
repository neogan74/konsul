# Phase 03 — Fix Handlers (kv.go + batch.go)

**Parent:** [plan.md](plan.md)
**Date:** 2026-03-27
**Priority:** 🔴 High
**Status:** Todo

---

## Context Links

- Depends on: [Phase 02](phase-02-node-cas-methods.md) (Node CAS methods)
- Scout: `scout/scout-01-codebase.md` §6, §7

---

## Overview

Replace manual command building in HTTP handlers with Node CAS methods.
Fix type assertions that silently fail in clustered mode.

---

## Key Insights

- `kv.go:144-165`: Manual CmdKVSetCAS command → replace with `h.raftNode.KVSetCAS()`
- `kv.go:256-266`: Manual CmdKVDeleteCAS → replace with `h.raftNode.KVDeleteCAS()`
- `batch.go:273-293`: BatchSetCAS type assertion `resp.(map[string]uint64)` fails → `h.raftNode.KVBatchSetCAS()`
- `batch.go:356-373`: BatchDeleteCAS → `h.raftNode.KVBatchDeleteCAS()`

---

## Related Code Files

| File | Lines | Change |
|------|-------|--------|
| `internal/handlers/kv.go` | 144-165 | Replace with `raftNode.KVSetCAS()` |
| `internal/handlers/kv.go` | 256-266 | Replace with `raftNode.KVDeleteCAS()` |
| `internal/handlers/batch.go` | 273-293 | Replace with `raftNode.KVBatchSetCAS()` |
| `internal/handlers/batch.go` | 356-373 | Replace with `raftNode.KVBatchDeleteCAS()` |

---

## Implementation Steps

### handlers/kv.go — Set with CAS

**Before (lines 144-165):**
```go
if h.raftNode != nil {
    cmd, marshalErr := konsulraft.NewCommand(konsulraft.CmdKVSetCAS, ...)
    resp, applyErr := h.raftNode.ApplyEntry(cmd, 5*time.Second)
    if applyErr == nil {
        if index, ok := resp.(uint64); ok { newIndex = index }
    }
    err = applyErr
}
```

**After:**
```go
if h.raftNode != nil {
    newIndex, err = h.raftNode.KVSetCAS(key, body.Value, *body.CAS)
}
```

### handlers/kv.go — Delete with CAS

**Before (lines 256-266):**
```go
if h.raftNode != nil {
    cmd, _ := konsulraft.NewCommand(konsulraft.CmdKVDeleteCAS, ...)
    _, applyErr := h.raftNode.ApplyEntry(cmd, 5*time.Second)
    err = applyErr
}
```

**After:**
```go
if h.raftNode != nil {
    err = h.raftNode.KVDeleteCAS(key, expectedIndex)
}
```

### handlers/batch.go — BatchSetCAS

**Before (lines 273-293):**
```go
if h.raftNode != nil {
    cmd, _ := konsulraft.NewCommand(konsulraft.CmdKVBatchSetCAS, ...)
    resp, applyErr := h.raftNode.ApplyEntry(cmd, 10*time.Second)
    if applyErr == nil {
        if cast, ok := resp.(map[string]uint64); ok { newIndices = cast }
    }
    err = applyErr
}
```

**After:**
```go
if h.raftNode != nil {
    newIndices, err = h.raftNode.KVBatchSetCAS(req.Items, req.ExpectedIndices)
}
```

### handlers/batch.go — BatchDeleteCAS

**Before (lines 356-373):**
```go
if h.raftNode != nil {
    cmd, _ := konsulraft.NewCommand(konsulraft.CmdKVBatchDeleteCAS, ...)
    _, applyErr := h.raftNode.ApplyEntry(cmd, 10*time.Second)
    err = applyErr
}
```

**After:**
```go
if h.raftNode != nil {
    err = h.raftNode.KVBatchDeleteCAS(req.Keys, req.ExpectedIndices)
}
```

---

## Todo

- [ ] Refactor `kv.go` CAS Set path to use `raftNode.KVSetCAS()`
- [ ] Refactor `kv.go` CAS Delete path to use `raftNode.KVDeleteCAS()`
- [ ] Refactor `batch.go` BatchSetCAS path to use `raftNode.KVBatchSetCAS()`
- [ ] Refactor `batch.go` BatchDeleteCAS path to use `raftNode.KVBatchDeleteCAS()`
- [ ] Verify response JSON includes `modify_index` in CAS Set response
- [ ] Run `make test`

---

## Success Criteria

- `PUT /kv/foo` with `?cas=0` in cluster mode returns new `modify_index`
- `PUT /batch/kv` with CAS in cluster mode returns `new_indices` map
- CAS conflict returns 409 with error body (same as standalone)
- `make test` passes

---

## Risk Assessment

- **Low** — replacing equivalent logic, same branches, cleaner code
- Potential: response JSON format must match standalone mode exactly

---

## Next Steps → Phase 04
