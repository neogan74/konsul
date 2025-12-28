# Clustering and High Availability Guide

This guide explains how to deploy Konsul in a clustered configuration for high availability using Raft consensus.

---

## Overview

Konsul supports clustering through the Raft consensus algorithm, providing:

- **High Availability**: Survives node failures automatically
- **Data Replication**: All data replicated across cluster nodes
- **Automatic Failover**: Sub-second leader election on failure
- **Strong Consistency**: Linearizable writes through consensus

### How It Works

```
┌─────────────────────────────────────────────────────────┐
│                  Konsul Cluster (3 nodes)               │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────┐      ┌──────────┐      ┌──────────┐     │
│  │  Node 1  │      │  Node 2  │      │  Node 3  │     │
│  │ (Leader) │◄────►│(Follower)│◄────►│(Follower)│     │
│  └──────────┘      └──────────┘      └──────────┘     │
│       │                 │                 │            │
│       └─────────────────┴─────────────────┘            │
│                   Raft Consensus                       │
│                                                         │
│  Client ──► Any Node ──► Leader ──► Replicate         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

- **Leader**: Handles all write operations, replicates to followers
- **Followers**: Replicate data, can serve read requests
- **Quorum**: Majority of nodes must agree for writes to commit

### Cluster Sizes

| Nodes | Fault Tolerance | Recommended For |
|-------|-----------------|-----------------|
| 3 | 1 failure | Development, small production |
| 5 | 2 failures | Production (recommended) |
| 7 | 3 failures | Large, critical deployments |

> **Note**: Always use an odd number of nodes to avoid split-brain scenarios.

---

## Configuration

### Environment Variables

```bash
# Enable clustering
KONSUL_RAFT_ENABLED=true

# Node identity (must be unique per node)
KONSUL_RAFT_NODE_ID=node1

# Raft bind address (internal cluster communication)
KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000

# Advertise address (how other nodes reach this node)
KONSUL_RAFT_ADVERTISE_ADDR=192.168.1.10:7000

# Data directory for Raft logs and snapshots
KONSUL_RAFT_DATA_DIR=./data/raft

# Bootstrap the cluster (only set true on first node, first time)
KONSUL_RAFT_BOOTSTRAP=true
```

### Advanced Configuration

```bash
# Timeouts (tune for your network latency)
KONSUL_RAFT_HEARTBEAT_TIMEOUT=1s
KONSUL_RAFT_ELECTION_TIMEOUT=1s
KONSUL_RAFT_LEADER_LEASE_TIMEOUT=500ms
KONSUL_RAFT_COMMIT_TIMEOUT=50ms

# Snapshot settings
KONSUL_RAFT_SNAPSHOT_INTERVAL=120s
KONSUL_RAFT_SNAPSHOT_THRESHOLD=8192
KONSUL_RAFT_SNAPSHOT_RETENTION=2

# Log settings
KONSUL_RAFT_MAX_APPEND_ENTRIES=64
KONSUL_RAFT_TRAILING_LOGS=10240
KONSUL_RAFT_LOG_LEVEL=info
```

### Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `KONSUL_RAFT_ENABLED` | `false` | Enable Raft clustering |
| `KONSUL_RAFT_NODE_ID` | (required) | Unique identifier for this node |
| `KONSUL_RAFT_BIND_ADDR` | `0.0.0.0:7000` | Address for Raft to listen on |
| `KONSUL_RAFT_ADVERTISE_ADDR` | (required) | Address other nodes use to connect |
| `KONSUL_RAFT_DATA_DIR` | `./data/raft` | Directory for Raft data |
| `KONSUL_RAFT_BOOTSTRAP` | `false` | Bootstrap a new cluster |
| `KONSUL_RAFT_HEARTBEAT_TIMEOUT` | `1s` | Time between heartbeats |
| `KONSUL_RAFT_ELECTION_TIMEOUT` | `1s` | Time before starting election |
| `KONSUL_RAFT_LEADER_LEASE_TIMEOUT` | `500ms` | Leader lease duration |
| `KONSUL_RAFT_COMMIT_TIMEOUT` | `50ms` | Timeout for commits |
| `KONSUL_RAFT_SNAPSHOT_INTERVAL` | `120s` | Time between snapshots |
| `KONSUL_RAFT_SNAPSHOT_THRESHOLD` | `8192` | Log entries before snapshot |
| `KONSUL_RAFT_SNAPSHOT_RETENTION` | `2` | Number of snapshots to keep |

---

## Deployment

### 3-Node Cluster Quick Start

#### Step 1: Start the First Node (Bootstrap)

```bash
# Node 1 - Bootstrap node
KONSUL_RAFT_ENABLED=true \
KONSUL_RAFT_NODE_ID=node1 \
KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000 \
KONSUL_RAFT_ADVERTISE_ADDR=192.168.1.10:7000 \
KONSUL_RAFT_DATA_DIR=./data/raft \
KONSUL_RAFT_BOOTSTRAP=true \
./konsul
```

Wait for the node to start and elect itself as leader.

#### Step 2: Start Additional Nodes

```bash
# Node 2
KONSUL_RAFT_ENABLED=true \
KONSUL_RAFT_NODE_ID=node2 \
KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000 \
KONSUL_RAFT_ADVERTISE_ADDR=192.168.1.11:7000 \
KONSUL_RAFT_DATA_DIR=./data/raft \
./konsul
```

```bash
# Node 3
KONSUL_RAFT_ENABLED=true \
KONSUL_RAFT_NODE_ID=node3 \
KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000 \
KONSUL_RAFT_ADVERTISE_ADDR=192.168.1.12:7000 \
KONSUL_RAFT_DATA_DIR=./data/raft \
./konsul
```

#### Step 3: Join Nodes to Cluster

```bash
# Join node2 to cluster (execute on leader or any node)
curl -X POST http://192.168.1.10:8500/cluster/join \
  -H "Content-Type: application/json" \
  -d '{"node_id": "node2", "address": "192.168.1.11:7000"}'

# Join node3 to cluster
curl -X POST http://192.168.1.10:8500/cluster/join \
  -H "Content-Type: application/json" \
  -d '{"node_id": "node3", "address": "192.168.1.12:7000"}'
```

#### Step 4: Verify Cluster Status

```bash
curl http://192.168.1.10:8500/cluster/status | jq
```

Expected output:
```json
{
  "node_id": "node1",
  "state": "Leader",
  "leader_id": "node1",
  "leader_addr": "192.168.1.10:7000",
  "peers": [
    {"id": "node1", "address": "192.168.1.10:7000", "voter": true},
    {"id": "node2", "address": "192.168.1.11:7000", "voter": true},
    {"id": "node3", "address": "192.168.1.12:7000", "voter": true}
  ],
  "commit_index": 5,
  "applied_index": 5,
  "last_index": 5
}
```

### Docker Compose Deployment

Create `docker-compose-cluster.yml`:

```yaml
version: '3.8'

services:
  konsul1:
    image: konsul:latest
    environment:
      - KONSUL_HOST=0.0.0.0
      - KONSUL_PORT=8500
      - KONSUL_RAFT_ENABLED=true
      - KONSUL_RAFT_NODE_ID=node1
      - KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000
      - KONSUL_RAFT_ADVERTISE_ADDR=konsul1:7000
      - KONSUL_RAFT_DATA_DIR=/data/raft
      - KONSUL_RAFT_BOOTSTRAP=true
    volumes:
      - konsul1-data:/data
    ports:
      - "8501:8500"
      - "7001:7000"
    networks:
      - konsul-net

  konsul2:
    image: konsul:latest
    environment:
      - KONSUL_HOST=0.0.0.0
      - KONSUL_PORT=8500
      - KONSUL_RAFT_ENABLED=true
      - KONSUL_RAFT_NODE_ID=node2
      - KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000
      - KONSUL_RAFT_ADVERTISE_ADDR=konsul2:7000
      - KONSUL_RAFT_DATA_DIR=/data/raft
    volumes:
      - konsul2-data:/data
    ports:
      - "8502:8500"
      - "7002:7000"
    networks:
      - konsul-net
    depends_on:
      - konsul1

  konsul3:
    image: konsul:latest
    environment:
      - KONSUL_HOST=0.0.0.0
      - KONSUL_PORT=8500
      - KONSUL_RAFT_ENABLED=true
      - KONSUL_RAFT_NODE_ID=node3
      - KONSUL_RAFT_BIND_ADDR=0.0.0.0:7000
      - KONSUL_RAFT_ADVERTISE_ADDR=konsul3:7000
      - KONSUL_RAFT_DATA_DIR=/data/raft
    volumes:
      - konsul3-data:/data
    ports:
      - "8503:8500"
      - "7003:7000"
    networks:
      - konsul-net
    depends_on:
      - konsul1

volumes:
  konsul1-data:
  konsul2-data:
  konsul3-data:

networks:
  konsul-net:
    driver: bridge
```

Start the cluster:

```bash
docker-compose -f docker-compose-cluster.yml up -d

# Wait for konsul1 to start, then join other nodes
sleep 10

# Join node2
curl -X POST http://localhost:8501/cluster/join \
  -H "Content-Type: application/json" \
  -d '{"node_id": "node2", "address": "konsul2:7000"}'

# Join node3
curl -X POST http://localhost:8501/cluster/join \
  -H "Content-Type: application/json" \
  -d '{"node_id": "node3", "address": "konsul3:7000"}'
```

---

## Cluster Management API

### Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/cluster/status` | GET | Full cluster status and diagnostics |
| `/cluster/leader` | GET | Current leader information |
| `/cluster/peers` | GET | List all cluster members |
| `/cluster/join` | POST | Add a new node to cluster |
| `/cluster/leave/:id` | DELETE | Remove a node from cluster |
| `/cluster/snapshot` | POST | Trigger a manual snapshot |

### Get Cluster Status

```bash
curl http://localhost:8500/cluster/status
```

Response:
```json
{
  "node_id": "node1",
  "state": "Leader",
  "leader_id": "node1",
  "leader_addr": "192.168.1.10:7000",
  "peers": [...],
  "commit_index": 1234,
  "applied_index": 1234,
  "last_index": 1234,
  "fsm_pending": 0
}
```

### Get Current Leader

```bash
curl http://localhost:8500/cluster/leader
```

Response:
```json
{
  "leader_id": "node1",
  "leader_addr": "192.168.1.10:7000"
}
```

### List Cluster Peers

```bash
curl http://localhost:8500/cluster/peers
```

Response:
```json
{
  "peers": [
    {"id": "node1", "address": "192.168.1.10:7000", "voter": true},
    {"id": "node2", "address": "192.168.1.11:7000", "voter": true},
    {"id": "node3", "address": "192.168.1.12:7000", "voter": true}
  ]
}
```

### Join a Node

```bash
curl -X POST http://localhost:8500/cluster/join \
  -H "Content-Type: application/json" \
  -d '{"node_id": "node4", "address": "192.168.1.13:7000"}'
```

### Remove a Node

```bash
curl -X DELETE http://localhost:8500/cluster/leave/node4
```

### Trigger Snapshot

```bash
curl -X POST http://localhost:8500/cluster/snapshot
```

---

## Client Behavior

### Write Operations

Write operations (KV set/delete, service register/deregister) must go through the leader:

1. Client sends write to any node
2. If node is not leader, returns `307 Temporary Redirect` with leader address
3. Client should retry request to leader

Example response when hitting a follower:

```json
{
  "error": "not leader",
  "message": "This node is not the leader. Redirect to leader for write operations.",
  "leader_addr": "192.168.1.10:7000"
}
```

### Read Operations

Read operations can be served by any node (eventual consistency):

```bash
# Read from any node
curl http://192.168.1.11:8500/kv/mykey
```

### Handling Redirects

Example client code (Go):

```go
func kvSet(baseURL, key, value string) error {
    for attempts := 0; attempts < 3; attempts++ {
        resp, err := http.Post(
            fmt.Sprintf("%s/kv/%s", baseURL, key),
            "application/json",
            strings.NewReader(fmt.Sprintf(`{"value":"%s"}`, value)),
        )
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode == http.StatusTemporaryRedirect {
            var result map[string]interface{}
            json.NewDecoder(resp.Body).Decode(&result)
            // Redirect to leader
            baseURL = fmt.Sprintf("http://%s", result["leader_addr"])
            continue
        }

        if resp.StatusCode == http.StatusOK {
            return nil
        }
    }
    return fmt.Errorf("failed after 3 attempts")
}
```

---

## Monitoring

### Prometheus Metrics

Raft exports the following metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `konsul_raft_state` | Gauge | Current Raft state (0=Follower, 1=Candidate, 2=Leader) |
| `konsul_raft_is_leader` | Gauge | 1 if this node is leader, 0 otherwise |
| `konsul_raft_peers_total` | Gauge | Number of peers in cluster |
| `konsul_raft_last_index` | Gauge | Last log index |
| `konsul_raft_commit_index` | Gauge | Committed log index |
| `konsul_raft_applied_index` | Gauge | Applied log index |
| `konsul_raft_apply_total` | Counter | Total apply operations |
| `konsul_raft_apply_errors_total` | Counter | Total apply errors |
| `konsul_raft_apply_duration_seconds` | Histogram | Apply operation duration |
| `konsul_raft_leader_changes_total` | Counter | Number of leader elections |
| `konsul_raft_snapshot_total` | Counter | Number of snapshots taken |

### Example Prometheus Alerts

```yaml
groups:
  - name: konsul-raft
    rules:
      - alert: KonsulNoLeader
        expr: sum(konsul_raft_is_leader) == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Konsul cluster has no leader"

      - alert: KonsulFrequentLeaderChanges
        expr: increase(konsul_raft_leader_changes_total[5m]) > 3
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Frequent leader elections detected"

      - alert: KonsulReplicationLag
        expr: konsul_raft_last_index - konsul_raft_applied_index > 1000
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "High replication lag on Konsul node"
```

### Health Checks

The `/health` endpoint includes cluster status:

```bash
curl http://localhost:8500/health
```

Response includes:
```json
{
  "status": "healthy",
  "cluster": {
    "enabled": true,
    "state": "Leader",
    "leader": "node1"
  }
}
```

---

## Operations

### Adding a Node

1. Start the new node with Raft enabled (without bootstrap)
2. Join it to the cluster via API
3. Wait for it to catch up with the log

```bash
# On new node
KONSUL_RAFT_ENABLED=true \
KONSUL_RAFT_NODE_ID=node4 \
KONSUL_RAFT_ADVERTISE_ADDR=192.168.1.13:7000 \
./konsul

# Join to cluster
curl -X POST http://192.168.1.10:8500/cluster/join \
  -d '{"node_id": "node4", "address": "192.168.1.13:7000"}'
```

### Removing a Node

1. Gracefully stop the node if possible
2. Remove from cluster via API

```bash
# Remove node
curl -X DELETE http://192.168.1.10:8500/cluster/leave/node4

# Then stop the node
```

### Rolling Restart

To restart nodes without downtime:

1. Restart followers first, one at a time
2. Wait for each to rejoin before continuing
3. Restart leader last (will trigger election)

```bash
# 1. Restart node3 (follower)
ssh node3 'systemctl restart konsul'
sleep 30  # Wait for rejoin

# 2. Restart node2 (follower)
ssh node2 'systemctl restart konsul'
sleep 30

# 3. Restart node1 (leader) - triggers election
ssh node1 'systemctl restart konsul'
```

### Backup and Restore

Raft data is stored in the configured data directory. To backup:

```bash
# Trigger snapshot first
curl -X POST http://localhost:8500/cluster/snapshot

# Backup the data directory
tar -czf konsul-raft-backup.tar.gz ./data/raft
```

---

## Troubleshooting

### No Leader Elected

**Symptoms**: Cluster shows no leader, writes fail

**Causes**:
- Not enough nodes running (need majority)
- Network partition
- Misconfigured advertise addresses

**Solutions**:
1. Ensure majority of nodes are running
2. Check network connectivity between nodes
3. Verify `KONSUL_RAFT_ADVERTISE_ADDR` is reachable from other nodes

### Frequent Leader Elections

**Symptoms**: `konsul_raft_leader_changes_total` increasing rapidly

**Causes**:
- Network instability
- Node overload (slow disk, CPU)
- Timeouts too aggressive

**Solutions**:
1. Check network latency between nodes
2. Increase timeout values:
   ```bash
   KONSUL_RAFT_HEARTBEAT_TIMEOUT=2s
   KONSUL_RAFT_ELECTION_TIMEOUT=2s
   ```
3. Check node resources (disk I/O, CPU)

### Node Won't Join

**Symptoms**: Join request fails or times out

**Causes**:
- Node ID already exists
- Address not reachable
- Not sending to leader

**Solutions**:
1. Use unique node IDs
2. Ensure Raft port (7000) is open
3. Send join request to current leader

### High Replication Lag

**Symptoms**: `applied_index` far behind `last_index`

**Causes**:
- Slow disk on follower
- Network bandwidth limitation
- Large operations

**Solutions**:
1. Check disk performance on lagging node
2. Check network throughput
3. Consider triggering a snapshot

### Split Brain

**Symptoms**: Multiple nodes claim to be leader

**Causes**:
- Network partition
- Incorrect quorum size

**Solutions**:
1. Fix network partition
2. Ensure odd number of nodes
3. Nodes without quorum will step down automatically

---

## Best Practices

### Network Configuration

- Use dedicated network for Raft traffic
- Ensure low latency between nodes (<10ms recommended)
- Open Raft port (default 7000) between all nodes

### Storage

- Use SSDs for Raft data directory
- Ensure sufficient disk space for logs and snapshots
- Monitor disk usage

### Security

- Run Raft on private network
- Use TLS for Raft transport (coming soon)
- Implement join tokens (coming soon)

### Monitoring

- Alert on no leader
- Alert on frequent elections
- Monitor replication lag
- Track commit latency

---

## Migration from Standalone

To migrate from a standalone Konsul to a cluster:

1. **Export existing data**:
   ```bash
   curl http://standalone:8500/export > backup.json
   ```

2. **Deploy cluster** (without data)

3. **Import data to cluster**:
   ```bash
   curl -X POST http://cluster-leader:8500/import \
     -H "Content-Type: application/json" \
     -d @backup.json
   ```

4. **Switch traffic** to cluster
5. **Decommission** standalone instance

---

## See Also

- [ADR-0011: Raft Clustering](adr/0011-raft-clustering-ha.md) - Architecture decision
- [ADR-0030: Raft Implementation Status](adr/0030-raft-integration-implementation.md) - Current implementation status (Phase 1)
- [ADR-0031: Raft Production Readiness](adr/0031-raft-production-readiness.md) - Phase 2 roadmap and production features
- [Metrics Guide](metrics.md) - Prometheus metrics reference
- [Deployment Guide](deployment.md) - General deployment information