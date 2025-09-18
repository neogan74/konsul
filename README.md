# Konsul service

In development now :> [!WARNING]

## KV storage (map with mutex)

| Method | endpoint  | Description  |
| ------ | --------- | ------------ |
| PUT    | /kv/<key> | Write value  |
| GET    | /kv/<key> | Read value   |
| DELETE | /kv/<key> | Delete value |

## Service Discovery (map)

| Method | endpoint  | Description                                |
| ------ | --------- | ------------                               |
| PUT    | /register | service registration                       |
| GET    | /services/ | get all registered services in JSON       |
| GET    | /services/<name> | get service with given name in JSON |
| DELETE | /deregister/<name> | deregister service                |
| PUT    | /heartbeat/<name> | update service TTL                |

#### Example:
```
PUT /register
{
  "name": "auth-service",
  "address": "10.0.0.1",
  "port": 8080
}
```

### Health Check TTL ✅

- Registration sets configurable TTL for service (default: 30s)
- TTL updated through `/heartbeat/<name>` endpoint
- Background process runs at configurable interval removing expired services (default: 60s)
- Services automatically expire if no heartbeat received within TTL

## Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_PORT` | `8888` | Server port |
| `KONSUL_HOST` | `` | Server host (empty = all interfaces) |
| `KONSUL_SERVICE_TTL` | `30s` | Service TTL duration |
| `KONSUL_CLEANUP_INTERVAL` | `60s` | Cleanup interval |
| `KONSUL_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `KONSUL_LOG_FORMAT` | `text` | Log format (text/json) |

**Examples:**
```bash
# Custom port
KONSUL_PORT=9999 ./konsul

# Production settings with JSON logging
KONSUL_HOST=0.0.0.0 KONSUL_PORT=80 KONSUL_LOG_FORMAT=json KONSUL_LOG_LEVEL=info ./konsul

# Debug mode with verbose logging
KONSUL_LOG_LEVEL=debug KONSUL_LOG_FORMAT=text ./konsul
```

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
