package main

import (
	"flag"
	"fmt"
)

// ClusterCommands handles all cluster management commands.
type ClusterCommands struct {
	cli *CLI
}

// NewClusterCommands creates a new cluster commands handler.
func NewClusterCommands(cli *CLI) *ClusterCommands {
	return &ClusterCommands{cli: cli}
}

// Handle routes cluster subcommands.
func (cc *ClusterCommands) Handle(args []string) {
	if len(args) == 0 {
		cc.cli.Errorln("Cluster subcommand required")
		cc.cli.Errorln("Usage: konsulctl cluster <status|leader|peers|join|leave|snapshot> [options]")
		cc.cli.Exit(1)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "status":
		cc.Status(subArgs)
	case "leader":
		cc.Leader(subArgs)
	case "peers":
		cc.Peers(subArgs)
	case "join":
		cc.Join(subArgs)
	case "leave":
		cc.Leave(subArgs)
	case "snapshot":
		cc.Snapshot(subArgs)
	default:
		cc.cli.Errorf("Unknown cluster subcommand: %s\n", subcommand)
		cc.cli.Errorln("Available: status, leader, peers, join, leave, snapshot")
		cc.cli.Exit(1)
	}
}

// Status prints the current cluster status.
func (cc *ClusterCommands) Status(args []string) {
	config, remaining, err := cc.cli.ParseGlobalFlags(args, "status")
	if err == flag.ErrHelp {
		cc.cli.Println("Usage: konsulctl cluster status [options]")
		return
	}
	cc.cli.HandleError(err, "parsing flags")
	cc.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl cluster status")

	client := cc.cli.CreateClient(config)

	info, err := client.ClusterStatus()
	cc.cli.HandleError(err, "fetching cluster status")

	cc.cli.Printf("Node ID:      %s\n", info.NodeID)
	cc.cli.Printf("State:        %s\n", info.State)
	cc.cli.Printf("Leader ID:    %s\n", info.LeaderID)
	cc.cli.Printf("Leader Addr:  %s\n", info.LeaderAddr)
	cc.cli.Printf("Last Index:   %d\n", info.LastIndex)
	cc.cli.Printf("Applied Idx:  %d\n", info.AppliedIdx)
	cc.cli.Printf("Commit Index: %d\n", info.CommitIndex)
	cc.cli.Printf("Peers (%d):\n", len(info.Peers))
	for _, p := range info.Peers {
		cc.cli.Printf("  - %s  %s  [%s]\n", p.ID, p.Address, p.State)
	}
	cc.cli.Printf("Raft Stats:\n")
	cc.cli.Printf("  Term:               %d\n", info.Stats.Term)
	cc.cli.Printf("  Last Log Index:     %d\n", info.Stats.LastLogIndex)
	cc.cli.Printf("  Commit Index:       %d\n", info.Stats.CommitIndex)
	cc.cli.Printf("  Applied Index:      %d\n", info.Stats.AppliedIndex)
	cc.cli.Printf("  Last Snapshot Idx:  %d\n", info.Stats.LastSnapshotIndex)
	cc.cli.Printf("  Num Peers:          %d\n", info.Stats.NumPeers)
}

// Leader prints the current leader information.
func (cc *ClusterCommands) Leader(args []string) {
	config, remaining, err := cc.cli.ParseGlobalFlags(args, "leader")
	if err == flag.ErrHelp {
		cc.cli.Println("Usage: konsulctl cluster leader [options]")
		return
	}
	cc.cli.HandleError(err, "parsing flags")
	cc.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl cluster leader")

	client := cc.cli.CreateClient(config)

	info, err := client.ClusterLeader()
	cc.cli.HandleError(err, "fetching cluster leader")

	cc.cli.Printf("Leader ID:   %s\n", info.LeaderID)
	cc.cli.Printf("Leader Addr: %s\n", info.LeaderAddr)
	if info.IsSelf {
		cc.cli.Println("(this node is the leader)")
	}
}

// Peers prints the list of cluster peers.
func (cc *ClusterCommands) Peers(args []string) {
	config, remaining, err := cc.cli.ParseGlobalFlags(args, "peers")
	if err == flag.ErrHelp {
		cc.cli.Println("Usage: konsulctl cluster peers [options]")
		return
	}
	cc.cli.HandleError(err, "parsing flags")
	cc.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl cluster peers")

	client := cc.cli.CreateClient(config)

	info, err := client.ClusterPeers()
	cc.cli.HandleError(err, "fetching cluster peers")

	cc.cli.Printf("Peers (%d):\n", info.Count)
	for _, p := range info.Peers {
		cc.cli.Printf("  %-20s  %-25s  %s\n", p.ID, p.Address, p.State)
	}
}

// Join adds a node to the cluster.
// Usage: konsulctl cluster join <node-id> <address>
func (cc *ClusterCommands) Join(args []string) {
	config, remaining, err := cc.cli.ParseGlobalFlags(args, "join")
	if err == flag.ErrHelp {
		cc.cli.Println("Usage: konsulctl cluster join <node-id> <address> [options]")
		cc.cli.Println("  node-id   Unique identifier of the node to join")
		cc.cli.Println("  address   Raft advertise address of the joining node (host:port)")
		return
	}
	cc.cli.HandleError(err, "parsing flags")
	cc.cli.ValidateExactArgs(remaining, 2, "Usage: konsulctl cluster join <node-id> <address>")

	nodeID := remaining[0]
	address := remaining[1]

	client := cc.cli.CreateClient(config)

	result, err := client.ClusterJoin(nodeID, address)
	cc.cli.HandleError(err, "joining cluster")

	cc.cli.Printf("Node %s (%s) joined successfully\n", result.NodeID, result.Address)
	if result.Message != "" {
		cc.cli.Printf("Message: %s\n", result.Message)
	}
}

// Leave removes a node from the cluster.
// Usage: konsulctl cluster leave <node-id>
func (cc *ClusterCommands) Leave(args []string) {
	config, remaining, err := cc.cli.ParseGlobalFlags(args, "leave")
	if err == flag.ErrHelp {
		cc.cli.Println("Usage: konsulctl cluster leave <node-id> [options]")
		cc.cli.Println("  node-id   Unique identifier of the node to remove")
		return
	}
	cc.cli.HandleError(err, "parsing flags")
	cc.cli.ValidateExactArgs(remaining, 1, "Usage: konsulctl cluster leave <node-id>")

	nodeID := remaining[0]

	client := cc.cli.CreateClient(config)

	result, err := client.ClusterLeave(nodeID)
	cc.cli.HandleError(err, "removing node from cluster")

	cc.cli.Printf("Node %q removed from cluster\n", nodeID)
	if result.Message != "" {
		cc.cli.Printf("Message: %s\n", result.Message)
	}
}

// Snapshot triggers a Raft snapshot.
func (cc *ClusterCommands) Snapshot(args []string) {
	config, remaining, err := cc.cli.ParseGlobalFlags(args, "snapshot")
	if err == flag.ErrHelp {
		cc.cli.Println("Usage: konsulctl cluster snapshot [options]")
		return
	}
	cc.cli.HandleError(err, "parsing flags")
	cc.cli.ValidateExactArgs(remaining, 0, "Usage: konsulctl cluster snapshot")

	client := cc.cli.CreateClient(config)

	result, err := client.ClusterSnapshot()
	cc.cli.HandleError(err, "triggering snapshot")

	msg := result.Message
	if msg == "" {
		msg = fmt.Sprintf("status=%s", result.Status)
	}
	cc.cli.Printf("Snapshot triggered: %s\n", msg)
}
