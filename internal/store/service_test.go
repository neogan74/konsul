package store

import (
	"testing"
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
