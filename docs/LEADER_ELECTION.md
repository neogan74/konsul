# Leader Election Implementation

**Status**: ✅ Implemented (BACK-002)
**Priority**: P0 (Critical)
**Related**: [CLUSTERING.md](CLUSTERING.md), [ADR-0011](../docs/adr/0011-raft-clustering-ha.md)

## Overview

This document describes the Leader Election implementation for Konsul's Raft-based clustering. Leader election is a critical component of the high-availability system, ensuring automatic failover and cluster coordination.

## Implementation Details

### Configuration

Leader election behavior is controlled by three key timeouts configured in `internal/raft/config.go`:

```go
// HeartbeatTimeout is the time between heartbeats from the leader.
// Default: 1s
HeartbeatTimeout: 1000 * time.Millisecond

// ElectionTimeout is the timeout for starting a new election.
// Should be larger than HeartbeatTimeout.
// Default: 1s
ElectionTimeout: 1000 * time.Millisecond

// LeaderLeaseTimeout is how long a leader will hold leadership without
// being able to contact a quorum.
// Default: 500ms
LeaderLeaseTimeout: 500 * time.Millisecond
```

### Environment Variables

These timeouts can be configured via environment variables:

```bash
KONSUL_RAFT_HEARTBEAT_TIMEOUT=1s
KONSUL_RAFT_ELECTION_TIMEOUT=1s
KONSUL_RAFT_LEADER_LEASE_TIMEOUT=500ms
```

### How Leader Election Works

1. **Initial Bootstrap**: When a cluster is first created, the bootstrap node (with `KONSUL_RAFT_BOOTSTRAP=true`) automatically becomes the leader.

2. **Heartbeat Mechanism**: The leader sends periodic heartbeats to all follower nodes at intervals defined by `HeartbeatTimeout`.

3. **Election Trigger**: If followers don't receive heartbeats within `ElectionTimeout`, they transition to the candidate state and start an election.

4. **Voting**: Candidates request votes from other nodes. A node becomes leader when it receives votes from a majority (quorum) of the cluster.

5. **Leader Lease**: The leader maintains its position as long as it can communicate with a quorum within `LeaderLeaseTimeout`.

6. **Automatic Failover**: If the leader fails or becomes unreachable, a new election is triggered automatically (typically completes in <300ms).

## Metrics

The implementation exports the following Prometheus metrics for monitoring leader elections:

### State Metrics

- **`konsul_raft_state{node_id}`** (Gauge)
  - Current Raft state: 0=Follower, 1=Candidate, 2=Leader, 3=Shutdown
  - Labels: `node_id`

- **`konsul_raft_is_leader`** (Gauge)
  - Whether this node is the leader: 1=yes, 0=no

### Election Metrics

- **`konsul_raft_leader_changes_total`** (Counter)
  - Total number of leader changes in the cluster
  - Incremented whenever a node transitions to/from leader state
  - High values indicate cluster instability

### Additional Metrics

- **`konsul_raft_peers_total`** (Gauge)
  - Number of peers in the cluster

- **`konsul_raft_last_index`** (Gauge)
  - Last log index

- **`konsul_raft_commit_index`** (Gauge)
  - Last committed log index

- **`konsul_raft_applied_index`** (Gauge)
  - Last applied log index

## API Endpoints

### Get Current Leader

```bash
GET /cluster/leader
```

**Response:**
```json
{
  "leader_id": "node1",
  "leader_addr": "192.168.1.10:7000",
  "is_self": true
}
```

**Error (No Leader):**
```json
{
  "error": "no leader",
  "message": "No leader is currently elected. The cluster may be initializing or partitioned."
}
```

### Get Cluster Status

```bash
GET /cluster/status
```

**Response includes:**
```json
{
  "node_id": "node1",
  "state": "Leader",
  "leader_id": "node1",
  "leader_addr": "192.168.1.10:7000",
  "peers": [...]
}
```

## Monitoring

### Prometheus Alerts

Example alerts for monitoring leader elections:

```yaml
groups:
  - name: konsul-leader-election
    rules:
      # Alert when cluster has no leader
      - alert: KonsulNoLeader
        expr: sum(konsul_raft_is_leader) == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Konsul cluster has no leader"
          description: "The cluster has been without a leader for 30 seconds"

      # Alert on frequent leader changes (instability)
      - alert: KonsulFrequentLeaderChanges
        expr: increase(konsul_raft_leader_changes_total[5m]) > 3
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Frequent leader elections detected"
          description: "{{ $value }} leader changes in the last 5 minutes"

      # Alert on leader election taking too long
      - alert: KonsulLeaderElectionSlow
        expr: konsul_raft_state == 1
        for: 30s
        labels:
          severity: warning
        annotations:
          summary: "Leader election taking too long"
          description: "Node {{ $labels.node_id }} has been in Candidate state for 30+ seconds"
```

### Grafana Visualization

```promql
# Current leader (should always be 1)
sum(konsul_raft_is_leader)

# Leader changes over time
increase(konsul_raft_leader_changes_total[1h])

# Cluster state distribution
count by (state) (konsul_raft_state)
```

## Tuning Guidelines

### Network Latency

For clusters with higher network latency (e.g., cross-region):

```bash
KONSUL_RAFT_HEARTBEAT_TIMEOUT=2s
KONSUL_RAFT_ELECTION_TIMEOUT=2s
KONSUL_RAFT_LEADER_LEASE_TIMEOUT=1s
```

### Low-Latency Networks

For clusters on fast local networks:

```bash
KONSUL_RAFT_HEARTBEAT_TIMEOUT=500ms
KONSUL_RAFT_ELECTION_TIMEOUT=1s
KONSUL_RAFT_LEADER_LEASE_TIMEOUT=250ms
```

### General Rules

1. **ElectionTimeout ≥ HeartbeatTimeout**: Prevents false elections
2. **LeaderLeaseTimeout ≤ HeartbeatTimeout**: Ensures timely failure detection
3. **Network Latency**: Set timeouts to 5-10x your network RTT
4. **Cluster Size**: Larger clusters may need slightly higher timeouts

## Implementation Status

### ✅ Completed

1. Timeout configuration (HeartbeatTimeout, ElectionTimeout, LeaderLeaseTimeout)
2. Metrics collection system
3. State monitoring goroutine
4. Leader change detection
5. `/cluster/leader` endpoint
6. Prometheus metrics export:
   - `konsul_raft_state`
   - `konsul_raft_is_leader`
   - `konsul_raft_leader_changes_total`
   - `konsul_raft_peers_total`
   - Index metrics (last, commit, applied)
7. Unit tests for Node creation and metrics
8. Documentation

### ⏳ Pending

1. Fix duplicate payload definitions in raft package (existing issue)
2. Integration tests for 3-node election scenarios
3. Network partition tests
4. Performance benchmarks (target: <300ms election time at p99)

## Testing

### Manual Testing

1. **Start 3-node cluster**:
```bash
# Node 1 (bootstrap)
KONSUL_RAFT_ENABLED=true KONSUL_RAFT_NODE_ID=node1 \
KONSUL_RAFT_ADVERTISE_ADDR=127.0.0.1:7001 \
KONSUL_RAFT_BOOTSTRAP=true ./konsul

# Node 2
KONSUL_RAFT_ENABLED=true KONSUL_RAFT_NODE_ID=node2 \
KONSUL_RAFT_ADVERTISE_ADDR=127.0.0.1:7002 ./konsul

# Node 3
KONSUL_RAFT_ENABLED=true KONSUL_RAFT_NODE_ID=node3 \
KONSUL_RAFT_ADVERTISE_ADDR=127.0.0.1:7003 ./konsul
```

2. **Join nodes to cluster**:
```bash
curl -X POST http://localhost:8500/cluster/join \
  -d '{"node_id": "node2", "address": "127.0.0.1:7002"}'

curl -X POST http://localhost:8500/cluster/join \
  -d '{"node_id": "node3", "address": "127.0.0.1:7003"}'
```

3. **Check leader**:
```bash
curl http://localhost:8500/cluster/leader | jq
```

4. **Monitor metrics**:
```bash
curl http://localhost:8080/metrics | grep konsul_raft
```

5. **Test failover** (kill leader and observe re-election):
```bash
# Kill the leader process
pkill -9 konsul

# Check new leader is elected
curl http://localhost:8501/cluster/leader | jq
```

## Troubleshooting

### No Leader Elected

**Symptoms**: `sum(konsul_raft_is_leader) == 0`

**Possible Causes**:
- Not enough nodes running (need majority)
- Network partition
- Misconfigured advertise addresses

**Solutions**:
1. Check that majority of nodes are running
2. Verify network connectivity between nodes
3. Check logs for election timeout messages
4. Increase `ElectionTimeout` if network is slow

### Frequent Leader Changes

**Symptoms**: `increase(konsul_raft_leader_changes_total[5m]) > 3`

**Possible Causes**:
- Network instability
- Node overload (slow disk, CPU)
- Timeouts too aggressive for network conditions

**Solutions**:
1. Check network latency between nodes (`ping`)
2. Increase timeout values
3. Check node resources (disk I/O, CPU usage)
4. Review system logs for errors

### Split-Brain (Multiple Leaders)

**Symptoms**: `sum(konsul_raft_is_leader) > 1`

**Note**: This should be impossible with Raft consensus, but indicates a serious bug if observed.

**Actions**:
1. Immediately investigate logs
2. Check for clock skew between nodes
3. Verify network partitioning rules
4. Restart cluster if necessary

## References

- [ADR-0011: Raft Clustering](../docs/adr/0011-raft-clustering-ha.md)
- [CLUSTERING.md](CLUSTERING.md) - General clustering guide
- [Raft Paper](https://raft.github.io/raft.pdf)
- [HashiCorp Raft](https://github.com/hashicorp/raft)
- [BACKLOG.md](BACKLOG.md) - BACK-002 task details

## Code References

- Configuration: `internal/raft/config.go`
- Metrics: `internal/raft/metrics.go`
- Node implementation: `internal/raft/node.go:566-633` (monitorState goroutine)
- Cluster handler: `internal/handlers/cluster.go:62-84` (/cluster/leader endpoint)
- Tests: `internal/raft/node_test.go`