package store

import (
	"sync"
	"time"
)

type Service struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type ServiceEntry struct {
	Service   Service
	ExpiresAt time.Time
}

type ServiceStore struct {
	Data  map[string]ServiceEntry
	Mutex sync.RWMutex
	TTL   time.Duration
}

func NewServiceStore() *ServiceStore {
	return &ServiceStore{
		Data: make(map[string]ServiceEntry),
		TTL:  30 * time.Second, // default TTL
	}
}

func NewServiceStoreWithTTL(ttl time.Duration) *ServiceStore {
	return &ServiceStore{
		Data: make(map[string]ServiceEntry),
		TTL:  ttl,
	}
}

func (s *ServiceStore) Register(service Service) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	entry := ServiceEntry{
		Service:   service,
		ExpiresAt: time.Now().Add(s.TTL),
	}
	s.Data[service.Name] = entry
}

func (s *ServiceStore) List() []Service {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	services := make([]Service, 0, len(s.Data))
	now := time.Now()
	for _, entry := range s.Data {
		if entry.ExpiresAt.After(now) {
			services = append(services, entry.Service)
		}
	}
	return services
}

func (s *ServiceStore) Get(name string) (Service, bool) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	entry, ok := s.Data[name]
	if !ok || entry.ExpiresAt.Before(time.Now()) {
		return Service{}, false
	}
	return entry.Service, true
}

func (s *ServiceStore) Heartbeat(name string) bool {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	entry, ok := s.Data[name]
	if !ok {
		return false
	}
	entry.ExpiresAt = time.Now().Add(s.TTL)
	s.Data[name] = entry
	return true
}

func (s *ServiceStore) Deregister(name string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	delete(s.Data, name)
}

func (s *ServiceStore) CleanupExpired() int {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	now := time.Now()
	count := 0
	for name, entry := range s.Data {
		if entry.ExpiresAt.Before(now) {
			delete(s.Data, name)
			count++
		}
	}
	return count
} 