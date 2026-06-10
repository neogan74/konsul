package rbac

import (
	"errors"
	"time"
)

// Role represents an RBAC role with associated policies and optional inheritance.
type Role struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Policies    []string  `json:"policies"`
	ParentRoles []string  `json:"parent_roles"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleAssignment maps a subject (user/token) to a set of roles.
type RoleAssignment struct {
	SubjectID string     `json:"subject_id"`
	RoleNames []string   `json:"role_names"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// GroupRoleMapping maps an external group to a set of roles.
type GroupRoleMapping struct {
	GroupID     string   `json:"group_id"`
	GroupSource string   `json:"group_source"`
	RoleNames   []string `json:"role_names"`
}

// Sentinel errors for RBAC operations.
var (
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleExists         = errors.New("role already exists")
	ErrAssignmentNotFound = errors.New("assignment not found")
	ErrCyclicDependency   = errors.New("cyclic dependency detected")
	ErrMaxDepthExceeded   = errors.New("max role inheritance depth exceeded")
)

// RoleManager defines the high-level RBAC operations used by handlers and middleware.
// Defined here (not in manager.go) so Phase 04 can mock it without importing the implementation.
type RoleManager interface {
	// GetEffectivePolicies returns all policies for a subject, including inherited ones.
	GetEffectivePolicies(subjectID string) ([]string, error)
	// Authorize checks whether a subject has access to the given resource/capability.
	Authorize(subjectID string, directPolicies []string, resource string, capability string) bool
	// CreateRole creates a new role.
	CreateRole(role *Role) error
	// GetRole retrieves a role by name.
	GetRole(name string) (*Role, error)
	// UpdateRole updates an existing role.
	UpdateRole(role *Role) error
	// DeleteRole removes a role by name.
	DeleteRole(name string) error
	// ListRoles returns all roles.
	ListRoles() ([]*Role, error)
	// AssignRole assigns one or more roles to a subject.
	AssignRole(subjectID string, roleNames []string, expiresAt *time.Time) error
	// UnassignRole removes all role assignments for a subject.
	UnassignRole(subjectID string) error
	// ListAssignments returns all role assignments.
	ListAssignments() ([]*RoleAssignment, error)
	// GetAssignment returns the role assignment for a subject.
	GetAssignment(subjectID string) (*RoleAssignment, error)
}
