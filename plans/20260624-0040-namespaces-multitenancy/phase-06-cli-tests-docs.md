# Phase 06: CLI --namespace Flag + Integration Tests + Docs

## Context Links
- Parent plan: [plan.md](plan.md)
- Depends on: ALL previous phases (01-05)
- Scout: [scout-01-codebase-touchpoints.md](scout/scout-01-codebase-touchpoints.md)
- Key files: `cmd/konsulctl/kv_commands.go`, `cmd/konsulctl/service_commands.go`, `cmd/konsulctl/client.go`, `cmd/konsulctl/cli.go`

## Parallelization Info
- **Wave:** 4 (final — depends on everything)
- **Can run in parallel with:** nothing
- **Depends on:** Phase 01-05 all complete and merged

## Overview
- **Priority:** High
- **Status:** pending
- **Description:** Add `--namespace` / `-n` global flag to konsulctl CLI. All KV and service commands send `X-Konsul-Namespace` header. Add end-to-end integration tests covering namespace isolation. Add user-facing documentation.

## Key Insights
- CLI should use a global persistent flag `--namespace` (not per-command) to reduce boilerplate — same pattern as `--token`, `--address` flags
- Client sends `X-Konsul-Namespace` header on every request when namespace != ""
- Default behavior (no flag) = no header sent = server defaults to "default" — fully backwards compatible
- Integration tests should cover the full stack: HTTP → Raft → Store → namespace isolation
- Docs: update API reference, add namespace guide, update getting-started

## Requirements

**Functional (CLI):**
- Add `--namespace` / `-n` flag to `CLI` struct in `cli.go`
- Persist namespace in `CLI.Namespace string`
- `client.go`: inject `X-Konsul-Namespace: <ns>` header when `cli.Namespace != ""`
- `kv_commands.go`: all KV subcommands (get, set, del, list, watch) pick up namespace from `cli.Namespace`
- `service_commands.go`: all service subcommands pick up namespace
- New `namespace_commands.go`: `konsulctl namespace list|create|delete` subcommands

**Functional (Integration Tests):**
- Test: KV set in `ns=team-a` not visible in `ns=team-b`
- Test: Service register in `ns=team-a` not in `ns=team-b`
- Test: Namespace CRUD (create/list/delete via HTTP)
- Test: BadgerDB startup migration (pre-migration keys accessible post-migration in default ns)
- Test: Raft replication preserves namespace across leader→follower
- Test: RBAC role in `team-a` not visible in `team-b`

**Functional (Docs):**
- `docs/namespaces.md` — namespace guide (concept, API, CLI usage)
- Update `docs/kv-watch-guide.md` — mention namespace header
- Update README.md — mention namespace support in feature list

**Non-functional:**
- CLI flag consistent with existing flag style (short `-n`)
- Integration tests use `go test -tags integration` or `_integration_test.go` naming convention (existing pattern)
- Docs ≤ 200 lines total

## Architecture

### CLI Changes

**`cmd/konsulctl/cli.go`** — add namespace flag:
```go
type CLI struct {
    Address   string
    Token     string
    Namespace string // NEW
    // ...
}

func (c *CLI) RegisterFlags(fs *flag.FlagSet) {
    // existing flags...
    fs.StringVar(&c.Namespace, "namespace", "", "Namespace to operate in (default: server default)")
    fs.StringVar(&c.Namespace, "n", "", "Namespace (shorthand)")
}
```

**`cmd/konsulctl/client.go`** — inject header:
```go
func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
    req, _ := http.NewRequest(method, c.url(path), body)
    // existing headers...
    if c.cli.Namespace != "" {
        req.Header.Set("X-Konsul-Namespace", c.cli.Namespace)
    }
    return c.http.Do(req)
}
```

**`cmd/konsulctl/kv_commands.go`** — no change needed (namespace flows through client)

**New `cmd/konsulctl/namespace_commands.go`**:
```go
type NamespaceCommands struct { cli *CLI }

func (n *NamespaceCommands) List(args []string) error { ... }
func (n *NamespaceCommands) Create(args []string) error { ... }
func (n *NamespaceCommands) Delete(args []string) error { ... }
```

### Integration Test Files

New files (following `_integration_test.go` naming convention):

**`internal/namespace/namespace_integration_test.go`**:
- Start test server (or use `setupApp()` pattern from `cmd/konsul/main_test.go`)
- Create namespace "team-a"
- KV set in team-a, verify not visible in default
- Service register in team-a, verify not visible in default

**`internal/raft/namespace_replication_integration_test.go`**:
- 3-node Raft cluster
- Create namespace + KV set on leader with `Namespace: "team-a"`
- Verify follower has same key under `ns:team-a:<key>`

**`internal/persistence/migration_integration_test.go`**:
- Create real BadgerDB with bare keys
- Run `MigrateToNamespacedKeys`
- Open KV store, verify keys accessible in "default" namespace
- Verify idempotency

### Documentation

**`docs/namespaces.md`** (new):
- What are namespaces, why use them
- Default namespace behavior
- HTTP API: header + query param
- Namespace CRUD endpoints
- CLI usage: `konsulctl --namespace team-a kv get foo`
- RBAC namespace scoping
- Migration notes for existing deployments

## File Ownership (exclusive to this phase)
- MODIFY `cmd/konsulctl/cli.go` — add Namespace field + flag
- MODIFY `cmd/konsulctl/client.go` — inject X-Konsul-Namespace header
- CREATE `cmd/konsulctl/namespace_commands.go`
- CREATE `cmd/konsulctl/namespace_commands_test.go`
- CREATE `internal/namespace/namespace_integration_test.go`
- CREATE `internal/raft/namespace_replication_integration_test.go`
- CREATE `internal/persistence/migration_integration_test.go`
- CREATE `docs/namespaces.md`
- MODIFY `docs/kv-watch-guide.md` — namespace mention
- MODIFY `README.md` — namespace feature mention

**Do NOT touch:** any files in `internal/handlers/`, `internal/store/`, `internal/raft/commands.go`, `internal/rbac/`

## Implementation Steps

1. **Modify `cmd/konsulctl/cli.go`**
   - Add `Namespace string` to `CLI` struct
   - Register `--namespace` and `-n` flags in `RegisterFlags`
   - Ensure flag is registered before subcommand parsing

2. **Modify `cmd/konsulctl/client.go`**
   - In the HTTP request builder, add `X-Konsul-Namespace` header when `cli.Namespace != ""`
   - Add helper `(c *Client) withNamespace(req *http.Request)` or inline in `do()`

3. **Create `cmd/konsulctl/namespace_commands.go`**
   - `NamespaceCommands` struct
   - `List`: `GET /namespaces` → print table
   - `Create`: `POST /namespaces` with `{"name": args[0], "description": args[1]}`
   - `Delete`: `DELETE /namespaces/<name>`
   - Register in `cmd/konsulctl/main.go` dispatch table

4. **Create `internal/namespace/namespace_integration_test.go`**
   - Use `net/http/httptest` or Fiber's `app.Test()` to test full HTTP stack
   - Setup: create real stores (memory), wire middleware + handlers
   - Test namespace isolation for KV and service operations
   - Test namespace CRUD API

5. **Create `internal/raft/namespace_replication_integration_test.go`**
   - Follow existing pattern in `data_replication_integration_test.go`
   - 3-node cluster; leader writes KV with `Namespace: "team-a"`
   - Wait for replication; verify follower's store has `ns:team-a:<key>`

6. **Create `internal/persistence/migration_integration_test.go`**
   - Open real temp-dir BadgerDB
   - Write 5 bare keys (no `ns:` prefix)
   - Call `MigrateToNamespacedKeys`
   - Verify `ns:default:<key>` exists; original key deleted
   - Call again; verify idempotency (no error, no duplication)

7. **Create `docs/namespaces.md`**
   - Concept section (2-3 sentences)
   - Quick start (5 curl examples)
   - CLI section
   - RBAC section (brief)
   - Migration section

8. **Update `docs/kv-watch-guide.md`** and `README.md`
   - Add one-liner about namespace support with link to namespaces.md

## Todo List
- [ ] Add `--namespace` / `-n` flag to `cli.go`
- [ ] Inject `X-Konsul-Namespace` header in `client.go`
- [ ] Create `namespace_commands.go` (list/create/delete)
- [ ] Register namespace commands in CLI dispatch
- [ ] Create `namespace_integration_test.go` for namespace HTTP isolation
- [ ] Create `namespace_replication_integration_test.go` for Raft namespace replication
- [ ] Create `migration_integration_test.go`
- [ ] Create `docs/namespaces.md`
- [ ] Update `docs/kv-watch-guide.md`
- [ ] Update `README.md`
- [ ] `go test ./cmd/konsulctl/...` passes
- [ ] `go test -timeout 5m ./internal/raft/...` passes (all integration tests)
- [ ] `go build ./cmd/konsulctl/...` succeeds
- [ ] `golangci-lint run ./cmd/konsulctl/...` passes

## Success Criteria
- `konsulctl --namespace team-a kv set foo bar` sends `X-Konsul-Namespace: team-a` header
- `konsulctl namespace create team-a` calls `POST /namespaces`
- `konsulctl kv get foo` (no flag) behaves identically to before this PR (backwards compat)
- All 3 integration tests pass
- Docs cover all user-facing features introduced

## Conflict Prevention
- CLI files are not touched by any earlier phase — safe to modify here
- Integration tests only call HTTP API (no internal package imports from phases 01-05) — no conflicts
- Docs-only files are exclusively owned here

## Risk Assessment
- **Low**: CLI flag ordering — `--namespace` must be parsed before subcommand runs. Using root-level `flag.FlagSet` (existing pattern) handles this.
- **Low**: Integration test setup complexity — follow exact pattern from `internal/raft/data_replication_integration_test.go` to avoid setup drift.
- **Medium**: Migration integration test needs real filesystem (`badger.Open` with temp dir). Use `t.TempDir()` which auto-cleans.

## Security Considerations
- CLI stores namespace in flag variable (not persisted to disk) — no secret storage concern
- CLI docs should note that namespace name appears in HTTP headers (visible in logs)
- Integration tests should not use production credentials

## Next Steps
- Future: `konsulctl namespace quota` — set resource quotas per namespace
- Future: JWT claims include `namespaces: ["team-a"]` — enforce at auth middleware
- Future: cross-namespace service discovery via `<svc>.svc.<ns>.konsul` DNS
- Future: namespace-scoped rate limits
