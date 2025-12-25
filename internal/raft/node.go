package raft

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	hashiraft "github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/neogan74/konsul/internal/config"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

type Node struct {
	raft *hashiraft.Raft
	log  logger.Logger
}

func NewNode(cfg config.RaftConfig, kvStore *store.KVStore, serviceStore *store.ServiceStore, log logger.Logger) (*Node, error) {
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create raft data dir: %w", err)
	}

	raftConfig := hashiraft.DefaultConfig()
	raftConfig.LocalID = hashiraft.ServerID(cfg.NodeID)
	raftConfig.ElectionTimeout = cfg.ElectionTimeout
	raftConfig.HeartbeatTimeout = cfg.HeartbeatTimeout
	raftConfig.LeaderLeaseTimeout = cfg.LeaderLeaseTimeout
	raftConfig.SnapshotInterval = cfg.SnapshotInterval
	raftConfig.SnapshotThreshold = cfg.SnapshotThreshold

	fsm := NewFSM(kvStore, serviceStore, log)

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.DataDir, "raft-log.bolt"))
	if err != nil {
		return nil, fmt.Errorf("failed to create raft log store: %w", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.DataDir, "raft-stable.bolt"))
	if err != nil {
		return nil, fmt.Errorf("failed to create raft stable store: %w", err)
	}

	snapshotDir := filepath.Join(cfg.DataDir, "snapshots")
	snapshotStore, err := hashiraft.NewFileSnapshotStore(snapshotDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft snapshot store: %w", err)
	}

	transport, err := hashiraft.NewTCPTransport(cfg.BindAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft transport: %w", err)
	}

	r, err := hashiraft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft node: %w", err)
	}

	if cfg.Bootstrap {
		hasState, err := hashiraft.HasExistingState(logStore, stableStore, snapshotStore)
		if err != nil {
			return nil, fmt.Errorf("failed to check raft state: %w", err)
		}
		if !hasState {
			if err := bootstrapCluster(r, cfg); err != nil {
				return nil, err
			}
		}
	}

	return &Node{raft: r, log: log}, nil
}

func (n *Node) IsLeader() bool {
	return n.raft.State() == hashiraft.Leader
}

func (n *Node) Leader() string {
	return string(n.raft.Leader())
}

func (n *Node) ApplyEntry(entry LogEntry, timeout time.Duration) (interface{}, error) {
	data, err := entry.Marshal()
	if err != nil {
		return nil, err
	}

	future := n.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		return nil, err
	}

	resp := future.Response()
	if respErr, ok := resp.(error); ok {
		return nil, respErr
	}
	return resp, nil
}

func (n *Node) Shutdown() error {
	future := n.raft.Shutdown()
	return future.Error()
}

func bootstrapCluster(r *hashiraft.Raft, cfg config.RaftConfig) error {
	servers := make([]hashiraft.Server, 0, len(cfg.Peers)+1)
	seen := map[string]bool{}

	localAddr := cfg.BindAddr
	if cfg.AdvertiseAddr != "" {
		localAddr = cfg.AdvertiseAddr
	}

	addServer := func(id, addr string) {
		if id == "" || addr == "" || seen[id] {
			return
		}
		servers = append(servers, hashiraft.Server{
			ID:      hashiraft.ServerID(id),
			Address: hashiraft.ServerAddress(addr),
		})
		seen[id] = true
	}

	addServer(cfg.NodeID, localAddr)
	for _, peer := range cfg.Peers {
		addServer(peer.ID, peer.Address)
	}

	if len(servers) == 0 {
		return errors.New("raft bootstrap requires at least one server")
	}

	future := r.BootstrapCluster(hashiraft.Configuration{Servers: servers})
	if err := future.Error(); err != nil {
		if errors.Is(err, hashiraft.ErrCantBootstrap) {
			return nil
		}
		return err
	}
	return nil
}
