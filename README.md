# Konsul service

In development now :> [!WARNING]

## KV storage (map with mutex)

| Method | endpoint  | Description  |
| ------ | --------- | ------------ |
| PUT    | /kv/<key> | Write value  |
| GET    | /kv/<key> | Read value   |
| DELETE | /kv/<key> | Delete value |

## Service Discovery (map)

| Method | Endpoint  | Description                                |
| ------ | --------- | ------------                               |
| PUT    | /register | Service registration                       |
| GET    | /services/ | Get all registered services in JSON       |
| GET    | /services/<name> | Get service with given name in JSON |
| DELETE | /deregister/<name> | Deregister service                |
| PUT    | /heartbeat/<name> | Update service TTL                |

## Authentication & Authorization

Konsul supports JWT-based authentication and API key authentication. When enabled, protected endpoints require a valid JWT token or API key.

### Authentication Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST   | /auth/login | Login and get JWT tokens | No |
| POST   | /auth/refresh | Refresh expired token | No |
| GET    | /auth/verify | Verify current token | Yes |

### API Key Management Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST   | /auth/apikeys | Create new API key | Yes (JWT) |
| GET    | /auth/apikeys | List all API keys | Yes (JWT) |
| GET    | /auth/apikeys/:id | Get specific API key | Yes (JWT) |
| PUT    | /auth/apikeys/:id | Update API key | Yes (JWT) |
| DELETE | /auth/apikeys/:id | Delete API key | Yes (JWT) |
| POST   | /auth/apikeys/:id/revoke | Revoke API key | Yes (JWT) |

### Using JWT Authentication

**1. Login to get tokens:**
```bash
curl -X POST http://localhost:8888/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "username": "admin",
    "roles": ["admin"]
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

**2. Use token in requests:**
```bash
curl http://localhost:8888/kv/mykey \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

**3. Refresh expired token:**
```bash
curl -X POST http://localhost:8888/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "username": "admin",
    "roles": ["admin"]
  }'
```

### Using API Key Authentication

**1. Create an API key (requires JWT):**
```bash
curl -X POST http://localhost:8888/auth/apikeys \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-api-key",
    "permissions": ["read", "write"],
    "metadata": {"env": "production"},
    "expires_in": 31536000
  }'
```

**Response:**
```json
{
  "key": "konsul_a1b2c3d4e5f6...",
  "api_key": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "name": "production-api-key",
    "permissions": ["read", "write"],
    "created_at": "2025-09-17T10:30:00Z",
    "enabled": true
  }
}
```

**2. Use API key in requests (two methods):**
```bash
# Method 1: X-API-Key header
curl http://localhost:8888/kv/mykey \
  -H "X-API-Key: konsul_a1b2c3d4e5f6..."

# Method 2: Authorization header
curl http://localhost:8888/kv/mykey \
  -H "Authorization: ApiKey konsul_a1b2c3d4e5f6..."
```

#### Example:
```
PUT /register
{
  "name": "auth-service",
  "address": "10.0.0.1",
  "port": 8080
}
```

## GraphQL API

Konsul provides a GraphQL API alongside the REST API for flexible querying of resources.

### Endpoints

| Endpoint | Description |
|----------|-------------|
| POST `/graphql` | GraphQL API endpoint |
| GET `/graphql/playground` | GraphQL Playground (development only) |

### Configuration

Enable GraphQL via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_GRAPHQL_ENABLED` | `false` | Enable GraphQL API |
| `KONSUL_GRAPHQL_PLAYGROUND_ENABLED` | `true` | Enable GraphQL Playground |

**Example:**
```bash
# Enable GraphQL with playground
KONSUL_GRAPHQL_ENABLED=true \
KONSUL_GRAPHQL_PLAYGROUND_ENABLED=true \
./konsul
```

### Example Queries

**Get system health:**
```graphql
query {
  health {
    status
    version
    uptime
    services {
      total
      active
    }
    kvStore {
      totalKeys
    }
  }
}
```

**Query KV store:**
```graphql
query {
  kv(key: "config/app") {
    key
    value
    createdAt
  }
}
```

**List services:**
```graphql
query {
  services {
    name
    address
    port
    status
    expiresAt
  }
}
```

**Complex nested query:**
```graphql
query Dashboard {
  health {
    status
    services {
      total
      active
    }
  }

  services {
    name
    address
    port
    checks {
      status
      output
    }
  }
}
```

### Using cURL

```bash
# Health query
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ health { status version } }"}'

# KV query
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ kv(key: \"mykey\") { key value } }"}'

# Services query
curl -X POST http://localhost:8888/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ services { name address port } }"}'
```

**See [GraphQL API Documentation](docs/graphql-api.md) for complete API reference.**

### Health Check TTL ✅

- Registration sets configurable TTL for service (default: 30s)
- TTL updated through `/heartbeat/<name>` endpoint
- Background process runs at configurable interval removing expired services (default: 60s)
- Services automatically expire if no heartbeat received within TTL

## CLI Tool (konsulctl)

The `konsulctl` command-line tool supports all TLS options for secure communication with the server:

**TLS Options (available for all commands):**
- `--server <url>` - Konsul server URL (use `https://` for TLS)
- `--tls-skip-verify` - Skip TLS certificate verification (for self-signed certs)
- `--ca-cert <file>` - Path to CA certificate file
- `--client-cert <file>` - Path to client certificate file (for mTLS)
- `--client-key <file>` - Path to client key file (for mTLS)

**Examples:**
```bash
# Connect to TLS server with self-signed certificate
konsulctl kv set --server https://localhost:8888 --tls-skip-verify mykey myvalue

# Connect with custom CA certificate
konsulctl service list --server https://localhost:8888 --ca-cert /path/to/ca.crt

# Connect with mutual TLS (client authentication)
konsulctl backup create --server https://localhost:8888 \
  --ca-cert /path/to/ca.crt \
  --client-cert /path/to/client.crt \
  --client-key /path/to/client.key
```

## Configuration

Configure via environment variables:

### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_PORT` | `8888` | Server port |
| `KONSUL_HOST` | `` | Server host (empty = all interfaces) |
| `KONSUL_SERVICE_TTL` | `30s` | Service TTL duration |
| `KONSUL_CLEANUP_INTERVAL` | `60s` | Cleanup interval |
| `KONSUL_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `KONSUL_LOG_FORMAT` | `text` | Log format (text/json) |

### TLS/SSL Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_TLS_ENABLED` | `false` | Enable TLS/SSL |
| `KONSUL_TLS_CERT_FILE` | `` | Path to TLS certificate file |
| `KONSUL_TLS_KEY_FILE` | `` | Path to TLS private key file |
| `KONSUL_TLS_AUTO_CERT` | `false` | Auto-generate self-signed certificate for development |

### Authentication Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_AUTH_ENABLED` | `false` | Enable authentication system |
| `KONSUL_JWT_SECRET` | `` | JWT signing secret (required if auth enabled) |
| `KONSUL_JWT_EXPIRY` | `15m` | JWT token expiry duration |
| `KONSUL_REFRESH_EXPIRY` | `168h` (7 days) | Refresh token expiry duration |
| `KONSUL_JWT_ISSUER` | `konsul` | JWT issuer name |
| `KONSUL_APIKEY_PREFIX` | `konsul` | API key prefix |
| `KONSUL_REQUIRE_AUTH` | `false` | Require authentication for all endpoints |
| `KONSUL_PUBLIC_PATHS` | `/health,/health/live,/health/ready,/metrics` | Comma-separated list of public paths |

### Persistence Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_PERSISTENCE_ENABLED` | `false` | Enable persistence |
| `KONSUL_PERSISTENCE_TYPE` | `badger` | Persistence type (memory/badger) |
| `KONSUL_DATA_DIR` | `./data` | Data directory for persistence |
| `KONSUL_BACKUP_DIR` | `./backups` | Backup directory |
| `KONSUL_SYNC_WRITES` | `true` | Enable synchronous writes |
| `KONSUL_WAL_ENABLED` | `true` | Enable write-ahead log |

### DNS Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_DNS_ENABLED` | `true` | Enable DNS server |
| `KONSUL_DNS_HOST` | `` | DNS server host |
| `KONSUL_DNS_PORT` | `8600` | DNS server port |
| `KONSUL_DNS_DOMAIN` | `consul` | DNS domain suffix |

**Examples:**
```bash
# Custom port
KONSUL_PORT=9999 ./konsul

# Production settings with JSON logging
KONSUL_HOST=0.0.0.0 KONSUL_PORT=80 KONSUL_LOG_FORMAT=json KONSUL_LOG_LEVEL=info ./konsul

# Debug mode with verbose logging
KONSUL_LOG_LEVEL=debug KONSUL_LOG_FORMAT=text ./konsul

# Enable TLS with auto-generated self-signed certificate (development)
KONSUL_TLS_ENABLED=true \
KONSUL_TLS_AUTO_CERT=true \
./konsul

# Enable TLS with custom certificates (production)
KONSUL_TLS_ENABLED=true \
KONSUL_TLS_CERT_FILE=/path/to/cert.pem \
KONSUL_TLS_KEY_FILE=/path/to/key.pem \
./konsul

# Enable rate limiting
KONSUL_RATE_LIMIT_ENABLED=true \
KONSUL_RATE_LIMIT_REQUESTS_PER_SEC=100 \
KONSUL_RATE_LIMIT_BURST=20 \
KONSUL_RATE_LIMIT_BY_IP=true \
# Enable authentication with JWT
KONSUL_AUTH_ENABLED=true \
KONSUL_JWT_SECRET="your-super-secret-key-min-32-chars" \
KONSUL_JWT_EXPIRY=30m \
KONSUL_REQUIRE_AUTH=true \
./konsul

# Enable authentication with custom public paths
KONSUL_AUTH_ENABLED=true \
KONSUL_JWT_SECRET="your-secret-key" \
KONSUL_PUBLIC_PATHS="/health,/health/live,/health/ready,/metrics,/custom/public" \
./konsul

# Enable persistence with authentication
KONSUL_AUTH_ENABLED=true \
KONSUL_JWT_SECRET="your-secret-key" \
KONSUL_PERSISTENCE_ENABLED=true \
KONSUL_PERSISTENCE_TYPE=badger \
KONSUL_DATA_DIR=/var/lib/konsul \
./konsul
```

## Rate Limiting

Konsul includes a token bucket rate limiter to protect against abuse and ensure fair usage.

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_RATE_LIMIT_ENABLED` | `false` | Enable rate limiting |
| `KONSUL_RATE_LIMIT_REQUESTS_PER_SEC` | `100.0` | Requests allowed per second |
| `KONSUL_RATE_LIMIT_BURST` | `20` | Burst size (max requests in short period) |
| `KONSUL_RATE_LIMIT_BY_IP` | `true` | Enable per-IP rate limiting |
| `KONSUL_RATE_LIMIT_BY_APIKEY` | `false` | Enable per-API-key rate limiting |
| `KONSUL_RATE_LIMIT_CLEANUP` | `5m` | Cleanup interval for unused limiters |

### How It Works

- **Token Bucket Algorithm**: Allows bursts while maintaining average rate
- **Per-IP Limiting**: Each IP address gets independent rate limit
- **Per-API-Key Limiting**: Each authenticated API key gets independent rate limit
- **Automatic Cleanup**: Unused limiters are cleaned up periodically

### Response Headers

When rate limited, responses include:
- `X-RateLimit-Limit: exceeded` - Rate limit status
- `X-RateLimit-Reset: <timestamp>` - When limit resets

### Error Response

```json
{
  "error": "rate limit exceeded",
  "message": "Too many requests. Please try again later.",
  "identifier": "ip:192.168.1.1"
}
```

### Metrics

Rate limiting exposes Prometheus metrics:
- `konsul_rate_limit_requests_total{limiter_type,status}` - Total requests checked
- `konsul_rate_limit_exceeded_total{limiter_type}` - Total violations
- `konsul_rate_limit_active_clients{limiter_type}` - Active clients being tracked

## Monitoring & Health Checks

### Health Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET    | `/health` | Detailed health status with system metrics |
| GET    | `/health/live` | Liveness probe (returns 200 if running) |
| GET    | `/health/ready` | Readiness probe (returns 200 if ready) |

**Health status response:**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "1h30m45s",
  "timestamp": "2025-09-18T10:30:00Z",
  "services": {
    "total": 5,
    "active": 4,
    "expired": 1
  },
  "kv_store": {
    "total_keys": 10
  },
  "system": {
    "goroutines": 8,
    "memory_alloc_bytes": 2097152,
    "memory_sys_bytes": 8388608,
    "num_gc": 3
  }
}
```

### Metrics (Prometheus)

| Endpoint | Description |
|----------|-------------|
| GET `/metrics` | Prometheus metrics endpoint |

**Available metrics:**
- `konsul_http_requests_total` - Total HTTP requests by method, path, status
- `konsul_http_request_duration_seconds` - Request latency histogram
- `konsul_http_requests_in_flight` - Current in-flight requests
- `konsul_kv_operations_total` - KV store operations by operation, status
- `konsul_kv_store_size` - Number of keys in KV store
- `konsul_service_operations_total` - Service operations by operation, status
- `konsul_registered_services_total` - Number of registered services
- `konsul_service_heartbeats_total` - Service heartbeats by service, status
- `konsul_expired_services_total` - Total expired services cleaned up
- `konsul_rate_limit_requests_total` - Total rate limit checks by type and status
- `konsul_rate_limit_exceeded_total` - Total rate limit violations by type
- `konsul_rate_limit_active_clients` - Number of active rate limited clients by type
- `konsul_build_info` - Build information (version, Go version)

## Error Handling

All API endpoints return structured error responses with:
- Descriptive error messages
- HTTP status codes
- Request correlation IDs for tracing
- Timestamps for debugging

**Example error response:**
```json
{
  "error": "Not Found",
  "message": "Service not found",
  "request_id": "123e4567-e89b-12d3-a456-426614174000",
  "timestamp": "2025-09-17T10:30:00Z",
  "path": "/services/nonexistent"
}
```

## Deployment

Konsul supports multiple deployment methods:

### Docker

```bash
# Quick start
docker run -d -p 8888:8888 -p 8600:8600/udp konsul:latest

# With persistence and TLS
docker run -d \
  -p 8888:8888 \
  -p 8600:8600/udp \
  -e KONSUL_PERSISTENCE_ENABLED=true \
  -e KONSUL_TLS_ENABLED=true \
  -e KONSUL_TLS_AUTO_CERT=true \
  -v konsul-data:/app/data \
  konsul:latest
```

### Kubernetes

```bash
# Using kubectl
kubectl apply -f k8s/

# Check status
kubectl get pods -n konsul
```

### Helm

```bash
# Install
helm install konsul ./helm/konsul --namespace konsul --create-namespace

# With custom values
helm install konsul ./helm/konsul \
  --namespace konsul \
  --create-namespace \
  --values my-values.yaml

# Upgrade
helm upgrade konsul ./helm/konsul --namespace konsul
```

**See [Deployment Guide](docs/deployment.md) for detailed instructions.**

## Documentation

- [GraphQL API](docs/graphql-api.md) - GraphQL API reference and examples
- [Deployment Guide](docs/deployment.md) - Production deployment instructions
- [Architecture Decision Records (ADRs)](docs/adr/) - Architectural decisions and rationale
- [TODO](docs/TODO.md) - Development roadmap and planned features
