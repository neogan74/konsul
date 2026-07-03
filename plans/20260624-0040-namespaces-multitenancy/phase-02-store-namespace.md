# Phase 02: Store Layer — KV Prefix Migration + NamespacedKV + Service Scoping

## Context Links
- Parent plan: [plan.md](plan.md)
- Depends on: [phase-01-namespace-core.md](phase-01-namespace-core.md)
- Research: [researcher-02-go-technical-patterns.md](research/researcher-02-go-technical-patterns.md)
- Scout: [scout-01-codebase-touchpoints.md](scout/scout-01-codebase-touchpoints.md)
- Key file: `internal/store/kv.go` (1,171 lines), `internal/store/service.go` (~800 lines)

## Parallelization Info
- **Wave:** 2 (parallel with Phase 03 and Phase 05)
- **Can run in parallel with:** Phase 03, Phase 05
- **Must NOT touch:** `internal/raft/` (Phase 03), `internal/rbac/` (Phase 05), `internal/handlers/` (Phase 04)
- **Depends on:** Phase 01 complete (namespace types available)

## Overview
- **Priority:** Critical
- **Status:** pending
- **Description:** Add `NamespacedKV` and `NamespacedService` wrapper types that decorate existing stores with namespace-prefixed key access. Add BadgerDB startup migration. Existing store method signatures remain unchanged — wrappers are the only new surface.

## Key Insights
- **Wrapper pattern (Option C from research)**: No interface signature changes. Handlers construct `NamespacedKV{store, ns}` per-request and call existing methods. Store is unaware of namespaces.
- **Key prefix format**: `ns:<namespace>:<original-key>` stored in BadgerDB. In-memory `KVStore.Data` map also uses this format post-migration.
- **Migration strategy**: One-time startup migration. Reads all keys from BadgerDB without `ns:` prefix, rewrites as `ns:default:<key>`. Sets flag `_migrated:v1` when done. Idempotent.
- **Service scoping**: Add `Namespace string` field to `store.Service` struct. Service store's in-memory `Data` map key becomes `ns:<ns>:<name>`. The `NamespacedService` wrapper prefixes the name before delegating.
- **Snapshot impact**: `GetAllData()` and `RestoreFromSnapshot()` see namespace-prefixed keys as-is — Raft FSM just stores/restores the map, namespace info is embedded in keys.

## Requirements

**Functional:**
- `NamespacedKV` wrapper: `Get/Set/Delete/List/SetCAS/SetWithFlags` all prepend `ns:<ns>:` to key before calling underlying store
- `NamespacedService` wrapper: same approach for `Register/Deregister/Get/List/Heartbeat`
- `store.Service` struct: add `Namespace string \`json:"namespace,omitempty"\``
- `store.ServiceDataSnapshot` struct: add `Namespace string`
- BadgerDB migration function `MigrateToNamespacedKeys(db *badger.DB) error`
- Migration called once at startup (before store construction), skipped if `_migrated:v1` key exists

**Non-functional:**
- Existing callers of `*KVStore` and `*ServiceStore` must compile unchanged
- All new wrapper methods covered by unit tests
- Migration is transactional (use BadgerDB `WriteBatch` or multiple `Txn`s with rollback-on-error)

## Architecture

```
internal/store/
  kv.go              — ADD: nothing (no signature changes)
  kv_namespaced.go   — NEW: NamespacedKV wrapper
  service.go         — MODIFY: add Namespace field to Service, ServiceDataSnapshot
  service_namespaced.go — NEW: NamespacedService wrapper
  snapshot.go        — no change (keys already namespaced in map)

internal/persistence/
  migration.go       — NEW: MigrateToNamespacedKeys(), IsMigrated()
  migration_test.go  — NEW: tests
```

### NamespacedKV Design
```go
// internal/store/kv_namespaced.go
type NamespacedKV struct {
    store     *KVStore
    namespace string
}

func NewNamespacedKV(store *KVStore, namespace string) *NamespacedKV {
    return &NamespacedKV{store: store, namespace: namespace}
}

func (n *NamespacedKV) nsKey(key string) string {
    return "ns:" + n.namespace + ":" + key
}

func (n *NamespacedKV) Get(key string) (string, bool) {
    return n.store.Get(n.nsKey(key))
}

func (n *NamespacedKV) Set(key, value string) {
    n.store.Set(n.nsKey(key), value)
}

func (n *NamespacedKV) List(prefix string) map[string]KVEntry {
    nsPrefix := "ns:" + n.namespace + ":" + prefix
    raw := n.store.List(nsPrefix)
    // Strip ns:<ns>: prefix from returned keys
    result := make(map[string]KVEntry, len(raw))
    strip := "ns:" + n.namespace + ":"
    for k, v := range raw {
        result[strings.TrimPrefix(k, strip)] = v
    }
    return result
}
// ... SetWithFlags, SetCAS, Delete, DeleteCAS, BatchSet, BatchSetCAS, BatchDelete, BatchDeleteCAS
// ... SetLocal, DeleteLocal, SetCASLocal variants for FSM calls
```

### NamespacedService Design
```go
// internal/store/service_namespaced.go
type NamespacedService struct {
    store     *ServiceStore
    namespace string
}

func (n *NamespacedService) nsName(name string) string {
    return "ns:" + n.namespace + ":" + name
}

// Register: set service.Name = nsName(service.Name), then delegate
// Deregister: nsName(name)
// Get: nsName(name), then strip prefix from returned Service.Name
// List: filter map by "ns:<ns>:" prefix, strip prefix from names
```

### Migration Design
```go
// internal/persistence/migration.go
const MigrationFlagKey = "_migrated:v1"

func IsMigrated(db *badger.DB) (bool, error) { ... }

func MigrateToNamespacedKeys(db *badger.DB) error {
    // 1. Check _migrated:v1 flag — return nil if set
    // 2. Collect all keys NOT starting with "ns:" or "_"
    // 3. For each key: write "ns:default:<key>" = value
    // 4. Delete old key
    // 5. Write _migrated:v1 = "1"
    // Use badger.DB.Update for each batch (groups of 100)
}
```

## File Ownership (exclusive to this phase)
- CREATE `internal/store/kv_namespaced.go`
- CREATE `internal/store/kv_namespaced_test.go`
- CREATE `internal/store/service_namespaced.go`
- CREATE `internal/store/service_namespaced_test.go`
- MODIFY `internal/store/service.go` — add `Namespace string` to `Service` and `ServiceDataSnapshot`
- CREATE `internal/persistence/migration.go`
- CREATE `internal/persistence/migration_test.go`

**Do NOT touch:** `internal/store/kv.go` (method signatures stay same), `internal/store/snapshot.go`

## Implementation Steps

1. **Modify `internal/store/service.go`**
   - Add `Namespace string \`json:"namespace,omitempty"\`` to `Service` struct
   - Add `Namespace string` to `ServiceDataSnapshot` struct
   - Existing callers compile fine (new field is additive)

2. **Create `internal/store/kv_namespaced.go`**
   - `NamespacedKV` struct and constructor
   - `nsKey(key string) string` helper
   - Implement: `Get`, `Set`, `SetWithFlags`, `SetCAS`, `Delete`, `DeleteCAS`, `List`, `ListEntries`
   - Implement batch variants: `BatchGet`, `BatchSet`, `BatchSetCAS`, `BatchDelete`, `BatchDeleteCAS`
   - Implement `*Local` variants (used by FSM): `SetLocal`, `DeleteLocal`, `SetCASLocal`, `DeleteCASLocal`, `BatchSetLocal`, `BatchSetCASLocal`, `BatchDeleteLocal`, `BatchDeleteCASLocal`
   - List methods: strip `ns:<ns>:` prefix from returned keys

3. **Create `internal/store/service_namespaced.go`**
   - `NamespacedService` struct and constructor
   - Implement: `Register`, `RegisterCAS`, `Deregister`, `DeregisterCAS`, `Get`, `GetEntry`, `List`, `ListAll`, `Heartbeat`, `UpdateTTLCheck`
   - Implement `*Local` variants for FSM
   - Name mangling: set `service.Namespace = n.namespace` on register; strip prefix on get/list
   - List: filter by `"ns:" + n.namespace + ":"` prefix

4. **Create `internal/persistence/migration.go`**
   - `IsMigrated(db *badger.DB) (bool, error)`
   - `MigrateToNamespacedKeys(db *badger.DB) error` — batch rewrite
   - Use `db.Update` in chunks of 100 keys to avoid large transactions
   - Write `_migrated:v1` flag last (atomic commit = migration is complete)

5. **Create tests**
   - `kv_namespaced_test.go`: verify `Get/Set/List` with prefix; test that two different-namespace keys don't collide; test List strips prefix
   - `service_namespaced_test.go`: verify Register/Get/List/Deregister namespace isolation
   - `migration_test.go`: use in-memory BadgerDB; populate bare keys; run migration; verify `ns:default:` prefix; verify idempotency (run twice, no error)

## Todo List
- [ ] Add `Namespace` field to `store.Service` and `store.ServiceDataSnapshot`
- [ ] Create `internal/store/kv_namespaced.go` with all KV method wrappers
- [ ] Create `internal/store/service_namespaced.go` with all service method wrappers
- [ ] Create `internal/persistence/migration.go`
- [ ] Create `internal/store/kv_namespaced_test.go`
- [ ] Create `internal/store/service_namespaced_test.go`
- [ ] Create `internal/persistence/migration_test.go`
- [ ] `go test ./internal/store/...` passes
- [ ] `go test ./internal/persistence/...` passes
- [ ] `golangci-lint run ./internal/store/... ./internal/persistence/...` passes

## Success Criteria
- `NamespacedKV.Set("key", "val")` stores under `ns:<ns>:key` in underlying store
- Two `NamespacedKV` instances with different namespaces cannot read each other's keys
- `List("")` returns keys without `ns:<ns>:` prefix
- Migration converts bare keys to `ns:default:` prefix; second run is no-op
- All existing `internal/store` tests still pass (regression)

## Conflict Prevention
- Phase 03 owns `internal/raft/store_interfaces.go` — if FSM interface needs `namespace` param, that's Phase 03's change. This phase provides wrappers that satisfy existing `KVStoreInterface` / `ServiceStoreInterface` without changes.
- Phase 04 owns `internal/handlers/` — handlers will construct `NamespacedKV` using types from this phase
- No overlap with Phase 05 (`internal/rbac/`)

## Risk Assessment
- **Medium**: Migration data loss if BadgerDB txn fails mid-batch. Mitigate: write new key BEFORE deleting old; if crash, re-run is safe (duplicate write is idempotent, old key still present).
- **Low**: NamespacedKV wrapping `*Local` methods — ensure all FSM-facing methods are wrapped or Phase 03 FSM will write bare keys. Comprehensive method list in step 2.
- **Low**: Existing tests for `kv.go` and `service.go` pass — wrappers add no changes to those files.

## Security Considerations
- Namespace name is validated by Phase 01 at HTTP boundary before reaching this layer
- Wrappers must not allow empty namespace to be passed (would create `ns::key` entries) — add guard: `if n.namespace == "" { panic("NamespacedKV: empty namespace") }` in constructor

## Next Steps
- Phase 03: FSM will call wrapper `*Local` methods after extracting namespace from Raft command payload
- Phase 04: HTTP handlers import `store.NamespacedKV` / `store.NamespacedService`, construct per-request
