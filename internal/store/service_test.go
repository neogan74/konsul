package store

import (
	"testing"
	"time"
)

func TestServiceStore_RegisterAndGet(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}
	s.Register(service)
	got, ok := s.Get("auth")
	if !ok {
		t.Fatalf("expected service to be registered")
	}
	if got != service {
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
	s.Register(service)
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
	s.Register(service)
	if !s.Heartbeat("auth") {
		t.Errorf("expected heartbeat to succeed for registered service")
	}

	// Verify service is still accessible after heartbeat
	got, ok := s.Get("auth")
	if !ok {
		t.Fatalf("expected service to be available after heartbeat")
	}
	if got != service {
		t.Errorf("got %+v, want %+v", got, service)
	}
}

func TestServiceStore_TTLExpiration(t *testing.T) {
	s := NewServiceStore()
	service := Service{Name: "auth", Address: "10.0.0.1", Port: 8080}

	// Register service
	s.Register(service)

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
	s.Register(service1)
	s.Register(service2)

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
	if services[0] != service2 {
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
	s.Register(service)

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
