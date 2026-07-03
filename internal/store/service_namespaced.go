package store

import "strings"

// NamespacedService wraps a ServiceStore and scopes all operations to a single namespace.
// Service names are stored internally as "ns:<namespace>:<name>"; the prefix is stripped
// from all returned values so callers see bare names.
type NamespacedService struct {
	store     *ServiceStore
	namespace string
	prefix    string // "ns:<namespace>:"
}

// NewNamespacedService returns a NamespacedService for the given namespace.
// Panics if namespace is empty.
func NewNamespacedService(store *ServiceStore, namespace string) *NamespacedService {
	if namespace == "" {
		panic("NamespacedService: empty namespace")
	}
	return &NamespacedService{
		store:     store,
		namespace: namespace,
		prefix:    "ns:" + namespace + ":",
	}
}

func (n *NamespacedService) nsName(name string) string { return n.prefix + name }

func (n *NamespacedService) stripPrefix(name string) string {
	return strings.TrimPrefix(name, n.prefix)
}

// prepareService returns a copy of svc with the name prefixed and Namespace set.
func (n *NamespacedService) prepareService(svc Service) Service {
	cp := svc
	cp.Namespace = n.namespace
	cp.Name = n.nsName(svc.Name)
	return cp
}

// fixService returns a copy with the namespace prefix stripped from Name.
func (n *NamespacedService) fixService(svc Service) Service {
	cp := svc
	cp.Name = n.stripPrefix(svc.Name)
	return cp
}

// Register registers a service scoped to this namespace.
func (n *NamespacedService) Register(svc Service) error {
	return n.store.Register(n.prepareService(svc))
}

// RegisterCAS registers a service with CAS scoped to this namespace.
func (n *NamespacedService) RegisterCAS(svc Service, expectedIndex uint64) (uint64, error) {
	return n.store.RegisterCAS(n.prepareService(svc), expectedIndex)
}

// Deregister removes a service by name from this namespace.
func (n *NamespacedService) Deregister(name string) {
	n.store.Deregister(n.nsName(name))
}

// DeregisterCAS removes a service with CAS from this namespace.
func (n *NamespacedService) DeregisterCAS(name string, expectedIndex uint64) error {
	return n.store.DeregisterCAS(n.nsName(name), expectedIndex)
}

// Get returns a service by name from this namespace.
func (n *NamespacedService) Get(name string) (Service, bool) {
	svc, ok := n.store.Get(n.nsName(name))
	if !ok {
		return Service{}, false
	}
	return n.fixService(svc), true
}

// GetEntry returns the full ServiceEntry by name from this namespace.
func (n *NamespacedService) GetEntry(name string) (ServiceEntry, bool) {
	entry, ok := n.store.GetEntry(n.nsName(name))
	if !ok {
		return ServiceEntry{}, false
	}
	entry.Service = n.fixService(entry.Service)
	return entry, true
}

// List returns all live services in this namespace.
func (n *NamespacedService) List() []Service {
	all := n.store.ListAll()
	out := make([]Service, 0)
	for _, entry := range all {
		if strings.HasPrefix(entry.Service.Name, n.prefix) {
			out = append(out, n.fixService(entry.Service))
		}
	}
	return out
}

// ListAll returns all service entries in this namespace.
func (n *NamespacedService) ListAll() []ServiceEntry {
	all := n.store.ListAll()
	out := make([]ServiceEntry, 0)
	for _, entry := range all {
		if strings.HasPrefix(entry.Service.Name, n.prefix) {
			entry.Service = n.fixService(entry.Service)
			out = append(out, entry)
		}
	}
	return out
}

// Heartbeat refreshes a service TTL in this namespace.
func (n *NamespacedService) Heartbeat(name string) bool {
	return n.store.Heartbeat(n.nsName(name))
}

// UpdateTTLCheck updates a TTL health check (check ID is global, not namespaced).
func (n *NamespacedService) UpdateTTLCheck(checkID string) error {
	return n.store.UpdateTTLCheck(checkID)
}

// --- Local (FSM) variants ---

// RegisterLocal registers a service snapshot scoped to this namespace.
func (n *NamespacedService) RegisterLocal(data ServiceDataSnapshot) error {
	cp := data
	cp.Namespace = n.namespace
	cp.Name = n.nsName(data.Name)
	return n.store.RegisterLocal(cp)
}

// RegisterCASLocal registers a service snapshot with CAS scoped to this namespace.
func (n *NamespacedService) RegisterCASLocal(data ServiceDataSnapshot, expectedIndex uint64) (uint64, error) {
	cp := data
	cp.Namespace = n.namespace
	cp.Name = n.nsName(data.Name)
	return n.store.RegisterCASLocal(cp, expectedIndex)
}

// DeregisterLocal removes a service by name from this namespace (FSM path).
func (n *NamespacedService) DeregisterLocal(name string) {
	n.store.DeregisterLocal(n.nsName(name))
}

// DeregisterCASLocal removes a service with CAS from this namespace (FSM path).
func (n *NamespacedService) DeregisterCASLocal(name string, expectedIndex uint64) error {
	return n.store.DeregisterCASLocal(n.nsName(name), expectedIndex)
}

// HeartbeatLocal refreshes a service TTL in this namespace (FSM path).
func (n *NamespacedService) HeartbeatLocal(name string) bool {
	return n.store.HeartbeatLocal(n.nsName(name))
}
