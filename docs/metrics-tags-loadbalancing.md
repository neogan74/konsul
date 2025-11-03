# Prometheus Metrics: Tags, Metadata, and Load Balancing

This document describes the Prometheus metrics exposed by Konsul for service query operations (tags/metadata) and load balancing functionality.

## Table of Contents

- [Overview](#overview)
- [Service Query Metrics](#service-query-metrics)
- [Load Balancer Metrics](#load-balancer-metrics)
- [Example Queries](#example-queries)
- [Grafana Dashboard](#grafana-dashboard)
- [Alerting Rules](#alerting-rules)

## Overview

All metrics are exposed at the `/metrics` endpoint in Prometheus format. These metrics provide comprehensive observability into:

- Service query performance and usage patterns
- Load balancer behavior and distribution
- Service registration patterns (tags/metadata usage)
- System health and performance

## Service Query Metrics

### konsul_service_query_total

**Type**: Counter
**Labels**: `query_type`, `status`

Total number of service queries by type.

**Query Types**:
- `tags` - Query by tags only
- `metadata` - Query by metadata only
- `combined` - Query by both tags and metadata

**Status Values**:
- `success` - Query completed successfully
- `error` - Query failed (invalid parameters, etc.)

**Example**:
```promql
# Rate of tag queries per second
rate(konsul_service_query_total{query_type="tags",status="success"}[5m])

# Total combined queries in last hour
increase(konsul_service_query_total{query_type="combined"}[1h])
```

---

### konsul_service_query_duration_seconds

**Type**: Histogram
**Labels**: `query_type`
**Buckets**: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0]`

Latency of service queries in seconds.

**Example**:
```promql
# 95th percentile query latency by type
histogram_quantile(0.95,
  rate(konsul_service_query_duration_seconds_bucket[5m])
)

# Average query duration for metadata queries
rate(konsul_service_query_duration_seconds_sum{query_type="metadata"}[5m]) /
rate(konsul_service_query_duration_seconds_count{query_type="metadata"}[5m])
```

---

### konsul_service_query_results_count

**Type**: Histogram
**Labels**: `query_type`
**Buckets**: `[0, 1, 5, 10, 25, 50, 100, 250, 500]`

Number of services returned by queries.

**Example**:
```promql
# Average number of results per query type
rate(konsul_service_query_results_count_sum[5m]) /
rate(konsul_service_query_results_count_count[5m])

# Percentage of queries returning zero results
rate(konsul_service_query_results_count_bucket{le="0"}[5m]) /
rate(konsul_service_query_results_count_count[5m]) * 100
```

---

### konsul_service_tags_per_service

**Type**: Histogram
**Labels**: None
**Buckets**: `[0, 1, 2, 5, 10, 20, 30, 50, 64]`

Distribution of the number of tags per registered service.

**Example**:
```promql
# 50th percentile (median) tags per service
histogram_quantile(0.50,
  rate(konsul_service_tags_per_service_bucket[5m])
)

# Services with more than 10 tags
rate(konsul_service_tags_per_service_bucket{le="64"}[5m]) -
rate(konsul_service_tags_per_service_bucket{le="10"}[5m])
```

---

### konsul_service_metadata_keys_per_service

**Type**: Histogram
**Labels**: None
**Buckets**: `[0, 1, 2, 5, 10, 20, 30, 50, 64]`

Distribution of the number of metadata keys per registered service.

**Example**:
```promql
# Average metadata keys per service
rate(konsul_service_metadata_keys_per_service_sum[5m]) /
rate(konsul_service_metadata_keys_per_service_count[5m])
```

---

## Load Balancer Metrics

### konsul_load_balancer_selections_total

**Type**: Counter
**Labels**: `strategy`, `selection_type`, `status`

Total number of load balancer service selections.

**Strategies**:
- `round-robin` - Even distribution
- `random` - Random selection
- `least-connections` - Fewest active connections

**Selection Types**:
- `service` - Selection by service tag
- `tags` - Selection by tags
- `metadata` - Selection by metadata
- `combined` - Selection by tags + metadata

**Status Values**:
- `success` - Instance selected successfully
- `not_found` - No matching instances
- `error` - Invalid request

**Example**:
```promql
# Selection rate by strategy
rate(konsul_load_balancer_selections_total{status="success"}[5m])

# Error rate for load balancer
rate(konsul_load_balancer_selections_total{status="not_found"}[5m]) +
rate(konsul_load_balancer_selections_total{status="error"}[5m])
```

---

### konsul_load_balancer_selection_duration_seconds

**Type**: Histogram
**Labels**: `strategy`, `selection_type`
**Buckets**: `[0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05]`

Load balancer selection latencies in seconds.

**Example**:
```promql
# 99th percentile selection time by strategy
histogram_quantile(0.99,
  rate(konsul_load_balancer_selection_duration_seconds_bucket[5m])
)

# Compare latency across strategies
avg by (strategy) (
  rate(konsul_load_balancer_selection_duration_seconds_sum[5m]) /
  rate(konsul_load_balancer_selection_duration_seconds_count[5m])
)
```

---

### konsul_load_balancer_active_connections

**Type**: Gauge
**Labels**: `service_name`, `instance`

Number of active connections per service instance (for least-connections strategy).

**Example**:
```promql
# Total active connections across all instances
sum(konsul_load_balancer_active_connections)

# Instances with highest connection count
topk(5, konsul_load_balancer_active_connections)

# Connection imbalance (std deviation)
stddev(konsul_load_balancer_active_connections)
```

---

### konsul_load_balancer_strategy_changes_total

**Type**: Counter
**Labels**: `from_strategy`, `to_strategy`

Total number of load balancing strategy changes.

**Example**:
```promql
# Strategy change rate
rate(konsul_load_balancer_strategy_changes_total[1h])

# Most common strategy transitions
topk(3, sum by (from_strategy, to_strategy) (
  konsul_load_balancer_strategy_changes_total
))
```

---

### konsul_load_balancer_current_strategy

**Type**: Gauge
**Labels**: `strategy`

Current load balancing strategy (1 for active, 0 for others).

**Example**:
```promql
# Show current active strategy
konsul_load_balancer_current_strategy == 1

# Alert if strategy changed unexpectedly
changes(konsul_load_balancer_current_strategy[5m]) > 0
```

---

### konsul_load_balancer_instance_pool_size

**Type**: Histogram
**Labels**: `selection_type`
**Buckets**: `[0, 1, 2, 5, 10, 20, 50, 100]`

Number of available instances in the load balancer pool.

**Example**:
```promql
# Average pool size by selection type
avg by (selection_type) (
  rate(konsul_load_balancer_instance_pool_size_sum[5m]) /
  rate(konsul_load_balancer_instance_pool_size_count[5m])
)

# Selections with zero instances available
rate(konsul_load_balancer_instance_pool_size_bucket{le="0"}[5m])
```

---

## Example Queries

### Service Query Performance

```promql
# Query success rate (%)
sum(rate(konsul_service_query_total{status="success"}[5m])) /
sum(rate(konsul_service_query_total[5m])) * 100

# Slow queries (> 100ms)
histogram_quantile(0.95,
  rate(konsul_service_query_duration_seconds_bucket[5m])
) > 0.1

# Most popular query type
topk(1, sum by (query_type) (
  rate(konsul_service_query_total[5m])
))
```

### Load Balancer Health

```promql
# Load balancer success rate
sum(rate(konsul_load_balancer_selections_total{status="success"}[5m])) /
sum(rate(konsul_load_balancer_selections_total[5m])) * 100

# Selection latency by strategy (avg)
avg by (strategy) (
  rate(konsul_load_balancer_selection_duration_seconds_sum[5m]) /
  rate(konsul_load_balancer_selection_duration_seconds_count[5m])
)

# Instances with no matching services
rate(konsul_load_balancer_selections_total{status="not_found"}[5m])
```

### Service Registration Patterns

```promql
# Services with many tags (>10)
histogram_quantile(0.90,
  rate(konsul_service_tags_per_service_bucket[5m])
)

# Services with metadata
1 - (
  rate(konsul_service_metadata_keys_per_service_bucket{le="0"}[5m]) /
  rate(konsul_service_metadata_keys_per_service_count[5m])
)
```

### Load Distribution

```promql
# Connection distribution across instances
stddev(konsul_load_balancer_active_connections) /
avg(konsul_load_balancer_active_connections)

# Instances handling most connections
topk(5, konsul_load_balancer_active_connections)
```

---

## Grafana Dashboard

### Recommended Panels

**Row 1: Service Queries**
- Query Rate by Type (Graph)
- Query Latency p95 (Graph)
- Query Results Distribution (Heatmap)
- Query Success Rate (Stat)

**Row 2: Load Balancer**
- Selection Rate by Strategy (Graph)
- Selection Latency by Strategy (Graph)
- Current Strategy (Stat)
- Instance Pool Sizes (Graph)

**Row 3: Service Patterns**
- Tags per Service (Histogram)
- Metadata Keys per Service (Histogram)
- Active Connections (Graph)
- Strategy Changes (Table)

### Example Panel JSON

```json
{
  "title": "Query Latency by Type",
  "targets": [{
    "expr": "histogram_quantile(0.95, rate(konsul_service_query_duration_seconds_bucket[5m]))",
    "legendFormat": "{{ query_type }} p95"
  }],
  "yaxes": [{
    "format": "s",
    "label": "Duration"
  }]
}
```

---

## Alerting Rules

### Critical Alerts

```yaml
groups:
  - name: konsul_service_query
    rules:
      # High query error rate
      - alert: HighServiceQueryErrorRate
        expr: |
          (sum(rate(konsul_service_query_total{status="error"}[5m])) /
           sum(rate(konsul_service_query_total[5m]))) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High service query error rate ({{ $value }}%)"

      # Slow queries
      - alert: SlowServiceQueries
        expr: |
          histogram_quantile(0.95,
            rate(konsul_service_query_duration_seconds_bucket[5m])
          ) > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "p95 query latency is {{ $value }}s"

  - name: konsul_load_balancer
    rules:
      # High load balancer failure rate
      - alert: HighLoadBalancerFailureRate
        expr: |
          (sum(rate(konsul_load_balancer_selections_total{status="not_found"}[5m])) /
           sum(rate(konsul_load_balancer_selections_total[5m]))) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "{{ $value }}% of load balancer selections failing"

      # No instances available
      - alert: NoInstancesAvailable
        expr: |
          rate(konsul_load_balancer_instance_pool_size_bucket{le="0"}[5m]) > 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Load balancer finding zero instances"

      # Connection imbalance
      - alert: LoadBalancerImbalance
        expr: |
          (stddev(konsul_load_balancer_active_connections) /
           avg(konsul_load_balancer_active_connections)) > 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High connection imbalance across instances"
```

---

## Best Practices

### 1. **Monitor Query Performance**
```promql
# Set up alerts for slow queries
histogram_quantile(0.99,
  rate(konsul_service_query_duration_seconds_bucket[5m])
) > 0.2
```

### 2. **Track Load Balancer Efficiency**
```promql
# Monitor selection success rate
sum(rate(konsul_load_balancer_selections_total{status="success"}[5m])) /
sum(rate(konsul_load_balancer_selections_total[5m]))
```

### 3. **Optimize Tag/Metadata Usage**
```promql
# Identify services with excessive tags
histogram_quantile(0.99,
  rate(konsul_service_tags_per_service_bucket[5m])
) > 30
```

### 4. **Monitor Strategy Effectiveness**
Compare strategies by latency and success rate:
```promql
avg by (strategy) (
  rate(konsul_load_balancer_selection_duration_seconds_sum[5m]) /
  rate(konsul_load_balancer_selection_duration_seconds_count[5m])
)
```

---

## Integration with Existing Metrics

These metrics complement existing Konsul metrics:

- `konsul_service_operations_total` - General service operations
- `konsul_registered_services_total` - Total active services
- `konsul_http_requests_total` - HTTP request metrics

**Example Combined Query**:
```promql
# Correlation between service count and query latency
(
  konsul_registered_services_total
) /
(
  histogram_quantile(0.95,
    rate(konsul_service_query_duration_seconds_bucket[5m])
  )
)
```

---

## Summary

The tags/metadata and load balancing metrics provide:

- ✅ **Query Performance Monitoring** - Track latency, throughput, and success rates
- ✅ **Load Balancer Observability** - Monitor distribution, strategy, and health
- ✅ **Service Pattern Analysis** - Understand tag/metadata usage patterns
- ✅ **Operational Insights** - Identify bottlenecks and optimize configuration

For more information, see:
- [Main Metrics Documentation](./METRICS.md)
- [API Reference](./api-tags-metadata-loadbalancing.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
