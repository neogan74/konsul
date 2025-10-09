# ADR-0001: Use Fiber Web Framework

**Date**: 2024-09-17

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: backend, web-framework, performance

## Context

Konsul requires a high-performance HTTP web framework for building RESTful APIs for KV storage and service discovery. The framework needs to:

- Provide excellent performance with low latency
- Support middleware for authentication, logging, and metrics
- Have intuitive routing and handler patterns
- Be actively maintained with good community support
- Minimize memory allocation and GC pressure
- Support graceful shutdown for production deployments

## Decision

We will use **Fiber v2** as the web framework for Konsul's HTTP API.

Fiber is an Express-inspired web framework built on top of Fasthttp, designed for speed and minimal memory footprint. It provides:

- Zero memory allocation router with tree-based routing
- Built-in middleware ecosystem (CORS, compression, logging, etc.)
- Express-like API that's familiar to many developers
- Excellent performance benchmarks (often faster than Gin)
- Active development and strong community

## Alternatives Considered

### Alternative 1: Gin
- **Pros**:
  - Most popular Go web framework
  - Large ecosystem and community
  - Good documentation and examples
  - Uses net/http under the hood (standard library)
- **Cons**:
  - Slower than Fiber in benchmarks
  - Higher memory allocation
  - Uses httprouter which is less flexible than Fiber's routing
- **Reason for rejection**: Performance requirements and modern API design favor Fiber

### Alternative 2: Echo
- **Pros**:
  - High performance
  - Clean and simple API
  - Good middleware support
- **Cons**:
  - Smaller community than Gin
  - Less expressive routing than Fiber
  - More verbose error handling
- **Reason for rejection**: Fiber provides better performance and more intuitive API

### Alternative 3: Standard library (net/http)
- **Pros**:
  - No external dependencies
  - Battle-tested and stable
  - Full control over implementation
- **Cons**:
  - Requires building routing, middleware, and utilities from scratch
  - More boilerplate code
  - Slower development velocity
- **Reason for rejection**: Too much boilerplate for a service discovery system

## Consequences

### Positive
- Excellent performance with minimal overhead
- Fast development with intuitive API
- Built-in middleware reduces custom code
- Easy integration with Prometheus for metrics
- Graceful shutdown support for production reliability

### Negative
- Uses Fasthttp instead of net/http (different from standard library patterns)
- Smaller ecosystem compared to Gin
- Some third-party packages may require adapters (e.g., prometheus)
- Less common in enterprise Go codebases

### Neutral
- Team needs to learn Fiber-specific patterns (though similar to Express)
- Middleware written for net/http requires adaptation

## Implementation Notes

Key implementation patterns:
- Use `fiber.New()` with custom configuration for production settings
- Implement middleware stack: logging → metrics → tracing → rate limiting → auth
- Use `adaptor` package for integrating net/http handlers (Prometheus)
- Implement graceful shutdown with signal handling

## References

- [Fiber Documentation](https://docs.gofiber.io/)
- [Fiber GitHub](https://github.com/gofiber/fiber)
- [Fasthttp Performance](https://github.com/valyala/fasthttp)
- [Go Web Framework Benchmarks](https://github.com/smallnest/go-web-framework-benchmark)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-17 | Konsul Team | Initial version |
