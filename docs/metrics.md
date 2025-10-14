# Prometheus Metrics - User Guide

Comprehensive guide for monitoring Konsul with Prometheus.

## Overview

Konsul exposes comprehensive metrics in Prometheus format for monitoring:
- HTTP request rates and latencies
- KV store operations and size
- Service discovery statistics
- Rate limiting effectiveness
- Authentication metrics
- Go runtime metrics

### Quick Start

**Access metrics:**
```bash
curl http://localhost:8500/metrics
```

**Sample output:**
```
# HELP konsul_http_requests_total Total number of HTTP requests
# TYPE konsul_http_requests_total counter
konsul_http_requests_total{method="GET",path="/kv/:key",status="200"} 1523
konsul_http_requests_total{method="POST",path="/services",status="201"} 45

# HELP konsul_http_request_duration_seconds HTTP request latencies in seconds
# TYPE konsul_http_request_duration_seconds histogram
konsul_http_request_duration_seconds_bucket{method="GET",path="/kv/:key",status="200",le="0.005"} 1200
konsul_http_request_duration_seconds_bucket{method="GET",path="/kv/:key",status="200",le="0.01"} 1450
konsul_http_request_duration_seconds_sum{method="GET",path="/kv/:key",status="200"} 2.5
konsul_http_request_duration_seconds_count{method="GET",path="/kv/:key",status="200"} 1523
```

---

## Configuration

### Enable Metrics Endpoint

Metrics endpoint is **enabled by default** at `/metrics`.

**Make it public** (no authentication required):
```bash
KONSUL_AUTH_ENABLED=true \
KONSUL_PUBLIC_PATHS="/health,/health/live,/health/ready,/metrics" \
./konsul
```

**Restrict access** (require authentication):
```bash
KONSUL_AUTH_ENABLED=true \
KONSUL_PUBLIC_PATHS="/health,/health/live,/health/ready" \
./konsul

# Access with token
curl http://localhost:8500/metrics \
  -H "Authorization: Bearer $TOKEN"
```

---

## Prometheus Setup

### Scrape Configuration

**prometheus.yml:**
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'konsul'
    static_configs:
      - targets: ['localhost:8500']
    metrics_path: /metrics
    scrape_interval: 10s
```

**With authentication:**
```yaml
scrape_configs:
  - job_name: 'konsul'
    static_configs:
      - targets: ['localhost:8500']
    metrics_path: /metrics
    bearer_token: 'your-jwt-token-here'
    # Or use bearer_token_file:
    # bearer_token_file: /etc/prometheus/konsul-token
```

**Multiple instances:**
```yaml
scrape_configs:
  - job_name: 'konsul'
    static_configs:
      - targets:
        - 'konsul-1:8500'
        - 'konsul-2:8500'
        - 'konsul-3:8500'
        labels:
          cluster: 'production'
```

---

## Available Metrics

### HTTP Metrics

#### `konsul_http_requests_total`

**Type:** Counter

**Description:** Total number of HTTP requests

**Labels:**
- `method` - HTTP method (GET, POST, DELETE, etc.)
- `path` - Request path pattern
- `status` - HTTP status code

**Example queries:**
```promql
# Total requests per second
rate(konsul_http_requests_total[5m])

# Requests by status code
sum by (status) (rate(konsul_http_requests_total[5m]))

# Error rate (4xx and 5xx)
sum(rate(konsul_http_requests_total{status=~"[45].."}[5m]))

# Success rate percentage
100 * sum(rate(konsul_http_requests_total{status=~"2.."}[5m])) /
  sum(rate(konsul_http_requests_total[5m]))
```

---

#### `konsul_http_request_duration_seconds`

**Type:** Histogram

**Description:** HTTP request latencies in seconds

**Labels:**
- `method` - HTTP method
- `path` - Request path pattern
- `status` - HTTP status code

**Buckets:** Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

**Example queries:**
```promql
# Average latency
rate(konsul_http_request_duration_seconds_sum[5m]) /
  rate(konsul_http_request_duration_seconds_count[5m])

# 95th percentile latency
histogram_quantile(0.95,
  rate(konsul_http_request_duration_seconds_bucket[5m]))

# 99th percentile latency
histogram_quantile(0.99,
  rate(konsul_http_request_duration_seconds_bucket[5m]))

# Requests under 10ms
sum(rate(konsul_http_request_duration_seconds_bucket{le="0.01"}[5m]))
```

---

#### `konsul_http_requests_in_flight`

**Type:** Gauge

**Description:** Number of HTTP requests currently being processed

**Example queries:**
```promql
# Current in-flight requests
konsul_http_requests_in_flight

# Max in-flight requests in last hour
max_over_time(konsul_http_requests_in_flight[1h])
```

---

### KV Store Metrics

#### `konsul_kv_operations_total`

**Type:** Counter

**Description:** Total number of KV store operations

**Labels:**
- `operation` - Operation type (get, set, delete, list)
- `status` - Result status (success, error)

**Example queries:**
```promql
# Operations per second by type
sum by (operation) (rate(konsul_kv_operations_total[5m]))

# Write operations (set, delete)
sum(rate(konsul_kv_operations_total{operation=~"set|delete"}[5m]))

# Error rate
sum(rate(konsul_kv_operations_total{status="error"}[5m]))

# Success rate percentage
100 * sum(rate(konsul_kv_operations_total{status="success"}[5m])) /
  sum(rate(konsul_kv_operations_total[5m]))
```

---

#### `konsul_kv_store_size`

**Type:** Gauge

**Description:** Number of keys in the KV store

**Example queries:**
```promql
# Current KV store size
konsul_kv_store_size

# Growth rate (keys per hour)
delta(konsul_kv_store_size[1h])

# Predict size in 24 hours
predict_linear(konsul_kv_store_size[6h], 24*3600)
```

---

### Service Discovery Metrics

#### `konsul_service_operations_total`

**Type:** Counter

**Description:** Total number of service operations

**Labels:**
- `operation` - Operation type (register, deregister, list, get, heartbeat)
- `status` - Result status (success, error)

**Example queries:**
```promql
# Registrations per second
rate(konsul_service_operations_total{operation="register"}[5m])

# Heartbeats per second
rate(konsul_service_operations_total{operation="heartbeat"}[5m])

# Failed operations
sum by (operation) (rate(konsul_service_operations_total{status="error"}[5m]))
```

---

#### `konsul_registered_services_total`

**Type:** Gauge

**Description:** Number of registered services

**Example queries:**
```promql
# Current registered services
konsul_registered_services_total

# Service churn (registrations - deregistrations)
rate(konsul_service_operations_total{operation="register"}[5m]) -
  rate(konsul_service_operations_total{operation="deregister"}[5m])
```

---

#### `konsul_service_heartbeats_total`

**Type:** Counter

**Description:** Total number of service heartbeats

**Labels:**
- `service` - Service name
- `status` - Heartbeat status (success, error, expired)

**Example queries:**
```promql
# Heartbeats per service
sum by (service) (rate(konsul_service_heartbeats_total[5m]))

# Failed heartbeats
sum(rate(konsul_service_heartbeats_total{status="error"}[5m]))
```

---

#### `konsul_expired_services_total`

**Type:** Counter

**Description:** Total number of expired services cleaned up

**Example queries:**
```promql
# Service expirations per hour
rate(konsul_expired_services_total[1h]) * 3600

# Total expired since start
konsul_expired_services_total
```

---

### Rate Limiting Metrics

#### `konsul_rate_limit_requests_total`

**Type:** Counter

**Description:** Total number of requests checked against rate limits

**Labels:**
- `limiter_type` - Type of limiter (ip, apikey)
- `status` - Check result (allowed, exceeded)

**Example queries:**
```promql
# Rate limit checks per second
rate(konsul_rate_limit_requests_total[5m])

# Checks by limiter type
sum by (limiter_type) (rate(konsul_rate_limit_requests_total[5m]))
```

---

#### `konsul_rate_limit_exceeded_total`

**Type:** Counter

**Description:** Total number of requests that exceeded rate limits

**Labels:**
- `limiter_type` - Type of limiter (ip, apikey)

**Example queries:**
```promql
# Rate limit violations per second
rate(konsul_rate_limit_exceeded_total[5m])

# Violation rate percentage
100 * sum(rate(konsul_rate_limit_exceeded_total[5m])) /
  sum(rate(konsul_rate_limit_requests_total[5m]))

# Most violated limiter type
topk(5, sum by (limiter_type) (rate(konsul_rate_limit_exceeded_total[5m])))
```

---

#### `konsul_rate_limit_active_clients`

**Type:** Gauge

**Description:** Number of active clients being rate limited

**Labels:**
- `limiter_type` - Type of limiter (ip, apikey)

**Example queries:**
```promql
# Active clients by type
sum by (limiter_type) (konsul_rate_limit_active_clients)

# Total active rate limited clients
sum(konsul_rate_limit_active_clients)
```

---

### System Metrics

#### `konsul_build_info`

**Type:** Gauge (value always 1)

**Description:** Build information about Konsul

**Labels:**
- `version` - Konsul version
- `go_version` - Go compiler version

**Example queries:**
```promql
# Current version
konsul_build_info

# Version by instance
konsul_build_info{version="0.1.0"}
```

---

### Go Runtime Metrics

Provided automatically by Prometheus Go client:

- `go_goroutines` - Number of goroutines
- `go_threads` - Number of OS threads
- `go_memstats_alloc_bytes` - Bytes allocated and in use
- `go_memstats_sys_bytes` - Bytes obtained from system
- `go_memstats_heap_alloc_bytes` - Heap bytes allocated
- `go_gc_duration_seconds` - GC invocation durations

**Example queries:**
```promql
# Memory usage
go_memstats_alloc_bytes

# Goroutine count
go_goroutines

# GC pause time (95th percentile)
histogram_quantile(0.95, rate(go_gc_duration_seconds[5m]))
```

---

## Grafana Dashboard

### Import Dashboard

1. Open Grafana → Dashboards → Import
2. Use Dashboard ID or upload JSON
3. Select Prometheus data source
4. Click Import

### Key Panels

**Request Rate:**
```promql
sum(rate(konsul_http_requests_total[5m]))
```

**Request Latency (p95):**
```promql
histogram_quantile(0.95,
  rate(konsul_http_request_duration_seconds_bucket[5m]))
```

**Error Rate:**
```promql
100 * sum(rate(konsul_http_requests_total{status=~"[45].."}[5m])) /
  sum(rate(konsul_http_requests_total[5m]))
```

**KV Store Size:**
```promql
konsul_kv_store_size
```

**Active Services:**
```promql
konsul_registered_services_total
```

**Rate Limit Violations:**
```promql
sum(rate(konsul_rate_limit_exceeded_total[5m]))
```

---

## Alerting Rules

### Critical Alerts

**File:** `alerts/konsul.yml`

```yaml
groups:
  - name: konsul
    interval: 30s
    rules:
      # High error rate
      - alert: KonsulHighErrorRate
        expr: |
          100 * sum(rate(konsul_http_requests_total{status=~"5.."}[5m])) /
            sum(rate(konsul_http_requests_total[5m])) > 5
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Konsul high error rate (>5%)"
          description: "{{ $value | humanizePercentage }} error rate"

      # High latency
      - alert: KonsulHighLatency
        expr: |
          histogram_quantile(0.95,
            rate(konsul_http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Konsul high latency (p95 > 1s)"
          description: "p95 latency: {{ $value | humanizeDuration }}"

      # Service expirations
      - alert: KonsulHighServiceExpiration
        expr: rate(konsul_expired_services_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High service expiration rate"
          description: "{{ $value }} services expiring per second"

      # KV store growing fast
      - alert: KonsulKVStoreGrowth
        expr: |
          predict_linear(konsul_kv_store_size[1h], 24*3600) > 1000000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "KV store predicted to exceed 1M keys in 24h"

      # Rate limiting active
      - alert: KonsulHighRateLimitViolations
        expr: |
          100 * sum(rate(konsul_rate_limit_exceeded_total[5m])) /
            sum(rate(konsul_rate_limit_requests_total[5m])) > 10
        for: 5m
        labels:
          severity: info
        annotations:
          summary: "High rate limit violation rate (>10%)"

      # Memory usage
      - alert: KonsulHighMemoryUsage
        expr: go_memstats_alloc_bytes > 1073741824  # 1GB
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Konsul high memory usage (>1GB)"
          description: "Memory: {{ $value | humanize }}B"

      # Goroutine leak
      - alert: KonsulGoroutineLeak
        expr: go_goroutines > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Possible goroutine leak (>1000)"
          description: "Goroutines: {{ $value }}"
```

---

## Best Practices

### 1. Scrape Interval

**Recommended:** 10-30 seconds

```yaml
scrape_interval: 15s
```

**Trade-offs:**
- Shorter (5s): More data points, higher overhead
- Longer (60s): Less overhead, coarser granularity

---

### 2. Retention Period

**Recommended:** 15 days for detailed metrics

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

# Command line
prometheus --storage.tsdb.retention.time=15d
```

---

### 3. Label Cardinality

**Avoid high-cardinality labels:**
- ❌ user_id
- ❌ api_key
- ❌ specific_key_name
- ✅ method
- ✅ status
- ✅ operation

**Why:** Each unique label combination creates a new time series, increasing memory usage.

---

### 4. Histogram Buckets

**Default buckets are appropriate** for most use cases:
```
[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
```

**Custom buckets** for specific needs (modify in code):
```go
prometheus.HistogramOpts{
    Name: "konsul_http_request_duration_seconds",
    Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
}
```

---

### 5. Recording Rules

Pre-compute expensive queries:

```yaml
groups:
  - name: konsul_rules
    interval: 30s
    rules:
      # Pre-compute request rate
      - record: konsul:http_requests:rate5m
        expr: rate(konsul_http_requests_total[5m])

      # Pre-compute p95 latency
      - record: konsul:http_latency:p95
        expr: |
          histogram_quantile(0.95,
            rate(konsul_http_request_duration_seconds_bucket[5m]))

      # Pre-compute error rate
      - record: konsul:http_error_rate:rate5m
        expr: |
          sum(rate(konsul_http_requests_total{status=~"[45].."}[5m])) /
            sum(rate(konsul_http_requests_total[5m]))
```

---

## Troubleshooting

### Issue: No metrics appearing

**Check endpoint:**
```bash
curl http://localhost:8500/metrics
```

**Check Prometheus targets:**
1. Open Prometheus UI → Status → Targets
2. Verify Konsul target is "UP"
3. Check "Last Scrape" timestamp

**Common causes:**
- Prometheus can't reach Konsul (network/firewall)
- Wrong metrics_path configuration
- Authentication required but not configured

---

### Issue: Missing specific metrics

**Verify metric exists:**
```bash
curl http://localhost:8500/metrics | grep konsul_kv_operations_total
```

**Possible causes:**
- Metric not emitted yet (no operations performed)
- Metric name typo in query
- Metric removed in newer version

---

### Issue: High cardinality warning

**Prometheus logs show:**
```
level=warn component=tsdb msg="Too many samples"
```

**Diagnosis:**
```promql
# Check series count
count({__name__=~"konsul_.*"})

# Most common metrics
topk(10, count by (__name__)({__name__=~"konsul_.*"}))

# High cardinality labels
count by (path) (konsul_http_requests_total)
```

**Solution:** Remove or aggregate high-cardinality labels

---

## See Also

- [Metrics API Reference](metrics-api.md)
- [ADR-0004](adr/0004-prometheus-metrics.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)
