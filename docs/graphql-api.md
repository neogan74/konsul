# GraphQL API Documentation

## Overview

Konsul provides a GraphQL API alongside the REST API for flexible querying of KV store and Service Discovery resources. This API enables clients to request exactly the data they need in a single request, reducing over-fetching and improving performance.

## Endpoint

- **GraphQL API**: `POST /graphql`
- **GraphQL Playground**: `GET /graphql/playground` (development only)

## Configuration

Enable GraphQL via environment variables:

```bash
# Enable GraphQL endpoint
KONSUL_GRAPHQL_ENABLED=true

# Enable GraphQL Playground (disable in production)
KONSUL_GRAPHQL_PLAYGROUND_ENABLED=true
```

Start Konsul with GraphQL:

```bash
KONSUL_GRAPHQL_ENABLED=true ./konsul
```

## Authentication

GraphQL API uses the same authentication as REST:

- JWT token via `Authorization: Bearer <token>` header
- API key via `X-API-Key: <key>` header

Note: The `health` query is public and does not require authentication.

## Features

### Phase 1 (Completed)

- ✅ **Read-only Queries**: KV store and Service Discovery
- ✅ **Pagination**: Limit and offset support
- ✅ **Filtering**: Prefix-based KV filtering
- ✅ **System Health**: Public health endpoint
- ✅ **Custom Scalars**: Time (RFC3339) and Duration types
- ✅ **GraphQL Playground**: Interactive query explorer

### Phase 2 (Completed)

- ✅ **Mutations**: Write operations for KV and Services
- ✅ **Subscriptions**: Real-time KV updates via WebSocket
- ✅ **WebSocket Transport**: Full subscription support with keep-alive
- ✅ **Pattern Matching**: Watch specific keys or prefixes

### Future Phases

- 🔜 **Service Subscriptions**: Real-time service change notifications (Phase 3)
- 🔜 **DataLoaders**: N+1 query optimization (Phase 3)
- 🔜 **Query Complexity Limits**: Protection against expensive queries (Phase 3)
- 🔜 **ACL Integration**: Field-level ACL enforcement (Phase 3)

## Schema

### Root Query Type

```graphql
type Query {
  # System health (public, no auth required)
  health: SystemHealth!

  # KV Store queries
  kv(key: String!): KVPair
  kvList(prefix: String, limit: Int, offset: Int): KVListResponse!

  # Service Discovery queries
  service(name: String!): Service
  services(limit: Int, offset: Int): [Service!]!
  servicesCount: Int!
}
```

### Types

#### SystemHealth

```graphql
type SystemHealth {
  status: String!
  version: String!
  uptime: String!
  timestamp: Time!
  services: ServiceStats!
  kvStore: KVStats!
}

type ServiceStats {
  total: Int!
  active: Int!
  expired: Int!
}

type KVStats {
  totalKeys: Int!
}
```

#### KV Store

```graphql
type KVPair {
  key: String!
  value: String!
  createdAt: Time
  updatedAt: Time
}

type KVListResponse {
  items: [KVPair!]!
  total: Int!
  hasMore: Boolean!
}
```

#### Service Discovery

```graphql
type Service {
  name: String!
  address: String!
  port: Int!
  status: ServiceStatus!
  expiresAt: Time!
  checks: [HealthCheck!]!
}

enum ServiceStatus {
  ACTIVE
  EXPIRED
}

type HealthCheck {
  id: String!
  serviceId: String!
  name: String!
  type: HealthCheckType!
  status: HealthCheckStatus!
  output: String
  interval: Duration
  timeout: Duration
  lastChecked: Time
}

enum HealthCheckType {
  HTTP
  TCP
  GRPC
  TTL
}

enum HealthCheckStatus {
  PASSING
  WARNING
  CRITICAL
}
```

#### Custom Scalars

```graphql
# RFC3339 timestamp (e.g., "2025-10-22T14:30:00Z")
scalar Time

# Duration string (e.g., "30s", "5m", "2h")
scalar Duration
```

## Example Queries

### 1. System Health

Get system health information (public endpoint):

```graphql
query {
  health {
    status
    version
    uptime
    timestamp
    services {
      total
      active
      expired
    }
    kvStore {
      totalKeys
    }
  }
}
```

**Response:**

```json
{
  "data": {
    "health": {
      "status": "healthy",
      "version": "0.1.0",
      "uptime": "5m30s",
      "timestamp": "2025-10-22T14:30:00+05:00",
      "services": {
        "total": 3,
        "active": 2,
        "expired": 1
      },
      "kvStore": {
        "totalKeys": 5
      }
    }
  }
}
```

### 2. Get KV Pair

Retrieve a single key-value pair:

```graphql
query {
  kv(key: "config/database") {
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
    "kv": {
      "key": "config/database",
      "value": "postgresql://localhost:5432/mydb",
      "createdAt": "2025-10-22T14:00:00+05:00",
      "updatedAt": "2025-10-22T14:00:00+05:00"
    }
  }
}
```

### 3. List KV Pairs

List all keys with optional prefix filtering and pagination:

```graphql
query {
  kvList(prefix: "config/", limit: 10, offset: 0) {
    items {
      key
      value
    }
    total
    hasMore
  }
}
```

**Response:**

```json
{
  "data": {
    "kvList": {
      "items": [
        {
          "key": "config/app",
          "value": "production"
        },
        {
          "key": "config/database",
          "value": "postgresql://localhost:5432/mydb"
        }
      ],
      "total": 2,
      "hasMore": false
    }
  }
}
```

### 4. Get Service

Retrieve a specific service by name:

```graphql
query {
  service(name: "web") {
    name
    address
    port
    status
    expiresAt
    checks {
      id
      name
      type
      status
      output
    }
  }
}
```

**Response:**

```json
{
  "data": {
    "service": {
      "name": "web",
      "address": "10.0.0.1",
      "port": 8080,
      "status": "ACTIVE",
      "expiresAt": "2025-10-22T14:45:00+05:00",
      "checks": [
        {
          "id": "check-1",
          "name": "web-health",
          "type": "HTTP",
          "status": "PASSING",
          "output": "HTTP 200 OK"
        }
      ]
    }
  }
}
```

### 5. List Services

List all services with pagination:

```graphql
query {
  services(limit: 10, offset: 0) {
    name
    address
    port
    status
  }

  servicesCount
}
```

**Response:**

```json
{
  "data": {
    "services": [
      {
        "name": "web",
        "address": "10.0.0.1",
        "port": 8080,
        "status": "ACTIVE"
      },
      {
        "name": "api",
        "address": "10.0.0.2",
        "port": 3000,
        "status": "ACTIVE"
      }
    ],
    "servicesCount": 2
  }
}
```

### 6. Complex Nested Query

Fetch multiple resources in a single request:

```graphql
query Dashboard {
  health {
    status
    version
    services {
      total
      active
    }
  }

  services {
    name
    address
    port
    status
    checks {
      status
      output
    }
  }

  kvList(prefix: "config/") {
    items {
      key
      value
    }
  }
}
```

## Using cURL

### Health Query

```bash
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ health { status version uptime } }"}'
```

### KV Query

```bash
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ kv(key: \"mykey\") { key value } }"}'
```

### Services Query

```bash
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ services { name address port } }"}'
```

### With Authentication

```bash
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{"query": "{ services { name address } }"}'
```

## GraphQL Playground

When `KONSUL_GRAPHQL_PLAYGROUND_ENABLED=true`, access the interactive playground at:

```
http://localhost:8888/graphql/playground
```

The playground provides:
- **Schema Explorer**: Browse the full GraphQL schema
- **Query Editor**: Write queries with autocomplete
- **Query History**: Access previous queries
- **Documentation**: Inline documentation for all types and fields

## Error Handling

GraphQL errors follow the GraphQL specification format:

```json
{
  "errors": [
    {
      "message": "key not found",
      "path": ["kv"],
      "extensions": {
        "code": "NOT_FOUND"
      }
    }
  ],
  "data": {
    "kv": null
  }
}
```

## Performance Considerations

### Advantages

- **Reduced Over-fetching**: Request only the fields you need
- **Single Request**: Fetch multiple resources in one query
- **Efficient**: Reduced bandwidth and latency
- **Flexible**: Clients control data structure

### Best Practices

1. **Use Field Selection**: Only request fields you need
2. **Implement Pagination**: Use `limit` and `offset` for large datasets
3. **Leverage Caching**: GraphQL responses are cacheable
4. **Monitor Query Complexity**: Use introspection to understand query costs

## Comparison with REST API

| Feature | REST | GraphQL |
|---------|------|---------|
| **Endpoint** | `/kv/:key`, `/services/:name` | `/graphql` |
| **Data Fetching** | Multiple requests | Single request |
| **Over-fetching** | Returns all fields | Returns only requested fields |
| **Under-fetching** | Multiple round-trips | Single query for nested data |
| **Versioning** | URL versioning | Schema evolution |
| **Caching** | Standard HTTP caching | Custom caching logic |

### When to Use GraphQL

✅ **Use GraphQL for:**
- Complex queries with nested data
- Mobile apps with bandwidth constraints
- Dashboards aggregating multiple data sources
- Applications requiring flexible data fetching

### When to Use REST

✅ **Use REST for:**
- Simple CRUD operations
- File uploads/downloads
- Existing integrations
- Tooling expecting REST endpoints

## Client Libraries

### JavaScript/TypeScript

```bash
npm install @apollo/client graphql
```

```typescript
import { ApolloClient, InMemoryCache, gql } from '@apollo/client';

const client = new ApolloClient({
  uri: 'http://localhost:8888/graphql',
  cache: new InMemoryCache(),
});

const { data } = await client.query({
  query: gql`
    query {
      health {
        status
        version
      }
    }
  `,
});
```

### Go

```bash
go get github.com/machinebox/graphql
```

```go
import "github.com/machinebox/graphql"

client := graphql.NewClient("http://localhost:8888/graphql")

req := graphql.NewRequest(`
    query {
        health {
            status
            version
        }
    }
`)

var response struct {
    Health struct {
        Status  string `json:"status"`
        Version string `json:"version"`
    } `json:"health"`
}

if err := client.Run(ctx, req, &response); err != nil {
    log.Fatal(err)
}
```

### Python

```bash
pip install gql
```

```python
from gql import gql, Client
from gql.transport.requests import RequestsHTTPTransport

transport = RequestsHTTPTransport(url='http://localhost:8888/graphql')
client = Client(transport=transport, fetch_schema_from_transport=True)

query = gql('''
    query {
        health {
            status
            version
        }
    }
''')

result = client.execute(query)
```

## Mutations (Phase 2)

GraphQL mutations enable write operations for both KV store and Service Discovery.

### Mutation Schema

```graphql
type Mutation {
  # KV Store mutations
  kvSet(key: String!, value: String!): KVPair!
  kvDelete(key: String!): Boolean!
  kvCAS(key: String!, value: String!, index: Int!): KVPair

  # Service mutations
  registerService(input: RegisterServiceInput!): Service!
  deregisterService(name: String!): Boolean!
  updateHeartbeat(name: String!): Service!
}

input RegisterServiceInput {
  name: String!
  address: String!
  port: Int!
  tags: [String!]
  metadata: [MetadataInput!]
}

input MetadataInput {
  key: String!
  value: String!
}
```

### Mutation Examples

#### 1. Set KV Pair

```graphql
mutation {
  kvSet(key: "config/database", value: "postgresql://localhost/mydb") {
    key
    value
    updatedAt
  }
}
```

#### 2. Delete KV Pair

```graphql
mutation {
  kvDelete(key: "old/config")
}
```

**Response:** `true` if deleted, `false` if key didn't exist

#### 3. Compare-And-Swap (Atomic Update)

```graphql
mutation {
  kvCAS(key: "counter", value: "42", index: 5) {
    key
    value
  }
}
```

**Response:** Returns updated KVPair on success, `null` on index mismatch

#### 4. Register Service

```graphql
mutation {
  registerService(input: {
    name: "web-api"
    address: "10.0.1.5"
    port: 8080
    tags: ["api", "v1"]
    metadata: [
      { key: "version", value: "1.2.3" }
      { key: "region", value: "us-west" }
    ]
  }) {
    name
    address
    port
    status
    expiresAt
  }
}
```

#### 5. Deregister Service

```graphql
mutation {
  deregisterService(name: "old-service")
}
```

#### 6. Update Service Heartbeat

```graphql
mutation {
  updateHeartbeat(name: "web-api") {
    name
    expiresAt
  }
}
```

## Subscriptions (Phase 2)

Real-time updates via WebSocket for KV store changes.

### Subscription Schema

```graphql
type Subscription {
  # Watch KV changes by key or prefix
  kvChanged(key: String, prefix: String): KVChangeEvent!

  # Service changes (not yet implemented)
  serviceChanged(name: String): ServiceChangeEvent!
}

type KVChangeEvent {
  type: KVEventType!
  key: String!
  value: String
  oldValue: String
  timestamp: Time!
}

enum KVEventType {
  SET
  DELETE
}
```

### Subscription Examples

#### 1. Watch Specific Key

```graphql
subscription {
  kvChanged(key: "config/database") {
    type
    key
    value
    oldValue
    timestamp
  }
}
```

**Events:**
```json
{
  "data": {
    "kvChanged": {
      "type": "SET",
      "key": "config/database",
      "value": "postgresql://newhost/db",
      "oldValue": "postgresql://localhost/db",
      "timestamp": "2025-12-03T10:30:00Z"
    }
  }
}
```

#### 2. Watch Prefix (All Matching Keys)

```graphql
subscription {
  kvChanged(prefix: "config/") {
    type
    key
    value
    timestamp
  }
}
```

**Receives events for:**
- `config/database`
- `config/app`
- `config/cache`
- Any key starting with `config/`

#### 3. Watch All Keys

```graphql
subscription {
  kvChanged {
    type
    key
    value
  }
}
```

### Using Subscriptions with JavaScript

```javascript
import { createClient } from 'graphql-ws';

const client = createClient({
  url: 'ws://localhost:8888/graphql',
});

client.subscribe(
  {
    query: `
      subscription {
        kvChanged(prefix: "app/") {
          type
          key
          value
          timestamp
        }
      }
    `,
  },
  {
    next: (data) => {
      console.log('KV changed:', data);
    },
    error: (error) => {
      console.error('Subscription error:', error);
    },
    complete: () => {
      console.log('Subscription complete');
    },
  }
);
```

### Using Subscriptions with Apollo Client

```typescript
import { ApolloClient, InMemoryCache, split, HttpLink } from '@apollo/client';
import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { createClient } from 'graphql-ws';
import { getMainDefinition } from '@apollo/client/utilities';

// HTTP link for queries and mutations
const httpLink = new HttpLink({
  uri: 'http://localhost:8888/graphql',
});

// WebSocket link for subscriptions
const wsLink = new GraphQLWsLink(
  createClient({
    url: 'ws://localhost:8888/graphql',
  })
);

// Split based on operation type
const splitLink = split(
  ({ query }) => {
    const definition = getMainDefinition(query);
    return (
      definition.kind === 'OperationDefinition' &&
      definition.operation === 'subscription'
    );
  },
  wsLink,
  httpLink
);

const client = new ApolloClient({
  link: splitLink,
  cache: new InMemoryCache(),
});

// Use subscription
client.subscribe({
  query: gql`
    subscription {
      kvChanged(prefix: "config/") {
        type
        key
        value
      }
    }
  `,
}).subscribe({
  next: (data) => console.log(data),
  error: (error) => console.error(error),
});
```

## Roadmap

### Phase 3: Advanced Features (Planned)

- Service subscriptions (real-time service change notifications)
- DataLoaders for N+1 query optimization
- Query complexity analysis and limits
- Query depth limiting
- Rate limiting per client
- Field-level ACL enforcement
- Persistent queries (query whitelisting)

## Support

For issues, feature requests, or questions:
- GitHub Issues: https://github.com/neogan74/konsul/issues
- Documentation: https://github.com/neogan74/konsul/tree/main/docs
