package raft

import (
	"github.com/hashicorp/raft"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains Prometheus metrics for Raft operations.
type Metrics struct {
	// State metrics
	state        *prometheus.GaugeVec
	isLeader     prometheus.Gauge
	numPeers     prometheus.Gauge
	lastIndex    prometheus.Gauge
	commitIndex  prometheus.Gauge
	appliedIndex prometheus.Gauge

	// Operation counters
	applyTotal    *prometheus.CounterVec
	applyErrors   *prometheus.CounterVec
	leaderChanges prometheus.Counter
	snapshotTotal prometheus.Counter
	restoreTotal  prometheus.Counter

	// Latency histograms
	applyLatency    *prometheus.HistogramVec
	commitLatency   prometheus.Histogram
	snapshotLatency prometheus.Histogram

	// Replication metrics
	replicationLag prometheus.Gauge
	fsmPending     prometheus.Gauge
}

// NewMetrics creates and registers Raft metrics.
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "konsul"
	}

	m := &Metrics{
		state: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "state",
				Help:      "Current Raft state (0=Follower, 1=Candidate, 2=Leader, 3=Shutdown)",
			},
			[]string{"node_id"},
		),

		isLeader: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "is_leader",
				Help:      "Whether this node is the leader (1) or not (0)",
			},
		),

		numPeers: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "peers_total",
				Help:      "Number of peers in the cluster",
			},
		),

		lastIndex: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "last_index",
				Help:      "Last log index",
			},
		),

		commitIndex: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "commit_index",
				Help:      "Committed log index",
			},
		),

		appliedIndex: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "applied_index",
				Help:      "Applied log index",
			},
		),

		applyTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "apply_total",
				Help:      "Total number of Raft apply operations",
			},
			[]string{"command_type"},
		),

		applyErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "apply_errors_total",
				Help:      "Total number of Raft apply errors",
			},
			[]string{"command_type", "error_type"},
		),

		leaderChanges: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "leader_changes_total",
				Help:      "Total number of leader changes",
			},
		),

		snapshotTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "snapshot_total",
				Help:      "Total number of snapshots taken",
			},
		),

		restoreTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "restore_total",
				Help:      "Total number of snapshot restores",
			},
		),

		applyLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "apply_duration_seconds",
				Help:      "Time to apply a Raft log entry",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"command_type"},
		),

		commitLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "commit_duration_seconds",
				Help:      "Time from apply to commit",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
		),

		snapshotLatency: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "snapshot_duration_seconds",
				Help:      "Time to create a snapshot",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
		),

		replicationLag: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "replication_lag",
				Help:      "Difference between last index and applied index",
			},
		),

		fsmPending: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "raft",
				Name:      "fsm_pending",
				Help:      "Number of pending FSM operations",
			},
		),
	}

	return m
}

// SetState updates the state metric.
func (m *Metrics) SetState(nodeID string, state raft.RaftState) {
	m.state.WithLabelValues(nodeID).Set(float64(state))
	if state == raft.Leader {
		m.isLeader.Set(1)
	} else {
		m.isLeader.Set(0)
	}
}

// SetNumPeers updates the number of peers metric.
func (m *Metrics) SetNumPeers(count int) {
	m.numPeers.Set(float64(count))
}

// SetIndices updates the index metrics.
func (m *Metrics) SetIndices(last, commit, applied uint64) {
	m.lastIndex.Set(float64(last))
	m.commitIndex.Set(float64(commit))
	m.appliedIndex.Set(float64(applied))
	m.replicationLag.Set(float64(last - applied))
}

// SetFSMPending updates the FSM pending operations metric.
func (m *Metrics) SetFSMPending(count uint64) {
	m.fsmPending.Set(float64(count))
}

// IncApply increments the apply counter for a command type.
func (m *Metrics) IncApply(cmdType string) {
	m.applyTotal.WithLabelValues(cmdType).Inc()
}

// IncApplyError increments the apply error counter.
func (m *Metrics) IncApplyError(cmdType, errorType string) {
	m.applyErrors.WithLabelValues(cmdType, errorType).Inc()
}

// IncLeaderChanges increments the leader changes counter.
func (m *Metrics) IncLeaderChanges() {
	m.leaderChanges.Inc()
}

// IncSnapshot increments the snapshot counter.
func (m *Metrics) IncSnapshot() {
	m.snapshotTotal.Inc()
}

// IncRestore increments the restore counter.
func (m *Metrics) IncRestore() {
	m.restoreTotal.Inc()
}

// ObserveApplyLatency records the apply latency for a command type.
func (m *Metrics) ObserveApplyLatency(cmdType string, seconds float64) {
	m.applyLatency.WithLabelValues(cmdType).Observe(seconds)
}

// ObserveCommitLatency records the commit latency.
func (m *Metrics) ObserveCommitLatency(seconds float64) {
	m.commitLatency.Observe(seconds)
}

// ObserveSnapshotLatency records the snapshot latency.
func (m *Metrics) ObserveSnapshotLatency(seconds float64) {
	m.snapshotLatency.Observe(seconds)
}
