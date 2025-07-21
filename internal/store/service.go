package store

import "sync"

type Service struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type ServiceStore struct {
	Data  map[string]Service
	Mutex sync.RWMutex
}

func NewServiceStore() *ServiceStore {
	return &ServiceStore{Data: make(map[string]Service)}
}

func (s *ServiceStore) Register(service Service) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	s.Data[service.Name] = service
}

func (s *ServiceStore) List() []Service {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	services := make([]Service, 0, len(s.Data))
	for _, svc := range s.Data {
		services = append(services, svc)
	}
	return services
}

func (s *ServiceStore) Get(name string) (Service, bool) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()
	svc, ok := s.Data[name]
	return svc, ok
}

func (s *ServiceStore) Deregister(name string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	delete(s.Data, name)
} 