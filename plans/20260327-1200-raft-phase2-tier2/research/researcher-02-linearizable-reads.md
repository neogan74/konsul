# Linearizable Reads in Raft: Research Summary

## 1. Raft ReadIndex Algorithm (§6.4, Ongaro Dissertation)

**Concept**: Leader confirms current leadership via heartbeat quorum before serving read, ensuring read reflects committed state.

**Steps**:
1. Leader records current commit index (readIndex) when read request arrives
2. Issue new heartbeat round to verify current leadership (quorum ack)
3. Once heartbeat acknowledged by majority, leader knows no new leader elected with higher term
4. Leader waits for state machine to apply entries up to readIndex
5. Serve read from applied state

**Key Guarantee**: Read cannot return stale data because readIndex ≥ highest index any client saw committed before the read was issued.

## 2. HashiCorp Raft APIs for Linearizable Reads

**Available APIs** (github.com/hashicorp/raft):

| API | Purpose |
|-----|---------|
| `raft.VerifyLeader()` | Returns error if node is not current leader (lightweight, no quorum round) |
| `raft.LastIndex()` | Get last log index applied |
| `raft.AppliedIndex()` | Get last index applied to FSM |
| `raft.Barrier(index)` | Wait until index applied to FSM (log read pattern) |

**No native ReadIndex**: hashicorp/raft does NOT implement ReadIndex natively. Must use Barrier() pattern or manual leadership verification.

**Implementation pattern**:
```go
// Verify leadership + wait for current state
leader := raft.Leader()
if leader != raftNode.Id() {
    return errors.New("not leader")
}
index := raft.LastIndex() // Current commit index
raft.Barrier(index).Await() // Wait for application
// Safe to read from FSM
```

## 3. Read Consistency Patterns: Stale vs Consistent

**Three strategies** (performance/consistency tradeoff):

| Pattern | Consistency | Implementation | Latency | Use Case |
|---------|-------------|-----------------|---------|----------|
| **Stale Read** | Eventual | Read local FSM directly | <1ms | Counters, metrics, non-critical |
| **Log Read** | Strong | raft.Barrier(lastIndex) | ~5-20ms | KV queries, config reads |
| **Lease Read** | Strong | Leadership lease (no heartbeat) | ~1-5ms | High throughput, bounded clock skew |

**API Design**:
```go
// Query parameter approach
GET /kv/key?consistent=true  // Log/ReadIndex read
GET /kv/key                  // Stale read from follower
```

**Client-facing**: Consul exposes `?consistent=true` query parameter; defaults to stale reads for performance.

## 4. Production Examples

**Consul Implementation**:
- Uses raft.VerifyLeader() + Barrier() pattern for linearizable reads
- Query API: `GET /v1/kv/key?consistent=true` triggers Barrier()
- Stale reads default for followers (performance optimization)
- Health checks default to stale; critical queries use consistent=true

**Open source projects using hashicorp/raft**:
- **etcd** (Go version): Implements ReadIndex in leader, serves reads without log replay
- **Nomad** (HashiCorp): Uses Barrier() for strong consistency
- **Vault** (HashiCorp): Stateless design; relies on consistent=true flag

## Key Takeaway

HashiCorp Raft requires **manual ReadIndex implementation** via Barrier() or verification patterns. No built-in ReadIndex algorithm. Recommended: implement Barrier() wrapper with optional lease-based reads for production throughput needs.

---

**References**: Ongaro dissertation §6.4; hashicorp/raft API docs; Consul source code
