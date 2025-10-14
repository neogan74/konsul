# ADR-0008: Migrate from Fiber to Chi Router

**Date**: 2025-10-09

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: backend, web-framework, refactoring, migration

## Context

Konsul currently uses Fiber (built on Fasthttp) as its web framework (see ADR-0001). While Fiber provides excellent performance, we are reconsidering this decision due to several factors:

### Current Pain Points with Fiber

1. **Fasthttp incompatibility**: Fiber uses Fasthttp instead of net/http, requiring adapters for standard middleware
2. **Ecosystem compatibility**: Many Go libraries expect net/http interfaces
3. **Context handling**: Fiber's context differs from Go's standard context.Context
4. **Middleware complexity**: Need adapters for Prometheus, OpenTelemetry, and other tools
5. **Testing complexity**: Mock frameworks typically work with net/http
6. **Community drift**: Most modern Go projects standardizing on net/http-based routers

### Requirements for New Framework

- Built on standard library net/http
- Minimal overhead and good performance
- Clean routing with URL parameters
- Compatible with standard middleware
- Active maintenance and community
- Easy migration path from Fiber
- Support for middleware chains

## Decision

We propose migrating from **Fiber** to **Chi router**.

Chi is a lightweight, idiomatic router built on net/http that provides:

- Standard net/http compatibility
- Zero external dependencies (besides net/http)
- Excellent routing performance
- Clean middleware composition
- Context-based routing
- URL parameters via standard context
- Active development and strong community
- Simple, Go-idiomatic API

### Migration Strategy

1. **Phase 1**: Add Chi alongside Fiber (dual routing)
2. **Phase 2**: Migrate endpoints incrementally
3. **Phase 3**: Update middleware to use standard net/http
4. **Phase 4**: Remove Fiber dependency
5. **Phase 5**: Clean up and optimize

## Alternatives Considered

### Alternative 1: Stay with Fiber
- **Pros**:
  - No migration effort needed
  - Current team familiar with it
  - Proven performance in our codebase
  - Working well for current needs
- **Cons**:
  - Fasthttp compatibility issues persist
  - Ecosystem friction continues
  - Harder to integrate standard tools
  - Adapters add complexity
- **Reason for rejection**: Technical debt increasing; better to migrate now

### Alternative 2: Gin
- **Pros**:
  - Most popular Go web framework
  - Large ecosystem and community
  - Good documentation
  - Built on net/http
- **Cons**:
  - More opinionated than Chi
  - Custom context (c.Context) instead of standard context
  - Heavier framework
  - Still has some net/http deviations
- **Reason for rejection**: Chi more lightweight and standard-library aligned

### Alternative 3: Echo
- **Pros**:
  - High performance
  - Good middleware support
  - Built on net/http
  - Clean API
- **Cons**:
  - Custom context (echo.Context)
  - More opinionated than Chi
  - Larger dependency tree
  - Less idiomatic than Chi
- **Reason for rejection**: Custom context reduces standard library benefits

### Alternative 4: Gorilla Mux
- **Pros**:
  - Very mature and stable
  - Built on net/http
  - Well-documented
  - Widely used
- **Cons**:
  - Slower than Chi and Fiber
  - Less active development
  - No middleware composition helpers
  - More verbose routing
- **Reason for rejection**: Performance and modernization considerations

### Alternative 5: Standard library only (http.ServeMux)
- **Pros**:
  - Zero dependencies
  - Maximum compatibility
  - Most stable option
  - Go 1.22+ has better routing
- **Cons**:
  - Limited routing features
  - No URL parameters (pre-1.22)
  - No middleware helpers
  - More boilerplate needed
- **Reason for rejection**: Too minimalist; Chi provides better DX with minimal overhead

## Consequences

### Positive
- **Standard library alignment**: Works with any net/http middleware
- **Ecosystem compatibility**: No more adapters for Prometheus, OTel, etc.
- **Testing**: Standard http test helpers work directly
- **Context propagation**: Uses standard context.Context throughout
- **Middleware reuse**: Can use any net/http middleware
- **Smaller binary**: Chi has minimal dependencies
- **Future-proof**: Aligned with Go community direction
- **Cleaner code**: Less adapter boilerplate

### Negative
- **Migration effort**: Need to refactor all handlers and middleware
- **Performance trade-off**: Chi slower than Fiber (but still fast enough)
- **Breaking change**: Major refactor required
- **Testing effort**: Need to test all endpoints after migration
- **Downtime risk**: Careful migration needed for production
- **Learning curve**: Team needs to learn Chi patterns
- **Temporary complexity**: Dual router during migration

### Neutral
- Different routing syntax (manageable)
- Need to update documentation
- Handler signatures change slightly
- Middleware stack changes format

## Implementation Notes

### Current Fiber Handler
```go
app.Get("/kv/:key", func(c *fiber.Ctx) error {
    key := c.Params("key")
    // handler logic
    return c.JSON(result)
})
```

### Chi Handler Equivalent
```go
r.Get("/kv/{key}", func(w http.ResponseWriter, r *http.Request) {
    key := chi.URLParam(r, "key")
    // handler logic
    json.NewEncoder(w).Encode(result)
})
```

### Migration Checklist

**Phase 1: Setup**
- [ ] Add Chi dependency
- [ ] Create Chi router alongside Fiber
- [ ] Update middleware to support both

**Phase 2: Handler Migration**
- [ ] Migrate health endpoints
- [ ] Migrate KV endpoints
- [ ] Migrate service discovery endpoints
- [ ] Migrate auth endpoints
- [ ] Migrate backup/restore endpoints

**Phase 3: Middleware Updates**
- [ ] Update logging middleware
- [ ] Update metrics middleware (remove adaptor)
- [ ] Update tracing middleware
- [ ] Update rate limiting middleware
- [ ] Update auth middleware

**Phase 4: Testing**
- [ ] Update integration tests
- [ ] Update unit tests for handlers
- [ ] Performance testing comparison
- [ ] Load testing

**Phase 5: Cleanup**
- [ ] Remove Fiber dependency
- [ ] Remove adapter code
- [ ] Update documentation
- [ ] Update examples

### Middleware Pattern

**Before (Fiber)**:
```go
app.Use(middleware.RequestLogging(logger))
app.Use(adaptor.HTTPHandler(promhttp.Handler()))
```

**After (Chi)**:
```go
r.Use(middleware.RequestLogging(logger))
r.Handle("/metrics", promhttp.Handler()) // No adapter!
```

### Performance Expectations

Based on benchmarks:
- Fiber: ~50,000 req/sec (current)
- Chi: ~45,000 req/sec (expected)
- Trade-off: ~10% performance for massive compatibility gains

For Konsul's use case (service discovery, KV store), this trade-off is acceptable.

### Risk Mitigation

1. **Feature flags**: Add flag to switch between Fiber/Chi during migration
2. **Incremental rollout**: Migrate endpoints one by one
3. **Monitoring**: Compare metrics before/after migration
4. **Rollback plan**: Keep Fiber code until Chi proven stable
5. **Testing**: Comprehensive test coverage before switching

### Timeline Estimate

- Phase 1 (Setup): 1 day
- Phase 2 (Handlers): 3-5 days
- Phase 3 (Middleware): 2-3 days
- Phase 4 (Testing): 2-3 days
- Phase 5 (Cleanup): 1 day

**Total**: ~2 weeks for complete migration

## References

- [Chi Router GitHub](https://github.com/go-chi/chi)
- [Chi Documentation](https://go-chi.io/)
- [Go HTTP Router Benchmark](https://github.com/julienschmidt/go-http-routing-benchmark)
- [ADR-0001: Use Fiber](./0001-use-fiber-web-framework.md) (supersedes)
- [Standard Library net/http](https://pkg.go.dev/net/http)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-09 | Konsul Team | Initial proposal |
