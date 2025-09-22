package store

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/persistence"
)

type Service struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type ServiceEntry struct {
	Service   Service   `json:"service"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ServiceStore struct {
	Data   map[string]ServiceEntry
	Mutex  sync.RWMutex
	TTL    time.Duration
	engine persistence.Engine
	log    logger.Logger
}

func NewServiceStore() *ServiceStore {
	return &ServiceStore{
		Data: make(map[string]ServiceEntry),
		TTL:  30 * time.Second, // default TTL
		log:  logger.GetDefault(),
	}
}

func NewServiceStoreWithTTL(ttl time.Duration) *ServiceStore {
	return &ServiceStore{
		Data: make(map[string]ServiceEntry),
		TTL:  ttl,
		log:  logger.GetDefault(),
	}
}

// NewServiceStoreWithPersistence creates a service store with persistence engine
func NewServiceStoreWithPersistence(ttl time.Duration, engine persistence.Engine, log logger.Logger) (*ServiceStore, error) {
	store := &ServiceStore{
		Data:   make(map[string]ServiceEntry),
		TTL:    ttl,
		engine: engine,
		log:    log,
	}

	// Load existing data from persistence if available
	if engine != nil {
		if err := store.loadFromPersistence(); err != nil {
			log.Warn("Failed to load service data from persistence", logger.Error(err))
		}
	}

	return store, nil
}

func (s *ServiceStore) loadFromPersistence() error {
	if s.engine == nil {
		return nil
	}

	services, err := s.engine.ListServices()
	if err != nil {
		return err
	}

	loaded := 0
	for _, name := range services {
		data, err := s.engine.GetService(name)
		if err != nil {
			s.log.Warn("Failed to load service from persistence",
				logger.String("service", name),
				logger.Error(err))
			continue
		}

		var entry ServiceEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			s.log.Warn("Failed to unmarshal service data",
				logger.String("service", name),
				logger.Error(err))
			continue
		}

		// Only load non-expired services
		if entry.ExpiresAt.After(time.Now()) {
			s.Data[name] = entry
			loaded++
		}
	}

	s.log.Info("Loaded service data from persistence",
		logger.Int("services", loaded))
	return nil
}

func (s *ServiceStore) Register(service Service) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	entry := ServiceEntry{
		Service:   service,
		ExpiresAt: time.Now().Add(s.TTL),
	}
	s.Data[service.Name] = entry

	// Persist to storage if engine is available
	if s.engine != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			s.log.Error("Failed to marshal service entry",
				logger.String("service", service.Name),
				logger.Error(err))
			return
		}

		if err := s.engine.SetService(service.Name, data, s.TTL); err != nil {
			s.log.Error("Failed to persist service",
				logger.String("service", service.Name),
				logger.Error(err))
		}
	}
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

func (s *ServiceStore) ListAll() []ServiceEntry {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	entries := make([]ServiceEntry, 0, len(s.Data))
	for _, entry := range s.Data {
		entries = append(entries, entry)
	}
	return entries
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

	// Update in persistence if engine is available
	if s.engine != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			s.log.Error("Failed to marshal service entry",
				logger.String("service", name),
				logger.Error(err))
			return true
		}

		if err := s.engine.SetService(name, data, s.TTL); err != nil {
			s.log.Error("Failed to update service heartbeat in persistence",
				logger.String("service", name),
				logger.Error(err))
		}
	}

	return true
}

func (s *ServiceStore) Deregister(name string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	delete(s.Data, name)

	// Delete from persistence if engine is available
	if s.engine != nil {
		if err := s.engine.DeleteService(name); err != nil {
			s.log.Error("Failed to delete service from persistence",
				logger.String("service", name),
				logger.Error(err))
		}
	}
}

func (s *ServiceStore) CleanupExpired() int {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	now := time.Now()
	count := 0
	expiredServices := make([]string, 0)

	for name, entry := range s.Data {
		if entry.ExpiresAt.Before(now) {
			delete(s.Data, name)
			expiredServices = append(expiredServices, name)
			count++
		}
	}

	// Delete expired services from persistence
	if s.engine != nil && len(expiredServices) > 0 {
		for _, name := range expiredServices {
			if err := s.engine.DeleteService(name); err != nil {
				s.log.Error("Failed to delete expired service from persistence",
					logger.String("service", name),
					logger.Error(err))
			}
		}
	}

	return count
}

// Close closes the persistence engine
func (s *ServiceStore) Close() error {
	if s.engine != nil {
		return s.engine.Close()
	}
	return nil
}