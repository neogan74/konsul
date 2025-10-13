# BadgerDB Persistence - Implementation Guide

Technical deep dive into Konsul's persistence layer implementation.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                       │
│              (HTTP Handlers, Business Logic)                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                   Persistence Interface                     │
│                    (Engine interface)                       │
│  ┌──────────────┐                    ┌─────────────────┐   │
│  │ Memory Engine│                    │ Badger Engine   │   │
│  │  (Testing)   │                    │  (Production)   │   │
│  └──────────────┘                    └────────┬────────┘   │
└───────────────────────────────────────────────┼────────────┘
                                                 │
                                                 ↓
                         ┌───────────────────────────────────┐
                         │         BadgerDB (v4)             │
                         │   - LSM-tree storage              │
                         │   - WAL for durability            │
                         │   - MVCC for concurrency          │
                         └───────────────────────────────────┘
                                                 │
                                                 ↓
                                          ┌──────────────┐
                                          │ File System  │
                                          └──────────────┘
```

---

## Component Structure

### Package Files

```
internal/persistence/
├── interface.go       # Engine and Transaction interfaces
├── factory.go         # Engine factory function
├── badger.go          # BadgerDB implementation (360 lines)
├── memory.go          # In-memory implementation
├── badger_test.go     # BadgerDB tests
└── memory_test.go     # Memory tests
```

---

## Core Implementation

### Engine Interface

**File**: `internal/persistence/interface.go`

```go
type Engine interface {
    // KV operations
    Get(key string) ([]byte, error)
    Set(key string, value []byte) error
    Delete(key string) error
    List(prefix string) ([]string, error)

    // Service operations
    GetService(name string) ([]byte, error)
    SetService(name string, data []byte, ttl time.Duration) error
    DeleteService(name string) error
    ListServices() ([]string, error)

    // Batch operations
    BatchSet(items map[string][]byte) error
    BatchDelete(keys []string) error

    // Management
    Close() error
    Backup(path string) error
    Restore(path string) error

    // Transactions
    BeginTx() (Transaction, error)
}
```

**Design rationale**:
- Interface segregation - testability
- Batch operations - performance
- Transaction support - consistency
- Backup/restore - disaster recovery

---

### BadgerDB Implementation

**File**: `internal/persistence/badger.go:22-25`

```go
type BadgerEngine struct {
    db  *badger.DB      // BadgerDB instance
    log logger.Logger   // Structured logger
}
```

**Initialization**:

```go
func NewBadgerEngine(dataDir string, syncWrites bool, log logger.Logger) (*BadgerEngine, error) {
    // Create directory
    os.MkdirAll(dataDir, 0755)

    // Configure BadgerDB
    opts := badger.DefaultOptions(dataDir)
    opts.SyncWrites = syncWrites
    opts.ValueLogFileSize = 64 << 20  // 64MB
    opts.MemTableSize = 64 << 20      // 64MB
    opts.Compression = 1              // Snappy

    db, err := badger.Open(opts)

    engine := &BadgerEngine{db: db, log: log}

    // Start background GC
    go engine.runGarbageCollection()

    return engine, nil
}
```

---

### Key Prefixing Strategy

**Problem**: BadgerDB is pure KV store - need to separate KV data from services

**Solution**: Namespace prefixes

```go
const (
    kvPrefix      = "kv:"
    servicePrefix = "svc:"
)

// User writes:
engine.Set("config/host", []byte("10.0.0.1"))

// Stored in BadgerDB as:
"kv:config/host" → "10.0.0.1"

// Services:
engine.SetService("web", data, ttl)

// Stored as:
"svc:web" → {service JSON data} + TTL
```

**Benefits**:
- Clean separation
- Fast prefix scans
- No key collisions

---

### KV Operations

#### Get Operation

**File**: `internal/persistence/badger.go:89-103`

```go
func (b *BadgerEngine) Get(key string) ([]byte, error) {
    var value []byte

    err := b.db.View(func(txn *badger.Txn) error {
        item, err := txn.Get([]byte(kvPrefix + key))
        if err != nil {
            return err
        }
        value, err = item.ValueCopy(nil)
        return err
    })

    if errors.Is(err, badger.ErrKeyNotFound) {
        return nil, errors.New("key not found")
    }
    return value, err
}
```

**Flow**:
1. Start read-only transaction (`View`)
2. Prefix key with `"kv:"`
3. Get item from BadgerDB
4. Copy value (important - BadgerDB uses memory mapped files)
5. Convert `ErrKeyNotFound` to user-friendly error

---

#### Set Operation

**File**: `internal/persistence/badger.go:105-109`

```go
func (b *BadgerEngine) Set(key string, value []byte) error {
    return b.db.Update(func(txn *badger.Txn) error {
        return txn.Set([]byte(kvPrefix+key), value)
    })
}
```

**Transaction wrapping**:
- `Update()` starts write transaction
- Automatic commit on success
- Automatic rollback on error

---

### Service Operations with TTL

**File**: `internal/persistence/badger.go:161-166`

```go
func (b *BadgerEngine) SetService(name string, data []byte, ttl time.Duration) error {
    return b.db.Update(func(txn *badger.Txn) error {
        e := badger.NewEntry([]byte(servicePrefix+name), data).WithTTL(ttl)
        return txn.SetEntry(e)
    })
}
```

**TTL Handling**:
- BadgerDB native TTL support
- Automatically expires entries
- No manual cleanup needed

**GetService with expiration check**:

```go
func (b *BadgerEngine) GetService(name string) ([]byte, error) {
    var data []byte

    err := b.db.View(func(txn *badger.Txn) error {
        item, err := txn.Get([]byte(servicePrefix + name))
        if err != nil {
            return err
        }

        // Check if expired
        if item.ExpiresAt() != 0 && time.Now().Unix() > int64(item.ExpiresAt()) {
            return errors.New("service expired")
        }

        data, err = item.ValueCopy(nil)
        return err
    })

    return data, err
}
```

---

### List Operations

**File**: `internal/persistence/badger.go:117-137`

```go
func (b *BadgerEngine) List(prefix string) ([]string, error) {
    var keys []string
    searchPrefix := kvPrefix + prefix

    err := b.db.View(func(txn *badger.Txn) error {
        opts := badger.DefaultIteratorOptions
        opts.PrefetchValues = false  // Only need keys

        it := txn.NewIterator(opts)
        defer it.Close()

        prefixBytes := []byte(searchPrefix)
        for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
            item := it.Item()
            key := string(item.Key())
            keys = append(keys, strings.TrimPrefix(key, kvPrefix))
        }
        return nil
    })

    return keys, err
}
```

**Iterator optimization**:
- `PrefetchValues = false` - don't load values (faster)
- `ValidForPrefix()` - stops when prefix doesn't match
- `Seek()` - jump to first matching key

---

### Batch Operations

**File**: `internal/persistence/badger.go:199-208`

```go
func (b *BadgerEngine) BatchSet(items map[string][]byte) error {
    return b.db.Update(func(txn *badger.Txn) error {
        for key, value := range items {
            if err := txn.Set([]byte(kvPrefix+key), value); err != nil {
                return err  // Rollback entire batch
            }
        }
        return nil  // Commit all or nothing
    })
}
```

**Atomicity**:
- Single transaction for all operations
- All succeed or all fail
- Much faster than individual writes

**Performance comparison**:
```
100 individual Sets: ~100ms  (100 transactions)
1 BatchSet(100):     ~10ms   (1 transaction)
```

---

### Garbage Collection

**File**: `internal/persistence/badger.go:77-87`

```go
func (b *BadgerEngine) runGarbageCollection() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        err := b.db.RunValueLogGC(0.5)
        if err != nil && !errors.Is(err, badger.ErrNoRewrite) {
            b.log.Warn("BadgerDB garbage collection failed", logger.Error(err))
        }
    }
}
```

**GC Strategy**:
- Runs every 5 minutes
- Threshold: 0.5 (50% garbage in value log)
- `ErrNoRewrite` is expected (nothing to GC)
- Runs in background goroutine
- Reclaims disk space from deleted/overwritten values

---

### Backup Implementation

**File**: `internal/persistence/badger.go:225-245`

```go
func (b *BadgerEngine) Backup(path string) error {
    // Create backup directory
    dir := filepath.Dir(path)
    os.MkdirAll(dir, 0755)

    file, err := os.Create(path)
    if err != nil {
        return fmt.Errorf("failed to create backup file: %w", err)
    }
    defer file.Close()

    _, err = b.db.Backup(file, 0)
    if err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }

    b.log.Info("Backup completed successfully", logger.String("path", path))
    return nil
}
```

**Backup characteristics**:
- Consistent snapshot (MVCC)
- Non-blocking (reads continue)
- Single file output
- Compressed by BadgerDB
- Safe to call while database in use

---

### Transaction Implementation

**File**: `internal/persistence/badger.go:263-288`

```go
func (b *BadgerEngine) BeginTx() (Transaction, error) {
    txn := b.db.NewTransaction(true)  // Read-write
    return &badgerTx{txn: txn}, nil
}

type badgerTx struct {
    txn *badger.Txn
}

func (tx *badgerTx) Set(key string, value []byte) error {
    return tx.txn.Set([]byte(kvPrefix+key), value)
}

func (tx *badgerTx) Commit() error {
    return tx.txn.Commit()
}

func (tx *badgerTx) Rollback() error {
    tx.txn.Discard()
    return nil
}
```

**Usage pattern**:
```go
tx, _ := engine.BeginTx()
defer tx.Rollback()  // Safe to call even after commit

tx.Set("counter", []byte("100"))
tx.Set("timestamp", []byte(time.Now().String()))

if err := tx.Commit(); err != nil {
    // Rollback already called by defer
    return err
}
```

---

## Testing Strategy

### Unit Tests

**File**: `internal/persistence/badger_test.go`

**Test coverage**:
- ✅ Basic KV operations
- ✅ Service operations with TTL
- ✅ Batch operations
- ✅ Backup/restore
- ✅ Transactions
- ✅ Edge cases (expiration, not found, etc.)

**Test results**:
```
=== RUN   TestBadgerEngine_Basic
--- PASS: TestBadgerEngine_Basic (0.05s)
=== RUN   TestBadgerEngine_Services
--- PASS: TestBadgerEngine_Services (6.05s)
=== RUN   TestBadgerEngine_BatchOperations
--- PASS: TestBadgerEngine_BatchOperations (0.04s)
=== RUN   TestBadgerEngine_BackupRestore
--- PASS: TestBadgerEngine_BackupRestore (0.08s)
=== RUN   TestBadgerEngine_Transactions
--- PASS: TestBadgerEngine_Transactions (0.03s)
PASS
ok      github.com/neogan74/konsul/internal/persistence  6.924s
```

---

## Design Patterns

### 1. Interface Segregation

```go
// Clean interface for testing
type Engine interface {
    Get(key string) ([]byte, error)
    Set(key string, value []byte) error
    // ...
}

// Mock for testing
type MockEngine struct {
    data map[string][]byte
}
```

**Benefits**:
- Easy to mock
- Swap implementations (memory vs BadgerDB)
- Clear contract

---

### 2. Factory Pattern

```go
func NewEngine(cfg Config, log logger.Logger) (Engine, error) {
    switch cfg.Type {
    case "memory":
        return NewMemoryEngine(), nil
    case "badger":
        return NewBadgerEngine(cfg.DataDir, cfg.SyncWrites, log)
    }
}
```

**Benefits**:
- Centralized creation
- Configuration-driven
- Easy to add new backends

---

### 3. Decorator Pattern (Transactions)

```go
type Transaction interface {
    Set(key string, value []byte) error
    Commit() error
    Rollback() error
}

type badgerTx struct {
    txn *badger.Txn  // Wraps BadgerDB transaction
}
```

**Benefits**:
- Clean abstraction
- Hide BadgerDB details
- Consistent with Engine interface

---

## Performance Optimizations

### 1. Batch Operations

**Implementation**:
```go
// Single transaction for multiple writes
func (b *BadgerEngine) BatchSet(items map[string][]byte) error {
    return b.db.Update(func(txn *badger.Txn) error {
        for key, value := range items {
            txn.Set([]byte(kvPrefix+key), value)
        }
        return nil
    })
}
```

**Speedup**: 10x faster than individual writes

---

### 2. Iterator Optimization

```go
opts := badger.DefaultIteratorOptions
opts.PrefetchValues = false  // Don't load values
opts.PrefetchSize = 100      // Prefetch 100 keys
```

---

### 3. Value Copy

```go
// Bad - value is invalid after transaction
item.Value()

// Good - copies value
item.ValueCopy(nil)
```

---

## Future Enhancements

### 1. Encryption at Rest

```go
opts.EncryptionKey = []byte("32-byte-key-here")
opts.EncryptionRotationDuration = 7 * 24 * time.Hour
```

### 2. Streaming Backup

```go
func (b *BadgerEngine) StreamBackup(w io.Writer) error {
    return b.db.Backup(w, 0)
}
```

### 3. Metrics Integration

```go
var (
    badgerReads = prometheus.NewCounter(...)
    badgerWrites = prometheus.NewCounter(...)
)
```

---

## See Also

- [Persistence User Guide](persistence-badger.md)
- [Persistence API Reference](persistence-api.md)
- [ADR-0002](adr/0002-badger-for-persistence.md)
- [BadgerDB Documentation](https://dgraph.io/docs/badger/)
