package rbac

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Package-level Prometheus metrics for RBAC operations.
var (
	// RBACRolesTotal tracks the total number of RBAC roles.
	RBACRolesTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "konsul_rbac_roles_total",
		Help: "Total number of RBAC roles.",
	})

	// RBACAssignmentsTotal tracks the total number of role assignments.
	RBACAssignmentsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "konsul_rbac_assignments_total",
		Help: "Total number of role assignments.",
	})

	// RBACAuthorizationDuration measures the duration of RBAC authorization checks.
	// Labels: result ("allow", "deny").
	RBACAuthorizationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "konsul_rbac_authorization_duration_seconds",
		Help:    "Duration of RBAC authorization checks.",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
	}, []string{"result"})

	// RBACCacheHitRatio tracks the cache hit ratio for role lookups.
	RBACCacheHitRatio = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "konsul_rbac_cache_hit_ratio",
		Help: "Cache hit ratio for role lookups.",
	})

	// RBACAssignmentsExpiredTotal counts expired role assignments cleaned up.
	RBACAssignmentsExpiredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "konsul_rbac_assignments_expired_total",
		Help: "Total number of expired role assignments cleaned up.",
	})
)
