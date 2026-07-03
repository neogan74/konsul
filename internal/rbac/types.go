package rbac

import (
	"errors"
	"time"
)

// Role represents an RBAC role with associated policies and optional inheritance.
type Role struct {
	Namespace   string    `json:"namespace,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Policies    []string  `json:"policies"`
	ParentRoles []string  `json:"parent_roles"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleAssignment maps a subject (user/token) to a set of roles within a namespace.
type RoleAssignment struct {
	Namespace string     `json:"namespace,omitempty"`
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
// All operations are scoped to a namespace. Pass "default" for the default namespace.
type RoleManager interface {
	// GetEffectivePolicies returns all policies for a subject within a namespace.
	GetEffectivePolicies(namespace, subjectID string) ([]string, error)
	// Authorize checks whether a subject has access to the given resource/capability.
	Authorize(namespace, subjectID string, directPolicies []string, resource string, capability string) bool
	// CreateRole creates a new role in the given namespace.
	CreateRole(namespace string, role *Role) error
	// GetRole retrieves a role by name from the given namespace.
	GetRole(namespace, name string) (*Role, error)
	// UpdateRole updates an existing role in the given namespace.
	UpdateRole(namespace string, role *Role) error
	// DeleteRole removes a role by name from the given namespace.
	DeleteRole(namespace, name string) error
	// ListRoles returns all roles in the given namespace.
	ListRoles(namespace string) ([]*Role, error)
	// AssignRole assigns one or more roles to a subject within a namespace.
	AssignRole(namespace, subjectID string, roleNames []string, expiresAt *time.Time) error
	// UnassignRole removes all role assignments for a subject within a namespace.
	UnassignRole(namespace, subjectID string) error
	// ListAssignments returns all role assignments within a namespace.
	ListAssignments(namespace string) ([]*RoleAssignment, error)
	// GetAssignment returns the role assignment for a subject within a namespace.
	GetAssignment(namespace, subjectID string) (*RoleAssignment, error)
}
