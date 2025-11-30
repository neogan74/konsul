# Compare-And-Swap (CAS) Implementation Summary

## Overview

This document summarizes the implementation of atomic Compare-And-Swap operations for Konsul, providing optimistic concurrency control for both the KV store and service registry.

## Implementation Scope

### ✅ Completed Features

#### 1. Core Data Structures
- **KVEntry** with version tracking:
  - `ModifyIndex` - monotonically increasing modification counter
  - `CreateIndex` - immutable creation timestamp
  - `Flags` - optional user-defined flags
  - Backward-compatible JSON serialization

- **ServiceEntry** with version tracking:
  - `ModifyIndex` - tracks service registration changes
  - `CreateIndex` - records initial registration
  - Preserves existing TTL and expiration logic

#### 2. Store-Level CAS Operations

**KVStore (`internal/store/kv.go`):**
- `SetCAS(key, value, expectedIndex)` - conditional set operation
- `DeleteCAS(key, expectedIndex)` - conditional delete operation
- `BatchSetCAS(items, expectedIndices)` - atomic batch set with CAS
- `BatchDeleteCAS(keys, expectedIndices)` - atomic batch delete with CAS
- `GetEntry(key)` - retrieve with metadata
- `SetWithFlags(key, value, flags)` - set with custom flags
- `ListEntries()` - list all entries with metadata

**ServiceStore (`internal/store/service.go`):**
- `RegisterCAS(service, expectedIndex)` - conditional registration
- `DeregisterCAS(name, expectedIndex)` - conditional deregistration
- `GetEntry(name)` - retrieve with metadata
- Heartbeat preserves ModifyIndex (doesn't trigger version increment)

#### 3. Error Handling (`internal/store/errors.go`)

**CASConflictError:**
- Provides detailed conflict information
- Includes expected vs. current indices
- Helper: `IsCASConflict(err)`

**NotFoundError:**
- Returned when CAS update targets non-existent resource
- Helper: `IsNotFound(err)`

#### 4. HTTP API Integration

**KV Handler (`internal/handlers/kv.go`):**
- `GET /kv/:key?metadata=true` - retrieve with indices
- `PUT /kv/:key` with `{"cas": N}` body - CAS set
- `DELETE /kv/:key?cas=N` - CAS delete
- Returns `modify_index` in all responses
- HTTP 409 Conflict on CAS failures
- Metrics tracking for CAS conflicts

**Service Handler (`internal/handlers/service.go`):**
- `GET /services/:name?metadata=true` - retrieve with indices
- `POST /services` with `{"cas": N}` body - CAS registration
- `DELETE /services/:name?cas=N` - CAS deregistration
- Returns `modify_index` in responses
- HTTP 409 Conflict on CAS failures
- Metrics tracking for CAS conflicts

#### 5. Comprehensive Test Suite

**KV Store Tests (`internal/store/kv_cas_test.go`):**
- ✅ Create-only semantics (CAS=0)
- ✅ Conditional update with version check
- ✅ Delete with CAS
- ✅ Batch operations with CAS
- ✅ Concurrency safety (10 iterations × 5 goroutines)
- ✅ Index monotonicity verification
- ✅ CreateIndex preservation across updates
- ✅ Flags preservation with CAS updates

**Service Store Tests (`internal/store/service_cas_test.go`):**
- ✅ Create-only service registration
- ✅ Conditional service updates
- ✅ Deregister with CAS
- ✅ Concurrency safety under load
- ✅ Index monotonicity
- ✅ CreateIndex preservation
- ✅ Heartbeat doesn't modify indices
- ✅ Tag and metadata index updates with CAS

#### 6. Documentation
- ✅ Comprehensive CAS guide (`docs/CAS.md`)
- ✅ API reference with examples
- ✅ Use cases and best practices
- ✅ Error handling patterns
- ✅ Comparison with Consul

## Architecture Decisions

### 1. Index Generation Strategy
**Decision:** Single global counter per store with atomic increments

**Rationale:**
- Simple and fast implementation
- Monotonically increasing guarantees
- No coordination needed between operations
- Compatible with future distributed consensus

**Implementation:**
```go
func (kv *KVStore) nextIndex() uint64 {
    return atomic.AddUint64(&kv.globalIndex, 1)
}
```

### 2. Batch Operation Atomicity
**Decision:** Two-phase validation and application

**Rationale:**
- All-or-nothing semantics
- Validates all CAS conditions before any modifications
- Prevents partial updates on conflicts
- Consistent with transaction expectations

**Implementation:**
```go
// Phase 1: Validate all CAS conditions
for key := range items {
    if !validateCAS(key, expectedIndices[key]) {
        return error // No modifications made
    }
}
// Phase 2: Apply all updates
for key, value := range items {
    applyUpdate(key, value)
}
```

### 3. Heartbeat Behavior
**Decision:** Heartbeats do NOT increment ModifyIndex

**Rationale:**
- Heartbeats are not semantic modifications
- Prevents unnecessary CAS conflicts
- Allows concurrent heartbeats and updates
- TTL extension is metadata, not data change

### 4. Persistence Strategy
**Decision:** Persist indices with data, migrate old entries on load

**Rationale:**
- Backward compatible with existing data
- Survives restarts
- Simple migration path
- Global index reconstructed from persisted data

### 5. HTTP API Design
**Decision:** Query parameter for GET, body field for mutations

**Rationale:**
- RESTful conventions (query params for read options)
- Body fields for write data
- Easy to make CAS optional
- Compatible with existing API

## File Changes Summary

### New Files
1. `internal/store/errors.go` - Error types for CAS operations
2. `internal/store/kv_cas_test.go` - KV CAS test suite (446 lines)
3. `internal/store/service_cas_test.go` - Service CAS test suite (434 lines)
4. `docs/CAS.md` - Comprehensive CAS documentation

### Modified Files
1. `internal/store/kv.go`:
   - Added KVEntry struct with indices
   - Implemented CAS methods (SetCAS, DeleteCAS, BatchSetCAS, BatchDeleteCAS)
   - Updated existing methods to use KVEntry
   - Added GetEntry and ListEntries methods
   - ~400 lines added

2. `internal/store/service.go`:
   - Added ModifyIndex/CreateIndex to ServiceEntry
   - Implemented RegisterCAS and DeregisterCAS
   - Added GetEntry method
   - Updated persistence to handle indices
   - ~200 lines added

3. `internal/handlers/kv.go`:
   - Added metadata query parameter support
   - Implemented CAS handling in Set/Delete
   - Enhanced error responses with conflict details
   - ~100 lines added

4. `internal/handlers/service.go`:
   - Added metadata query parameter support
   - Implemented CAS handling in Register/Deregister
   - Enhanced error responses
   - ~100 lines added

### Lines of Code
- **New code:** ~1,680 lines
- **Modified code:** ~800 lines
- **Test code:** ~880 lines
- **Documentation:** ~350 lines

## Testing Results

### Unit Tests
```
=== All CAS Tests ===
✅ TestKVStore_SetCAS_CreateOnly
✅ TestKVStore_SetCAS_Update
✅ TestKVStore_SetCAS_NotFound
✅ TestKVStore_DeleteCAS
✅ TestKVStore_BatchSetCAS
✅ TestKVStore_BatchDeleteCAS
✅ TestKVStore_CAS_Concurrency (10 successes, 40 conflicts)
✅ TestKVStore_CAS_IndexMonotonicity
✅ TestKVStore_CAS_PreservesCreateIndex
✅ TestKVStore_CAS_PreservesFlags
✅ TestServiceStore_RegisterCAS_CreateOnly
✅ TestServiceStore_RegisterCAS_Update
✅ TestServiceStore_RegisterCAS_NotFound
✅ TestServiceStore_DeregisterCAS
✅ TestServiceStore_CAS_Concurrency (10 successes, 40 conflicts)
✅ TestServiceStore_CAS_IndexMonotonicity
✅ TestServiceStore_CAS_PreservesCreateIndex
✅ TestServiceStore_CAS_HeartbeatPreservesIndex
✅ TestServiceStore_CAS_TagAndMetaIndexes

PASS - All 19 tests passed
```

### Concurrency Testing
- Verified with 5 concurrent goroutines × 10 iterations
- Exactly 1 success per iteration (deterministic)
- All conflicts properly detected and handled
- No race conditions detected

## Performance Characteristics

### Time Complexity
- `SetCAS`: O(1) - atomic counter increment + map update
- `GetEntry`: O(1) - map lookup
- `BatchSetCAS`: O(n) - validates n items, applies n updates
- Index generation: O(1) - atomic operation

### Space Complexity
- Per entry overhead: 16 bytes (2 × uint64 for indices)
- No additional data structures needed
- Minimal memory footprint

### Concurrency
- Lock contention: Same as non-CAS operations
- No additional synchronization overhead
- CAS validation happens within existing critical sections

## API Examples

### KV Store CAS
```bash
# Get with metadata
curl http://localhost:8500/kv/config?metadata=true

# Create-only
curl -X PUT http://localhost:8500/kv/config \
  -d '{"value": "v1", "cas": 0}'

# Conditional update
curl -X PUT http://localhost:8500/kv/config \
  -d '{"value": "v2", "cas": 1}'

# CAS delete
curl -X DELETE "http://localhost:8500/kv/config?cas=2"
```

### Service Registry CAS
```bash
# Get with metadata
curl http://localhost:8500/services/api?metadata=true

# Register with CAS
curl -X POST http://localhost:8500/services \
  -d '{
    "name": "api",
    "address": "10.0.0.1",
    "port": 8080,
    "cas": 5
  }'

# Deregister with CAS
curl -X DELETE "http://localhost:8500/services/api?cas=6"
```

## Migration Guide

### For Existing Data

**Automatic Migration:**
- Old KV entries (plain strings) are migrated on first load
- Default indices: `ModifyIndex = 1`, `CreateIndex = 1`
- Global index set to max of all loaded indices
- No manual intervention required

**New Entries:**
- Automatically created with proper indices
- Indices start from (max loaded index + 1)

### For API Clients

**Backward Compatible:**
- Existing API calls work without changes
- CAS is optional - don't include `cas` field for standard operations
- Responses now include `modify_index` but old clients can ignore it

**Opt-In CAS:**
- Add `?metadata=true` to GET requests for indices
- Add `"cas": N` to PUT/POST requests for CAS semantics
- Handle HTTP 409 conflicts in client code

## Monitoring and Observability

### Prometheus Metrics

New metric labels:
```
kv_operations_total{operation="set", status="cas_conflict"}
kv_operations_total{operation="delete", status="cas_conflict"}
service_operations_total{operation="register", status="cas_conflict"}
service_operations_total{operation="deregister", status="cas_conflict"}
```

### Logging

CAS operations log additional context:
- Expected vs. current indices on conflict
- Success/failure of CAS operations
- Index values in successful operations

## Future Enhancements

### Short Term
- [ ] HTTP API for batch CAS operations
- [ ] Client library with automatic retry
- [ ] CAS metrics dashboard

### Medium Term
- [ ] Transaction API (multi-key CAS)
- [ ] Watch API with index-based blocking
- [ ] Index-based pagination for List operations

### Long Term
- [ ] Distributed consensus (Raft) for multi-node CAS
- [ ] Snapshot and restore with index preservation
- [ ] Cross-datacenter index synchronization

## Known Limitations

1. **Single Node Only:**
   - CAS is local to single node
   - No distributed coordination yet
   - Plan: Add Raft consensus in future

2. **Index Overflow:**
   - uint64 allows ~18 quintillion operations
   - Practically unlimited for single node
   - Would need index compaction for multi-node

3. **No Transactions:**
   - Batch CAS is all-or-nothing but limited to same operation type
   - Can't mix set/delete in single transaction
   - Plan: Add transaction API

4. **Persistence Lag:**
   - Persistence happens after in-memory update
   - Brief window where index in memory ≠ index on disk
   - Acceptable for single node, needs fixing for distributed

## Security Considerations

- CAS operations don't introduce new security vectors
- Same authentication/authorization as non-CAS operations
- Index values are not security-sensitive
- No timing attacks possible (indices are monotonic counters)

## Conclusion

The CAS implementation provides:
- ✅ Atomic operations with optimistic concurrency control
- ✅ Backward-compatible API changes
- ✅ Comprehensive test coverage
- ✅ Production-ready error handling
- ✅ Detailed documentation
- ✅ Foundation for future distributed features

All tests pass, code compiles, and the feature is ready for production use.
