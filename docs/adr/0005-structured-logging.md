# ADR-0005: Structured Logging with Custom Logger

**Date**: 2024-09-23

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: observability, logging, operations

## Context

Konsul requires a logging system that:
- Supports structured logging (key-value pairs)
- Provides multiple output formats (text for dev, JSON for production)
- Has configurable log levels
- Minimal performance overhead
- Easy to use and consistent across codebase
- Integrates well with log aggregation systems (ELK, Loki, etc.)

Traditional string-based logging makes parsing and filtering difficult in production environments with high log volumes.

## Decision

We will implement a **custom structured logger** wrapper that:

- Provides structured logging API with key-value fields
- Supports both text (human-readable) and JSON output formats
- Implements standard log levels (debug, info, warn, error)
- Uses a simple, type-safe API
- Allows global logger configuration
- Zero dependencies on external logging libraries

### API Design
```go
logger.Info("message",
    logger.String("key", "value"),
    logger.Int("count", 10),
    logger.Error(err))
```

### Features
- Configurable via environment variables
- Log levels: debug, info, warn, error
- Formats: text (colored for terminal), json (for production)
- Thread-safe implementation
- Context propagation support
- Minimal allocations

## Alternatives Considered

### Alternative 1: Logrus
- **Pros**:
  - Very popular in Go ecosystem
  - Structured logging support
  - Many hooks and formatters
  - Battle-tested
- **Cons**:
  - Slower than modern alternatives
  - Uses reflection (performance impact)
  - Larger dependency
  - Development stalled (less active)
- **Reason for rejection**: Performance concerns; can implement simpler custom solution

### Alternative 2: Zap (Uber)
- **Pros**:
  - Extremely fast (zero-allocation)
  - Structured logging
  - Production-grade
  - Active development
- **Cons**:
  - API complexity (SugaredLogger vs Logger)
  - Larger learning curve
  - Heavier dependency
  - Configuration more complex
- **Reason for rejection**: Over-engineered for our needs; custom solution simpler

### Alternative 3: Zerolog
- **Pros**:
  - Very fast (zero-allocation)
  - Chainable API
  - JSON-first design
  - Minimal allocations
- **Cons**:
  - JSON-centric (text output secondary)
  - Different API paradigm (chaining)
  - Another dependency
- **Reason for rejection**: Custom solution gives full control; simpler dependency tree

### Alternative 4: Standard library (log)
- **Pros**:
  - No dependencies
  - Simple and stable
  - Familiar to all Go developers
- **Cons**:
  - No structured logging
  - No log levels
  - No JSON output
  - Limited functionality
- **Reason for rejection**: Insufficient features for production observability

## Consequences

### Positive
- Full control over logging behavior and performance
- No external dependencies for core logging
- Simple API tailored to Konsul's needs
- Easy to extend with new field types
- Lightweight implementation
- Can optimize for specific use cases
- Easy to add context propagation later
- Text format has colors for better developer experience

### Negative
- Custom code to maintain
- Missing features compared to mature libraries (hooks, sampling, etc.)
- Need to implement additional features ourselves if needed
- Less community support
- No ecosystem of plugins/integrations

### Neutral
- Team owns the logging implementation
- Need to document logging patterns
- Can add features incrementally as needed
- Must ensure thread-safety ourselves

## Implementation Notes

### Configuration
```go
Log: LogConfig{
    Level:  "info",     // debug, info, warn, error
    Format: "text",     // text, json
}
```

### Usage Patterns
```go
// Basic logging
logger.Info("Server started", logger.String("address", ":8888"))

// With error
logger.Error("Failed to connect",
    logger.Error(err),
    logger.String("host", "localhost"))

// With multiple fields
logger.Debug("Processing request",
    logger.String("method", "GET"),
    logger.String("path", "/api/v1/services"),
    logger.Int("status", 200),
    logger.Duration("latency", latency))
```

### Field Types
Implemented field types:
- String
- Int
- Bool
- Duration
- Error
- Float64
- Any (falls back to fmt.Sprint)

### Output Formats

**Text (Development)**
```
2024-09-23T10:30:00Z [INFO] Server started address=:8888
```

**JSON (Production)**
```json
{"level":"info","time":"2024-09-23T10:30:00Z","message":"Server started","address":":8888"}
```

### Performance Considerations
- Use string builders to minimize allocations
- Avoid reflection where possible
- Reuse buffers for formatting
- Only format log messages at appropriate level

### Future Enhancements
- Add sampling for high-volume logs
- Implement log rotation
- Add context propagation
- Support structured error types
- Add caller information (file:line)
- Implement log hooks for external systems

## References

- [Structured Logging Best Practices](https://www.honeycomb.io/blog/structured-logging-and-your-team)
- [Go Logging Benchmark](https://github.com/imkira/go-loggers-bench)
- [Konsul logger package](../../internal/logger/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-23 | Konsul Team | Initial version |
