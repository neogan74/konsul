# BadgerDB Persistence - User Guide

Comprehensive guide for using BadgerDB persistence in Konsul.

## Overview

Konsul uses BadgerDB as its embedded persistence engine to provide durable storage for:
- **Key-Value Store** - Persistent KV data across restarts
- **Service Registry** - Service registrations with TTL support
- **Configuration Data** - System configuration and state

BadgerDB is a fast, embeddable key-value database written in pure Go, featuring:
- LSM-tree architecture for excellent write performance
- MVCC (Multi-Version Concurrency Control) for concurrent reads
- Write-Ahead Log (WAL) for crash recovery
- Built-in compression and garbage collection
- ACID transactions

---

## Quick Start

### 1. Enable Persistence

**Via Environment Variables:**
```bash
KONSUL_PERSISTENCE_ENABLED=true \
KONSUL_PERSISTENCE_TYPE=badger \
KONSUL_DATA_DIR=./data \
./konsul
```

**Via Configuration File:**
```yaml
# config.yaml
persistence:
  enabled: true
  type: badger
  data_dir: ./data
  sync_writes: true
  wal_enabled: true
```

### 2. Verify Persistence

```bash
# Write data
curl -X POST http://localhost:8500/kv/test \
  -H "Content-Type: application/json" \
  -d '{"value": "persistent_data"}'

# Restart Konsul
kill $(pgrep konsul)
./konsul

# Verify data persists
curl http://localhost:8500/kv/test
# Returns: {"value": "persistent_data"}
```

---

## Configuration Options

### Basic Configuration

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| Enabled | `KONSUL_PERSISTENCE_ENABLED` | `false` | Enable persistence |
| Type | `KONSUL_PERSISTENCE_TYPE` | `badger` | Persistence engine type |
| Data Dir | `KONSUL_DATA_DIR` | `./data` | Directory for database files |
| Sync Writes | `KONSUL_SYNC_WRITES` | `true` | Fsync after each write (durability) |
| WAL Enabled | `KONSUL_WAL_ENABLED` | `true` | Enable write-ahead log |

### Advanced Configuration

| Option | Default | Description |
|--------|---------|-------------|
| ValueLogFileSize | 64MB | Size of value log files |
| MemTableSize | 64MB | Size of memtable |
| NumMemtables | 5 | Number of memtables in memory |
| NumLevelZeroTables | 5 | L0 tables before compaction |
| Compression | Snappy | Compression algorithm |

---

## Performance Modes

### Maximum Durability (Production)

Best for production environments requiring data safety:

```bash
KONSUL_PERSISTENCE_ENABLED=true \
KONSUL_PERSISTENCE_TYPE=badger \
KONSUL_SYNC_WRITES=true \
KONSUL_WAL_ENABLED=true \
./konsul
```

**Characteristics:**
- ✅ ACID compliant
- ✅ Crash-safe (WAL enabled)
- ✅ Fsync on every write
- ⚠️ Lower write throughput (~1,000 writes/sec)
- ⚠️ Higher latency (~1-2ms per write)

---

### Maximum Performance (Development)

Best for development and testing:

```bash
KONSUL_PERSISTENCE_ENABLED=true \
KONSUL_PERSISTENCE_TYPE=badger \
KONSUL_SYNC_WRITES=false \
KONSUL_WAL_ENABLED=true \
./konsul
```

**Characteristics:**
- ✅ High write throughput (~50,000 writes/sec)
- ✅ Low latency (~100µs per write)
- ⚠️ Data loss risk on crash (buffered writes)
- ⚠️ Not recommended for production

---

### Balanced Mode (Recommended)

Good balance for most deployments:

```bash
KONSUL_PERSISTENCE_ENABLED=true \
KONSUL_PERSISTENCE_TYPE=badger \
KONSUL_SYNC_WRITES=true \
KONSUL_WAL_ENABLED=true \
# Accept slightly higher latency for durability
./konsul
```

**Characteristics:**
- ✅ Durable with WAL
- ✅ Good performance (~10,000 writes/sec)
- ✅ Reasonable latency (~500µs)
- ✅ Production-ready

---

## Data Organization

### Directory Structure

```
./data/
├── KEYREGISTRY          # Registry for LSM-tree
├── MANIFEST             # Database manifest
├── 000000.vlog          # Value log files
├── 000001.vlog
├── 000000.sst           # Sorted string tables (SSTables)
├── 000001.sst
└── LOCK                 # Lock file
```

**File Types:**
- **MANIFEST** - Tracks database state
- **\*.sst** - Sorted String Tables (immutable)
- **\*.vlog** - Value log files (large values)
- **KEYREGISTRY** - Key registry for fast lookups
- **LOCK** - Prevents multiple processes

---

### Key Prefixes

BadgerDB internally uses prefixes to separate data:

```
kv:mykey         → KV store data
svc:web          → Service registry data
```

**User keys are automatically prefixed**:
```bash
# You write:
PUT /kv/config/host "10.0.0.1"

# Stored as:
kv:config/host → "10.0.0.1"
```

---

## Operations

### Backup

Create a backup of the database:

```bash
curl -X POST http://localhost:8500/backup \
  -H "Content-Type: application/json" \
  -d '{"path": "./backups/backup-2025-01-12.db"}'
```

**Response:**
```json
{
  "message": "Backup completed successfully",
  "path": "./backups/backup-2025-01-12.db",
  "size_bytes": 1048576,
  "duration_ms": 150
}
```

**Backup Characteristics:**
- ✅ Consistent snapshot (MVCC)
- ✅ Non-blocking (reads continue during backup)
- ✅ Compresses data automatically
- ⚠️ Size depends on database contents

---

### Restore

Restore from a backup:

```bash
curl -X POST http://localhost:8500/restore \
  -H "Content-Type: application/json" \
  -d '{"path": "./backups/backup-2025-01-12.db"}'
```

**⚠️ Warning:** Restore overwrites existing data!

**Steps:**
1. Stop Konsul
2. Backup current data (optional)
3. Restore from backup file
4. Start Konsul

```bash
# Safe restore procedure
systemctl stop konsul
cp -r ./data ./data.backup
curl -X POST http://localhost:8500/restore -d '{"path": "./backups/backup.db"}'
systemctl start konsul
```

---

### Export/Import Data

Export data to JSON for inspection:

**Export:**
```bash
curl http://localhost:8500/export > data.json
```

**Export Format:**
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

**Import:**
```bash
curl -X POST http://localhost:8500/import \
  -H "Content-Type: application/json" \
  -d @data.json
```

---

## Maintenance

### Garbage Collection

BadgerDB automatically runs garbage collection every 5 minutes to reclaim space:

```
Value Log GC triggered every 5 minutes
Threshold: 50% (removes files with >50% stale data)
```

**Manual GC** (future):
```bash
curl -X POST http://localhost:8500/admin/gc
```

---

### Disk Space Management

**Monitor disk usage:**
```bash
du -sh ./data
# 128M    ./data

# Breakdown by file type
du -sh ./data/*.vlog ./data/*.sst
# 64M     ./data/000000.vlog
# 32M     ./data/000001.vlog
# 16M     ./data/000000.sst
```

**Compaction:**

BadgerDB automatically compacts LSM-tree levels:
- **L0 → L1:** When 5+ L0 tables exist
- **L1 → L2:** When level size exceeds threshold
- **Background Process:** Non-blocking

**Space reclamation:**
- Delete old data → Space freed after GC
- Compaction → Reduces SSTable count
- Compression → Snappy compression enabled by default

---

## Monitoring

### Metrics

BadgerDB operations are tracked via Prometheus metrics:

```
# Total persistence operations
konsul_persistence_operations_total{operation="set",status="success"} 1000
konsul_persistence_operations_total{operation="get",status="success"} 5000

# Operation latency
konsul_persistence_operation_duration_seconds{operation="set"} 0.001

# Database size
konsul_persistence_database_size_bytes{type="vlog"} 67108864
konsul_persistence_database_size_bytes{type="sst"} 33554432
```

---

### Health Checks

Check persistence health:

```bash
curl http://localhost:8500/health
```

**Response includes persistence stats:**
```json
{
  "status": "healthy",
  "persistence": {
    "enabled": true,
    "type": "badger",
    "healthy": true,
    "disk_usage_bytes": 134217728,
    "num_keys": 1000,
    "num_services": 50
  }
}
```

---

## Best Practices

### 1. Data Directory Location

**Use separate disk for data:**
```bash
# Mount dedicated volume
mount /dev/sdb1 /var/lib/konsul

# Configure Konsul
KONSUL_DATA_DIR=/var/lib/konsul/data ./konsul
```

**Benefits:**
- Isolate I/O from OS disk
- Easier capacity management
- Better performance monitoring

---

### 2. Backup Strategy

**Automated backups:**
```bash
#!/bin/bash
# backup.sh - Run via cron

BACKUP_DIR="/backups/konsul"
DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="$BACKUP_DIR/konsul-$DATE.db"

# Create backup
curl -X POST http://localhost:8500/backup \
  -d "{\"path\": \"$BACKUP_FILE\"}"

# Keep last 7 days
find $BACKUP_DIR -name "konsul-*.db" -mtime +7 -delete
```

**Crontab:**
```
# Backup every 6 hours
0 */6 * * * /usr/local/bin/backup.sh
```

---

### 3. Monitoring Disk Space

**Alert on high usage:**
```bash
# Prometheus alert rule
- alert: KonsulDiskSpaceHigh
  expr: konsul_persistence_database_size_bytes > 10737418240  # 10GB
  for: 5m
  annotations:
    summary: "Konsul database size exceeds 10GB"
```

---

### 4. Capacity Planning

**Estimate storage needs:**

```
Total Size = (KV Data Size) + (Service Data Size) + (Overhead)

KV Data: 1,000 keys × 1KB avg = 1MB
Services: 100 services × 500B = 50KB
Overhead: ~2-3x (LSM amplification)

Estimated: ~3-4MB
```

**Growth rate:**
- Track `konsul_persistence_database_size_bytes` over time
- Plan for 2-3x growth buffer
- Set up alerts at 80% capacity

---

## Troubleshooting

### Issue 1: Database Won't Start

**Symptom:**
```
ERROR: failed to open BadgerDB: Cannot acquire directory lock
```

**Cause:** Another Konsul instance is running or didn't shut down cleanly

**Solution:**
```bash
# Check for running instances
ps aux | grep konsul

# Remove stale lock file (only if no process running!)
rm ./data/LOCK

# Restart
./konsul
```

---

### Issue 2: High Disk Usage

**Symptom:**
```bash
du -sh ./data
# 10G    ./data  # Too large!
```

**Diagnosis:**
```bash
# Check value log size
ls -lh ./data/*.vlog

# Check SSTable count
ls -1 ./data/*.sst | wc -l
```

**Solution:**

**Manual GC trigger** (requires code change):
```go
// In maintenance script
db.RunValueLogGC(0.5)  // Remove files with >50% garbage
```

**Or delete old data:**
```bash
# Remove old keys via API
curl -X DELETE http://localhost:8500/kv/old/data
```

---

### Issue 3: Slow Performance

**Symptom:** Write latency > 10ms

**Diagnosis:**
```bash
# Check sync_writes setting
ps aux | grep konsul | grep SYNC_WRITES

# Monitor disk I/O
iostat -x 1
```

**Solutions:**

**1. Disable sync writes (dev only):**
```bash
KONSUL_SYNC_WRITES=false ./konsul
```

**2. Use faster disk (SSD):**
- Migrate to NVMe SSD
- Enable write caching

**3. Batch operations:**
```bash
# Instead of individual writes:
curl -X POST http://localhost:8500/kv/key1 -d '{"value":"val1"}'
curl -X POST http://localhost:8500/kv/key2 -d '{"value":"val2"}'

# Use batch API:
curl -X POST http://localhost:8500/batch/set -d '{
  "items": {
    "key1": "val1",
    "key2": "val2"
  }
}'
```

---

### Issue 4: Corruption After Crash

**Symptom:**
```
ERROR: failed to open BadgerDB: Corruption detected
```

**Recovery:**

**1. Try auto-recovery:**
```bash
# BadgerDB has built-in recovery
rm ./data/MANIFEST  # Force manifest rebuild
./konsul
```

**2. Restore from backup:**
```bash
# Stop Konsul
systemctl stop konsul

# Remove corrupted data
mv ./data ./data.corrupted

# Restore from latest backup
mkdir -p ./data
curl -X POST http://localhost:8500/restore \
  -d '{"path": "./backups/latest.db"}'

# Start Konsul
systemctl start konsul
```

---

## Migration

### From In-Memory to BadgerDB

**Zero-downtime migration:**

```bash
# 1. Enable persistence while running
curl -X POST http://localhost:8500/admin/config \
  -d '{
    "persistence": {
      "enabled": true,
      "type": "badger"
    }
  }'

# 2. Trigger full sync
curl -X POST http://localhost:8500/admin/sync

# 3. Verify data in database
ls -lh ./data

# 4. Restart with persistence
systemctl restart konsul
```

---

### From BadgerDB v3 to v4

Konsul uses BadgerDB v4. If migrating from older version:

```bash
# 1. Backup old database
./konsul-old export > data-backup.json

# 2. Install new Konsul
wget https://github.com/konsul/releases/latest/konsul

# 3. Import data
./konsul &
sleep 5
curl -X POST http://localhost:8500/import -d @data-backup.json
```

---

## Performance Tuning

### Write-Heavy Workloads

```bash
# Increase memtable size
BADGER_MEMTABLE_SIZE=128MB

# Increase value log file size
BADGER_VLOG_SIZE=128MB

# More L0 tables before compaction
BADGER_L0_TABLES=10
```

### Read-Heavy Workloads

```bash
# Enable block cache
BADGER_BLOCK_CACHE=256MB

# Prefetch values
BADGER_PREFETCH_SIZE=10
```

### Constrained Memory

```bash
# Reduce memtable size
BADGER_MEMTABLE_SIZE=8MB

# Fewer memtables
BADGER_NUM_MEMTABLES=2

# Disable block cache
BADGER_BLOCK_CACHE=0
```

---

## Security

### File Permissions

```bash
# Secure data directory
chmod 700 ./data
chown konsul:konsul ./data

# Secure backup files
chmod 600 ./backups/*.db
```

### Encryption at Rest

**Current:** BadgerDB supports encryption (not yet exposed in Konsul)

**Future:**
```yaml
persistence:
  enabled: true
  encryption:
    enabled: true
    key_file: /etc/konsul/encryption.key
```

---

## See Also

- [Persistence API Reference](persistence-api.md)
- [Persistence Implementation](persistence-implementation.md)
- [Persistence Troubleshooting](persistence-troubleshooting.md)
- [ADR-0002](adr/0002-badger-for-persistence.md)
- [BadgerDB Documentation](https://dgraph.io/docs/badger/)
