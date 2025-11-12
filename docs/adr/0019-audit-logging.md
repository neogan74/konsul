# ADR-0019: Audit Logging for Operational Changes

**Date**: 2025-02-14

**Status**: Proposed

**Deciders**: Konsul Maintainers

**Tags**: security, compliance, observability

## Context

Enterprise users need a tamper-evident history of **who changed what and when** across Konsul’s control plane (KV writes, service lifecycle events, ACL updates, rate-limit overrides, backup actions, etc.). Today we only emit general-purpose structured logs (`ADR-0005`), which are verbose, lack a canonical schema, and cannot guarantee full coverage when operators ingest logs downstream. The roadmap (`docs/TODO.md:147`) explicitly calls out “Audit Logging – Track all operations and changes,” and several ADRs (ACL system, rate-limit management) already assume such records exist for compliance reviews, incident response, and cross-tenant billing.

Constraints & requirements:
- Must capture both successful and denied attempts after auth/ACL evaluation and include actor identity, auth mechanism, source IP, resource, action, request metadata hash, and outcome.
- Records must be immutable, queryable, and exportable to SIEMs without blocking the request path.
- Configuration must expose sinks (JSON file, stdout, OTLP/HTTP) and retention controls, plus guard against log loss on crash (buffer + fsync window).
- Implementation should reuse existing logger primitives and middleware stack without leaking secrets.

## Decision

Introduce a first-class **audit logging subsystem** that instruments privileged surfaces via middleware and shared helpers:

1. **Audit Package**: New `internal/audit` module defines the event schema, writer interface, and pluggable sinks (file, stdout, OTLP in later phase). Each event contains structured fields (`event_id`, `timestamp`, `actor.id`, `actor.type`, `auth.method`, `source.ip`, `resource.type`, `resource.id`, `action`, `result`, `http.status`, `request.hash`, `metadata`).
2. **Config Surface**: Add `KONSUL_AUDIT_ENABLED`, `KONSUL_AUDIT_SINK` (`file|stdout|otlp`), `KONSUL_AUDIT_FILE_PATH`, `KONSUL_AUDIT_BUFFER_SIZE`, and `KONSUL_AUDIT_DROP_POLICY` env vars (documented in README). Default sink writes newline-delimited JSON to `./logs/audit.log`.
3. **Middleware Hooks**: Extend auth, ACL, KV, service, rate-limit, backup handlers, and `konsulctl` admin commands to call `audit.Logger.Record(ctx, event)` after authorization decisions. Observability-critical paths (KV watch, template engine) also emit events when mutating state.
4. **Delivery Guarantees**: Writers batch events asynchronously with bounded queues and fsync interval (e.g., 1s, configurable). When the buffer is full we either block or drop with metrics, based on `KONSUL_AUDIT_DROP_POLICY`.
5. **Tooling & Docs**: Provide `konsulctl audit tail` and `audit export` commands plus a “Security & Audit Logging” section in `README.md`.

This approach keeps audit data separated from regular logs, enforces a consistent schema, and enables future sinks without rewriting business logic.

## Alternatives Considered

### Alternative 1: Reuse Existing Structured Logs
- **Pros**:
  - Zero new infrastructure.
  - Already integrated everywhere.
- **Cons**:
  - No guarantee every mutation emits a log or carries required fields.
  - Operators must sift through high-volume logs; tamper detection impossible.
- **Reason for rejection**: Fails compliance and completeness requirements; no dedicated schema or retention controls.

### Alternative 2: External SIEM Only (send events via OTLP/HTTP, no local sink)
- **Pros**:
  - Offloads storage/retention to enterprise tooling.
  - Advanced search and correlation for free.
- **Cons**:
  - Requires network connectivity and credentials; not viable for air-gapped installs.
  - Hard to test locally; cannot satisfy “track all operations” if exporter misconfigured.
- **Reason for rejection**: Needs a reliable local audit trail even when SIEM is unavailable.

### Alternative 3: Database/Persistence Triggers
- **Pros**:
  - Guaranteed capture for data mutations (KV, services).
  - Minimal app-layer changes.
- **Cons**:
  - Misses rejected operations (auth failures, ACL denies).
  - Konsul supports multiple storage engines; would duplicate logic per backend.
- **Reason for rejection**: Incomplete coverage and higher maintenance burden.

## Consequences

### Positive
- Provides compliance-grade visibility for every privileged action.
- Simplifies incident response and tenant auditing with a consistent JSON schema.
- Enables future automation (alerting on suspicious patterns) because data is normalized.

### Negative
- Additional write path adds latency/CPU (mitigated via async buffers).
- Requires storage/retention management and new configuration knobs.
- Developers must maintain audit coverage as new endpoints are added.

### Neutral
- Introduces new operational metrics (buffer depth, drops) that platform teams must monitor.
- Encourages stricter access modeling since every action is recorded.

## Implementation Notes

Phase plan:
1. **Foundations**: Implement `internal/audit` package, JSON file sink, env configuration, and unit tests.
2. **HTTP/CLI Instrumentation**: Wrap Fiber handlers and `konsulctl` admin commands with audit decorators; add integration tests ensuring events fire on KV/service mutations and ACL denials.
3. **Observability**: Emit Prometheus metrics (`konsul_audit_events_total`, `konsul_audit_dropped_events_total`, `konsul_audit_writer_latency_seconds`) and log warnings when buffers overflow.
4. **Extended Sinks**: Add OTLP exporter and optional local rotation/compression.
5. **Docs & Tooling**: Update README, add `docs/security-audit-logging.md`, and ship CLI helpers for tailing/exporting logs.

Risks: accidental leakage of sensitive payloads (mitigated by hashing request bodies), performance regressions if operators force synchronous fsync, and ensuring audit writes cannot be bypassed (enforce via lint/tests).

## References

- `docs/TODO.md:147` – Enterprise requirement for audit logging.
- `docs/adr/0005-structured-logging.md` – Existing logging strategy.
- `docs/adr/0010-acl-system.md` & `docs/adr/0014-rate-limiting-management-api.md` – Features assuming audit records.
- `docs/rate-limiting-api.md:700-740` – Example security guidance referencing audit logs.

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-02-14 | Codex Agent | Initial version |
