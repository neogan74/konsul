package raft

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRaftCluster_TLS(t *testing.T) {
	// Create a 3-node cluster with TLS enabled
	dir := t.TempDir()

	// Generate CA
	caCert, _, caObj, caKey := generateCert(t, dir, "ca", true, nil, nil)
	_ = caKey // Key used for signing

	// Create configs for 3 nodes
	nodes := make([]*Node, 3)
	configs := make([]*Config, 3)

	ports := []int{17001, 17002, 17003}

	// Store stores for verification
	kvStores := make([]*MockKVStore, 3)

	for i := 0; i < 3; i++ {
		nodeID := fmt.Sprintf("node-%d", i+1)

		// Generate cert for this node
		certFile, keyFile, _, _ := generateCert(t, dir, nodeID, false, caObj, caKey)

		cfg := DefaultConfig()
		cfg.NodeID = nodeID
		cfg.BindAddr = fmt.Sprintf("127.0.0.1:%d", ports[i])
		cfg.DataDir = fmt.Sprintf("%s/data-%d", dir, i)
		cfg.Bootstrap = (i == 0) // Only first node bootstraps
		cfg.LogLevel = "error"   // Keep logs quiet

		// Enable mTLS
		cfg.TLS.Enabled = true
		cfg.TLS.CertFile = certFile
		cfg.TLS.KeyFile = keyFile
		cfg.TLS.CAFile = caCert
		cfg.TLS.VerifyPeer = true
		cfg.TLS.MinVersion = "1.2"

		configs[i] = cfg

		// Create node
		kvStore := NewMockKVStore()
		kvStores[i] = kvStore
		svcStore := NewMockServiceStore()

		node, err := NewNode(cfg, kvStore, svcStore)
		require.NoError(t, err)
		nodes[i] = node
	}

	// Shutdown nodes at the end
	defer func() {
		for _, n := range nodes {
			n.Shutdown()
		}
	}()

	// Wait for Leader on Node 1 (Bootstrap node)
	t.Log("Waiting for Node 1 to become leader...")
	err := nodes[0].WaitForLeader(10 * time.Second)
	require.NoError(t, err)
	assert.True(t, nodes[0].IsLeader())

	// Join other nodes to the cluster
	t.Log("Joining other nodes to the cluster...")
	for i := 1; i < 3; i++ {
		// Join command sent to Leader (Node 1)
		err := nodes[0].Join(configs[i].NodeID, configs[i].BindAddr)
		require.NoError(t, err)
	}

	// Wait for cluster to stabilize
	time.Sleep(5 * time.Second)

	// Verify all nodes see the leader
	leaderID := nodes[0].LeaderID()
	for i := 0; i < 3; i++ {
		assert.Equal(t, leaderID, nodes[i].LeaderID(), "Node %d should agree on leader", i+1)
	}

	// Verify Replication
	t.Log("Verifying replication...")
	key := "secure-key"
	value := "encrypted-value"

	// Write to leader
	err = nodes[0].KVSet(key, value)
	require.NoError(t, err)

	// Read from follower (eventually consistent)
	require.Eventually(t, func() bool {
		for i := 1; i < 3; i++ {
			val, found, err := kvStores[i].Get(key)
			if err != nil || !found || val != value {
				return false
			}
		}
		return true
	}, 10*time.Second, 100*time.Millisecond, "All nodes should receive the update")
}
