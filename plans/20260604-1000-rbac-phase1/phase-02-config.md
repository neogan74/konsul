# Phase 02 — Config RBACConfig

**Links:** [plan.md](plan.md)
**Wave:** 1 (parallel with Phase 01)

## Parallelization Info

- Runs concurrently with Phase 01.
- No dependencies on any other phase.
- Phase 03 and Phase 05 depend on `RBACConfig` existing in config struct.
- **Scope is minimal**: only additive changes to `internal/config/config.go`.

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-06-04 |
| Priority | P0 — required before manager + wiring |
| Status | pending |

Adds `RBACConfig` struct and wires it into the top-level `Config` struct with env var parsing.

## Key Insights

- Existing pattern: each subsystem has a dedicated `XConfig` struct, loaded via `getEnvBool/getEnvDuration/getEnvString` helpers. All env vars use `KONSUL_<SUBSYSTEM>_<FIELD>` format.
- `Config.Validate()` enforces cross-subsystem rules — RBAC requires Auth enabled (same constraint as ACL).
- No JSON struct tags used in config — purely env var driven.

## Requirements

```go
type RBACConfig struct {
    Enabled                 bool
    CacheTTL                time.Duration // default 5m
    ExpirationCheckInterval time.Duration // default 1m
}
```

Env vars:
- `KONSUL_RBAC_ENABLED` → `RBACConfig.Enabled` (default: `false`)
- `KONSUL_RBAC_CACHE_TTL` → `RBACConfig.CacheTTL` (default: `5m`)
- `KONSUL_RBAC_EXPIRATION_CHECK_INTERVAL` → `RBACConfig.ExpirationCheckInterval` (default: `1m`)

Validation rule: if `RBAC.Enabled && !Auth.Enabled` → return error `"RBAC requires Auth to be enabled"`.

## Architecture

Single struct addition + two code locations in `config.go`:

1. New struct `RBACConfig` (add after `ACLConfig`, ~line 111).
2. Add `RBAC RBACConfig` field to `Config` struct (add after `ACL ACLConfig`, ~line 21).
3. Add loading block in `Load()` function.
4. Add validation rule in `Validate()` function.

## File Ownership

Exclusive files owned by this phase:
- `internal/config/config.go` (MODIFIED — additive only)

## Implementation Steps

1. Open `internal/config/config.go`.
2. After `ACLConfig` struct definition, add `RBACConfig` struct:
   ```go
   // RBACConfig contains Role-Based Access Control configuration
   type RBACConfig struct {
       Enabled                 bool
       CacheTTL                time.Duration
       ExpirationCheckInterval time.Duration
   }
   ```
3. In `Config` struct, add `RBAC RBACConfig` field after `ACL ACLConfig`.
4. In `Load()` function, find the ACL loading block and add the RBAC block immediately after:
   ```go
   RBAC: RBACConfig{
       Enabled:                 getEnvBool("KONSUL_RBAC_ENABLED", false),
       CacheTTL:                getEnvDuration("KONSUL_RBAC_CACHE_TTL", 5*time.Minute),
       ExpirationCheckInterval: getEnvDuration("KONSUL_RBAC_EXPIRATION_CHECK_INTERVAL", time.Minute),
   },
   ```
5. In `Validate()` function, add after ACL validation:
   ```go
   if c.RBAC.Enabled && !c.Auth.Enabled {
       return fmt.Errorf("RBAC requires Auth to be enabled")
   }
   ```
6. Run `go build ./internal/config/...` to verify.
7. Run `go test ./internal/config/...` to verify existing tests still pass.

## Todo

- [ ] Add `RBACConfig` struct definition
- [ ] Add `RBAC RBACConfig` to `Config` struct
- [ ] Add RBAC loading block in `Load()`
- [ ] Add validation rule in `Validate()`
- [ ] `go build ./internal/config/...` passes
- [ ] `go test ./internal/config/...` passes (existing tests unaffected)

## Success Criteria

- `cfg.RBAC.Enabled`, `cfg.RBAC.CacheTTL`, `cfg.RBAC.ExpirationCheckInterval` accessible from `main.go`.
- Validation rejects `RBAC.Enabled=true` + `Auth.Enabled=false` combination.
- All existing config tests pass without modification.
- `config_test.go` does not need updating (no existing tests reference RBAC).

## Conflict Prevention

- Only modifies `internal/config/config.go` — no other phase touches this file.
- Changes are purely additive (new struct, new field, new loading block, new validation rule).
- Do not modify any existing struct fields or loading logic — risk of breaking existing tests.
- Do not import `internal/rbac` from config — that creates a circular dependency.

## Risk Assessment

- **Very Low**: Additive-only change to a well-structured file.
- **Watch**: Ensure `Validate()` function edit is in the correct location (after ACL validation, before return nil).

## Security Considerations

- Default `Enabled: false` ensures zero impact on existing deployments.
- `CacheTTL` lower bound should be enforced by `Validate()` if desired (e.g., minimum 30s), though not required for Phase 1.

## Next Steps

After Phase 02 completes: `cfg.RBAC` is available for Phase 03 (manager constructor) and Phase 05 (main.go wiring).
