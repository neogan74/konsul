# Phase 05 — Middleware Integration + main.go Wiring + Manager Tests

**Links:** [plan.md](plan.md) | [phase-03](phase-03-manager.md) | [phase-04](phase-04-handlers.md)
**Wave:** 3 (sequential — runs after Phases 03 + 04 complete)

## Parallelization Info

- Must run after both Phase 03 and Phase 04 complete.
- Sequential — single agent only.
- This is the integration phase: touches existing production files.

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-06-04 |
| Priority | P1 — makes RBAC active in the system |
| Status | pending |

Three concerns handled sequentially:
1. Middleware: augment `claims.Policies` with RBAC-resolved policies in `acl.go`.
2. Wiring: instantiate `RoleManager`, register routes in `main.go`.
3. Manager tests: full integration tests for `RoleManager` business logic.

## Key Insights

- Middleware change is **additive and guarded**: only runs if `cfg.RBAC.Enabled`. Original `claims.Policies` preserved; RBAC-resolved policies are merged (union, deduplicated).
- `main.go` wiring follows existing pattern: instantiate after ACL evaluator (which is passed to `NewRoleManager`), before route registration.
- `manager_test.go` tests the `RoleManager` directly (not via HTTP), using in-memory stores — no BadgerDB, no Raft.
- Middleware must not panic if `RoleManager` is nil (e.g., RBAC disabled).

## Requirements

### `internal/middleware/acl.go` — RBAC policy augmentation

Add a new exported function (not modify existing `ACLMiddleware`):
```go
// RBACPolicyAugmentor returns middleware that merges RBAC-resolved policies
// into claims.Policies before ACL evaluation.
func RBACPolicyAugmentor(manager rbac.RoleManager) fiber.Handler {
    return func(c *fiber.Ctx) error {
        claims := GetClaims(c)
        if claims == nil || manager == nil {
            return c.Next()
        }
        rbacPolicies, err := manager.GetEffectivePolicies(claims.UserID)
        if err == nil && len(rbacPolicies) > 0 {
            claims.Policies = mergePolicies(claims.Policies, rbacPolicies)
            c.Locals("claims", claims)
        }
        return c.Next()
    }
}

func mergePolicies(a, b []string) []string {
    // union deduplication via map
}
```

**Insert point in middleware chain**: After `JWTAuth`, before `ACLMiddleware`.

### `cmd/konsul/main.go` — RBAC wiring

After ACL evaluator initialization (~line 390), add:
```go
var roleManager *rbac.RoleManager
if cfg.RBAC.Enabled {
    var roleStore rbac.RoleStore
    var assignStore rbac.AssignmentStore
    if cfg.Persistence.Enabled && engine != nil {
        roleStore = rbac.NewBadgerRoleStore(engine)
        assignStore = rbac.NewBadgerAssignmentStore(engine)
    } else {
        roleStore = rbac.NewMemoryRoleStore()
        assignStore = rbac.NewMemoryAssignmentStore()
    }
    rm := rbac.NewRoleManager(roleStore, assignStore, aclEvaluator,
        cfg.RBAC.CacheTTL, cfg.RBAC.ExpirationCheckInterval, appLogger)
    roleManager = &rm
    defer func() { rm.Close() }()
}
```

Route group (after ACL routes):
```go
if cfg.RBAC.Enabled {
    rbacHandler := handlers.NewRBACHandler(*roleManager, appLogger)
    rbacRoutes := app.Group("/rbac")
    if cfg.Auth.Enabled {
        rbacRoutes.Use(middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths))
        rbacRoutes.Use(middleware.ACLMiddleware(aclEvaluator, acl.ResourceTypeAdmin, acl.CapabilityWrite))
    }
    rbacRoutes.Post("/roles", rbacHandler.CreateRole)
    rbacRoutes.Get("/roles", rbacHandler.ListRoles)
    rbacRoutes.Get("/roles/:name", rbacHandler.GetRole)
    rbacRoutes.Delete("/roles/:name", rbacHandler.DeleteRole)
    rbacRoutes.Post("/assignments", rbacHandler.AssignRole)
    rbacRoutes.Get("/assignments/:userID", rbacHandler.GetAssignment)
    rbacRoutes.Delete("/assignments/:userID", rbacHandler.UnassignRole)
}
```

Augment existing protected routes to include RBAC policy augmentor:
```go
// In KV route group (and other protected groups):
if cfg.Auth.Enabled && cfg.RBAC.Enabled && roleManager != nil {
    kvRoutes.Use(middleware.RBACPolicyAugmentor(*roleManager))
}
```

### `internal/rbac/manager_test.go`

Full test coverage for `RoleManager` business logic.

## Architecture

Changes span 3 files:
```
internal/middleware/acl.go    ← additive: new function RBACPolicyAugmentor + mergePolicies
cmd/konsul/main.go            ← additive: RBAC init block + route group + augmentor in protected groups
internal/rbac/manager_test.go ← NEW file: manager integration tests
```

## File Ownership

Exclusive files owned by this phase:
- `internal/middleware/acl.go` (MODIFIED — additive only)
- `cmd/konsul/main.go` (MODIFIED — additive only)
- `internal/rbac/manager_test.go` (NEW)

## Implementation Steps

1. **`internal/middleware/acl.go`**:
   a. Add import `"github.com/neogan74/konsul/internal/rbac"`.
   b. Add `mergePolicies(a, b []string) []string` (unexported helper).
   c. Add `RBACPolicyAugmentor(manager rbac.RoleManager) fiber.Handler`.
   d. Do NOT modify existing `ACLMiddleware` or `DynamicACLMiddleware` functions.

2. **`internal/rbac/manager_test.go`**:
   a. `TestRoleManager_CreateAndGet` — create role, get it back, verify fields.
   b. `TestRoleManager_ListRoles` — create 3 roles, list, verify count.
   c. `TestRoleManager_DeleteRole` — create + delete, get → ErrRoleNotFound.
   d. `TestRoleManager_LinearInheritance` — roles A(policies:[p1]) → B(policies:[p2]) → C(policies:[p3]); GetEffectivePolicies for user with role C returns [p1,p2,p3].
   e. `TestRoleManager_DiamondInheritance` — A→B, A→C, B→D, C→D; GetEffectivePolicies returns D's+B's+C's+A's policies without duplicates.
   f. `TestRoleManager_CycleDetection` — A→B→A setup; CreateRole returns error OR GetEffectivePolicies returns ErrCyclicDependency.
   g. `TestRoleManager_MaxDepth` — chain of 6 roles (exceeds depth 5); GetEffectivePolicies returns ErrMaxDepthExceeded.
   h. `TestRoleManager_AssignmentExpiration` — create assignment with ExpiresAt 1ms in future; sleep briefly; call GetEffectivePolicies; expect empty/not-found.
   i. `TestRoleManager_CacheInvalidation` — assign role, GetEffectivePolicies (cached), modify role, GetEffectivePolicies again (should reflect change after invalidation).
   j. `TestRoleManager_CacheHit` — two consecutive GetEffectivePolicies calls; second should be served from cache (verify via RBACCacheHitRatio metric or mock timing).

   Helper:
   ```go
   func newTestManager(t *testing.T) *rbac.RoleManager {
       roles := rbac.NewMemoryRoleStore()
       assigns := rbac.NewMemoryAssignmentStore()
       // Use a real acl.Evaluator (no-op for policy tests is fine)
       evaluator := acl.NewEvaluator(logger.GetDefault())
       mgr := rbac.NewRoleManager(roles, assigns, evaluator, 5*time.Minute, time.Minute, logger.GetDefault())
       t.Cleanup(func() { mgr.Close() })
       return mgr
   }
   ```

3. **`cmd/konsul/main.go`**:
   a. Add import `"github.com/neogan74/konsul/internal/rbac"`.
   b. Add RBAC init block after ACL evaluator block.
   c. Add RBAC route group block after ACL routes block.
   d. Add `RBACPolicyAugmentor` to each protected route group (KV, service, health) guarded by `cfg.RBAC.Enabled`.

4. Run full test suite: `go test ./...` — verify no regressions.
5. Run linter: `golangci-lint run` — fix any lint issues.
6. Verify build: `make build`.

## Todo

- [ ] `acl.go`: add `mergePolicies` helper
- [ ] `acl.go`: add `RBACPolicyAugmentor` function
- [ ] `acl.go`: add `internal/rbac` import
- [ ] `manager_test.go`: newTestManager helper
- [ ] `manager_test.go`: TestRoleManager_CreateAndGet
- [ ] `manager_test.go`: TestRoleManager_ListRoles
- [ ] `manager_test.go`: TestRoleManager_DeleteRole
- [ ] `manager_test.go`: TestRoleManager_LinearInheritance
- [ ] `manager_test.go`: TestRoleManager_DiamondInheritance
- [ ] `manager_test.go`: TestRoleManager_CycleDetection
- [ ] `manager_test.go`: TestRoleManager_MaxDepth
- [ ] `manager_test.go`: TestRoleManager_AssignmentExpiration
- [ ] `manager_test.go`: TestRoleManager_CacheInvalidation
- [ ] `main.go`: import rbac package
- [ ] `main.go`: RBAC store selection (badger vs memory)
- [ ] `main.go`: NewRoleManager + defer Close
- [ ] `main.go`: RBAC route group (7 endpoints)
- [ ] `main.go`: RBACPolicyAugmentor in KV/service/health route groups
- [ ] `go test ./...` passes (no regressions)
- [ ] `golangci-lint run` passes
- [ ] `make build` succeeds

## Success Criteria

- `KONSUL_RBAC_ENABLED=false` (default): zero behavior change, all existing tests pass.
- `KONSUL_RBAC_ENABLED=true`: RBAC routes available, policies merged into claims before ACL evaluation.
- All 10 manager tests pass.
- `make build` produces working binary.
- `golangci-lint run` reports no new issues.

## Conflict Prevention

- `acl.go` changes are purely additive (new functions only) — existing `ACLMiddleware` unchanged.
- `main.go` changes are purely additive (new blocks inside `if cfg.RBAC.Enabled` guards) — all existing routes unchanged.
- `manager_test.go` is a new file — no overlap with `store_test.go` (Phase 03).
- RBAC route group uses `/rbac` prefix — no collision with `/acl`, `/kv`, `/services`, etc.
- `RBACPolicyAugmentor` is opt-in per route group — guarded by `cfg.RBAC.Enabled` check.

## Risk Assessment

- **Medium**: `main.go` is the largest and most critical file (~500+ lines). Edits must be surgical.
- **Watch**: Import cycle risk — verify `internal/middleware` can import `internal/rbac` without cycles. Current imports in `acl.go` are `internal/acl` and `github.com/gofiber/fiber/v2`. Adding `internal/rbac` is safe as long as `rbac` does not import `middleware`.
- **Watch**: `defer rm.Close()` must be placed correctly relative to other defers to avoid use-after-close.

## Security Considerations

- `RBACPolicyAugmentor` only augments — never removes from `claims.Policies`. Existing token-granted policies remain valid.
- Nil guard on `manager` in augmentor prevents panic when RBAC disabled mid-request.
- RBAC admin routes protected by `acl.ResourceTypeAdmin` + `acl.CapabilityWrite` — same protection as ACL policy management.
- `mergePolicies` deduplication prevents policy name amplification attacks.

## Next Steps

After Phase 05: RBAC Phase 1 is complete. Phase 2 scope includes CLI tooling, Web UI integration, and LDAP/OIDC group mapping (out of scope for this plan).
