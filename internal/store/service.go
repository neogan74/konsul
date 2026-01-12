package store

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neogan74/konsul/internal/healthcheck"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/persistence"
)

type Service struct {
	Name    string                         `json:"name"`
	Address string                         `json:"address"`
	Port    int                            `json:"port"`
	Tags    []string                       `json:"tags,omitempty"` // Service tags for filtering and categorization
	Meta    map[string]string              `json:"meta,omitempty"` // Service metadata (key-value pairs)
	Checks  []*healthcheck.CheckDefinition `json:"checks,omitempty"`
}

type ServiceEntry struct {
	Service     Service   `json:"service"`
	ExpiresAt   time.Time `json:"expires_at"`
	ModifyIndex uint64    `json:"modify_index"`
	CreateIndex uint64    `json:"create_index"`
}

type ServiceStore struct {
	Data          map[string]ServiceEntry
	TagIndex      map[string]map[string]bool     // Tag → {ServiceName: true} - for fast tag queries
	MetaIndex     map[string]map[string][]string // MetaKey → {MetaValue: [ServiceNames]} - for fast metadata queries
	Mutex         sync.RWMutex
	globalIndex   uint64 // Monotonically increasing global index
	TTL           time.Duration
	engine        persistence.Engine
	log           logger.Logger
	healthManager *healthcheck.Manager
}

func NewServiceStore() *ServiceStore {
	return &ServiceStore{
		Data:          make(map[string]ServiceEntry),
		TagIndex:      make(map[string]map[string]bool),
		MetaIndex:     make(map[string]map[string][]string),
		globalIndex:   0,
		TTL:           30 * time.Second, // default TTL
		log:           logger.GetDefault(),
		healthManager: healthcheck.NewManager(logger.GetDefault()),
	}
}

func NewServiceStoreWithTTL(ttl time.Duration) *ServiceStore {
	return &ServiceStore{
		Data:          make(map[string]ServiceEntry),
		TagIndex:      make(map[string]map[string]bool),
		MetaIndex:     make(map[string]map[string][]string),
		globalIndex:   0,
		TTL:           ttl,
		log:           logger.GetDefault(),
		healthManager: healthcheck.NewManager(logger.GetDefault()),
	}
}

// NewServiceStoreWithPersistence creates a service store with persistence engine
func NewServiceStoreWithPersistence(ttl time.Duration, engine persistence.Engine, log logger.Logger) (*ServiceStore, error) {
	store := &ServiceStore{
		Data:          make(map[string]ServiceEntry),
		TagIndex:      make(map[string]map[string]bool),
		MetaIndex:     make(map[string]map[string][]string),
		globalIndex:   0,
		TTL:           ttl,
		engine:        engine,
		log:           log,
		healthManager: healthcheck.NewManager(log),
	}

	// Load existing data from persistence if available
	if engine != nil {
		if err := store.loadFromPersistence(); err != nil {
			log.Warn("Failed to load service data from persistence", logger.Error(err))
		}
	}

	return store, nil
}

// nextIndex atomically increments and returns the next global index
func (s *ServiceStore) nextIndex() uint64 {
	return atomic.AddUint64(&s.globalIndex, 1)
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
	var maxIndex uint64
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

		// Migrate old entries without indices
		if entry.ModifyIndex == 0 {
			entry.ModifyIndex = 1
			entry.CreateIndex = 1
		}

		// Only load non-expired services
		if entry.ExpiresAt.After(time.Now()) {
			s.Data[name] = entry
			// Rebuild indexes for loaded services
			s.addToTagIndex(name, entry.Service.Tags)
			s.addToMetaIndex(name, entry.Service.Meta)
			if entry.ModifyIndex > maxIndex {
				maxIndex = entry.ModifyIndex
			}
			loaded++
		}
	}

	// Set global index to max found index
	s.globalIndex = maxIndex

	s.log.Info("Loaded service data from persistence",
		logger.Int("services", loaded),
		logger.String("max_index", fmt.Sprintf("%d", maxIndex)))
	return nil
}

func (s *ServiceStore) Register(service Service) error {
	// Validate service including tags and metadata
	if err := ValidateService(&service); err != nil {
		s.log.Error("Service validation failed",
			logger.String("service", service.Name),
			logger.Error(err))
		return err
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	// Remove old indexes if service exists (for re-registration)
	oldEntry, existed := s.Data[service.Name]
	if existed {
		s.removeFromTagIndex(service.Name, oldEntry.Service.Tags)
		s.removeFromMetaIndex(service.Name, oldEntry.Service.Meta)
	}

	newIndex := s.nextIndex()
	entry := ServiceEntry{
		Service:     service,
		ExpiresAt:   time.Now().Add(s.TTL),
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
	} else {
		entry.CreateIndex = newIndex
	}
	s.Data[service.Name] = entry

	// Add to tag and metadata indexes
	s.addToTagIndex(service.Name, service.Tags)
	s.addToMetaIndex(service.Name, service.Meta)

	// Register health checks
	for _, checkDef := range service.Checks {
		// Set service ID for the check
		if checkDef.ServiceID == "" {
			checkDef.ServiceID = service.Name
		}

		// Set check name if not provided
		if checkDef.Name == "" {
			checkDef.Name = fmt.Sprintf("%s-health", service.Name)
		}

		_, err := s.healthManager.AddCheck(checkDef)
		if err != nil {
			s.log.Error("Failed to add health check",
				logger.String("service", service.Name),
				logger.String("check", checkDef.Name),
				logger.Error(err))
		}
	}

	// Persist to storage if engine is available
	if s.engine != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			s.log.Error("Failed to marshal service entry",
				logger.String("service", service.Name),
				logger.Error(err))
			return err
		}

		if err := s.engine.SetService(service.Name, data, s.TTL); err != nil {
			s.log.Error("Failed to persist service",
				logger.String("service", service.Name),
				logger.Error(err))
			return err
		}
	}

	s.log.Info("Service registered with tags/metadata",
		logger.String("service", service.Name),
		logger.Int("tags", len(service.Tags)),
		logger.Int("metadata_keys", len(service.Meta)))

	return nil
}

// RegisterCAS performs a Compare-And-Swap registration operation
// It will only register the service if the current ModifyIndex matches the expected index
// If expectedIndex is 0, it means "create only if not exists"
// Returns the new ModifyIndex on success, or error on conflict
func (s *ServiceStore) RegisterCAS(service Service, expectedIndex uint64) (uint64, error) {
	// Validate service including tags and metadata
	if err := ValidateService(&service); err != nil {
		s.log.Error("Service validation failed",
			logger.String("service", service.Name),
			logger.Error(err))
		return 0, err
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	oldEntry, existed := s.Data[service.Name]

	// Check CAS condition
	if expectedIndex == 0 {
		// Create only if not exists
		if existed {
			return 0, &CASConflictError{
				Key:           service.Name,
				ExpectedIndex: 0,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "service",
			}
		}
	} else {
		// Update only if index matches
		if !existed {
			return 0, &NotFoundError{Type: "service", Key: service.Name}
		}
		if oldEntry.ModifyIndex != expectedIndex {
			return 0, &CASConflictError{
				Key:           service.Name,
				ExpectedIndex: expectedIndex,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "service",
			}
		}
	}

	// Remove old indexes if service exists
	if existed {
		s.removeFromTagIndex(service.Name, oldEntry.Service.Tags)
		s.removeFromMetaIndex(service.Name, oldEntry.Service.Meta)
	}

	newIndex := s.nextIndex()
	entry := ServiceEntry{
		Service:     service,
		ExpiresAt:   time.Now().Add(s.TTL),
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
	} else {
		entry.CreateIndex = newIndex
	}
	s.Data[service.Name] = entry

	// Add to tag and metadata indexes
	s.addToTagIndex(service.Name, service.Tags)
	s.addToMetaIndex(service.Name, service.Meta)

	// Register health checks
	for _, checkDef := range service.Checks {
		// Set service ID for the check
		if checkDef.ServiceID == "" {
			checkDef.ServiceID = service.Name
		}

		// Set check name if not provided
		if checkDef.Name == "" {
			checkDef.Name = fmt.Sprintf("%s-health", service.Name)
		}

		_, err := s.healthManager.AddCheck(checkDef)
		if err != nil {
			s.log.Error("Failed to add health check",
				logger.String("service", service.Name),
				logger.String("check", checkDef.Name),
				logger.Error(err))
		}
	}

	// Persist to storage if engine is available
	if s.engine != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			s.log.Error("Failed to marshal service entry",
				logger.String("service", service.Name),
				logger.Error(err))
			return newIndex, err
		}

		if err := s.engine.SetService(service.Name, data, s.TTL); err != nil {
			s.log.Error("Failed to persist service",
				logger.String("service", service.Name),
				logger.Error(err))
			return newIndex, err
		}
	}

	s.log.Info("Service registered with CAS",
		logger.String("service", service.Name),
		logger.String("new_index", fmt.Sprintf("%d", newIndex)))

	return newIndex, nil
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

// GetEntry returns the full ServiceEntry with version information
func (s *ServiceStore) GetEntry(name string) (ServiceEntry, bool) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	entry, ok := s.Data[name]
	if !ok || entry.ExpiresAt.Before(time.Now()) {
		return ServiceEntry{}, false
	}
	return entry, true
}

func (s *ServiceStore) Heartbeat(name string) bool {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	entry, ok := s.Data[name]
	if !ok {
		return false
	}

	// Update TTL but preserve indices
	entry.ExpiresAt = time.Now().Add(s.TTL)
	// Heartbeat is not a modification, so don't update ModifyIndex
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

	// Remove from indexes before deleting
	if entry, exists := s.Data[name]; exists {
		s.removeFromTagIndex(name, entry.Service.Tags)
		s.removeFromMetaIndex(name, entry.Service.Meta)
	}

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

// DeregisterCAS performs a Compare-And-Swap deregistration operation
// It will only deregister the service if the current ModifyIndex matches the expected index
// Returns error on conflict
func (s *ServiceStore) DeregisterCAS(name string, expectedIndex uint64) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	entry, existed := s.Data[name]
	if !existed {
		return &NotFoundError{Type: "service", Key: name}
	}

	if entry.ModifyIndex != expectedIndex {
		return &CASConflictError{
			Key:           name,
			ExpectedIndex: expectedIndex,
			CurrentIndex:  entry.ModifyIndex,
			OperationType: "service",
		}
	}

	// Remove from indexes before deleting
	s.removeFromTagIndex(name, entry.Service.Tags)
	s.removeFromMetaIndex(name, entry.Service.Meta)

	delete(s.Data, name)

	// Delete from persistence if engine is available
	if s.engine != nil {
		if err := s.engine.DeleteService(name); err != nil {
			s.log.Error("Failed to delete service from persistence",
				logger.String("service", name),
				logger.Error(err))
			return err
		}
	}

	s.log.Info("Service deregistered with CAS",
		logger.String("service", name),
		logger.String("index", fmt.Sprintf("%d", expectedIndex)))

	return nil
}

func (s *ServiceStore) CleanupExpired() int {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	now := time.Now()
	count := 0
	expiredServices := make([]string, 0)

	for name, entry := range s.Data {
		if entry.ExpiresAt.Before(now) {
			// Remove from indexes before deleting
			s.removeFromTagIndex(name, entry.Service.Tags)
			s.removeFromMetaIndex(name, entry.Service.Meta)

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

// GetHealthChecks returns all health checks for a service
func (s *ServiceStore) GetHealthChecks(serviceName string) []*healthcheck.Check {
	checks := s.healthManager.ListChecks()
	var serviceChecks []*healthcheck.Check
	for _, check := range checks {
		if check.ServiceID == serviceName {
			serviceChecks = append(serviceChecks, check)
		}
	}
	return serviceChecks
}

// GetAllHealthChecks returns all health checks
func (s *ServiceStore) GetAllHealthChecks() []*healthcheck.Check {
	return s.healthManager.ListChecks()
}

// UpdateTTLCheck updates a TTL-based health check
func (s *ServiceStore) UpdateTTLCheck(checkID string) error {
	return s.healthManager.UpdateTTLCheck(checkID)
}

// Close closes the persistence engine and health manager
func (s *ServiceStore) Close() error {
	s.healthManager.Stop()
	if s.engine != nil {
		return s.engine.Close()
	}
	return nil
}

// =============================================================================
// Raft Integration Methods
// These methods are used by the Raft FSM to apply changes without persistence.
// Raft handles durability through log replication, so we skip the persistence layer.
// =============================================================================

// ServiceDataSnapshot represents service data for Raft commands and snapshots.
// This is decoupled from internal Service struct to avoid circular dependencies.
type ServiceDataSnapshot struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Port    int               `json:"port"`
	Tags    []string          `json:"tags,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// ServiceEntrySnapshot represents service entry data for Raft snapshots.
type ServiceEntrySnapshot struct {
	Service     ServiceDataSnapshot `json:"service"`
	ExpiresAt   time.Time           `json:"expires_at"`
	ModifyIndex uint64              `json:"modify_index"`
	CreateIndex uint64              `json:"create_index"`
}

// RegisterLocal registers a service without persisting to the storage engine.
// This is used by Raft FSM when applying committed log entries.
// Note: Health checks are NOT registered here - they should be handled locally on each node.
func (s *ServiceStore) RegisterLocal(serviceData ServiceDataSnapshot) error {
	// Convert to internal Service type
	service := Service{
		Name:    serviceData.Name,
		Address: serviceData.Address,
		Port:    serviceData.Port,
		Tags:    serviceData.Tags,
		Meta:    serviceData.Meta,
		// Note: Checks are not included - they are local to each node
	}

	// Validate service
	if err := ValidateService(&service); err != nil {
		s.log.Error("Service validation failed",
			logger.String("service", service.Name),
			logger.Error(err))
		return err
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	// Remove old indexes if service exists (for re-registration)
	oldEntry, existed := s.Data[service.Name]
	if existed {
		s.removeFromTagIndex(service.Name, oldEntry.Service.Tags)
		s.removeFromMetaIndex(service.Name, oldEntry.Service.Meta)
	}

	newIndex := s.nextIndex()
	entry := ServiceEntry{
		Service:     service,
		ExpiresAt:   time.Now().Add(s.TTL),
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
	} else {
		entry.CreateIndex = newIndex
	}
	s.Data[service.Name] = entry

	// Add to tag and metadata indexes
	s.addToTagIndex(service.Name, service.Tags)
	s.addToMetaIndex(service.Name, service.Meta)

	s.log.Debug("Service registered via Raft",
		logger.String("service", service.Name),
		logger.Int("tags", len(service.Tags)))

	return nil
}

// DeregisterLocal removes a service without persisting the deletion.
// This is used by Raft FSM when applying committed log entries.
func (s *ServiceStore) DeregisterLocal(name string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	// Remove from indexes before deleting
	if entry, exists := s.Data[name]; exists {
		s.removeFromTagIndex(name, entry.Service.Tags)
		s.removeFromMetaIndex(name, entry.Service.Meta)
	}

	delete(s.Data, name)

	s.log.Debug("Service deregistered via Raft",
		logger.String("service", name))
}

// RegisterCASLocal performs Compare-And-Swap registration without persistence.
func (s *ServiceStore) RegisterCASLocal(serviceData ServiceDataSnapshot, expectedIndex uint64) (uint64, error) {
	// Convert to internal Service type
	service := Service{
		Name:    serviceData.Name,
		Address: serviceData.Address,
		Port:    serviceData.Port,
		Tags:    serviceData.Tags,
		Meta:    serviceData.Meta,
	}

	// Validate service
	if err := ValidateService(&service); err != nil {
		s.log.Error("Service validation failed",
			logger.String("service", service.Name),
			logger.Error(err))
		return 0, err
	}

	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	oldEntry, existed := s.Data[service.Name]

	// Check CAS condition
	if expectedIndex == 0 {
		if existed {
			return 0, &CASConflictError{
				Key:           service.Name,
				ExpectedIndex: 0,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "service",
			}
		}
	} else {
		if !existed {
			return 0, &NotFoundError{Type: "service", Key: service.Name}
		}
		if oldEntry.ModifyIndex != expectedIndex {
			return 0, &CASConflictError{
				Key:           service.Name,
				ExpectedIndex: expectedIndex,
				CurrentIndex:  oldEntry.ModifyIndex,
				OperationType: "service",
			}
		}
	}

	// Remove old indexes if service exists
	if existed {
		s.removeFromTagIndex(service.Name, oldEntry.Service.Tags)
		s.removeFromMetaIndex(service.Name, oldEntry.Service.Meta)
	}

	newIndex := s.nextIndex()
	entry := ServiceEntry{
		Service:     service,
		ExpiresAt:   time.Now().Add(s.TTL),
		ModifyIndex: newIndex,
	}
	if existed {
		entry.CreateIndex = oldEntry.CreateIndex
	} else {
		entry.CreateIndex = newIndex
	}
	s.Data[service.Name] = entry

	// Add to tag and metadata indexes
	s.addToTagIndex(service.Name, service.Tags)
	s.addToMetaIndex(service.Name, service.Meta)

	s.log.Debug("Service registered via Raft CAS",
		logger.String("service", service.Name),
		logger.String("new_index", fmt.Sprintf("%d", newIndex)))

	return newIndex, nil
}

// DeregisterCASLocal performs Compare-And-Swap deregistration without persistence.
func (s *ServiceStore) DeregisterCASLocal(name string, expectedIndex uint64) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	entry, existed := s.Data[name]
	if !existed {
		return &NotFoundError{Type: "service", Key: name}
	}

	if entry.ModifyIndex != expectedIndex {
		return &CASConflictError{
			Key:           name,
			ExpectedIndex: expectedIndex,
			CurrentIndex:  entry.ModifyIndex,
			OperationType: "service",
		}
	}

	// Remove from indexes before deleting
	s.removeFromTagIndex(name, entry.Service.Tags)
	s.removeFromMetaIndex(name, entry.Service.Meta)

	delete(s.Data, name)

	s.log.Debug("Service deregistered via Raft CAS",
		logger.String("service", name),
		logger.String("index", fmt.Sprintf("%d", expectedIndex)))

	return nil
}

// GetEntrySnapshot returns a snapshot of a service entry.
func (s *ServiceStore) GetEntrySnapshot(name string) (ServiceEntrySnapshot, bool) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	entry, ok := s.Data[name]
	if !ok || entry.ExpiresAt.Before(time.Now()) {
		return ServiceEntrySnapshot{}, false
	}

	// Deep copy tags
	var tags []string
	if len(entry.Service.Tags) > 0 {
		tags = make([]string, len(entry.Service.Tags))
		copy(tags, entry.Service.Tags)
	}

	// Deep copy meta
	var meta map[string]string
	if len(entry.Service.Meta) > 0 {
		meta = make(map[string]string, len(entry.Service.Meta))
		for k, v := range entry.Service.Meta {
			meta[k] = v
		}
	}

	return ServiceEntrySnapshot{
		Service: ServiceDataSnapshot{
			Name:    entry.Service.Name,
			Address: entry.Service.Address,
			Port:    entry.Service.Port,
			Tags:    tags,
			Meta:    meta,
		},
		ExpiresAt:   entry.ExpiresAt,
		ModifyIndex: entry.ModifyIndex,
		CreateIndex: entry.CreateIndex,
	}, true
}

// HeartbeatLocal updates service TTL without persisting.
// This is used by Raft FSM when applying committed log entries.
func (s *ServiceStore) HeartbeatLocal(name string) bool {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	entry, ok := s.Data[name]
	if !ok {
		return false
	}

	// Update TTL but preserve indices
	entry.ExpiresAt = time.Now().Add(s.TTL)
	s.Data[name] = entry

	return true
}

// GetAllData returns all service data for Raft snapshotting.
// Returns a deep copy to ensure snapshot consistency.
func (s *ServiceStore) GetAllData() map[string]ServiceEntrySnapshot {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	result := make(map[string]ServiceEntrySnapshot, len(s.Data))
	for name, entry := range s.Data {
		// Deep copy tags
		var tags []string
		if len(entry.Service.Tags) > 0 {
			tags = make([]string, len(entry.Service.Tags))
			copy(tags, entry.Service.Tags)
		}

		// Deep copy meta
		var meta map[string]string
		if len(entry.Service.Meta) > 0 {
			meta = make(map[string]string, len(entry.Service.Meta))
			for k, v := range entry.Service.Meta {
				meta[k] = v
			}
		}

		result[name] = ServiceEntrySnapshot{
			Service: ServiceDataSnapshot{
				Name:    entry.Service.Name,
				Address: entry.Service.Address,
				Port:    entry.Service.Port,
				Tags:    tags,
				Meta:    meta,
			},
			ExpiresAt:   entry.ExpiresAt,
			ModifyIndex: entry.ModifyIndex,
			CreateIndex: entry.CreateIndex,
		}
	}
	return result
}

// RestoreFromSnapshot restores service data from a Raft snapshot.
// This replaces all existing data with the snapshot data.
// Note: Health checks are NOT restored - they should be re-registered locally.
func (s *ServiceStore) RestoreFromSnapshot(data map[string]ServiceEntrySnapshot) error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	// Clear existing data and indexes
	s.Data = make(map[string]ServiceEntry, len(data))
	s.TagIndex = make(map[string]map[string]bool)
	s.MetaIndex = make(map[string]map[string][]string)

	// Restore from snapshot
	var maxIndex uint64
	for name, snapshot := range data {
		// Convert snapshot to internal types
		service := Service{
			Name:    snapshot.Service.Name,
			Address: snapshot.Service.Address,
			Port:    snapshot.Service.Port,
			Tags:    snapshot.Service.Tags,
			Meta:    snapshot.Service.Meta,
		}

		entry := ServiceEntry{
			Service:     service,
			ExpiresAt:   snapshot.ExpiresAt,
			ModifyIndex: snapshot.ModifyIndex,
			CreateIndex: snapshot.CreateIndex,
		}

		s.Data[name] = entry

		// Rebuild indexes
		s.addToTagIndex(name, service.Tags)
		s.addToMetaIndex(name, service.Meta)

		if snapshot.ModifyIndex > maxIndex {
			maxIndex = snapshot.ModifyIndex
		}
	}

	// Update global index to max found index
	s.globalIndex = maxIndex

	s.log.Info("Service store restored from Raft snapshot",
		logger.Int("services", len(data)),
		logger.String("max_index", fmt.Sprintf("%d", maxIndex)))

	return nil
}
