# Plan: Multi-Tenancy Namespace Support for Konsul

**Date:** 2026-06-24  
**Status:** Planning  
**Goal:** Add namespace isolation so teams operate in independent partitions without breaking existing clients.

## Execution Strategy

```text
Wave 1 (parallel):   Phase 01
Wave 2 (parallel):   Phase 02 | Phase 03 | Phase 05
Wave 3 (sequential): Phase 04
Wave 4 (sequential): Phase 06
```

## Phases

| # | Phase | Status | Wave | Deps | File |
| --- | --- | --- | --- | --- | --- |
| 01 | Namespace core types + store + persistence | pending | 1 | none | [phase-01](phase-01-namespace-core.md) |
| 02 | Store layer: KV prefix migration + NamespacedKV wrapper | pending | 2 | 01 | [phase-02](phase-02-store-namespace.md) |
| 03 | Raft commands + FSM namespace support | pending | 2 | 01 | [phase-03](phase-03-raft-namespace.md) |
| 04 | HTTP middleware + namespace extraction + handler updates | pending | 3 | 01,02,03 | [phase-04](phase-04-http-handlers.md) |
| 05 | RBAC namespace scoping | pending | 2 | 01 | [phase-05](phase-05-rbac-namespace.md) |
| 06 | CLI --namespace flag + integration tests + docs | pending | 4 | all | [phase-06](phase-06-cli-tests-docs.md) |

## File Ownership Matrix

| Phase | Owns (create/modify) |
| --- | --- |
| 01 | `internal/namespace/` (new), `internal/config/config.go` |
| 02 | `internal/store/kv.go`, `internal/store/service.go`, `internal/store/snapshot.go`, `internal/persistence/` (migration) |
| 03 | `internal/raft/commands.go`, `internal/raft/fsm.go`, `internal/raft/store_interfaces.go` |
| 04 | `internal/handlers/kv.go`, `internal/handlers/service.go`, `internal/handlers/batch.go`, `internal/handlers/health.go`, `internal/handlers/healthcheck.go`, `internal/handlers/kv_watch.go`, `internal/handlers/loadbalancer.go`, `internal/middleware/` (new namespace middleware), `cmd/konsul/main.go` |
| 05 | `internal/rbac/types.go`, `internal/rbac/manager.go`, `internal/rbac/store.go`, `internal/handlers/rbac.go` |
| 06 | `cmd/konsulctl/kv_commands.go`, `cmd/konsulctl/service_commands.go`, `cmd/konsulctl/client.go`, `docs/`, integration test files `*_integration_test.go` |

## Key Decisions

- KV keys stored as `ns:<ns>:<key>` in BadgerDB (flat map, prefix-scannable)
- HTTP: `X-Konsul-Namespace` header OR `?namespace=` query param; missing = `"default"`
- Store interface unchanged; `NamespacedKV`/`NamespacedService` wrappers handle prefix injection
- Raft commands: add `Namespace string \`json:"namespace,omitempty"\`` — empty deserializes to `"default"` in FSM
- BadgerDB startup migration: rewrite bare keys → `ns:default:<key>`, skip if `_migrated:v1` present
- RBAC: add `Namespace` to `Role`, `RoleAssignment`, all `RoleManager` methods
