package namespace

import (
	"fmt"
	"sync"
	"time"
)

// MemoryNamespaceStore is an in-memory, thread-safe NamespaceStore.
type MemoryNamespaceStore struct {
	mu   sync.RWMutex
	data map[string]*Namespace
}

// NewMemoryStore returns a MemoryNamespaceStore pre-seeded with the default namespace.
func NewMemoryStore() *MemoryNamespaceStore {
	s := &MemoryNamespaceStore{data: make(map[string]*Namespace)}
	now := time.Now().UTC()
	s.data[DefaultNamespace] = &Namespace{
		Name:      DefaultNamespace,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s
}

func (s *MemoryNamespaceStore) Create(ns *Namespace) error {
	if err := ValidateName(ns.Name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[ns.Name]; ok {
		return fmt.Errorf("%w: %s", ErrNamespaceExists, ns.Name)
	}
	now := time.Now().UTC()
	cp := *ns
	cp.CreatedAt = now
	cp.UpdatedAt = now
	s.data[ns.Name] = &cp
	return nil
}

func (s *MemoryNamespaceStore) Get(name string) (*Namespace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.data[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNamespaceNotFound, name)
	}
	cp := *ns
	return &cp, nil
}

func (s *MemoryNamespaceStore) List() ([]*Namespace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Namespace, 0, len(s.data))
	for _, ns := range s.data {
		cp := *ns
		out = append(out, &cp)
	}
	return out, nil
}

func (s *MemoryNamespaceStore) Delete(name string) error {
	if IsProtected(name) {
		return fmt.Errorf("%w: %s", ErrNamespaceReserved, name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[name]; !ok {
		return fmt.Errorf("%w: %s", ErrNamespaceNotFound, name)
	}
	delete(s.data, name)
	return nil
}

func (s *MemoryNamespaceStore) Exists(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[name]
	return ok, nil
}
