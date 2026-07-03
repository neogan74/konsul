package rbac

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/neogan74/konsul/internal/persistence"
)

// RoleStore defines namespace-aware persistence operations for roles.
type RoleStore interface {
	GetRole(namespace, name string) (*Role, error)
	SetRole(namespace string, role *Role) error
	DeleteRole(namespace, name string) error
	ListRoles(namespace string) ([]*Role, error)
}

// AssignmentStore defines namespace-aware persistence operations for role assignments.
type AssignmentStore interface {
	GetAssignment(namespace, subjectID string) (*RoleAssignment, error)
	SetAssignment(namespace string, assignment *RoleAssignment) error
	DeleteAssignment(namespace, subjectID string) error
	ListAssignments(namespace string) ([]*RoleAssignment, error)
	// ListAllAssignments returns every assignment across all namespaces (used for expiry sweeps).
	ListAllAssignments() ([]*RoleAssignment, error)
}

// MemoryRoleStore is an in-memory, namespace-aware implementation of RoleStore.
// Outer map key is namespace; inner map key is role name.
type MemoryRoleStore struct {
	mu    sync.RWMutex
	roles map[string]map[string]*Role // namespace → name → role
}

// NewMemoryRoleStore creates a new in-memory role store.
func NewMemoryRoleStore() *MemoryRoleStore {
	return &MemoryRoleStore{roles: make(map[string]map[string]*Role)}
}

func (s *MemoryRoleStore) GetRole(namespace, name string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ns, ok := s.roles[namespace]; ok {
		if r, ok := ns[name]; ok {
			return r, nil
		}
	}
	return nil, ErrRoleNotFound
}

func (s *MemoryRoleStore) SetRole(namespace string, role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roles[namespace] == nil {
		s.roles[namespace] = make(map[string]*Role)
	}
	role.Namespace = namespace
	s.roles[namespace][role.Name] = role
	return nil
}

func (s *MemoryRoleStore) DeleteRole(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ns, ok := s.roles[namespace]; ok {
		if _, ok := ns[name]; ok {
			delete(ns, name)
			return nil
		}
	}
	return ErrRoleNotFound
}

func (s *MemoryRoleStore) ListRoles(namespace string) ([]*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns := s.roles[namespace]
	out := make([]*Role, 0, len(ns))
	for _, r := range ns {
		out = append(out, r)
	}
	return out, nil
}

// MemoryAssignmentStore is an in-memory, namespace-aware implementation of AssignmentStore.
// Composite key is "namespace:subjectID".
type MemoryAssignmentStore struct {
	mu          sync.RWMutex
	assignments map[string]*RoleAssignment // "namespace:subjectID" → assignment
}

// NewMemoryAssignmentStore creates a new in-memory assignment store.
func NewMemoryAssignmentStore() *MemoryAssignmentStore {
	return &MemoryAssignmentStore{assignments: make(map[string]*RoleAssignment)}
}

func assignKey(namespace, subjectID string) string { return namespace + ":" + subjectID }

func (s *MemoryAssignmentStore) GetAssignment(namespace, subjectID string) (*RoleAssignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.assignments[assignKey(namespace, subjectID)]
	if !ok {
		return nil, ErrAssignmentNotFound
	}
	return a, nil
}

func (s *MemoryAssignmentStore) SetAssignment(namespace string, assignment *RoleAssignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	assignment.Namespace = namespace
	s.assignments[assignKey(namespace, assignment.SubjectID)] = assignment
	return nil
}

func (s *MemoryAssignmentStore) DeleteAssignment(namespace, subjectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := assignKey(namespace, subjectID)
	if _, ok := s.assignments[k]; !ok {
		return ErrAssignmentNotFound
	}
	delete(s.assignments, k)
	return nil
}

func (s *MemoryAssignmentStore) ListAssignments(namespace string) ([]*RoleAssignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := namespace + ":"
	out := make([]*RoleAssignment, 0)
	for k, a := range s.assignments {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, a)
		}
	}
	return out, nil
}

func (s *MemoryAssignmentStore) ListAllAssignments() ([]*RoleAssignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RoleAssignment, 0, len(s.assignments))
	for _, a := range s.assignments {
		out = append(out, a)
	}
	return out, nil
}

// BadgerRoleStore is a BadgerDB-backed implementation of RoleStore.
// Key format: "rbac:ns:<namespace>:role:<name>"
type BadgerRoleStore struct {
	engine persistence.Engine
}

// NewBadgerRoleStore creates a new BadgerDB-backed role store.
func NewBadgerRoleStore(engine persistence.Engine) *BadgerRoleStore {
	return &BadgerRoleStore{engine: engine}
}

func badgerRoleKey(namespace, name string) string {
	return "rbac:ns:" + namespace + ":role:" + name
}

func badgerRolePrefix(namespace string) string {
	return "rbac:ns:" + namespace + ":role:"
}

func (s *BadgerRoleStore) GetRole(namespace, name string) (*Role, error) {
	data, err := s.engine.Get(badgerRoleKey(namespace, name))
	if err != nil {
		return nil, ErrRoleNotFound
	}
	var role Role
	if err := json.Unmarshal(data, &role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role %q: %w", name, err)
	}
	return &role, nil
}

func (s *BadgerRoleStore) SetRole(namespace string, role *Role) error {
	role.Namespace = namespace
	data, err := json.Marshal(role)
	if err != nil {
		return fmt.Errorf("failed to marshal role %q: %w", role.Name, err)
	}
	return s.engine.Set(badgerRoleKey(namespace, role.Name), data)
}

func (s *BadgerRoleStore) DeleteRole(namespace, name string) error {
	return s.engine.Delete(badgerRoleKey(namespace, name))
}

func (s *BadgerRoleStore) ListRoles(namespace string) ([]*Role, error) {
	prefix := badgerRolePrefix(namespace)
	keys, err := s.engine.List(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	roles := make([]*Role, 0, len(keys))
	for _, key := range keys {
		name := strings.TrimPrefix(key, prefix)
		role, err := s.GetRole(namespace, name)
		if err != nil {
			continue
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// BadgerAssignmentStore is a BadgerDB-backed implementation of AssignmentStore.
// Key format: "rbac:ns:<namespace>:assign:<subjectID>"
type BadgerAssignmentStore struct {
	engine persistence.Engine
}

// NewBadgerAssignmentStore creates a new BadgerDB-backed assignment store.
func NewBadgerAssignmentStore(engine persistence.Engine) *BadgerAssignmentStore {
	return &BadgerAssignmentStore{engine: engine}
}

func badgerAssignKey(namespace, subjectID string) string {
	return "rbac:ns:" + namespace + ":assign:" + subjectID
}

func badgerAssignPrefix(namespace string) string {
	return "rbac:ns:" + namespace + ":assign:"
}

func (s *BadgerAssignmentStore) GetAssignment(namespace, subjectID string) (*RoleAssignment, error) {
	data, err := s.engine.Get(badgerAssignKey(namespace, subjectID))
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	var a RoleAssignment
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("failed to unmarshal assignment for %q: %w", subjectID, err)
	}
	return &a, nil
}

func (s *BadgerAssignmentStore) SetAssignment(namespace string, assignment *RoleAssignment) error {
	assignment.Namespace = namespace
	data, err := json.Marshal(assignment)
	if err != nil {
		return fmt.Errorf("failed to marshal assignment for %q: %w", assignment.SubjectID, err)
	}
	return s.engine.Set(badgerAssignKey(namespace, assignment.SubjectID), data)
}

func (s *BadgerAssignmentStore) DeleteAssignment(namespace, subjectID string) error {
	return s.engine.Delete(badgerAssignKey(namespace, subjectID))
}

func (s *BadgerAssignmentStore) ListAssignments(namespace string) ([]*RoleAssignment, error) {
	prefix := badgerAssignPrefix(namespace)
	keys, err := s.engine.List(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	assignments := make([]*RoleAssignment, 0, len(keys))
	for _, key := range keys {
		subjectID := strings.TrimPrefix(key, prefix)
		a, err := s.GetAssignment(namespace, subjectID)
		if err != nil {
			continue
		}
		assignments = append(assignments, a)
	}
	return assignments, nil
}

func (s *BadgerAssignmentStore) ListAllAssignments() ([]*RoleAssignment, error) {
	keys, err := s.engine.List("rbac:ns:")
	if err != nil {
		return nil, fmt.Errorf("failed to list all assignments: %w", err)
	}
	assignments := make([]*RoleAssignment, 0)
	for _, key := range keys {
		if !strings.Contains(key, ":assign:") {
			continue
		}
		data, err := s.engine.Get(key)
		if err != nil {
			continue
		}
		var a RoleAssignment
		if err := json.Unmarshal(data, &a); err != nil {
			continue
		}
		assignments = append(assignments, &a)
	}
	return assignments, nil
}
