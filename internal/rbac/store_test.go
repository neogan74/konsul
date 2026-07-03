package rbac

import (
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testNS = "default"

// ---------- MemoryRoleStore ----------

func TestMemoryRoleStore_CRUD(t *testing.T) {
	store := NewMemoryRoleStore()

	role := &Role{
		Name:        "admin",
		Description: "Administrator role",
		Policies:    []string{"read:*", "write:*"},
		ParentRoles: []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	require.NoError(t, store.SetRole(testNS, role))

	got, err := store.GetRole(testNS, "admin")
	require.NoError(t, err)
	assert.Equal(t, role.Name, got.Name)
	assert.Equal(t, role.Policies, got.Policies)

	role.Description = "Updated"
	require.NoError(t, store.SetRole(testNS, role))
	got2, err := store.GetRole(testNS, "admin")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got2.Description)

	require.NoError(t, store.DeleteRole(testNS, "admin"))

	_, err = store.GetRole(testNS, "admin")
	assert.ErrorIs(t, err, ErrRoleNotFound)

	assert.ErrorIs(t, store.DeleteRole(testNS, "admin"), ErrRoleNotFound)
}

func TestMemoryRoleStore_ListRoles(t *testing.T) {
	store := NewMemoryRoleStore()

	roles := []*Role{
		{Name: "reader", Policies: []string{"read:*"}},
		{Name: "writer", Policies: []string{"write:*"}},
		{Name: "admin", Policies: []string{"read:*", "write:*"}},
	}
	for _, r := range roles {
		require.NoError(t, store.SetRole(testNS, r))
	}

	all, err := store.ListRoles(testNS)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	names := make(map[string]bool)
	for _, r := range all {
		names[r.Name] = true
	}
	assert.True(t, names["reader"])
	assert.True(t, names["writer"])
	assert.True(t, names["admin"])
}

func TestMemoryRoleStore_NamespaceIsolation(t *testing.T) {
	store := NewMemoryRoleStore()
	require.NoError(t, store.SetRole("team-a", &Role{Name: "admin", Policies: []string{"write:*"}}))

	_, err := store.GetRole("team-b", "admin")
	assert.ErrorIs(t, err, ErrRoleNotFound)
}

func TestMemoryRoleStore_DuplicateCreate(t *testing.T) {
	store := NewMemoryRoleStore()
	roleStore := NewMemoryRoleStore()

	assignStore := NewMemoryAssignmentStore()
	mgr := NewManager(roleStore, assignStore, time.Minute, time.Hour, logger.GetDefault())
	t.Cleanup(mgr.Close)

	role := &Role{Name: "duplicate", Policies: []string{"read:kv"}}
	require.NoError(t, mgr.CreateRole(testNS, role))

	err := mgr.CreateRole(testNS, &Role{Name: "duplicate"})
	assert.ErrorIs(t, err, ErrRoleExists)

	// MemoryRoleStore.SetRole is idempotent (no duplicate check).
	require.NoError(t, store.SetRole(testNS, role))
	require.NoError(t, store.SetRole(testNS, role))
}

// ---------- MemoryAssignmentStore ----------

func TestMemoryAssignmentStore_CRUD(t *testing.T) {
	store := NewMemoryAssignmentStore()

	exp := time.Now().Add(time.Hour)
	assignment := &RoleAssignment{
		SubjectID: "user-1",
		RoleNames: []string{"admin", "reader"},
		ExpiresAt: &exp,
	}

	require.NoError(t, store.SetAssignment(testNS, assignment))

	got, err := store.GetAssignment(testNS, "user-1")
	require.NoError(t, err)
	assert.Equal(t, assignment.SubjectID, got.SubjectID)
	assert.Equal(t, assignment.RoleNames, got.RoleNames)
	require.NotNil(t, got.ExpiresAt)
	assert.WithinDuration(t, exp, *got.ExpiresAt, time.Second)

	assignment.RoleNames = []string{"reader"}
	require.NoError(t, store.SetAssignment(testNS, assignment))
	got2, err := store.GetAssignment(testNS, "user-1")
	require.NoError(t, err)
	assert.Equal(t, []string{"reader"}, got2.RoleNames)

	require.NoError(t, store.DeleteAssignment(testNS, "user-1"))

	_, err = store.GetAssignment(testNS, "user-1")
	assert.ErrorIs(t, err, ErrAssignmentNotFound)

	assert.ErrorIs(t, store.DeleteAssignment(testNS, "user-1"), ErrAssignmentNotFound)
}

func TestMemoryAssignmentStore_List(t *testing.T) {
	store := NewMemoryAssignmentStore()

	subjects := []string{"user-1", "user-2", "user-3"}
	for _, s := range subjects {
		require.NoError(t, store.SetAssignment(testNS, &RoleAssignment{
			SubjectID: s,
			RoleNames: []string{"reader"},
		}))
	}

	all, err := store.ListAssignments(testNS)
	require.NoError(t, err)
	assert.Len(t, all, 3)

	ids := make(map[string]bool)
	for _, a := range all {
		ids[a.SubjectID] = true
	}
	for _, s := range subjects {
		assert.True(t, ids[s], "expected subject %s in list", s)
	}
}

// ---------- BadgerRoleStore persistence round-trip ----------

func TestBadgerRoleStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	log := logger.GetDefault()

	engine1, err := persistence.NewBadgerEngine(dir, true, log)
	require.NoError(t, err)

	store1 := NewBadgerRoleStore(engine1)
	role := &Role{
		Name:        "persist-role",
		Description: "survives restart",
		Policies:    []string{"read:kv"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, store1.SetRole(testNS, role))
	require.NoError(t, engine1.Close())

	engine2, err := persistence.NewBadgerEngine(dir, true, log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine2.Close() })

	store2 := NewBadgerRoleStore(engine2)
	got, err := store2.GetRole(testNS, "persist-role")
	require.NoError(t, err)
	assert.Equal(t, role.Name, got.Name)
	assert.Equal(t, role.Policies, got.Policies)
	assert.Equal(t, role.Description, got.Description)
}

// ---------- BadgerAssignmentStore persistence round-trip ----------

func TestBadgerAssignmentStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	log := logger.GetDefault()

	engine1, err := persistence.NewBadgerEngine(dir, true, log)
	require.NoError(t, err)

	store1 := NewBadgerAssignmentStore(engine1)
	exp := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	assignment := &RoleAssignment{
		SubjectID: "persist-user",
		RoleNames: []string{"admin"},
		ExpiresAt: &exp,
	}
	require.NoError(t, store1.SetAssignment(testNS, assignment))
	require.NoError(t, engine1.Close())

	engine2, err := persistence.NewBadgerEngine(dir, true, log)
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine2.Close() })

	store2 := NewBadgerAssignmentStore(engine2)
	got, err := store2.GetAssignment(testNS, "persist-user")
	require.NoError(t, err)
	assert.Equal(t, assignment.SubjectID, got.SubjectID)
	assert.Equal(t, assignment.RoleNames, got.RoleNames)
	require.NotNil(t, got.ExpiresAt)
	assert.WithinDuration(t, exp, *got.ExpiresAt, time.Second)
}
