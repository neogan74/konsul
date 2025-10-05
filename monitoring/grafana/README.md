# Konsul Grafana Dashboard

Comprehensive Grafana dashboard for monitoring Konsul service discovery and KV store.

## Features

The dashboard provides monitoring for:

### Overview
- System uptime
- Total registered services
- KV store key count
- HTTP request rate

### HTTP Metrics
- Request rate by status code
- Request duration percentiles (p50, p95, p99)
- In-flight requests

### KV Store Metrics
- KV operation rates (get, set, delete)
- KV store size over time
- Operation success/failure rates

### Service Discovery Metrics
- Service registration/deregistration rates
- Total registered services
- Service heartbeat rates
- Expired service cleanup

### Rate Limiting
- Rate limit check rates
- Rate limit violations
- Active rate-limited clients (by IP and API key)

### System Metrics
- Memory usage (allocated and system)
- Goroutine count
- Garbage collection duration

## Installation

### Prerequisites

- Grafana 8.0+
- Prometheus data source configured
- Konsul exporting metrics to Prometheus

### Method 1: Import JSON Dashboard

1. Open Grafana UI
2. Navigate to **Dashboards** → **Import**
3. Upload `dashboards/konsul-dashboard.json`
4. Select your Prometheus data source
5. Click **Import**

### Method 2: Use ConfigMap (Kubernetes)

```bash
# Create ConfigMap from dashboard
kubectl create configmap konsul-dashboard \
  --from-file=konsul.json=dashboards/konsul-dashboard.json \
  -n monitoring

# Add label for Grafana sidecar discovery
kubectl label configmap konsul-dashboard \
  grafana_dashboard=1 \
  -n monitoring
```

### Method 3: Provision via Helm

Add to your Grafana Helm values:

```yaml
dashboardProviders:
  dashboardproviders.yaml:
    apiVersion: 1
    providers:
    - name: 'konsul'
      orgId: 1
      folder: 'Konsul'
      type: file
      disableDeletion: false
      editable: true
      options:
        path: /var/lib/grafana/dashboards/konsul

dashboards:
  konsul:
    konsul-main:
      file: dashboards/konsul-dashboard.json
```

## Rebuilding from Jsonnet

The dashboard is generated from Jsonnet for easy maintenance and customization.

### Prerequisites

```bash
# Install jsonnet
brew install jsonnet  # macOS
# or
apt-get install jsonnet  # Ubuntu/Debian

# Clone Grafonnet library (already included in vendor/)
```

### Generate Dashboard

```bash
# From repository root
make dashboard

# Or manually
jsonnet -J vendor \
  monitoring/grafana/dashboards/konsul.jsonnet \
  > monitoring/grafana/dashboards/konsul-dashboard.json
```

### Customize Dashboard

Edit `dashboards/konsul.jsonnet` to:
- Add new panels
- Modify queries
- Change thresholds
- Adjust layouts

Then regenerate:
```bash
make dashboard
```

## Prometheus Configuration

Ensure Prometheus is scraping Konsul metrics:

```yaml
scrape_configs:
  - job_name: 'konsul'
    static_configs:
      - targets: ['konsul:8888']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

Or use ServiceMonitor (with Prometheus Operator):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: konsul
  namespace: konsul
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: konsul
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
```

## Dashboard Panels

### Panel Overview

| Panel | Query | Description |
|-------|-------|-------------|
| Uptime | `time() - process_start_time_seconds` | Service uptime |
| Total Services | `konsul_registered_services_total` | Number of registered services |
| KV Store Keys | `konsul_kv_store_size` | Number of keys in KV store |
| Request Rate | `rate(konsul_http_requests_total[5m])` | HTTP requests per second |
| Request Duration | `histogram_quantile(0.95, ...)` | Request latency percentiles |
| KV Operations | `rate(konsul_kv_operations_total[5m])` | KV operation rate |
| Memory Usage | `go_memstats_alloc_bytes` | Memory allocation |
| Goroutines | `go_goroutines` | Active goroutines |

## Alerts (Optional)

Recommended alerts based on dashboard metrics:

```yaml
groups:
- name: konsul
  rules:
  - alert: KonsulDown
    expr: up{job="konsul"} == 0
    for: 1m
    annotations:
      summary: "Konsul instance is down"

  - alert: HighErrorRate
    expr: rate(konsul_http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High error rate in Konsul"

  - alert: RateLimitExceeded
    expr: rate(konsul_rate_limit_exceeded_total[5m]) > 10
    for: 5m
    annotations:
      summary: "High rate limit violations"

  - alert: HighMemoryUsage
    expr: go_memstats_alloc_bytes > 500000000
    for: 10m
    annotations:
      summary: "Konsul memory usage is high"
```

## Troubleshooting

**No data in panels:**
- Verify Prometheus is scraping Konsul (`/metrics` endpoint)
- Check Prometheus target status
- Verify data source configuration in Grafana

**Missing metrics:**
- Ensure Konsul is running with metrics enabled
- Check Prometheus scrape configuration
- Verify `instance` template variable matches your setup

**Dashboard not loading:**
- Validate JSON syntax: `jq . dashboards/konsul-dashboard.json`
- Check Grafana version compatibility
- Review Grafana logs for errors

## Screenshots

The dashboard includes:
- 📊 19 panels across 7 rows
- 🎯 Auto-refresh every 30 seconds
- 📈 1-hour time window (configurable)
- 🔍 Instance selector for multi-instance deployments

## Contributing

To add new panels:

1. Edit `dashboards/konsul.jsonnet`
2. Add panel using Grafonnet library
3. Regenerate: `make dashboard`
4. Test in Grafana
5. Submit PR

## License

Same as Konsul project.
