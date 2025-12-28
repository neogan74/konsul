package raft

import (
	"fmt"
	"time"
)

// Config contains configuration for the Raft node.
type Config struct {
	// NodeID is the unique identifier for this node in the cluster.
	// Must be unique across all nodes.
	NodeID string

	// BindAddr is the address to bind for Raft communication (e.g., "0.0.0.0:7000").
	// This should be reachable by other nodes in the cluster.
	BindAddr string

	// AdvertiseAddr is the address advertised to other nodes (e.g., "192.168.1.10:7000").
	// If empty, BindAddr is used. Use this when running behind NAT.
	AdvertiseAddr string

	// DataDir is the directory for storing Raft data (logs, snapshots, stable store).
	DataDir string

	// Bootstrap indicates whether this node should bootstrap a new cluster.
	// Only set to true for the FIRST node in a new cluster.
	Bootstrap bool

	// HeartbeatTimeout is the time between heartbeats from the leader.
	// Default: 1s
	HeartbeatTimeout time.Duration

	// ElectionTimeout is the timeout for starting a new election.
	// Should be larger than HeartbeatTimeout.
	// Default: 1s
	ElectionTimeout time.Duration

	// LeaderLeaseTimeout is how long a leader will hold leadership without
	// being able to contact a quorum.
	// Default: 500ms
	LeaderLeaseTimeout time.Duration

	// CommitTimeout is the timeout for committing a log entry.
	// Default: 50ms
	CommitTimeout time.Duration

	// SnapshotInterval is how often to check if a snapshot should be taken.
	// Default: 120s
	SnapshotInterval time.Duration

	// SnapshotThreshold is the number of log entries after which a snapshot is taken.
	// Default: 8192
	SnapshotThreshold uint64

	// SnapshotRetention is the number of snapshots to retain.
	// Default: 2
	SnapshotRetention int

	// MaxAppendEntries is the maximum number of log entries to send in a single AppendEntries RPC.
	// Default: 64
	MaxAppendEntries int

	// TrailingLogs is the number of logs to leave after a snapshot.
	// Default: 10240
	TrailingLogs uint64

	// LogLevel sets the logging level for Raft (debug, info, warn, error).
	// Default: info
	LogLevel string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		HeartbeatTimeout:   1000 * time.Millisecond,
		ElectionTimeout:    1000 * time.Millisecond,
		LeaderLeaseTimeout: 500 * time.Millisecond,
		CommitTimeout:      50 * time.Millisecond,
		SnapshotInterval:   120 * time.Second,
		SnapshotThreshold:  8192,
		SnapshotRetention:  2,
		MaxAppendEntries:   64,
		TrailingLogs:       10240,
		LogLevel:           "info",
	}
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.NodeID == "" {
		return fmt.Errorf("NodeID is required")
	}
	if c.BindAddr == "" {
		return fmt.Errorf("BindAddr is required")
	}
	if c.DataDir == "" {
		return fmt.Errorf("DataDir is required")
	}
	if c.HeartbeatTimeout <= 0 {
		return fmt.Errorf("HeartbeatTimeout must be positive")
	}
	if c.ElectionTimeout <= 0 {
		return fmt.Errorf("ElectionTimeout must be positive")
	}
	if c.ElectionTimeout < c.HeartbeatTimeout {
		return fmt.Errorf("ElectionTimeout should be >= HeartbeatTimeout")
	}
	if c.SnapshotThreshold == 0 {
		return fmt.Errorf("SnapshotThreshold must be positive")
	}
	return nil
}

// GetAdvertiseAddr returns the advertise address, defaulting to BindAddr if not set.
func (c *Config) GetAdvertiseAddr() string {
	if c.AdvertiseAddr != "" {
		return c.AdvertiseAddr
	}
	return c.BindAddr
}
