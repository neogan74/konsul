# Raft Phase 2 Tier 2 — Implementation Plan

**Date:** 2026-03-27
**Goal:** Fix CAS operations via Raft + Atomic Batch ops + Linearizable Reads
**Status:** 🔴 Not started

---

## Context

- Parent: `docs/adr/0031-raft-production-readiness.md`
- Research: `research/researcher-01-cas-batch-raft.md`
- Research: `research/researcher-02-linearizable-reads.md`
- Scout: `scout/scout-01-codebase.md`

## Key Finding

All Raft command types and payloads for CAS/Batch are **already defined**.
The bug is narrow: `fsm.go` discards CAS return values (new indices), breaking clustered mode.
Standalone mode works correctly today.

## Phases

| # | Phase | Status | File |
|---|-------|--------|------|
| 01 | FSM CAS Result type + fix return values | ✅ Done | [phase-01](phase-01-fsm-cas-result.md) |
| 02 | Node-level CAS methods | ✅ Done | [phase-02](phase-02-node-cas-methods.md) |
| 03 | Fix Handlers (kv + batch) | 🔴 Todo | [phase-03](phase-03-handlers-cas.md) |
| 04 | Linearizable Reads (ReadIndex/Barrier) | 🔴 Todo | [phase-04](phase-04-linearizable-reads.md) |
| 05 | Tests — unskip integration tests | 🔴 Todo | [phase-05](phase-05-tests.md) |

## Estimated Effort

- Phase 01: ~1h (narrow fix, 3 lines in fsm.go + new type)
- Phase 02: ~1h (add 4 methods to node.go)
- Phase 03: ~1h (refactor 2 handlers)
- Phase 04: ~1h (add LinearizableRead + handler param)
- Phase 05: ~2h (unskip + implement integration tests)

**Total: ~6h**

## Success Criteria

- CAS ops in clustered mode return correct new ModifyIndex
- Batch CAS atomic — all-or-nothing across cluster
- `GET /kv/:key?consistent=true` returns linearizable data
- All 11 consistency tests + 10 batch tests passing
