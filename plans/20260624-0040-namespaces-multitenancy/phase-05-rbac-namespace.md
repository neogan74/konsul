# Phase 05: RBAC Namespace Scoping

## Context Links
- Parent plan: [plan.md](plan.md)
- Depends on: [phase-01-namespace-core.md](phase-01-namespace-core.md)
- Research: [researcher-01-namespace-patterns.md](research/researcher-01-namespace-patterns.md)
- Scout: [scout-01-codebase-touchpoints.md](scout/scout-01-codebase-touchpoints.md)
- Key files: `internal/rbac/types.go` (67 lines), `internal/rbac/manager.go` (200+ lines), `internal/rbac/store.go` (150+ lines), `internal/handlers/rbac.go`

## Parallelization Info
- **Wave:** 2 (parallel with Phase 02 and Phase 03)
- **Can run in parallel with:** Phase 02, Phase 03
- **Must NOT touch:** `internal/store/` (Phase 02), `internal/raft/` (Phase 03), `internal/handlers/kv.go` etc. (Phase 04)
- **Depends on:** Phase 01 complete (namespace types for validation)
- **Note:** Phase 04 depends on this phase being complete before wiring rbac.go

## Overview
- **Priority:** High
- **Status:** pending
- **Description:** Add `Namespace string` to `Role` and `RoleAssignment`. Extend `RoleManager` interface methods with `namespace string` parameter. Update `Manager` implementation, `MemoryRoleStore`, and `BadgerRoleStore` to be namespace-aware. Update `handlers/rbac.go` to extract namespace from request context.

## Key Insights
- Consul RBAC: policies can be namespace-local or global; roles are namespace-scoped
- Konsul approach (KISS): add `Namespace` field to `Role`/`RoleAssignment`; all `RoleManager` methods take explicit `namespace string` param
- This IS a breaking interface change — all callers of `RoleManager` must pass namespace. Only 2-3 callers: `handlers/rbac.go` and auth middleware. Both are in files owned by this phase or Phase 04.
- Cache key change: `{subjectID}` → `{namespace}:{subjectID}` (prevent cache poisoning across namespaces)
- BadgerDB RBAC key format: `rbac:ns:<namespace>:role:<name>` and `rbac:ns:<namespace>:assign:<subjectID>`
- `MemoryRoleStore`: `map[string]*Role` → `map[string]map[string]*Role` (outer = namespace, inner = name)

## Requirements

**Functional:**
- Add `Namespace string \`json:"namespace,omitempty"\`` to `Role` struct
- Add `Namespace string \`json:"namespace,omitempty"\`` to `RoleAssignment` struct
- `RoleManager` interface: all methods gain `namespace string` as first parameter
- `Manager.GetEffectivePolicies(namespace, subjectID string)` — cache key = `namespace:subjectID`
- `Manager.Authorize(namespace, subjectID string, ...)` — resolves policies scoped to namespace
- All CRUD methods: `CreateRole(namespace string, role *Role)`, `GetRole(namespace, name string)`, etc.
- `MemoryRoleStore`: nested map structure
- `BadgerRoleStore`: namespace-prefixed keys
- `handlers/rbac.go`: extract namespace from `c.Locals("namespace")`, pass to manager

**Non-functional:**
- Default namespace `"default"` used when namespace field is empty (for backwards compat in store deserialization)
- All existing RBAC tests updated to pass namespace param
- Handler tests updated to include `X-Konsul-Namespace` header

## Architecture

### types.go Changes
```go
type Role struct {
    Namespace   string    `json:"namespace,omitempty"` // NEW
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Policies    []string  `json:"policies"`
    ParentRoles []string  `json:"parent_roles"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type RoleAssignment struct {
    Namespace string     `json:"namespace,omitempty"` // NEW
    SubjectID string     `json:"subject_id"`
    RoleNames []string   `json:"role_names"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Updated RoleManager interface
type RoleManager interface {
    GetEffectivePolicies(namespace, subjectID string) ([]string, error)
    Authorize(namespace, subjectID string, directPolicies []string, resource, capability string) bool
    CreateRole(namespace string, role *Role) error
    GetRole(namespace, name string) (*Role, error)
    UpdateRole(namespace string, role *Role) error
    DeleteRole(namespace, name string) error
    ListRoles(namespace string) ([]*Role, error)
    AssignRole(namespace, subjectID string, roleNames []string, expiresAt *time.Time) error
    UnassignRole(namespace, subjectID string) error
    ListAssignments(namespace string) ([]*RoleAssignment, error)
    GetAssignment(namespace, subjectID string) (*RoleAssignment, error)
}
```

### RoleStore / AssignmentStore Interface Changes (store.go)
```go
type RoleStore interface {
    CreateRole(namespace string, role *Role) error
    GetRole(namespace, name string) (*Role, error)
    UpdateRole(namespace string, role *Role) error
    DeleteRole(namespace, name string) error
    ListRoles(namespace string) ([]*Role, error)
}

type AssignmentStore interface {
    AssignRole(namespace, subjectID string, roleNames []string, expiresAt *time.Time) error
    UnassignRole(namespace, subjectID string) error
    ListAssignments(namespace string) ([]*RoleAssignment, error)
    GetAssignment(namespace, subjectID string) (*RoleAssignment, error)
}
```

### MemoryRoleStore Changes
```go
type MemoryRoleStore struct {
    mu    sync.RWMutex
    roles map[string]map[string]*Role // namespace → name → role
}

func (s *MemoryRoleStore) GetRole(namespace, name string) (*Role, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    if ns, ok := s.roles[namespace]; ok {
        if r, ok := ns[name]; ok {
            return r, nil
        }
    }
    return nil, ErrRoleNotFound
}
```

### BadgerRoleStore Key Format
- Role: `rbac:ns:<namespace>:role:<name>`
- Assignment: `rbac:ns:<namespace>:assign:<subjectID>`
- List: prefix scan `rbac:ns:<namespace>:role:` or `rbac:ns:<namespace>:assign:`

### Manager Cache Key
```go
func cacheKey(namespace, subjectID string) string {
    return namespace + ":" + subjectID
}
```

### handlers/rbac.go Changes
```go
func (h *RBACHandler) CreateRole(c *fiber.Ctx) error {
    ns := middleware.GetNamespace(c)
    var role rbac.Role
    // ... parse body
    return h.manager.CreateRole(ns, &role)
}
// Same pattern for all other RBAC handler methods
```

## File Ownership (exclusive to this phase)
- MODIFY `internal/rbac/types.go` — add Namespace to Role, RoleAssignment, update RoleManager interface
- MODIFY `internal/rbac/store.go` — update RoleStore, AssignmentStore interfaces + MemoryRoleStore, BadgerRoleStore implementations
- MODIFY `internal/rbac/manager.go` — update Manager to accept namespace params; update cache key
- MODIFY `internal/handlers/rbac.go` — extract namespace, pass to manager

**Do NOT touch:** any `internal/store/`, `internal/raft/`, other handler files

## Implementation Steps

1. **Modify `internal/rbac/types.go`**
   - Add `Namespace string` to `Role` (json:"namespace,omitempty")
   - Add `Namespace string` to `RoleAssignment` (json:"namespace,omitempty")
   - Update `RoleManager` interface: add `namespace string` as first param to all 10 methods

2. **Modify `internal/rbac/store.go`**
   - Update `RoleStore` interface: all methods + `namespace string` param
   - Update `AssignmentStore` interface: all methods + `namespace string` param
   - Update `MemoryRoleStore`:
     - Change `roles map[string]*Role` → `roles map[string]map[string]*Role`
     - Update all methods to use `roles[namespace][name]`
     - `ListRoles(namespace)` returns roles for that namespace only
   - Update `BadgerRoleStore`:
     - Change key format to `rbac:ns:<namespace>:role:<name>` and `rbac:ns:<namespace>:assign:<subjectID>`
     - Update all scan prefixes

3. **Modify `internal/rbac/manager.go`**
   - Update `Manager.GetEffectivePolicies(namespace, subjectID string)` — cache key = `namespace:subjectID`
   - Update all other `Manager` methods to accept and pass through `namespace`
   - Update cache eviction to use new cache key format
   - Update `Authorize` to use namespace-scoped policy resolution

4. **Modify `internal/handlers/rbac.go`**
   - Each handler method: add `ns := middleware.GetNamespace(c)` at top
   - Pass `ns` to all `h.manager.*` calls

5. **Update RBAC tests**
   - `internal/rbac/*_test.go`: add `"default"` as first arg to all method calls
   - `internal/handlers/rbac_test.go`: add `X-Konsul-Namespace` header in test requests (or rely on default)
   - Add new test: roles in namespace `"team-a"` not visible from namespace `"team-b"`

## Todo List
- [ ] Add `Namespace` field to `Role` and `RoleAssignment` in `types.go`
- [ ] Update `RoleManager` interface (10 methods)
- [ ] Update `RoleStore` interface (5 methods)
- [ ] Update `AssignmentStore` interface (4 methods)
- [ ] Update `MemoryRoleStore` — nested map, all methods
- [ ] Update `BadgerRoleStore` — new key format, all methods
- [ ] Update `Manager` — namespace params, cache key
- [ ] Update `handlers/rbac.go` — extract namespace in all handlers
- [ ] Update all RBAC unit tests
- [ ] Update RBAC handler tests
- [ ] `go test ./internal/rbac/... ./internal/handlers/...` passes
- [ ] `golangci-lint run ./internal/rbac/...` passes

## Success Criteria
- `CreateRole("team-a", &Role{Name: "admin"})` creates role in namespace `team-a`
- `GetRole("team-b", "admin")` returns `ErrRoleNotFound` (different namespace)
- Cache does not return stale cross-namespace results
- `ListRoles("team-a")` returns only team-a roles
- HTTP `GET /rbac/roles` with `X-Konsul-Namespace: team-a` returns only team-a roles
- All existing RBAC tests pass after adding `"default"` namespace param

## Conflict Prevention
- This phase owns `handlers/rbac.go` exclusively — Phase 04 does NOT touch it
- `internal/auth/jwt.go` is NOT modified here (auth namespace scoping is future work per YAGNI)
- No imports from `internal/store/` or `internal/raft/` (RBAC is independent layer)
- `middleware.GetNamespace(c)` is from `internal/middleware/namespace.go` (Phase 04) — if Phase 05 runs in parallel with Phase 04, Phase 05 can stub this call or add a simple inline extraction and reconcile in Phase 04

**Parallel conflict resolution**: Since Phase 05 runs before Phase 04 completes, `handlers/rbac.go` should inline namespace extraction temporarily:
```go
ns, _ := c.Locals("namespace").(string)
if ns == "" { ns = "default" }
```
Then Phase 04 replaces it with `middleware.GetNamespace(c)` — but since Phase 04 doesn't touch `handlers/rbac.go`, the inline stays. Add `middleware.GetNamespace` import in Phase 05 if Phase 01 middleware is available.

Actually, since `internal/middleware/namespace.go` is created in Phase 04, Phase 05 must either:
- Define `getNamespace` locally in `handlers/rbac.go` (one-liner), OR
- Wait for Phase 04's middleware to be merged before updating `rbac.go`

**Recommendation**: Phase 05 defines inline helper in `rbac.go`; Phase 04 can optionally update the import after merge.

## Risk Assessment
- **Medium**: Interface change to `RoleManager` affects all callers. Callers: `handlers/rbac.go` (owned here), auth middleware (`internal/middleware/auth.go` — check if it calls `RoleManager`). If auth middleware calls RBAC, it needs namespace param too.
- **Action**: Read `internal/middleware/auth.go` during implementation to check RBAC usage. If used, extract namespace from Fiber context there too.
- **Low**: `MemoryRoleStore` map structure change — existing tests will fail until updated. All tests must be updated in this phase.

## Security Considerations
- Namespace-scoped RBAC prevents privilege escalation: token with admin role in `team-a` cannot access `team-b` resources
- Cache must be namespace-keyed — cross-namespace cache hit would be a security bug
- `Authorize` check must use the request's namespace, not the token's namespace claim
- Future: add ACL policy to restrict which namespaces a token can access (out of scope for this plan per YAGNI)

## Next Steps
- Phase 06: integration tests validate namespace-scoped RBAC end-to-end
- Future: ACL token namespace restrictions (token can only access listed namespaces)
- Future: cross-namespace role inheritance (global roles visible in all namespaces)
