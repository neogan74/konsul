# Phase 04: HTTP Middleware + Namespace Extraction + Handler Updates

## Context Links
- Parent plan: [plan.md](plan.md)
- Depends on: Phase 01, 02, 03 (all Wave 2 phases must be complete)
- Research: [researcher-01-namespace-patterns.md](research/researcher-01-namespace-patterns.md)
- Research: [researcher-02-go-technical-patterns.md](research/researcher-02-go-technical-patterns.md)
- Scout: [scout-01-codebase-touchpoints.md](scout/scout-01-codebase-touchpoints.md)

## Parallelization Info
- **Wave:** 3 (sequential — depends on all Wave 2 phases)
- **Can run in parallel with:** nothing in this wave (single phase)
- **Depends on:** Phase 01 (namespace types), Phase 02 (store wrappers), Phase 03 (Raft commands with Namespace field)

## Overview
- **Priority:** Critical
- **Status:** pending
- **Description:** Add Fiber namespace middleware, update all HTTP handlers to extract namespace from request context and construct `NamespacedKV`/`NamespacedService` wrappers per-request. Add Namespace CRUD HTTP endpoints. Wire everything in `main.go`. Add startup migration call.

## Key Insights
- Fiber's `c.Locals("namespace", ns)` is the propagation mechanism — type assertion `.(string)` in handlers
- Middleware runs once per request; all handlers read from `c.Locals`
- Per-request wrapper construction: `store.NewNamespacedKV(h.store, ns)` is cheap (no allocation besides struct)
- KV watch and health handlers also need namespace — cannot skip
- `main.go` wires namespace middleware before all routes and calls `MigrateToNamespacedKeys` at startup
- Namespace validation at middleware boundary: invalid names → 400 before handler runs
- Namespace existence check: optional (lazy — non-existent namespace returns empty results, same as Consul's behavior)

## Requirements

**Functional:**
- `internal/middleware/namespace.go` — new Fiber middleware
  - Reads `X-Konsul-Namespace` header, falls back to `?namespace=` query param, defaults to `"default"`
  - Validates name with `namespace.ValidateName()`
  - Stores in `c.Locals("namespace", ns)`
- All KV handlers: extract namespace, construct `NamespacedKV`, use wrapper
- All service handlers: extract namespace, construct `NamespacedService`, use wrapper
- Batch handlers: extract namespace, apply to all items
- KV watch handler: namespace-scoped watch
- Health/healthcheck handlers: filter services by namespace
- Load balancer handler: filter by namespace
- Namespace CRUD handler (`internal/handlers/namespace.go`): Create/List/Delete namespaces
- Route registration in `main.go`: mount namespace middleware, register `/namespaces` routes
- Startup migration: call `persistence.MigrateToNamespacedKeys(db)` before store init

**Non-functional:**
- Zero breaking change for clients that don't send namespace header (get "default")
- Missing namespace header = "default" (not 400)
- Invalid namespace name = 400 Bad Request with clear error message

## Architecture

### Middleware (internal/middleware/namespace.go)
```go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/neogan74/konsul/internal/namespace"
)

func Namespace() fiber.Handler {
    return func(c *fiber.Ctx) error {
        ns := c.Get("X-Konsul-Namespace")
        if ns == "" {
            ns = c.Query("namespace", "default")
        }
        if err := namespace.ValidateName(ns); err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "invalid namespace: " + err.Error(),
            })
        }
        c.Locals("namespace", ns)
        return c.Next()
    }
}

// GetNamespace extracts namespace from Fiber context (helper for handlers)
func GetNamespace(c *fiber.Ctx) string {
    if ns, ok := c.Locals("namespace").(string); ok && ns != "" {
        return ns
    }
    return "default"
}
```

### KV Handler Changes (internal/handlers/kv.go)
```go
func (h *KVHandler) Get(c *fiber.Ctx) error {
    ns := middleware.GetNamespace(c)
    nkv := store.NewNamespacedKV(h.store, ns)
    key := c.Params("key")
    val, ok := nkv.Get(key)
    // ... rest unchanged
}

func (h *KVHandler) Set(c *fiber.Ctx) error {
    ns := middleware.GetNamespace(c)
    // For Raft: pass ns in payload
    if h.isRaftEnabled() {
        cmd, _ := raft.NewCommand(raft.CmdKVSet, raft.KVSetPayload{
            Namespace: ns, Key: key, Value: value,
        })
        // apply via raft
        return nil
    }
    // Standalone: use NamespacedKV wrapper
    nkv := store.NewNamespacedKV(h.store, ns)
    nkv.Set(key, value)
}
```

### Namespace Handler (internal/handlers/namespace.go)
```go
type NamespaceHandler struct {
    store    namespace.NamespaceStore
    raftNode *konsulraft.Node
}

// POST /namespaces — create
// GET  /namespaces — list
// DELETE /namespaces/:name — delete
```

### Route Registration in main.go
```go
// After existing middleware setup:
app.Use(middleware.Namespace())

// Namespace CRUD routes
nsHandler := handlers.NewNamespaceHandler(nsStore, raftNode)
app.Get("/namespaces", nsHandler.List)
app.Post("/namespaces", nsHandler.Create)
app.Delete("/namespaces/:name", nsHandler.Delete)
```

### Startup migration in main.go
```go
if cfg.Persistence.Enabled && cfg.Persistence.Type == "badger" {
    if err := persistence.MigrateToNamespacedKeys(badgerDB); err != nil {
        log.Fatalf("Namespace migration failed: %v", err)
    }
}
```

## File Ownership (exclusive to this phase)
- CREATE `internal/middleware/namespace.go`
- CREATE `internal/middleware/namespace_test.go`
- CREATE `internal/handlers/namespace.go`
- CREATE `internal/handlers/namespace_test.go`
- MODIFY `internal/handlers/kv.go` — extract namespace, use NamespacedKV
- MODIFY `internal/handlers/kv_watch.go` — namespace-scoped watch
- MODIFY `internal/handlers/service.go` — extract namespace, use NamespacedService
- MODIFY `internal/handlers/batch.go` — extract namespace for all batch ops
- MODIFY `internal/handlers/health.go` — filter by namespace
- MODIFY `internal/handlers/healthcheck.go` — filter by namespace
- MODIFY `internal/handlers/loadbalancer.go` — filter by namespace
- MODIFY `cmd/konsul/main.go` — namespace middleware, namespace routes, startup migration

**Do NOT touch:** `internal/rbac/` (Phase 05 owns `internal/handlers/rbac.go` too — see below)

**Special note on `internal/handlers/rbac.go`**: This file is owned by Phase 05 since RBAC handler needs RBAC-specific namespace changes. Phase 04 does NOT modify it.

## Implementation Steps

1. **Create `internal/middleware/namespace.go`**
   - `Namespace() fiber.Handler` — header → query → "default" fallback
   - `GetNamespace(c *fiber.Ctx) string` helper
   - Validate with `namespace.ValidateName(ns)` (ValidateName allows "default")

2. **Create `internal/middleware/namespace_test.go`**
   - Test header takes precedence over query param
   - Test missing header + param → "default"
   - Test invalid name → 400
   - Test valid custom namespace stored in Locals

3. **Modify `internal/handlers/kv.go`**
   - `Get`: extract ns, use `NamespacedKV` for read
   - `Set`: extract ns, include in Raft payload OR use `NamespacedKV` for standalone
   - `Delete`: same pattern
   - `List`: extract ns, use `NamespacedKV.List`
   - `SetWithFlags`, `SetCAS`, `DeleteCAS`, all CAS variants: same ns extraction

4. **Modify `internal/handlers/service.go`**
   - `Register`: extract ns, set `service.Namespace = ns`, include in Raft payload
   - `Deregister`: extract ns, use in Raft payload
   - `Get`, `List`: extract ns, use `NamespacedService`
   - `Heartbeat`, query endpoints: same pattern

5. **Modify `internal/handlers/batch.go`**
   - Extract ns once at top of each handler
   - Pass ns to all Raft command payloads or `NamespacedKV` batch calls

6. **Modify `internal/handlers/kv_watch.go`**
   - Extract ns from context
   - Filter watch events to only emit events for keys under `ns:<ns>:` prefix
   - Strip prefix before sending to client (client shouldn't see internal prefix)

7. **Modify `internal/handlers/health.go` and `healthcheck.go`**
   - Filter service health results by namespace using `NamespacedService`

8. **Modify `internal/handlers/loadbalancer.go`**
   - Filter service selection to `ns:<ns>:` prefix

9. **Create `internal/handlers/namespace.go`**
   - `NamespaceHandler` with `nsStore namespace.NamespaceStore` and `raftNode`
   - `List(c) error` — return all namespaces as JSON array
   - `Create(c) error` — parse body `{name, description}`, validate, create; if Raft: apply `CmdNamespaceCreate`
   - `Delete(c) error` — parse `:name` param, reject "default", delete; if Raft: apply `CmdNamespaceDelete`

10. **Modify `cmd/konsul/main.go`**
    - Call `persistence.MigrateToNamespacedKeys(db)` after BadgerDB init, before store construction
    - Create `NamespaceStore` (BadgerDB or memory depending on config)
    - Add `app.Use(middleware.Namespace())` before route registration (after auth middleware)
    - Create and register `NamespaceHandler`
    - Pass `NSStore` to `FSMConfig`

11. **Create `internal/handlers/namespace_test.go`**
    - Table-driven tests: Create namespace, List, Delete
    - Test: delete "default" → 400
    - Test: create with invalid name → 400

## Todo List
- [ ] Create `internal/middleware/namespace.go`
- [ ] Create `internal/middleware/namespace_test.go`
- [ ] Modify `internal/handlers/kv.go` (all KV methods)
- [ ] Modify `internal/handlers/service.go` (all service methods)
- [ ] Modify `internal/handlers/batch.go`
- [ ] Modify `internal/handlers/kv_watch.go`
- [ ] Modify `internal/handlers/health.go`
- [ ] Modify `internal/handlers/healthcheck.go`
- [ ] Modify `internal/handlers/loadbalancer.go`
- [ ] Create `internal/handlers/namespace.go`
- [ ] Create `internal/handlers/namespace_test.go`
- [ ] Modify `cmd/konsul/main.go` (migration + middleware + routes + NSStore wiring)
- [ ] `go test ./internal/middleware/... ./internal/handlers/...` passes
- [ ] `go build ./cmd/konsul/...` succeeds

## Success Criteria
- `curl -H "X-Konsul-Namespace: team-a" /kv/foo` reads from `team-a` namespace
- `curl /kv/foo` (no header) reads from `default` namespace
- `curl -X POST /namespaces -d '{"name":"team-a"}'` creates namespace
- `curl -X DELETE /namespaces/default` returns 400
- All existing handler tests pass (no namespace header = default behavior unchanged)
- `go vet ./...` and `golangci-lint run` pass

## Conflict Prevention
- `internal/handlers/rbac.go` is Phase 05's file — do NOT modify here
- Middleware order in `main.go`: auth middleware runs before namespace middleware (namespace doesn't require auth, but auth needed for write protection)
- Phase 06 adds CLI flags that send `X-Konsul-Namespace` header — no conflict, that's client-side

## Risk Assessment
- **Medium**: KV watch (`kv_watch.go`) namespace filtering may be complex if watch events use bare keys internally. Mitigation: watch manager emits events with the full prefixed key; filter by prefix, strip before sending.
- **Low**: Load balancer handler: existing logic may use service name directly — ensure namespace-scoped lookup doesn't break existing health-weighted selection.
- **Low**: `main.go` ordering of middleware — test that namespace middleware comes after CORS but before route handlers.

## Security Considerations
- Namespace middleware runs AFTER auth middleware — auth is still enforced before namespace is resolved
- ACL policies should restrict which namespaces a token can access (Phase 06 / future work)
- Namespace header value is validated before use — prevents `ns:../../admin:key` style attacks via crafted headers (regex rejects colons, slashes, uppercase)
- Write operations to non-default namespaces should require appropriate ACL policy (future: namespace-scoped ACL tokens)

## Next Steps
- Phase 05: RBAC handler namespace extraction follows same pattern (Phase 05 owns rbac.go)
- Phase 06: CLI sends `X-Konsul-Namespace` header based on `--namespace` flag
- Future: cross-namespace service discovery via DNS `<svc>.<ns>.service.konsul`
