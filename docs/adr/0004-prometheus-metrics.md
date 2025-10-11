# ADR-0004: Prometheus for Metrics

**Date**: 2024-09-22

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: observability, monitoring, metrics, operations

## Context

Konsul requires comprehensive monitoring and metrics collection for:
- Request rates and latencies
- Service health and availability
- KV store operations
- Resource utilization
- Rate limiting effectiveness
- Authentication success/failure rates

Requirements:
- Industry-standard metrics format
- Minimal performance overhead
- Easy integration with monitoring stacks
- Support for custom labels and dimensions
- Pull-based model preferred (no external dependencies)
- Works with Grafana for visualization

## Decision

We will use **Prometheus client library** for metrics instrumentation with the following approach:

- Export metrics via `/metrics` endpoint
- Use standard Prometheus metric types (Counter, Gauge, Histogram)
- Follow Prometheus naming conventions
- Implement custom metrics for domain-specific operations
- Integrate with Fiber via middleware
- Expose Grafana dashboards in repository

### Metric Categories

1. **HTTP Metrics**
   - Request count by method, path, status
   - Request duration histogram
   - In-flight request gauge

2. **KV Store Metrics**
   - Operation count by type (get, set, delete)
   - Store size gauge
   - Operation error count

3. **Service Discovery Metrics**
   - Registered services gauge
   - Service operations by type
   - Heartbeat count
   - Expired services counter

4. **Rate Limiting Metrics**
   - Requests checked counter
   - Rate limit exceeded counter
   - Active clients gauge

5. **System Metrics**
   - Build info
   - Go runtime metrics (via default collector)

## Alternatives Considered

### Alternative 1: StatsD/DogStatsD
- **Pros**:
  - Simple UDP protocol
  - Low overhead
  - Language agnostic
  - Works with DataDog, Grafana
- **Cons**:
  - Requires external agent/collector
  - UDP means potential data loss
  - Push-based (needs statsd server)
  - Less rich histogram support
- **Reason for rejection**: Prometheus pull model simpler; no agent needed

### Alternative 2: OpenTelemetry Metrics
- **Pros**:
  - Modern, vendor-neutral standard
  - Unified observability (traces + metrics + logs)
  - Growing ecosystem
  - Multiple export formats
- **Cons**:
  - More complex setup than Prometheus
  - Still maturing compared to Prometheus
  - Requires collector infrastructure
  - Larger binary size
- **Reason for rejection**: Prometheus more mature and simpler; OTel adds complexity

### Alternative 3: InfluxDB
- **Pros**:
  - Purpose-built time-series database
  - High write throughput
  - Rich query language
  - Good for custom metrics
- **Cons**:
  - Requires separate database service
  - More operational overhead
  - Push-based (requires client)
  - Less common in Kubernetes environments
- **Reason for rejection**: Prometheus native in Kubernetes; simpler operations

### Alternative 4: Custom Metrics API
- **Pros**:
  - Full control over format
  - Minimal dependencies
  - Can optimize for specific use case
- **Cons**:
  - Requires building entire metrics stack
  - No ecosystem integration
  - Custom tooling needed
  - Reinventing the wheel
- **Reason for rejection**: Prometheus is battle-tested standard

## Consequences

### Positive
- Industry-standard format works with existing monitoring tools
- Pull-based model simplifies deployment (no push configuration)
- Rich histogram support for latency percentiles
- Native Kubernetes integration
- Grafana has excellent Prometheus support
- Go client library is mature and performant
- Default collectors provide runtime metrics automatically
- Can scrape multiple Konsul instances easily

### Negative
- Pull model requires Prometheus server to be configured
- Cardinality explosions possible with too many labels
- Metrics endpoint could be hammered (needs rate limiting consideration)
- High-cardinality metrics can increase memory usage
- Need to design metric retention policies separately

### Neutral
- Need to establish naming conventions for consistency
- Metric labels must be carefully chosen (avoid high cardinality)
- Dashboard creation requires Grafana knowledge
- Alerting rules need to be defined in Prometheus

## Implementation Notes

### Naming Convention
Follow Prometheus best practices:
- Prefix: `konsul_`
- Use underscores: `konsul_http_requests_total`
- Suffix counters with `_total`
- Units in name: `_seconds`, `_bytes`

### Label Strategy
Keep cardinality reasonable:
- Use: method, path pattern, status_code, operation
- Avoid: user_id, api_key, specific_key_name, timestamp

### Middleware Integration
```go
app.Use(middleware.MetricsMiddleware())
```

Records:
- Request duration (histogram)
- Request count (counter)
- In-flight requests (gauge)

### Dashboard Automation
- Provide Grafana dashboard JSON in `/monitoring/grafana`
- Include example alerts
- Document dashboard variables

### Performance Considerations
- Use summary for high-throughput endpoints if histograms cause issues
- Monitor Prometheus memory usage
- Set appropriate histogram buckets
- Consider sampling for extremely high-traffic scenarios

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)
- [Konsul Grafana Dashboard](../../monitoring/grafana/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-22 | Konsul Team | Initial version |
