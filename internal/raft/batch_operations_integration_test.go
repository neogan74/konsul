package raft

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBatchOperations_BatchSetSuccess verifies batch set replicates to all nodes.
func TestBatchOperations_BatchSetSuccess(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	batch := prepareBatchSet(10)
	require.NoError(t, leader.KVBatchSet(batch))

	time.Sleep(400 * time.Millisecond)

	// All nodes must have all 10 keys
	for _, node := range nodes {
		kv := node.fsm.kvStore.(*MockKVStore)
		for k, v := range batch {
			e, ok := kv.GetEntrySnapshot(k)
			assert.True(t, ok, "node %s missing key %s", node.config.NodeID, k)
			assert.Equal(t, v, e.Value, "node %s key %s value mismatch", node.config.NodeID, k)
		}
	}
}

// TestBatchOperations_BatchDeleteSuccess verifies batch delete replicates to all nodes.
func TestBatchOperations_BatchDeleteSuccess(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write 6 keys, then delete 3
	batch := prepareBatchSet(6)
	require.NoError(t, leader.KVBatchSet(batch))
	time.Sleep(300 * time.Millisecond)

	toDelete := make([]string, 0, 3)
	toKeep := make([]string, 0, 3)
	i := 0
	for k := range batch {
		if i < 3 {
			toDelete = append(toDelete, k)
		} else {
			toKeep = append(toKeep, k)
		}
		i++
	}

	require.NoError(t, leader.KVBatchDelete(toDelete))
	time.Sleep(400 * time.Millisecond)

	for _, node := range nodes {
		kv := node.fsm.kvStore.(*MockKVStore)
		for _, k := range toDelete {
			_, ok := kv.GetEntrySnapshot(k)
			assert.False(t, ok, "node %s should not have deleted key %s", node.config.NodeID, k)
		}
		for _, k := range toKeep {
			_, ok := kv.GetEntrySnapshot(k)
			assert.True(t, ok, "node %s should still have key %s", node.config.NodeID, k)
		}
	}
}

// TestBatchOperations_BatchCASSuccess verifies BatchSetCAS returns new indices and replicates.
func TestBatchOperations_BatchCASSuccess(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	// Write 3 initial keys
	require.NoError(t, leader.KVBatchSet(map[string]string{
		"bk1": "v1", "bk2": "v2", "bk3": "v3",
	}))
	time.Sleep(500 * time.Millisecond)

	// Gather expected indices from leader FSM
	kv := leader.fsm.kvStore.(*MockKVStore)
	expectedIndices := make(map[string]uint64)
	for _, k := range []string{"bk1", "bk2", "bk3"} {
		e, ok := kv.GetEntrySnapshot(k)
		require.True(t, ok, "key %s must exist", k)
		expectedIndices[k] = e.ModifyIndex
	}

	// BatchSetCAS with correct indices
	updates := map[string]string{"bk1": "u1", "bk2": "u2", "bk3": "u3"}
	newIndices, err := leader.KVBatchSetCAS(updates, expectedIndices)
	require.NoError(t, err, "BatchSetCAS with correct indices must succeed")
	require.Len(t, newIndices, 3)
	for k, newIdx := range newIndices {
		assert.Greater(t, newIdx, expectedIndices[k], "new index for %s must be greater", k)
	}

	// Poll each node until it sees the updated values.
	// Use a generous timeout to handle follower replication lag in CI.
	for _, node := range nodes {
		nodeKV := node.fsm.kvStore.(*MockKVStore)
		for k, v := range updates {
			waitForKVValue(t, nodeKV, k, v, 15*time.Second)
		}
	}
}

// TestBatchOperations_BatchCASPartialFailure verifies that one bad index fails the whole batch.
func TestBatchOperations_BatchCASPartialFailure(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	require.NoError(t, leader.KVBatchSet(map[string]string{
		"pk1": "v1", "pk2": "v2",
	}))
	time.Sleep(300 * time.Millisecond)

	kv := leader.fsm.kvStore.(*MockKVStore)
	e1, _ := kv.GetEntrySnapshot("pk1")
	// pk2 uses a wrong expected index — triggers conflict for the whole batch
	badIndices := map[string]uint64{"pk1": e1.ModifyIndex, "pk2": 9999}

	_, err := leader.KVBatchSetCAS(
		map[string]string{"pk1": "new1", "pk2": "new2"},
		badIndices,
	)
	require.Error(t, err, "BatchSetCAS with one bad index must fail")

	time.Sleep(300 * time.Millisecond)

	// Neither key should be modified on any node
	for _, node := range nodes {
		nodeKV := node.fsm.kvStore.(*MockKVStore)
		e, ok := nodeKV.GetEntrySnapshot("pk1")
		assert.True(t, ok)
		assert.Equal(t, "v1", e.Value, "node %s: pk1 must be unchanged", node.config.NodeID)
		e2, ok := nodeKV.GetEntrySnapshot("pk2")
		assert.True(t, ok)
		assert.Equal(t, "v2", e2.Value, "node %s: pk2 must be unchanged", node.config.NodeID)
	}
}

// TestBatchOperations_Atomicity verifies a batch is all-or-nothing via Raft log.
func TestBatchOperations_Atomicity(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	batch := prepareBatchSet(5)
	require.NoError(t, leader.KVBatchSet(batch))
	time.Sleep(400 * time.Millisecond)

	// All nodes must have EXACTLY the same keys
	for _, node := range nodes {
		kv := node.fsm.kvStore.(*MockKVStore)
		for k, v := range batch {
			e, ok := kv.GetEntrySnapshot(k)
			assert.True(t, ok, "node %s should have key %s", node.config.NodeID, k)
			assert.Equal(t, v, e.Value)
		}
	}
}

// TestBatchOperations_LargeBatch verifies handling of large batch sizes.
func TestBatchOperations_LargeBatch(t *testing.T) {
	t.Skip("TODO: Implement large batch test")

	// Test plan:
	// 1. Create a 3-node cluster
	// 2. Prepare batch with 10,000 operations
	// 3. Execute batch operation
	// 4. Verify all operations complete successfully
	// 5. Measure performance (ops/sec)
	// 6. Verify no timeout or memory issues
}

// TestBatchOperations_ConcurrentBatches verifies concurrent batch operations.
func TestBatchOperations_ConcurrentBatches(t *testing.T) {
	t.Skip("TODO: Implement concurrent batches test")

	// Test plan:
	// 1. Create a 3-node cluster
	// 2. Submit 10 batch operations concurrently
	// 3. Verify all batches complete successfully
	// 4. Verify no data loss or corruption
	// 5. Verify all nodes eventually consistent
	// 6. Verify batches don't interfere with each other
}

// TestBatchOperations_BatchMixedOperations verifies mixed operation batch.
func TestBatchOperations_BatchMixedOperations(t *testing.T) {
	t.Skip("TODO: Implement mixed operations batch test")

	// Test plan:
	// 1. Create a 3-node cluster
	// 2. Prepare batch with sets, deletes, and CAS ops
	// 3. Execute mixed batch
	// 4. Verify all operations applied correctly
	// 5. Verify atomicity across different op types
	// 6. Verify all nodes consistent
}

// TestBatchOperations_BatchReplication verifies a batch written on leader appears on all followers.
func TestBatchOperations_BatchReplication(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)

	batch := prepareBatchSet(20)
	require.NoError(t, leader.KVBatchSet(batch))
	time.Sleep(500 * time.Millisecond)

	followers := make([]*Node, 0, 2)
	for _, n := range nodes {
		if n != leader {
			followers = append(followers, n)
		}
	}
	require.Len(t, followers, 2, "should have exactly 2 followers")

	for _, follower := range followers {
		kv := follower.fsm.kvStore.(*MockKVStore)
		for k, v := range batch {
			e, ok := kv.GetEntrySnapshot(k)
			assert.True(t, ok, "follower %s missing replicated key %s", follower.config.NodeID, k)
			assert.Equal(t, v, e.Value, "follower %s value mismatch for key %s", follower.config.NodeID, k)
		}
	}
}

// TestBatchOperations_BatchWithLeaderChange verifies batch during leader change.
func TestBatchOperations_BatchWithLeaderChange(t *testing.T) {
	t.Skip("TODO: Implement batch with leader change test")

	// Test plan:
	// 1. Create a 3-node cluster
	// 2. Start batch operation on leader
	// 3. Trigger leader election mid-batch
	// 4. Verify batch either completes on old leader or fails cleanly
	// 5. Verify no partial application
	// 6. Retry batch on new leader
	// 7. Verify retry succeeds
}

// TestBatchOperations_BatchRetry verifies batch retry logic.
func TestBatchOperations_BatchRetry(t *testing.T) {
	t.Skip("TODO: Implement batch retry test")

	// Test plan:
	// 1. Create a 3-node cluster
	// 2. Execute batch that fails (e.g., non-leader)
	// 3. Implement retry with exponential backoff
	// 4. Verify batch eventually succeeds
	// 5. Verify idempotency (retry doesn't duplicate)
}

// Helper functions for batch operations tests

// prepareBatchSet prepares a batch set operation.
func prepareBatchSet(count int) map[string]string {
	batch := make(map[string]string, count)
	for i := 0; i < count; i++ {
		batch[fmt.Sprintf("batch-key-%d", i)] = fmt.Sprintf("batch-value-%d", i)
	}
	return batch
}
