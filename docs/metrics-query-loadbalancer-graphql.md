# Prometheus Metrics: Service Queries, Load Balancing, and GraphQL

This document describes comprehensive Prometheus metrics for service discovery query operations, load balancing, and GraphQL API in Konsul.

## Table of Contents

- [Overview](#overview)
- [Service Query Metrics (HTTP)](#service-query-metrics-http)
- [Load Balancer Metrics](#load-balancer-metrics)
- [GraphQL Metrics](#graphql-metrics)
- [Example Queries](#example-queries)
- [Grafana Dashboard](#grafana-dashboard)
- [Alerting Rules](#alerting-rules)

## Overview

All metrics are exposed at the `/metrics` endpoint in Prometheus format. These metrics provide comprehensive observability into:

- Service query performance (tags/metadata via HTTP and GraphQL)
- Load balancer behavior and distribution
- GraphQL query patterns and performance
- Service registration patterns

**Total New Metrics**: 17 metrics across 3 categories

---

## Service Query Metrics (HTTP)

### konsul_service_query_total

**Type**: Counter
**Labels**: `query_type`, `status`

Total number of HTTP service queries by type.

**Query Types**:
- `tags` - Query by tags only
- `metadata` - Query by metadata only
- `combined` - Query by both tags and metadata

**Status Values**:
- `success` - Query completed successfully
- `error` - Query failed (invalid parameters)

**Example**:
```promql
# HTTP query rate by type
rate(konsul_service_query_total{status="success"}[5m])

# HTTP query error rate
rate(konsul_service_query_total{status="error"}[5m]) /
rate(konsul_service_query_total[5m]) * 100
```

---

### konsul_service_query_duration_seconds

**Type**: Histogram
**Labels**: `query_type`
**Buckets**: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0]`

HTTP service query latencies in seconds.

**Example**:
```promql
# p95 HTTP query latency by type
histogram_quantile(0.95,
  rate(konsul_service_query_duration_seconds_bucket[5m])
)
```

---

### konsul_service_query_results_count

**Type**: Histogram
**Labels**: `query_type`
**Buckets**: `[0, 1, 5, 10, 25, 50, 100, 250, 500]`

Number of services returned by HTTP queries.

---

### konsul_service_tags_per_service

**Type**: Histogram
**Labels**: None
**Buckets**: `[0, 1, 2, 5, 10, 20, 30, 50, 64]`

Distribution of tags per registered service.

---

### konsul_service_metadata_keys_per_service

**Type**: Histogram
**Labels**: None
**Buckets**: `[0, 1, 2, 5, 10, 20, 30, 50, 64]`

Distribution of metadata keys per registered service.

---

## Load Balancer Metrics

### konsul_load_balancer_selections_total

**Type**: Counter
**Labels**: `strategy`, `selection_type`, `status`

Total number of load balancer service selections.

**Strategies**: `round-robin`, `random`, `least-connections`
**Selection Types**: `service`, `tags`, `metadata`, `combined`
**Status**: `success`, `not_found`, `error`

**Example**:
```promql
# Selection rate by strategy
sum by (strategy) (
  rate(konsul_load_balancer_selections_total{status="success"}[5m])
)
```

---

### konsul_load_balancer_selection_duration_seconds

**Type**: Histogram
**Labels**: `strategy`, `selection_type`
**Buckets**: `[0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05]`

Load balancer selection latencies in seconds.

---

### konsul_load_balancer_active_connections

**Type**: Gauge
**Labels**: `service_name`, `instance`

Active connections per instance (for least-connections strategy).

**Example**:
```promql
# Connection imbalance
stddev(konsul_load_balancer_active_connections) /
avg(konsul_load_balancer_active_connections)
```

---

### konsul_load_balancer_strategy_changes_total

**Type**: Counter
**Labels**: `from_strategy`, `to_strategy`

Total strategy changes.

---

### konsul_load_balancer_current_strategy

**Type**: Gauge
**Labels**: `strategy`

Current strategy (1=active, 0=inactive).

---

### konsul_load_balancer_instance_pool_size

**Type**: Histogram
**Labels**: `selection_type`
**Buckets**: `[0, 1, 2, 5, 10, 20, 50, 100]`

Number of available instances in the pool.

---

## GraphQL Metrics

### konsul_graphql_queries_total

**Type**: Counter
**Labels**: `query_name`, `status`

Total number of GraphQL queries executed.

**Query Names**:
- `servicesByTags` - Query services by tags
- `servicesByMetadata` - Query services by metadata
- `servicesByQuery` - Combined tags + metadata query

**Status Values**:
- `success` - Query completed successfully
- `error` - Query failed

**Example**:
```promql
# GraphQL query rate by type
rate(konsul_graphql_queries_total{status="success"}[5m])

# Most popular GraphQL queries
topk(5, sum by (query_name) (
  rate(konsul_graphql_queries_total[5m])
))
```

---

### konsul_graphql_query_duration_seconds

**Type**: Histogram
**Labels**: `query_name`
**Buckets**: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]`

GraphQL query execution latencies in seconds.

**Example**:
```promql
# p99 GraphQL query latency
histogram_quantile(0.99,
  rate(konsul_graphql_query_duration_seconds_bucket[5m])
)

# Compare GraphQL vs HTTP query performance
histogram_quantile(0.95,
  rate(konsul_graphql_query_duration_seconds_bucket{query_name="servicesByTags"}[5m])
) vs
histogram_quantile(0.95,
  rate(konsul_service_query_duration_seconds_bucket{query_type="tags"}[5m])
)
```

---

### konsul_graphql_query_results_count

**Type**: Histogram
**Labels**: `query_name`
**Buckets**: `[0, 1, 5, 10, 25, 50, 100, 250, 500, 1000]`

Number of results returned by GraphQL queries.

**Example**:
```promql
# Average results per GraphQL query
rate(konsul_graphql_query_results_count_sum[5m]) /
rate(konsul_graphql_query_results_count_count[5m])
```

---

### konsul_graphql_resolver_duration_seconds

**Type**: Histogram
**Labels**: `resolver`
**Buckets**: `[0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1]`

GraphQL resolver execution latencies in seconds.

**Example**:
```promql
# Slowest resolvers
topk(5, avg by (resolver) (
  rate(konsul_graphql_resolver_duration_seconds_sum[5m]) /
  rate(konsul_graphql_resolver_duration_seconds_count[5m])
))
```

---

### konsul_graphql_errors_total

**Type**: Counter
**Labels**: `query_name`, `error_type`

Total number of GraphQL errors.

**Example**:
```promql
# GraphQL error rate
rate(konsul_graphql_errors_total[5m])

# Error rate by query type
sum by (query_name) (
  rate(konsul_graphql_errors_total[5m])
)
```

---

### konsul_graphql_query_complexity

**Type**: Histogram
**Labels**: `query_name`
**Buckets**: `[1, 5, 10, 25, 50, 100, 250, 500]`

Complexity score of GraphQL queries (based on number of filters/tags).

**Example**:
```promql
# Average query complexity
rate(konsul_graphql_query_complexity_sum[5m]) /
rate(konsul_graphql_query_complexity_count[5m])

# Complex queries (>50 filters/tags)
rate(konsul_graphql_query_complexity_bucket{le="+Inf"}[5m]) -
rate(konsul_graphql_query_complexity_bucket{le="50"}[5m])
```

---

## Example Queries

### Performance Comparison

```promql
# Compare HTTP vs GraphQL query latency
# HTTP tags query
histogram_quantile(0.95,
  rate(konsul_service_query_duration_seconds_bucket{query_type="tags"}[5m])
)
vs
# GraphQL tags query
histogram_quantile(0.95,
  rate(konsul_graphql_query_duration_seconds_bucket{query_name="servicesByTags"}[5m])
)
```

### Load Balancer Health

```promql
# Load balancer success rate
sum(rate(konsul_load_balancer_selections_total{status="success"}[5m])) /
sum(rate(konsul_load_balancer_selections_total[5m])) * 100

# Most effective strategy (by latency)
min by (strategy) (
  histogram_quantile(0.95,
    rate(konsul_load_balancer_selection_duration_seconds_bucket[5m])
  )
)
```

### GraphQL Insights

```promql
# GraphQL vs HTTP API usage
sum(rate(konsul_graphql_queries_total[5m]))
vs
sum(rate(konsul_service_query_total[5m]))

# GraphQL query complexity trends
histogram_quantile(0.90,
  rate(konsul_graphql_query_complexity_bucket[5m])
)
```

---

## Grafana Dashboard

### Recommended Dashboard Layout

**Row 1: Service Queries (HTTP)**
- Query Rate by Type (Graph)
- Query Latency p95 (Graph)
- Query Success Rate (Stat)
- Results Distribution (Heatmap)

**Row 2: Load Balancer**
- Selection Rate by Strategy (Graph)
- Selection Latency (Graph)
- Current Strategy (Stat)
- Connection Distribution (Graph)

**Row 3: GraphQL**
- GraphQL Query Rate (Graph)
- GraphQL Latency vs HTTP (Graph)
- Query Complexity (Histogram)
- Error Rate (Stat)

**Row 4: Service Patterns**
- Tags per Service (Histogram)
- Metadata per Service (Histogram)
- Instance Pool Sizes (Graph)

### Example Grafana Panel

```json
{
  "title": "HTTP vs GraphQL Query Latency Comparison",
  "targets": [
    {
      "expr": "histogram_quantile(0.95, rate(konsul_service_query_duration_seconds_bucket{query_type=\"tags\"}[5m]))",
      "legendFormat": "HTTP Tags p95"
    },
    {
      "expr": "histogram_quantile(0.95, rate(konsul_graphql_query_duration_seconds_bucket{query_name=\"servicesByTags\"}[5m]))",
      "legendFormat": "GraphQL Tags p95"
    }
  ],
  "yaxes": [{
    "format": "s",
    "label": "Latency"
  }]
}
```

---

## Alerting Rules

```yaml
groups:
  - name: konsul_service_queries
    rules:
      # HTTP query errors
      - alert: HighHTTPQueryErrorRate
        expr: |
          (sum(rate(konsul_service_query_total{status="error"}[5m])) /
           sum(rate(konsul_service_query_total[5m]))) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High HTTP query error rate: {{ $value | humanizePercentage }}"

      # Slow HTTP queries
      - alert: SlowHTTPQueries
        expr: |
          histogram_quantile(0.95,
            rate(konsul_service_query_duration_seconds_bucket[5m])
          ) > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "p95 HTTP query latency: {{ $value }}s"

  - name: konsul_load_balancer
    rules:
      # Load balancer failures
      - alert: HighLoadBalancerFailureRate
        expr: |
          (sum(rate(konsul_load_balancer_selections_total{status="not_found"}[5m])) /
           sum(rate(konsul_load_balancer_selections_total[5m]))) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "{{ $value | humanizePercentage }} of selections failing"

      # Connection imbalance
      - alert: LoadBalancerImbalance
        expr: |
          (stddev(konsul_load_balancer_active_connections) /
           avg(konsul_load_balancer_active_connections)) > 2
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High connection imbalance"

  - name: konsul_graphql
    rules:
      # GraphQL errors
      - alert: HighGraphQLErrorRate
        expr: |
          sum(rate(konsul_graphql_errors_total[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "GraphQL error rate: {{ $value }}/s"

      # Slow GraphQL queries
      - alert: SlowGraphQLQueries
        expr: |
          histogram_quantile(0.95,
            rate(konsul_graphql_query_duration_seconds_bucket[5m])
          ) > 1.0
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "p95 GraphQL query latency: {{ $value }}s"

      # Complex GraphQL queries
      - alert: HighGraphQLComplexity
        expr: |
          histogram_quantile(0.99,
            rate(konsul_graphql_query_complexity_bucket[5m])
          ) > 100
        for: 15m
        labels:
          severity: info
        annotations:
          summary: "Very complex GraphQL queries detected"
```

---

## Metrics Summary Table

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `konsul_service_query_total` | Counter | query_type, status | HTTP query operations |
| `konsul_service_query_duration_seconds` | Histogram | query_type | HTTP query latency |
| `konsul_service_query_results_count` | Histogram | query_type | HTTP results per query |
| `konsul_service_tags_per_service` | Histogram | - | Tags usage pattern |
| `konsul_service_metadata_keys_per_service` | Histogram | - | Metadata usage pattern |
| `konsul_load_balancer_selections_total` | Counter | strategy, selection_type, status | LB operations |
| `konsul_load_balancer_selection_duration_seconds` | Histogram | strategy, selection_type | LB latency |
| `konsul_load_balancer_active_connections` | Gauge | service_name, instance | Active connections |
| `konsul_load_balancer_strategy_changes_total` | Counter | from_strategy, to_strategy | Strategy changes |
| `konsul_load_balancer_current_strategy` | Gauge | strategy | Current LB strategy |
| `konsul_load_balancer_instance_pool_size` | Histogram | selection_type | Available instances |
| `konsul_graphql_queries_total` | Counter | query_name, status | GraphQL operations |
| `konsul_graphql_query_duration_seconds` | Histogram | query_name | GraphQL query latency |
| `konsul_graphql_query_results_count` | Histogram | query_name | GraphQL results per query |
| `konsul_graphql_resolver_duration_seconds` | Histogram | resolver | Resolver latency |
| `konsul_graphql_errors_total` | Counter | query_name, error_type | GraphQL errors |
| `konsul_graphql_query_complexity` | Histogram | query_name | Query complexity score |

---

## Best Practices

1. **Monitor both HTTP and GraphQL APIs** to understand usage patterns
2. **Track query complexity** to prevent expensive queries
3. **Alert on load balancer failures** to ensure high availability
4. **Compare HTTP vs GraphQL performance** to optimize client usage
5. **Monitor service registration patterns** (tags/metadata) for optimization

---

For more information, see:
- [Main Metrics Documentation](./METRICS.md)
- [Tags/Metadata API Reference](./api-tags-metadata-loadbalancing.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
