# Structured Logging - Complete Documentation

Comprehensive guide for structured logging in Konsul using Zap.

## Overview

Konsul implements **structured logging** with Uber's Zap library, providing high-performance, structured, and leveled logging with automatic request correlation.

### Quick Start

**Configure logging:**
```bash
# JSON format for production
KONSUL_LOG_LEVEL=info \
KONSUL_LOG_FORMAT=json \
./konsul

# Text format for development
KONSUL_LOG_LEVEL=debug \
KONSUL_LOG_FORMAT=text \
./konsul
```

**Log output (JSON):**
```json
{
  "level": "info",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Request completed",
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "method": "GET",
  "path": "/kv/mykey",
  "status": 200,
  "duration": 0.123
}
```

**Log output (Text):**
```
2025-10-14T10:30:00.123Z	INFO	Request completed	request_id=550e8400-e29b-41d4-a935-446655440000 method=GET path=/kv/mykey status=200 duration=0.123s
```

---

## Table of Contents

- [Configuration](#configuration)
- [Log Levels](#log-levels)
- [Log Formats](#log-formats)
- [Request Logging](#request-logging)
- [Correlation IDs](#correlation-ids)
- [Field Types](#field-types)
- [Best Practices](#best-practices)
- [Log Aggregation](#log-aggregation)
- [Querying Logs](#querying-logs)
- [Troubleshooting](#troubleshooting)
- [Implementation Details](#implementation-details)

---

## Configuration

### Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `KONSUL_LOG_LEVEL` | string | `info` | Minimum log level |
| `KONSUL_LOG_FORMAT` | string | `text` | Log output format |

### Log Levels

| Level | Value | Use Case |
|-------|-------|----------|
| `debug` | Most verbose | Development, troubleshooting |
| `info` | Standard | Production, normal operations |
| `warn` | Warnings | Potential issues |
| `error` | Errors | Application errors |

**Level hierarchy:**
```
debug ← info ← warn ← error
  ↑      ↑      ↑      ↑
  │      │      │      └─── Only errors
  │      │      └────────── Warnings and errors
  │      └───────────────── Info, warnings, and errors
  └──────────────────────── Everything
```

---

### Configuration Examples

**Production (JSON, Info level):**
```bash
export KONSUL_LOG_LEVEL=info
export KONSUL_LOG_FORMAT=json
./konsul
```

**Development (Text, Debug level):**
```bash
export KONSUL_LOG_LEVEL=debug
export KONSUL_LOG_FORMAT=text
./konsul
```

**Quiet mode (Errors only):**
```bash
export KONSUL_LOG_LEVEL=error
export KONSUL_LOG_FORMAT=json
./konsul
```

---

## Log Levels

### DEBUG

**Purpose:** Detailed information for diagnosing issues

**When:** Development, debugging production issues

**Example:**
```go
log.Debug("Cache lookup",
    logger.String("key", "app/config"),
    logger.Bool("found", true),
    logger.Int("size", 1024),
)
```

**Output:**
```json
{
  "level": "debug",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Cache lookup",
  "key": "app/config",
  "found": true,
  "size": 1024
}
```

---

### INFO

**Purpose:** General informational messages

**When:** Production, normal operations

**Example:**
```go
log.Info("Server started",
    logger.String("host", "0.0.0.0"),
    logger.Int("port", 8888),
    logger.String("version", "0.1.0"),
)
```

**Output:**
```json
{
  "level": "info",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Server started",
  "host": "0.0.0.0",
  "port": 8888,
  "version": "0.1.0"
}
```

---

### WARN

**Purpose:** Warning messages for potentially harmful situations

**When:** Deprecated features, resource limits, recoverable errors

**Example:**
```go
log.Warn("Rate limit approaching",
    logger.String("client_ip", "192.168.1.100"),
    logger.Int("tokens_remaining", 5),
    logger.Int("limit", 100),
)
```

**Output:**
```json
{
  "level": "warn",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Rate limit approaching",
  "client_ip": "192.168.1.100",
  "tokens_remaining": 5,
  "limit": 100
}
```

---

### ERROR

**Purpose:** Error events that might still allow the application to continue

**When:** Request failures, external service errors, validation errors

**Example:**
```go
log.Error("Failed to connect to database",
    logger.Error(err),
    logger.String("host", "db.example.com"),
    logger.Int("port", 5432),
    logger.Duration("timeout", 30*time.Second),
)
```

**Output:**
```json
{
  "level": "error",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Failed to connect to database",
  "error": "connection timeout",
  "host": "db.example.com",
  "port": 5432,
  "timeout": "30s"
}
```

---

## Log Formats

### JSON Format

**Purpose:** Machine-readable, production logs

**Characteristics:**
- ✅ Easy to parse and query
- ✅ Works with log aggregation systems
- ✅ Consistent structure
- ✅ Compact

**Example:**
```json
{
  "level": "info",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Request completed",
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "method": "GET",
  "path": "/kv/mykey",
  "status": 200,
  "duration": 0.123
}
```

**Configuration:**
```bash
KONSUL_LOG_FORMAT=json
```

---

### Text Format

**Purpose:** Human-readable, development logs

**Characteristics:**
- ✅ Easy to read
- ✅ Colored output (in terminal)
- ✅ Good for debugging
- ❌ Harder to parse programmatically

**Example:**
```
2025-10-14T10:30:00.123Z	INFO	Request completed	request_id=550e8400-e29b-41d4-a935-446655440000 method=GET path=/kv/mykey status=200 duration=0.123s
```

**Configuration:**
```bash
KONSUL_LOG_FORMAT=text
```

---

## Request Logging

### Automatic Request Logging

**All HTTP requests are automatically logged** with:
- Request ID (UUID)
- Method (GET, POST, etc.)
- Path
- Client IP
- User agent
- Response status
- Duration
- Response size

**Request start:**
```json
{
  "level": "info",
  "ts": "2025-10-14T10:30:00.000Z",
  "msg": "Request started",
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "method": "GET",
  "path": "/kv/mykey",
  "ip": "192.168.1.100",
  "user_agent": "curl/7.68.0"
}
```

**Request completion:**
```json
{
  "level": "info",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Request completed",
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "method": "GET",
  "path": "/kv/mykey",
  "status": 200,
  "duration": 0.123,
  "response_size": 1024
}
```

---

### Log Level by Status Code

**Automatic log level selection:**

| Status Code | Log Level | Example |
|-------------|-----------|---------|
| 200-299 | `INFO` | Successful requests |
| 400-499 | `WARN` | Client errors |
| 500-599 | `ERROR` | Server errors |

**Example (client error):**
```json
{
  "level": "warn",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Request completed",
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "method": "GET",
  "path": "/kv/nonexistent",
  "status": 404,
  "duration": 0.005
}
```

**Example (server error):**
```json
{
  "level": "error",
  "ts": "2025-10-14T10:30:00.123Z",
  "msg": "Request completed",
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "method": "POST",
  "path": "/kv/mykey",
  "status": 500,
  "duration": 0.050
}
```

---

## Correlation IDs

### Request ID

**Every request gets a unique UUID** for correlation.

**Generated in middleware:**
```go
requestID := uuid.New().String()
c.Locals("request_id", requestID)
```

**Used in all logs for that request:**
```json
{
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "msg": "Processing KV operation"
}
```

**Access in handlers:**
```go
requestID := middleware.GetRequestID(c)
log.Info("Custom log", logger.String("request_id", requestID))
```

---

### Trace ID

**If tracing is enabled**, trace ID is also included:

```json
{
  "level": "info",
  "msg": "Request completed",
  "request_id": "550e8400-e29b-41d4-a935-446655440000",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "method": "GET",
  "path": "/kv/mykey"
}
```

**Correlation:**
- Find all logs for a request: filter by `request_id`
- Find all logs for a distributed trace: filter by `trace_id`

---

## Field Types

### String Fields

```go
logger.String("key", "value")
logger.String("method", "GET")
logger.String("path", "/kv/mykey")
```

**Output:**
```json
{"key": "value", "method": "GET", "path": "/kv/mykey"}
```

---

### Numeric Fields

**Integer:**
```go
logger.Int("status", 200)
logger.Int("port", 8888)
logger.Int("count", 42)
```

**Output:**
```json
{"status": 200, "port": 8888, "count": 42}
```

---

### Duration Fields

```go
logger.Duration("duration", 123*time.Millisecond)
logger.Duration("timeout", 30*time.Second)
```

**Output:**
```json
{"duration": 0.123, "timeout": 30}
```

---

### Error Fields

```go
err := errors.New("connection timeout")
logger.Error(err)
```

**Output:**
```json
{"error": "connection timeout"}
```

---

### Custom Fields

**Multiple fields:**
```go
log.Info("Operation completed",
    logger.String("operation", "backup"),
    logger.Int("records", 1000),
    logger.Duration("duration", 5*time.Second),
    logger.Bool("success", true),
)
```

**Output:**
```json
{
  "level": "info",
  "msg": "Operation completed",
  "operation": "backup",
  "records": 1000,
  "duration": 5,
  "success": true
}
```

---

## Best Practices

### 1. Use Appropriate Log Levels

**Guidelines:**

```go
// DEBUG: Detailed diagnostic info
log.Debug("Checking cache", logger.String("key", key))

// INFO: Important business events
log.Info("Service registered", logger.String("name", serviceName))

// WARN: Unexpected but handled situations
log.Warn("Retry attempt", logger.Int("attempt", 2))

// ERROR: Errors requiring attention
log.Error("Failed to save", logger.Error(err))
```

**Don't:**
```go
// ❌ Using wrong level
log.Info("Critical database failure", logger.Error(err))  // Should be ERROR

// ❌ Using DEBUG for business events
log.Debug("Payment processed", logger.Float64("amount", 99.99))  // Should be INFO
```

---

### 2. Include Context

**Good:**
```go
log.Error("Failed to fetch user",
    logger.Error(err),
    logger.String("user_id", userID),
    logger.String("endpoint", "/api/users"),
    logger.Duration("timeout", timeout),
)
```

**Bad:**
```go
log.Error("Error occurred", logger.Error(err))  // No context
```

---

### 3. Use Structured Fields

**Good:**
```go
log.Info("Request processed",
    logger.String("method", "GET"),
    logger.String("path", "/kv/mykey"),
    logger.Int("status", 200),
    logger.Duration("duration", duration),
)
```

**Bad:**
```go
log.Info(fmt.Sprintf("Processed GET /kv/mykey with status 200 in %s", duration))
// Unstructured, hard to query
```

---

### 4. Don't Log Sensitive Data

**Never log:**
- Passwords
- API keys
- JWT tokens
- Credit card numbers
- Personal identifiable information (PII)

**Bad:**
```go
log.Info("User login", logger.String("password", password))  // ❌
log.Info("API call", logger.String("api_key", apiKey))  // ❌
```

**Good:**
```go
log.Info("User login", logger.String("username", username))  // ✅
log.Info("API call", logger.String("api_key_id", keyID))  // ✅ (ID, not key)
```

---

### 5. Use Request-Scoped Logger

**Get request-scoped logger:**
```go
// In handler
log := middleware.GetLogger(c)

log.Info("Processing request")  // Automatically includes request_id
```

**Manual correlation:**
```go
requestID := middleware.GetRequestID(c)
log.Info("Custom operation",
    logger.String("request_id", requestID),
)
```

---

### 6. Log at Decision Points

**Good places to log:**
```go
// Service start
log.Info("Server starting", logger.Int("port", port))

// Business operations
log.Info("Service registered", logger.String("name", name))

// Errors
log.Error("Operation failed", logger.Error(err))

// Important state changes
log.Info("Configuration reloaded")

// Performance warnings
log.Warn("Slow query", logger.Duration("duration", duration))
```

---

### 7. Consistent Field Names

**Use standard field names:**

| Field | Name | Type |
|-------|------|------|
| Request ID | `request_id` | string |
| Trace ID | `trace_id` | string |
| User ID | `user_id` | string |
| Service name | `service` | string |
| Duration | `duration` | float (seconds) |
| Error | `error` | string |
| Status code | `status` | int |
| HTTP method | `method` | string |
| Path | `path` | string |

---

## Log Aggregation

### Grafana Loki

**Docker Compose setup:**
```yaml
version: '3.8'
services:
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    volumes:
      - ./loki-config.yaml:/etc/loki/local-config.yaml

  promtail:
    image: grafana/promtail:latest
    volumes:
      - /var/log:/var/log
      - ./promtail-config.yaml:/etc/promtail/config.yaml
    depends_on:
      - loki

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    depends_on:
      - loki

  konsul:
    image: konsul:latest
    environment:
      - KONSUL_LOG_FORMAT=json
      - KONSUL_LOG_LEVEL=info
    ports:
      - "8888:8888"
```

**Loki config (`loki-config.yaml`):**
```yaml
auth_enabled: false

server:
  http_listen_port: 3100

ingester:
  lifecycler:
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

storage_config:
  boltdb_shipper:
    active_index_directory: /tmp/loki/boltdb-shipper-active
    cache_location: /tmp/loki/boltdb-shipper-cache
    shared_store: filesystem
  filesystem:
    directory: /tmp/loki/chunks
```

**Promtail config (`promtail-config.yaml`):**
```yaml
server:
  http_listen_port: 9080

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: konsul
    static_configs:
      - targets:
          - localhost
        labels:
          job: konsul
          __path__: /var/log/konsul/*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            msg: msg
            request_id: request_id
      - labels:
          level:
          request_id:
```

---

### ELK Stack (Elasticsearch, Logstash, Kibana)

**Filebeat config:**
```yaml
filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /var/log/konsul/*.log
    json.keys_under_root: true
    json.add_error_key: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "konsul-%{+yyyy.MM.dd}"

setup.ilm.enabled: false
```

---

### Fluentd

**Fluentd config:**
```xml
<source>
  @type tail
  path /var/log/konsul/*.log
  pos_file /var/log/td-agent/konsul.pos
  tag konsul
  <parse>
    @type json
    time_key ts
    time_format %Y-%m-%dT%H:%M:%S.%NZ
  </parse>
</source>

<match konsul>
  @type elasticsearch
  host elasticsearch
  port 9200
  index_name konsul
  type_name _doc
</match>
```

---

## Querying Logs

### Grafana Loki Queries

**Find all logs for a request:**
```logql
{job="konsul"} |= "550e8400-e29b-41d4-a935-446655440000"
```

**Find errors:**
```logql
{job="konsul"} | json | level="error"
```

**Find slow requests (>1s):**
```logql
{job="konsul"} | json | duration > 1
```

**Find requests by path:**
```logql
{job="konsul"} | json | path="/kv/mykey"
```

**Find client errors:**
```logql
{job="konsul"} | json | status >= 400 and status < 500
```

**Count errors per minute:**
```logql
sum(rate({job="konsul"} | json | level="error" [1m]))
```

---

### jq Queries

**Filter by level:**
```bash
cat konsul.log | jq 'select(.level == "error")'
```

**Filter by request ID:**
```bash
cat konsul.log | jq 'select(.request_id == "550e8400-e29b-41d4-a935-446655440000")'
```

**Extract specific fields:**
```bash
cat konsul.log | jq '{time: .ts, level: .level, message: .msg, status: .status}'
```

**Count by log level:**
```bash
cat konsul.log | jq -r '.level' | sort | uniq -c
```

**Find slow requests:**
```bash
cat konsul.log | jq 'select(.duration > 1)'
```

---

### grep Examples

**Find errors:**
```bash
grep '"level":"error"' konsul.log
```

**Find requests by path:**
```bash
grep '"/kv/mykey"' konsul.log
```

**Find by request ID:**
```bash
grep '550e8400-e29b-41d4-a935-446655440000' konsul.log
```

---

## Troubleshooting

### Issue: No Logs Appearing

**Check:**
1. Log level is not too restrictive
2. Application is writing to expected output

**Solutions:**
```bash
# Lower log level
KONSUL_LOG_LEVEL=debug

# Check if logs are going to stdout
docker logs konsul
```

---

### Issue: Logs Not in JSON Format

**Check configuration:**
```bash
echo $KONSUL_LOG_FORMAT
```

**Fix:**
```bash
KONSUL_LOG_FORMAT=json
```

---

### Issue: Missing Fields in Logs

**Check:**
1. Using correct logger instance
2. Fields are being added

**Example:**
```go
// Use request-scoped logger
log := middleware.GetLogger(c)
log.Info("Message", logger.String("field", "value"))
```

---

### Issue: Too Verbose Logs

**Solution:**
```bash
# Increase log level
KONSUL_LOG_LEVEL=info  # or warn, or error
```

---

## Implementation Details

### Architecture

**Components:**

1. **Logger Interface** (`internal/logger/logger.go`)
   - Abstraction over Zap logger
   - Debug, Info, Warn, Error methods
   - Field helpers

2. **zapLogger** (`internal/logger/logger.go`)
   - Implements Logger interface
   - Wraps zap.Logger

3. **RequestLogging Middleware** (`internal/middleware/logging.go`)
   - Generates request IDs
   - Creates request-scoped loggers
   - Logs request start and completion

---

### Logger Interface

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    WithRequest(requestID string) Logger
    WithFields(fields ...Field) Logger
}
```

---

### Creating Loggers

**Application logger:**
```go
log := logger.NewFromConfig("info", "json")
```

**Request-scoped logger:**
```go
requestLogger := log.WithRequest(requestID)
requestLogger.Info("Processing request")
// Automatically includes request_id field
```

**Logger with custom fields:**
```go
serviceLogger := log.WithFields(
    logger.String("service", "kv-store"),
    logger.String("version", "1.0.0"),
)
serviceLogger.Info("Service initialized")
```

---

### Field Helpers

```go
// String field
logger.String(key, value string) Field

// Integer field
logger.Int(key string, value int) Field

// Duration field (converts to seconds)
logger.Duration(key string, value time.Duration) Field

// Error field
logger.Error(err error) Field
```

---

### Middleware Integration

**Request logging middleware:**
```go
func RequestLogging(log logger.Logger) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Generate request ID
        requestID := uuid.New().String()
        c.Locals("request_id", requestID)

        // Create request-scoped logger
        requestLogger := log.WithRequest(requestID)
        c.Locals("logger", requestLogger)

        // Log request start
        requestLogger.Info("Request started", ...)

        // Process request
        err := c.Next()

        // Log request completion
        requestLogger.Info("Request completed", ...)

        return err
    }
}
```

---

## See Also

- [OpenTelemetry Tracing Documentation](tracing.md)
- [Metrics Documentation](metrics.md)
- [Grafana Loki](https://grafana.com/oss/loki/)
- [Uber Zap](https://github.com/uber-go/zap)

---

## Changelog

- **2025-10-14**: Initial comprehensive documentation
- **Version**: 0.1.0
- **Status**: ✅ Production Ready

---

## Future Enhancements

Planned improvements for logging:

- [ ] Log rotation configuration
- [ ] Log sampling (reduce high-volume logs)
- [ ] Contextual logging (automatic field propagation)
- [ ] Dynamic log level adjustment (via API)
- [ ] Log filtering rules
- [ ] Sensitive data masking
- [ ] Log archival and retention policies
- [ ] Integration with external logging services (DataDog, New Relic)
