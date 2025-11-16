# Batch Operations API

The Batch Operations API allows you to perform multiple KV store and service operations in a single request. This improves performance by reducing network round-trips and enables atomic operations.

## Overview

**Base URL**: `/batch`

**Features**:
- Batch KV operations (get, set, delete)
- Batch service operations (get, register, deregister)
- Request validation and size limits
- Partial success handling for service operations
- Audit logging integration
- Prometheus metrics

## Endpoints

### KV Store Operations

#### Batch Get Keys
Retrieve multiple keys in a single request.

```
POST /batch/kv/get
```

**Request Body**:
```json
{
  "keys": ["config/app", "config/db", "config/cache"]
}
```

**Response** (200 OK):
```json
{
  "found": {
    "config/app": "{\"name\":\"myapp\",\"version\":\"1.0\"}",
    "config/db": "postgres://localhost:5432/mydb"
  },
  "not_found": ["config/cache"]
}
```

**Limits**:
- Maximum 1000 keys per request
- Keys array cannot be empty

**Example**:
```bash
curl -X POST http://localhost:8500/batch/kv/get \
  -H "Content-Type: application/json" \
  -d '{"keys": ["app/config", "app/secrets", "app/features"]}'
```

---

#### Batch Set Keys
Set multiple key-value pairs atomically.

```
POST /batch/kv/set
```

**Request Body**:
```json
{
  "items": {
    "config/database/host": "localhost",
    "config/database/port": "5432",
    "config/database/name": "mydb",
    "config/database/pool_size": "10"
  }
}
```

**Response** (200 OK):
```json
{
  "message": "Successfully set 4 keys",
  "keys": ["config/database/host", "config/database/port", "config/database/name", "config/database/pool_size"],
  "count": 4
}
```

**Limits**:
- Maximum 1000 items per request
- Items map cannot be empty

**Example**:
```bash
curl -X POST http://localhost:8500/batch/kv/set \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "feature/dark_mode": "true",
      "feature/beta_ui": "false",
      "feature/rate_limit": "100"
    }
  }'
```

---

#### Batch Delete Keys
Delete multiple keys atomically.

```
POST /batch/kv/delete
```

**Request Body**:
```json
{
  "keys": ["temp/cache1", "temp/cache2", "temp/old_config"]
}
```

**Response** (200 OK):
```json
{
  "message": "Successfully deleted 3 keys",
  "keys": ["temp/cache1", "temp/cache2", "temp/old_config"],
  "count": 3
}
```

**Limits**:
- Maximum 1000 keys per request
- Keys array cannot be empty

**Example**:
```bash
curl -X POST http://localhost:8500/batch/kv/delete \
  -H "Content-Type: application/json" \
  -d '{"keys": ["cache/user/123", "cache/user/456", "cache/session/expired"]}'
```

---

### Service Operations

#### Batch Get Services
Retrieve multiple services by name.

```
POST /batch/services/get
```

**Request Body**:
```json
{
  "names": ["web-frontend", "api-gateway", "auth-service"]
}
```

**Response** (200 OK):
```json
{
  "found": {
    "web-frontend": {
      "name": "web-frontend",
      "address": "10.0.1.10",
      "port": 3000,
      "tags": ["web", "frontend", "v2"],
      "meta": {
        "version": "2.1.0",
        "env": "production"
      }
    },
    "api-gateway": {
      "name": "api-gateway",
      "address": "10.0.1.20",
      "port": 8080,
      "tags": ["api", "gateway"],
      "meta": {
        "version": "1.5.0"
      }
    }
  },
  "not_found": ["auth-service"]
}
```

**Limits**:
- Maximum 100 services per request
- Names array cannot be empty

**Example**:
```bash
curl -X POST http://localhost:8500/batch/services/get \
  -H "Content-Type: application/json" \
  -d '{"names": ["redis", "postgres", "elasticsearch"]}'
```

---

#### Batch Register Services
Register multiple services at once.

```
POST /batch/services/register
```

**Request Body**:
```json
{
  "services": [
    {
      "name": "web-1",
      "address": "10.0.1.101",
      "port": 3000,
      "tags": ["web", "frontend"],
      "meta": {
        "version": "1.0.0",
        "datacenter": "dc1"
      }
    },
    {
      "name": "web-2",
      "address": "10.0.1.102",
      "port": 3000,
      "tags": ["web", "frontend"],
      "meta": {
        "version": "1.0.0",
        "datacenter": "dc1"
      }
    },
    {
      "name": "api-1",
      "address": "10.0.1.201",
      "port": 8080,
      "tags": ["api", "backend"]
    }
  ]
}
```

**Response** (200 OK):
```json
{
  "message": "Registered 3 services",
  "registered": ["web-1", "web-2", "api-1"],
  "count": 3
}
```

**Response with failures** (200 OK):
```json
{
  "message": "Registered 2 services, 1 failed",
  "registered": ["web-1", "web-2"],
  "failed": ["invalid-service"],
  "count": 2
}
```

**Validation**:
- Service name is required
- Address is required
- Port must be between 1 and 65535
- Tags and metadata are optional

**Limits**:
- Maximum 100 services per request
- Services array cannot be empty

**Example**:
```bash
curl -X POST http://localhost:8500/batch/services/register \
  -H "Content-Type: application/json" \
  -d '{
    "services": [
      {"name": "cache-1", "address": "10.0.2.1", "port": 6379, "tags": ["cache", "redis"]},
      {"name": "cache-2", "address": "10.0.2.2", "port": 6379, "tags": ["cache", "redis"]},
      {"name": "cache-3", "address": "10.0.2.3", "port": 6379, "tags": ["cache", "redis"]}
    ]
  }'
```

---

#### Batch Deregister Services
Deregister multiple services at once.

```
POST /batch/services/deregister
```

**Request Body**:
```json
{
  "names": ["web-old-1", "web-old-2", "deprecated-api"]
}
```

**Response** (200 OK):
```json
{
  "message": "Deregistered 3 services",
  "deregistered": ["web-old-1", "web-old-2", "deprecated-api"],
  "count": 3
}
```

**Limits**:
- Maximum 100 services per request
- Names array cannot be empty

**Example**:
```bash
curl -X POST http://localhost:8500/batch/services/deregister \
  -H "Content-Type: application/json" \
  -d '{"names": ["old-service-1", "old-service-2"]}'
```

---

## Error Handling

### Common Errors

**400 Bad Request** - Invalid input:
```json
{
  "error": "Bad Request",
  "message": "Keys array cannot be empty"
}
```

**400 Bad Request** - Limit exceeded:
```json
{
  "error": "Bad Request",
  "message": "Maximum 1000 keys per batch request"
}
```

**400 Bad Request** - Invalid JSON:
```json
{
  "error": "Bad Request",
  "message": "Invalid JSON body"
}
```

**500 Internal Server Error** - Storage failure:
```json
{
  "error": "Internal Server Error",
  "message": "Failed to set keys"
}
```

---

## Performance Considerations

### Benefits of Batch Operations

1. **Reduced Network Overhead**: Single HTTP request instead of multiple
2. **Atomic Operations**: KV batch set/delete are atomic
3. **Better Throughput**: Process thousands of items efficiently
4. **Lower Latency**: Eliminate round-trip delays

### Best Practices

1. **Chunk Large Operations**: Split very large batches into manageable chunks
   ```javascript
   // Instead of 10000 keys at once
   for (let i = 0; i < keys.length; i += 1000) {
     await batchSet(keys.slice(i, i + 1000));
   }
   ```

2. **Handle Partial Failures**: Service registration can have partial success
   ```javascript
   const result = await batchRegisterServices(services);
   if (result.failed && result.failed.length > 0) {
     console.error("Failed to register:", result.failed);
   }
   ```

3. **Monitor Metrics**: Track batch operation performance
   ```promql
   rate(konsul_kv_operations_total{operation="batch_set"}[5m])
   ```

### Size Limits

| Operation | Limit | Reason |
|-----------|-------|--------|
| KV Batch Get | 1000 keys | Memory efficiency |
| KV Batch Set | 1000 items | Atomic transaction size |
| KV Batch Delete | 1000 keys | Atomic transaction size |
| Service Batch Get | 100 services | Response size |
| Service Batch Register | 100 services | Validation overhead |
| Service Batch Deregister | 100 services | Cleanup overhead |

---

## Authentication & Authorization

When authentication is enabled, include the appropriate headers:

**JWT Authentication**:
```bash
curl -X POST http://localhost:8500/batch/kv/set \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"items": {"key1": "value1"}}'
```

**API Key Authentication**:
```bash
curl -X POST http://localhost:8500/batch/kv/set \
  -H "X-API-Key: <api_key>" \
  -H "Content-Type: application/json" \
  -d '{"items": {"key1": "value1"}}'
```

### ACL Permissions

Batch operations require the same permissions as individual operations:
- KV operations require appropriate `kv` resource permissions
- Service operations require `service` resource permissions

---

## Audit Logging

All batch operations are automatically logged to the audit system (when enabled):

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2025-11-16T12:00:00Z",
  "action": "batch.create",
  "result": "success",
  "resource": {
    "type": "batch"
  },
  "actor": {
    "id": "user-123",
    "type": "user"
  },
  "http_method": "POST",
  "http_path": "/batch/kv/set",
  "http_status": 200
}
```

---

## Metrics

Monitor batch operation performance with Prometheus metrics:

```promql
# Batch KV operations
konsul_kv_operations_total{operation="batch_get"}
konsul_kv_operations_total{operation="batch_set"}
konsul_kv_operations_total{operation="batch_delete"}

# Batch service operations
konsul_service_operations_total{operation="batch_get"}
konsul_service_operations_total{operation="batch_register"}
konsul_service_operations_total{operation="batch_deregister"}
```

---

## Use Cases

### 1. Microservice Deployment

Register multiple service instances during deployment:

```bash
curl -X POST http://localhost:8500/batch/services/register \
  -H "Content-Type: application/json" \
  -d '{
    "services": [
      {"name": "api-v2-1", "address": "10.0.1.1", "port": 8080, "tags": ["api", "v2"]},
      {"name": "api-v2-2", "address": "10.0.1.2", "port": 8080, "tags": ["api", "v2"]},
      {"name": "api-v2-3", "address": "10.0.1.3", "port": 8080, "tags": ["api", "v2"]}
    ]
  }'
```

### 2. Configuration Management

Update multiple configuration values atomically:

```bash
curl -X POST http://localhost:8500/batch/kv/set \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "config/db/primary": "postgres://primary:5432",
      "config/db/replica": "postgres://replica:5432",
      "config/db/pool_size": "50",
      "config/db/timeout": "30s"
    }
  }'
```

### 3. Cache Invalidation

Delete multiple cache entries:

```bash
curl -X POST http://localhost:8500/batch/kv/delete \
  -H "Content-Type: application/json" \
  -d '{
    "keys": [
      "cache/user/123",
      "cache/user/456",
      "cache/products/list",
      "cache/categories"
    ]
  }'
```

### 4. Service Discovery

Query multiple services for load balancing:

```bash
curl -X POST http://localhost:8500/batch/services/get \
  -H "Content-Type: application/json" \
  -d '{
    "names": ["web", "api", "db", "cache", "queue"]
  }'
```

---

## Client Libraries

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type BatchKVSetRequest struct {
    Items map[string]string `json:"items"`
}

func BatchSetKV(items map[string]string) error {
    req := BatchKVSetRequest{Items: items}
    body, _ := json.Marshal(req)

    resp, err := http.Post(
        "http://localhost:8500/batch/kv/set",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}

func main() {
    items := map[string]string{
        "config/app": `{"name":"myapp"}`,
        "config/db":  "postgres://localhost:5432",
    }

    if err := BatchSetKV(items); err != nil {
        panic(err)
    }
}
```

### JavaScript

```javascript
async function batchSetKV(items) {
  const response = await fetch('http://localhost:8500/batch/kv/set', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ items }),
  });

  if (!response.ok) {
    throw new Error(`Batch set failed: ${response.statusText}`);
  }

  return response.json();
}

// Usage
const result = await batchSetKV({
  'config/app': '{"name":"myapp"}',
  'config/db': 'postgres://localhost:5432',
});
console.log(`Set ${result.count} keys`);
```

### Python

```python
import requests

def batch_set_kv(items: dict) -> dict:
    response = requests.post(
        'http://localhost:8500/batch/kv/set',
        json={'items': items}
    )
    response.raise_for_status()
    return response.json()

# Usage
result = batch_set_kv({
    'config/app': '{"name":"myapp"}',
    'config/db': 'postgres://localhost:5432',
})
print(f"Set {result['count']} keys")
```

---

## Testing

Run batch operations tests:

```bash
# Run all batch tests
go test -v ./internal/handlers -run TestBatch

# Run specific test
go test -v ./internal/handlers -run TestBatchKVSet_Success
```

---

## See Also

- [KV Store API](./kv-store-api.md) - Single KV operations
- [Service Discovery API](./service-discovery-api.md) - Single service operations
- [Audit Logging](./audit-logging.md) - Audit events for batch operations
- [Authentication Guide](./authentication.md) - JWT and API key authentication
