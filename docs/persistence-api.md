# Persistence - API Reference

Complete API reference for Konsul persistence layer.

## Package `github.com/neogan74/konsul/internal/persistence`

### Types

#### `Config`

Persistence configuration structure.

```go
type Config struct {
    Enabled    bool   // Enable persistence
    Type       string // Engine type: "memory", "badger"
    DataDir    string // Data directory path
    BackupDir  string // Backup directory path
    SyncWrites bool   // Fsync after writes
    WALEnabled bool   // Enable write-ahead log
}
```

**Fields:**

- **Enabled** - Master switch for persistence
- **Type** - Persistence engine (`"memory"` or `"badger"`)
- **DataDir** - Directory for database files
- **BackupDir** - Directory for backup files
- **SyncWrites** - Force fsync after each write (durability vs performance)
- **WALEnabled** - Enable write-ahead logging for crash recovery

**Example:**
```go
cfg := persistence.Config{
    Enabled:    true,
    Type:       "badger",
    DataDir:    "./data",
    BackupDir:  "./backups",
    SyncWrites: true,
    WALEnabled: true,
}
```

---

#### `Engine`

Main persistence interface implemented by all backends.

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

    // Transaction support
    BeginTx() (Transaction, error)
}
```

**Implementations:**
- `BadgerEngine` - BadgerDB backend
- `MemoryEngine` - In-memory backend (testing)

---

#### `Transaction`

Database transaction interface for ACID operations.

```go
type Transaction interface {
    Set(key string, value []byte) error
    Delete(key string) error
    Commit() error
    Rollback() error
}
```

---

#### `BadgerEngine`

BadgerDB implementation of Engine.

```go
type BadgerEngine struct {
    // Unexported fields
}
```

**Internal fields:**
- `db` - BadgerDB instance
- `log` - Logger

---

### Functions

#### `NewEngine`

Factory function to create persistence engine.

```go
func NewEngine(cfg Config, log logger.Logger) (Engine, error)
```

**Parameters:**
- `cfg` - Persistence configuration
- `log` - Logger for structured logging

**Returns:** Configured engine or error

**Example:**
```go
cfg := persistence.Config{
    Enabled: true,
    Type:    "badger",
    DataDir: "./data",
}

log := logger.GetDefault()
engine, err := persistence.NewEngine(cfg, log)
if err != nil {
    log.Fatal("Failed to create engine", zap.Error(err))
}
defer engine.Close()
```

---

#### `NewBadgerEngine`

Create BadgerDB engine directly.

```go
func NewBadgerEngine(dataDir string, syncWrites bool, log logger.Logger) (*BadgerEngine, error)
```

**Parameters:**
- `dataDir` - Directory for database files
- `syncWrites` - Enable synchronous writes
- `log` - Logger instance

**Returns:** BadgerDB engine or error

**Example:**
```go
engine, err := persistence.NewBadgerEngine("./data", true, log)
if err != nil {
    return err
}
```

---

### Engine Methods

#### KV Operations

##### `Get`

Retrieve value for a key.

```go
func (e *Engine) Get(key string) ([]byte, error)
```

**Parameters:**
- `key` - Key to retrieve

**Returns:** Value bytes or error

**Errors:**
- `"key not found"` - Key doesn't exist

**Example:**
```go
value, err := engine.Get("config/host")
if err != nil {
    if err.Error() == "key not found" {
        // Handle missing key
    }
    return err
}

fmt.Printf("Value: %s\n", string(value))
```

---

##### `Set`

Store key-value pair.

```go
func (e *Engine) Set(key string, value []byte) error
```

**Parameters:**
- `key` - Key to store
- `value` - Value bytes

**Returns:** Error if write fails

**Example:**
```go
err := engine.Set("config/host", []byte("10.0.0.1"))
if err != nil {
    log.Error("Failed to set key", zap.Error(err))
}
```

---

##### `Delete`

Remove a key.

```go
func (e *Engine) Delete(key string) error
```

**Parameters:**
- `key` - Key to delete

**Returns:** Error if delete fails

**Example:**
```go
err := engine.Delete("config/host")
```

---

##### `List`

List keys with given prefix.

```go
func (e *Engine) List(prefix string) ([]string, error)
```

**Parameters:**
- `prefix` - Key prefix to match

**Returns:** Array of matching keys

**Example:**
```go
keys, err := engine.List("config/")
// Returns: ["config/host", "config/port", "config/db"]

for _, key := range keys {
    fmt.Println(key)
}
```

---

#### Service Operations

##### `GetService`

Retrieve service data.

```go
func (e *Engine) GetService(name string) ([]byte, error)
```

**Parameters:**
- `name` - Service name

**Returns:** Service data (JSON bytes) or error

**Errors:**
- `"service not found"` - Service doesn't exist
- `"service expired"` - Service TTL expired

**Example:**
```go
data, err := engine.GetService("web")
if err != nil {
    return err
}

var service Service
json.Unmarshal(data, &service)
```

---

##### `SetService`

Store service with TTL.

```go
func (e *Engine) SetService(name string, data []byte, ttl time.Duration) error
```

**Parameters:**
- `name` - Service name
- `data` - Service data (JSON)
- `ttl` - Time-to-live duration

**Returns:** Error if write fails

**Example:**
```go
service := Service{
    Name:    "web",
    Address: "10.0.0.1",
    Port:    8080,
}

data, _ := json.Marshal(service)
err := engine.SetService("web", data, 30*time.Second)
```

---

##### `DeleteService`

Remove a service.

```go
func (e *Engine) DeleteService(name string) error
```

**Parameters:**
- `name` - Service name

**Returns:** Error if delete fails

---

##### `ListServices`

List all non-expired services.

```go
func (e *Engine) ListServices() ([]string, error)
```

**Returns:** Array of service names

**Example:**
```go
services, err := engine.ListServices()
// Returns: ["web", "api", "db"]
```

---

#### Batch Operations

##### `BatchSet`

Set multiple keys atomically.

```go
func (e *Engine) BatchSet(items map[string][]byte) error
```

**Parameters:**
- `items` - Map of key-value pairs

**Returns:** Error if any write fails (all or nothing)

**Example:**
```go
items := map[string][]byte{
    "config/host": []byte("10.0.0.1"),
    "config/port": []byte("8080"),
    "config/db":   []byte("postgres"),
}

err := engine.BatchSet(items)
```

**Performance:** Much faster than individual Sets
```
100 individual Sets: ~100ms
1 BatchSet(100):     ~10ms
```

---

##### `BatchDelete`

Delete multiple keys atomically.

```go
func (e *Engine) BatchDelete(keys []string) error
```

**Parameters:**
- `keys` - Array of keys to delete

**Returns:** Error if any delete fails

**Example:**
```go
keys := []string{"old/key1", "old/key2", "old/key3"}
err := engine.BatchDelete(keys)
```

---

#### Management Operations

##### `Close`

Close the database.

```go
func (e *Engine) Close() error
```

**Returns:** Error if close fails

**Behavior:**
- Flushes pending writes
- Closes file handles
- Stops background goroutines (GC)
- Should be called before exit

**Example:**
```go
defer engine.Close()

// Or explicit close
if err := engine.Close(); err != nil {
    log.Error("Failed to close database", zap.Error(err))
}
```

---

##### `Backup`

Create database backup.

```go
func (e *Engine) Backup(path string) error
```

**Parameters:**
- `path` - Destination backup file path

**Returns:** Error if backup fails

**Behavior:**
- Creates consistent snapshot
- Non-blocking (reads continue)
- Creates parent directories
- Overwrites existing file

**Example:**
```go
timestamp := time.Now().Format("20060102-150405")
backupPath := fmt.Sprintf("./backups/konsul-%s.db", timestamp)

err := engine.Backup(backupPath)
if err != nil {
    log.Error("Backup failed", zap.Error(err))
}
```

---

##### `Restore`

Restore from backup.

```go
func (e *Engine) Restore(path string) error
```

**Parameters:**
- `path` - Source backup file path

**Returns:** Error if restore fails

**⚠️ Warning:** Overwrites existing data!

**Example:**
```go
err := engine.Restore("./backups/konsul-20250112.db")
if err != nil {
    log.Error("Restore failed", zap.Error(err))
}
```

---

#### Transaction Operations

##### `BeginTx`

Start a new transaction.

```go
func (e *Engine) BeginTx() (Transaction, error)
```

**Returns:** Transaction object or error

**Example:**
```go
tx, err := engine.BeginTx()
if err != nil {
    return err
}
defer tx.Rollback()  // Rollback if commit not called

// Do operations
tx.Set("key1", []byte("value1"))
tx.Set("key2", []byte("value2"))
tx.Delete("old_key")

// Commit atomically
if err := tx.Commit(); err != nil {
    return err
}
```

---

### Transaction Methods

##### `Set`

Set key-value in transaction.

```go
func (tx *Transaction) Set(key string, value []byte) error
```

---

##### `Delete`

Delete key in transaction.

```go
func (tx *Transaction) Delete(key string) error
```

---

##### `Commit`

Commit transaction atomically.

```go
func (tx *Transaction) Commit() error
```

**Behavior:**
- All operations applied atomically
- Other transactions see results
- Cannot use transaction after commit

---

##### `Rollback`

Abort transaction and discard changes.

```go
func (tx *Transaction) Rollback() error
```

**Behavior:**
- All operations discarded
- No effect on database
- Safe to call multiple times

---

## BadgerDB Specific Methods

### `ExportData`

Export all data to JSON format.

```go
func (b *BadgerEngine) ExportData() (map[string]interface{}, error)
```

**Returns:** Map with "kv" and "services" keys

**Example:**
```go
data, err := engine.(*persistence.BadgerEngine).ExportData()
if err != nil {
    return err
}

jsonData, _ := json.MarshalIndent(data, "", "  ")
os.WriteFile("export.json", jsonData, 0644)
```

---

### `ImportData`

Import data from JSON format.

```go
func (b *BadgerEngine) ImportData(data map[string]interface{}) error
```

**Parameters:**
- `data` - Data map with "kv" and "services" keys

**Example:**
```go
var data map[string]interface{}
jsonData, _ := os.ReadFile("export.json")
json.Unmarshal(jsonData, &data)

err := engine.(*persistence.BadgerEngine).ImportData(data)
```

---

## Internal Implementation Details

### Key Prefixes

BadgerDB uses prefixes to namespace data:

```go
const (
    kvPrefix      = "kv:"
    servicePrefix = "svc:"
)

// Examples:
// User key "config/host" → Stored as "kv:config/host"
// Service "web"          → Stored as "svc:web"
```

---

### BadgerDB Options

Default configuration:

```go
opts := badger.DefaultOptions(dataDir)
opts.SyncWrites = true              // Fsync on write
opts.ValueLogFileSize = 64 << 20    // 64MB value logs
opts.MemTableSize = 64 << 20        // 64MB memtable
opts.NumMemtables = 5               // 5 memtables in RAM
opts.NumLevelZeroTables = 5         // L0 compaction trigger
opts.Compression = 1                // Snappy compression
```

---

### Garbage Collection

Automatic GC runs every 5 minutes:

```go
func (b *BadgerEngine) runGarbageCollection() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        err := b.db.RunValueLogGC(0.5)  // 50% threshold
        // Remove value log files with >50% garbage
    }
}
```

---

## HTTP API Endpoints

### Backup Endpoint

```http
POST /backup
Content-Type: application/json

{
  "path": "./backups/backup-20250112.db"
}
```

**Response:**
```json
{
  "message": "Backup completed successfully",
  "path": "./backups/backup-20250112.db",
  "size_bytes": 1048576,
  "duration_ms": 150
}
```

---

### Restore Endpoint

```http
POST /restore
Content-Type: application/json

{
  "path": "./backups/backup-20250112.db"
}
```

**Response:**
```json
{
  "message": "Restore completed successfully",
  "path": "./backups/backup-20250112.db",
  "duration_ms": 200
}
```

---

### Export Endpoint

```http
GET /export
```

**Response:**
```json
{
  "kv": {
    "config/host": "10.0.0.1",
    "config/port": "8080"
  },
  "services": {
    "web": {
      "name": "web",
      "address": "10.0.0.1",
      "port": 8080
    }
  }
}
```

---

### Import Endpoint

```http
POST /import
Content-Type: application/json

{
  "kv": {
    "config/host": "10.0.0.1"
  },
  "services": {
    "web": {...}
  }
}
```

---

## Error Handling

### Common Errors

| Error | Meaning | Recovery |
|-------|---------|----------|
| `"key not found"` | Key doesn't exist | Check key name, create if needed |
| `"service not found"` | Service doesn't exist | Register service |
| `"service expired"` | Service TTL expired | Re-register with heartbeat |
| `"failed to open BadgerDB: Cannot acquire directory lock"` | Another instance running | Stop other instance |
| `"Corruption detected"` | Database corrupted | Restore from backup |

---

## Performance Characteristics

### Operation Latency

| Operation | Sync Writes | Async Writes |
|-----------|-------------|--------------|
| Get | ~100µs | ~100µs |
| Set (single) | ~1-2ms | ~100µs |
| BatchSet (100) | ~10ms | ~1ms |
| List (100 keys) | ~1ms | ~1ms |
| Backup (1GB) | ~5s | ~5s |

### Throughput

| Workload | Sync Writes | Async Writes |
|----------|-------------|--------------|
| Write-only | ~1,000/sec | ~50,000/sec |
| Read-only | ~100,000/sec | ~100,000/sec |
| Mixed (50/50) | ~5,000/sec | ~30,000/sec |

---

## Best Practices

### 1. Use Transactions for Related Operations

```go
// Bad - not atomic
engine.Set("counter", []byte("100"))
engine.Set("timestamp", []byte("2025-01-12"))

// Good - atomic
tx, _ := engine.BeginTx()
tx.Set("counter", []byte("100"))
tx.Set("timestamp", []byte("2025-01-12"))
tx.Commit()
```

---

### 2. Use Batch Operations

```go
// Bad - slow
for key, value := range items {
    engine.Set(key, value)  // N round trips
}

// Good - fast
engine.BatchSet(items)  // 1 transaction
```

---

### 3. Handle Errors Appropriately

```go
value, err := engine.Get("key")
if err != nil {
    if err.Error() == "key not found" {
        // Use default value
        value = []byte("default")
    } else {
        // Real error - log and return
        return err
    }
}
```

---

### 4. Close Resources

```go
func main() {
    engine, err := persistence.NewEngine(cfg, log)
    if err != nil {
        log.Fatal(err)
    }
    defer engine.Close()  // Always close

    // ... use engine ...
}
```

---

## Testing

### Mock Engine

For testing without actual database:

```go
engine := persistence.NewMemoryEngine()
defer engine.Close()

// Use in tests
engine.Set("test", []byte("value"))
value, _ := engine.Get("test")
```

---

### Integration Tests

```go
func TestBadgerIntegration(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "test")
    defer os.RemoveAll(tmpDir)

    engine, err := persistence.NewBadgerEngine(tmpDir, true, testLogger)
    require.NoError(t, err)
    defer engine.Close()

    // Test operations
    err = engine.Set("key", []byte("value"))
    assert.NoError(t, err)

    value, err := engine.Get("key")
    assert.NoError(t, err)
    assert.Equal(t, []byte("value"), value)
}
```

---

## See Also

- [Persistence User Guide](persistence-badger.md)
- [Persistence Implementation](persistence-implementation.md)
- [Persistence Troubleshooting](persistence-troubleshooting.md)
- [ADR-0002](adr/0002-badger-for-persistence.md)
- [BadgerDB Go Docs](https://pkg.go.dev/github.com/dgraph-io/badger/v4)
