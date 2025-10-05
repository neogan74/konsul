# Konsul Observability Stack

Complete observability stack for Konsul with metrics, logs, and distributed tracing.

## Stack Components

- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization and dashboarding
- **Loki** - Log aggregation
- **Promtail** - Log shipping to Loki
- **Tempo** - Distributed tracing
- **OpenTelemetry Collector** - Centralized telemetry collection

## Quick Start

### 1. Start the Observability Stack

```bash
# From repository root
docker-compose -f docker-compose.observability.yml up -d
```

This starts:
- **Konsul** on http://localhost:8888
- **Grafana** on http://localhost:3000 (admin/admin)
- **Prometheus** on http://localhost:9090
- **Loki** on http://localhost:3100
- **Tempo** on http://localhost:3200
- **OTLP receivers** on ports 4317 (gRPC) and 4318 (HTTP)

### 2. Access Grafana

1. Open http://localhost:3000
2. Login with `admin` / `admin`
3. Navigate to **Dashboards** → **Konsul** folder
4. Open **Konsul Dashboard**

All datasources (Prometheus, Loki, Tempo) are pre-configured and linked for correlation.

### 3. Test the Stack

```bash
# Generate some traffic
for i in {1..100}; do
  curl http://localhost:8888/health
  curl -X POST http://localhost:8888/kv/test-$i -d "value-$i"
done

# Register services
curl -X POST http://localhost:8888/services \
  -H "Content-Type: application/json" \
  -d '{
    "id": "web-1",
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080,
    "tags": ["production", "v1.0"]
  }'
```

## Configuration

### Environment Variables

Konsul observability is configured via environment variables:

#### Logging
```bash
KONSUL_LOG_LEVEL=info          # debug, info, warn, error
KONSUL_LOG_FORMAT=json         # json or text
```

#### Tracing
```bash
KONSUL_TRACING_ENABLED=true
KONSUL_TRACING_ENDPOINT=otel-collector:4318
KONSUL_TRACING_SERVICE_NAME=konsul
KONSUL_TRACING_SERVICE_VERSION=1.0.0
KONSUL_TRACING_ENVIRONMENT=development
KONSUL_TRACING_SAMPLING_RATIO=1.0   # 0.0 to 1.0 (1.0 = 100%)
KONSUL_TRACING_INSECURE=true
```

#### Example with Tracing Enabled
```bash
docker-compose -f docker-compose.observability.yml up -d
```

The docker-compose file already includes these settings.

## Features

### 📊 Metrics (Prometheus + Grafana)

**Exported Metrics:**
- `konsul_http_requests_total` - Total HTTP requests
- `konsul_http_request_duration_seconds` - Request duration histogram
- `konsul_http_requests_in_flight` - Current in-flight requests
- `konsul_kv_operations_total` - KV store operations
- `konsul_kv_store_size` - Number of keys in KV store
- `konsul_service_operations_total` - Service discovery operations
- `konsul_registered_services_total` - Number of registered services
- `konsul_service_heartbeats_total` - Service heartbeat count
- `konsul_expired_services_total` - Expired services cleanup
- `konsul_rate_limit_requests_total` - Rate limit checks
- `konsul_rate_limit_exceeded_total` - Rate limit violations
- `konsul_rate_limit_active_clients` - Active rate-limited clients
- `konsul_build_info` - Build information
- Standard Go metrics (memory, goroutines, GC)

**Pre-built Dashboard:**
- 19 panels across 7 categories
- Real-time metrics
- Auto-refresh every 30 seconds

### 📝 Logs (Loki + Promtail)

**Features:**
- JSON structured logging
- Automatic label extraction
- Log correlation with traces via trace_id
- Docker container log collection
- Multi-line log support

**Log Fields:**
```json
{
  "level": "info",
  "time": "2024-01-15T10:30:45.123Z",
  "msg": "HTTP request",
  "method": "GET",
  "path": "/kv/mykey",
  "status": 200,
  "duration": "1.234ms",
  "trace_id": "abc123..."
}
```

**Query Examples:**
```logql
# All Konsul logs
{service="konsul"}

# Error logs only
{service="konsul"} |= "level=error"

# Slow requests (>100ms)
{service="konsul"} | json | duration > 100ms

# Logs for specific trace
{service="konsul"} | json | trace_id="abc123..."
```

### 🔍 Distributed Tracing (Tempo + OpenTelemetry)

**Features:**
- Automatic HTTP request tracing
- W3C Trace Context propagation
- Configurable sampling
- Trace-to-logs correlation
- Trace-to-metrics correlation

**Span Attributes:**
- `http.method`
- `http.url`
- `http.route`
- `http.status_code`
- `http.client_ip`
- `http.user_agent`
- `service.name`
- `service.version`
- `deployment.environment`

**Trace Flow:**
```
Client Request
    ↓
[OTLP HTTP Exporter] → [OTEL Collector] → [Tempo]
    ↓                           ↓              ↓
[Fiber Middleware]        [Metrics]      [Storage]
    ↓
[HTTP Handler]
    ↓
Response
```

## Architecture

```
┌─────────────┐
│   Konsul    │
│             │
│  :8888      │
└──────┬──────┘
       │
       ├─── Metrics ───────→ Prometheus :9090 ───→ Grafana :3000
       │
       ├─── Logs ──────────→ Promtail ──→ Loki :3100 ───→ Grafana :3000
       │
       └─── Traces ────────→ OTEL Collector ───→ Tempo :3200 ───→ Grafana :3000
                                  │
                                  └─────→ Prometheus (span metrics)
```

## Customization

### Modify Prometheus Scrape Configs

Edit `monitoring/prometheus/prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'konsul'
    static_configs:
      - targets: ['konsul:8888']
    scrape_interval: 10s  # Adjust as needed
```

### Add Prometheus Alerts

Create `monitoring/prometheus/alerts/konsul.yml`:

```yaml
groups:
  - name: konsul
    rules:
      - alert: KonsulDown
        expr: up{job="konsul"} == 0
        for: 1m
        annotations:
          summary: "Konsul is down"

      - alert: HighErrorRate
        expr: rate(konsul_http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High error rate detected"
```

### Modify Log Retention

Edit `monitoring/loki/loki-config.yml`:

```yaml
limits_config:
  retention_period: 168h  # 7 days (default)
```

### Adjust Trace Sampling

Lower sampling for production to reduce overhead:

```bash
KONSUL_TRACING_SAMPLING_RATIO=0.1  # 10% sampling
```

### Modify Dashboard

```bash
# Edit Jsonnet source
vi monitoring/grafana/dashboards/konsul.jsonnet

# Regenerate JSON
make dashboard

# Restart Grafana to reload
docker-compose -f docker-compose.observability.yml restart grafana
```

## Production Considerations

### Security

1. **Change default passwords:**
   ```yaml
   # In docker-compose.observability.yml
   environment:
     - GF_SECURITY_ADMIN_PASSWORD=<strong-password>
   ```

2. **Enable authentication in Prometheus:**
   Use reverse proxy with auth (nginx, Traefik)

3. **Secure OTLP endpoint:**
   Set `KONSUL_TRACING_INSECURE=false` and configure TLS

### Performance

1. **Reduce sampling in production:**
   ```bash
   KONSUL_TRACING_SAMPLING_RATIO=0.01  # 1% sampling
   ```

2. **Adjust Promtail pipeline:**
   Filter unnecessary logs to reduce Loki storage

3. **Configure Tempo retention:**
   ```yaml
   # monitoring/tempo/tempo.yml
   compactor:
     compaction:
       block_retention: 48h  # Reduce for production
   ```

4. **Set Prometheus retention:**
   ```yaml
   # In docker-compose.observability.yml, Prometheus command
   '--storage.tsdb.retention.time=30d'
   ```

### High Availability

- Use external Prometheus/Loki/Tempo (e.g., Grafana Cloud, Thanos, Cortex)
- Deploy multiple Konsul instances with load balancer
- Use remote storage for Tempo and Loki
- Enable replication in Loki and Tempo

## Troubleshooting

### No Metrics in Grafana

1. Check Prometheus targets: http://localhost:9090/targets
2. Verify Konsul is exposing metrics: `curl http://localhost:8888/metrics`
3. Check Grafana datasource: **Configuration** → **Data Sources** → **Prometheus** → **Test**

### No Logs in Loki

1. Check Promtail is running: `docker ps | grep promtail`
2. Verify Promtail config: `docker logs promtail`
3. Check Loki ingestion: `curl http://localhost:3100/ready`
4. Ensure Konsul is using JSON logging: `KONSUL_LOG_FORMAT=json`

### No Traces in Tempo

1. Verify tracing is enabled: `KONSUL_TRACING_ENABLED=true`
2. Check OTEL collector logs: `docker logs otel-collector`
3. Test OTLP endpoint: `curl http://localhost:4318/v1/traces`
4. Check Tempo ingestion: `curl http://localhost:3200/ready`
5. Generate test traffic and search in Grafana Explore

### Trace ID Not in Logs

1. Ensure JSON logging: `KONSUL_LOG_FORMAT=json`
2. Check tracing middleware is enabled
3. Verify trace context propagation in requests

## Resource Requirements

### Minimum (Development)
- **CPU:** 2 cores
- **Memory:** 4 GB RAM
- **Disk:** 10 GB

### Recommended (Production)
- **CPU:** 4+ cores
- **Memory:** 8+ GB RAM
- **Disk:** 50+ GB (with retention)

## Monitoring Stack Endpoints

| Service | Endpoint | Purpose |
|---------|----------|---------|
| Konsul | http://localhost:8888 | Application |
| Grafana | http://localhost:3000 | Dashboards and visualization |
| Prometheus | http://localhost:9090 | Metrics database and queries |
| Loki | http://localhost:3100 | Log aggregation |
| Tempo | http://localhost:3200 | Trace storage |
| OTLP gRPC | localhost:4317 | Trace ingestion (gRPC) |
| OTLP HTTP | localhost:4318 | Trace ingestion (HTTP) |

## Screenshots

The Grafana dashboard includes:
- **Overview:** Uptime, services, KV keys, request rate
- **HTTP Metrics:** Request rates, latency percentiles, in-flight requests
- **KV Store:** Operation rates, store size
- **Service Discovery:** Service operations, heartbeats, expirations
- **Rate Limiting:** Check rates, violations, active clients
- **System Metrics:** Memory, goroutines, GC

## Integration Examples

### Trace Context Propagation

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func MyHandler(c *fiber.Ctx) error {
    // Get tracer from context
    ctx := c.UserContext()
    tracer := otel.Tracer("konsul")

    // Create child span
    ctx, span := tracer.Start(ctx, "my-operation")
    defer span.End()

    // Use context for downstream operations
    // ...

    return c.JSON(result)
}
```

### Structured Logging with Trace ID

```go
import "github.com/neogan74/konsul/internal/logger"

func Handler(c *fiber.Ctx) error {
    traceID := c.Locals("trace_id").(string)

    logger.Info("Processing request",
        logger.String("trace_id", traceID),
        logger.String("method", c.Method()),
        logger.String("path", c.Path()))

    return c.SendStatus(200)
}
```

## Cleanup

```bash
# Stop all services
docker-compose -f docker-compose.observability.yml down

# Remove volumes (deletes all data)
docker-compose -f docker-compose.observability.yml down -v
```

## Contributing

To add new metrics, logs, or traces:

1. Add instrumentation to Konsul code
2. Update Prometheus scrape config if needed
3. Add relevant panels to Grafana dashboard
4. Regenerate dashboard: `make dashboard`
5. Test with observability stack
6. Update this documentation

## License

Same as Konsul project.
