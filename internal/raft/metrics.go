package raft

import (
	"sync"

	"github.com/hashicorp/raft"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

// Metrics contains Prometheus metrics for Raft operations.
type Metrics struct {
	// State metrics
	state        *prometheus.GaugeVec
	isLeader     *prometheus.GaugeVec
	numPeers     *prometheus.GaugeVec
	lastIndex    *prometheus.GaugeVec
	commitIndex  *prometheus.GaugeVec
	appliedIndex *prometheus.GaugeVec

	// Operation counters
	applyTotal    *prometheus.CounterVec
	applyErrors   *prometheus.CounterVec
	leaderChanges *prometheus.CounterVec
	snapshotTotal *prometheus.CounterVec
	restoreTotal  *prometheus.CounterVec

	// Latency histograms
	applyLatency    *prometheus.HistogramVec
	commitLatency   *prometheus.HistogramVec
	snapshotLatency *prometheus.HistogramVec

	// Replication metrics
	replicationLag *prometheus.GaugeVec
	fsmPending     *prometheus.GaugeVec
}

// NewMetrics creates and registers Raft metrics.
// NewMetrics creates and registers Raft metrics.
// Returns a singleton instance.
func NewMetrics(namespace string) *Metrics {
	metricsOnce.Do(func() {
		if namespace == "" {
			namespace = "konsul"
		}

		metricsInstance = &Metrics{
			state: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "state",
					Help:      "Current Raft state (0=Follower, 1=Candidate, 2=Leader, 3=Shutdown)",
				},
				[]string{"node_id"},
			),

			isLeader: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "is_leader",
					Help:      "Whether this node is the leader (1) or not (0)",
				},
				[]string{"node_id"},
			),

			numPeers: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "peers_total",
					Help:      "Number of peers in the cluster",
				},
				[]string{"node_id"},
			),

			lastIndex: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "last_index",
					Help:      "Last log index",
				},
				[]string{"node_id"},
			),

			commitIndex: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "commit_index",
					Help:      "Committed log index",
				},
				[]string{"node_id"},
			),

			appliedIndex: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "applied_index",
					Help:      "Applied log index",
				},
				[]string{"node_id"},
			),

			applyTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "apply_total",
					Help:      "Total number of Raft apply operations",
				},
				[]string{"node_id", "command_type"},
			),

			applyErrors: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "apply_errors_total",
					Help:      "Total number of Raft apply errors",
				},
				[]string{"node_id", "command_type", "error_type"},
			),

			leaderChanges: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "leader_changes_total",
					Help:      "Total number of leader changes",
				},
				[]string{"node_id"},
			),

			snapshotTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "snapshot_total",
					Help:      "Total number of snapshots taken",
				},
				[]string{"node_id"},
			),

			restoreTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "restore_total",
					Help:      "Total number of snapshot restores",
				},
				[]string{"node_id"},
			),

			applyLatency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "apply_duration_seconds",
					Help:      "Time to apply a Raft log entry",
					Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
				},
				[]string{"node_id", "command_type"},
			),

			commitLatency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "commit_duration_seconds",
					Help:      "Time from apply to commit",
					Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
				},
				[]string{"node_id"},
			),

			snapshotLatency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "snapshot_duration_seconds",
					Help:      "Time to create a snapshot",
					Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
				},
				[]string{"node_id"},
			),

			replicationLag: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "replication_lag",
					Help:      "Difference between last index and applied index",
				},
				[]string{"node_id"},
			),

			fsmPending: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "raft",
					Name:      "fsm_pending",
					Help:      "Number of pending FSM operations",
				},
				[]string{"node_id"},
			),
		}
	})

	return metricsInstance
}

// SetState updates the state metric.
func (m *Metrics) SetState(nodeID string, state raft.RaftState) {
	m.state.WithLabelValues(nodeID).Set(float64(state))
	if state == raft.Leader {
		m.isLeader.WithLabelValues(nodeID).Set(1)
	} else {
		m.isLeader.WithLabelValues(nodeID).Set(0)
	}
}

// SetNumPeers updates the number of peers metric.
func (m *Metrics) SetNumPeers(nodeID string, count int) {
	m.numPeers.WithLabelValues(nodeID).Set(float64(count))
}

// SetIndices updates the index metrics.
func (m *Metrics) SetIndices(nodeID string, last, commit, applied uint64) {
	m.lastIndex.WithLabelValues(nodeID).Set(float64(last))
	m.commitIndex.WithLabelValues(nodeID).Set(float64(commit))
	m.appliedIndex.WithLabelValues(nodeID).Set(float64(applied))
	m.replicationLag.WithLabelValues(nodeID).Set(float64(last - applied))
}

// SetFSMPending updates the FSM pending operations metric.
func (m *Metrics) SetFSMPending(nodeID string, count uint64) {
	m.fsmPending.WithLabelValues(nodeID).Set(float64(count))
}

// IncApply increments the apply counter for a command type.
func (m *Metrics) IncApply(nodeID, cmdType string) {
	m.applyTotal.WithLabelValues(nodeID, cmdType).Inc()
}

// IncApplyError increments the apply error counter.
func (m *Metrics) IncApplyError(nodeID, cmdType, errorType string) {
	m.applyErrors.WithLabelValues(nodeID, cmdType, errorType).Inc()
}

// IncLeaderChanges increments the leader changes counter.
func (m *Metrics) IncLeaderChanges(nodeID string) {
	m.leaderChanges.WithLabelValues(nodeID).Inc()
}

// IncSnapshot increments the snapshot counter.
func (m *Metrics) IncSnapshot(nodeID string) {
	m.snapshotTotal.WithLabelValues(nodeID).Inc()
}

// IncRestore increments the restore counter.
func (m *Metrics) IncRestore(nodeID string) {
	m.restoreTotal.WithLabelValues(nodeID).Inc()
}

// ObserveApplyLatency records the apply latency for a command type.
func (m *Metrics) ObserveApplyLatency(nodeID, cmdType string, seconds float64) {
	m.applyLatency.WithLabelValues(nodeID, cmdType).Observe(seconds)
}

// ObserveCommitLatency records the commit latency.
func (m *Metrics) ObserveCommitLatency(nodeID string, seconds float64) {
	m.commitLatency.WithLabelValues(nodeID).Observe(seconds)
}

// ObserveSnapshotLatency records the snapshot latency.
func (m *Metrics) ObserveSnapshotLatency(nodeID string, seconds float64) {
	m.snapshotLatency.WithLabelValues(nodeID).Observe(seconds)
}
