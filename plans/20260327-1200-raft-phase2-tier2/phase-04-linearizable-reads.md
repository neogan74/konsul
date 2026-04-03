# Phase 04 — Linearizable Reads (ReadIndex/Barrier)

**Parent:** [plan.md](plan.md)
**Date:** 2026-03-27
**Priority:** 🟡 Medium
**Status:** Todo

---

## Context Links

- Research: `research/researcher-02-linearizable-reads.md`
- Depends on: Phases 01-03 (independent, can be done in parallel)

---

## Overview

Add linearizable reads via `raft.Barrier()` + `raft.VerifyLeader()`.
Clients opt-in via `?consistent=true` query param.
Default (no param) = stale reads from local state (current behaviour, faster).

---

## Key Insights

- HashiCorp Raft has no native ReadIndex, but `Barrier(timeout)` achieves same effect
- `Barrier()` waits until all committed log entries are applied to FSM
- `VerifyLeader()` confirms node is still the leader (prevents stale leader reads)
- Consistent read = `VerifyLeader()` → `Barrier()` → read local state
- Consul uses this exact pattern
- Performance impact: ~1 RTT to followers (heartbeat round)

---

## Architecture

```
Client: GET /kv/foo?consistent=true
            ↓
Handler: h.raftNode != nil && consistent
            ↓
Node.LinearizableRead(5s)
  1. VerifyLeader() — confirm still leader
  2. Barrier(timeout) — wait for all entries applied
            ↓
Read from local kvStore (guaranteed up-to-date)
```

---

## Related Code Files

| File | Lines | Change |
|------|-------|--------|
| `internal/raft/node.go` | new method | Add `LinearizableRead(timeout)` |
| `internal/handlers/kv.go` | GET handler | Parse `?consistent` + call LinearizableRead |
| `internal/handlers/batch.go` | BatchGet | Parse `?consistent` + call LinearizableRead |

---

## Implementation Steps

### 1. Add `LinearizableRead()` to `node.go`

```go
// LinearizableRead ensures the node is the current leader and all committed
// log entries have been applied to the FSM before a read.
// Use this before serving reads that require linearizability.
func (n *Node) LinearizableRead(timeout time.Duration) error {
    // Step 1: Verify we are still the leader
    if err := n.raft.VerifyLeader().Error(); err != nil {
        return fmt.Errorf("linearizable read: not leader: %w", err)
    }
    // Step 2: Wait for all committed entries to be applied
    if err := n.raft.Barrier(timeout).Error(); err != nil {
        return fmt.Errorf("linearizable read: barrier failed: %w", err)
    }
    return nil
}
```

### 2. Update `handlers/kv.go` — GET handler

Parse query param and conditionally use linearizable read:

```go
// In GET /kv/:key handler, before reading from store:
consistent := c.Query("consistent") == "true"

if consistent && h.raftNode != nil {
    if err := h.raftNode.LinearizableRead(5 * time.Second); err != nil {
        return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
            "error": "linearizable read failed: " + err.Error(),
        })
    }
}

// Then read from store as normal
value, err := h.store.Get(key)
```

### 3. Update `handlers/kv.go` — List/Prefix handlers

Same pattern for `GET /kv` (list all) and any prefix-scan endpoints.

### 4. Update `handlers/batch.go` — BatchGet

```go
// In BatchGet handler:
consistent := c.Query("consistent") == "true"

if consistent && h.raftNode != nil {
    if err := h.raftNode.LinearizableRead(5 * time.Second); err != nil {
        return c.Status(fiber.StatusServiceUnavailable).JSON(...)
    }
}
```

### 5. Document in API

Add query param docs to `docs/kv-watch-guide.md` and OpenAPI spec if exists.

---

## Todo

- [ ] Add `LinearizableRead(timeout time.Duration) error` to `node.go`
- [ ] Update GET `/kv/:key` to parse `?consistent=true`
- [ ] Update GET `/kv` (list) for `?consistent=true`
- [ ] Update `POST /batch/kv/get` for `?consistent=true`
- [ ] Handle non-leader: return 503 (not 500) when consistent read fails
- [ ] Update `docs/authentication.md` or create `docs/kv-consistent-reads.md`
- [ ] Run `make test`

---

## Success Criteria

- `GET /kv/foo` (no param) returns immediately (stale read, current behavior)
- `GET /kv/foo?consistent=true` on leader: waits for Barrier, returns fresh data
- `GET /kv/foo?consistent=true` on follower: returns 503 (not leader)
- Write then consistent read: guaranteed to see the write

---

## Risk Assessment

- **Low-Medium** — additive only, default behavior unchanged
- Performance: consistent reads add ~1 heartbeat RTT (~50-100ms in tests)
- Risk: if leader is under load, Barrier may timeout → 503 to client

## Security Considerations

- No additional auth needed — existing JWT/ACL applies to all reads
- DoS consideration: `?consistent=true` is more expensive — could be rate-limited separately

---

## Next Steps → Phase 05
