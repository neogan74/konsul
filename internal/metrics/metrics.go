package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "konsul_http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "konsul_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// KV Store metrics
	KVOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_kv_operations_total",
			Help: "Total number of KV store operations",
		},
		[]string{"operation", "status"},
	)

	KVStoreSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "konsul_kv_store_size",
			Help: "Number of keys in the KV store",
		},
	)

	// Service Discovery metrics
	ServiceOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_service_operations_total",
			Help: "Total number of service operations",
		},
		[]string{"operation", "status"},
	)

	RegisteredServicesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "konsul_registered_services_total",
			Help: "Number of registered services",
		},
	)

	ServiceHeartbeatsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_service_heartbeats_total",
			Help: "Total number of service heartbeats",
		},
		[]string{"service", "status"},
	)

	ExpiredServicesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "konsul_expired_services_total",
			Help: "Total number of expired services cleaned up",
		},
	)

	// System metrics
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "konsul_build_info",
			Help: "Build information about Konsul",
		},
		[]string{"version", "go_version"},
	)

	// Rate limiting metrics
	RateLimitRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_rate_limit_requests_total",
			Help: "Total number of requests checked against rate limits",
		},
		[]string{"limiter_type", "status"},
	)

	RateLimitExceeded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_rate_limit_exceeded_total",
			Help: "Total number of requests that exceeded rate limits",
		},
		[]string{"limiter_type"},
	)

	RateLimitActiveClients = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "konsul_rate_limit_active_clients",
			Help: "Number of active clients being rate limited",
		},
		[]string{"limiter_type"},
	)
)
