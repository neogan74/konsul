package raft

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AutopilotConfig configures the autopilot dead-server cleanup behaviour.
type AutopilotConfig struct {
	// Enabled turns autopilot on or off.
	Enabled bool

	// CleanupDeadServers controls whether unreachable servers are removed
	// from the cluster automatically. When false, only health tracking runs.
	CleanupDeadServers bool

	// LastContactThreshold is the TCP dial timeout used when probing a peer.
	// A peer that cannot be reached within this window counts as one failure.
	// Default: 10s.
	LastContactThreshold time.Duration

	// MaxFailures is the number of consecutive failed probes before a server
	// is considered dead and eligible for removal.
	// Default: 3.
	MaxFailures int

	// ServerStabilizationTime is how long a recovered server must be reachable
	// before its failure counter is reset.
	// Default: 10s.
	ServerStabilizationTime time.Duration

	// CleanupInterval controls how often the health check loop runs.
	// Default: 10s.
	CleanupInterval time.Duration
}

// DefaultAutopilotConfig returns a config with conservative, safe defaults.
func DefaultAutopilotConfig() *AutopilotConfig {
	return &AutopilotConfig{
		Enabled:                 false,
		CleanupDeadServers:      true,
		LastContactThreshold:    10 * time.Second,
		MaxFailures:             3,
		ServerStabilizationTime: 10 * time.Second,
		CleanupInterval:         10 * time.Second,
	}
}

// serverHealth tracks consecutive probe failures for a single server.
type serverHealth struct {
	failures    int
	lastOK      time.Time
	lastChecked time.Time
}

// healthy returns true when the server has had no recent failures.
func (h *serverHealth) healthy(maxFailures int) bool {
	return h.failures < maxFailures
}

// autopilotMetrics are Prometheus counters/gauges for autopilot events.
// They are registered once per process via promauto.
type autopilotMetrics struct {
	checksTotal   prometheus.Counter
	removalsTotal prometheus.Counter
	deadServers   prometheus.Gauge
}

var (
	apMetricsOnce sync.Once
	apMetrics     *autopilotMetrics
)

func getAutopilotMetrics() *autopilotMetrics {
	apMetricsOnce.Do(func() {
		apMetrics = &autopilotMetrics{
			checksTotal: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: "konsul",
				Subsystem: "raft_autopilot",
				Name:      "checks_total",
				Help:      "Total number of autopilot health check rounds.",
			}),
			removalsTotal: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: "konsul",
				Subsystem: "raft_autopilot",
				Name:      "removals_total",
				Help:      "Total number of dead servers removed by autopilot.",
			}),
			deadServers: promauto.NewGauge(prometheus.GaugeOpts{
				Namespace: "konsul",
				Subsystem: "raft_autopilot",
				Name:      "dead_servers",
				Help:      "Current number of servers considered dead by autopilot.",
			}),
		}
	})
	return apMetrics
}

// Autopilot monitors cluster health and removes dead servers from the Raft
// configuration when it is safe to do so (quorum is maintained).
//
// It only takes action when this node is the cluster leader.
type Autopilot struct {
	node    *Node
	cfg     *AutopilotConfig
	metrics *autopilotMetrics

	mu      sync.Mutex
	health  map[raft.ServerID]*serverHealth
	stopCh  chan struct{}
	stopped chan struct{}
}

// NewAutopilot creates a new Autopilot instance. Call Start to run it.
func NewAutopilot(node *Node, cfg *AutopilotConfig) *Autopilot {
	if cfg == nil {
		cfg = DefaultAutopilotConfig()
	}
	return &Autopilot{
		node:    node,
		cfg:     cfg,
		metrics: getAutopilotMetrics(),
		health:  make(map[raft.ServerID]*serverHealth),
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// Start launches the autopilot loop. It is a no-op if autopilot is disabled.
// The loop stops when ctx is cancelled or Stop is called.
func (ap *Autopilot) Start(ctx context.Context) {
	if !ap.cfg.Enabled {
		close(ap.stopped)
		return
	}

	go func() {
		defer close(ap.stopped)

		interval := ap.cfg.CleanupInterval
		if interval <= 0 {
			interval = 10 * time.Second
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		ap.node.logger.Info("autopilot started",
			"cleanup_dead_servers", ap.cfg.CleanupDeadServers,
			"cleanup_interval", interval,
			"max_failures", ap.cfg.MaxFailures,
		)

		for {
			select {
			case <-ctx.Done():
				ap.node.logger.Info("autopilot stopped (context cancelled)")
				return
			case <-ap.stopCh:
				ap.node.logger.Info("autopilot stopped")
				return
			case <-ticker.C:
				ap.runCheck()
			}
		}
	}()
}

// Stop requests the autopilot loop to stop and waits for it to exit.
func (ap *Autopilot) Stop() {
	select {
	case <-ap.stopCh:
	default:
		close(ap.stopCh)
	}
	<-ap.stopped
}

// runCheck is one iteration of the autopilot health-check loop.
func (ap *Autopilot) runCheck() {
	// Only the leader performs cleanup; followers just maintain health state.
	isLeader := ap.node.IsLeader()

	cfg, err := ap.node.GetConfiguration()
	if err != nil {
		ap.node.logger.Warn("autopilot: failed to get cluster configuration", "error", err)
		return
	}

	ap.metrics.checksTotal.Inc()

	ap.mu.Lock()
	defer ap.mu.Unlock()

	selfID := raft.ServerID(ap.node.config.NodeID)
	now := time.Now()

	// Probe every non-self server and update health state.
	for _, srv := range cfg.Servers {
		if srv.ID == selfID {
			continue
		}

		h, ok := ap.health[srv.ID]
		if !ok {
			h = &serverHealth{}
			ap.health[srv.ID] = h
		}
		h.lastChecked = now

		if ap.probePeer(string(srv.Address)) {
			// Peer responded — reset failures after stabilization time.
			if !h.lastOK.IsZero() && now.Sub(h.lastOK) >= ap.cfg.ServerStabilizationTime {
				h.failures = 0
			}
			h.lastOK = now
		} else {
			h.failures++
			ap.node.logger.Warn("autopilot: peer unreachable",
				"node_id", srv.ID,
				"addr", srv.Address,
				"consecutive_failures", h.failures,
			)
		}
	}

	// Remove stale entries for servers no longer in configuration.
	active := make(map[raft.ServerID]struct{})
	for _, srv := range cfg.Servers {
		active[srv.ID] = struct{}{}
	}
	for id := range ap.health {
		if _, ok := active[id]; !ok {
			delete(ap.health, id)
		}
	}

	// Count dead servers and update metric.
	dead := ap.deadServers()
	ap.metrics.deadServers.Set(float64(len(dead)))

	if !isLeader || !ap.cfg.CleanupDeadServers || len(dead) == 0 {
		return
	}

	// Remove dead servers one at a time, rechecking quorum each time.
	totalVoters := len(cfg.Servers)
	for _, srv := range dead {
		// Would removing this server maintain quorum?
		// Quorum requires strictly more than half of voters to be healthy.
		// After removal: healthy = totalVoters - 1 (removed) - len(dead)+1 (this one)
		// Actually recalculate: after removal, cluster has (totalVoters - 1) servers.
		// We need healthy_servers > (totalVoters-1)/2.
		healthyAfterRemoval := ap.countHealthyExcluding(cfg.Servers, selfID, srv.ID)
		quorumAfterRemoval := (totalVoters - 1) / 2

		if healthyAfterRemoval <= quorumAfterRemoval {
			ap.node.logger.Warn("autopilot: skipping removal to preserve quorum",
				"dead_node_id", srv.ID,
				"healthy_after", healthyAfterRemoval,
				"quorum_needed", quorumAfterRemoval+1,
			)
			continue
		}

		ap.node.logger.Info("autopilot: removing dead server",
			"node_id", srv.ID,
			"addr", srv.Address,
			"consecutive_failures", ap.health[srv.ID].failures,
		)

		if err := ap.node.Leave(string(srv.ID)); err != nil {
			ap.node.logger.Error("autopilot: failed to remove dead server",
				"node_id", srv.ID,
				"error", err,
			)
		} else {
			ap.metrics.removalsTotal.Inc()
			delete(ap.health, srv.ID)
			totalVoters-- // Adjust for subsequent iterations.
		}
	}
}

// probePeer attempts a TCP connection to addr. Returns true if reachable.
func (ap *Autopilot) probePeer(addr string) bool {
	timeout := ap.cfg.LastContactThreshold
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// deadServers returns servers whose consecutive failure count exceeds MaxFailures.
func (ap *Autopilot) deadServers() []raft.Server {
	cfg, err := ap.node.GetConfiguration()
	if err != nil {
		return nil
	}

	selfID := raft.ServerID(ap.node.config.NodeID)
	var dead []raft.Server

	for _, srv := range cfg.Servers {
		if srv.ID == selfID {
			continue
		}
		h, ok := ap.health[srv.ID]
		if !ok {
			continue
		}
		if !h.healthy(ap.cfg.MaxFailures) {
			dead = append(dead, srv)
		}
	}
	return dead
}

// countHealthyExcluding counts servers that are healthy, excluding selfID and
// excludeID from consideration.
func (ap *Autopilot) countHealthyExcluding(servers []raft.Server, selfID, excludeID raft.ServerID) int {
	count := 0
	for _, srv := range servers {
		if srv.ID == excludeID {
			continue
		}
		if srv.ID == selfID {
			count++ // This node is always considered healthy.
			continue
		}
		h, ok := ap.health[srv.ID]
		if !ok || h.healthy(ap.cfg.MaxFailures) {
			count++
		}
	}
	return count
}

// HealthReport returns a snapshot of the current peer health for diagnostics.
func (ap *Autopilot) HealthReport() map[string]string {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	report := make(map[string]string, len(ap.health))
	for id, h := range ap.health {
		status := "healthy"
		if !h.healthy(ap.cfg.MaxFailures) {
			status = fmt.Sprintf("dead (failures=%d)", h.failures)
		} else if h.failures > 0 {
			status = fmt.Sprintf("degraded (failures=%d)", h.failures)
		}
		report[string(id)] = status
	}
	return report
}
