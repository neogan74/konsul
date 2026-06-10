# Phase 01 — RBAC Types, Store, Metrics

**Links:** [plan.md](plan.md)
**Wave:** 1 (parallel with Phase 02)

## Parallelization Info

- Runs concurrently with Phase 02 (config changes).
- No dependencies on any other phase.
- Phase 03 and 04 both depend on this phase completing first.
- **Critical output:** `RoleManager` interface must be defined here in `types.go` so Phase 04 can mock it independently.

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-06-04 |
| Priority | P0 — foundation for all other phases |
| Status | pending |

Creates the entire `internal/rbac/` package foundation: data types, store interfaces + implementations, and Prometheus metrics.

## Key Insights

- `persistence.Engine` interface uses `Get/Set/Delete/List` with string keys — RBAC store wraps it using prefix `"rbac:role:"` and `"rbac:assign:"`, same pattern as `"kv:"` and `"svc:"` prefixes.
- Serialization: JSON marshal/unmarshal (consistent with service store pattern).
- `promauto` package-level vars — no explicit registration boilerplate.
- `RoleManager` interface is defined here (not in `manager.go`) so Phase 04 can reference it without importing manager implementation.

## Requirements

- `Role`: Name, Description, Policies[]string, ParentRoles[]string, CreatedAt, UpdatedAt
- `RoleAssignment`: UserID, RoleNames[]string, ExpiresAt *time.Time
- `GroupRoleMapping`: GroupID, RoleNames[]string
- Sentinel errors: `ErrRoleNotFound`, `ErrRoleExists`, `ErrAssignmentNotFound`, `ErrCyclicDependency`, `ErrMaxDepthExceeded`
- `RoleStore` interface: `GetRole`, `SetRole`, `DeleteRole`, `ListRoles`
- `AssignmentStore` interface: `GetAssignment`, `SetAssignment`, `DeleteAssignment`, `ListAssignments`
- In-memory implementations of both interfaces (used in tests, optionally wrapped by BadgerDB impl)
- BadgerDB-backed implementations wrapping `persistence.Engine`
- Metrics: `roles_total` (Gauge), `assignments_total` (Gauge), `authorization_duration_seconds` (Histogram), `cache_hit_ratio` (Gauge), `assignments_expired_total` (Counter)
- `RoleManager` interface (for Phase 04 mocking): `GetEffectivePolicies`, `Authorize`, `CreateRole`, `GetRole`, `DeleteRole`, `AssignRole`, `UnassignRole`, `ListRoles`

## Architecture

```
internal/rbac/
├── types.go       ← structs, errors, RoleManager interface
├── store.go       ← RoleStore + AssignmentStore interfaces + 2 impls each
└── metrics.go     ← promauto package vars
```

**Store implementation pattern** (mirrors existing persistence usage):
```go
// BadgerRoleStore wraps persistence.Engine
type BadgerRoleStore struct {
    engine persistence.Engine
}
const rolePrefix = "rbac:role:"

func (s *BadgerRoleStore) GetRole(name string) (*Role, error) {
    data, err := s.engine.Get(rolePrefix + name)
    // unmarshal JSON → *Role
}
func (s *BadgerRoleStore) ListRoles() ([]*Role, error) {
    keys, err := s.engine.List(rolePrefix)
    // fetch each key, unmarshal
}
```

**In-memory store** uses `sync.RWMutex` + `map[string]*Role` (no external deps, fast for tests).

## File Ownership

Exclusive files owned by this phase:
- `internal/rbac/types.go` (NEW)
- `internal/rbac/store.go` (NEW)
- `internal/rbac/metrics.go` (NEW)

## Implementation Steps

1. Create directory `internal/rbac/`.
2. **`types.go`**:
   a. Define `Role` struct with all fields + JSON tags.
   b. Define `RoleAssignment` struct (UserID, RoleNames, ExpiresAt).
   c. Define `GroupRoleMapping` struct.
   d. Define sentinel errors using `errors.New()`.
   e. Define `RoleManager` interface with 8 methods (see Requirements).
3. **`store.go`**:
   a. Define `RoleStore` interface (GetRole, SetRole, DeleteRole, ListRoles).
   b. Define `AssignmentStore` interface (GetAssignment, SetAssignment, DeleteAssignment, ListAssignments).
   c. Implement `MemoryRoleStore` with `sync.RWMutex` + map.
   d. Implement `MemoryAssignmentStore` with `sync.RWMutex` + map.
   e. Implement `BadgerRoleStore` wrapping `persistence.Engine` with `"rbac:role:"` prefix.
   f. Implement `BadgerAssignmentStore` wrapping `persistence.Engine` with `"rbac:assign:"` prefix.
   g. Add constructor functions: `NewMemoryRoleStore()`, `NewMemoryAssignmentStore()`, `NewBadgerRoleStore(engine)`, `NewBadgerAssignmentStore(engine)`.
4. **`metrics.go`**:
   a. Declare package-level vars using `promauto`.
   b. `RBACRolesTotal` — Gauge, `konsul_rbac_roles_total`.
   c. `RBACAssignmentsTotal` — Gauge, `konsul_rbac_assignments_total`.
   d. `RBACAuthorizationDuration` — HistogramVec, `konsul_rbac_authorization_duration_seconds`, labels: `["result"]`, buckets same as ACL: `[0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1]`.
   e. `RBACCacheHitRatio` — Gauge, `konsul_rbac_cache_hit_ratio`.
   f. `RBACAssignmentsExpiredTotal` — Counter, `konsul_rbac_assignments_expired_total`.

## Todo

- [ ] Create `internal/rbac/` directory
- [ ] `types.go`: Role struct with json tags
- [ ] `types.go`: RoleAssignment struct
- [ ] `types.go`: GroupRoleMapping struct
- [ ] `types.go`: sentinel errors (5 errors)
- [ ] `types.go`: RoleManager interface
- [ ] `store.go`: RoleStore interface
- [ ] `store.go`: AssignmentStore interface
- [ ] `store.go`: MemoryRoleStore impl
- [ ] `store.go`: MemoryAssignmentStore impl
- [ ] `store.go`: BadgerRoleStore impl
- [ ] `store.go`: BadgerAssignmentStore impl
- [ ] `store.go`: 4 constructor functions
- [ ] `metrics.go`: 5 promauto vars
- [ ] Verify `go build ./internal/rbac/...` passes

## Success Criteria

- `go build ./internal/rbac/` compiles without errors.
- `RoleManager` interface is exported and usable as a mock target.
- Both BadgerDB store impls correctly prepend/strip key prefixes.
- All 5 metrics use `konsul_rbac_` prefix, no naming collision with `internal/metrics/metrics.go`.

## Conflict Prevention

- This phase creates all-new files in a new directory — zero overlap with other phases.
- `RoleManager` interface in `types.go` is the only shared contract with Phase 04; it must not be changed after Phase 01 completes without coordinating with Phase 04.
- Do not import `internal/config` — Phase 02 owns that file. Manager uses raw values (TTL duration, etc.) passed at construction time.
- Metrics var names must not clash with `internal/metrics/metrics.go` ACL metrics (`ACLEvaluationsTotal` etc.).

## Risk Assessment

- **Low**: Pure new code, no modification of existing files.
- **Watch**: `persistence.Engine.List(prefix)` must return keys with prefix stripped or intact — verify behavior in `badger.go` before implementing `BadgerRoleStore`.

## Security Considerations

- Sentinel errors must not leak internal state in HTTP responses (handler layer handles translation).
- `RoleAssignment.ExpiresAt` must be checked by manager, never trusted from storage without validation.

## Next Steps

After Phase 01 completes: unblocks Phase 03 (manager) and Phase 04 (handlers mock).
