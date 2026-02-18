package raft

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/neogan74/konsul/internal/store"
)

// Node represents a Raft node in the cluster.
type Node struct {
	config    *Config
	raft      *raft.Raft
	fsm       *KonsulFSM
	transport *raft.NetworkTransport
	logStore  *raftboltdb.BoltStore
	stable    *raftboltdb.BoltStore
	snapshots *raft.FileSnapshotStore

	logger  hclog.Logger
	metrics *Metrics

	shutdownCh chan struct{}
	mu         sync.RWMutex
}

// NewNode creates a new Raft node with the given configuration.
func NewNode(cfg *Config, kvStore KVStoreInterface, serviceStore ServiceStoreInterface) (*Node, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create logger
	logLevel := hclog.LevelFromString(cfg.LogLevel)
	if logLevel == hclog.NoLevel {
		logLevel = hclog.Info
	}
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "raft",
		Level:  logLevel,
		Output: os.Stderr,
	})

	// Create FSM
	fsm := NewFSM(FSMConfig{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
	})

	// Create Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(cfg.NodeID)
	raftConfig.HeartbeatTimeout = cfg.HeartbeatTimeout
	raftConfig.ElectionTimeout = cfg.ElectionTimeout
	raftConfig.LeaderLeaseTimeout = cfg.LeaderLeaseTimeout
	raftConfig.CommitTimeout = cfg.CommitTimeout
	raftConfig.SnapshotInterval = cfg.SnapshotInterval
	raftConfig.SnapshotThreshold = cfg.SnapshotThreshold
	raftConfig.TrailingLogs = cfg.TrailingLogs
	raftConfig.MaxAppendEntries = cfg.MaxAppendEntries
	raftConfig.Logger = logger

	// Create TCP transport
	advertiseAddr := cfg.GetAdvertiseAddr()
	addr, err := net.ResolveTCPAddr("tcp", advertiseAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve advertise address: %w", err)
	}

	transport, err := raft.NewTCPTransport(cfg.BindAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	// Create BoltDB store for logs
	logStorePath := filepath.Join(cfg.DataDir, "raft-log.db")
	logStore, err := raftboltdb.NewBoltStore(logStorePath)
	if err != nil {
		if closeErr := transport.Close(); closeErr != nil {
			logger.Warn("failed to close transport", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create log store: %w", err)
	}

	// Create BoltDB store for stable storage (term, vote, etc.)
	stablePath := filepath.Join(cfg.DataDir, "raft-stable.db")
	stable, err := raftboltdb.NewBoltStore(stablePath)
	if err != nil {
		if closeErr := logStore.Close(); closeErr != nil {
			logger.Warn("failed to close log store", "error", closeErr)
		}
		if closeErr := transport.Close(); closeErr != nil {
			logger.Warn("failed to close transport", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create stable store: %w", err)
	}

	// Create snapshot store
	snapshots, err := raft.NewFileSnapshotStore(cfg.DataDir, cfg.SnapshotRetention, os.Stderr)
	if err != nil {
		if closeErr := stable.Close(); closeErr != nil {
			logger.Warn("failed to close stable store", "error", closeErr)
		}
		if closeErr := logStore.Close(); closeErr != nil {
			logger.Warn("failed to close log store", "error", closeErr)
		}
		if closeErr := transport.Close(); closeErr != nil {
			logger.Warn("failed to close transport", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create snapshot store: %w", err)
	}

	// Create Raft instance
	r, err := raft.NewRaft(raftConfig, fsm, logStore, stable, snapshots, transport)
	if err != nil {
		if closeErr := stable.Close(); closeErr != nil {
			logger.Warn("failed to close stable store", "error", closeErr)
		}
		if closeErr := logStore.Close(); closeErr != nil {
			logger.Warn("failed to close log store", "error", closeErr)
		}
		if closeErr := transport.Close(); closeErr != nil {
			logger.Warn("failed to close transport", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create raft: %w", err)
	}

	// Create metrics
	metrics := NewMetrics("konsul")

	node := &Node{
		config:     cfg,
		raft:       r,
		fsm:        fsm,
		transport:  transport,
		logStore:   logStore,
		stable:     stable,
		snapshots:  snapshots,
		logger:     logger,
		metrics:    metrics,
		shutdownCh: make(chan struct{}),
	}

	// Start metrics monitoring goroutine
	go node.monitorState()

	// Bootstrap if this is the first node
	if cfg.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(cfg.NodeID),
					Address: raft.ServerAddress(advertiseAddr),
				},
			},
		}

		future := r.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			// It's okay if already bootstrapped
			if err != raft.ErrCantBootstrap {
				logger.Warn("bootstrap failed (may already be bootstrapped)", "error", err)
			}
		} else {
			logger.Info("cluster bootstrapped successfully")
		}
	}

	logger.Info("raft node created",
		"node_id", cfg.NodeID,
		"bind_addr", cfg.BindAddr,
		"advertise_addr", advertiseAddr,
		"bootstrap", cfg.Bootstrap,
	)

	return node, nil
}

// IsLeader returns true if this node is the current leader.
func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}

// LeaderAddr returns the address of the current leader.
// Returns empty string if there is no leader.
func (n *Node) LeaderAddr() string {
	addr, _ := n.raft.LeaderWithID()
	return string(addr)
}

// LeaderID returns the ID of the current leader.
// Returns empty string if there is no leader.
func (n *Node) LeaderID() string {
	_, id := n.raft.LeaderWithID()
	return string(id)
}

// Leader returns the address of the current leader (alias for LeaderAddr).
// This is provided for compatibility with handler code.
func (n *Node) Leader() string {
	return n.LeaderAddr()
}

// State returns the current Raft state (follower, candidate, leader, shutdown).
func (n *Node) State() raft.RaftState {
	return n.raft.State()
}

// EnsureLinearizableRead blocks until this leader has applied all prior writes.
// Call this before serving linearizable reads from the local state machine.
func (n *Node) EnsureLinearizableRead(timeout time.Duration) error {
	if n.raft.State() != raft.Leader {
		return ErrNotLeader
	}

	if err := n.raft.VerifyLeader().Error(); err != nil {
		if errors.Is(err, raft.ErrNotLeader) {
			return ErrNotLeader
		}
		return fmt.Errorf("raft verify leader failed: %w", err)
	}

	if err := n.raft.Barrier(timeout).Error(); err != nil {
		if errors.Is(err, raft.ErrNotLeader) || errors.Is(err, raft.ErrLeadershipLost) {
			return ErrNotLeader
		}
		if errors.Is(err, raft.ErrRaftShutdown) {
			return ErrShutdown
		}
		return fmt.Errorf("raft barrier failed: %w", err)
	}

	return nil
}

// Stats returns Raft statistics.
func (n *Node) Stats() map[string]string {
	return n.raft.Stats()
}

// LastIndex returns the last log index.
func (n *Node) LastIndex() uint64 {
	return n.raft.LastIndex()
}

// AppliedIndex returns the last applied index.
func (n *Node) AppliedIndex() uint64 {
	return n.raft.AppliedIndex()
}

// GetConfiguration returns the current cluster configuration.
func (n *Node) GetConfiguration() (raft.Configuration, error) {
	future := n.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return raft.Configuration{}, err
	}
	return future.Configuration(), nil
}

// Shutdown gracefully shuts down the Raft node.
func (n *Node) Shutdown() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.logger.Info("shutting down raft node")

	// Close shutdown channel
	select {
	case <-n.shutdownCh:
		// Already closed
	default:
		close(n.shutdownCh)
	}

	// Shutdown Raft
	if err := n.raft.Shutdown().Error(); err != nil {
		n.logger.Error("failed to shutdown raft", "error", err)
	}

	// Close stores
	if err := n.logStore.Close(); err != nil {
		n.logger.Error("failed to close log store", "error", err)
	}

	if err := n.stable.Close(); err != nil {
		n.logger.Error("failed to close stable store", "error", err)
	}

	// Close transport
	if err := n.transport.Close(); err != nil {
		n.logger.Error("failed to close transport", "error", err)
	}

	n.logger.Info("raft node shutdown complete")
	return nil
}

// WaitForLeader blocks until a leader is elected or the timeout expires.
func (n *Node) WaitForLeader(timeout time.Duration) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if leader := n.LeaderAddr(); leader != "" {
				return nil
			}
		case <-timer.C:
			return ErrNoLeader
		case <-n.shutdownCh:
			return ErrShutdown
		}
	}
}

// =============================================================================
// Apply Methods - These send commands through Raft
// =============================================================================

// ApplyEntry applies a Command through Raft consensus.
// This method bridges the handler calls with the Raft apply logic.
// Returns the response from FSM and any error.
func (n *Node) ApplyEntry(cmd *Command, timeout time.Duration) (interface{}, error) {
	if n.raft.State() != raft.Leader {
		return nil, ErrNotLeader
	}

	data, err := cmd.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	future := n.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		if err == raft.ErrLeadershipLost {
			return nil, ErrNotLeader
		}
		return nil, fmt.Errorf("raft apply failed: %w", err)
	}

	// Return the response from FSM.Apply
	return future.Response(), nil
}

func (n *Node) applyCommand(cmd *Command, timeout time.Duration) error {
	_, err := n.ApplyEntry(cmd, timeout)
	return err
}

// KVSet sets a key-value pair through Raft consensus.
func (n *Node) KVSet(key, value string) error {
	cmd, err := NewCommand(CmdKVSet, KVSetPayload{Key: key, Value: value})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 5*time.Second)
}

// KVSetWithFlags sets a key-value pair with flags through Raft consensus.
func (n *Node) KVSetWithFlags(key, value string, flags uint64) error {
	cmd, err := NewCommand(CmdKVSetWithFlags, KVSetWithFlagsPayload{Key: key, Value: value, Flags: flags})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 5*time.Second)
}

// KVDelete deletes a key through Raft consensus.
func (n *Node) KVDelete(key string) error {
	cmd, err := NewCommand(CmdKVDelete, KVDeletePayload{Key: key})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 5*time.Second)
}

// KVBatchSet sets multiple key-value pairs through Raft consensus.
func (n *Node) KVBatchSet(items map[string]string) error {
	cmd, err := NewCommand(CmdKVBatchSet, KVBatchSetPayload{Items: items})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 10*time.Second)
}

// KVBatchDelete deletes multiple keys through Raft consensus.
func (n *Node) KVBatchDelete(keys []string) error {
	cmd, err := NewCommand(CmdKVBatchDelete, KVBatchDeletePayload{Keys: keys})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 10*time.Second)
}

// ServiceRegister registers a service through Raft consensus.
func (n *Node) ServiceRegister(name, address string, port int, tags []string, meta map[string]string) error {
	cmd, err := NewCommand(CmdServiceRegister, ServiceRegisterPayload{
		Service: store.Service{
			Name:    name,
			Address: address,
			Port:    port,
			Tags:    tags,
			Meta:    meta,
		},
	})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 5*time.Second)
}

// ServiceDeregister deregisters a service through Raft consensus.
func (n *Node) ServiceDeregister(name string) error {
	cmd, err := NewCommand(CmdServiceDeregister, ServiceDeregisterPayload{Name: name})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 5*time.Second)
}

// ServiceHeartbeat updates service TTL through Raft consensus.
func (n *Node) ServiceHeartbeat(name string) error {
	cmd, err := NewCommand(CmdServiceHeartbeat, ServiceHeartbeatPayload{Name: name})
	if err != nil {
		return err
	}
	return n.applyCommand(cmd, 5*time.Second)
}

// =============================================================================
// Cluster Management
// =============================================================================

// Join adds a new node to the cluster.
// Must be called on the leader.
func (n *Node) Join(nodeID, addr string) error {
	if n.raft.State() != raft.Leader {
		return ErrNotLeader
	}

	n.logger.Info("joining node to cluster", "node_id", nodeID, "addr", addr)

	// Check if node is already in the cluster
	configFuture := n.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) {
			if srv.Address == raft.ServerAddress(addr) {
				n.logger.Info("node already member of cluster", "node_id", nodeID)
				return nil
			}
			// Node exists but with different address - remove first
			removeFuture := n.raft.RemoveServer(srv.ID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				return fmt.Errorf("failed to remove existing node: %w", err)
			}
		}
	}

	// Add the new node as a voter
	future := n.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to add voter: %w", err)
	}

	n.logger.Info("node joined cluster successfully", "node_id", nodeID, "addr", addr)
	return nil
}

// Leave removes a node from the cluster.
// Must be called on the leader.
func (n *Node) Leave(nodeID string) error {
	if n.raft.State() != raft.Leader {
		return ErrNotLeader
	}

	n.logger.Info("removing node from cluster", "node_id", nodeID)

	future := n.raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	n.logger.Info("node removed from cluster", "node_id", nodeID)
	return nil
}

// Snapshot triggers a manual snapshot.
func (n *Node) Snapshot() error {
	future := n.raft.Snapshot()
	return future.Error()
}

// =============================================================================
// Cluster Info
// =============================================================================

// ClusterInfo contains information about the cluster.
type ClusterInfo struct {
	NodeID      string     `json:"node_id"`
	State       string     `json:"state"`
	LeaderID    string     `json:"leader_id"`
	LeaderAddr  string     `json:"leader_addr"`
	Peers       []PeerInfo `json:"peers"`
	LastIndex   uint64     `json:"last_index"`
	AppliedIdx  uint64     `json:"applied_index"`
	CommitIndex uint64     `json:"commit_index"`
	Stats       RaftStats  `json:"stats"`
}

// PeerInfo contains information about a peer node.
type PeerInfo struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	State   string `json:"state"` // Voter, Nonvoter, Staging
}

// RaftStats contains Raft statistics.
type RaftStats struct {
	Term               uint64 `json:"term"`
	LastLogIndex       uint64 `json:"last_log_index"`
	LastLogTerm        uint64 `json:"last_log_term"`
	CommitIndex        uint64 `json:"commit_index"`
	AppliedIndex       uint64 `json:"applied_index"`
	FSMPending         uint64 `json:"fsm_pending"`
	LastSnapshotIndex  uint64 `json:"last_snapshot_index"`
	LastSnapshotTerm   uint64 `json:"last_snapshot_term"`
	NumPeers           int    `json:"num_peers"`
	ProtocolVersion    int    `json:"protocol_version"`
	SnapshotVersionMax int    `json:"snapshot_version_max"`
}

// GetClusterInfo returns comprehensive cluster information.
func (n *Node) GetClusterInfo() (*ClusterInfo, error) {
	// Get configuration
	configFuture := n.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	// Build peer list
	var peers []PeerInfo
	for _, srv := range configFuture.Configuration().Servers {
		state := "Voter"
		switch srv.Suffrage {
		case raft.Nonvoter:
			state = "Nonvoter"
		case raft.Staging:
			state = "Staging"
		}
		peers = append(peers, PeerInfo{
			ID:      string(srv.ID),
			Address: string(srv.Address),
			State:   state,
		})
	}

	// Get stats
	stats := n.raft.Stats()
	raftStats := RaftStats{
		NumPeers: len(peers),
	}
	// Parse stats (they're all strings)
	_, _ = fmt.Sscanf(stats["term"], "%d", &raftStats.Term)
	_, _ = fmt.Sscanf(stats["last_log_index"], "%d", &raftStats.LastLogIndex)
	_, _ = fmt.Sscanf(stats["last_log_term"], "%d", &raftStats.LastLogTerm)
	_, _ = fmt.Sscanf(stats["commit_index"], "%d", &raftStats.CommitIndex)
	_, _ = fmt.Sscanf(stats["applied_index"], "%d", &raftStats.AppliedIndex)
	_, _ = fmt.Sscanf(stats["fsm_pending"], "%d", &raftStats.FSMPending)
	_, _ = fmt.Sscanf(stats["last_snapshot_index"], "%d", &raftStats.LastSnapshotIndex)
	_, _ = fmt.Sscanf(stats["last_snapshot_term"], "%d", &raftStats.LastSnapshotTerm)
	_, _ = fmt.Sscanf(stats["protocol_version"], "%d", &raftStats.ProtocolVersion)
	_, _ = fmt.Sscanf(stats["snapshot_version_max"], "%d", &raftStats.SnapshotVersionMax)

	return &ClusterInfo{
		NodeID:      n.config.NodeID,
		State:       n.raft.State().String(),
		LeaderID:    n.LeaderID(),
		LeaderAddr:  n.LeaderAddr(),
		Peers:       peers,
		LastIndex:   n.raft.LastIndex(),
		AppliedIdx:  n.raft.AppliedIndex(),
		CommitIndex: raftStats.CommitIndex,
		Stats:       raftStats,
	}, nil
}

// GetClusterInfoJSON returns cluster info as JSON bytes.
func (n *Node) GetClusterInfoJSON() ([]byte, error) {
	info, err := n.GetClusterInfo()
	if err != nil {
		return nil, err
	}
	return json.Marshal(info)
}

// =============================================================================
// Metrics Monitoring
// =============================================================================

// monitorState monitors Raft state changes and updates metrics.
// This runs in a background goroutine for the lifetime of the node.
func (n *Node) monitorState() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastState := n.raft.State()
	n.metrics.SetState(n.config.NodeID, lastState)

	n.logger.Debug("starting state monitor goroutine")

	for {
		select {
		case <-ticker.C:
			// Get current state
			currentState := n.raft.State()

			// Update state metric
			n.metrics.SetState(n.config.NodeID, currentState)

			// Detect leader changes
			if lastState != currentState {
				n.logger.Info("raft state changed",
					"node_id", n.config.NodeID,
					"old_state", lastState.String(),
					"new_state", currentState.String(),
				)

				// If we transitioned to/from leader, increment leader changes
				if lastState == raft.Leader || currentState == raft.Leader {
					n.metrics.IncLeaderChanges()
					n.logger.Info("leader change detected",
						"node_id", n.config.NodeID,
						"is_leader", currentState == raft.Leader,
					)
				}

				lastState = currentState
			}

			// Update indices metrics
			lastIndex := n.raft.LastIndex()
			appliedIndex := n.raft.AppliedIndex()

			// Get commit index from stats
			stats := n.raft.Stats()
			var commitIndex uint64
			_, _ = fmt.Sscanf(stats["commit_index"], "%d", &commitIndex)

			n.metrics.SetIndices(lastIndex, commitIndex, appliedIndex)

			// Update peer count
			config, err := n.GetConfiguration()
			if err == nil {
				n.metrics.SetNumPeers(len(config.Servers))
			}

			// Update FSM pending
			var fsmPending uint64
			_, _ = fmt.Sscanf(stats["fsm_pending"], "%d", &fsmPending)
			n.metrics.SetFSMPending(fsmPending)

		case <-n.shutdownCh:
			n.logger.Debug("stopping state monitor goroutine")
			return
		}
	}
}
