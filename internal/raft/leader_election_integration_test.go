package raft

import (
	"math"
	"net"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"
)

type clusterOptions struct {
	heartbeat   time.Duration
	election    time.Duration
	leaderLease time.Duration
}

func (o clusterOptions) withDefaults() clusterOptions {
	if o.heartbeat == 0 {
		o.heartbeat = 250 * time.Millisecond
	}
	if o.election == 0 {
		o.election = 500 * time.Millisecond
	}
	if o.leaderLease == 0 {
		o.leaderLease = 200 * time.Millisecond
	}
	return o
}

func getFreeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	return ln.Addr().String()
}

func newClusterConfig(t *testing.T, nodeID, addr string, bootstrap bool, opts clusterOptions) *Config {
	t.Helper()

	opts = opts.withDefaults()
	resetPrometheusRegistry()

	cfg := DefaultConfig()
	cfg.NodeID = nodeID
	cfg.BindAddr = addr
	cfg.AdvertiseAddr = addr
	cfg.DataDir = t.TempDir()
	cfg.Bootstrap = bootstrap
	cfg.HeartbeatTimeout = opts.heartbeat
	cfg.ElectionTimeout = opts.election
	cfg.LeaderLeaseTimeout = opts.leaderLease
	cfg.CommitTimeout = 20 * time.Millisecond
	cfg.SnapshotInterval = 2 * time.Second
	cfg.SnapshotThreshold = 128
	cfg.SnapshotRetention = 1
	cfg.MaxAppendEntries = 64
	cfg.TrailingLogs = 64
	return cfg
}

func startTestNode(t *testing.T, cfg *Config) *Node {
	t.Helper()

	kvStore := NewMockKVStore()
	serviceStore := NewMockServiceStore()
	node, err := NewNode(cfg, kvStore, serviceStore)
	require.NoError(t, err)
	return node
}

func waitForSingleLeader(t *testing.T, nodes []*Node, timeout time.Duration) *Node {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var leader *Node
		leaders := 0
		for _, node := range nodes {
			if node.IsLeader() {
				leader = node
				leaders++
			}
		}
		if leaders == 1 {
			return leader
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for single leader")
	return nil
}

func waitForConfigSize(t *testing.T, node *Node, expected int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cfg, err := node.GetConfiguration()
		if err == nil && len(cfg.Servers) == expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for config size %d", expected)
}

func newThreeNodeCluster(t *testing.T, opts clusterOptions) ([]*Node, func()) {
	t.Helper()

	addr1 := getFreeAddr(t)
	cfg1 := newClusterConfig(t, "node-1", addr1, true, opts)
	node1 := startTestNode(t, cfg1)
	require.NoError(t, node1.WaitForLeader(5*time.Second))

	addr2 := getFreeAddr(t)
	cfg2 := newClusterConfig(t, "node-2", addr2, false, opts)
	node2 := startTestNode(t, cfg2)

	addr3 := getFreeAddr(t)
	cfg3 := newClusterConfig(t, "node-3", addr3, false, opts)
	node3 := startTestNode(t, cfg3)

	require.NoError(t, node1.Join(cfg2.NodeID, cfg2.AdvertiseAddr))
	require.NoError(t, node1.Join(cfg3.NodeID, cfg3.AdvertiseAddr))

	nodes := []*Node{node1, node2, node3}
	waitForSingleLeader(t, nodes, 5*time.Second)

	cleanup := func() {
		for _, node := range nodes {
			_ = node.Shutdown()
		}
	}

	return nodes, cleanup
}

func TestLeaderElection_ThreeNodeCluster(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)
	for _, node := range nodes {
		if node == leader {
			require.True(t, node.IsLeader())
		} else {
			require.False(t, node.IsLeader())
		}
	}
}

func TestClusterJoinLeave(t *testing.T) {
	opts := clusterOptions{}
	addr1 := getFreeAddr(t)
	cfg1 := newClusterConfig(t, "node-1", addr1, true, opts)
	node1 := startTestNode(t, cfg1)
	require.NoError(t, node1.WaitForLeader(5*time.Second))

	addr2 := getFreeAddr(t)
	cfg2 := newClusterConfig(t, "node-2", addr2, false, opts)
	node2 := startTestNode(t, cfg2)

	addr3 := getFreeAddr(t)
	cfg3 := newClusterConfig(t, "node-3", addr3, false, opts)
	node3 := startTestNode(t, cfg3)

	defer func() {
		_ = node1.Shutdown()
		_ = node2.Shutdown()
		_ = node3.Shutdown()
	}()

	require.NoError(t, node1.Join(cfg2.NodeID, cfg2.AdvertiseAddr))
	waitForConfigSize(t, node1, 2, 5*time.Second)

	require.NoError(t, node1.Join(cfg3.NodeID, cfg3.AdvertiseAddr))
	waitForConfigSize(t, node1, 3, 5*time.Second)

	require.NoError(t, node1.Leave(cfg2.NodeID))
	waitForConfigSize(t, node1, 2, 5*time.Second)

	cfg, err := node1.GetConfiguration()
	require.NoError(t, err)
	for _, srv := range cfg.Servers {
		require.NotEqual(t, raft.ServerID(cfg2.NodeID), srv.ID)
	}
}

func TestClusterJoinNonLeader(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)
	var follower *Node
	for _, node := range nodes {
		if node != leader {
			follower = node
			break
		}
	}
	require.NotNil(t, follower)

	err := follower.Join("node-x", getFreeAddr(t))
	require.ErrorIs(t, err, ErrNotLeader)
}

func TestLinearizableRead_LeaderOnly(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)
	require.NoError(t, leader.EnsureLinearizableRead(2*time.Second))

	var follower *Node
	for _, node := range nodes {
		if node != leader {
			follower = node
			break
		}
	}
	require.NotNil(t, follower)
	require.ErrorIs(t, follower.EnsureLinearizableRead(2*time.Second), ErrNotLeader)
}

func TestLeaderElection_LeaderFailureReelection(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)
	require.NoError(t, leader.Shutdown())

	var remaining []*Node
	for _, node := range nodes {
		if node != leader {
			remaining = append(remaining, node)
		}
	}

	newLeader := waitForSingleLeader(t, remaining, 5*time.Second)
	require.NotNil(t, newLeader)
	require.NotEqual(t, leader, newLeader)
}

func TestLeaderElection_PartitionMinorityNoLeader(t *testing.T) {
	nodes, cleanup := newThreeNodeCluster(t, clusterOptions{})
	defer cleanup()

	leader := waitForSingleLeader(t, nodes, 5*time.Second)
	var isolated *Node
	for _, node := range nodes {
		if node != leader {
			isolated = node
			break
		}
	}
	require.NotNil(t, isolated)

	// Close the transport to simulate a network partition for the isolated node.
	require.NoError(t, isolated.transport.Close())
	time.Sleep(isolated.config.ElectionTimeout * 3)

	require.NotEqual(t, raft.Leader, isolated.State())

	var majority []*Node
	for _, node := range nodes {
		if node != isolated {
			majority = append(majority, node)
		}
	}
	majorityLeader := waitForSingleLeader(t, majority, 5*time.Second)
	require.NotNil(t, majorityLeader)
}

func TestLeaderElection_PerfP99(t *testing.T) {
	if os.Getenv("KONSUL_PERF_TEST") != "1" {
		t.Skip("set KONSUL_PERF_TEST=1 to enable leader election perf test")
	}

	const iterations = 10
	durations := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		opts := clusterOptions{
			heartbeat:   50 * time.Millisecond,
			election:    100 * time.Millisecond,
			leaderLease: 50 * time.Millisecond,
		}
		nodes, cleanup := newThreeNodeCluster(t, opts)

		leader := waitForSingleLeader(t, nodes, 5*time.Second)

		var remaining []*Node
		for _, node := range nodes {
			if node != leader {
				remaining = append(remaining, node)
			}
		}

		start := time.Now()
		require.NoError(t, leader.Shutdown())
		waitForSingleLeader(t, remaining, 5*time.Second)
		durations = append(durations, time.Since(start))

		cleanup()
	}

	p99 := percentileDuration(durations, 0.99)
	if p99 > 300*time.Millisecond {
		t.Fatalf("leader election p99 too slow: %s", p99)
	}
}

func percentileDuration(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	idx := int(math.Ceil(p*float64(len(values)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}
