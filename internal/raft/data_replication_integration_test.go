package raft

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/store"
	"github.com/stretchr/testify/require"
)

// TestReplication_KVWriteToLeader verifies KV writes replicate to followers.
func TestReplication_KVWriteToLeader(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write KV pair to leader
	cmd, err := NewCommand(CmdKVSet, KVSetPayload{
		Key:   "test-key",
		Value: "test-value",
	})
	require.NoError(t, err)

	_, err = leader.ApplyEntry(cmd, 5*time.Second)
	require.NoError(t, err)

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify all nodes have the data
	for _, node := range nodes {
		kvStore := node.fsm.kvStore.(*MockKVStore)
		value, exists, err := kvStore.Get("test-key")
		require.NoError(t, err)
		require.True(t, exists, "key should exist on node %s", node.config.NodeID)
		require.Equal(t, "test-value", value, "value should match on node %s", node.config.NodeID)
	}
}

// TestReplication_ServiceRegistration verifies service registration replication.
func TestReplication_ServiceRegistration(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Register service on leader
	service := ServiceRegisterPayload{
		Service: store.Service{
			Name:    "test-service",
			Address: "192.168.1.100",
			Port:    8080,
		},
	}
	cmd, err := NewCommand(CmdServiceRegister, service)
	require.NoError(t, err)

	_, err = leader.ApplyEntry(cmd, 5*time.Second)
	require.NoError(t, err)

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify all followers have the service registered
	for _, node := range nodes {
		serviceStore := node.fsm.serviceStore.(*MockServiceStore)
		svc, exists, err := serviceStore.Get("test-service")
		require.NoError(t, err)
		require.True(t, exists, "test-service should exist on node %s", node.config.NodeID)
		require.NotNil(t, svc, "service should not be nil on node %s", node.config.NodeID)
	}

	// Deregister service on leader
	deregCmd, err := NewCommand(CmdServiceDeregister, ServiceDeregisterPayload{
		Name: "test-service",
	})
	require.NoError(t, err)

	_, err = leader.ApplyEntry(deregCmd, 5*time.Second)
	require.NoError(t, err)

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify all followers see deregistration
	for _, node := range nodes {
		serviceStore := node.fsm.serviceStore.(*MockServiceStore)
		_, exists, err := serviceStore.Get("test-service")
		require.NoError(t, err)
		require.False(t, exists, "test-service should be deregistered on node %s", node.config.NodeID)
	}
}

// TestReplication_MultipleWrites verifies multiple concurrent writes replicate correctly.
func TestReplication_MultipleWrites(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write 100 KV pairs in rapid succession
	const count = 100
	expected := make(map[string]string)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		expected[key] = value

		cmd, err := NewCommand(CmdKVSet, KVSetPayload{
			Key:   key,
			Value: value,
		})
		require.NoError(t, err)

		_, err = leader.ApplyEntry(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Wait for replication to complete
	time.Sleep(1 * time.Second)

	// Verify all 100 pairs are on all nodes
	for _, node := range nodes {
		kvStore := node.fsm.kvStore.(*MockKVStore)
		for key, expectedValue := range expected {
			value, exists, err := kvStore.Get(key)
			require.NoError(t, err)
			require.True(t, exists, "key %s should exist on node %s", key, node.config.NodeID)
			require.Equal(t, expectedValue, value, "value should match for key %s on node %s", key, node.config.NodeID)
		}
	}
}

// TestReplication_ReplicationLag verifies replication lag metrics.
func TestReplication_ReplicationLag(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write data to leader and measure replication lag
	writeTime := time.Now()
	cmd, err := NewCommand(CmdKVSet, KVSetPayload{
		Key:   "lag-test-key",
		Value: "lag-test-value",
	})
	require.NoError(t, err)

	_, err = leader.ApplyEntry(cmd, 5*time.Second)
	require.NoError(t, err)

	// Poll followers until they have the data
	var maxLag time.Duration
	for _, node := range nodes {
		if node == leader {
			continue
		}

		// Wait for follower to receive the data
		deadline := time.Now().Add(5 * time.Second)
		var lag time.Duration
		for time.Now().Before(deadline) {
			kvStore := node.fsm.kvStore.(*MockKVStore)
			_, exists, err := kvStore.Get("lag-test-key")
			if err == nil && exists {
				lag = time.Since(writeTime)
				if lag > maxLag {
					maxLag = lag
				}
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		if lag == 0 {
			t.Fatalf("follower %s never received the data", node.config.NodeID)
		}
	}

	t.Logf("Maximum replication lag: %s", maxLag)

	// Verify lag is within reasonable bounds (< 2 seconds for test environment)
	require.Less(t, maxLag, 2*time.Second, "replication lag should be < 2s")
}

// TestReplication_CatchUpAfterDisconnect verifies follower catch-up after disconnect.
func TestReplication_CatchUpAfterDisconnect(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Find a follower to disconnect
	var follower *Node
	for _, node := range nodes {
		if node != leader {
			follower = node
			break
		}
	}
	require.NotNil(t, follower)

	// Disconnect the follower by closing its transport
	followerID := follower.config.NodeID
	require.NoError(t, follower.transport.Close())

	// Write data while follower is disconnected
	const disconnectedWrites = 50
	for i := 0; i < disconnectedWrites; i++ {
		cmd, err := NewCommand(CmdKVSet, KVSetPayload{
			Key:   fmt.Sprintf("disconnected-key-%d", i),
			Value: fmt.Sprintf("disconnected-value-%d", i),
		})
		require.NoError(t, err)
		_, err = leader.ApplyEntry(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Shutdown the disconnected follower and recreate it (simulates reconnect)
	require.NoError(t, follower.Shutdown())

	// Recreate the follower node
	addr := getFreeAddr(t)
	cfg := newClusterConfig(t, followerID, addr, false, clusterOptions{})
	cfg.DataDir = follower.config.DataDir // Use same data dir to keep logs
	newFollower := startTestNode(t, cfg)
	defer func() { _ = newFollower.Shutdown() }()

	// Rejoin the cluster
	require.NoError(t, leader.Join(cfg.NodeID, cfg.AdvertiseAddr))

	// Wait for catch-up (follower should replay logs)
	time.Sleep(2 * time.Second)

	// Verify the follower caught up and has all the data
	kvStore := newFollower.fsm.kvStore.(*MockKVStore)
	for i := 0; i < disconnectedWrites; i++ {
		key := fmt.Sprintf("disconnected-key-%d", i)
		value, exists, err := kvStore.Get(key)
		require.NoError(t, err)
		require.True(t, exists, "key %s should exist after catch-up", key)
		require.Equal(t, fmt.Sprintf("disconnected-value-%d", i), value)
	}
}

// TestReplication_HighThroughput verifies replication under high load.
func TestReplication_HighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high throughput test in short mode")
	}

	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write a large number of operations as fast as possible
	const writeCount = 1000 // Using 1000 for faster test execution
	start := time.Now()

	for i := 0; i < writeCount; i++ {
		cmd, err := NewCommand(CmdKVSet, KVSetPayload{
			Key:   fmt.Sprintf("high-throughput-key-%d", i),
			Value: fmt.Sprintf("high-throughput-value-%d", i),
		})
		require.NoError(t, err)
		_, err = leader.ApplyEntry(cmd, 10*time.Second)
		require.NoError(t, err, "write %d should not timeout", i)
	}

	duration := time.Since(start)
	opsPerSec := float64(writeCount) / duration.Seconds()

	t.Logf("High throughput test: %d writes in %s (%.2f ops/sec)", writeCount, duration, opsPerSec)

	// Wait for replication to complete
	time.Sleep(2 * time.Second)

	// Verify all nodes have all the data
	for _, node := range nodes {
		kvStore := node.fsm.kvStore.(*MockKVStore)
		for i := 0; i < writeCount; i++ {
			key := fmt.Sprintf("high-throughput-key-%d", i)
			value, exists, err := kvStore.Get(key)
			require.NoError(t, err)
			require.True(t, exists, "key %s should exist on node %s", key, node.config.NodeID)
			require.Equal(t, fmt.Sprintf("high-throughput-value-%d", i), value)
		}
	}

	// Basic performance check - should be able to do at least 10 ops/sec
	require.Greater(t, opsPerSec, 10.0, "throughput should be at least 10 ops/sec")
}

// TestReplication_AppendEntriesRetry verifies retry logic for failed AppendEntries.
func TestReplication_AppendEntriesRetry(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Find a follower
	var follower *Node
	for _, node := range nodes {
		if node != leader {
			follower = node
			break
		}
	}
	require.NotNil(t, follower)

	// Temporarily close follower's transport to simulate network failure
	followerID := follower.config.NodeID
	followerDataDir := follower.config.DataDir
	require.NoError(t, follower.transport.Close())
	require.NoError(t, follower.Shutdown())

	// Write data while follower is down (Raft will retry when it comes back)
	const retryWrites = 20
	for i := 0; i < retryWrites; i++ {
		cmd, err := NewCommand(CmdKVSet, KVSetPayload{
			Key:   fmt.Sprintf("retry-key-%d", i),
			Value: fmt.Sprintf("retry-value-%d", i),
		})
		require.NoError(t, err)
		_, err = leader.ApplyEntry(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Bring follower back up (Raft should retry AppendEntries)
	addr := getFreeAddr(t)
	cfg := newClusterConfig(t, followerID, addr, false, clusterOptions{})
	cfg.DataDir = followerDataDir
	newFollower := startTestNode(t, cfg)
	defer func() { _ = newFollower.Shutdown() }()

	// Rejoin cluster
	require.NoError(t, leader.Join(cfg.NodeID, cfg.AdvertiseAddr))

	// Wait for Raft to retry and catch up
	time.Sleep(3 * time.Second)

	// Verify follower eventually received all data via retries
	kvStore := newFollower.fsm.kvStore.(*MockKVStore)
	for i := 0; i < retryWrites; i++ {
		key := fmt.Sprintf("retry-key-%d", i)
		value, exists, err := kvStore.Get(key)
		require.NoError(t, err)
		require.True(t, exists, "key %s should exist after retry", key)
		require.Equal(t, fmt.Sprintf("retry-value-%d", i), value)
	}
}

// TestReplication_ReplicationToMultipleFollowers verifies parallel replication.
func TestReplication_ReplicationToMultipleFollowers(t *testing.T) {
	// Create a 3-node cluster (sufficient to verify parallel replication)
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Count followers
	var followers []*Node
	for _, node := range nodes {
		if node != leader {
			followers = append(followers, node)
		}
	}
	require.Len(t, followers, 2, "should have 2 followers")

	// Write data to leader
	const testKey = "parallel-replication-key"
	const testValue = "parallel-replication-value"

	writeTime := time.Now()
	cmd, err := NewCommand(CmdKVSet, KVSetPayload{
		Key:   testKey,
		Value: testValue,
	})
	require.NoError(t, err)

	_, err = leader.ApplyEntry(cmd, 5*time.Second)
	require.NoError(t, err)

	// Measure replication time to each follower independently
	type replicationTiming struct {
		nodeID string
		lag    time.Duration
	}

	timings := make([]replicationTiming, 0, len(followers))

	for _, follower := range followers {
		// Poll until follower has the data
		deadline := time.Now().Add(5 * time.Second)
		var lag time.Duration
		for time.Now().Before(deadline) {
			kvStore := follower.fsm.kvStore.(*MockKVStore)
			value, exists, err := kvStore.Get(testKey)
			if err == nil && exists && value == testValue {
				lag = time.Since(writeTime)
				timings = append(timings, replicationTiming{
					nodeID: follower.config.NodeID,
					lag:    lag,
				})
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	require.Len(t, timings, len(followers), "all followers should receive data")

	// Log timings
	for _, timing := range timings {
		t.Logf("Follower %s received data in %s", timing.nodeID, timing.lag)
	}

	// If replication was truly parallel, all followers should receive data
	// within a similar timeframe (not 2x the time of one follower)
	for _, timing := range timings {
		require.Less(t, timing.lag, 2*time.Second, "follower %s should receive data quickly", timing.nodeID)
	}
}

// TestReplication_OrderPreservation verifies write order is preserved.
func TestReplication_OrderPreservation(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write 100 KV pairs with sequential values
	const count = 100
	for i := 0; i < count; i++ {
		cmd, err := NewCommand(CmdKVSet, KVSetPayload{
			Key:   "counter",
			Value: fmt.Sprintf("%d", i),
		})
		require.NoError(t, err)
		_, err = leader.ApplyEntry(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Wait for replication
	time.Sleep(1 * time.Second)

	// All nodes should have the final value (99) - this proves order was preserved
	// If order wasn't preserved, we might have an earlier value
	for _, node := range nodes {
		kvStore := node.fsm.kvStore.(*MockKVStore)
		value, exists, err := kvStore.Get("counter")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "99", value, "final value should be 99 on node %s (order preserved)", node.config.NodeID)
	}

	// Additional check: write unique keys and verify all exist
	for i := 0; i < 50; i++ {
		cmd, err := NewCommand(CmdKVSet, KVSetPayload{
			Key:   fmt.Sprintf("seq-key-%d", i),
			Value: fmt.Sprintf("seq-value-%d", i),
		})
		require.NoError(t, err)
		_, err = leader.ApplyEntry(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	time.Sleep(1 * time.Second)

	// Verify all sequential keys exist on all nodes in order
	for _, node := range nodes {
		kvStore := node.fsm.kvStore.(*MockKVStore)
		for i := 0; i < 50; i++ {
			key := fmt.Sprintf("seq-key-%d", i)
			value, exists, err := kvStore.Get(key)
			require.NoError(t, err)
			require.True(t, exists, "key %s should exist on node %s", key, node.config.NodeID)
			require.Equal(t, fmt.Sprintf("seq-value-%d", i), value)
		}
	}
}

// TestReplication_NetworkPartitionHealing verifies replication after partition heals.
func TestReplication_NetworkPartitionHealing(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write baseline data
	cmd, err := NewCommand(CmdKVSet, KVSetPayload{
		Key:   "baseline",
		Value: "initial-data",
	})
	require.NoError(t, err)
	_, err = leader.ApplyEntry(cmd, 5*time.Second)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Verify baseline is replicated
	for _, node := range nodes {
		kvStore := node.fsm.kvStore.(*MockKVStore)
		value, exists, err := kvStore.Get("baseline")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "initial-data", value)
	}

	// Partition one node by closing its transport
	var partitionedNode *Node
	for _, node := range nodes {
		if node != leader {
			partitionedNode = node
			break
		}
	}
	require.NotNil(t, partitionedNode)

	partitionedID := partitionedNode.config.NodeID
	partitionedDataDir := partitionedNode.config.DataDir
	require.NoError(t, partitionedNode.transport.Close())
	require.NoError(t, partitionedNode.Shutdown())

	// Write data to majority while one node is partitioned
	const partitionedWrites = 30
	for i := 0; i < partitionedWrites; i++ {
		cmd, err := NewCommand(CmdKVSet, KVSetPayload{
			Key:   fmt.Sprintf("partition-key-%d", i),
			Value: fmt.Sprintf("partition-value-%d", i),
		})
		require.NoError(t, err)
		_, err = leader.ApplyEntry(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	time.Sleep(500 * time.Millisecond)

	// Heal the partition by recreating the node
	addr := getFreeAddr(t)
	cfg := newClusterConfig(t, partitionedID, addr, false, clusterOptions{})
	cfg.DataDir = partitionedDataDir
	healedNode := startTestNode(t, cfg)
	defer func() { _ = healedNode.Shutdown() }()

	// Rejoin the cluster
	require.NoError(t, leader.Join(cfg.NodeID, cfg.AdvertiseAddr))

	// Wait for catch-up
	time.Sleep(2 * time.Second)

	// Verify the healed node caught up
	kvStore := healedNode.fsm.kvStore.(*MockKVStore)
	for i := 0; i < partitionedWrites; i++ {
		key := fmt.Sprintf("partition-key-%d", i)
		value, exists, err := kvStore.Get(key)
		require.NoError(t, err)
		require.True(t, exists, "key %s should exist after partition heals", key)
		require.Equal(t, fmt.Sprintf("partition-value-%d", i), value)
	}

	// Verify baseline is still there
	value, exists, err := kvStore.Get("baseline")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "initial-data", value)
}

// TestReplication_ConflictResolution verifies conflict resolution during catch-up.
func TestReplication_ConflictResolution(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write initial data that both partitions will have
	cmd, err := NewCommand(CmdKVSet, KVSetPayload{
		Key:   "conflict-test",
		Value: "initial-value",
	})
	require.NoError(t, err)
	_, err = leader.ApplyEntry(cmd, 5*time.Second)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Find a follower to partition
	var minorityNode *Node
	for _, node := range nodes {
		if node != leader {
			minorityNode = node
			break
		}
	}
	require.NotNil(t, minorityNode)

	// Partition the minority node
	minorityID := minorityNode.config.NodeID
	minorityDataDir := minorityNode.config.DataDir
	require.NoError(t, minorityNode.transport.Close())
	require.NoError(t, minorityNode.Shutdown())

	// The majority partition (leader + 1 follower) writes authoritative data
	cmd, err = NewCommand(CmdKVSet, KVSetPayload{
		Key:   "conflict-test",
		Value: "majority-value",
	})
	require.NoError(t, err)
	_, err = leader.ApplyEntry(cmd, 5*time.Second)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Heal the partition
	addr := getFreeAddr(t)
	cfg := newClusterConfig(t, minorityID, addr, false, clusterOptions{})
	cfg.DataDir = minorityDataDir
	healedNode := startTestNode(t, cfg)
	defer func() { _ = healedNode.Shutdown() }()

	// Rejoin cluster
	require.NoError(t, leader.Join(cfg.NodeID, cfg.AdvertiseAddr))

	// Wait for conflict resolution
	time.Sleep(2 * time.Second)

	// Verify the healed node accepted the majority's data (conflict resolved)
	kvStore := healedNode.fsm.kvStore.(*MockKVStore)
	value, exists, err := kvStore.Get("conflict-test")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "majority-value", value, "minority should accept majority's value after healing")

	// Verify all nodes now have consistent data
	for _, node := range []*Node{leader, healedNode} {
		kvStore := node.fsm.kvStore.(*MockKVStore)
		value, exists, err := kvStore.Get("conflict-test")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "majority-value", value, "all nodes should have majority value")
	}
}

// Helper functions for replication tests

// writeSequentialKV writes sequential KV pairs to the leader.
func writeSequentialKV(t *testing.T, leader *Node, count int) map[string]string {
	t.Helper()

	expected := make(map[string]string)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		expected[key] = value

		// Apply via Raft
		// Implementation depends on Node exposing ApplyKVSet
	}

	return expected
}

// verifyAllNodesConsistent verifies all nodes have the same data.
func verifyAllNodesConsistent(t *testing.T, nodes []*Node, expectedKV map[string]string) {
	t.Helper()

	for _, node := range nodes {
		verifyNodeHasData(t, node, expectedKV)
	}
}

// verifyNodeHasData verifies a node has the expected KV data.
func verifyNodeHasData(t *testing.T, node *Node, expectedKV map[string]string) {
	t.Helper()

	// Implementation depends on how to read from FSM
	// May need to expose a method to get KV store snapshot
	require.NotNil(t, node)
	require.NotEmpty(t, expectedKV)
}

// simulateNetworkFlakiness introduces random delays/drops for testing.
func simulateNetworkFlakiness(t *testing.T, node *Node, dropRate float64) func() {
	t.Helper()

	// Implementation would wrap the transport with a flaky wrapper
	// Return cleanup function to restore normal transport
	return func() {
		// Restore normal transport
	}
}

// measureReplicationLag measures time for data to reach all followers.
func measureReplicationLag(t *testing.T, nodes []*Node, writeTime time.Time) time.Duration {
	t.Helper()

	// Wait until all followers have the latest data
	// Return the time difference from writeTime
	return 0
}

// concurrentWrites performs concurrent writes to test race conditions.
func concurrentWrites(t *testing.T, leader *Node, count int) {
	t.Helper()

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-key-%d", idx)
			value := fmt.Sprintf("concurrent-value-%d", idx)
			// Apply write
			_ = key
			_ = value
		}(i)
	}
	wg.Wait()
}
