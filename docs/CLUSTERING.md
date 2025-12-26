# Raft Clustering (MVP)

This is a minimal Raft integration aimed at 3-node clusters. Writes must be
handled by the leader. Non-leader nodes respond with `503` and include the
current leader address.

## Limitations
- No dynamic join/leave API yet.
- No automatic leader redirect (client must retry against leader).

## 3-Node Local Example

Start three nodes with unique data dirs and raft ports:

```bash
# Node 1 (bootstrap)
KONSUL_PORT=8888 \
KONSUL_RAFT_ENABLED=true \
KONSUL_RAFT_NODE_ID=node1 \
KONSUL_RAFT_BIND_ADDR=127.0.0.1:7001 \
KONSUL_RAFT_DATA_DIR=./data/raft/node1 \
KONSUL_RAFT_BOOTSTRAP=true \
KONSUL_RAFT_PEERS="node1@127.0.0.1:7001,node2@127.0.0.1:7002,node3@127.0.0.1:7003" \
./konsul

# Node 2
KONSUL_PORT=8889 \
KONSUL_RAFT_ENABLED=true \
KONSUL_RAFT_NODE_ID=node2 \
KONSUL_RAFT_BIND_ADDR=127.0.0.1:7002 \
KONSUL_RAFT_DATA_DIR=./data/raft/node2 \
./konsul

# Node 3
KONSUL_PORT=8890 \
KONSUL_RAFT_ENABLED=true \
KONSUL_RAFT_NODE_ID=node3 \
KONSUL_RAFT_BIND_ADDR=127.0.0.1:7003 \
KONSUL_RAFT_DATA_DIR=./data/raft/node3 \
./konsul
```

If nodes are on different hosts, set `KONSUL_RAFT_ADVERTISE_ADDR` to the
reachable address for each node.
