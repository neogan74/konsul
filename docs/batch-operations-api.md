# Batch Operations API

The Batch Operations API allows you to perform multiple KV store and service operations in a single request. This improves performance by reducing network round-trips and enables atomic operations.

## Overview

**Base URL**: `/batch`

**Features**:
- Batch KV operations (get, set, delete)
- Batch KV operations with CAS (set-cas, delete-cas)
- Batch service operations (get, register, deregister)
- Atomic operations with Compare-And-Swap (CAS)
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

#### Batch Set Keys with CAS
Set multiple key-value pairs with Compare-And-Swap (CAS) checks for atomic conditional updates.

```
POST /batch/kv/set-cas
```

**Request Body**:
```json
{
  "items": {
    "config/database/host": "new-db-host",
    "config/database/port": "5433",
    "feature/new_ui": "true"
  },
  "expected_indices": {
    "config/database/host": 42,
    "config/database/port": 43,
    "feature/new_ui": 0
  }
}
```

**Response** (200 OK):
```json
{
  "message": "Successfully set 3 keys with CAS",
  "new_indices": {
    "config/database/host": 45,
    "config/database/port": 46,
    "feature/new_ui": 1
  },
  "count": 3
}
```

**Response** (409 Conflict) - CAS mismatch:
```json
{
  "error": "CAS conflict",
  "message": "CAS conflict for key 'config/database/host': expected ModifyIndex 42, but current is 45"
}
```

**Response** (404 Not Found) - Key doesn't exist when expected:
```json
{
  "error": "Not Found",
  "message": "Key 'config/database/host' not found (expected for CAS update)"
}
```

**CAS Semantics**:
- `expected_indices: 0` - Create-only (key must not exist)
- `expected_indices: N` - Update-only (key must exist with ModifyIndex = N)
- All operations are atomic - if any key fails CAS check, entire batch is rolled back

**Limits**:
- Maximum 1000 items per request
- Items map cannot be empty
- Expected index must be provided for every key in items

**Example - Create-only batch**:
```bash
curl -X POST http://localhost:8500/batch/kv/set-cas \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "config/new_feature": "enabled",
      "config/new_setting": "value"
    },
    "expected_indices": {
      "config/new_feature": 0,
      "config/new_setting": 0
    }
  }'
```

**Example - Conditional update batch**:
```bash
# First, get current indices
curl -X GET http://localhost:8500/kv/config/database/host

# Then update with CAS
curl -X POST http://localhost:8500/batch/kv/set-cas \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "config/database/host": "new-host",
      "config/database/port": "5433"
    },
    "expected_indices": {
      "config/database/host": 42,
      "config/database/port": 43
    }
  }'
```

---

#### Batch Delete Keys with CAS
Delete multiple keys with Compare-And-Swap (CAS) checks for atomic conditional deletes.

```
POST /batch/kv/delete-cas
```

**Request Body**:
```json
{
  "keys": ["temp/cache1", "temp/cache2", "temp/old_config"],
  "expected_indices": {
    "temp/cache1": 10,
    "temp/cache2": 11,
    "temp/old_config": 12
  }
}
```

**Response** (200 OK):
```json
{
  "message": "Successfully deleted 3 keys with CAS",
  "keys": ["temp/cache1", "temp/cache2", "temp/old_config"],
  "count": 3
}
```

**Response** (409 Conflict) - CAS mismatch:
```json
{
  "error": "CAS conflict",
  "message": "CAS conflict for key 'temp/cache1': expected ModifyIndex 10, but current is 15"
}
```

**Response** (404 Not Found) - Key doesn't exist:
```json
{
  "error": "Not Found",
  "message": "Key 'temp/cache1' not found"
}
```

**CAS Semantics**:
- All operations are atomic - if any key fails CAS check, entire batch is rolled back
- Expected index must match current ModifyIndex for each key

**Limits**:
- Maximum 1000 keys per request
- Keys array cannot be empty
- Expected index must be provided for every key in the array

**Example**:
```bash
# First, get current indices for keys to delete
curl -X GET http://localhost:8500/kv/cache/user/123

# Then delete with CAS
curl -X POST http://localhost:8500/batch/kv/delete-cas \
  -H "Content-Type: application/json" \
  -d '{
    "keys": ["cache/user/123", "cache/user/456"],
    "expected_indices": {
      "cache/user/123": 20,
      "cache/user/456": 21
    }
  }'
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

**400 Bad Request** - Missing CAS index:
```json
{
  "error": "Bad Request",
  "message": "Missing expected index for key: config/database"
}
```

**409 Conflict** - CAS check failed:
```json
{
  "error": "CAS conflict",
  "message": "CAS conflict for key 'config/app': expected ModifyIndex 10, but current is 15"
}
```

**404 Not Found** - Key not found for CAS update:
```json
{
  "error": "Not Found",
  "message": "Key 'config/missing' not found (expected for CAS update)"
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
2. **Atomic Operations**: KV batch set/delete are atomic (including CAS operations)
3. **Better Throughput**: Process thousands of items efficiently
4. **Lower Latency**: Eliminate round-trip delays
5. **Strong Consistency**: CAS operations ensure no lost updates in concurrent environments

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
| KV Batch Set CAS | 1000 items | Atomic transaction size |
| KV Batch Delete CAS | 1000 keys | Atomic transaction size |
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

# Batch KV CAS operations
konsul_kv_operations_total{operation="batch_set_cas",status="success"}
konsul_kv_operations_total{operation="batch_set_cas",status="cas_conflict"}
konsul_kv_operations_total{operation="batch_delete_cas",status="success"}
konsul_kv_operations_total{operation="batch_delete_cas",status="cas_conflict"}

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

### 5. Distributed Counter with CAS

Safely update multiple counters without race conditions:

```bash
# Get current values and indices
curl -X GET http://localhost:8500/kv/counter/page_views
curl -X GET http://localhost:8500/kv/counter/api_calls

# Atomically update both counters with CAS
curl -X POST http://localhost:8500/batch/kv/set-cas \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "counter/page_views": "15042",
      "counter/api_calls": "8521"
    },
    "expected_indices": {
      "counter/page_views": 120,
      "counter/api_calls": 95
    }
  }'
```

### 6. Leader Election / Distributed Lock

Implement distributed locking with create-only CAS:

```bash
# Try to acquire multiple locks atomically
curl -X POST http://localhost:8500/batch/kv/set-cas \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "locks/resource-1": "node-123",
      "locks/resource-2": "node-123",
      "locks/resource-3": "node-123"
    },
    "expected_indices": {
      "locks/resource-1": 0,
      "locks/resource-2": 0,
      "locks/resource-3": 0
    }
  }'

# Release locks with CAS (only if you still hold them)
curl -X POST http://localhost:8500/batch/kv/delete-cas \
  -H "Content-Type: application/json" \
  -d '{
    "keys": ["locks/resource-1", "locks/resource-2", "locks/resource-3"],
    "expected_indices": {
      "locks/resource-1": 1,
      "locks/resource-2": 1,
      "locks/resource-3": 1
    }
  }'
```

### 7. Atomic Configuration Migration

Migrate configuration keys safely with CAS to ensure no concurrent modifications:

```bash
# Read old config and get indices
curl -X GET http://localhost:8500/kv/old/config/db
curl -X GET http://localhost:8500/kv/old/config/cache

# Atomically create new config and delete old config
# First, create new config with CAS
curl -X POST http://localhost:8500/batch/kv/set-cas \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "config/v2/database": "postgres://new-host:5432/db",
      "config/v2/cache": "redis://new-cache:6379"
    },
    "expected_indices": {
      "config/v2/database": 0,
      "config/v2/cache": 0
    }
  }'

# Then, delete old config with CAS (ensures no changes during migration)
curl -X POST http://localhost:8500/batch/kv/delete-cas \
  -H "Content-Type: application/json" \
  -d '{
    "keys": ["old/config/db", "old/config/cache"],
    "expected_indices": {
      "old/config/db": 42,
      "old/config/cache": 38
    }
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

// Batch Set with CAS
type BatchKVSetCASRequest struct {
    Items           map[string]string  `json:"items"`
    ExpectedIndices map[string]uint64  `json:"expected_indices"`
}

func BatchSetKVWithCAS(items map[string]string, indices map[string]uint64) error {
    req := BatchKVSetCASRequest{
        Items:           items,
        ExpectedIndices: indices,
    }
    body, _ := json.Marshal(req)

    resp, err := http.Post(
        "http://localhost:8500/batch/kv/set-cas",
        "application/json",
        bytes.NewReader(body),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode == 409 {
        return fmt.Errorf("CAS conflict: concurrent modification detected")
    }

    return nil
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

// Batch Set with CAS
async function batchSetKVWithCAS(items, expectedIndices) {
  const response = await fetch('http://localhost:8500/batch/kv/set-cas', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ items, expected_indices: expectedIndices }),
  });

  if (response.status === 409) {
    throw new Error('CAS conflict: concurrent modification detected');
  }

  if (!response.ok) {
    throw new Error(`Batch set CAS failed: ${response.statusText}`);
  }

  return response.json();
}

// Usage with CAS (create-only)
try {
  const result = await batchSetKVWithCAS(
    {
      'locks/resource-1': 'node-123',
      'locks/resource-2': 'node-123',
    },
    {
      'locks/resource-1': 0,  // Create only
      'locks/resource-2': 0,  // Create only
    }
  );
  console.log(`Acquired ${result.count} locks`);
} catch (error) {
  console.error('Failed to acquire locks:', error.message);
}
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

# Batch Set with CAS
def batch_set_kv_cas(items: dict, expected_indices: dict) -> dict:
    response = requests.post(
        'http://localhost:8500/batch/kv/set-cas',
        json={
            'items': items,
            'expected_indices': expected_indices
        }
    )

    if response.status_code == 409:
        raise Exception('CAS conflict: concurrent modification detected')

    response.raise_for_status()
    return response.json()

# Usage with CAS (conditional update)
try:
    result = batch_set_kv_cas(
        items={
            'config/database/host': 'new-host',
            'config/database/port': '5433'
        },
        expected_indices={
            'config/database/host': 42,
            'config/database/port': 43
        }
    )
    print(f"Updated {result['count']} keys with CAS")
except Exception as e:
    print(f"CAS update failed: {e}")
```

---

## Testing

Run batch operations tests:

```bash
# Run all batch tests (20 tests total)
go test -v ./internal/handlers -run TestBatch

# Run specific test
go test -v ./internal/handlers -run TestBatchKVSet_Success

# Run CAS-specific tests
go test -v ./internal/handlers -run TestBatchKV.*CAS
```

**Test Coverage**:
- Batch KV Get: 3 tests
- Batch KV Set: 2 tests
- Batch KV Delete: 2 tests
- Batch KV Set CAS: 4 tests (create-only, conditional update, conflict, missing indices)
- Batch KV Delete CAS: 3 tests (success, conflict, missing indices)
- Batch Service operations: 6 tests

---

## See Also

- [CAS Operations Guide](./CAS.md) - Complete guide to Compare-And-Swap operations
- [KV Store API](./kv-store-api.md) - Single KV operations
- [Service Discovery API](./service-discovery-api.md) - Single service operations
- [Audit Logging](./audit-logging.md) - Audit events for batch operations
- [Authentication Guide](./authentication.md) - JWT and API key authentication
