# Konsul Observability Stack

This guide explains how to run Konsul with a complete observability stack including monitoring, logging, tracing, and demo services showcasing the Admin UI and GraphQL API.

## Overview

The `docker-compose.observability.yml` provides a production-like environment with:

- **Konsul** - Service mesh with Admin UI and GraphQL API enabled
- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization dashboards for metrics, logs, and traces
- **Loki** - Log aggregation
- **Promtail** - Log shipping to Loki
- **Tempo** - Distributed tracing
- **OpenTelemetry Collector** - Centralized telemetry collection
- **Demo Services** - Example services demonstrating Konsul features

## Quick Start

### 1. Start the Stack

```bash
docker-compose -f docker-compose.observability.yml up -d
```

### 2. Wait for Services to Start

```bash
# Watch Konsul logs
docker-compose -f docker-compose.observability.yml logs -f konsul

# Check all services are healthy
docker-compose -f docker-compose.observability.yml ps
```

### 3. Access the Interfaces

| Service | URL | Credentials |
|---------|-----|-------------|
| **Konsul Admin UI** | http://localhost:8888/admin | None |
| **GraphQL Playground** | http://localhost:8888/graphql/playground | None |
| **GraphQL Endpoint** | http://localhost:8888/graphql | None |
| **Konsul REST API** | http://localhost:8888 | None |
| **Grafana** | http://localhost:3000 | admin / admin |
| **Prometheus** | http://localhost:9090 | None |
| **Tempo** | http://localhost:3200 | None |

## Admin UI Features

The Admin UI (http://localhost:8888/admin) provides:

1. **Service Discovery Dashboard**
   - View all registered services in real-time
   - See health status and TTL information
   - Filter by tags and metadata
   - Monitor service instances for load balancing

2. **KV Store Browser**
   - Browse all key-value pairs
   - Create, update, and delete entries
   - JSON syntax highlighting
   - Search and filter capabilities

3. **Metrics Visualization**
   - System metrics (CPU, memory, requests)
   - Service-level metrics
   - Health check statistics
   - Integration with Prometheus

4. **Service Details**
   - View service metadata
   - Check health status
   - See associated tags
   - Monitor service endpoints

## GraphQL API Examples

### Accessing GraphQL Playground

1. Open http://localhost:8888/graphql/playground in your browser
2. The playground provides auto-complete, documentation, and schema exploration
3. Try the example queries below

### Example Queries

**1. List All Services:**
```graphql
{
  services {
    name
    address
    port
    tags
    metadata {
      key
      value
    }
  }
}
```

**2. Get Specific Service:**
```graphql
{
  service(name: "demo-api") {
    name
    address
    port
    tags
    healthy
    lastCheckTime
    metadata {
      key
      value
    }
  }
}
```

**3. Query KV Store:**
```graphql
{
  kvEntries {
    key
    value
  }
}
```

**4. Get Specific KV Entry:**
```graphql
{
  kvEntry(key: "config/demo-web/feature-flags") {
    key
    value
  }
}
```

**5. Health Check:**
```graphql
{
  health {
    healthy
    version
    uptime
    kvStoreSize
    registeredServices
  }
}
```

**6. Query Services by Tags:**
```graphql
{
  servicesByTags(tags: ["production", "rest"]) {
    name
    address
    port
    tags
  }
}
```

### Example Mutations

**1. Set KV Entry:**
```graphql
mutation {
  setKV(key: "my-config", value: "{\"enabled\": true}") {
    success
    message
  }
}
```

**2. Delete KV Entry:**
```graphql
mutation {
  deleteKV(key: "my-config") {
    success
    message
  }
}
```

**3. Register Service:**
```graphql
mutation {
  registerService(
    name: "my-service"
    address: "my-service.local"
    port: 8080
    tags: ["api", "v1"]
    metadata: [
      { key: "version", value: "1.0.0" }
      { key: "region", value: "us-east-1" }
    ]
  ) {
    success
    message
  }
}
```

**4. Deregister Service:**
```graphql
mutation {
  deregisterService(name: "my-service") {
    success
    message
  }
}
```

### Using GraphQL with curl

```bash
# Query services
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ services { name address port } }"}'

# Set KV value
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { setKV(key: \"test\", value: \"hello\") { success } }"}'
```

## Demo Services

The stack includes three demo services that automatically register with Konsul:

### demo-api-1 & demo-api-2
- **Purpose**: Demonstrate load balancing with multiple instances
- **Service Name**: `demo-api`
- **Tags**: `v1`, `production`, `rest`
- **Metadata**: Version, region, environment
- **Heartbeat**: Every 15 seconds

### demo-web
- **Purpose**: Simulate a frontend service
- **Service Name**: `demo-web`
- **Tags**: `frontend`, `react`, `v2`
- **Metadata**: Version, framework, environment
- **Heartbeat**: Every 20 seconds
- **KV Config**: Stores API endpoint and feature flags

### graphql-example
- **Purpose**: Demonstrate GraphQL API usage
- **Actions**:
  - Queries all services
  - Queries specific service
  - Queries KV store
  - Performs health checks
- **Output**: Check logs with `docker-compose -f docker-compose.observability.yml logs graphql-example`

## Viewing Demo Data

### 1. Via Admin UI
Open http://localhost:8888/admin and explore:
- Services: See `demo-api` (2 instances) and `demo-web`
- KV Store: View config entries for demo services

### 2. Via GraphQL Playground
Open http://localhost:8888/graphql/playground and run:
```graphql
{
  services {
    name
    address
    tags
  }
  kvEntries {
    key
    value
  }
}
```

### 3. Via REST API
```bash
# List services
curl http://localhost:8888/services/

# Get KV entries
curl http://localhost:8888/kv/
```

### 4. Via CLI (inside containers)
```bash
# Check graphql-example output
docker-compose -f docker-compose.observability.yml logs graphql-example

# Check service registration
docker-compose -f docker-compose.observability.yml logs demo-api-1
```

## Observability Features

### Metrics (Prometheus + Grafana)

1. **Prometheus Metrics** - http://localhost:9090
   - Konsul exposes metrics at `/metrics`
   - Demo services are monitored
   - Custom dashboards available in Grafana

2. **Grafana Dashboards** - http://localhost:3000
   - Pre-configured data sources (Prometheus, Loki, Tempo)
   - Service discovery metrics
   - KV store statistics
   - Request rates and latencies

### Logs (Loki + Promtail)

- **Promtail** collects logs from all Docker containers
- **Loki** aggregates and indexes logs
- **Grafana** provides log exploration and visualization
- Query logs by service, level, or custom labels

### Traces (Tempo + OpenTelemetry)

- **Konsul** sends traces to Tempo via OTLP
- **Tempo** stores and indexes distributed traces
- **Grafana** provides trace visualization
- End-to-end request tracing across services

## Configuration

### Konsul Environment Variables

The stack enables the following Konsul features:

```yaml
environment:
  # Admin UI
  - KONSUL_ADMIN_UI_ENABLED=true
  - KONSUL_ADMIN_UI_PATH=/admin

  # GraphQL API
  - KONSUL_GRAPHQL_ENABLED=true
  - KONSUL_GRAPHQL_PLAYGROUND_ENABLED=true

  # Persistence
  - KONSUL_PERSISTENCE_ENABLED=true
  - KONSUL_PERSISTENCE_TYPE=badger

  # Tracing
  - KONSUL_TRACING_ENABLED=true
  - KONSUL_TRACING_ENDPOINT=http://tempo:4317
```

### Customization

To customize the stack:

1. **Disable Demo Services**: Comment out `demo-api-1`, `demo-api-2`, `demo-web`, `graphql-example`
2. **Disable GraphQL**: Set `KONSUL_GRAPHQL_ENABLED=false`
3. **Disable UI**: Set `KONSUL_ADMIN_UI_ENABLED=false`
4. **Change UI Path**: Set `KONSUL_ADMIN_UI_PATH=/custom-path`
5. **Disable Tracing**: Set `KONSUL_TRACING_ENABLED=false`

## Testing the Stack

### 1. Verify Service Discovery

```bash
# List all services
curl http://localhost:8888/services/ | jq

# Get specific service
curl http://localhost:8888/services/demo-api | jq

# Load balance across demo-api instances
curl http://localhost:8888/lb/service/demo-api | jq
```

### 2. Test KV Store

```bash
# List all KV entries
curl http://localhost:8888/kv/ | jq

# Get specific config
curl http://localhost:8888/kv/config/demo-web/feature-flags | jq

# Set new value
curl -X PUT http://localhost:8888/kv/test/key -d "test-value"

# Verify via GraphQL
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ kvEntry(key: \"test/key\") { key value } }"}' | jq
```

### 3. Test Health Checks

```bash
# Overall health
curl http://localhost:8888/health | jq

# Readiness check
curl http://localhost:8888/health/ready | jq

# Liveness check
curl http://localhost:8888/health/live | jq
```

### 4. Test GraphQL Subscriptions

GraphQL subscriptions are available for real-time updates via WebSocket.

## Troubleshooting

### Services not appearing in UI

```bash
# Check Konsul logs
docker-compose -f docker-compose.observability.yml logs konsul

# Check demo service logs
docker-compose -f docker-compose.observability.yml logs demo-api-1

# Verify service is running
docker-compose -f docker-compose.observability.yml ps
```

### GraphQL Playground not loading

```bash
# Ensure GraphQL is enabled
docker-compose -f docker-compose.observability.yml exec konsul \
  env | grep GRAPHQL

# Check Konsul logs for GraphQL initialization
docker-compose -f docker-compose.observability.yml logs konsul | grep -i graphql
```

### Grafana not showing data

```bash
# Check Prometheus targets
# Visit http://localhost:9090/targets

# Check Grafana data sources
# Visit http://localhost:3000/datasources

# Verify Promtail is collecting logs
docker-compose -f docker-compose.observability.yml logs promtail
```

### Traces not appearing in Tempo

```bash
# Check Tempo logs
docker-compose -f docker-compose.observability.yml logs tempo

# Verify OTLP endpoint is reachable
docker-compose -f docker-compose.observability.yml exec konsul \
  wget -O- http://tempo:4317 2>&1 | head
```

## Stopping the Stack

```bash
# Stop all services
docker-compose -f docker-compose.observability.yml down

# Stop and remove volumes (data will be lost)
docker-compose -f docker-compose.observability.yml down -v

# Stop and remove images
docker-compose -f docker-compose.observability.yml down --rmi all -v
```

## Production Considerations

When deploying to production:

1. **Security**
   - Enable authentication (JWT or API keys)
   - Configure ACL policies
   - Use TLS for all endpoints
   - Restrict CORS origins

2. **Persistence**
   - Use external volumes for data
   - Configure backup strategies
   - Set up monitoring for data integrity

3. **Observability**
   - Configure retention policies
   - Set up alerting rules in Prometheus
   - Enable audit logging
   - Configure log rotation

4. **Scalability**
   - Use external Prometheus/Loki/Tempo
   - Configure resource limits
   - Set up horizontal scaling
   - Use external load balancers

## Further Reading

- [Admin UI Documentation](./admin-ui.md)
- [GraphQL API Documentation](./graphql.md)
- [Architecture Decisions](./adr/0009-react-admin-ui.md)
- [Main README](../README.md)