package rbac_test

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap/zapcore"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/rbac"
)

func newTestManager(t *testing.T) *rbac.Manager {
	t.Helper()
	roleStore := rbac.NewMemoryRoleStore()
	assignStore := rbac.NewMemoryAssignmentStore()
	log := logger.New(zapcore.ErrorLevel, "json")
	mgr := rbac.NewManager(roleStore, assignStore, 5*time.Minute, time.Minute, log)
	t.Cleanup(mgr.Close)
	return mgr
}

func TestManager_CreateAndGetRole(t *testing.T) {
	mgr := newTestManager(t)

	role := &rbac.Role{
		Name:        "admin",
		Description: "Administrator role",
		Policies:    []string{"read:*", "write:*"},
	}
	require.NoError(t, mgr.CreateRole(role))

	got, err := mgr.GetRole("admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", got.Name)
	assert.Equal(t, "Administrator role", got.Description)
	assert.ElementsMatch(t, []string{"read:*", "write:*"}, got.Policies)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestManager_CreateRole_Duplicate(t *testing.T) {
	mgr := newTestManager(t)

	role := &rbac.Role{Name: "viewer", Policies: []string{"read:*"}}
	require.NoError(t, mgr.CreateRole(role))

	err := mgr.CreateRole(&rbac.Role{Name: "viewer", Policies: []string{"read:kv"}})
	require.ErrorIs(t, err, rbac.ErrRoleExists)
}

func TestManager_DeleteRole_NotFound(t *testing.T) {
	mgr := newTestManager(t)

	err := mgr.DeleteRole("nonexistent")
	require.ErrorIs(t, err, rbac.ErrRoleNotFound)
}

func TestManager_ListRoles(t *testing.T) {
	mgr := newTestManager(t)

	names := []string{"alpha", "beta", "gamma"}
	for _, n := range names {
		require.NoError(t, mgr.CreateRole(&rbac.Role{Name: n, Policies: []string{"read:*"}}))
	}

	roles, err := mgr.ListRoles()
	require.NoError(t, err)
	assert.Len(t, roles, 3)

	got := make([]string, 0, len(roles))
	for _, r := range roles {
		got = append(got, r.Name)
	}
	sort.Strings(got)
	assert.Equal(t, names, got)
}

func TestManager_AssignAndGetEffectivePolicies(t *testing.T) {
	mgr := newTestManager(t)

	require.NoError(t, mgr.CreateRole(&rbac.Role{
		Name:     "editor",
		Policies: []string{"read:kv", "write:kv"},
	}))

	require.NoError(t, mgr.AssignRole("user-1", []string{"editor"}, nil))

	policies, err := mgr.GetEffectivePolicies("user-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:kv", "write:kv"}, policies)
}

func TestManager_InheritancePolicies(t *testing.T) {
	mgr := newTestManager(t)

	// base role
	require.NoError(t, mgr.CreateRole(&rbac.Role{
		Name:     "base",
		Policies: []string{"read:*"},
	}))
	// derived inherits from base and adds write
	require.NoError(t, mgr.CreateRole(&rbac.Role{
		Name:        "derived",
		Policies:    []string{"write:kv"},
		ParentRoles: []string{"base"},
	}))

	require.NoError(t, mgr.AssignRole("user-2", []string{"derived"}, nil))

	policies, err := mgr.GetEffectivePolicies("user-2")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:*", "write:kv"}, policies)
}

func TestManager_CycleDetection(t *testing.T) {
	mgr := newTestManager(t)

	// Create A → B → A cycle by having A inherit B and B inherit A.
	// We create A first without parents, then B pointing to A,
	// then update A to point to B.
	require.NoError(t, mgr.CreateRole(&rbac.Role{Name: "roleA", Policies: []string{"p1"}}))
	require.NoError(t, mgr.CreateRole(&rbac.Role{
		Name:        "roleB",
		Policies:    []string{"p2"},
		ParentRoles: []string{"roleA"},
	}))
	// Update A to point back at B — creates the cycle.
	require.NoError(t, mgr.UpdateRole(&rbac.Role{
		Name:        "roleA",
		Policies:    []string{"p1"},
		ParentRoles: []string{"roleB"},
	}))

	require.NoError(t, mgr.AssignRole("user-cycle", []string{"roleA"}, nil))

	_, err := mgr.GetEffectivePolicies("user-cycle")
	require.ErrorIs(t, err, rbac.ErrCyclicDependency)
}

func TestManager_MaxDepth(t *testing.T) {
	mgr := newTestManager(t)

	// Build a 7-level chain: L0 → L1 → L2 → L3 → L4 → L5 → L6
	// resolveInheritance uses depth > 5 as the limit, so 6 levels deep (depth 6) exceeds it.
	prev := ""
	for i := 6; i >= 0; i-- {
		name := "level" + string(rune('0'+i))
		role := &rbac.Role{Name: name, Policies: []string{"p" + string(rune('0'+i))}}
		if prev != "" {
			role.ParentRoles = []string{prev}
		}
		require.NoError(t, mgr.CreateRole(role))
		prev = name
	}

	// Assign the top-level role (level0 which inherits level1 → ... → level6)
	require.NoError(t, mgr.AssignRole("user-deep", []string{"level0"}, nil))

	_, err := mgr.GetEffectivePolicies("user-deep")
	require.ErrorIs(t, err, rbac.ErrMaxDepthExceeded)
}

func TestManager_ExpiredAssignment(t *testing.T) {
	mgr := newTestManager(t)

	require.NoError(t, mgr.CreateRole(&rbac.Role{Name: "temp", Policies: []string{"read:*"}}))

	past := time.Now().Add(-time.Minute)
	require.NoError(t, mgr.AssignRole("user-expired", []string{"temp"}, &past))

	policies, err := mgr.GetEffectivePolicies("user-expired")
	require.NoError(t, err)
	assert.Empty(t, policies, "expired assignment should yield no policies")
}

func TestManager_CacheInvalidation(t *testing.T) {
	mgr := newTestManager(t)

	require.NoError(t, mgr.CreateRole(&rbac.Role{Name: "r1", Policies: []string{"read:kv"}}))
	require.NoError(t, mgr.AssignRole("user-cache", []string{"r1"}, nil))

	// First call — populates cache.
	policies, err := mgr.GetEffectivePolicies("user-cache")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:kv"}, policies)

	// Update role — should invalidate entire cache.
	require.NoError(t, mgr.UpdateRole(&rbac.Role{
		Name:     "r1",
		Policies: []string{"read:kv", "write:kv"},
	}))

	// Second call — cache miss, should reflect updated policies.
	policies2, err := mgr.GetEffectivePolicies("user-cache")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:kv", "write:kv"}, policies2)
}
