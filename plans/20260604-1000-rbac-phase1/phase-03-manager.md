# Phase 03 — RoleManager + Store Tests

**Links:** [plan.md](plan.md) | [phase-01](phase-01-rbac-types-store-metrics.md) | [phase-02](phase-02-config.md)
**Wave:** 2 (parallel with Phase 04)

## Parallelization Info

- Starts after Wave 1 (Phase 01 + Phase 02) complete.
- Runs concurrently with Phase 04.
- Phase 05 depends on this phase completing.
- Phase 04 does NOT depend on this phase — it uses a mock of the `RoleManager` interface from Phase 01.

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-06-04 |
| Priority | P1 — core business logic |
| Status | pending |

Implements `RoleManager` (satisfying the interface from `types.go`) and `store_test.go`.

## Key Insights

- DFS cycle detection: track visited set during recursive traversal; return `ErrCyclicDependency` if a node is visited twice.
- Max depth 5: pass depth counter through recursion; return `ErrMaxDepthExceeded` at depth > 5.
- TTL cache: `map[string]cacheEntry` where `cacheEntry = {policies []string, expiresAt time.Time}`. Guarded by `sync.RWMutex`.
- Background expiration goroutine: ticker at `cfg.ExpirationCheckInterval`, scans all assignments, removes expired ones, updates `RBACAssignmentsExpiredTotal` metric.
- `GetEffectivePolicies(userID)`: (1) get assignment for userID, (2) for each role in assignment, call `ResolveInheritance(roleName, visited, depth)` which returns merged policy slice, (3) cache result, (4) return.
- `Authorize(userID, resource, capability)`: call `GetEffectivePolicies` → pass result to `acl.Evaluator.Evaluate()`.
- Cache invalidation: any write operation (CreateRole, DeleteRole, AssignRole, UnassignRole) clears the entire cache (simple strategy, acceptable for Phase 1).

## Requirements

### `manager.go` — `RoleManager` struct

```go
type RoleManager struct {
    roles       RoleStore
    assignments AssignmentStore
    evaluator   *acl.Evaluator
    cache       map[string]cacheEntry
    cacheMu     sync.RWMutex
    cacheTTL    time.Duration
    stopCh      chan struct{}
    log         logger.Logger
}
```

Constructor: `NewRoleManager(roles RoleStore, assignments AssignmentStore, evaluator *acl.Evaluator, cacheTTL time.Duration, expirationInterval time.Duration, log logger.Logger) *RoleManager`

Starts background goroutine in constructor; `Close()` method sends to `stopCh`.

### Key Methods

| Method | Behavior |
|--------|----------|
| `GetEffectivePolicies(userID string) ([]string, error)` | Cache-first; resolve assignments → inheritance |
| `ResolveInheritance(roleName string, visited map[string]bool, depth int) ([]string, error)` | DFS, max depth 5, cycle detect |
| `Authorize(userID string, resource acl.Resource, capability acl.Capability) (bool, error)` | GetEffectivePolicies → evaluator.Evaluate |
| `CreateRole(role *Role) error` | Validate, SetRole, clear cache, update metric |
| `GetRole(name string) (*Role, error)` | Delegate to store |
| `DeleteRole(name string) error` | Delete from store, clear cache, update metric |
| `AssignRole(userID string, roleNames []string, expiresAt *time.Time) error` | SetAssignment, clear cache, update metric |
| `UnassignRole(userID string) error` | DeleteAssignment, clear cache, update metric |
| `ListRoles() ([]*Role, error)` | Delegate to store |

### `store_test.go`

Tests for `MemoryRoleStore` and `MemoryAssignmentStore` (in-memory impls). BadgerDB impls tested with a temp directory.

## Architecture

```
internal/rbac/
├── manager.go      ← RoleManager struct + all methods
└── store_test.go   ← store CRUD + persistence round-trip
```

**Cache entry**:
```go
type cacheEntry struct {
    policies  []string
    expiresAt time.Time
}
```

**ResolveInheritance** algorithm:
```
func resolveInheritance(roleName, visited, depth):
    if depth > 5: return ErrMaxDepthExceeded
    if visited[roleName]: return ErrCyclicDependency
    visited[roleName] = true
    role = store.GetRole(roleName)
    policies = role.Policies (copy)
    for each parent in role.ParentRoles:
        parentPolicies = resolveInheritance(parent, visited, depth+1)
        policies = union(policies, parentPolicies)
    return deduped policies
```

Note: `visited` map is passed by value or cloned at each branch to allow diamond inheritance (A→B, A→C, B→D, C→D — D visited twice but through different paths is valid, only true cycles are invalid).

## File Ownership

Exclusive files owned by this phase:
- `internal/rbac/manager.go` (NEW)
- `internal/rbac/store_test.go` (NEW)

## Implementation Steps

1. **`manager.go`**:
   a. Define `cacheEntry` struct.
   b. Define `RoleManager` struct with all fields.
   c. Implement `NewRoleManager()` — init cache map, start `go r.runExpirationLoop(interval)`.
   d. Implement `Close()` — close `stopCh`.
   e. Implement `runExpirationLoop(interval)` — ticker loop, call `expireAssignments()` on each tick.
   f. Implement `expireAssignments()` — list all assignments, remove expired, increment `RBACAssignmentsExpiredTotal`.
   g. Implement `invalidateCache()` — clear entire cache map under write lock.
   h. Implement `GetEffectivePolicies()` — check cache (read lock), on miss: resolve + store in cache (write lock).
   i. Implement `resolveForUser(userID)` — get assignment, iterate roles, call `resolveInheritance` for each.
   j. Implement `resolveInheritance(name, visited map[string]bool, depth int)` — DFS with cycle/depth guards.
   k. Implement `Authorize()` — GetEffectivePolicies → evaluator.Evaluate → record duration metric.
   l. Implement CRUD methods: CreateRole, GetRole, DeleteRole, AssignRole, UnassignRole, ListRoles.
   m. Update metrics in each mutation (RBACRolesTotal, RBACAssignmentsTotal).

2. **`store_test.go`**:
   a. `TestMemoryRoleStore_CRUD` — create, get, list, delete, get-after-delete → ErrRoleNotFound.
   b. `TestMemoryAssignmentStore_CRUD` — same for assignments.
   c. `TestBadgerRoleStore_Persistence` — create BadgerEngine with `t.TempDir()`, store role, close, reopen, verify role still present.
   d. `TestBadgerAssignmentStore_Persistence` — same for assignments.
   e. `TestMemoryRoleStore_DuplicateCreate` — expect `ErrRoleExists`.

## Todo

- [ ] `manager.go`: cacheEntry struct
- [ ] `manager.go`: RoleManager struct
- [ ] `manager.go`: NewRoleManager constructor
- [ ] `manager.go`: Close() method
- [ ] `manager.go`: runExpirationLoop + expireAssignments
- [ ] `manager.go`: invalidateCache
- [ ] `manager.go`: GetEffectivePolicies (cache-first)
- [ ] `manager.go`: resolveForUser
- [ ] `manager.go`: resolveInheritance (DFS, cycle detect, depth limit)
- [ ] `manager.go`: Authorize
- [ ] `manager.go`: CreateRole, GetRole, DeleteRole
- [ ] `manager.go`: AssignRole, UnassignRole, ListRoles
- [ ] `store_test.go`: MemoryRoleStore CRUD tests
- [ ] `store_test.go`: MemoryAssignmentStore CRUD tests
- [ ] `store_test.go`: Badger persistence round-trip tests
- [ ] `store_test.go`: duplicate create error test
- [ ] `go test ./internal/rbac/... -run TestMemory` passes
- [ ] `go test ./internal/rbac/... -run TestBadger` passes

## Success Criteria

- `RoleManager` satisfies the `RoleManager` interface defined in `types.go` (compile check).
- Linear inheritance resolves correctly: A→B→C returns merged policies of A+B+C.
- Diamond inheritance (A→B, A→C, B→D, C→D) resolves D's policies once (deduplication).
- Cycle (A→B→A) returns `ErrCyclicDependency`.
- Depth > 5 returns `ErrMaxDepthExceeded`.
- Expired assignments are removed by background goroutine.
- Cache returns stale data within TTL; invalidated after any write.
- Badger store round-trip: role survives store close + reopen.

## Conflict Prevention

- Owns only new files in `internal/rbac/` — no modification of existing files.
- Does not touch `internal/middleware/acl.go` or `cmd/konsul/main.go` (Phase 05 owns those).
- Uses `*acl.Evaluator` from existing package — no modifications to `internal/acl/`.
- `store_test.go` tests only store interfaces, not manager logic (manager tests are in Phase 05's `manager_test.go`).

## Risk Assessment

- **Medium**: DFS algorithm for cycle detection requires careful visited-map cloning for diamond inheritance.
- **Watch**: `persistence.Engine.List(prefix)` return format — verify whether keys are returned with prefix or without before writing `BadgerRoleStore.ListRoles()`.
- **Watch**: Background goroutine must be stopped cleanly in tests (call `manager.Close()` in `t.Cleanup()`).

## Security Considerations

- `ResolveInheritance` max depth 5 prevents unbounded recursion / DoS via deeply nested roles.
- Cycle detection prevents infinite loops.
- Cache entries carry expiry — stale cache cannot outlive `CacheTTL`.

## Next Steps

Phase 05 imports `RoleManager` and wires it into middleware + main.go. Phase 05 also writes `manager_test.go` for full integration testing of the manager.
