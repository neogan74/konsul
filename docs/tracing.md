# OpenTelemetry Tracing - Complete Documentation

Comprehensive guide for distributed tracing in Konsul using OpenTelemetry.

## Overview

Konsul implements **OpenTelemetry** distributed tracing to provide visibility into request flows, performance bottlenecks, and system behavior. All HTTP requests are automatically traced with detailed span information.

### Quick Start

**Enable tracing:**
```bash
KONSUL_TRACING_ENABLED=true \
KONSUL_TRACING_ENDPOINT=tempo:4318 \
KONSUL_TRACING_SAMPLING_RATIO=1.0 \
./konsul
```

**View traces in Grafana Tempo:**
1. Open Grafana
2. Navigate to Explore → Tempo
3. Search for traces with service name: `konsul`

**Trace ID in response:**
```bash
curl -v http://localhost:8888/kv/mykey
# Look for header:
# X-Trace-Id: 4bf92f3577b34da6a3ce929d0e0e4736
```

---

## Table of Contents

- [Configuration](#configuration)
- [How It Works](#how-it-works)
- [Span Attributes](#span-attributes)
- [Context Propagation](#context-propagation)
- [Integration](#integration)
- [Visualization](#visualization)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Implementation Details](#implementation-details)

---

## Configuration

### Environment Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `KONSUL_TRACING_ENABLED` | bool | `false` | Enable tracing system |
| `KONSUL_TRACING_ENDPOINT` | string | `localhost:4318` | OTLP HTTP endpoint |
| `KONSUL_TRACING_SERVICE_NAME` | string | `konsul` | Service name in traces |
| `KONSUL_TRACING_SERVICE_VERSION` | string | `0.1.0` | Service version |
| `KONSUL_TRACING_ENVIRONMENT` | string | `production` | Deployment environment |
| `KONSUL_TRACING_SAMPLING_RATIO` | float | `1.0` | Sampling ratio (0.0-1.0) |
| `KONSUL_TRACING_INSECURE` | bool | `false` | Use insecure connection |

### Configuration Examples

**Basic setup with Tempo:**
```bash
export KONSUL_TRACING_ENABLED=true
export KONSUL_TRACING_ENDPOINT=tempo:4318
export KONSUL_TRACING_INSECURE=true
```

**Production with Jaeger:**
```bash
export KONSUL_TRACING_ENABLED=true
export KONSUL_TRACING_ENDPOINT=jaeger-collector:4318
export KONSUL_TRACING_SERVICE_NAME=konsul
export KONSUL_TRACING_SERVICE_VERSION=1.0.0
export KONSUL_TRACING_ENVIRONMENT=production
export KONSUL_TRACING_SAMPLING_RATIO=0.1  # Sample 10%
```

**Development setup:**
```bash
export KONSUL_TRACING_ENABLED=true
export KONSUL_TRACING_ENDPOINT=localhost:4318
export KONSUL_TRACING_ENVIRONMENT=development
export KONSUL_TRACING_SAMPLING_RATIO=1.0  # Sample 100%
export KONSUL_TRACING_INSECURE=true
```

**Disabled (no overhead):**
```bash
export KONSUL_TRACING_ENABLED=false
```

---

## How It Works

### Tracing Architecture

```
┌────────────────────────────────────────────────────────┐
│                    HTTP Request                        │
└─────────────────────┬──────────────────────────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  TracingMiddleware      │
         │  (extract/create span)  │
         └────────────┬────────────┘
                      │
                      ├─── Extract W3C Trace Context headers
                      ├─── Start new span
                      ├─── Add span attributes (method, path, etc.)
                      ├─── Inject trace ID into context
                      │
                      ▼
         ┌─────────────────────────┐
         │   Request Handler       │
         │   (business logic)      │
         └────────────┬────────────┘
                      │
                      ├─── Process request
                      ├─── Access trace ID from context
                      │
                      ▼
         ┌─────────────────────────┐
         │  End Span               │
         │  (record result)        │
         └────────────┬────────────┘
                      │
                      ├─── Set status code attribute
                      ├─── Record errors if any
                      ├─── Set span status
                      │
                      ▼
         ┌─────────────────────────┐
         │  OTLP HTTP Exporter     │
         │  (batch & send)         │
         └────────────┬────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │  Tracing Backend        │
         │  (Tempo/Jaeger/etc.)    │
         └─────────────────────────┘
```

### Trace Components

**1. Trace:** End-to-end request journey
- Unique Trace ID
- Spans multiple services
- Root span + child spans

**2. Span:** Single operation unit
- Unique Span ID
- Start and end time
- Attributes (metadata)
- Status (Ok, Error)
- Events and errors

**3. Context:** Propagation mechanism
- W3C Trace Context headers
- `traceparent` header format
- Cross-service correlation

---

## Span Attributes

### Automatically Collected

Every HTTP request span includes:

| Attribute | Example | Source |
|-----------|---------|--------|
| `http.method` | `GET` | Request method |
| `http.url` | `http://localhost:8888/kv/mykey` | Full URL |
| `http.route` | `/kv/:key` | Route pattern |
| `http.scheme` | `http` | Protocol |
| `http.target` | `/kv/mykey` | Path |
| `net.host.name` | `localhost` | Hostname |
| `http.user_agent` | `curl/7.68.0` | User agent |
| `http.client_ip` | `192.168.1.100` | Client IP |
| `http.status_code` | `200` | Response status |

**Example span:**
```json
{
  "traceId": "4bf92f3577b34da6a3ce929d0e0e4736",
  "spanId": "00f067aa0ba902b7",
  "name": "GET /kv/:key",
  "kind": "SERVER",
  "startTime": "2025-10-14T10:30:00.000Z",
  "endTime": "2025-10-14T10:30:00.123Z",
  "attributes": {
    "http.method": "GET",
    "http.url": "http://localhost:8888/kv/mykey",
    "http.route": "/kv/:key",
    "http.status_code": 200,
    "http.client_ip": "192.168.1.100"
  },
  "status": {
    "code": "OK"
  }
}
```

---

### Span Status

Spans are marked with appropriate status:

| HTTP Status | Span Status | Reason |
|-------------|-------------|--------|
| `200-299` | `OK` | Successful request |
| `400-499` | `ERROR` | Client error |
| `500-599` | `ERROR` | Server error |

**Error recording:**
```json
{
  "status": {
    "code": "ERROR",
    "message": "Internal server error"
  },
  "events": [
    {
      "name": "exception",
      "timestamp": "2025-10-14T10:30:00.123Z",
      "attributes": {
        "exception.type": "RuntimeError",
        "exception.message": "Failed to connect to database"
      }
    }
  ]
}
```

---

## Context Propagation

### W3C Trace Context

Konsul uses **W3C Trace Context** standard for distributed tracing.

**Request headers (incoming):**
```http
GET /kv/mykey HTTP/1.1
Host: localhost:8888
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
tracestate: vendorname=value
```

**Format:**
```
traceparent: {version}-{trace-id}-{parent-span-id}-{trace-flags}
```

**Response headers (outgoing):**
```http
HTTP/1.1 200 OK
X-Trace-Id: 4bf92f3577b34da6a3ce929d0e0e4736
```

---

### Propagation Example

**Client → Konsul → Database:**

```
1. Client creates trace
   Trace-ID: 4bf92f...
   Span-ID: 00f067...

2. Client sends request with traceparent header
   GET /kv/mykey
   traceparent: 00-4bf92f...-00f067...-01

3. Konsul extracts context and creates child span
   Trace-ID: 4bf92f...  (same)
   Parent-Span-ID: 00f067...
   Span-ID: 12ab34...  (new)

4. Konsul makes database call with propagated context
   (If database supports tracing, it creates another child span)

5. Complete trace shows:
   Client Span (00f067...)
   └─ Konsul Span (12ab34...)
      └─ Database Span (cd56ef...)
```

---

### Manual Propagation

**For custom HTTP clients:**

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// Create HTTP request
req, _ := http.NewRequest("GET", "http://backend:8080/api", nil)

// Inject trace context into headers
propagator := otel.GetTextMapPropagator()
propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

// Make request (trace context propagated)
resp, err := client.Do(req)
```

---

## Integration

### Grafana Tempo

**Docker Compose setup:**
```yaml
version: '3.8'
services:
  tempo:
    image: grafana/tempo:latest
    command: ["-config.file=/etc/tempo.yaml"]
    volumes:
      - ./tempo.yaml:/etc/tempo.yaml
    ports:
      - "4318:4318"  # OTLP HTTP
      - "3200:3200"  # Tempo HTTP

  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
    ports:
      - "3000:3000"
    depends_on:
      - tempo

  konsul:
    image: konsul:latest
    environment:
      - KONSUL_TRACING_ENABLED=true
      - KONSUL_TRACING_ENDPOINT=tempo:4318
      - KONSUL_TRACING_INSECURE=true
    ports:
      - "8888:8888"
    depends_on:
      - tempo
```

**Tempo configuration (`tempo.yaml`):**
```yaml
server:
  http_listen_port: 3200

distributor:
  receivers:
    otlp:
      protocols:
        http:
          endpoint: 0.0.0.0:4318

storage:
  trace:
    backend: local
    local:
      path: /tmp/tempo/blocks
```

**Grafana data source:**
```yaml
apiVersion: 1
datasources:
  - name: Tempo
    type: tempo
    access: proxy
    url: http://tempo:3200
    uid: tempo
```

---

### Jaeger

**Docker Compose:**
```yaml
services:
  jaeger:
    image: jaegertracing/all-in-one:latest
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "16686:16686"  # UI
      - "4318:4318"    # OTLP HTTP

  konsul:
    image: konsul:latest
    environment:
      - KONSUL_TRACING_ENABLED=true
      - KONSUL_TRACING_ENDPOINT=jaeger:4318
    ports:
      - "8888:8888"
```

**Access Jaeger UI:** http://localhost:16686

---

### Honeycomb

**Configuration:**
```bash
export KONSUL_TRACING_ENABLED=true
export KONSUL_TRACING_ENDPOINT=api.honeycomb.io:443
export KONSUL_TRACING_INSECURE=false
export HONEYCOMB_API_KEY=your-api-key

# Note: You may need to modify the exporter to include API key header
```

---

### Cloud Providers

**AWS X-Ray:**
```bash
# Use AWS OTEL Collector
export KONSUL_TRACING_ENABLED=true
export KONSUL_TRACING_ENDPOINT=localhost:4318
# Run AWS OTEL Collector sidecar
```

**Google Cloud Trace:**
```bash
# Use Google Cloud OTEL Collector
export KONSUL_TRACING_ENABLED=true
export KONSUL_TRACING_ENDPOINT=localhost:4318
```

**Azure Monitor:**
```bash
# Use Azure Monitor OTEL Collector
export KONSUL_TRACING_ENABLED=true
export KONSUL_TRACING_ENDPOINT=localhost:4318
```

---

## Visualization

### Grafana Tempo Queries

**Find all traces for Konsul:**
```
{ service.name="konsul" }
```

**Find slow requests (>1s):**
```
{ service.name="konsul" && duration > 1s }
```

**Find errors:**
```
{ service.name="konsul" && status=error }
```

**Find specific operation:**
```
{ service.name="konsul" && name="GET /kv/:key" }
```

**Find by HTTP status:**
```
{ service.name="konsul" && http.status_code=500 }
```

---

### Trace Visualization

**Example trace view:**
```
Request: GET /kv/mykey
─────────────────────────────────────────────────────
│ Trace ID: 4bf92f3577b34da6a3ce929d0e0e4736
│ Duration: 123ms
│ Spans: 1
│ Status: OK
─────────────────────────────────────────────────────

Timeline:
┌─ GET /kv/:key ────────────────────────┐ 123ms
│  Start: 10:30:00.000                  │
│  End:   10:30:00.123                  │
│                                        │
│  Attributes:                           │
│    http.method: GET                   │
│    http.status_code: 200              │
│    http.route: /kv/:key               │
│    http.client_ip: 192.168.1.100     │
└────────────────────────────────────────┘
```

---

## Performance

### Overhead

**Memory:**
- Base: ~5 MB (OTLP exporter)
- Per trace: ~2-5 KB (buffered)
- Batch size: 512 spans

**CPU:**
- Per request: ~10-50 µs
- Negligible impact (<0.1% overhead)

**Network:**
- Batched exports every 5 seconds
- Configurable batch size (default: 512)
- Configurable queue size (default: 2048)

---

### Sampling

Control trace volume with sampling:

**Always sample (development):**
```bash
KONSUL_TRACING_SAMPLING_RATIO=1.0  # 100%
```

**Sample 10% (production):**
```bash
KONSUL_TRACING_SAMPLING_RATIO=0.1  # 10%
```

**Never sample (disabled):**
```bash
KONSUL_TRACING_SAMPLING_RATIO=0.0  # 0%
# Or just disable tracing:
KONSUL_TRACING_ENABLED=false
```

**Sampling behavior:**
- Decision made at trace start
- All spans in a trace are sampled together
- TraceID-based sampling (consistent across services)

---

### Batch Configuration

**In code (`internal/telemetry/tracing.go`):**
```go
sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter,
        sdktrace.WithMaxExportBatchSize(512),      // Batch size
        sdktrace.WithBatchTimeout(5*time.Second),  // Batch interval
        sdktrace.WithMaxQueueSize(2048),           // Queue size
    ),
    // ...
)
```

**Tuning:**
- **High traffic:** Increase batch size, reduce timeout
- **Low traffic:** Decrease batch size, increase timeout
- **Memory constrained:** Reduce queue size

---

## Troubleshooting

### Issue: No Traces Appearing

**Symptoms:**
- Grafana/Jaeger shows no traces
- X-Trace-Id header present in responses

**Diagnosis:**
```bash
# Check if tracing is enabled
curl http://localhost:8888/metrics | grep trace

# Check logs for export errors
docker logs konsul 2>&1 | grep -i trace
```

**Solutions:**
1. **Verify tracing is enabled:**
   ```bash
   KONSUL_TRACING_ENABLED=true
   ```

2. **Check endpoint connectivity:**
   ```bash
   # Test OTLP endpoint
   curl http://tempo:4318/v1/traces
   ```

3. **Check sampling ratio:**
   ```bash
   KONSUL_TRACING_SAMPLING_RATIO=1.0  # Ensure not 0.0
   ```

4. **Verify collector is receiving:**
   ```bash
   # Check Tempo logs
   docker logs tempo
   ```

---

### Issue: Trace Context Not Propagating

**Symptoms:**
- Each service shows isolated traces
- No parent-child relationship

**Diagnosis:**
```bash
# Check if traceparent header is sent
curl -v http://localhost:8888/kv/mykey \
  -H "traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
```

**Solutions:**
1. **Ensure W3C propagation:**
   ```go
   // Should be in InitTracing
   otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
       propagation.TraceContext{},
       propagation.Baggage{},
   ))
   ```

2. **Verify middleware extracts context:**
   ```go
   // In TracingMiddleware
   ctx := otel.GetTextMapPropagator().Extract(c.UserContext(), &fiberCarrier{c: c})
   ```

---

### Issue: High Memory Usage

**Symptoms:**
- Memory grows over time
- OOM errors

**Diagnosis:**
```promql
# Check Go memory metrics
go_memstats_alloc_bytes
```

**Solutions:**
1. **Reduce queue size:**
   ```go
   sdktrace.WithMaxQueueSize(1024)  // Was 2048
   ```

2. **Reduce sampling:**
   ```bash
   KONSUL_TRACING_SAMPLING_RATIO=0.1  # Was 1.0
   ```

3. **Reduce batch timeout:**
   ```go
   sdktrace.WithBatchTimeout(2*time.Second)  // Was 5s
   ```

---

### Issue: Spans Missing Attributes

**Symptoms:**
- Some attributes not showing in traces
- Empty attribute values

**Diagnosis:**
```bash
# Check trace in Tempo/Jaeger
# Look for missing attributes
```

**Solutions:**
1. **Verify middleware setup:**
   ```go
   // Ensure TracingMiddleware is registered
   app.Use(middleware.TracingMiddleware("konsul"))
   ```

2. **Check attribute collection:**
   ```go
   // In middleware/tracing.go
   span.SetAttributes(
       semconv.HTTPMethod(c.Method()),
       // ... ensure all attributes are set
   )
   ```

---

## Best Practices

### 1. Use Appropriate Sampling

**Recommendations:**

| Environment | Sampling | Reason |
|-------------|----------|--------|
| Development | 100% | See all requests |
| Staging | 50-100% | Thorough testing |
| Production (low traffic) | 100% | Affordable |
| Production (high traffic) | 1-10% | Reduce overhead |

---

### 2. Correlate with Logs

**Use trace ID in logs:**
```go
// Trace ID is automatically added to context
traceID := c.Locals("trace_id").(string)

log.Info("Processing request",
    logger.String("trace_id", traceID),
    logger.String("method", c.Method()),
)
```

**In Grafana:**
- Click trace ID in logs → Jump to trace
- Click trace → Filter logs by trace ID

---

### 3. Monitor Trace Export

**Key metrics:**
```promql
# Export failures (if exposed)
rate(otlp_exporter_errors_total[5m])

# Queue depth (if exposed)
otlp_exporter_queue_depth
```

---

### 4. Add Custom Spans

**For important operations:**
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func ProcessData(ctx context.Context, data []byte) error {
    tracer := otel.Tracer("konsul")
    ctx, span := tracer.Start(ctx, "ProcessData")
    defer span.End()

    span.SetAttributes(
        attribute.Int("data_size", len(data)),
    )

    // Process data...
    if err != nil {
        span.RecordError(err)
        return err
    }

    return nil
}
```

---

### 5. Set Up Alerts

**Alert on high error rate:**
```yaml
- alert: HighTraceErrorRate
  expr: |
    100 * sum(rate(trace_spans{status="error"}[5m])) /
      sum(rate(trace_spans[5m])) > 5
  for: 5m
  annotations:
    summary: "High trace error rate in Konsul"
```

---

## Implementation Details

### Architecture

**Components:**

1. **TracingMiddleware** (`internal/middleware/tracing.go`)
   - Intercepts HTTP requests
   - Creates server spans
   - Injects trace ID into response

2. **TracingConfig** (`internal/telemetry/tracing.go`)
   - Configuration struct
   - Environment variable mapping

3. **InitTracing** (`internal/telemetry/tracing.go`)
   - Initializes OpenTelemetry SDK
   - Configures OTLP exporter
   - Sets up propagators

4. **fiberCarrier** (`internal/middleware/tracing.go`)
   - Adapts Fiber context to OTEL propagation
   - Implements TextMapCarrier interface

---

### Code Flow

**1. Initialization:**
```go
// In main.go
tracingConfig := telemetry.TracingConfig{
    Enabled:        true,
    Endpoint:       "tempo:4318",
    ServiceName:    "konsul",
    SamplingRatio:  1.0,
}

tracerProvider, err := telemetry.InitTracing(ctx, tracingConfig)
defer tracerProvider.Shutdown(ctx)
```

**2. Middleware registration:**
```go
app.Use(middleware.TracingMiddleware("konsul"))
```

**3. Request handling:**
```go
func TracingMiddleware(serviceName string) fiber.Handler {
    tracer := otel.Tracer(serviceName)

    return func(c *fiber.Ctx) error {
        // Extract context
        ctx := otel.GetTextMapPropagator().Extract(...)

        // Start span
        ctx, span := tracer.Start(ctx, spanName)
        defer span.End()

        // Set attributes
        span.SetAttributes(...)

        // Continue processing
        err := c.Next()

        // Record result
        span.SetStatus(...)

        return err
    }
}
```

---

### Configuration Structure

```go
type TracingConfig struct {
    Enabled        bool
    Endpoint       string  // OTLP endpoint
    ServiceName    string
    ServiceVersion string
    Environment    string
    SamplingRatio  float64  // 0.0 to 1.0
    InsecureConn   bool
}
```

---

## See Also

- [Structured Logging Documentation](logging.md)
- [Metrics Documentation](metrics.md)
- [OpenTelemetry Official Docs](https://opentelemetry.io/docs/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [Grafana Tempo](https://grafana.com/oss/tempo/)
- [Jaeger](https://www.jaegertracing.io/)

---

## Changelog

- **2025-10-14**: Initial comprehensive documentation
- **Version**: 0.1.0
- **Status**: ✅ Production Ready

---

## Future Enhancements

Planned improvements for tracing:

- [ ] Span metrics (RED metrics from traces)
- [ ] Custom instrumentation helpers
- [ ] Automatic database query tracing
- [ ] Service-to-service correlation
- [ ] Trace sampling strategies (error-based, latency-based)
- [ ] Exemplars (link metrics to traces)
- [ ] Baggage propagation for custom context
- [ ] Trace-based alerting
