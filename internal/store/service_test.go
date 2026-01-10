package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/healthcheck"
)

func TestServiceStore_RegisterAndGet(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	got, ok := s.Get("auth")
	if !ok {
		t.Fatalf("expected service to be registered")
	}
	if got.Name != service.Name || got.Address != service.Address || got.Port != service.Port {
		t.Errorf("got %+v, want %+v", got, service)
	}
}

func TestServiceStore_List(t *testing.T) {
	s := NewServiceStore()
	services := []Service{
		{Name: "auth", Address: "10.0.0.1", Port: 8080},
		{Name: "db", Address: "10.0.0.2", Port: 5432},
	}
	for _, svc := range services {
		s.Register(svc)
	}
	list := s.List()
	if len(list) != 2 {
		t.Errorf("expected 2 services, got %d", len(list))
	}
}

func TestServiceStore_Deregister(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	s.Deregister("auth")
	_, ok := s.Get("auth")
	if ok {
		t.Errorf("expected service to be deregistered")
	}
}

func TestServiceStore_Heartbeat(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}

	// Test heartbeat on non-existent service
	if s.Heartbeat("nonexistent") {
		t.Errorf("expected heartbeat to fail for non-existent service")
	}

	// Register service and test successful heartbeat
	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !s.Heartbeat("auth") {
		t.Errorf("expected heartbeat to succeed for registered service")
	}

	// Verify service is still accessible after heartbeat
	got, ok := s.Get("auth")
	if !ok {
		t.Fatalf("expected service to be available after heartbeat")
	}
	if got.Name != service.Name || got.Address != service.Address || got.Port != service.Port {
		t.Errorf("got %+v, want %+v", got, service)
	}
}

func TestServiceStore_TTLExpiration(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}

	// Register service
	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Service should be available immediately
	_, ok := s.Get("auth")
	if !ok {
		t.Errorf("expected service to be available immediately after registration")
	}

	// Manually expire the service by setting past expiration time
	s.Mutex.Lock()
	entry := s.Data["auth"]
	entry.ExpiresAt = time.Now().Add(-1 * time.Second)
	s.Data["auth"] = entry
	s.Mutex.Unlock()

	// Service should no longer be accessible
	_, ok = s.Get("auth")
	if ok {
		t.Errorf("expected expired service to not be accessible")
	}

	// Service should not appear in list
	services := s.List()
	if len(services) != 0 {
		t.Errorf("expected no services in list, got %d", len(services))
	}
}

func TestServiceStore_CleanupExpired(t *testing.T) {
	s := NewServiceStore()
	service1 := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	service2 := Service{Name: "db", Address: "10.0.0.2", Port: 5432}

	// Register both services
	if err := s.Register(service1); err != nil {
		t.Fatalf("Register service1 failed: %v", err)
	}
	if err := s.Register(service2); err != nil {
		t.Fatalf("Register service2 failed: %v", err)
	}

	// Manually expire one service
	s.Mutex.Lock()
	entry := s.Data["auth"]
	entry.ExpiresAt = time.Now().Add(-1 * time.Second)
	s.Data["auth"] = entry
	s.Mutex.Unlock()

	// Run cleanup
	count := s.CleanupExpired()
	if count != 1 {
		t.Errorf("expected to clean up 1 service, got %d", count)
	}

	// Verify only non-expired service remains
	services := s.List()
	if len(services) != 1 {
		t.Errorf("expected 1 service after cleanup, got %d", len(services))
	}
	if services[0].Name != service2.Name || services[0].Address != service2.Address || services[0].Port != service2.Port {
		t.Errorf("expected service2 to remain, got %+v", services[0])
	}

	// Verify expired service is completely gone
	_, ok := s.Get("auth")
	if ok {
		t.Errorf("expected expired service to be completely removed")
	}
}

func TestServiceStore_HeartbeatExtendsExpiration(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}

	// Register service
	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get initial expiration time
	s.Mutex.RLock()
	initialExpiry := s.Data["auth"].ExpiresAt
	s.Mutex.RUnlock()

	// Wait a small amount of time, then send heartbeat
	time.Sleep(10 * time.Millisecond)
	s.Heartbeat("auth")

	// Get new expiration time
	s.Mutex.RLock()
	newExpiry := s.Data["auth"].ExpiresAt
	s.Mutex.RUnlock()

	// New expiry should be later than initial
	if !newExpiry.After(initialExpiry) {
		t.Errorf("expected heartbeat to extend expiration time")
	}
}

func TestServiceStore_NewServiceStoreWithTTL(t *testing.T) {
	customTTL := 60 * time.Second
	s := NewServiceStoreWithTTL(customTTL)

	if s.TTL != customTTL {
		t.Errorf("expected TTL %v, got %v", customTTL, s.TTL)
	}

	// Register service and verify TTL is used
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	s.Mutex.RLock()
	entry := s.Data["auth"]
	s.Mutex.RUnlock()

	expectedExpiry := time.Now().Add(customTTL)
	// Allow 1 second tolerance
	if entry.ExpiresAt.Before(expectedExpiry.Add(-1*time.Second)) || entry.ExpiresAt.After(expectedExpiry.Add(1*time.Second)) {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, entry.ExpiresAt)
	}
}

func TestServiceStore_ListAll(t *testing.T) {
	s := NewServiceStoreWithTTL(100 * time.Millisecond)

	service1 := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	service2 := Service{Name: "db", Address: "10.0.0.2", Port: 5432}

	if err := s.Register(service1); err != nil {
		t.Fatalf("Register service1 failed: %v", err)
	}
	if err := s.Register(service2); err != nil {
		t.Fatalf("Register service2 failed: %v", err)
	}

	// Manually expire one service
	s.Mutex.Lock()
	entry := s.Data["auth"]
	entry.ExpiresAt = time.Now().Add(-1 * time.Second)
	s.Data["auth"] = entry
	s.Mutex.Unlock()

	// ListAll should return all entries including expired
	allEntries := s.ListAll()
	if len(allEntries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(allEntries))
	}

	// List should return only non-expired
	services := s.List()
	if len(services) != 1 {
		t.Errorf("expected 1 non-expired service, got %d", len(services))
	}
}

func TestServiceStore_GetNonExistent(t *testing.T) {
	s := NewServiceStore()

	_, ok := s.Get("nonexistent")
	if ok {
		t.Error("expected Get to return false for non-existent service")
	}
}

func TestServiceStore_GetExpired(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}

	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Manually expire the service
	s.Mutex.Lock()
	entry := s.Data["auth"]
	entry.ExpiresAt = time.Now().Add(-1 * time.Second)
	s.Data["auth"] = entry
	s.Mutex.Unlock()

	// Get should return false for expired service
	_, ok := s.Get("auth")
	if ok {
		t.Error("expected Get to return false for expired service")
	}
}

func TestServiceStore_DeregisterNonExistent(t *testing.T) {
	s := NewServiceStore()

	// Deregister non-existent service should not panic
	s.Deregister("nonexistent")

	// Store should still work
	service := Service{Name: "test", Address: "10.0.0.1", Port: 8080}
	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	got, ok := s.Get("test")
	if !ok || got.Name != "test" {
		t.Error("store should still work after deregistering non-existent service")
	}
}

func TestServiceStore_CleanupExpiredMultiple(t *testing.T) {
	s := NewServiceStore()

	// Register several services
	services := []Service{
		{Name: "service1", Address: "10.0.0.1", Port: 8081},
		{Name: "service2", Address: "10.0.0.2", Port: 8082},
		{Name: "service3", Address: "10.0.0.3", Port: 8083},
		{Name: "service4", Address: "10.0.0.4", Port: 8084},
	}

	for _, svc := range services {
		if err := s.Register(svc); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	// Expire half of them
	s.Mutex.Lock()
	for _, name := range []string{"service1", "service3"} {
		entry := s.Data[name]
		entry.ExpiresAt = time.Now().Add(-1 * time.Second)
		s.Data[name] = entry
	}
	s.Mutex.Unlock()

	// Run cleanup
	count := s.CleanupExpired()
	if count != 2 {
		t.Errorf("expected to clean up 2 services, got %d", count)
	}

	// Verify only non-expired services remain
	remaining := s.List()
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining services, got %d", len(remaining))
	}

	// Verify specific services
	expectedNames := map[string]bool{"service2": true, "service4": true}
	for _, svc := range remaining {
		if !expectedNames[svc.Name] {
			t.Errorf("unexpected service %s in remaining list", svc.Name)
		}
	}
}

func TestServiceStore_CleanupExpiredNone(t *testing.T) {
	s := NewServiceStore()

	// Register services
	if err := s.Register(Service{Name: "service1", Address: "10.0.0.1", Port: 8081}); err != nil {
		t.Fatalf("Register service1 failed: %v", err)
	}
	if err := s.Register(Service{Name: "service2", Address: "10.0.0.2", Port: 8082}); err != nil {
		t.Fatalf("Register service2 failed: %v", err)
	}

	// None expired
	count := s.CleanupExpired()
	if count != 0 {
		t.Errorf("expected to clean up 0 services, got %d", count)
	}

	// All should remain
	services := s.List()
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}

func TestServiceStore_Close(t *testing.T) {
	s := NewServiceStore()

	// Close should not error without persistence
	err := s.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestServiceStore_ConcurrentRegister(t *testing.T) {
	s := NewServiceStore()
	const numGoroutines = 100

	var wg sync.WaitGroup

	// Concurrent registrations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			service := Service{
				Name:    fmt.Sprintf("service-%d", id),
				Address: fmt.Sprintf("10.0.0.%d", id%255),
				Port:    8000 + id,
			}
			_ = s.Register(service)
		}(i)
	}
	wg.Wait()

	// Verify all services registered
	services := s.List()
	if len(services) != numGoroutines {
		t.Errorf("expected %d services, got %d", numGoroutines, len(services))
	}
}

func TestServiceStore_ConcurrentHeartbeat(t *testing.T) {
	s := NewServiceStore()
	const numServices = 50

	// Register services
	for i := 0; i < numServices; i++ {
		service := Service{
			Name:    fmt.Sprintf("service-%d", i),
			Address: fmt.Sprintf("10.0.0.%d", i%255),
			Port:    8000 + i,
		}
		if err := s.Register(service); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	var wg sync.WaitGroup

	// Concurrent heartbeats
	wg.Add(numServices)
	for i := 0; i < numServices; i++ {
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("service-%d", id)
			for j := 0; j < 10; j++ {
				success := s.Heartbeat(name)
				if !success {
					t.Errorf("heartbeat failed for %s", name)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	// All services should still be available
	services := s.List()
	if len(services) != numServices {
		t.Errorf("expected %d services after heartbeats, got %d", numServices, len(services))
	}
}

func TestServiceStore_ConcurrentMixedOperations(t *testing.T) {
	s := NewServiceStore()
	const numGoroutines = 50

	var wg sync.WaitGroup

	// Mixed operations: register, get, heartbeat, deregister
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("service-%d", id)
			service := Service{
				Name:    name,
				Address: fmt.Sprintf("10.0.0.%d", id%255),
				Port:    8000 + id,
			}

			// Register
			_ = s.Register(service)

			// Get
			got, ok := s.Get(name)
			if !ok || got.Name != name {
				t.Errorf("Get failed for %s", name)
				return
			}

			// Heartbeat
			if !s.Heartbeat(name) {
				t.Errorf("Heartbeat failed for %s", name)
				return
			}

			// Deregister half of them
			if id%2 == 0 {
				s.Deregister(name)
			}
		}(i)
	}
	wg.Wait()

	// Should have roughly half the services remaining
	services := s.List()
	expectedCount := numGoroutines / 2
	// Allow some tolerance
	if len(services) < expectedCount-5 || len(services) > expectedCount+5 {
		t.Errorf("expected around %d services, got %d", expectedCount, len(services))
	}
}

func TestServiceStore_RegisterOverwrite(t *testing.T) {
	s := NewServiceStore()

	// Register service
	service1 := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	if err := s.Register(service1); err != nil {
		t.Fatalf("Register service1 failed: %v", err)
	}

	// Register again with different address
	service2 := Service{Name: "auth", Address: "10.0.0.2", Port: 9090}
	if err := s.Register(service2); err != nil {
		t.Fatalf("Register service2 failed: %v", err)
	}

	// Should get the updated service
	got, ok := s.Get("auth")
	if !ok {
		t.Fatal("expected service to exist")
	}

	if got.Address != "10.0.0.2" || got.Port != 9090 {
		t.Errorf("expected updated service, got %+v", got)
	}
}

func TestServiceStore_ListEmptyStore(t *testing.T) {
	s := NewServiceStore()

	services := s.List()
	if len(services) != 0 {
		t.Errorf("expected empty list, got %d services", len(services))
	}

	allEntries := s.ListAll()
	if len(allEntries) != 0 {
		t.Errorf("expected empty list, got %d entries", len(allEntries))
	}
}

func TestServiceStore_MultipleDeregister(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}

	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Deregister multiple times should not error
	s.Deregister("auth")
	s.Deregister("auth")
	s.Deregister("auth")

	// Service should not exist
	_, ok := s.Get("auth")
	if ok {
		t.Error("expected service to not exist")
	}
}

func TestServiceStore_RegisterWithHealthChecks(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Register service with health check
	service := Service{
		Name:    "web",
		Address: "10.0.0.1",
		Port:    8080,
		Checks: []*healthcheck.CheckDefinition{
			{
				Name:     "http-check",
				HTTP:     "http://10.0.0.1:8080/health",
				Interval: "10s",
				Timeout:  "5s",
			},
		},
	}

	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify service is registered
	got, ok := s.Get("web")
	if !ok {
		t.Fatal("expected service to be registered")
	}
	if got.Name != "web" {
		t.Errorf("expected service name 'web', got '%s'", got.Name)
	}

	// Verify health check was added
	checks := s.GetHealthChecks("web")
	if len(checks) == 0 {
		t.Error("expected health checks to be registered")
	}
}

func TestServiceStore_GetHealthChecks(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Register service with multiple health checks
	service := Service{
		Name:    "api",
		Address: "10.0.0.1",
		Port:    8080,
		Checks: []*healthcheck.CheckDefinition{
			{
				Name:     "http-check",
				HTTP:     "http://10.0.0.1:8080/health",
				Interval: "10s",
			},
			{
				Name:     "tcp-check",
				TCP:      "10.0.0.1:8080",
				Interval: "30s",
			},
		},
	}

	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get health checks for the service
	checks := s.GetHealthChecks("api")
	if len(checks) != 2 {
		t.Errorf("expected 2 health checks, got %d", len(checks))
	}

	// Verify health checks belong to the service
	for _, check := range checks {
		if check.ServiceID != "api" {
			t.Errorf("expected check to belong to service 'api', got '%s'", check.ServiceID)
		}
	}
}

func TestServiceStore_GetHealthChecksNonExistent(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Get health checks for non-existent service
	checks := s.GetHealthChecks("nonexistent")
	if len(checks) != 0 {
		t.Errorf("expected 0 health checks for non-existent service, got %d", len(checks))
	}
}

func TestServiceStore_GetAllHealthChecks(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Register multiple services with health checks
	service1 := Service{
		Name:    "api",
		Address: "10.0.0.1",
		Port:    8080,
		Checks: []*healthcheck.CheckDefinition{
			{
				HTTP:     "http://10.0.0.1:8080/health",
				Interval: "10s",
			},
		},
	}

	service2 := Service{
		Name:    "db",
		Address: "10.0.0.2",
		Port:    5432,
		Checks: []*healthcheck.CheckDefinition{
			{
				TCP:      "10.0.0.2:5432",
				Interval: "30s",
			},
		},
	}

	if err := s.Register(service1); err != nil {
		t.Fatalf("Register service1 failed: %v", err)
	}
	if err := s.Register(service2); err != nil {
		t.Fatalf("Register service2 failed: %v", err)
	}

	// Get all health checks
	allChecks := s.GetAllHealthChecks()
	if len(allChecks) != 2 {
		t.Errorf("expected 2 total health checks, got %d", len(allChecks))
	}
}

func TestServiceStore_UpdateTTLCheck(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Register service with TTL check
	service := Service{
		Name:    "worker",
		Address: "10.0.0.1",
		Port:    8080,
		Checks: []*healthcheck.CheckDefinition{
			{
				ID:   "worker-ttl",
				Name: "worker-health",
				TTL:  "60s",
			},
		},
	}

	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Update the TTL check
	err := s.UpdateTTLCheck("worker-ttl")
	if err != nil {
		t.Fatalf("UpdateTTLCheck failed: %v", err)
	}

	// Verify check status changed
	checks := s.GetHealthChecks("worker")
	if len(checks) == 0 {
		t.Fatal("expected health check to exist")
	}

	// The check should be passing after update
	if checks[0].Status != healthcheck.StatusPassing {
		t.Errorf("expected status Passing, got %s", checks[0].Status)
	}
}

func TestServiceStore_UpdateTTLCheckNonExistent(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Update non-existent check
	err := s.UpdateTTLCheck("nonexistent")
	if err == nil {
		t.Error("expected error when updating non-existent check")
	}
}

func TestServiceStore_HealthCheckDefaultName(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Register service with health check without name
	service := Service{
		Name:    "api",
		Address: "10.0.0.1",
		Port:    8080,
		Checks: []*healthcheck.CheckDefinition{
			{
				HTTP:     "http://10.0.0.1:8080/health",
				Interval: "10s",
			},
		},
	}

	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Health check should have default name
	checks := s.GetHealthChecks("api")
	if len(checks) == 0 {
		t.Fatal("expected health check to be registered")
	}

	if checks[0].Name != "api-health" {
		t.Errorf("expected default name 'api-health', got '%s'", checks[0].Name)
	}
}

func TestServiceStore_HealthCheckServiceID(t *testing.T) {
	s := NewServiceStore()
	defer s.Close()

	// Register service with health check without explicit ServiceID
	service := Service{
		Name:    "backend",
		Address: "10.0.0.1",
		Port:    8080,
		Checks: []*healthcheck.CheckDefinition{
			{
				Name:     "custom-check",
				HTTP:     "http://10.0.0.1:8080/health",
				Interval: "10s",
			},
		},
	}

	if err := s.Register(service); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Health check should have service ID set
	checks := s.GetHealthChecks("backend")
	if len(checks) == 0 {
		t.Fatal("expected health check to be registered")
	}

	if checks[0].ServiceID != "backend" {
		t.Errorf("expected ServiceID 'backend', got '%s'", checks[0].ServiceID)
	}
}
