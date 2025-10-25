# GraphQL Phase 2 Implementation Plan

## Overview

Phase 2 adds write operations (mutations) and real-time updates (subscriptions) to Konsul's GraphQL API. This document provides a comprehensive implementation plan, technical design decisions, and testing strategy.

## Goals

- **Mutations**: Enable write operations for KV store and Service Discovery
- **Subscriptions**: Provide real-time updates via WebSocket
- **Consistency**: Maintain parity with existing REST API functionality
- **Performance**: Minimize overhead while ensuring reliability
- **Security**: Enforce authentication and authorization consistently

## Technical Architecture

### 1. Mutations Layer

Mutations will leverage existing business logic from REST handlers while providing GraphQL-specific error handling and validation.

**Architecture:**
```
GraphQL Mutation Resolver
    ↓
  Validation Layer (GraphQL-specific)
    ↓
  Existing Business Logic (shared with REST)
    ↓
  Storage Layer (KV/Service store)
```

**Design Decisions:**
- Reuse existing service layer logic to maintain consistency
- Add GraphQL-specific input validation
- Return rich error types with extensions
- Support optimistic concurrency control (CAS operations)
- Maintain transaction semantics where applicable

### 2. Subscriptions Layer

Subscriptions require WebSocket support and an event broadcasting mechanism.

**Architecture:**
```
GraphQL Subscription Resolver
    ↓
  WebSocket Connection Manager
    ↓
  Event Broadcasting System (Pub/Sub)
    ↓
  Storage Layer Events
```

**Components:**

1. **WebSocket Transport**
   - Protocol: `graphql-ws` (GraphQL over WebSocket Protocol)
   - Library: `github.com/99designs/gqlgen/graphql/handler/transport`
   - Connection lifecycle management
   - Authentication on connection initialization

2. **Event Broadcasting**
   - In-memory pub/sub for single instance
   - Redis pub/sub for multi-instance deployments
   - Event channels per subscription type
   - Automatic cleanup of inactive subscriptions

3. **Event Types**
   - KV changes (create, update, delete)
   - Service registration/deregistration
   - Service health status changes
   - Heartbeat updates

## Schema Design

### Mutation Types

```graphql
type Mutation {
  # KV Store mutations
  kvSet(key: String!, value: String!, flags: Int): KVPair!
  kvSetCAS(key: String!, value: String!, modifyIndex: Int!): KVSetCASResult!
  kvDelete(key: String!): Boolean!
  kvDeleteCAS(key: String!, modifyIndex: Int!): Boolean!
  kvDeleteTree(prefix: String!): KVDeleteTreeResult!

  # Service mutations
  registerService(input: RegisterServiceInput!): Service!
  deregisterService(name: String!): Boolean!
  updateHeartbeat(name: String!): Service!
  updateServiceCheck(serviceId: String!, checkId: String!, status: HealthCheckStatus!, output: String): HealthCheck!
}

# Input types
input RegisterServiceInput {
  name: String!
  address: String!
  port: Int!
  checks: [HealthCheckInput!]
}

input HealthCheckInput {
  name: String!
  type: HealthCheckType!
  interval: Duration!
  timeout: Duration!
  http: HTTPCheckInput
  tcp: TCPCheckInput
  grpc: GRPCCheckInput
  ttl: Duration
}

input HTTPCheckInput {
  url: String!
  method: String
  headers: [HeaderInput!]
  body: String
  tlsSkipVerify: Boolean
}

input TCPCheckInput {
  address: String!
}

input GRPCCheckInput {
  target: String!
  service: String
  useTLS: Boolean
}

input HeaderInput {
  name: String!
  value: String!
}

# Result types
type KVSetCASResult {
  success: Boolean!
  kvPair: KVPair
}

type KVDeleteTreeResult {
  deleted: Int!
  keys: [String!]!
}
```

### Subscription Types

```graphql
type Subscription {
  # KV Store subscriptions
  kvChanged(prefix: String): KVChangeEvent!
  kvWatchKey(key: String!): KVChangeEvent!

  # Service Discovery subscriptions
  serviceChanged(name: String): ServiceChangeEvent!
  serviceHealthChanged(name: String): ServiceHealthChangeEvent!
  allServicesChanged: ServiceChangeEvent!
}

# Event types
type KVChangeEvent {
  operation: KVOperation!
  key: String!
  kvPair: KVPair
  previousValue: String
  timestamp: Time!
}

enum KVOperation {
  SET
  DELETE
  DELETE_TREE
}

type ServiceChangeEvent {
  operation: ServiceOperation!
  service: Service!
  timestamp: Time!
}

enum ServiceOperation {
  REGISTERED
  DEREGISTERED
  HEARTBEAT_UPDATED
}

type ServiceHealthChangeEvent {
  service: Service!
  check: HealthCheck!
  previousStatus: HealthCheckStatus!
  newStatus: HealthCheckStatus!
  timestamp: Time!
}
```

## Implementation Steps

### Stage 1: Foundation (Week 1)

**1.1 Event System**
- [ ] Design event interface and event types
- [ ] Implement in-memory pub/sub system
- [ ] Add event emitters to KV store operations
- [ ] Add event emitters to Service operations
- [ ] Write unit tests for event system

**1.2 WebSocket Support**
- [ ] Add `graphql-ws` transport to GraphQL handler
- [ ] Implement connection lifecycle management
- [ ] Add authentication to WebSocket connections
- [ ] Configure connection limits and timeouts
- [ ] Write connection handling tests

### Stage 2: Mutations (Week 2-3)

**2.1 KV Store Mutations**
- [ ] Implement `kvSet` resolver
- [ ] Implement `kvSetCAS` resolver with optimistic locking
- [ ] Implement `kvDelete` resolver
- [ ] Implement `kvDeleteCAS` resolver
- [ ] Implement `kvDeleteTree` resolver
- [ ] Add input validation
- [ ] Write mutation unit tests
- [ ] Write mutation integration tests

**2.2 Service Mutations**
- [ ] Implement `registerService` resolver
- [ ] Implement `deregisterService` resolver
- [ ] Implement `updateHeartbeat` resolver
- [ ] Implement `updateServiceCheck` resolver
- [ ] Add input validation for health checks
- [ ] Write mutation unit tests
- [ ] Write mutation integration tests

**2.3 Error Handling**
- [ ] Define GraphQL error extensions
- [ ] Map business logic errors to GraphQL errors
- [ ] Add error codes (CONFLICT, NOT_FOUND, INVALID_INPUT, etc.)
- [ ] Implement partial success handling for batch operations
- [ ] Write error handling tests

### Stage 3: Subscriptions (Week 3-4)

**3.1 KV Subscriptions**
- [ ] Implement `kvChanged` resolver with prefix filtering
- [ ] Implement `kvWatchKey` resolver
- [ ] Add subscription lifecycle management
- [ ] Implement event filtering logic
- [ ] Write subscription unit tests
- [ ] Write subscription integration tests

**3.2 Service Subscriptions**
- [ ] Implement `serviceChanged` resolver with name filtering
- [ ] Implement `serviceHealthChanged` resolver
- [ ] Implement `allServicesChanged` resolver
- [ ] Add subscription lifecycle management
- [ ] Write subscription unit tests
- [ ] Write subscription integration tests

**3.3 Performance Optimization**
- [ ] Implement subscription connection pooling
- [ ] Add rate limiting per connection
- [ ] Implement backpressure handling
- [ ] Add memory limits for event buffers
- [ ] Profile and optimize event dispatch

### Stage 4: Testing & Documentation (Week 4-5)

**4.1 Integration Testing**
- [ ] Write end-to-end mutation tests
- [ ] Write end-to-end subscription tests
- [ ] Test concurrent mutations
- [ ] Test subscription reconnection scenarios
- [ ] Load test with multiple concurrent subscriptions
- [ ] Test authentication enforcement

**4.2 Documentation**
- [ ] Update main GraphQL documentation
- [ ] Add mutation examples with cURL
- [ ] Add subscription examples with various clients
- [ ] Document error codes and meanings
- [ ] Create WebSocket connection guide
- [ ] Add troubleshooting section

**4.3 Configuration**
- [ ] Add feature flags for mutations and subscriptions
- [ ] Add configuration for WebSocket limits
- [ ] Add configuration for event buffer sizes
- [ ] Document all new environment variables
- [ ] Create deployment guide

### Stage 5: Production Readiness (Week 5-6)

**5.1 Monitoring**
- [ ] Add Prometheus metrics for mutations
- [ ] Add Prometheus metrics for subscriptions
- [ ] Add metrics for WebSocket connections
- [ ] Add metrics for event queue sizes
- [ ] Create Grafana dashboard

**5.2 Security**
- [ ] Audit authentication enforcement
- [ ] Add rate limiting for mutations
- [ ] Add rate limiting for subscriptions
- [ ] Implement connection limits
- [ ] Security review and penetration testing

**5.3 Multi-Instance Support (Optional)**
- [ ] Implement Redis pub/sub adapter
- [ ] Add configuration for Redis backend
- [ ] Test multi-instance subscriptions
- [ ] Document distributed deployment

## Testing Strategy

### Unit Tests

**Mutations:**
- Input validation
- Business logic execution
- Error handling and mapping
- Optimistic locking (CAS operations)

**Subscriptions:**
- Event filtering logic
- Connection lifecycle
- Event dispatch
- Memory cleanup

**Coverage Target:** 80%+

### Integration Tests

**Scenarios:**
1. Mutation sequences (create → update → delete)
2. Concurrent mutations on same resource
3. Subscription receives events from mutations
4. Multiple subscriptions on same resource
5. Subscription reconnection after disconnect
6. Authentication failures
7. Rate limiting enforcement

### Load Tests

**Metrics to measure:**
- Mutations per second throughput
- Subscription connection limits
- Event dispatch latency
- Memory usage under load
- WebSocket connection overhead

**Test scenarios:**
- 1,000 concurrent subscriptions
- 100 mutations/sec sustained load
- Peak load: 500 mutations/sec
- Long-running subscriptions (24h+)

### E2E Tests

**Client scenarios:**
- JavaScript client (Apollo)
- Go client
- Python client
- Mixed query + mutation transactions
- Subscription reconnection logic

## Example Usage

### Mutations

#### Set KV Pair

```graphql
mutation {
  kvSet(key: "config/database", value: "postgresql://localhost:5432/mydb") {
    key
    value
    createdAt
    updatedAt
  }
}
```

**Response:**
```json
{
  "data": {
    "kvSet": {
      "key": "config/database",
      "value": "postgresql://localhost:5432/mydb",
      "createdAt": "2025-10-22T15:00:00Z",
      "updatedAt": "2025-10-22T15:00:00Z"
    }
  }
}
```

#### Register Service

```graphql
mutation {
  registerService(input: {
    name: "web"
    address: "10.0.0.1"
    port: 8080
    checks: [
      {
        name: "web-health"
        type: HTTP
        interval: "30s"
        timeout: "5s"
        http: {
          url: "http://10.0.0.1:8080/health"
          method: "GET"
        }
      }
    ]
  }) {
    name
    address
    port
    status
    checks {
      id
      name
      type
      status
    }
  }
}
```

### Subscriptions

#### Watch KV Changes

```graphql
subscription {
  kvChanged(prefix: "config/") {
    operation
    key
    kvPair {
      key
      value
    }
    timestamp
  }
}
```

**Event stream:**
```json
{
  "data": {
    "kvChanged": {
      "operation": "SET",
      "key": "config/database",
      "kvPair": {
        "key": "config/database",
        "value": "postgresql://localhost:5432/mydb"
      },
      "timestamp": "2025-10-22T15:00:00Z"
    }
  }
}
```

#### Watch Service Health

```graphql
subscription {
  serviceHealthChanged(name: "web") {
    service {
      name
      status
    }
    check {
      name
      status
      output
    }
    previousStatus
    newStatus
    timestamp
  }
}
```

## Configuration

### Environment Variables

```bash
# Phase 2 Feature Flags
KONSUL_GRAPHQL_MUTATIONS_ENABLED=true
KONSUL_GRAPHQL_SUBSCRIPTIONS_ENABLED=true

# WebSocket Configuration
KONSUL_GRAPHQL_WS_MAX_CONNECTIONS=1000
KONSUL_GRAPHQL_WS_READ_TIMEOUT=60s
KONSUL_GRAPHQL_WS_WRITE_TIMEOUT=10s
KONSUL_GRAPHQL_WS_PING_INTERVAL=30s

# Event System Configuration
KONSUL_GRAPHQL_EVENT_BUFFER_SIZE=100
KONSUL_GRAPHQL_EVENT_CLEANUP_INTERVAL=5m

# Rate Limiting
KONSUL_GRAPHQL_MUTATION_RATE_LIMIT=100  # per client per minute
KONSUL_GRAPHQL_SUBSCRIPTION_RATE_LIMIT=10  # per client

# Multi-Instance Support (Optional)
KONSUL_GRAPHQL_PUBSUB_BACKEND=redis  # or "memory"
KONSUL_GRAPHQL_REDIS_URL=redis://localhost:6379
```

### Code Configuration

```go
// internal/graphql/config.go
type Phase2Config struct {
    MutationsEnabled     bool
    SubscriptionsEnabled bool
    WebSocket            WebSocketConfig
    EventSystem          EventConfig
    RateLimit            RateLimitConfig
}

type WebSocketConfig struct {
    MaxConnections int
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    PingInterval   time.Duration
}

type EventConfig struct {
    BufferSize      int
    CleanupInterval time.Duration
    Backend         string  // "memory" or "redis"
    RedisURL        string
}

type RateLimitConfig struct {
    MutationRateLimit     int
    SubscriptionRateLimit int
}
```

## Error Codes

GraphQL errors will include extension codes for programmatic handling:

| Code | Description | HTTP Equivalent |
|------|-------------|-----------------|
| `NOT_FOUND` | Resource not found | 404 |
| `ALREADY_EXISTS` | Resource already exists | 409 |
| `INVALID_INPUT` | Validation failed | 400 |
| `CONFLICT` | CAS operation failed | 409 |
| `UNAUTHORIZED` | Authentication required | 401 |
| `FORBIDDEN` | Authorization failed | 403 |
| `RATE_LIMITED` | Rate limit exceeded | 429 |
| `INTERNAL_ERROR` | Server error | 500 |

**Example error:**
```json
{
  "errors": [
    {
      "message": "key already exists",
      "path": ["kvSet"],
      "extensions": {
        "code": "ALREADY_EXISTS",
        "key": "config/database"
      }
    }
  ],
  "data": {
    "kvSet": null
  }
}
```

## Performance Considerations

### Mutations

**Optimization strategies:**
- Batch mutations where possible
- Reuse existing transaction logic
- Minimize lock contention
- Cache validation results

**Expected performance:**
- Single mutation: < 10ms (p95)
- Batch mutation (10 items): < 50ms (p95)
- CAS operation: < 15ms (p95)

### Subscriptions

**Optimization strategies:**
- Connection pooling
- Event batching for high-frequency updates
- Backpressure handling
- Automatic client disconnection on slow consumers
- Memory limits on event buffers

**Expected performance:**
- Event dispatch latency: < 50ms (p95)
- Max concurrent connections: 1,000+ per instance
- Memory per connection: ~10KB baseline
- Event throughput: 10,000+ events/sec

## Migration Strategy

### Rollout Plan

**Phase 2a: Mutations (Week 1-3)**
1. Deploy with `KONSUL_GRAPHQL_MUTATIONS_ENABLED=false`
2. Enable in staging environment
3. Run load tests and validate metrics
4. Enable in production with monitoring
5. Gradual rollout to clients

**Phase 2b: Subscriptions (Week 3-6)**
1. Deploy with `KONSUL_GRAPHQL_SUBSCRIPTIONS_ENABLED=false`
2. Enable in staging with synthetic load
3. Validate WebSocket connection handling
4. Enable in production with connection limits
5. Gradual increase of connection limits

### Backward Compatibility

- All Phase 1 queries remain unchanged
- New mutations are opt-in via feature flag
- Subscriptions require explicit client support
- REST API remains primary interface (unchanged)
- GraphQL schema is additive only (no breaking changes)

### Rollback Plan

If issues arise:
1. Disable feature flags immediately
2. Gracefully close WebSocket connections
3. Drain in-flight mutations
4. Fall back to REST API
5. Investigate and fix issues
6. Re-enable with fixes

## Monitoring & Observability

### Prometheus Metrics

```
# Mutations
konsul_graphql_mutations_total{operation, status}
konsul_graphql_mutation_duration_seconds{operation}
konsul_graphql_mutation_errors_total{operation, error_code}

# Subscriptions
konsul_graphql_subscriptions_active{type}
konsul_graphql_subscription_events_total{type}
konsul_graphql_subscription_event_latency_seconds{type}
konsul_graphql_websocket_connections_active
konsul_graphql_websocket_connections_total
konsul_graphql_websocket_disconnections_total{reason}

# Event System
konsul_graphql_events_published_total{type}
konsul_graphql_events_dropped_total{type, reason}
konsul_graphql_event_buffer_size{type}
```

### Logging

**Log levels:**
- `INFO`: Connection lifecycle, mutation operations
- `WARN`: Rate limiting, slow consumers, buffer full
- `ERROR`: Authentication failures, internal errors
- `DEBUG`: Event dispatch, subscription filtering

**Structured logging fields:**
- `subscription_id`: Unique subscription identifier
- `connection_id`: WebSocket connection ID
- `user_id`: Authenticated user
- `operation`: Mutation or subscription type
- `duration_ms`: Operation duration

### Alerting

**Critical alerts:**
- WebSocket connection pool exhausted
- Event buffer full (possible memory leak)
- High mutation error rate (> 5%)
- Subscription event latency (p95 > 1s)

**Warning alerts:**
- WebSocket connections approaching limit
- Mutation latency degradation (p95 > 50ms)
- High rate limiting activity

## Security Considerations

### Authentication

- JWT tokens validated on every mutation
- WebSocket connections authenticated during handshake
- Token expiration enforced
- Refresh token support for long-lived subscriptions

### Authorization

- Reuse existing ACL system from REST API
- Per-resource authorization checks
- Namespace-level permissions
- Rate limiting per authenticated user

### Attack Vectors

**Mutation abuse:**
- Mitigation: Rate limiting, input validation
- Monitor: Mutation error rates per client

**Subscription flooding:**
- Mitigation: Connection limits, per-client subscription limits
- Monitor: Active subscriptions per client

**Event amplification:**
- Mitigation: Subscription filtering, backpressure
- Monitor: Event dispatch rate, dropped events

**Memory exhaustion:**
- Mitigation: Event buffer limits, connection limits
- Monitor: Memory usage, connection count

## Dependencies

### Go Libraries

```go
// go.mod additions
require (
    github.com/99designs/gqlgen v0.17.49
    github.com/gorilla/websocket v1.5.3
    github.com/redis/go-redis/v9 v9.6.0  // optional for multi-instance
)
```

### Testing Tools

- `github.com/stretchr/testify` - assertions
- `github.com/gorilla/websocket` - WebSocket client for tests
- Load testing: `k6` or `hey`

## Success Criteria

Phase 2 is complete when:

- [ ] All mutations implemented and tested
- [ ] All subscriptions implemented and tested
- [ ] Load tests pass with target metrics
- [ ] Documentation complete
- [ ] Security audit passed
- [ ] Monitoring dashboards created
- [ ] Production deployment successful
- [ ] Zero critical bugs in first 2 weeks

## Future Enhancements (Phase 3)

After Phase 2 stabilizes, consider:

- **DataLoaders**: Batch and cache database queries
- **Query Complexity Analysis**: Prevent expensive queries
- **Persistent Queries**: Query whitelisting for security
- **Distributed Tracing**: OpenTelemetry integration
- **GraphQL Federation**: Split schema across services

## References

- [GraphQL Specification](https://spec.graphql.org/)
- [graphql-ws Protocol](https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md)
- [gqlgen Documentation](https://gqlgen.com/)
- [Konsul GraphQL API Documentation](./graphql-api.md)

---

**Document Version:** 1.0
**Last Updated:** 2025-10-22
**Author:** Development Team
**Status:** Planning
