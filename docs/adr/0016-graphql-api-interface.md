# ADR-0016: GraphQL API Interface

**Date**: 2025-10-20

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: api, graphql, integration, query-interface

## Context

Konsul currently provides a RESTful HTTP API for all operations (KV store, service discovery, health checks, ACLs, rate limiting, backups). While REST is simple and well-understood, users increasingly need:

- **Flexible querying**: Fetch only the fields needed, reducing over-fetching
- **Multiple resource aggregation**: Retrieve related data in a single request (e.g., service + health checks + KV config)
- **Type safety**: Strongly-typed schema for better tooling and validation
- **Real-time updates**: Subscriptions for live data changes
- **Developer experience**: Self-documenting API with introspection
- **Reduced API calls**: Complex queries without multiple round-trips
- **Modern tooling**: Integration with GraphQL ecosystem (Apollo, Relay, GraphiQL)

Common use cases include:
- Dashboard UIs fetching multiple data types efficiently
- Mobile apps with bandwidth constraints needing precise data
- Analytics tools aggregating service + KV + metrics data
- Real-time monitoring dashboards with live updates
- Third-party integrations expecting GraphQL interfaces
- GraphQL federation with other microservices

The REST API will remain the primary interface, but GraphQL will provide an alternative for complex query scenarios.

## Decision

We will implement a **GraphQL API layer** alongside the existing REST API that:

- Provides read-optimized access to all Konsul resources
- Supports mutations for write operations (optional phase 2)
- Implements subscriptions for real-time updates
- Uses `gqlgen` (Go GraphQL library) for code generation
- Exposes a single GraphQL endpoint at `/graphql`
- Includes GraphiQL playground at `/graphql/playground`
- Shares authentication/authorization with REST API
- Maintains same RBAC/ACL rules as REST endpoints

### Architecture Components

**1. GraphQL Layer Structure**
```
internal/graphql/
├── schema/
│   ├── schema.graphql       # Main schema definition
│   ├── types.graphql        # Common types
│   ├── kv.graphql          # KV store types
│   ├── service.graphql     # Service discovery types
│   └── acl.graphql         # ACL types
├── resolver/
│   ├── resolver.go         # Root resolver
│   ├── kv.go              # KV resolvers
│   ├── service.go         # Service resolvers
│   └── subscription.go    # Real-time subscriptions
├── middleware/
│   ├── auth.go            # GraphQL auth middleware
│   └── complexity.go      # Query complexity limits
├── dataloaders/
│   └── loaders.go         # N+1 query prevention
└── server.go              # GraphQL server setup
```

**2. Core Schema Design**

```graphql
# Root Query Type
type Query {
  # KV Store
  kv(key: String!): KVPair
  kvList(prefix: String, limit: Int): [KVPair!]!
  kvTree(prefix: String!): KVTree!

  # Service Discovery
  service(name: String!): Service
  services(tag: String, healthy: Boolean): [Service!]!

  # Health Checks
  healthCheck(serviceId: String!): HealthCheck
  healthChecks(service: String, status: HealthStatus): [HealthCheck!]!

  # ACLs
  aclPolicy(id: String!): ACLPolicy
  aclPolicies: [ACLPolicy!]!

  # Rate Limiting
  rateLimitStatus(identifier: String!): RateLimitStatus

  # System
  health: SystemHealth!
  metrics: Metrics!
}

# Mutations (write operations)
type Mutation {
  # KV Store
  kvSet(input: KVSetInput!): KVPair!
  kvDelete(key: String!): Boolean!

  # Service Discovery
  registerService(input: RegisterServiceInput!): Service!
  deregisterService(name: String!): Boolean!
  updateHeartbeat(name: String!): Service!

  # ACLs
  createACLPolicy(input: CreateACLPolicyInput!): ACLPolicy!
  updateACLPolicy(id: String!, input: UpdateACLPolicyInput!): ACLPolicy!
  deleteACLPolicy(id: String!): Boolean!
}

# Subscriptions (real-time updates)
type Subscription {
  # KV Store changes
  kvChanged(prefix: String): KVChangeEvent!

  # Service changes
  serviceChanged(name: String): ServiceChangeEvent!

  # Health check updates
  healthCheckChanged(service: String): HealthCheckEvent!
}

# Types
type KVPair {
  key: String!
  value: String!
  flags: Int
  createIndex: Int!
  modifyIndex: Int!
  lockIndex: Int
  session: String
  createdAt: Time!
  updatedAt: Time!
}

type Service {
  id: String!
  name: String!
  address: String!
  port: Int!
  tags: [String!]
  meta: JSONObject
  registeredAt: Time!
  lastHeartbeat: Time!
  ttl: Duration!
  status: ServiceStatus!

  # Nested queries
  healthChecks: [HealthCheck!]!
  kvConfig(prefix: String): [KVPair!]!
}

type HealthCheck {
  id: String!
  serviceId: String!
  name: String!
  status: HealthStatus!
  output: String
  interval: Duration!
  timeout: Duration!
  lastChecked: Time!

  # Related service
  service: Service!
}

enum ServiceStatus {
  ACTIVE
  EXPIRED
  CRITICAL
}

enum HealthStatus {
  PASSING
  WARNING
  CRITICAL
}

# Custom scalars
scalar Time
scalar Duration
scalar JSONObject

# Input types
input KVSetInput {
  key: String!
  value: String!
  flags: Int
  cas: Int  # Compare-and-set
}

input RegisterServiceInput {
  name: String!
  address: String!
  port: Int!
  tags: [String!]
  meta: JSONObject
  checks: [HealthCheckInput!]
}
```

**3. GraphQL Server Implementation**

```go
// internal/graphql/server.go
package graphql

import (
    "github.com/99designs/gqlgen/graphql/handler"
    "github.com/99designs/gqlgen/graphql/playground"
)

type Server struct {
    kvStore     store.KVStore
    serviceReg  store.ServiceRegistry
    aclEval     acl.Evaluator
    config      Config
}

func NewServer(deps Dependencies) *Server {
    return &Server{
        kvStore:    deps.KVStore,
        serviceReg: deps.ServiceRegistry,
        aclEval:    deps.ACLEvaluator,
    }
}

func (s *Server) Handler() http.Handler {
    srv := handler.NewDefaultServer(
        generated.NewExecutableSchema(
            generated.Config{
                Resolvers: &Resolver{
                    kvStore:    s.kvStore,
                    serviceReg: s.serviceReg,
                    aclEval:    s.aclEval,
                },
            },
        ),
    )

    // Add middleware
    srv.Use(extension.FixedComplexityLimit(1000))
    srv.AroundOperations(s.authMiddleware)
    srv.AroundFields(s.aclMiddleware)

    return srv
}

func (s *Server) PlaygroundHandler() http.Handler {
    return playground.Handler("GraphQL Playground", "/graphql")
}
```

**4. Resolver Implementation**

```go
// internal/graphql/resolver/kv.go
package resolver

func (r *queryResolver) Kv(ctx context.Context, key string) (*model.KVPair, error) {
    // Check ACL permissions
    if err := r.aclEval.Authorize(ctx, acl.ResourceKV, key, acl.ActionRead); err != nil {
        return nil, err
    }

    // Fetch from store
    value, err := r.kvStore.Get(ctx, key)
    if err != nil {
        return nil, err
    }

    return &model.KVPair{
        Key:   key,
        Value: value,
        // ... other fields
    }, nil
}

func (r *queryResolver) Service(ctx context.Context, name string) (*model.Service, error) {
    // Check ACL
    if err := r.aclEval.Authorize(ctx, acl.ResourceService, name, acl.ActionRead); err != nil {
        return nil, err
    }

    svc, err := r.serviceReg.Get(ctx, name)
    if err != nil {
        return nil, err
    }

    return &model.Service{
        ID:      svc.ID,
        Name:    svc.Name,
        Address: svc.Address,
        // ... other fields
    }, nil
}
```

**5. Subscription Support (WebSocket)**

```go
// internal/graphql/resolver/subscription.go
package resolver

func (r *subscriptionResolver) KvChanged(ctx context.Context, prefix *string) (<-chan *model.KVChangeEvent, error) {
    events := make(chan *model.KVChangeEvent)

    // Subscribe to KV store changes
    go func() {
        defer close(events)

        watcher := r.kvStore.Watch(ctx, *prefix)
        for event := range watcher {
            events <- &model.KVChangeEvent{
                Key:       event.Key,
                Value:     event.Value,
                EventType: mapEventType(event.Type),
            }
        }
    }()

    return events, nil
}
```

### Example Queries

**Complex nested query:**
```graphql
query GetServiceWithConfig {
  service(name: "web") {
    id
    name
    address
    port
    status
    healthChecks {
      status
      output
      lastChecked
    }
    kvConfig(prefix: "config/web/") {
      key
      value
    }
  }
}
```

**Multiple resources in one query:**
```graphql
query Dashboard {
  health {
    status
    uptime
    services {
      total
      active
    }
  }

  services(healthy: true) {
    name
    address
    port
    status
  }

  criticalChecks: healthChecks(status: CRITICAL) {
    serviceId
    name
    output
  }
}
```

**Real-time subscription:**
```graphql
subscription WatchServiceChanges {
  serviceChanged(name: "web") {
    service {
      name
      status
      lastHeartbeat
    }
    changeType
    timestamp
  }
}
```

### Integration with REST API

```go
// cmd/konsul/main.go
func setupRoutes(r chi.Router, deps dependencies) {
    // Existing REST routes
    r.Route("/kv", func(r chi.Router) {
        r.Get("/{key}", kvHandler.Get)
        r.Put("/{key}", kvHandler.Set)
    })

    // GraphQL routes
    gqlServer := graphql.NewServer(deps)
    r.Handle("/graphql", gqlServer.Handler())
    r.Handle("/graphql/playground", gqlServer.PlaygroundHandler())
}
```

## Alternatives Considered

### Alternative 1: REST API with JSON:API specification
- **Pros**:
  - Standardized REST approach
  - Supports sparse fieldsets and includes
  - Simpler than GraphQL
  - No new dependencies
- **Cons**:
  - Still requires multiple requests for complex data
  - Less flexible than GraphQL
  - No real-time subscriptions
  - Query language less expressive
- **Reason for rejection**: Doesn't solve the core problem of multiple round-trips and flexible querying

### Alternative 2: gRPC with gRPC-Gateway
- **Pros**:
  - High performance binary protocol
  - Strong typing with protobuf
  - Bidirectional streaming
  - Code generation
- **Cons**:
  - Not web-friendly (requires transcoding)
  - Steeper learning curve
  - Less developer-friendly than GraphQL
  - Limited browser support
  - Tooling less mature than GraphQL
- **Reason for rejection**: gRPC better suited for service-to-service; GraphQL better for web/mobile clients

### Alternative 3: OData protocol
- **Pros**:
  - Rich query capabilities
  - Standardized by OASIS
  - Supports filtering, sorting, pagination
- **Cons**:
  - Complex specification
  - Poor tooling ecosystem
  - Less popular than GraphQL
  - Steeper learning curve
  - Limited community support
- **Reason for rejection**: GraphQL has better tooling and community adoption

### Alternative 4: Custom query DSL over HTTP
- **Pros**:
  - Full control over design
  - Can optimize for specific use cases
  - No external dependencies
- **Cons**:
  - Reinventing the wheel
  - Poor tooling support
  - Documentation burden
  - Users need to learn custom syntax
  - No ecosystem integration
- **Reason for rejection**: GraphQL provides proven solution with great ecosystem

### Alternative 5: Keep REST-only, use client-side data composition
- **Pros**:
  - No server changes needed
  - Simpler backend
  - Clients control data fetching
- **Cons**:
  - Multiple network requests
  - Bandwidth inefficient
  - Increased latency
  - Complex client logic
  - No server-side optimization
- **Reason for rejection**: Doesn't address the core problem of efficient data fetching

## Consequences

### Positive
- Flexible, efficient queries reducing over-fetching and under-fetching
- Single request for complex data aggregations
- Self-documenting API via introspection
- Strong typing catches errors early
- Rich ecosystem of client libraries (Apollo, Relay, urql)
- Real-time updates via subscriptions
- Better developer experience with GraphiQL playground
- Reduced bandwidth usage for mobile clients
- Modern API expected by many developers
- Easy to add new fields without versioning
- Field-level ACL enforcement possible

### Negative
- Additional complexity in codebase
- Another API surface to maintain and secure
- Learning curve for operators unfamiliar with GraphQL
- Query complexity attacks require rate limiting/depth limiting
- Caching more complex than REST
- Debugging can be harder than REST
- Schema evolution requires coordination
- Performance overhead of resolver execution
- May encourage overly complex queries
- Subscription infrastructure requires WebSocket support

### Neutral
- Two API styles to support (REST + GraphQL)
- Documentation needs for both APIs
- Testing coverage for both interfaces
- Monitoring and metrics for GraphQL operations
- Schema design requires upfront planning
- Need to educate users on when to use GraphQL vs REST
- Client library choices (REST vs GraphQL clients)

## Implementation Notes

### Phase 1: Core GraphQL Server (MVP)
- Setup gqlgen with basic schema
- Implement Query resolvers for KV and Service resources
- Add GraphiQL playground
- Basic authentication integration (JWT/API key)
- Schema introspection
- Error handling and logging
- **Timeline**: 2-3 weeks

### Phase 2: Mutations and Subscriptions
- Implement Mutation resolvers (write operations)
- Add WebSocket support for subscriptions
- Implement KV and Service change subscriptions
- Real-time event streaming
- **Timeline**: 2-3 weeks

### Phase 3: Performance and Security
- Implement DataLoaders to prevent N+1 queries
- Add query complexity analysis and limits
- Query depth limiting
- Rate limiting per client
- Field-level ACL enforcement
- Persistent query support (query whitelisting)
- **Timeline**: 2 weeks

### Phase 4: Enhanced Features
- Add ACL, Health Check, and Rate Limit schemas
- Implement batch operations
- Add pagination (cursor-based)
- Schema stitching support for future modularity
- Metrics and tracing integration
- **Timeline**: 2-3 weeks

### Technology Stack

- **gqlgen** (v0.17+): GraphQL server library with code generation
- **graphql-go** tools: Schema parsing and validation
- **WebSocket library**: For subscription support (gorilla/websocket)
- **DataLoader pattern**: Batch and cache data fetching

### Configuration

```bash
# Environment variables
KONSUL_GRAPHQL_ENABLED=true              # Enable GraphQL endpoint
KONSUL_GRAPHQL_PLAYGROUND_ENABLED=true   # Enable playground in dev
KONSUL_GRAPHQL_MAX_QUERY_DEPTH=10        # Query depth limit
KONSUL_GRAPHQL_MAX_COMPLEXITY=1000       # Complexity limit
KONSUL_GRAPHQL_INTROSPECTION_ENABLED=true # Allow introspection
```

### Security Considerations

**Query Complexity Attacks:**
- Limit query depth (max 10 levels)
- Analyze query complexity before execution
- Timeout long-running queries (30s max)
- Rate limit GraphQL endpoint separately

**Authorization:**
- Reuse existing ACL system
- Field-level authorization in resolvers
- Deny introspection in production (optional)
- Audit log all mutations

**Input Validation:**
- Validate all input types
- Sanitize string inputs
- Limit array/list sizes
- Reject malformed queries early

**Subscription Security:**
- Authenticate WebSocket connections
- Limit concurrent subscriptions per client
- Timeout inactive subscriptions
- Resource limits on subscription filters

### Error Handling

```graphql
# GraphQL errors follow spec
{
  "errors": [
    {
      "message": "Access denied",
      "path": ["service", "name"],
      "extensions": {
        "code": "FORBIDDEN",
        "resource": "service:web",
        "action": "read"
      }
    }
  ],
  "data": null
}
```

### Metrics to Track

- `konsul_graphql_requests_total{operation_type}` - Total GraphQL requests
- `konsul_graphql_request_duration_seconds{operation}` - Query latency
- `konsul_graphql_errors_total{error_code}` - GraphQL errors
- `konsul_graphql_query_depth{operation}` - Query depth distribution
- `konsul_graphql_complexity{operation}` - Query complexity
- `konsul_graphql_resolver_duration_seconds{resolver}` - Resolver performance
- `konsul_graphql_active_subscriptions` - Active WebSocket connections

### Testing Strategy

- **Unit tests**: Resolver logic with mock stores
- **Schema tests**: Validate GraphQL schema correctness
- **Integration tests**: End-to-end GraphQL queries
- **Performance tests**: N+1 query prevention, DataLoader efficiency
- **Security tests**: Authorization, complexity limits, injection
- **Subscription tests**: WebSocket lifecycle, event delivery

### Documentation

- GraphQL schema documentation (auto-generated)
- Query examples for common use cases
- Migration guide from REST to GraphQL
- Best practices for complex queries
- Subscription usage guide
- Performance optimization tips
- Security guidelines

### Migration Path

**For API consumers:**
1. GraphQL is additive - REST API remains unchanged
2. Gradually migrate complex multi-request flows to GraphQL
3. Use GraphQL for real-time features (subscriptions)
4. Keep using REST for simple CRUD operations
5. No breaking changes to existing integrations

**Recommended usage:**
- **Use REST for**: Simple CRUD, existing integrations, tooling expecting REST
- **Use GraphQL for**: Complex queries, dashboards, real-time updates, mobile apps

### Future Enhancements

- **GraphQL Federation**: Allow splitting schema across services
- **Persisted Queries**: Whitelist queries for better security
- **Automatic Persisted Queries**: Client-driven query registration
- **Schema Stitching**: Combine with other GraphQL services
- **GraphQL Code Generation**: Generate client SDKs
- **Advanced Subscriptions**: Filtered subscriptions, backpressure
- **GraphQL Analytics**: Query analytics dashboard
- **Hybrid Query Execution**: Combine multiple data sources

## References

- [GraphQL Specification](https://spec.graphql.org/)
- [gqlgen Documentation](https://gqlgen.com/)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices/)
- [Securing GraphQL APIs](https://www.apollographql.com/blog/graphql/security/)
- [DataLoader Pattern](https://github.com/graphql/dataloader)
- [GraphQL over WebSocket Protocol](https://github.com/enisdenjo/graphql-ws)
- [GraphQL Complexity Analysis](https://github.com/slicknode/graphql-query-complexity)
- [Apollo Server](https://www.apollographql.com/docs/apollo-server/)
- [Hasura GraphQL Engine](https://hasura.io/) (inspiration for real-time features)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-20 | Konsul Team | Initial version |
