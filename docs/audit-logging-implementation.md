# Audit Logging Implementation Plan

## Goals
- Deliver the capabilities defined in `docs/adr/0019-audit-logging.md`.
- Capture every privileged operation (successful or denied) with consistent, queryable records.
- Provide configurable delivery (file/stdout initially) without impacting request latency.
- Expose tooling and documentation so operators can tail, export, and monitor audit streams.

## Architecture Summary
1. **Audit Core (`internal/audit`)**
   - Event schema struct with helpers for common resources (kv, service, acl, auth, backup, rate-limit, template).
   - `Writer` interface + implementations: synchronous stdout writer, buffered file writer (default), stub OTLP writer for later phases.
   - `Manager` that owns buffer, batching goroutine, metrics, and drop policy logic.
2. **Configuration Surface**
   - Env vars documented in README: `KONSUL_AUDIT_ENABLED` (default `false`), `KONSUL_AUDIT_SINK=file|stdout`, `KONSUL_AUDIT_FILE_PATH=./logs/audit.log`, `KONSUL_AUDIT_BUFFER_SIZE=1024`, `KONSUL_AUDIT_FLUSH_INTERVAL=1s`, `KONSUL_AUDIT_DROP_POLICY=block|drop`.
   - Config struct in `internal/config` and validation (paths, flush interval bounds).
3. **Instrumentation**
   - Middleware wrapper attaches request metadata (actor info, auth method, source IP, trace/span IDs) and calls `audit.Record(ctx, event)` after ACL decision.
   - Handlers emit events for KV mutations, service mutations, ACL policy CRUD, auth flows, rate-limit adjustments, backups, template executions.
   - `konsulctl` admin commands emit CLI audit events via RPC headers.
4. **Tooling & Docs**
   - `konsulctl audit tail [--since --follow]` reads local file or hits new `/admin/audit/stream` endpoint (if enabled).
   - README + new `docs/security-audit-logging.md` explaining setup, retention, SIEM integrations.

## Phase Breakdown

### Phase 1 – Foundations
1. Create `internal/audit` package with event struct, writer interface, in-memory buffer, stdout writer, file writer (uses `os.OpenFile(O_APPEND)` + buffered channel + periodic fsync).
2. Extend config (`internal/config/config.go`) and environment parsing; add defaults and validation errors.
3. Hook audit manager into application bootstrap (`cmd/konsul/main.go`) and ensure graceful shutdown flushes buffers.
4. Add Prometheus metrics (`konsul_audit_events_total`, `..._dropped_total`, `..._flush_duration_seconds`) in `internal/metrics`.
5. Unit tests for writers (buffer overflow, drop vs block) and config parsing.

### Phase 2 – HTTP/CLI Instrumentation
1. Build Fiber middleware `middleware/audit.go` that inspects request context (auth claims, API key metadata, ACL evaluation result) and produces a base event.
2. Wrap critical handlers:
   - `internal/handlers/kv.go`: set/delete/batch operations.
   - `internal/handlers/service.go`: register/deregister/update, heartbeat.
   - `internal/handlers/acl.go`, `auth.go`, `backup.go`, `ratelimit.go`.
   - `internal/graphql` mutations via resolver helper.
3. Ensure denied requests (missing auth/ACL reject) also emit events with `result=denied`.
4. Extend `konsulctl` admin commands to include actor metadata headers and print audit IDs returned by server (optional).
5. Add integration tests (using in-memory writer) validating event contents for representative routes.

### Phase 3 – Tooling & Operator Experience
1. Implement `konsulctl audit tail` command (file-based tail with `--json` output) and `konsulctl audit export --since --until`.
2. Introduce optional HTTP endpoints:
   - `GET /admin/audit/events` (paginated).
   - `GET /admin/audit/stream` (SSE) for real-time tailing.
   Both endpoints require admin ACL scope and reuse audit middleware.
3. Document deployment patterns, disk rotation suggestions (logrotate example), and SIEM forwarding tips in `docs/security-audit-logging.md`.
4. Update `README.md` and `docs/TODO.md` to mark audit logging items complete once delivered.

### Phase 4 – Extended Sinks & Hardening
1. Add OTLP/HTTP exporter (config: endpoint, headers, timeout) using OpenTelemetry SDK already in repo.
2. Provide compression/rotation options for file sink (daily rotation, max size, retention days).
3. Add stress/performance tests measuring throughput and latency under heavy audit load.
4. Create CI checks ensuring new admin handlers call the audit helper (lint rule or test harness).

## Testing Strategy
- **Unit tests**: audit package (buffering, serialization), middleware (context extraction), config parsing.
- **Integration tests**: start server with in-memory audit sink; execute HTTP/CLI flows to assert events.
- **Performance tests**: go test benchmarks for writer throughput; load test to measure tail-latency impact.
- **Security tests**: ensure sensitive payloads are hashed or redacted; verify events still emit when handlers panic (defer wrappers).

## Risks & Mitigations
- **Performance regression**: mitigate with async buffers and metrics to detect pressure; allow drop policy.
- **Sensitive data leakage**: implement field scrubbers (hash body, limit metadata) and document guidelines.
- **Partial coverage**: require code owners to add audit instrumentation when touching privileged handlers; consider automated lint.
- **Storage exhaustion**: document rotation, allow operators to point at external volumes, expose metrics/alerts.

## Deliverables Checklist
- [ ] `internal/audit` package with writers, metrics, tests.
- [ ] Config flags + README updates.
- [ ] Middleware instrumentation covering all privileged routes (HTTP, GraphQL, CLI).
- [ ] CLI tools (`konsulctl audit tail/export`) and optional HTTP endpoints.
- [ ] Documentation (`docs/security-audit-logging.md`) and updated roadmap.
