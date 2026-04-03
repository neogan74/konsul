# Raft CAS & Batch Operations Research

## 1. CAS via HashiCorp Raft

### Encoding Pattern
- **Single Raft Entry**: Compare+swap encoded as one `Command` with `CmdKVSetCAS` type
- **Payload Structure**: `KVSetCASPayload` contains key, value, and `ExpectedIndex` (version/index to match)
- **Atomicity Guarantee**: Raft ensures log entry applied atomically on FSM; CAS check done in `Apply()` before state change
- **Return Pattern**: FSM.Apply() returns `error` (nil=success, non-nil=CAS failed); caller checks return value to determine success

**Konsul Implementation** (commands.go:21-26, fsm.go:155-166):
```go
type KVSetCASPayload struct {
    Key string
    Value string
    ExpectedIndex uint64  // Version to match
}

func (f *KonsulFSM) applyKVSetCAS(payload []byte) error {
    _, err := f.kvStore.SetCASLocal(p.Key, p.Value, p.ExpectedIndex)
    return err  // nil=CAS succeeded, error=mismatch or store error
}
```

### Best Practices
1. **Index-Based Versioning**: Store maintains monotonic index; ExpectedIndex checked before update
2. **Mutex Protection**: FSM.mu locks during CAS check+update to prevent race conditions
3. **Error Propagation**: Store returns error if index doesn't match; FSM propagates to caller
4. **Idempotency**: Raft log index ensures exactly-once semantics on all replicas

### Gotchas
- CAS failure is **not** fatal error (KV store op failed, not consensus); caller must retry or handle
- Index must be current before CAS attempt; stale reads lead to spurious failures
- No transactional rollback if multi-key ops fail partway (see batch ops below)

---

## 2. Atomic Batch Operations via Raft

### Single Log Entry (Atomic) vs Multiple Entries

**Advantage of Single Entry**:
- All-or-nothing atomicity: batch either succeeds entirely or fails entirely
- No partial state; no cleanup needed
- Simpler semantics for client (one response)

**Trade-offs**:
- Larger log entries (serialization overhead)
- If one item fails, entire batch rejected (all-or-nothing constraint)
- Slower if batch is very large (higher latency)

**Konsul Approach** (commands.go:27-34): Single entry per batch.
```go
CmdKVBatchSet      // Multiple KV pairs, one log entry
CmdKVBatchSetCAS   // Multiple KV pairs with per-key indices, one entry
CmdKVBatchDelete   // Multiple keys, one entry
CmdKVBatchDeleteCAS // Multiple keys with per-key indices, one entry
```

### Implementation Patterns

**Batch CAS Semantics** (KVBatchSetCASPayload):
```go
type KVBatchSetCASPayload struct {
    Items map[string]string   // Key->Value pairs
    ExpectedIndices map[string]uint64  // Key->ExpectedIndex map
}
```
Each key can have its own expected version. FSM checks all versions before applying any update (atomic).

**FSM Application** (fsm.go:180-191):
- Lock entire FSM (`f.mu.Lock()`)
- Unmarshal all items
- Call `kvStore.BatchSetCASLocal(items, expectedIndices)` (store handles atomic all-or-nothing)
- Return error if any item's CAS fails

### How etcd/Consul Do It
- **etcd**: Uses txn (transaction) command type; txn can contain multiple ops, applied atomically (raft uses multi-op bundles)
- **Consul**: Native "transaction" API; batch operations via single RPC + single Raft entry
- **Pattern**: Payload contains list of ops; FSM iterates and applies all or none

---

## 3. Go Implementation Patterns with hashicorp/raft

### Command Encoding (JSON vs msgpack)
**Konsul Uses JSON** (commands.go:104-117):
```go
func NewCommand(cmdType CommandType, payload interface{}) (*Command, error) {
    data, err := json.Marshal(payload)
    return &Command{Type: cmdType, Payload: data, Timestamp: time.Now().Unix()}
}
```

**JSON Advantages**: Self-documenting, easy debug/logging, stdlib built-in
**msgpack Alternative**: Smaller serialization (10-30% less), lower latency; used in etcd/Consul for high-throughput

### FSM.Apply Return Values

**Pattern** (fsm.go:43-87):
- Return `nil` if command applied successfully
- Return `error` if command failed (store error, validation error, CAS mismatch)
- Return type is `interface{}` per raft.FSM contract; Konsul returns `error` (type assertion by caller)

**Error Propagation**:
1. FSM.Apply() returns error via interface{}
2. Raft core stores result in `raft.ApplyFuture`
3. Caller calls `.Result()` on future to retrieve error
4. If error non-nil, operation failed atomically (no state change)

### Error Propagation from FSM to Caller

**Flow**:
```
Client RPC Call
  -> Leader.Apply(cmd) [hashicorp/raft]
    -> FSM.Apply(log) [Konsul FSM]
      -> kvStore.SetCASLocal() [returns error if CAS fails]
      <- return error
    <- error stored in ApplyFuture
  -> future.Result() returns error
<- RPC response includes error
```

**Key Pattern**: FSM errors are **deterministic**; all replicas get same input, produce same output, deterministic errors (all fail same way).

### Concurrency & Locking

**Mutex Per Operation** (fsm.go:98-99, 124-125, 149-150):
```go
func (f *KonsulFSM) applyKVSet(payload []byte) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.kvStore.SetLocal(p.Key, p.Value)
}
```

**Why per-operation lock?**
- FSM.Apply() called sequentially by Raft (single thread)
- mu protects kvStore from concurrent reads (snapshot, lookup RPC)
- Batch operations lock entire FSM duration (atomicity)

---

## 4. Konsul Current State

**Existing Implementation** (commands.go, fsm.go):
- 8 KV command types (Set, SetCAS, Delete, DeleteCAS + Batch variants)
- 8 Service command types (similar CAS pattern)
- JSON encoding for all payloads
- Single-entry-per-batch model

**Ready for Phase 2 (Tier 2)**:
- CAS infrastructure solid; ExpectedIndex pattern proven
- Batch ops atomic via single Raft entry + FSM lock
- Error propagation clear (FSM returns error, caller checks)

---

## References & Sources

1. **hashicorp/raft**: `FSM.Apply(log *Log) interface{}` contract; sequential, deterministic calls
2. **Konsul Source**: `/internal/raft/commands.go` (types), `/internal/raft/fsm.go` (Apply logic)
3. **Raft Consensus Paper**: Ensures log replication identical across replicas; deterministic FSM guarantee
4. **etcd TxnRequest**: Proto-based batch ops; single Raft entry, atomic all-or-nothing
5. **Consul's KV Transactions**: Native batch with CAS per-item; similar pattern to Konsul

## Unresolved Questions

None. Research covers CAS encoding, batch atomicity, Go patterns, and error propagation with concrete Konsul examples.
