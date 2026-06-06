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

// expireAssignments scans all assignments and deletes those that have expired.
func (m *Manager) expireAssignments() {
	all, err := m.assignments.ListAssignments()
	if err != nil {
		m.log.Warn("failed to list assignments for expiration", logger.Error(err))
		return
	}
	now := time.Now()
	for _, a := range all {
		if a.ExpiresAt != nil && now.After(*a.ExpiresAt) {
			if delErr := m.assignments.DeleteAssignment(a.SubjectID); delErr == nil {
				RBACAssignmentsExpiredTotal.Inc()
				RBACAssignmentsTotal.Dec()
				m.invalidateCacheFor(a.SubjectID)
				m.log.Info("expired assignment removed", logger.String("subject", a.SubjectID))
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

// invalidateCacheFor removes the cache entry for a single subject.
func (m *Manager) invalidateCacheFor(subjectID string) {
	m.cacheMu.Lock()
	delete(m.cache, subjectID)
	m.cacheMu.Unlock()
}

// GetEffectivePolicies returns all policies for a subject (including inherited).
// The result is cached until cacheTTL expires.
func (m *Manager) GetEffectivePolicies(subjectID string) ([]string, error) {
	// Fast path: cache hit.
	m.cacheMu.RLock()
	entry, ok := m.cache[subjectID]
	m.cacheMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.policies, nil
	}

	policies, err := m.resolveForUser(subjectID)
	if err != nil {
		return nil, err
	}

	// Store in cache.
	m.cacheMu.Lock()
	m.cache[subjectID] = cacheEntry{
		policies:  policies,
		expiresAt: time.Now().Add(m.cacheTTL),
	}
	m.cacheMu.Unlock()
	return policies, nil
}

// resolveForUser fetches the assignment for a subject and resolves all role policies.
func (m *Manager) resolveForUser(subjectID string) ([]string, error) {
	assignment, err := m.assignments.GetAssignment(subjectID)
	if err == ErrAssignmentNotFound {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	// Check assignment expiry.
	if assignment.ExpiresAt != nil && time.Now().After(*assignment.ExpiresAt) {
		return []string{}, nil
	}

	seen := make(map[string]bool)
	for _, roleName := range assignment.RoleNames {
		visited := make(map[string]bool)
		policies, err := m.resolveInheritance(roleName, visited, 0)
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

// resolveInheritance performs a depth-first traversal of the role hierarchy.
// visited tracks the current DFS path for cycle detection (cloned at each branch
// so diamond inheritance is allowed; only true cycles are rejected).
func (m *Manager) resolveInheritance(roleName string, visited map[string]bool, depth int) ([]string, error) {
	if depth > 5 {
		return nil, ErrMaxDepthExceeded
	}
	if visited[roleName] {
		return nil, ErrCyclicDependency
	}

	role, err := m.roles.GetRole(roleName)
	if err != nil {
		// Missing role: skip silently (role may have been deleted after assignment).
		return []string{}, nil
	}

	// Clone visited for this branch so sibling branches remain unaffected.
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
		parentPolicies, err := m.resolveInheritance(parent, branchVisited, depth+1)
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

// Authorize checks whether a subject has access to the given resource/capability.
// It merges subjectID's effective policies with any directly provided policies,
// then checks for a matching policy string: "<capability>:<resource>" or "*".
// Authorization duration is recorded as a Prometheus metric.
func (m *Manager) Authorize(subjectID string, directPolicies []string, resource string, capability string) bool {
	start := time.Now()

	effective, err := m.GetEffectivePolicies(subjectID)
	if err != nil {
		m.log.Warn("failed to get effective policies for authorization",
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

// CreateRole creates a new role; fails if the role already exists.
func (m *Manager) CreateRole(role *Role) error {
	if _, err := m.roles.GetRole(role.Name); err == nil {
		return ErrRoleExists
	}
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now
	if err := m.roles.SetRole(role); err != nil {
		return err
	}
	m.invalidateCache()
	RBACRolesTotal.Inc()
	return nil
}

// GetRole retrieves a role by name.
func (m *Manager) GetRole(name string) (*Role, error) {
	return m.roles.GetRole(name)
}

// UpdateRole updates an existing role; fails if the role does not exist.
func (m *Manager) UpdateRole(role *Role) error {
	existing, err := m.roles.GetRole(role.Name)
	if err != nil {
		return err // ErrRoleNotFound
	}
	role.CreatedAt = existing.CreatedAt
	role.UpdatedAt = time.Now()
	if err := m.roles.SetRole(role); err != nil {
		return err
	}
	m.invalidateCache()
	return nil
}

// DeleteRole removes a role by name.
func (m *Manager) DeleteRole(name string) error {
	if err := m.roles.DeleteRole(name); err != nil {
		return err
	}
	m.invalidateCache()
	RBACRolesTotal.Dec()
	return nil
}

// ListRoles returns all roles.
func (m *Manager) ListRoles() ([]*Role, error) {
	return m.roles.ListRoles()
}

// AssignRole assigns one or more roles to a subject.
func (m *Manager) AssignRole(subjectID string, roleNames []string, expiresAt *time.Time) error {
	assignment := &RoleAssignment{
		SubjectID: subjectID,
		RoleNames: roleNames,
		ExpiresAt: expiresAt,
	}
	if err := m.assignments.SetAssignment(assignment); err != nil {
		return err
	}
	m.invalidateCacheFor(subjectID)
	RBACAssignmentsTotal.Inc()
	return nil
}

// UnassignRole removes all role assignments for a subject.
func (m *Manager) UnassignRole(subjectID string) error {
	if err := m.assignments.DeleteAssignment(subjectID); err != nil {
		return err
	}
	m.invalidateCacheFor(subjectID)
	RBACAssignmentsTotal.Dec()
	return nil
}

// ListAssignments returns all role assignments.
func (m *Manager) ListAssignments() ([]*RoleAssignment, error) {
	return m.assignments.ListAssignments()
}

// GetAssignment returns the role assignment for a subject.
func (m *Manager) GetAssignment(subjectID string) (*RoleAssignment, error) {
	return m.assignments.GetAssignment(subjectID)
}
