# Audit Logging Routes Coverage

This document summarizes which routes in Konsul have audit logging enabled.

## ✅ Routes with Audit Logging

### 1. KV Store Operations (`/kv/*`)
**Middleware**: `KVActionMapper`
- `GET /kv/` - List keys (action: `kv.list`)
- `GET /kv/:key` - Read key (action: `kv.get`)
- `PUT /kv/:key` - Set key (action: `kv.set`)
- `DELETE /kv/:key` - Delete key (action: `kv.delete`)

**Events captured**: All KV mutations and reads

### 2. Service Discovery Operations
**Middleware**: `ServiceActionMapper`
- `PUT /register` - Register service (action: `service.register`)
- `DELETE /deregister/:name` - Deregister service (action: `service.deregister`)
- `PUT /heartbeat/:name` - Service heartbeat (action: `service.heartbeat`)

**Events captured**: Service registration, deregistration, and health updates

**Note**: Read-only endpoints (`GET /services/*`) are **not** audited to reduce log volume.

### 3. ACL Policy Management (`/acl/*`)
**Middleware**: `ACLActionMapper`
- `POST /acl/policies` - Create policy (action: `acl.policy.create`)
- `GET /acl/policies` - List policies (action: `acl.policy.read`)
- `GET /acl/policies/:name` - Get policy (action: `acl.policy.read`)
- `PUT /acl/policies/:name` - Update policy (action: `acl.policy.update`)
- `DELETE /acl/policies/:name` - Delete policy (action: `acl.policy.delete`)
- `POST /acl/test` - Test policy (action: `acl.policy.test`)

**Events captured**: All ACL policy changes (create, update, delete)

### 4. API Key Management (`/auth/apikeys/*`)
**Middleware**: Default action mapper
- `POST /auth/apikeys/` - Create API key (action: `auth.create`)
- `GET /auth/apikeys/` - List API keys (action: `auth.list`)
- `GET /auth/apikeys/:id` - Get API key (action: `auth.read`)
- `PUT /auth/apikeys/:id` - Update API key (action: `auth.update`)
- `DELETE /auth/apikeys/:id` - Delete API key (action: `auth.delete`)
- `POST /auth/apikeys/:id/revoke` - Revoke API key (action: `auth.create`)

**Events captured**: All API key lifecycle operations

### 5. Rate Limit Admin (`/admin/ratelimit/*`)
**Middleware**: `AdminActionMapper`
- `GET /admin/ratelimit/stats` - Get stats (action: `admin.ratelimit.get`)
- `GET /admin/ratelimit/config` - Get config (action: `admin.ratelimit.get`)
- `GET /admin/ratelimit/clients` - Get clients (action: `admin.ratelimit.get`)
- `GET /admin/ratelimit/client/:identifier` - Get client status (action: `admin.ratelimit.get`)
- `POST /admin/ratelimit/reset/ip/:ip` - Reset IP limit (action: `admin.ratelimit.create`)
- `POST /admin/ratelimit/reset/apikey/:key_id` - Reset API key limit (action: `admin.ratelimit.create`)
- `POST /admin/ratelimit/reset/all` - Reset all limits (action: `admin.ratelimit.create`)
- `PUT /admin/ratelimit/config` - Update config (action: `admin.ratelimit.update`)

**Events captured**: All rate limit configuration changes

### 6. Backup/Restore Operations
**Middleware**: `BackupActionMapper`
- `POST /backup` - Create backup (action: `backup.create`)
- `POST /restore` - Restore backup (action: `backup.restore`)
- `GET /export` - Export data (action: `backup.download`)
- `POST /import` - Import data (action: `backup.restore`)
- `GET /backups` - List backups (action: `backup.list`)

**Events captured**: All backup creation, restore, and data export operations

## ❌ Routes WITHOUT Audit Logging

### Health Endpoints (by design - high volume)
- `GET /health`
- `GET /health/live`
- `GET /health/ready`
- `GET /health/checks`
- `GET /health/service/:name`
- `PUT /health/check/:id`

### Load Balancer Endpoints (read-only, high volume)
- `GET /lb/service/:name`
- `GET /lb/tags`
- `GET /lb/metadata`
- `GET /lb/query`
- `GET /lb/strategy`
- `PUT /lb/strategy` - **Could be added** if strategy changes need auditing

### Service Query Endpoints (read-only)
- `GET /services/`
- `GET /services/:name`
- `GET /services/query/tags`
- `GET /services/query/metadata`
- `GET /services/query`

### Auth Endpoints (public, no sensitive operations)
- `POST /auth/login` - Authentication attempts are logged via request logging
- `POST /auth/refresh`
- `GET /auth/verify`

### Metrics & Admin UI
- `GET /metrics` - Prometheus metrics
- `GET /admin/*` - Admin UI static files
- `GET /` - Root redirect

### GraphQL (not yet implemented)
- `POST /graphql` - Could be added with custom action mapper

## Configuration

Audit logging is controlled by environment variables:

```bash
# Enable audit logging
export KONSUL_AUDIT_ENABLED=true

# Choose sink: file or stdout
export KONSUL_AUDIT_SINK=file
export KONSUL_AUDIT_FILE_PATH=./logs/audit.log

# Buffer configuration
export KONSUL_AUDIT_BUFFER_SIZE=1024
export KONSUL_AUDIT_FLUSH_INTERVAL=1s
export KONSUL_AUDIT_DROP_POLICY=drop
```

When `KONSUL_AUDIT_ENABLED=false`, the middleware becomes a no-op and adds minimal overhead.

## Event Schema

Each audit event includes:
- **event_id**: Unique UUID
- **timestamp**: When the event occurred (UTC)
- **action**: Operation performed (e.g., `kv.set`, `service.register`)
- **result**: Outcome (`success`, `denied`, `error`)
- **resource**: What was accessed (type, ID, namespace)
- **actor**: Who performed the action (ID, type, roles, token ID)
- **source_ip**: Client IP address
- **auth_method**: How authenticated (`jwt`, `api_key`, `anonymous`)
- **http_method**: HTTP verb
- **http_path**: Request path
- **http_status**: Response code
- **request_hash**: SHA-256 hash of request body
- **trace_id/span_id**: Distributed tracing IDs

## Prometheus Metrics

Monitor audit logging health:

```promql
# Total events processed
konsul_audit_events_total{sink="file",status="written"}

# Events dropped (alerts should fire on this)
konsul_audit_events_dropped_total{sink="file",reason="buffer_full"}

# Flush latency
konsul_audit_writer_flush_duration_seconds{sink="file"}
```

## Future Enhancements

Consider adding audit logging to:
1. **Load balancer strategy changes** (`PUT /lb/strategy`)
2. **GraphQL mutations** (custom action mapper needed)
3. **Health check updates** (`PUT /health/check/:id`) - if TTL updates are sensitive
4. **Service queries** - if you need to audit who accessed service data

## See Also

- [Audit Logging Guide](../audit-logging.md) - Complete documentation
- [Audit Integration Example](./audit-integration-example.md) - Code examples
- [ADR-0019: Audit Logging](../adr/0019-audit-logging.md) - Design decisions
