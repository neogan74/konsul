# BACK-006 RBAC Phase 1 — Core Engine

**Date:** 2026-06-04 | **Status:** Ready to implement

## Dependency Graph

```
┌─────────────────────────────────────────────────────┐
│  PARALLEL WAVE 1 (no dependencies)                  │
│  Phase 01: rbac/types.go + store.go + metrics.go    │
│  Phase 02: config/config.go (RBACConfig only)       │
└──────────────────┬──────────────────────────────────┘
                   │ both complete
        ┌──────────▼──────────────┐
        │  PARALLEL WAVE 2        │
        │  Phase 03: manager.go   │  Phase 04: handlers/rbac.go
        │          + store_test.go│          + handlers/rbac_test.go
        └──────────┬──────────────┘
                   │ both complete
        ┌──────────▼──────────────┐
        │  SEQUENTIAL WAVE 3      │
        │  Phase 05: middleware   │
        │    + main.go wiring     │
        │    + manager_test.go    │
        └─────────────────────────┘
```

## Execution Strategy

- **Wave 1** (parallel): Agents A + B work simultaneously on Phase 01 and Phase 02.
- **Wave 2** (parallel after Wave 1): Agents C + D work simultaneously on Phase 03 and Phase 04. Phase 04 uses mock `RoleManager` interface — no dependency on Phase 03 implementation.
- **Wave 3** (sequential after Wave 2): Single agent wires everything together.

## File Ownership Matrix

| File | Phase | Wave |
|------|-------|------|
| `internal/rbac/types.go` | 01 | 1 |
| `internal/rbac/store.go` | 01 | 1 |
| `internal/rbac/metrics.go` | 01 | 1 |
| `internal/config/config.go` | 02 | 1 |
| `internal/rbac/manager.go` | 03 | 2 |
| `internal/rbac/store_test.go` | 03 | 2 |
| `internal/handlers/rbac.go` | 04 | 2 |
| `internal/handlers/rbac_test.go` | 04 | 2 |
| `internal/middleware/acl.go` | 05 | 3 |
| `cmd/konsul/main.go` | 05 | 3 |
| `internal/rbac/manager_test.go` | 05 | 3 |

## Phase Overview

| # | Name | Wave | Depends On | Status |
|---|------|------|------------|--------|
| [01](phase-01-rbac-types-store-metrics.md) | RBAC Types + Store + Metrics | 1 | — | pending |
| [02](phase-02-config.md) | Config RBACConfig | 1 | — | pending |
| [03](phase-03-manager.md) | RoleManager + Store Tests | 2 | 01, 02 | pending |
| [04](phase-04-handlers.md) | RBAC Handlers + Handler Tests | 2 | 01, 02 | pending |
| [05](phase-05-wiring-tests.md) | Middleware + Wiring + Manager Tests | 3 | 03, 04 | pending |

## Key Constraints

- RBAC key prefixes: `"rbac:role:"`, `"rbac:assign:"` — no collision with `"kv:"`, `"svc:"`
- Middleware change is additive: RBAC-resolved policies are **merged** into `claims.Policies`, not replaced
- `RoleManager` interface defined in Phase 01 (`types.go`) so Phase 04 can mock it without Phase 03
- All new metrics use `konsul_rbac_` prefix to avoid collision with existing ACL metrics
