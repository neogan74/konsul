 # ADR-0002: BadgerDB for Persistence Layer

**Date**: 2024-09-18

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: database, persistence, storage, performance

## Context

Konsul needs an embedded persistent storage engine for:
- Key-value store persistence across restarts
- Service registry durability
- Backup and restore capabilities
- Low-latency reads and writes
- No external database dependencies

Requirements:
- Embedded (no separate database server)
- ACID compliance for consistency
- Write-ahead logging (WAL) for crash recovery
- Good read/write performance
- Concurrent access support
- Reasonable disk space usage

## Decision

We will use **BadgerDB** as the primary persistence engine for Konsul.

BadgerDB is an embeddable, persistent key-value (KV) database written in pure Go. It provides:

- LSM-tree based storage with excellent write performance
- MVCC (Multi-Version Concurrency Control) for concurrent reads
- Write-ahead log (WAL) for durability
- Built-in compression and encryption support
- Configurable sync modes for performance vs durability trade-offs
- Native Go implementation (no CGO dependencies)
- Battle-tested in production by Dgraph and others

## Alternatives Considered

### Alternative 1: BoltDB (bbolt)
- **Pros**:
  - Simple B+tree design, easy to understand
  - Very stable and mature
  - Used by etcd (proven in production)
  - Lower memory usage
- **Cons**:
  - Write performance slower than BadgerDB
  - No built-in compression
  - Single writer lock limits concurrent writes
  - Larger database files without compression
- **Reason for rejection**: Write performance is critical for service registration workloads

### Alternative 2: LevelDB (goleveldb)
- **Pros**:
  - Well-established LSM-tree implementation
  - Good read/write performance
  - Proven design from Google
- **Cons**:
  - No active maintenance (last commit 2+ years ago)
  - No native WAL configuration
  - Less feature-rich than BadgerDB
  - No built-in encryption
- **Reason for rejection**: Lack of maintenance is a risk; BadgerDB is more modern

### Alternative 3: SQLite (via CGO)
- **Pros**:
  - Industry standard embedded database
  - SQL query capabilities
  - Extremely well-tested
  - ACID compliant
- **Cons**:
  - Requires CGO (complicates cross-compilation)
  - SQL overhead for simple KV operations
  - Less performant for KV workloads
  - Larger binary size
- **Reason for rejection**: KV workload doesn't need SQL; CGO dependency adds complexity

### Alternative 4: Pebble
- **Pros**:
  - Modern LSM-tree implementation
  - Used by CockroachDB
  - Better space amplification than BadgerDB
  - Active development
- **Cons**:
  - Less mature than BadgerDB
  - Smaller community
  - More complex configuration
- **Reason for rejection**: BadgerDB more proven for our use case; community larger

## Consequences

### Positive
- Excellent write performance for service registrations
- Native Go implementation (no CGO) simplifies deployment
- Built-in compression reduces disk usage
- WAL provides crash recovery
- MVCC allows concurrent reads during writes
- Configurable sync modes for latency vs durability trade-offs
- Active development and community support

### Negative
- Higher memory usage than BoltDB
- Compaction can cause latency spikes under heavy load
- More complex internals (LSM-tree vs B+tree)
- Requires periodic value log GC
- Database files can grow without compression enabled

### Neutral
- Need to monitor and tune compaction settings
- Backup strategy must account for LSM-tree structure
- Memory vs disk trade-offs require configuration tuning

## Implementation Notes

Configuration approach:
```go
// Enable WAL for durability
WALEnabled: true

// Sync writes for critical data
SyncWrites: true

// Use compression to reduce disk usage
// (configured in BadgerDB options)
```

Key operations:
- Use transactions for consistency
- Implement periodic value log GC (every 10 minutes)
- Configure memory tables for workload patterns
- Monitor compaction metrics

Memory abstraction:
- Provide `persistence.Engine` interface
- Support both memory and BadgerDB backends
- Allow switching via configuration

## References

- [BadgerDB Documentation](https://dgraph.io/docs/badger/)
- [BadgerDB GitHub](https://github.com/dgraph-io/badger)
- [LSM-tree Architecture](https://en.wikipedia.org/wiki/Log-structured_merge-tree)
- [Konsul persistence package](../../internal/persistence/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-18 | Konsul Team | Initial version |
