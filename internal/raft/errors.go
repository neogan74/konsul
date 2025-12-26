package raft

import "errors"

var (
	// ErrNotLeader is returned when a write operation is attempted on a non-leader node.
	ErrNotLeader = errors.New("not the leader")

	// ErrNoLeader is returned when there is no elected leader in the cluster.
	ErrNoLeader = errors.New("no leader elected")

	// ErrNodeNotFound is returned when attempting to remove a node that doesn't exist.
	ErrNodeNotFound = errors.New("node not found in cluster")

	// ErrAlreadyMember is returned when attempting to add a node that's already in the cluster.
	ErrAlreadyMember = errors.New("node is already a cluster member")

	// ErrNotBootstrapped is returned when cluster operations are attempted before bootstrap.
	ErrNotBootstrapped = errors.New("cluster not bootstrapped")

	// ErrAlreadyBootstrapped is returned when attempting to bootstrap an already bootstrapped cluster.
	ErrAlreadyBootstrapped = errors.New("cluster already bootstrapped")

	// ErrApplyTimeout is returned when a Raft apply operation times out.
	ErrApplyTimeout = errors.New("raft apply timeout")

	// ErrShutdown is returned when operations are attempted on a shutdown Raft node.
	ErrShutdown = errors.New("raft node is shut down")
)
