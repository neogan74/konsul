package rbac

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/neogan74/konsul/internal/persistence"
)

const (
	rolePrefix   = "rbac:role:"
	assignPrefix = "rbac:assign:"
)

// RoleStore defines persistence operations for roles.
type RoleStore interface {
	GetRole(name string) (*Role, error)
	SetRole(role *Role) error
	DeleteRole(name string) error
	ListRoles() ([]*Role, error)
}

// AssignmentStore defines persistence operations for role assignments.
type AssignmentStore interface {
	GetAssignment(subjectID string) (*RoleAssignment, error)
	SetAssignment(assignment *RoleAssignment) error
	DeleteAssignment(subjectID string) error
	ListAssignments() ([]*RoleAssignment, error)
}

// MemoryRoleStore is an in-memory implementation of RoleStore.
type MemoryRoleStore struct {
	mu    sync.RWMutex
	roles map[string]*Role
}

// NewMemoryRoleStore creates a new in-memory role store.
func NewMemoryRoleStore() *MemoryRoleStore {
	return &MemoryRoleStore{
		roles: make(map[string]*Role),
	}
}

// GetRole retrieves a role by name.
func (s *MemoryRoleStore) GetRole(name string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	role, ok := s.roles[name]
	if !ok {
		return nil, ErrRoleNotFound
	}
	return role, nil
}

// SetRole creates or updates a role.
func (s *MemoryRoleStore) SetRole(role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[role.Name] = role
	return nil
}

// DeleteRole removes a role by name.
func (s *MemoryRoleStore) DeleteRole(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.roles[name]; !ok {
		return ErrRoleNotFound
	}
	delete(s.roles, name)
	return nil
}

// ListRoles returns all roles.
func (s *MemoryRoleStore) ListRoles() ([]*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roles := make([]*Role, 0, len(s.roles))
	for _, r := range s.roles {
		roles = append(roles, r)
	}
	return roles, nil
}

// MemoryAssignmentStore is an in-memory implementation of AssignmentStore.
type MemoryAssignmentStore struct {
	mu          sync.RWMutex
	assignments map[string]*RoleAssignment
}

// NewMemoryAssignmentStore creates a new in-memory assignment store.
func NewMemoryAssignmentStore() *MemoryAssignmentStore {
	return &MemoryAssignmentStore{
		assignments: make(map[string]*RoleAssignment),
	}
}

// GetAssignment retrieves the role assignment for a subject.
func (s *MemoryAssignmentStore) GetAssignment(subjectID string) (*RoleAssignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.assignments[subjectID]
	if !ok {
		return nil, ErrAssignmentNotFound
	}
	return a, nil
}

// SetAssignment creates or updates a role assignment.
func (s *MemoryAssignmentStore) SetAssignment(assignment *RoleAssignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assignments[assignment.SubjectID] = assignment
	return nil
}

// DeleteAssignment removes all role assignments for a subject.
func (s *MemoryAssignmentStore) DeleteAssignment(subjectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.assignments[subjectID]; !ok {
		return ErrAssignmentNotFound
	}
	delete(s.assignments, subjectID)
	return nil
}

// ListAssignments returns all role assignments.
func (s *MemoryAssignmentStore) ListAssignments() ([]*RoleAssignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	assignments := make([]*RoleAssignment, 0, len(s.assignments))
	for _, a := range s.assignments {
		assignments = append(assignments, a)
	}
	return assignments, nil
}

// BadgerRoleStore is a BadgerDB-backed implementation of RoleStore.
// Keys are stored under the Engine's KV namespace with the "rbac:role:" prefix.
type BadgerRoleStore struct {
	engine persistence.Engine
}

// NewBadgerRoleStore creates a new BadgerDB-backed role store.
func NewBadgerRoleStore(engine persistence.Engine) *BadgerRoleStore {
	return &BadgerRoleStore{engine: engine}
}

// GetRole retrieves a role by name from BadgerDB.
func (s *BadgerRoleStore) GetRole(name string) (*Role, error) {
	data, err := s.engine.Get(rolePrefix + name)
	if err != nil {
		return nil, ErrRoleNotFound
	}
	var role Role
	if err := json.Unmarshal(data, &role); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role %q: %w", name, err)
	}
	return &role, nil
}

// SetRole persists a role to BadgerDB.
func (s *BadgerRoleStore) SetRole(role *Role) error {
	data, err := json.Marshal(role)
	if err != nil {
		return fmt.Errorf("failed to marshal role %q: %w", role.Name, err)
	}
	return s.engine.Set(rolePrefix+role.Name, data)
}

// DeleteRole removes a role from BadgerDB.
func (s *BadgerRoleStore) DeleteRole(name string) error {
	return s.engine.Delete(rolePrefix + name)
}

// ListRoles returns all roles stored in BadgerDB.
func (s *BadgerRoleStore) ListRoles() ([]*Role, error) {
	keys, err := s.engine.List(rolePrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	roles := make([]*Role, 0, len(keys))
	for _, key := range keys {
		// engine.List returns keys with kvPrefix stripped but rolePrefix intact.
		name := strings.TrimPrefix(key, rolePrefix)
		role, err := s.GetRole(name)
		if err != nil {
			continue
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// BadgerAssignmentStore is a BadgerDB-backed implementation of AssignmentStore.
// Keys are stored under the Engine's KV namespace with the "rbac:assign:" prefix.
type BadgerAssignmentStore struct {
	engine persistence.Engine
}

// NewBadgerAssignmentStore creates a new BadgerDB-backed assignment store.
func NewBadgerAssignmentStore(engine persistence.Engine) *BadgerAssignmentStore {
	return &BadgerAssignmentStore{engine: engine}
}

// GetAssignment retrieves a role assignment by subject ID from BadgerDB.
func (s *BadgerAssignmentStore) GetAssignment(subjectID string) (*RoleAssignment, error) {
	data, err := s.engine.Get(assignPrefix + subjectID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	var assignment RoleAssignment
	if err := json.Unmarshal(data, &assignment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal assignment for %q: %w", subjectID, err)
	}
	return &assignment, nil
}

// SetAssignment persists a role assignment to BadgerDB.
func (s *BadgerAssignmentStore) SetAssignment(assignment *RoleAssignment) error {
	data, err := json.Marshal(assignment)
	if err != nil {
		return fmt.Errorf("failed to marshal assignment for %q: %w", assignment.SubjectID, err)
	}
	return s.engine.Set(assignPrefix+assignment.SubjectID, data)
}

// DeleteAssignment removes a role assignment from BadgerDB.
func (s *BadgerAssignmentStore) DeleteAssignment(subjectID string) error {
	return s.engine.Delete(assignPrefix + subjectID)
}

// ListAssignments returns all role assignments stored in BadgerDB.
func (s *BadgerAssignmentStore) ListAssignments() ([]*RoleAssignment, error) {
	keys, err := s.engine.List(assignPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list assignments: %w", err)
	}
	assignments := make([]*RoleAssignment, 0, len(keys))
	for _, key := range keys {
		subjectID := strings.TrimPrefix(key, assignPrefix)
		assignment, err := s.GetAssignment(subjectID)
		if err != nil {
			continue
		}
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}
