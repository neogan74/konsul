# Phase 03: Raft Commands + FSM Namespace Support

## Context Links
- Parent plan: [plan.md](plan.md)
- Depends on: [phase-01-namespace-core.md](phase-01-namespace-core.md)
- Research: [researcher-02-go-technical-patterns.md](research/researcher-02-go-technical-patterns.md)
- Scout: [scout-01-codebase-touchpoints.md](scout/scout-01-codebase-touchpoints.md)
- Key files: `internal/raft/commands.go` (212 lines), `internal/raft/fsm.go` (300+ lines), `internal/raft/store_interfaces.go`

## Parallelization Info
- **Wave:** 2 (parallel with Phase 02 and Phase 05)
- **Can run in parallel with:** Phase 02, Phase 05
- **Must NOT touch:** `internal/store/` (Phase 02), `internal/rbac/` (Phase 05), `internal/handlers/` (Phase 04)
- **Depends on:** Phase 01 complete

## Overview
- **Priority:** Critical
- **Status:** pending
- **Description:** Extend all Raft command payload structs with `Namespace string \`json:"namespace,omitempty"\`` field. Update FSM Apply methods to normalize empty namespace to `"default"` and pass it to store interface methods. Extend `KVStoreInterface` and `ServiceStoreInterface` with namespace-aware `*Local` method signatures.

## Key Insights
- `json:"namespace,omitempty"` means old log entries (no namespace field) deserialize with `Namespace = ""` — normalized to `"default"` in FSM. **Zero breaking change for existing Raft logs.**
- FSM currently calls e.g. `f.kvStore.SetLocal(key, value)`. After this phase, calls become `f.kvStore.SetLocalNS(ns, key, value)` OR the store interface accepts namespace-prefixed keys directly.
- **Chosen approach**: extend `KVStoreInterface` and `ServiceStoreInterface` with `NS` variants. The concrete `NamespacedKV` wrapper (Phase 02) implements them. This avoids FSM knowing about prefix format.
- Alternative simpler approach: FSM constructs the prefixed key itself (`ns:<ns>:<key>`) and calls existing `SetLocal`. **Chosen for simplicity (KISS)** — FSM knows the prefix format but doesn't need Phase 02 wrappers.
- `normalizeNS("")` → `"default"` applied at FSM Apply boundary, once, centrally.

## Requirements

**Functional:**
- Add `Namespace string \`json:"namespace,omitempty"\`` to ALL 14 KV+Service payload structs
- Add `normalizeNS(ns string) string` helper in `fsm.go`
- All FSM `applyKV*` and `applyService*` methods extract and normalize namespace
- FSM passes namespace-prefixed key/name to store: `"ns:" + ns + ":" + key`
- Existing `KVStoreInterface` and `ServiceStoreInterface` signatures unchanged (FSM builds the prefixed key itself)
- New `CmdNamespaceCreate`, `CmdNamespaceDelete` command types for namespace CRUD replication

**Non-functional:**
- Raft log replay of old entries (no namespace) must work correctly
- All FSM apply tests updated to pass namespace
- `commands.go` changes are purely additive (new fields, new command types)

## Architecture

### Payload Changes (commands.go)
Add `Namespace string \`json:"namespace,omitempty"\`` to:
```
KVSetPayload, KVSetWithFlagsPayload, KVSetCASPayload
KVDeletePayload, KVDeleteCASPayload
KVBatchSetPayload, KVBatchSetCASPayload
KVBatchDeletePayload, KVBatchDeleteCASPayload
ServiceRegisterPayload, ServiceRegisterCASPayload
ServiceDeregisterPayload, ServiceDeregisterCASPayload
ServiceHeartbeatPayload, HealthTTLUpdatePayload
```

New command types:
```go
CmdNamespaceCreate  // payload: NamespaceCreatePayload{Name, Description string}
CmdNamespaceDelete  // payload: NamespaceDeletePayload{Name string}
```

New payload structs:
```go
type NamespaceCreatePayload struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}
type NamespaceDeletePayload struct {
    Name string `json:"name"`
}
```

### FSM Changes (fsm.go)
```go
// Normalizer — called at every Apply method boundary
func normalizeNS(ns string) string {
    if ns == "" {
        return "default"
    }
    return ns
}

// nsKey builds the namespaced key for store operations
func nsKey(ns, key string) string {
    return "ns:" + ns + ":" + key
}

// nsName builds the namespaced service name
func nsName(ns, name string) string {
    return "ns:" + ns + ":" + name
}

// applyKVSet example
func (f *KonsulFSM) applyKVSet(payload []byte) error {
    var p KVSetPayload
    if err := json.Unmarshal(payload, &p); err != nil {
        return err
    }
    ns := normalizeNS(p.Namespace)
    f.kvStore.SetLocal(nsKey(ns, p.Key), p.Value)
    return nil
}
```

### FSM KonsulFSM struct addition
```go
type KonsulFSM struct {
    // existing fields...
    nsStore namespace.NamespaceStore // for applyNamespaceCreate/Delete
}
```

### Switch case addition in Apply()
```go
case CmdNamespaceCreate:
    return f.applyNamespaceCreate(cmd.Payload)
case CmdNamespaceDelete:
    return f.applyNamespaceDelete(cmd.Payload)
```

### FSMConfig addition
```go
type FSMConfig struct {
    KVStore      KVStoreInterface
    ServiceStore ServiceStoreInterface
    NSStore      namespace.NamespaceStore // NEW
    OnApply      func(...)
}
```

## File Ownership (exclusive to this phase)
- MODIFY `internal/raft/commands.go` — add `Namespace` field to all payload structs; new Cmd types + payload structs
- MODIFY `internal/raft/fsm.go` — add `normalizeNS`, `nsKey`, `nsName` helpers; update all `applyKV*`/`applyService*`; add `nsStore` to FSM; add namespace apply methods
- MODIFY `internal/raft/store_interfaces.go` — NO CHANGES (FSM builds prefixed keys, existing `SetLocal(key, value)` used as-is)

**Do NOT touch:** `internal/store/` (Phase 02), `internal/handlers/` (Phase 04)

## Implementation Steps

1. **Modify `internal/raft/commands.go`**
   - Add `Namespace string \`json:"namespace,omitempty"\`` to all 14 payload structs listed above
   - Add `CmdNamespaceCreate`, `CmdNamespaceDelete` to `CommandType` const block
   - Update `String()` switch for new types
   - Add `NamespaceCreatePayload`, `NamespaceDeletePayload` structs

2. **Modify `internal/raft/fsm.go`**
   - Add `nsStore namespace.NamespaceStore` field to `KonsulFSM`
   - Add `NSStore namespace.NamespaceStore` to `FSMConfig`
   - Wire `nsStore` in `NewFSM`
   - Add `normalizeNS(ns string) string` function
   - Add `nsKey(ns, key string) string` and `nsName(ns, name string) string` helpers
   - Update all `applyKVSet`, `applyKVSetWithFlags`, `applyKVSetCAS`, `applyKVDelete`, `applyKVDeleteCAS` to extract `p.Namespace`, normalize, then use `nsKey(ns, p.Key)` when calling `f.kvStore.SetLocal(...)`
   - Update batch apply methods: for `KVBatchSetPayload.Items map[string]string`, rebuild map with namespace-prefixed keys
   - Update `applyServiceRegister`, `applyServiceRegisterCAS`, `applyServiceDeregister`, `applyServiceDeregisterCAS`, `applyServiceHeartbeat`, `applyHealthTTLUpdate`
   - For service payloads: pass `nsName(ns, p.Service.Name)` as the service name; also set `p.Service.Namespace = ns` in the snapshot
   - Add `applyNamespaceCreate(payload []byte) error` and `applyNamespaceDelete(payload []byte) error`
   - Add `CmdNamespaceCreate`, `CmdNamespaceDelete` cases to `Apply()` switch

3. **Update FSM tests** (`internal/raft/fsm_test.go`)
   - Existing tests should still pass (no namespace = "default" normalized)
   - Add test cases with explicit namespace: verify `applyKVSet` with `Namespace: "team-a"` stores under `ns:team-a:<key>`
   - Add test for `applyNamespaceCreate` / `applyNamespaceDelete`

4. **Update integration tests** (files in `internal/raft/`)
   - Check that namespace field is preserved through leader→follower replication
   - Test old-format command replay (empty namespace → "default")

## Todo List
- [ ] Add `Namespace` field to all 14 payload structs in `commands.go`
- [ ] Add `CmdNamespaceCreate`, `CmdNamespaceDelete` command types
- [ ] Add `NamespaceCreatePayload`, `NamespaceDeletePayload` structs
- [ ] Update `String()` switch for new command types
- [ ] Add `nsStore` to `KonsulFSM` and `FSMConfig`
- [ ] Implement `normalizeNS`, `nsKey`, `nsName` helpers in `fsm.go`
- [ ] Update all `applyKV*` methods (9 methods)
- [ ] Update all `applyService*` methods (5 methods)
- [ ] Add `applyNamespaceCreate`, `applyNamespaceDelete`
- [ ] Update `Apply()` switch for new command types
- [ ] Update `fsm_test.go` with namespace test cases
- [ ] Verify `go test ./internal/raft/...` passes

## Success Criteria
- Old Raft log entry with no namespace field replays correctly as "default"
- `KVSetPayload{Namespace: "team-a", Key: "foo", Value: "bar"}` causes store to write `ns:team-a:foo`
- Namespace create/delete commands are applied to `nsStore`
- All existing Raft integration tests pass
- `golangci-lint` passes on modified files

## Conflict Prevention
- `store_interfaces.go` NOT modified — prevents conflict with Phase 02
- Only `commands.go` and `fsm.go` are modified — no handler file overlap with Phase 04
- Phase 02 `NamespacedKV` wrappers are NOT used by FSM (FSM builds prefixed keys directly) — clear separation of concerns, no import cycle

## Risk Assessment
- **Medium**: Batch payload structs use `map[string]string` for Items — need to rebuild the map with ns-prefixed keys in FSM. Don't modify the payload in-place; build new map.
- **Low**: `CmdNamespaceCreate/Delete` command types added at end of const block — existing iota values unchanged
- **Low**: fsm_test.go mock stores (`mockKVStore`) must implement `SetLocal(key, value string)` with same signature — no change needed since FSM now pre-pends prefix before calling.

## Security Considerations
- FSM trusts the namespace value from Raft log (validated at HTTP boundary in Phase 04 before being written to Raft)
- `applyNamespaceDelete` must refuse to delete `"default"` namespace (call `nsStore.Delete` which has the guard from Phase 01)
- No auth check in FSM — auth is HTTP-layer responsibility

## Next Steps
- Phase 04: HTTP handlers call `raftNode.Apply(NewCommand(CmdKVSet, KVSetPayload{Namespace: ns, Key: k, Value: v}))` using the resolved namespace from middleware
- Phase 04: Add namespace CRUD HTTP handler that uses `CmdNamespaceCreate/Delete` commands when Raft enabled
