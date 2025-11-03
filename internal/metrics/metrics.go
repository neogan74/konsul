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

	// ACL metrics
	ACLEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_acl_evaluations_total",
			Help: "Total number of ACL policy evaluations",
		},
		[]string{"resource_type", "capability", "result"},
	)

	ACLEvaluationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "konsul_acl_evaluation_duration_seconds",
			Help:    "ACL policy evaluation latencies in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"resource_type"},
	)

	ACLPoliciesLoaded = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "konsul_acl_policies_loaded",
			Help: "Number of ACL policies currently loaded",
		},
	)

	ACLPolicyLoadErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "konsul_acl_policy_load_errors_total",
			Help: "Total number of ACL policy load errors",
		},
	)

	// Service Query metrics (tags/metadata)
	ServiceQueryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_service_query_total",
			Help: "Total number of service queries by type",
		},
		[]string{"query_type", "status"},
	)

	ServiceQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "konsul_service_query_duration_seconds",
			Help:    "Service query latencies in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"query_type"},
	)

	ServiceQueryResultsCount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "konsul_service_query_results_count",
			Help:    "Number of services returned by queries",
			Buckets: []float64{0, 1, 5, 10, 25, 50, 100, 250, 500},
		},
		[]string{"query_type"},
	)

	ServiceTagsPerService = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "konsul_service_tags_per_service",
			Help:    "Number of tags per registered service",
			Buckets: []float64{0, 1, 2, 5, 10, 20, 30, 50, 64},
		},
	)

	ServiceMetadataKeysPerService = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "konsul_service_metadata_keys_per_service",
			Help:    "Number of metadata keys per registered service",
			Buckets: []float64{0, 1, 2, 5, 10, 20, 30, 50, 64},
		},
	)

	// Load Balancer metrics
	LoadBalancerSelectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_load_balancer_selections_total",
			Help: "Total number of load balancer service selections",
		},
		[]string{"strategy", "selection_type", "status"},
	)

	LoadBalancerSelectionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "konsul_load_balancer_selection_duration_seconds",
			Help:    "Load balancer selection latencies in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05},
		},
		[]string{"strategy", "selection_type"},
	)

	LoadBalancerActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "konsul_load_balancer_active_connections",
			Help: "Number of active connections per service instance (for least-connections strategy)",
		},
		[]string{"service_name", "instance"},
	)

	LoadBalancerStrategyChanges = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_load_balancer_strategy_changes_total",
			Help: "Total number of load balancing strategy changes",
		},
		[]string{"from_strategy", "to_strategy"},
	)

	LoadBalancerCurrentStrategy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "konsul_load_balancer_current_strategy",
			Help: "Current load balancing strategy (1 for active strategy, 0 for others)",
		},
		[]string{"strategy"},
	)

	LoadBalancerInstancePoolSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "konsul_load_balancer_instance_pool_size",
			Help:    "Number of available instances in the load balancer pool",
			Buckets: []float64{0, 1, 2, 5, 10, 20, 50, 100},
		},
		[]string{"selection_type"},
	)
)
