# Audit Logging Guide

## Overview

Konsul's audit logging subsystem provides a tamper-evident history of **who changed what and when** across all privileged operations. This includes KV writes, service lifecycle events, ACL updates, rate-limit overrides, backup actions, and more.

## Features

- **Comprehensive Event Capture**: Records both successful and denied attempts with actor identity, auth mechanism, source IP, resource, action, and outcome
- **Immutable Records**: JSON-structured events suitable for SIEM ingestion
- **Non-Blocking**: Asynchronous buffering ensures audit writes don't impact request latency
- **Multiple Sinks**: File (with rotation), stdout, and future OTLP support
- **Configurable Behavior**: Buffer size, flush interval, and drop policy controls
- **Prometheus Metrics**: Track events processed, dropped, and flush performance

## Configuration

Configure audit logging via environment variables:

```bash
# Enable audit logging
export KONSUL_AUDIT_ENABLED=true

# Choose sink: file or stdout
export KONSUL_AUDIT_SINK=file

# File path (when sink=file)
export KONSUL_AUDIT_FILE_PATH=./logs/audit.log

# Buffer configuration
export KONSUL_AUDIT_BUFFER_SIZE=1024
export KONSUL_AUDIT_FLUSH_INTERVAL=1s

# Behavior when buffer is full: drop or block
export KONSUL_AUDIT_DROP_POLICY=drop
```

### Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_AUDIT_ENABLED` | `false` | Enable the audit logging subsystem |
| `KONSUL_AUDIT_SINK` | `file` | Destination for audit events (`file` or `stdout`) |
| `KONSUL_AUDIT_FILE_PATH` | `./logs/audit.log` | Path to the audit log when `sink=file` |
| `KONSUL_AUDIT_BUFFER_SIZE` | `1024` | Channel size for pending audit events |
| `KONSUL_AUDIT_FLUSH_INTERVAL` | `1s` | Interval for flushing buffered events |
| `KONSUL_AUDIT_DROP_POLICY` | `drop` | Behavior when the buffer is full (`drop` or `block`) |

### Drop Policy

- **`drop`** (default): When the buffer is full, new events are dropped and a metric is incremented. Use this in high-throughput environments to prevent blocking.
- **`block`**: When the buffer is full, the caller blocks until space is available. Use this when audit completeness is critical.

## Event Schema

Each audit event contains the following fields:

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-02-14T10:30:00Z",
  "action": "kv.set",
  "result": "success",
  "resource": {
    "type": "kv",
    "id": "config/app/database",
    "namespace": "production"
  },
  "actor": {
    "id": "user123",
    "type": "user",
    "name": "john.doe",
    "roles": ["admin", "developer"],
    "token_id": "jwt-abc123"
  },
  "source_ip": "192.168.1.100",
  "auth_method": "jwt",
  "http_method": "PUT",
  "http_path": "/api/v1/kv/config/app/database",
  "http_status": 200,
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "request_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "metadata": {
    "namespace": "production",
    "ttl": "3600s"
  }
}
```

### Field Descriptions

- **`event_id`**: Unique identifier for this event (UUID)
- **`timestamp`**: When the event occurred (UTC)
- **`action`**: What operation was performed (e.g., `kv.set`, `service.register`, `acl.token.create`)
- **`result`**: Outcome - `success`, `denied`, `error`
- **`resource`**: What was accessed (type, ID, optional namespace)
- **`actor`**: Who performed the action (identity, type, roles, token ID)
- **`source_ip`**: Client IP address
- **`auth_method`**: How the actor authenticated (`jwt`, `api_key`, `anonymous`)
- **`http_method`**: HTTP verb (GET, POST, PUT, DELETE)
- **`http_path`**: Request path
- **`http_status`**: HTTP response code
- **`trace_id`** / **`span_id`**: Distributed tracing correlation IDs
- **`request_hash`**: SHA-256 hash of request body (for integrity verification)
- **`metadata`**: Additional contextual key-value pairs

## Action Types

### KV Operations
- `kv.set` - Create or update a key
- `kv.get` - Read a key
- `kv.delete` - Delete a key
- `kv.list` - List keys

### Service Operations
- `service.register` - Register a new service instance
- `service.deregister` - Remove a service instance
- `service.heartbeat` - Update service health/TTL
- `service.get` - Read service details
- `service.list` - List services

### ACL Operations
- `acl.token.create` - Create a new token
- `acl.token.revoke` - Revoke an existing token
- `acl.token.read` - Read token details
- `acl.policy.create` - Create a policy
- `acl.policy.update` - Update a policy
- `acl.policy.delete` - Delete a policy
- `acl.policy.read` - Read policy details

### Backup Operations
- `backup.create` - Create a new backup
- `backup.restore` - Restore from backup
- `backup.delete` - Delete a backup
- `backup.download` - Download backup data
- `backup.list` - List available backups

### Admin Operations
- `admin.ratelimit.create` - Create rate limit override
- `admin.ratelimit.update` - Update rate limit
- `admin.ratelimit.delete` - Remove rate limit override
- `admin.ratelimit.get` - Read rate limit config
- `admin.config.*` - Configuration changes

## Integration Examples

### HTTP Handler with Audit Logging

```go
import (
    "github.com/gofiber/fiber/v2"
    "github.com/neogan74/konsul/internal/audit"
    "github.com/neogan74/konsul/internal/middleware"
)

func setupRoutes(app *fiber.App, auditMgr *audit.Manager) {
    // Apply audit middleware to KV routes
    kvGroup := app.Group("/api/v1/kv")
    kvGroup.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditMgr,
        ResourceType: "kv",
        ActionMapper: middleware.KVActionMapper,
    }))

    kvGroup.Put("/:key", handleKVSet)
    kvGroup.Get("/:key", handleKVGet)
    kvGroup.Delete("/:key", handleKVDelete)

    // Apply audit middleware to service routes
    svcGroup := app.Group("/api/v1/service")
    svcGroup.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditMgr,
        ResourceType: "service",
        ActionMapper: middleware.ServiceActionMapper,
    }))

    svcGroup.Post("/", handleServiceRegister)
    svcGroup.Delete("/:name", handleServiceDeregister)
}
```

### Manual Event Recording

For operations not covered by HTTP middleware:

```go
import (
    "context"
    "github.com/neogan74/konsul/internal/audit"
)

func performBackgroundTask(ctx context.Context, mgr *audit.Manager) error {
    // ... perform operation ...

    // Record audit event
    event := &audit.Event{
        Action: "background.cleanup",
        Result: "success",
        Resource: audit.Resource{
            Type: "service",
            ID:   "expired-instances",
        },
        Actor: audit.Actor{
            Type: "system",
            Name: "cleanup-job",
        },
        Metadata: map[string]string{
            "cleaned_count": "42",
        },
    }

    eventID, err := mgr.Record(ctx, event)
    if err != nil {
        log.Error("Failed to record audit event", logger.Error(err))
    }

    return nil
}
```

## Monitoring & Metrics

Konsul exports Prometheus metrics for audit logging:

```promql
# Total events processed by sink and status
konsul_audit_events_total{sink="file",status="written"}

# Events dropped (by sink and reason)
konsul_audit_events_dropped_total{sink="file",reason="buffer_full"}
konsul_audit_events_dropped_total{sink="file",reason="manager_closed"}

# Flush operation duration
konsul_audit_writer_flush_duration_seconds{sink="file"}
```

### Alerting Examples

```yaml
# Alert when audit buffer is dropping events
- alert: AuditEventsDropped
  expr: rate(konsul_audit_events_dropped_total[5m]) > 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Audit events are being dropped"
    description: "{{ $value }} events/sec are being dropped from {{ $labels.sink }}"

# Alert on high flush latency
- alert: AuditFlushSlow
  expr: histogram_quantile(0.99, rate(konsul_audit_writer_flush_duration_seconds_bucket[5m])) > 0.1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Audit log flush is slow"
    description: "P99 flush duration is {{ $value }}s for {{ $labels.sink }}"
```

## Log Rotation

When using `sink=file`, implement log rotation to manage disk space:

### Using logrotate (Linux)

Create `/etc/logrotate.d/konsul-audit`:

```
/var/log/konsul/audit.log {
    daily
    rotate 90
    compress
    delaycompress
    missingok
    notifempty
    create 0644 konsul konsul
    postrotate
        kill -USR1 $(cat /var/run/konsul.pid)
    endscript
}
```

### Using Docker Volumes

```yaml
version: '3.8'
services:
  konsul:
    image: konsul:latest
    volumes:
      - audit-logs:/var/log/konsul
    environment:
      KONSUL_AUDIT_ENABLED: "true"
      KONSUL_AUDIT_FILE_PATH: /var/log/konsul/audit.log

volumes:
  audit-logs:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/konsul/audit-logs
```

## Querying Audit Logs

### Using jq

```bash
# Find all failed KV operations
cat audit.log | jq 'select(.action | startswith("kv.")) | select(.result == "denied")'

# Count operations by actor
cat audit.log | jq -r '.actor.name' | sort | uniq -c | sort -rn

# Find all admin actions in the last hour
cat audit.log | jq "select(.timestamp > \"$(date -u -d '1 hour ago' -Iseconds)\" and .action | startswith(\"admin.\"))"

# Export events to CSV
cat audit.log | jq -r '[.timestamp, .action, .actor.name, .result, .http_status] | @csv'
```

### SIEM Integration

#### Splunk

```bash
# Forward to Splunk HEC
tail -F /var/log/konsul/audit.log | \
  while read line; do
    curl -k https://splunk:8088/services/collector \
      -H "Authorization: Splunk YOUR-HEC-TOKEN" \
      -d "{\"event\":$line}"
  done
```

#### Elasticsearch

```bash
# Ingest into Elasticsearch
cat audit.log | jq -c '{"index": {"_index": "konsul-audit"}}, .' | \
  curl -X POST 'localhost:9200/_bulk' -H 'Content-Type: application/x-ndjson' --data-binary @-
```

## Security Best Practices

1. **Separate Storage**: Store audit logs on a separate volume/partition from application data
2. **Immutable Logging**: Use write-once storage or forward to a SIEM immediately
3. **Access Control**: Restrict audit log file permissions to `0600` or `0644`
4. **Regular Review**: Monitor audit logs for suspicious patterns (failed auth, unusual access patterns)
5. **Retention Policy**: Keep logs for compliance period (typically 90-365 days)
6. **Alerting**: Set up alerts for critical events (ACL changes, backup access, admin actions)

## Compliance

Audit logging helps satisfy requirements from:

- **SOC 2**: Access monitoring and change tracking
- **HIPAA**: Audit controls and access logs
- **PCI DSS**: Logging and monitoring of access to system components
- **GDPR**: Accountability and data access records
- **ISO 27001**: Security event logging and monitoring

## Troubleshooting

### No Events Being Written

1. Check if audit is enabled: `KONSUL_AUDIT_ENABLED=true`
2. Verify file path permissions
3. Check metrics for drop counters
4. Review application logs for audit initialization errors

### High Memory Usage

1. Reduce `KONSUL_AUDIT_BUFFER_SIZE`
2. Decrease `KONSUL_AUDIT_FLUSH_INTERVAL`
3. Switch to `KONSUL_AUDIT_DROP_POLICY=drop`

### Events Being Dropped

1. Increase `KONSUL_AUDIT_BUFFER_SIZE`
2. Use faster storage for audit logs (SSD)
3. Switch to `KONSUL_AUDIT_SINK=stdout` and use external log aggregation

## Future Enhancements

Planned features for future releases:

- OTLP/HTTP sink for direct OpenTelemetry integration
- Automatic log compression and rotation
- Event filtering by action type or actor
- Batch writing optimizations
- Encrypted audit logs
- Remote syslog support

## References

- ADR-0019: [Audit Logging for Operational Changes](./adr/0019-audit-logging.md)
- [Structured Logging](./adr/0005-structured-logging.md)
- [ACL System](./adr/0010-acl-system.md)
- [Rate Limiting Management API](./adr/0014-rate-limiting-management-api.md)
