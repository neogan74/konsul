# ADR-0007: OpenTelemetry for Distributed Tracing

**Date**: 2024-10-05

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: observability, tracing, performance, debugging

## Context

As Konsul is deployed in distributed environments, understanding request flows and debugging performance issues becomes challenging. Requirements:

- Trace requests across service boundaries
- Measure latency at different stages
- Identify bottlenecks and slow operations
- Correlate traces with logs and metrics
- Support for common observability backends (Jaeger, Tempo, etc.)
- Minimal performance overhead
- Industry-standard protocol

Traditional logging and metrics don't show the complete picture of request execution paths in distributed systems.

## Decision

We will implement **OpenTelemetry distributed tracing** with:

- OpenTelemetry SDK for instrumentation
- OTLP (OpenTelemetry Protocol) over gRPC for export
- Automatic instrumentation via middleware
- Manual span creation for critical operations
- Configurable sampling ratio
- Support for both development and production deployments
- Integration with OpenTelemetry Collector

### Implementation Approach

1. **HTTP Middleware**: Automatically traces all HTTP requests
2. **Manual Spans**: Trace critical operations (persistence, service registration)
3. **Context Propagation**: Pass trace context via HTTP headers (W3C Trace Context)
4. **Flexible Backend**: Export to any OTLP-compatible collector
5. **Configurable Sampling**: Adjust trace volume vs detail trade-off

### Trace Structure
```
HTTP Request [span]
├── Validate Input [span]
├── KV Store Operation [span]
│   └── BadgerDB Write [span]
└── Send Response [span]
```

## Alternatives Considered

### Alternative 1: Jaeger Client Directly
- **Pros**:
  - Mature, well-established
  - Good documentation
  - Direct integration
  - Lower level control
- **Cons**:
  - Vendor lock-in to Jaeger
  - Cannot easily switch backends
  - Less standardized
  - Older protocol (Thrift)
- **Reason for rejection**: OpenTelemetry is vendor-neutral standard; better future-proofing

### Alternative 2: Zipkin
- **Pros**:
  - Simple, lightweight
  - Good for smaller deployments
  - Easy to set up
  - Twitter-proven
- **Cons**:
  - Older than OpenTelemetry
  - Less ecosystem support
  - Vendor-specific format
  - Losing mindshare to OpenTelemetry
- **Reason for rejection**: OpenTelemetry is becoming industry standard

### Alternative 3: AWS X-Ray
- **Pros**:
  - Native AWS integration
  - Good for AWS deployments
  - Managed service
- **Cons**:
  - AWS lock-in
  - Not portable to other clouds
  - Requires AWS infrastructure
  - Cost considerations
- **Reason for rejection**: Konsul should be cloud-agnostic

### Alternative 4: No Tracing (Logs + Metrics Only)
- **Pros**:
  - Simpler implementation
  - No additional backend needed
  - Lower overhead
  - Fewer dependencies
- **Cons**:
  - Cannot trace request flow
  - Difficult to debug distributed issues
  - No latency breakdown
  - Missing critical observability dimension
- **Reason for rejection**: Tracing essential for production debugging

### Alternative 5: Custom Tracing Implementation
- **Pros**:
  - Full control
  - Minimal dependencies
  - Can optimize for specific needs
- **Cons**:
  - Reinventing the wheel
  - No ecosystem integration
  - Maintenance burden
  - Non-standard format
- **Reason for rejection**: OpenTelemetry provides standard, mature solution

## Consequences

### Positive
- Complete request flow visibility
- Identify performance bottlenecks quickly
- Correlate traces with metrics and logs
- Vendor-neutral (works with Jaeger, Tempo, Honeycomb, etc.)
- Industry standard protocol and format
- Context propagation for distributed tracing
- Sampling reduces production overhead
- Good Go SDK support
- Integration with observability stack (Grafana, etc.)

### Negative
- Additional complexity in codebase
- Performance overhead (even with sampling)
- Requires OpenTelemetry Collector infrastructure
- More configuration to manage
- Trace storage can be expensive
- Learning curve for team
- Need to handle context propagation carefully

### Neutral
- Need to choose appropriate sampling ratio
- Backend choice affects operational complexity
- Span naming and attributes need conventions
- Must monitor collector performance
- Trace retention policies needed

## Implementation Notes

### Configuration
```go
Tracing: TracingConfig{
    Enabled:        true,
    Endpoint:       "otel-collector:4318",
    ServiceName:    "konsul",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    SamplingRatio:  0.1,  // 10% sampling
    InsecureConn:   false,
}
```

### Middleware Integration
```go
if cfg.Tracing.Enabled {
    app.Use(middleware.TracingMiddleware(cfg.Tracing.ServiceName))
}
```

### Manual Span Creation
```go
ctx, span := tracer.Start(ctx, "kv.Set")
defer span.End()

span.SetAttributes(
    attribute.String("key", key),
    attribute.Int("value.size", len(value)),
)
```

### Trace Context Propagation
Uses W3C Trace Context standard:
```
traceparent: 00-<trace-id>-<span-id>-<flags>
```

### Sampling Strategy
- **Development**: 100% sampling (1.0)
- **Staging**: 50% sampling (0.5)
- **Production**: 10% sampling (0.1) or adaptive

### Backend Options
Works with any OTLP collector:
- Jaeger
- Grafana Tempo
- Honeycomb
- Lightstep
- New Relic
- DataDog (with OTLP support)

### Performance Considerations
- Sampling reduces overhead in production
- Async export doesn't block request handling
- Batch exporting for efficiency
- Monitor exporter queue size
- Resource detection adds metadata

### Span Attributes
Standard attributes to include:
- Service name and version
- Environment (dev/staging/prod)
- HTTP method, path, status code
- Error information
- Custom business metrics

### Future Enhancements
- Add exemplars linking traces to metrics
- Implement tail sampling
- Add trace-based alerting
- Integrate with error tracking
- Custom span processors
- Baggage propagation for metadata

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OTLP Specification](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go)
- [Konsul telemetry package](../../internal/telemetry/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-10-05 | Konsul Team | Initial version |
