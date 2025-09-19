package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsRegistration(t *testing.T) {
	// Test that all metrics are properly registered
	// by checking that they can be collected without error
	registry := prometheus.NewRegistry()

	// Create new instances to avoid conflicts with global registry
	httpRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_http_requests_total",
			Help: "Test HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	kvOps := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_kv_operations_total",
			Help: "Test KV operations",
		},
		[]string{"operation", "status"},
	)

	kvSize := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "test_kv_store_size",
			Help: "Test KV store size",
		},
	)

	// Register metrics
	if err := registry.Register(httpRequests); err != nil {
		t.Fatalf("Failed to register HTTP requests metric: %v", err)
	}

	if err := registry.Register(kvOps); err != nil {
		t.Fatalf("Failed to register KV operations metric: %v", err)
	}

	if err := registry.Register(kvSize); err != nil {
		t.Fatalf("Failed to register KV size metric: %v", err)
	}

	// Test metric updates
	httpRequests.WithLabelValues("GET", "/test", "200").Inc()
	kvOps.WithLabelValues("get", "success").Inc()
	kvSize.Set(42)

	// Gather metrics to ensure they're working
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(metricFamilies) != 3 {
		t.Errorf("Expected 3 metric families, got %d", len(metricFamilies))
	}
}

func TestHTTPMetrics(t *testing.T) {
	// Test that HTTP metrics can be updated without panicking
	HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
	HTTPRequestDuration.WithLabelValues("GET", "/test", "200").Observe(0.1)

	// Test in-flight gauge
	HTTPRequestsInFlight.Inc()
	HTTPRequestsInFlight.Dec()
}

func TestKVMetrics(t *testing.T) {
	// Test that KV metrics can be updated without panicking
	KVOperationsTotal.WithLabelValues("get", "success").Inc()
	KVOperationsTotal.WithLabelValues("set", "success").Inc()
	KVOperationsTotal.WithLabelValues("delete", "success").Inc()

	KVStoreSize.Set(10)
}

func TestServiceMetrics(t *testing.T) {
	// Test that service metrics can be updated without panicking
	ServiceOperationsTotal.WithLabelValues("register", "success").Inc()
	ServiceOperationsTotal.WithLabelValues("deregister", "success").Inc()
	ServiceOperationsTotal.WithLabelValues("get", "not_found").Inc()

	RegisteredServicesTotal.Set(5)

	ServiceHeartbeatsTotal.WithLabelValues("service1", "success").Inc()
	ServiceHeartbeatsTotal.WithLabelValues("service2", "not_found").Inc()

	ExpiredServicesTotal.Add(2)
}

func TestBuildMetrics(t *testing.T) {
	// Test that build info can be set without panicking
	BuildInfo.WithLabelValues("1.0.0", "go1.24").Set(1)
}