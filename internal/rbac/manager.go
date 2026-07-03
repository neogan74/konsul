package rbac

import (
	"strings"
	"sync"
	"time"

	"github.com/neogan74/konsul/internal/logger"
)

// cacheEntry holds cached effective policies for a subject with an expiry.
type cacheEntry struct {
	policies  []string
	expiresAt time.Time
}

// RoleManager is the concrete implementation of the RoleManager interface.
// It wraps RoleStore and AssignmentStore with an in-memory TTL cache and a
// background goroutine that evicts expired assignments.
type Manager struct {
	roles       RoleStore
	assignments AssignmentStore
	cache       map[string]cacheEntry
	cacheMu     sync.RWMutex
	cacheTTL    time.Duration
	stopCh      chan struct{}
	log         logger.Logger
}

// NewManager constructs a Manager and starts the background expiration loop.
func NewManager(
	roles RoleStore,
	assignments AssignmentStore,
	cacheTTL time.Duration,
	expirationInterval time.Duration,
	log logger.Logger,
) *Manager {
	m := &Manager{
		roles:       roles,
		assignments: assignments,
		cache:       make(map[string]cacheEntry),
		cacheTTL:    cacheTTL,
		stopCh:      make(chan struct{}),
		log:         log,
	}
	go m.runExpirationLoop(expirationInterval)
	return m
}

// Close stops the background expiration goroutine.
func (m *Manager) Close() {
	close(m.stopCh)
}

// runExpirationLoop ticks at the given interval and removes expired assignments.
func (m *Manager) runExpirationLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.expireAssignments()
		case <-m.stopCh:
			return
		}
	}
}

// expireAssignments scans all assignments across all namespaces and deletes expired ones.
func (m *Manager) expireAssignments() {
	all, err := m.assignments.ListAllAssignments()
	if err != nil {
		m.log.Warn("failed to list assignments for expiration", logger.Error(err))
		return
	}
	now := time.Now()
	for _, a := range all {
		if a.ExpiresAt != nil && now.After(*a.ExpiresAt) {
			ns := a.Namespace
			if ns == "" {
				ns = "default"
			}
			if delErr := m.assignments.DeleteAssignment(ns, a.SubjectID); delErr == nil {
				RBACAssignmentsExpiredTotal.Inc()
				RBACAssignmentsTotal.Dec()
				m.invalidateCacheFor(ns + ":" + a.SubjectID)
				m.log.Info("expired assignment removed",
					logger.String("namespace", ns),
					logger.String("subject", a.SubjectID))
			}
		}
	}
}

// invalidateCache clears all cached policy entries (used after role mutations).
func (m *Manager) invalidateCache() {
	m.cacheMu.Lock()
	m.cache = make(map[string]cacheEntry)
	m.cacheMu.Unlock()
}

// invalidateCacheFor removes the cache entry for a "namespace:subjectID" key.
func (m *Manager) invalidateCacheFor(cacheKey string) {
	m.cacheMu.Lock()
	delete(m.cache, cacheKey)
	m.cacheMu.Unlock()
}

func cacheKey(namespace, subjectID string) string { return namespace + ":" + subjectID }

// GetEffectivePolicies returns all policies for a subject within a namespace.
// The result is cached until cacheTTL expires.
func (m *Manager) GetEffectivePolicies(namespace, subjectID string) ([]string, error) {
	key := cacheKey(namespace, subjectID)
	m.cacheMu.RLock()
	entry, ok := m.cache[key]
	m.cacheMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.policies, nil
	}

	policies, err := m.resolveForUser(namespace, subjectID)
	if err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	m.cache[key] = cacheEntry{
		policies:  policies,
		expiresAt: time.Now().Add(m.cacheTTL),
	}
	m.cacheMu.Unlock()
	return policies, nil
}

// resolveForUser fetches the assignment for a subject and resolves all role policies.
func (m *Manager) resolveForUser(namespace, subjectID string) ([]string, error) {
	assignment, err := m.assignments.GetAssignment(namespace, subjectID)
	if err == ErrAssignmentNotFound {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	if assignment.ExpiresAt != nil && time.Now().After(*assignment.ExpiresAt) {
		return []string{}, nil
	}

	seen := make(map[string]bool)
	for _, roleName := range assignment.RoleNames {
		visited := make(map[string]bool)
		policies, err := m.resolveInheritance(namespace, roleName, visited, 0)
		if err != nil {
			return nil, err
		}
		for _, p := range policies {
			seen[p] = true
		}
	}

	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	return result, nil
}

// resolveInheritance performs a depth-first traversal of the role hierarchy within a namespace.
func (m *Manager) resolveInheritance(namespace, roleName string, visited map[string]bool, depth int) ([]string, error) {
	if depth > 5 {
		return nil, ErrMaxDepthExceeded
	}
	if visited[roleName] {
		return nil, ErrCyclicDependency
	}

	role, err := m.roles.GetRole(namespace, roleName)
	if err != nil {
		return []string{}, nil
	}

	branchVisited := make(map[string]bool, len(visited)+1)
	for k, v := range visited {
		branchVisited[k] = v
	}
	branchVisited[roleName] = true

	seen := make(map[string]bool)
	for _, p := range role.Policies {
		seen[p] = true
	}

	for _, parent := range role.ParentRoles {
		parentPolicies, err := m.resolveInheritance(namespace, parent, branchVisited, depth+1)
		if err != nil {
			return nil, err
		}
		for _, p := range parentPolicies {
			seen[p] = true
		}
	}

	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	return result, nil
}

// Authorize checks whether a subject has access to the given resource/capability within a namespace.
func (m *Manager) Authorize(namespace, subjectID string, directPolicies []string, resource string, capability string) bool {
	start := time.Now()

	effective, err := m.GetEffectivePolicies(namespace, subjectID)
	if err != nil {
		m.log.Warn("failed to get effective policies for authorization",
			logger.String("namespace", namespace),
			logger.String("subject", subjectID), logger.Error(err))
		RBACAuthorizationDuration.WithLabelValues("deny").Observe(time.Since(start).Seconds())
		return false
	}

	// Merge effective + direct policies (deduplicate).
	allPolicies := make(map[string]bool, len(effective)+len(directPolicies))
	for _, p := range effective {
		allPolicies[p] = true
	}
	for _, p := range directPolicies {
		allPolicies[p] = true
	}

	target := capability + ":" + resource
	allowed := false
	for p := range allPolicies {
		if p == "*" || p == target || strings.HasPrefix(p, capability+":*") {
			allowed = true
			break
		}
	}

	result := "deny"
	if allowed {
		result = "allow"
	}
	RBACAuthorizationDuration.WithLabelValues(result).Observe(time.Since(start).Seconds())
	return allowed
}

// CreateRole creates a new role in the given namespace; fails if the role already exists.
func (m *Manager) CreateRole(namespace string, role *Role) error {
	if _, err := m.roles.GetRole(namespace, role.Name); err == nil {
		return ErrRoleExists
	}
	now := time.Now()
	role.Namespace = namespace
	role.CreatedAt = now
	role.UpdatedAt = now
	if err := m.roles.SetRole(namespace, role); err != nil {
		return err
	}
	m.invalidateCache()
	RBACRolesTotal.Inc()
	return nil
}

// GetRole retrieves a role by name from the given namespace.
func (m *Manager) GetRole(namespace, name string) (*Role, error) {
	return m.roles.GetRole(namespace, name)
}

// UpdateRole updates an existing role in the given namespace.
func (m *Manager) UpdateRole(namespace string, role *Role) error {
	existing, err := m.roles.GetRole(namespace, role.Name)
	if err != nil {
		return err
	}
	role.Namespace = namespace
	role.CreatedAt = existing.CreatedAt
	role.UpdatedAt = time.Now()
	if err := m.roles.SetRole(namespace, role); err != nil {
		return err
	}
	m.invalidateCache()
	return nil
}

// DeleteRole removes a role by name from the given namespace.
func (m *Manager) DeleteRole(namespace, name string) error {
	if err := m.roles.DeleteRole(namespace, name); err != nil {
		return err
	}
	m.invalidateCache()
	RBACRolesTotal.Dec()
	return nil
}

// ListRoles returns all roles in the given namespace.
func (m *Manager) ListRoles(namespace string) ([]*Role, error) {
	return m.roles.ListRoles(namespace)
}

// AssignRole assigns one or more roles to a subject within a namespace.
func (m *Manager) AssignRole(namespace, subjectID string, roleNames []string, expiresAt *time.Time) error {
	assignment := &RoleAssignment{
		Namespace: namespace,
		SubjectID: subjectID,
		RoleNames: roleNames,
		ExpiresAt: expiresAt,
	}
	if err := m.assignments.SetAssignment(namespace, assignment); err != nil {
		return err
	}
	m.invalidateCacheFor(cacheKey(namespace, subjectID))
	RBACAssignmentsTotal.Inc()
	return nil
}

// UnassignRole removes all role assignments for a subject within a namespace.
func (m *Manager) UnassignRole(namespace, subjectID string) error {
	if err := m.assignments.DeleteAssignment(namespace, subjectID); err != nil {
		return err
	}
	m.invalidateCacheFor(cacheKey(namespace, subjectID))
	RBACAssignmentsTotal.Dec()
	return nil
}

// ListAssignments returns all role assignments within a namespace.
func (m *Manager) ListAssignments(namespace string) ([]*RoleAssignment, error) {
	return m.assignments.ListAssignments(namespace)
}

// GetAssignment returns the role assignment for a subject within a namespace.
func (m *Manager) GetAssignment(namespace, subjectID string) (*RoleAssignment, error) {
	return m.assignments.GetAssignment(namespace, subjectID)
}
