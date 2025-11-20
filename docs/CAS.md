# Compare-And-Swap (CAS) Operations

## Overview

Konsul implements atomic Compare-And-Swap (CAS) operations for both KV store and service registry. CAS operations enable safe concurrent modifications by ensuring updates only succeed when the resource hasn't been modified since it was last read.

## Key Concepts

### ModifyIndex
Every KV entry and service registration has a `ModifyIndex` - a monotonically increasing counter that changes with each modification. This index is used to detect concurrent modifications.

### CreateIndex
Records when the resource was first created. This index never changes, even when the resource is updated.

### CAS Semantics

**Create-Only (CAS=0):**
- Only succeeds if the key/service doesn't exist
- Useful for distributed locking or ensuring unique creation

**Conditional Update (CAS=N):**
- Only succeeds if the current ModifyIndex matches N
- Prevents lost updates from concurrent modifications
- Returns conflict error if the index has changed

## KV Store CAS Operations

### Get with Metadata

Retrieve a key with version information:

```bash
# GET /kv/:key?metadata=true
curl http://localhost:8500/kv/my-key?metadata=true
```

Response:
```json
{
  "key": "my-key",
  "value": "my-value",
  "modify_index": 42,
  "create_index": 10,
  "flags": 0
}
```

### Set with CAS

**Create-Only (CAS=0):**
```bash
# Only creates if key doesn't exist
curl -X PUT http://localhost:8500/kv/my-key \
  -H "Content-Type: application/json" \
  -d '{"value": "initial-value", "cas": 0}'
```

**Conditional Update:**
```bash
# Only updates if ModifyIndex is 42
curl -X PUT http://localhost:8500/kv/my-key \
  -H "Content-Type: application/json" \
  -d '{"value": "updated-value", "cas": 42}'
```

Success Response (200):
```json
{
  "message": "key set",
  "key": "my-key",
  "modify_index": 43
}
```

Conflict Response (409):
```json
{
  "error": "CAS conflict",
  "message": "CAS conflict for key 'my-key': expected ModifyIndex 42, but current is 50"
}
```

### Delete with CAS

```bash
# Only deletes if ModifyIndex is 50
curl -X DELETE "http://localhost:8500/kv/my-key?cas=50"
```

### Set with Flags

```bash
curl -X PUT http://localhost:8500/kv/my-key \
  -H "Content-Type: application/json" \
  -d '{"value": "my-value", "flags": 42}'
```

## Service Registry CAS Operations

### Get Service with Metadata

```bash
# GET /services/:name?metadata=true
curl http://localhost:8500/services/web-api?metadata=true
```

Response:
```json
{
  "service": {
    "name": "web-api",
    "address": "192.168.1.10",
    "port": 8080,
    "tags": ["production", "v2"]
  },
  "expires_at": "2025-11-20T12:00:00Z",
  "modify_index": 15,
  "create_index": 5
}
```

### Register with CAS

**Create-Only (CAS=0):**
```bash
# Only registers if service doesn't exist
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-api",
    "address": "192.168.1.10",
    "port": 8080,
    "tags": ["production"],
    "cas": 0
  }'
```

**Conditional Update:**
```bash
# Only updates if ModifyIndex is 15
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-api",
    "address": "192.168.1.10",
    "port": 9090,
    "tags": ["production", "v2"],
    "cas": 15
  }'
```

Success Response (200):
```json
{
  "message": "service registered",
  "service": {
    "name": "web-api",
    "address": "192.168.1.10",
    "port": 9090,
    "tags": ["production", "v2"]
  },
  "modify_index": 16
}
```

### Deregister with CAS

```bash
# Only deregisters if ModifyIndex is 16
curl -X DELETE "http://localhost:8500/services/web-api?cas=16"
```

## Batch Operations with CAS

### Batch Set with CAS

```go
items := map[string]string{
    "key1": "new-value1",
    "key2": "new-value2",
    "key3": "value3", // New key
}
expectedIndices := map[string]uint64{
    "key1": 10, // Must match current index
    "key2": 20, // Must match current index
    "key3": 0,  // Create-only
}

newIndices, err := kvStore.BatchSetCAS(items, expectedIndices)
if err != nil {
    // All-or-nothing: no keys are modified on conflict
    if store.IsCASConflict(err) {
        fmt.Println("CAS conflict:", err)
    }
}
```

### Batch Delete with CAS

```go
keys := []string{"key1", "key2", "key3"}
expectedIndices := map[string]uint64{
    "key1": 11,
    "key2": 21,
    "key3": 5,
}

err := kvStore.BatchDeleteCAS(keys, expectedIndices)
if err != nil {
    // All-or-nothing: no keys are deleted on conflict
    if store.IsCASConflict(err) {
        fmt.Println("CAS conflict:", err)
    }
}
```

## Error Handling

### CAS Conflict Error

HTTP Status: `409 Conflict`

```json
{
  "error": "CAS conflict",
  "message": "CAS conflict for key 'my-key': expected ModifyIndex 42, but current is 50"
}
```

Check in code:
```go
if store.IsCASConflict(err) {
    // Retry with updated index
}
```

### Not Found Error

HTTP Status: `404 Not Found`

Occurs when attempting CAS update (non-zero index) on non-existent resource.

```go
if store.IsNotFound(err) {
    // Resource doesn't exist
}
```

## Use Cases

### Distributed Locking

```bash
# Acquire lock (create-only)
curl -X PUT http://localhost:8500/kv/locks/my-resource \
  -H "Content-Type: application/json" \
  -d '{"value": "owner-id-123", "cas": 0}'

# Release lock (delete with CAS)
curl -X DELETE "http://localhost:8500/kv/locks/my-resource?cas=1"
```

### Safe Configuration Updates

```bash
# Read current config
RESPONSE=$(curl http://localhost:8500/kv/config/app?metadata=true)
CURRENT_INDEX=$(echo $RESPONSE | jq -r '.modify_index')

# Update only if unchanged
curl -X PUT http://localhost:8500/kv/config/app \
  -H "Content-Type: application/json" \
  -d "{\"value\": \"new-config\", \"cas\": $CURRENT_INDEX}"
```

### Service Registration with Version Control

```bash
# Get current service state
RESPONSE=$(curl http://localhost:8500/services/web-api?metadata=true)
CURRENT_INDEX=$(echo $RESPONSE | jq -r '.modify_index')

# Update service configuration
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"web-api\",
    \"address\": \"192.168.1.10\",
    \"port\": 9090,
    \"cas\": $CURRENT_INDEX
  }"
```

### Optimistic Concurrency Control

```go
func UpdateKeyWithRetry(store *store.KVStore, key string, updateFn func(string) string) error {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        // Read current value and index
        entry, ok := store.GetEntry(key)
        if !ok {
            return fmt.Errorf("key not found")
        }

        // Apply transformation
        newValue := updateFn(entry.Value)

        // Try CAS update
        _, err := store.SetCAS(key, newValue, entry.ModifyIndex)
        if err == nil {
            return nil // Success
        }

        if !store.IsCASConflict(err) {
            return err // Non-conflict error
        }

        // CAS conflict - retry with backoff
        time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
    }
    return fmt.Errorf("max retries exceeded")
}
```

## Implementation Details

### Index Generation

- Indices are monotonically increasing using atomic operations
- Each store maintains a global counter: `atomic.AddUint64(&globalIndex, 1)`
- Indices are never reused, even after deletion

### Atomicity Guarantees

**Single Operations:**
- CAS operations hold an exclusive lock during validation and update
- Either the entire operation succeeds or it fails atomically

**Batch Operations:**
- Two-phase approach: validate all CAS conditions first, then apply all updates
- If any CAS check fails, no modifications are made (all-or-nothing)

### Concurrency Safety

- All CAS operations use mutex-protected critical sections
- Index generation uses atomic operations
- Persistence happens after in-memory update (best-effort durability)

### Persistence

- ModifyIndex and CreateIndex are persisted with the data
- Old entries without indices are migrated on load (default index: 1)
- On restart, global index is set to max(all persisted indices)

## Best Practices

1. **Always retrieve metadata before CAS updates:**
   ```bash
   GET /kv/my-key?metadata=true
   ```

2. **Handle CAS conflicts gracefully:**
   - Implement retry logic with exponential backoff
   - Inform users of concurrent modifications
   - Consider using create-only (CAS=0) for unique resources

3. **Use batch operations for related updates:**
   - Atomic guarantees across multiple keys
   - Better performance than individual CAS operations

4. **Don't rely on specific index values:**
   - Indices are implementation details
   - Use them for comparison, not as business identifiers

5. **Monitor CAS conflict rates:**
   - High conflict rates indicate contention
   - Consider redesigning data model or access patterns

## Metrics

CAS operations are tracked in Prometheus metrics:

- `kv_operations_total{operation="set", status="cas_conflict"}`
- `kv_operations_total{operation="delete", status="cas_conflict"}`
- `service_operations_total{operation="register", status="cas_conflict"}`
- `service_operations_total{operation="deregister", status="cas_conflict"}`

## API Reference

### KV Store

| Endpoint | Method | Query Params | Body Fields | Description |
|----------|--------|--------------|-------------|-------------|
| `/kv/:key` | GET | `metadata=true` | - | Get with indices |
| `/kv/:key` | PUT | - | `value`, `cas`, `flags` | Set with CAS |
| `/kv/:key` | DELETE | `cas=N` | - | Delete with CAS |

### Service Registry

| Endpoint | Method | Query Params | Body Fields | Description |
|----------|--------|--------------|-------------|-------------|
| `/services/:name` | GET | `metadata=true` | - | Get with indices |
| `/services` | POST | - | `name`, `address`, `port`, `cas` | Register with CAS |
| `/services/:name` | DELETE | `cas=N` | - | Deregister with CAS |

## Comparison with Consul

Konsul's CAS implementation is inspired by HashiCorp Consul but simplified:

**Similarities:**
- ModifyIndex for version tracking
- CAS=0 for create-only semantics
- HTTP 409 Conflict on CAS failure

**Differences:**
- Konsul uses `modify_index` instead of `ModifyIndex` in JSON
- Batch CAS operations are explicit (not implicit transactions)
- Simpler error messages
- No Raft consensus (single-node for now)

## Future Enhancements

- [ ] Distributed consensus for multi-node CAS
- [ ] Transaction API for complex multi-key operations
- [ ] Watch API with index-based blocking queries
- [ ] CAS support for batch endpoints via HTTP API
- [ ] Automatic retry helpers in client libraries
