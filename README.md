# Konsul

> [!WARNING]
> In active development

**Konsul** is a lightweight, cloud-native service mesh and discovery platform built in Go. It provides essential infrastructure services for distributed systems including service registration and discovery, distributed key-value storage, health monitoring, and DNS-based service resolution.

## Overview

Konsul helps microservices find and communicate with each other in dynamic cloud environments. It offers:

- **Service Discovery** - Register services and discover them via REST API or DNS
- **Health Checking** - Automatic health monitoring with configurable TTL
- **KV Store** - Distributed configuration storage with RESTful API
- **DNS Interface** - Service discovery via standard DNS queries
- **Authentication** - JWT and API key-based authentication
- **Access Control** - Fine-grained ACL system for authorization
- **GraphQL API** - Flexible querying alongside REST endpoints
- **Admin UI** - Modern React-based web interface
- **Metrics & Monitoring** - Prometheus integration with health endpoints
- **Rate Limiting** - Token bucket algorithm for API protection
- **TLS Support** - Encrypted communication with mutual TLS
- **Persistence** - BadgerDB backend for data durability
- **Backup & Restore** - Built-in backup management

Konsul is designed to be simple to deploy, easy to operate, and production-ready out of the box.

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

## Web Admin UI

Konsul includes a built-in **React-based web interface** for managing services and the KV store through an intuitive dashboard.

### Features

- **Service Discovery Dashboard** - View all registered services with real-time health status
- **KV Store Browser** - Browse, create, update, and delete key-value pairs
- **Health Monitoring** - Visual health check status and history
- **Metrics Visualization** - System and service metrics (when Prometheus is enabled)
- **Backup Management** - Create and restore backups through the UI
- **Modern UX** - Built with React 19, Vite, and Tailwind CSS v4

### Accessing the UI

The Admin UI is **enabled by default** and accessible at:

```
http://localhost:8888/admin
```

### Configuration

```bash
# Enable/disable the Admin UI (default: true)
export KONSUL_ADMIN_UI_ENABLED=true

# Change the base path (default: /admin)
export KONSUL_ADMIN_UI_PATH=/admin
```

### Security Features

The Admin UI includes production-ready security features:

- **Security Headers** - XSS protection, content security policy, frame options
- **Compression** - Gzip compression for all assets
- **Caching** - Optimized cache headers (1-year cache for hashed assets)
- **CORS Support** - Configurable cross-origin resource sharing
- **Authentication** - API calls from UI support JWT and API key auth

### Screenshots

The UI provides:
- Dashboard with service overview
- Service list with health indicators
- KV store editor with JSON support
- Real-time updates and notifications

### Documentation

For complete documentation, see:
- [Admin UI User Guide](docs/admin-ui.md)
- [Architecture Decision (ADR-0009)](docs/adr/0009-react-admin-ui.md)
- [Implementation Plan](docs/admin-ui-integration-plan.md)

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

### Access Control Lists (ACL)

Konsul implements a fine-grained **ACL system** for authorization, allowing you to control access to resources (KV store, services, health checks, backups, admin operations) using policies.

#### Features

- **Fine-grained permissions** - Control access at the resource level
- **Path-based rules** - Use wildcards (`*`, `**`) for flexible matching
- **Policy composition** - Attach multiple policies to tokens
- **Deny-by-default** - Secure by default security model
- **Explicit deny** - Block specific resources explicitly
- **File-based policies** - Store policies as JSON files

#### Quick Start

**1. Enable ACL system:**
```yaml
# config.yaml
acl:
  enabled: true
  policy_dir: ./policies
```

**2. Create a policy** (`policies/developer.json`):
```json
{
  "name": "developer",
  "description": "Developer access",
  "kv": [
    {
      "path": "app/config/*",
      "capabilities": ["read", "list"]
    }
  ],
  "service": [
    {
      "name": "web-*",
      "capabilities": ["read", "register", "deregister"]
    }
  ]
}
```

**3. Load policy:**
```bash
konsulctl acl policy create policies/developer.json
```

**4. Generate token with policies:**
```bash
curl -X POST http://localhost:8888/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "username": "alice",
    "roles": ["developer"],
    "policies": ["developer", "readonly"]
  }'
```

#### CLI Commands

```bash
# Policy management
konsulctl acl policy list
konsulctl acl policy get <name>
konsulctl acl policy create <file>
konsulctl acl policy update <file>
konsulctl acl policy delete <name>

# Test permissions
konsulctl acl test developer kv app/config read
```

#### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/acl/policies` | List all policies |
| GET | `/acl/policies/:name` | Get policy details |
| POST | `/acl/policies` | Create new policy |
| PUT | `/acl/policies/:name` | Update policy |
| DELETE | `/acl/policies/:name` | Delete policy |
| POST | `/acl/test` | Test ACL permissions |

#### Documentation

- [Complete ACL Guide](docs/acl.md) - Full documentation
- [Policy Examples](policies/README.md) - Pre-built policy templates
- [ADR-0010](docs/adr/0010-acl-system.md) - Architecture decision

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

## Web Admin UI

Konsul includes a modern web-based Admin UI built with React 19, Vite, and Tailwind CSS v4. The UI is embedded in the binary and served directly by the Go server.

### Features

- **Dashboard**: System overview with health metrics and statistics
- **Service Management**: Browse, register, and manage services
- **KV Store Browser**: View and edit key-value pairs
- **Real-time Updates**: Live service status and health checks
- **Modern Design**: Responsive interface with Tailwind CSS v4

### Access the Admin UI

Once Konsul is running, access the Admin UI at:
```
http://localhost:8888/admin
```

The root URL (`http://localhost:8888/`) automatically redirects to the Admin UI.

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_ADMIN_UI_ENABLED` | `true` | Enable/disable Admin UI |
| `KONSUL_ADMIN_UI_PATH` | `/admin` | Base path for Admin UI |

**Examples:**
```bash
# Disable Admin UI
KONSUL_ADMIN_UI_ENABLED=false ./konsul

# Custom UI path
KONSUL_ADMIN_UI_PATH=/ui ./konsul
```

### Development

The Admin UI source is located in `web/admin/`. To rebuild the UI:

```bash
# Build the UI
cd web/admin
npm run build

# Copy to embed location
cd ../..
cp -r web/admin/dist cmd/konsul/ui

# Rebuild Konsul
go build -o konsul ./cmd/konsul
```

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

### Admin UI Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_ADMIN_UI_ENABLED` | `true` | Enable Admin UI |
| `KONSUL_ADMIN_UI_PATH` | `/admin` | Base path for Admin UI |

### Audit Logging Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_AUDIT_ENABLED` | `false` | Enable the audit logging subsystem |
| `KONSUL_AUDIT_SINK` | `file` | Destination for audit events (`file` or `stdout`) |
| `KONSUL_AUDIT_FILE_PATH` | `./logs/audit.log` | Path to the audit log when `sink=file` |
| `KONSUL_AUDIT_BUFFER_SIZE` | `1024` | Channel size for pending audit events |
| `KONSUL_AUDIT_FLUSH_INTERVAL` | `1s` | Interval for flushing buffered events |
| `KONSUL_AUDIT_DROP_POLICY` | `drop` | Behavior when the buffer is full (`drop` or `block`) |

> Audit logging plumbing is in active development (see `docs/adr/0019-audit-logging.md`); future phases will emit events for each privileged operation.

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

### Management API

The rate limiting system includes a comprehensive management API for runtime control:

**Endpoints:**
- `GET /admin/ratelimit/stats` - View statistics and configuration
- `GET /admin/ratelimit/clients` - List active rate-limited clients
- `GET /admin/ratelimit/client/:identifier` - Get specific client status
- `POST /admin/ratelimit/reset/ip/:ip` - Reset IP rate limit
- `POST /admin/ratelimit/reset/apikey/:key_id` - Reset API key rate limit
- `POST /admin/ratelimit/reset/all` - Reset all rate limits
- `PUT /admin/ratelimit/config` - Update global configuration
- `PUT /admin/ratelimit/client/:type/:id` - Adjust client-specific limits
- `GET /admin/ratelimit/whitelist` - List whitelisted clients
- `POST /admin/ratelimit/whitelist` - Add to whitelist
- `DELETE /admin/ratelimit/whitelist/:identifier` - Remove from whitelist
- `GET /admin/ratelimit/blacklist` - List blacklisted clients
- `POST /admin/ratelimit/blacklist` - Add to blacklist
- `DELETE /admin/ratelimit/blacklist/:identifier` - Remove from blacklist

### CLI Commands

**konsulctl** provides comprehensive rate limiting management:

```bash
# View statistics
konsulctl ratelimit stats
konsulctl ratelimit config

# List active clients
konsulctl ratelimit clients
konsulctl ratelimit clients --type ip
konsulctl ratelimit clients --type apikey

# Get client status
konsulctl ratelimit client 192.168.1.100

# Reset rate limits
konsulctl ratelimit reset ip 192.168.1.100
konsulctl ratelimit reset apikey key-abc-123
konsulctl ratelimit reset all --type ip

# Update global configuration
konsulctl ratelimit update --rate 200 --burst 50

# Adjust client-specific limits (temporary)
konsulctl ratelimit adjust --type ip --id 192.168.1.100 \
  --rate 500 --burst 100 --duration 1h

# Whitelist management
konsulctl ratelimit whitelist list
konsulctl ratelimit whitelist add --id 10.0.1.10 --type ip \
  --reason "Internal monitoring" --duration 24h
konsulctl ratelimit whitelist remove 10.0.1.10

# Blacklist management
konsulctl ratelimit blacklist list
konsulctl ratelimit blacklist add --id 203.0.113.50 --type ip \
  --reason "Malicious activity" --duration 24h
konsulctl ratelimit blacklist remove 203.0.113.50
```

**Documentation:**
- [ADR-0013: Token Bucket Rate Limiting](docs/adr/0013-token-bucket-rate-limiting.md)
- [ADR-0014: Rate Limiting Management API](docs/adr/0014-rate-limiting-management-api.md)
- [ADR-0022: Comprehensive Testing Strategy](docs/adr/0022-rate-limiting-comprehensive-testing.md)

## Testing & Quality Assurance

Konsul maintains high code quality through comprehensive testing across all components.

### Test Coverage

```
Package: internal/ratelimit     Coverage: 86.8%    Tests: 44
Package: internal/handlers      Coverage: 41.5%    Tests: 35 (ratelimit)
Package: internal/middleware    Coverage: 84.5%    Tests: 15 (ratelimit)

Total Rate Limiting Tests: 94
Average Execution Time: ~3.0s
```

### Test Categories

**Unit Tests (53 tests)**:
- Token bucket algorithm
- Access lists (whitelist/blacklist)
- Custom rate configurations
- Expiry logic
- Validation and error handling
- RFC 6585 headers
- Statistics and violation tracking

**Handler Tests (35 tests)**:
- All admin API endpoints
- Whitelist/blacklist management
- Client limit adjustments
- Configuration updates
- Error scenarios and validation

**Integration Tests (8 tests)**:
- Service-level operations
- Multi-store interactions
- Cross-component workflows
- End-to-end scenarios

### Running Tests

```bash
# Run all tests
go test ./...

# Run rate limiting tests with coverage
go test ./internal/ratelimit/... -cover

# Run specific test suite
go test -v ./internal/handlers/ratelimit_test.go

# Run with verbose output
go test -v ./internal/ratelimit/...
```

### Documentation

- [ADR-0022: Comprehensive Testing Strategy](docs/adr/0022-rate-limiting-comprehensive-testing.md) - Complete testing documentation

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
