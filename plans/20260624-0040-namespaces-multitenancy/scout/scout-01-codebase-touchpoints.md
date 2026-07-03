# Scout Report: Konsul Codebase Touchpoints for Namespace/Multi-Tenancy

## CRITICAL FILES — Core Data Stores

### `internal/store/kv.go` (1,171 lines)
- `KVStore` struct: `Data: map[string]KVEntry`
- All methods need `namespace` param: `Get/Set/Delete/GetEntry/SetCAS/SetWithFlags`, `List/ListEntries/BatchGet/BatchSet/BatchDelete`, `SetLocal/DeleteLocal/SetCASLocal/DeleteCASLocal/BatchSetLocal/BatchDeleteLocal`, `GetAllData/RestoreFromSnapshot`
- Data structure: flat `map[string]KVEntry` → prefix keys with `ns:<namespace>:` OR nested `map[string]map[string]KVEntry`

### `internal/store/service.go` (~800 lines)
- `ServiceStore` struct: `Data: map[string]ServiceEntry`, `TagIndex`, `MetaIndex`
- All methods need `namespace` param: `Register/RegisterCAS/Deregister/DeregisterCAS`, `Get/GetEntry/List/ListAll/Heartbeat`, all index methods, all query methods
- Indexes must be namespaced

## CRITICAL FILES — Raft Consensus Layer

### `internal/raft/commands.go` (212 lines)
- Add `Namespace string` to ALL payload structs:
  - `KVSetPayload`, `KVSetWithFlagsPayload`, `KVSetCASPayload`
  - `KVDeletePayload`, `KVDeleteCASPayload`
  - `KVBatchSetPayload`, `KVBatchSetCASPayload`, `KVBatchDeletePayload`, `KVBatchDeleteCASPayload`
  - `ServiceRegisterPayload`, `ServiceRegisterCASPayload`
  - `ServiceDeregisterPayload`, `ServiceDeregisterCASPayload`
  - `ServiceHeartbeatPayload`, `HealthTTLUpdatePayload`
- `CommandType` enum and marshaling: no change needed

### `internal/raft/fsm.go` (300+ lines)
- `KonsulFSM.Apply(log *raft.Log)` dispatches to all apply methods
- All `applyKV*` and `applyService*` methods need to extract namespace from payload and pass to store
- Pattern: `f.kvStore.SetLocal(p.Namespace, p.Key, p.Value)` (was `f.kvStore.SetLocal(p.Key, p.Value)`)

## CRITICAL FILES — HTTP Handlers

### `internal/handlers/kv.go` (250+ lines)
- `KVHandler`: has `store *store.KVStore`, `raftNode *konsulraft.Node`
- Methods: `Get`, `Set`, `Delete`, `List` — all need namespace extraction
- Current routes: `/kv/:key`, `/kv/*`

### `internal/handlers/service.go` (250+ lines)
- `ServiceHandler`: has `store *store.ServiceStore`, `raftNode *konsulraft.Node`
- Methods: `Register`, `Deregister`, `Heartbeat`, `Get`, `List`, query methods
- Current routes: `/register`, `/deregister/:name`, `/services/:name`, `/services/query/*`

### `internal/handlers/batch.go`
- Batch KV/service operations all need namespace param

### `internal/handlers/rbac.go` (200+ lines)
- `RBACHandler`: has `manager rbac.RoleManager`
- Current routes: `/rbac/roles`, `/rbac/assignments`
- Need namespace-scoped routes

### `internal/handlers/health.go`, `healthcheck.go`
- Health status must filter by namespace (services are per-namespace)

### `internal/handlers/kv_watch.go`
- Watch endpoints need namespace scoping

### `internal/handlers/loadbalancer.go`
- Service selection must filter by namespace

## HIGH PRIORITY — RBAC System

### `internal/rbac/types.go` (67 lines)
- `Role` struct: add `Namespace string`
- `RoleAssignment` struct: add `Namespace string`
- `RoleManager` interface: all methods need `namespace string` param

### `internal/rbac/manager.go` (200+ lines)
- Cache key must include namespace: `{namespace}:{subjectID}`
- All store calls need namespace param

### `internal/rbac/store.go` (150+ lines)
- `RoleStore` / `AssignmentStore` interfaces: all methods need namespace param
- `MemoryRoleStore`: `map[string]*Role` → `map[string]map[string]*Role`
- `BadgerRoleStore`: keys become `rbac:ns:{namespace}:role:{name}`

## MEDIUM PRIORITY — Auth & Config

### `internal/auth/jwt.go` (200 lines)
- `Claims` struct: add `Namespaces []string` (which namespaces user can access)
- `GenerateTokenWithPolicies`: include namespace list

### `internal/config/config.go` (200+ lines)
- Add `NamespaceConfig` struct:
  ```go
  type NamespaceConfig struct {
      Enabled          bool   // default: true
      DefaultNamespace string // default: "default"
      AllowImplicit    bool   // allow requests without namespace header
  }
  ```

### `cmd/konsul/main.go` (650+ lines)
- Route registration: add namespace middleware
- Handler wiring: unchanged

## CLI

### `cmd/konsulctl/` (multiple command files)
- Add `--namespace` / `-n` flag to all KV and service commands
- Default: "default"
- Pass via `X-Konsul-Namespace` header

## Summary

| Category | Files | Impact |
|---|---|---|
| Core Stores | 2 | Critical |
| Raft Layer | 2 | Critical |
| HTTP Handlers | 8+ | Critical/High |
| RBAC System | 3 | High |
| Auth & Config | 2 | Medium |
| CLI | 5+ | Medium |
| **Total** | **22+** | — |

## Key Decisions Needed
1. Store data structure: nested maps vs flat with key prefix?
2. Namespace in HTTP: header `X-Konsul-Namespace` + query param `?namespace=` (recommended) vs URL path `/ns/{ns}/`?
3. BadgerDB migration: lazy dual-read vs one-time startup migration?
4. Cross-namespace queries: allowed at all? By whom?
