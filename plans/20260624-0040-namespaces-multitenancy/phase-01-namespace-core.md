# Phase 01: Namespace Core Types + Store + Persistence

## Context Links

- Parent plan: [plan.md](plan.md)
- Research: [researcher-01-namespace-patterns.md](research/researcher-01-namespace-patterns.md)
- Research: [researcher-02-go-technical-patterns.md](research/researcher-02-go-technical-patterns.md)
- Scout: [scout-01-codebase-touchpoints.md](scout/scout-01-codebase-touchpoints.md)

## Parallelization Info

- **Wave:** 1 (first — no deps)
- **Can run in parallel with:** nothing (foundational)
- **Must complete before:** Phase 02, 03, 04, 05, 06
- **Blocks:** everything

## Overview

- **Priority:** Critical
- **Status:** pending
- **Description:** Create `internal/namespace/` package with `Namespace` type, `NamespaceStore` interface, in-memory + BadgerDB implementations, CRUD logic, and `NamespaceConfig` in config. This is the foundation all other phases depend on.

## Key Insights

- Consul and K8s both treat namespace as a first-class resource with its own CRUD API
- Namespace `"default"` must always exist; cannot be deleted
- Validation: DNS-compatible names `^[a-z0-9][a-z0-9-]{0,62}$`; reserved names: `default`, `system`
- BadgerDB key for namespace metadata: `_namespace:<name>` (avoids collision with `ns:` prefix used for KV data)
- Config should allow disabling multi-namespace mode (single-tenant deployments)

## Requirements

**Functional:**

- `Namespace` struct with Name, Description, CreatedAt, UpdatedAt
- `NamespaceStore` interface: Create, Get, List, Delete, Exists
- In-memory implementation (default, test-friendly)
- BadgerDB implementation with `_namespace:<name>` keys
- `"default"` namespace auto-created on startup
- Validation: regex + reserved names check
- `NamespaceConfig` added to `internal/config/config.go`

**Non-functional:**

- Thread-safe in-memory store (sync.RWMutex)
- All errors wrapped with `fmt.Errorf("%w")`
- Full test coverage for all interface methods

## Architecture

```text
internal/namespace/
  types.go          — Namespace struct, sentinel errors
  store.go          — NamespaceStore interface
  memory_store.go   — in-memory implementation
  badger_store.go   — BadgerDB implementation
  validate.go       — name validation logic
  namespace_test.go — unit tests for both impls
```

Config addition in `internal/config/config.go`:

```go
type NamespaceConfig struct {
    Enabled          bool   // env: KONSUL_NAMESPACE_ENABLED, default: true
    DefaultNamespace string // env: KONSUL_NAMESPACE_DEFAULT, default: "default"
    AllowImplicit    bool   // allow requests without namespace header, default: true
}
```

Namespace type:

```go
type Namespace struct {
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

Sentinel errors:

```go
var (
    ErrNamespaceNotFound = errors.New("namespace not found")
    ErrNamespaceExists   = errors.New("namespace already exists")
    ErrNamespaceReserved = errors.New("namespace name is reserved")
    ErrInvalidNamespace  = errors.New("invalid namespace name")
)
```

## File Ownership (exclusive to this phase)

- CREATE `internal/namespace/types.go`
- CREATE `internal/namespace/store.go`
- CREATE `internal/namespace/memory_store.go`
- CREATE `internal/namespace/badger_store.go`
- CREATE `internal/namespace/validate.go`
- CREATE `internal/namespace/namespace_test.go`
- MODIFY `internal/config/config.go` — add `Namespace NamespaceConfig` field + env loading

## Implementation Steps

1. **Create `internal/namespace/types.go`**
   - Define `Namespace` struct with JSON tags
   - Define sentinel errors: `ErrNamespaceNotFound`, `ErrNamespaceExists`, `ErrNamespaceReserved`, `ErrInvalidNamespace`

2. **Create `internal/namespace/store.go`**
   - Define `NamespaceStore` interface:

     ```go
     type NamespaceStore interface {
         Create(ns *Namespace) error
         Get(name string) (*Namespace, error)
         List() ([]*Namespace, error)
         Delete(name string) error
         Exists(name string) (bool, error)
     }
     ```

3. **Create `internal/namespace/validate.go`**
   - `ValidateName(name string) error` — regex `^[a-z0-9][a-z0-9-]{0,62}$`
   - `IsReserved(name string) bool` — check against `{"default", "system"}`
   - Note: `"default"` IS a valid name (not rejected by reserved check for read ops), only blocked for Delete

4. **Create `internal/namespace/memory_store.go`**
   - `MemoryNamespaceStore` struct with `mu sync.RWMutex`, `data map[string]*Namespace`
   - Implement all 5 interface methods
   - `Create`: validate name, check exists, store copy
   - `Get`: return copy (not pointer to internal)
   - `Delete`: reject if `name == "default"`

5. **Create `internal/namespace/badger_store.go`**
   - `BadgerNamespaceStore` wraps `*badger.DB`
   - Key format: `_namespace:<name>` (byte prefix)
   - Serialize/deserialize with `encoding/json`
   - `List`: scan prefix `_namespace:`, decode each
   - Same delete guard for `"default"`

6. **Modify `internal/config/config.go`**
   - Add `Namespace NamespaceConfig` to `Config` struct
   - Add env loading in `Load()`:
     - `KONSUL_NAMESPACE_ENABLED` (bool, default true)
     - `KONSUL_NAMESPACE_DEFAULT` (string, default "default")
     - `KONSUL_NAMESPACE_ALLOW_IMPLICIT` (bool, default true)

7. **Create `internal/namespace/namespace_test.go`**
   - Table-driven tests for `MemoryNamespaceStore`: Create, Get, List, Delete, Exists
   - Test: duplicate create returns `ErrNamespaceExists`
   - Test: delete "default" returns error
   - Test: `ValidateName` edge cases (empty, too long, uppercase, leading dash)
   - Test `BadgerNamespaceStore` with `badger.Open(badger.DefaultOptions("").WithInMemory(true))`

## Todo List

- [ ] Create `internal/namespace/types.go`
- [ ] Create `internal/namespace/store.go` (interface)
- [ ] Create `internal/namespace/validate.go`
- [ ] Create `internal/namespace/memory_store.go`
- [ ] Create `internal/namespace/badger_store.go`
- [ ] Modify `internal/config/config.go` — NamespaceConfig
- [ ] Create `internal/namespace/namespace_test.go`
- [ ] `go test ./internal/namespace/...` passes
- [ ] `golangci-lint run ./internal/namespace/...` passes

## Success Criteria

- All 5 `NamespaceStore` methods implemented and tested for both backends
- `"default"` namespace auto-exists / cannot be deleted
- Validation rejects invalid names (wrong chars, too long, empty)
- Config loads from env vars with correct defaults
- Zero compilation errors in module

## Conflict Prevention

- Only touches `internal/namespace/` (new dir) and `internal/config/config.go`
- No other phase touches `internal/config/config.go`
- All other phases import `internal/namespace` — no circular deps (namespace has no imports from handlers/raft/store)

## Risk Assessment

- **Low**: New package, no existing callers to break
- **Medium**: BadgerDB key collision if `_namespace:` prefix overlaps — mitigated by underscore prefix (KV user keys never start with `_`)
- **Low**: Config env var naming conflicts — verified no existing `KONSUL_NAMESPACE_*` vars

## Security Considerations

- Namespace name validation at creation prevents injection attacks in key prefixes
- Reserved names `system`/`default` cannot be created by users (guard in Create)
- `default` cannot be deleted (hard guard)

## Next Steps

- Phase 02: Store layer uses `internal/namespace` types for validation
- Phase 03: Raft FSM imports `namespace.DefaultNamespace` constant
- Phase 04: HTTP handler for `/namespaces` CRUD uses `NamespaceStore`
- Phase 05: RBAC types reference namespace names (string, validated elsewhere)
