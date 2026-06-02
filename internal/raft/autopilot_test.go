package raft

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DefaultAutopilotConfig ---

func TestDefaultAutopilotConfig(t *testing.T) {
	cfg := DefaultAutopilotConfig()
	assert.False(t, cfg.Enabled)
	assert.True(t, cfg.CleanupDeadServers)
	assert.Equal(t, 10*time.Second, cfg.LastContactThreshold)
	assert.Equal(t, 3, cfg.MaxFailures)
	assert.Equal(t, 10*time.Second, cfg.CleanupInterval)
}

// --- serverHealth ---

func TestServerHealth_HealthyWhenNoFailures(t *testing.T) {
	h := &serverHealth{}
	assert.True(t, h.healthy(3))
}

func TestServerHealth_UnhealthyAtMaxFailures(t *testing.T) {
	h := &serverHealth{failures: 3}
	assert.False(t, h.healthy(3))
}

func TestServerHealth_HealthyBelowMaxFailures(t *testing.T) {
	h := &serverHealth{failures: 2}
	assert.True(t, h.healthy(3))
}

// --- Autopilot.probePeer ---

func TestAutopilot_ProbePeer_ReachableServer(t *testing.T) {
	// Start a listener to simulate a reachable Raft peer.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	ap := &Autopilot{
		cfg: &AutopilotConfig{LastContactThreshold: time.Second},
	}
	assert.True(t, ap.probePeer(ln.Addr().String()))
}

func TestAutopilot_ProbePeer_UnreachableServer(t *testing.T) {
	ap := &Autopilot{
		cfg: &AutopilotConfig{LastContactThreshold: 100 * time.Millisecond},
	}
	// Port 1 is privileged and always refused.
	assert.False(t, ap.probePeer("127.0.0.1:1"))
}

// --- Autopilot.countHealthyExcluding ---

func TestAutopilot_CountHealthyExcluding(t *testing.T) {
	servers := []raft.Server{
		{ID: "node1"},
		{ID: "node2"},
		{ID: "node3"},
	}
	ap := &Autopilot{
		cfg: &AutopilotConfig{MaxFailures: 3},
		health: map[raft.ServerID]*serverHealth{
			"node2": {failures: 0}, // healthy
			"node3": {failures: 5}, // dead
		},
	}

	// Excluding node3 (dead), count healthy: self(node1) + node2 = 2
	count := ap.countHealthyExcluding(servers, "node1", "node3")
	assert.Equal(t, 2, count)
}

func TestAutopilot_CountHealthyExcluding_SelfAlwaysHealthy(t *testing.T) {
	servers := []raft.Server{
		{ID: "node1"},
		{ID: "node2"},
	}
	ap := &Autopilot{
		cfg:    &AutopilotConfig{MaxFailures: 3},
		health: map[raft.ServerID]*serverHealth{},
	}

	// Self is always counted as healthy.
	count := ap.countHealthyExcluding(servers, "node1", "node2")
	assert.Equal(t, 1, count)
}

// --- Autopilot.HealthReport ---

func TestAutopilot_HealthReport_Empty(t *testing.T) {
	ap := &Autopilot{
		cfg:    &AutopilotConfig{MaxFailures: 3},
		health: map[raft.ServerID]*serverHealth{},
	}
	assert.Empty(t, ap.HealthReport())
}

func TestAutopilot_HealthReport_Classifications(t *testing.T) {
	ap := &Autopilot{
		cfg: &AutopilotConfig{MaxFailures: 3},
		health: map[raft.ServerID]*serverHealth{
			"node2": {failures: 0},
			"node3": {failures: 1},
			"node4": {failures: 3},
		},
	}
	report := ap.HealthReport()
	assert.Equal(t, "healthy", report["node2"])
	assert.Contains(t, report["node3"], "degraded")
	assert.Contains(t, report["node4"], "dead")
}

// --- Autopilot Start/Stop lifecycle ---

func TestAutopilot_StartStop_DisabledIsNoop(t *testing.T) {
	ap := &Autopilot{
		cfg:     &AutopilotConfig{Enabled: false},
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ap.Start(ctx)

	// stopped channel should be closed immediately when disabled.
	select {
	case <-ap.stopped:
	case <-time.After(time.Second):
		t.Fatal("autopilot did not stop immediately when disabled")
	}
}

func TestAutopilot_Stop_CanBeCalledMultipleTimes(t *testing.T) {
	// A disabled autopilot closes stopped in Start immediately.
	ap := &Autopilot{
		cfg:     &AutopilotConfig{Enabled: false},
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
	ap.Start(context.Background())

	// Calling Stop multiple times must not panic.
	ap.Stop()
	ap.Stop()
	ap.Stop()
}
