# Phase 04 — RBAC Handlers + Handler Tests

**Links:** [plan.md](plan.md) | [phase-01](phase-01-rbac-types-store-metrics.md) | [phase-02](phase-02-config.md)
**Wave:** 2 (parallel with Phase 03)

## Parallelization Info

- Starts after Wave 1 (Phase 01 + Phase 02) complete.
- Runs concurrently with Phase 03.
- Does NOT depend on Phase 03 — uses a local mock of `RoleManager` interface from `types.go`.
- Phase 05 depends on this phase completing.

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-06-04 |
| Priority | P1 — REST API surface |
| Status | pending |

Implements REST CRUD endpoints for roles and role assignments, following the `ACLHandler` pattern exactly.

## Key Insights

- `RoleManager` interface is defined in `types.go` (Phase 01) — handler depends on the interface, not the concrete implementation. This enables full decoupling from Phase 03.
- Pattern: handler struct holds interface field, not concrete type.
- Error mapping: `ErrRoleNotFound` → 404, `ErrRoleExists` → 409, all others → 500.
- Tests inject a mock struct that implements `RoleManager` interface; no Raft, no BadgerDB in handler tests.
- Auth is bypassed in tests via logger injection pattern (same as `acl_test.go`).

## Requirements

### Endpoints

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| POST | `/rbac/roles` | CreateRole | admin write |
| GET | `/rbac/roles` | ListRoles | admin read |
| GET | `/rbac/roles/:name` | GetRole | admin read |
| DELETE | `/rbac/roles/:name` | DeleteRole | admin write |
| POST | `/rbac/assignments` | AssignRole | admin write |
| GET | `/rbac/assignments/:userID` | GetAssignment | admin read |
| DELETE | `/rbac/assignments/:userID` | UnassignRole | admin write |

### Request/Response Shapes

**CreateRole** `POST /rbac/roles`:
- Request: `{"name":"string","description":"string","policies":["p1"],"parent_roles":["r1"]}`
- Response 201: `{"message":"role created","role":{...}}`

**ListRoles** `GET /rbac/roles`:
- Response 200: `{"roles":[{...}],"count":N}`

**GetRole** `GET /rbac/roles/:name`:
- Response 200: Role object
- Response 404: `{"error":"role not found"}`

**DeleteRole** `DELETE /rbac/roles/:name`:
- Response 200: `{"message":"role deleted","name":"..."}`

**AssignRole** `POST /rbac/assignments`:
- Request: `{"user_id":"string","role_names":["r1","r2"],"expires_at":"2026-01-01T00:00:00Z"}` (expires_at optional)
- Response 201: `{"message":"roles assigned","user_id":"..."}`

**GetAssignment** `GET /rbac/assignments/:userID`:
- Response 200: RoleAssignment object
- Response 404 if not found

**UnassignRole** `DELETE /rbac/assignments/:userID`:
- Response 200: `{"message":"roles unassigned","user_id":"..."}`

## Architecture

```
internal/handlers/
├── rbac.go       ← RBACHandler struct + 7 handler methods
└── rbac_test.go  ← mock RoleManager + 7+ test functions
```

**Handler struct**:
```go
type RBACHandler struct {
    manager rbac.RoleManager  // interface, not concrete type
    log     logger.Logger
}

func NewRBACHandler(manager rbac.RoleManager, log logger.Logger) *RBACHandler
```

**Mock for tests** (defined in `rbac_test.go`):
```go
type mockRoleManager struct {
    roles       map[string]*rbac.Role
    assignments map[string]*rbac.RoleAssignment
}
// implements all RoleManager interface methods
```

## File Ownership

Exclusive files owned by this phase:
- `internal/handlers/rbac.go` (NEW)
- `internal/handlers/rbac_test.go` (NEW)

## Implementation Steps

1. **`rbac.go`**:
   a. Define `RBACHandler` struct with `manager rbac.RoleManager` and `log logger.Logger`.
   b. `NewRBACHandler(manager rbac.RoleManager, log logger.Logger) *RBACHandler`.
   c. `CreateRole(c *fiber.Ctx) error`:
      - Parse body into `rbac.Role`.
      - Validate: name required.
      - Call `h.manager.CreateRole(&role)`.
      - Map errors: `rbac.ErrRoleExists` → `middleware.Conflict`, others → `middleware.InternalError`.
      - Return 201 with `{message, role}`.
   d. `ListRoles(c *fiber.Ctx) error`:
      - Call `h.manager.ListRoles()`.
      - Return 200 with `{roles, count}`.
   e. `GetRole(c *fiber.Ctx) error`:
      - `name := c.Params("name")`.
      - Call `h.manager.GetRole(name)`.
      - Map `rbac.ErrRoleNotFound` → `middleware.NotFound`.
      - Return 200 with role.
   f. `DeleteRole(c *fiber.Ctx) error`:
      - Get name param, call `h.manager.DeleteRole(name)`.
      - Map errors, return 200.
   g. `AssignRole(c *fiber.Ctx) error`:
      - Parse body: `struct{ UserID string; RoleNames []string; ExpiresAt *time.Time }`.
      - Validate: UserID and RoleNames required.
      - Call `h.manager.AssignRole(req.UserID, req.RoleNames, req.ExpiresAt)`.
      - Return 201.
   h. `GetAssignment(c *fiber.Ctx) error`:
      - `userID := c.Params("userID")`.
      - Call `h.manager.GetAssignment(userID)` — note: need this on the interface or use a narrower method.
        - Actually use `GetEffectivePolicies` is not right here. Need to add `GetAssignment(userID) (*RoleAssignment, error)` to the `RoleManager` interface in `types.go` (Phase 01 must include this). **Coordination note**: Phase 01 must include `GetAssignment` in the `RoleManager` interface.
      - Map `ErrAssignmentNotFound` → NotFound.
      - Return 200 with assignment.
   i. `UnassignRole(c *fiber.Ctx) error`:
      - Get userID param, call `h.manager.UnassignRole(userID)`.
      - Return 200.

2. **`rbac_test.go`**:
   a. Define `mockRoleManager` struct implementing full `RoleManager` interface.
   b. `setupRBACHandler()` helper: creates mock + handler + fiber.App with logger injected.
   c. `TestRBACHandler_CreateRole` — valid body → 201.
   d. `TestRBACHandler_CreateRole_InvalidJSON` — bad body → 400.
   e. `TestRBACHandler_CreateRole_Duplicate` — mock returns `ErrRoleExists` → 409.
   f. `TestRBACHandler_ListRoles` — returns list with count.
   g. `TestRBACHandler_GetRole` — existing role → 200; missing → 404.
   h. `TestRBACHandler_DeleteRole` — 200 on success.
   i. `TestRBACHandler_AssignRole` — valid body → 201; missing UserID → 400.
   j. `TestRBACHandler_GetAssignment` — existing → 200; missing → 404.
   k. `TestRBACHandler_UnassignRole` — 200 on success.

## Todo

- [ ] `rbac.go`: RBACHandler struct
- [ ] `rbac.go`: NewRBACHandler constructor
- [ ] `rbac.go`: CreateRole handler
- [ ] `rbac.go`: ListRoles handler
- [ ] `rbac.go`: GetRole handler
- [ ] `rbac.go`: DeleteRole handler
- [ ] `rbac.go`: AssignRole handler
- [ ] `rbac.go`: GetAssignment handler
- [ ] `rbac.go`: UnassignRole handler
- [ ] `rbac_test.go`: mockRoleManager implementing full interface
- [ ] `rbac_test.go`: setupRBACHandler helper
- [ ] `rbac_test.go`: 9+ test functions covering all handlers
- [ ] Verify `go build ./internal/handlers/...` passes
- [ ] Verify `go test ./internal/handlers/... -run TestRBAC` passes

## Success Criteria

- All 7 handlers compile and respond with correct HTTP status codes.
- Error mapping: `ErrRoleExists`→409, `ErrRoleNotFound`→404, `ErrAssignmentNotFound`→404.
- Handler tests use mock — no real BadgerDB or Raft in scope.
- `go test ./internal/handlers/... -run TestRBAC` passes.

## Conflict Prevention

- Owns only two new files in `internal/handlers/` — no modification of existing handler files.
- Does not modify `internal/middleware/acl.go` or `cmd/konsul/main.go` (Phase 05).
- Does not import `internal/rbac/manager.go` — only imports `internal/rbac` package for types and interface.
- **Coordination with Phase 01**: `RoleManager` interface must include `GetAssignment(userID string) (*RoleAssignment, error)` — communicate this requirement to Phase 01 agent before implementation starts.

## Risk Assessment

- **Low**: Pure new files following existing handler pattern.
- **Watch**: `RoleManager` interface definition in Phase 01 must include `GetAssignment` method — if Phase 01 misses this, add it and notify Phase 03 to implement.

## Security Considerations

- All `/rbac/*` routes will require admin ACL capability (wired in Phase 05). Handler tests bypass auth intentionally for isolation.
- Handler never exposes internal error details — all errors mapped through `middleware.*` helpers.
- `ExpiresAt` field: accept RFC3339 time string, parse with `time.Parse(time.RFC3339, ...)`, reject invalid formats with 400.

## Next Steps

Phase 05 registers the `RBACHandler` routes in `main.go` with JWT + ACL middleware applied at the group level.
