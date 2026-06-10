package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/rbac"
)

// mockRoleManager implements rbac.RoleManager for testing.
type mockRoleManager struct {
	roles       map[string]*rbac.Role
	assignments map[string]*rbac.RoleAssignment
	injectErr   error
}

func newMockRoleManager() *mockRoleManager {
	return &mockRoleManager{
		roles:       make(map[string]*rbac.Role),
		assignments: make(map[string]*rbac.RoleAssignment),
	}
}

func (m *mockRoleManager) CreateRole(role *rbac.Role) error {
	if m.injectErr != nil {
		return m.injectErr
	}
	if _, exists := m.roles[role.Name]; exists {
		return rbac.ErrRoleExists
	}
	m.roles[role.Name] = role
	return nil
}

func (m *mockRoleManager) GetRole(name string) (*rbac.Role, error) {
	if m.injectErr != nil {
		return nil, m.injectErr
	}
	r, ok := m.roles[name]
	if !ok {
		return nil, rbac.ErrRoleNotFound
	}
	return r, nil
}

func (m *mockRoleManager) UpdateRole(role *rbac.Role) error {
	if m.injectErr != nil {
		return m.injectErr
	}
	if _, ok := m.roles[role.Name]; !ok {
		return rbac.ErrRoleNotFound
	}
	m.roles[role.Name] = role
	return nil
}

func (m *mockRoleManager) DeleteRole(name string) error {
	if m.injectErr != nil {
		return m.injectErr
	}
	if _, ok := m.roles[name]; !ok {
		return rbac.ErrRoleNotFound
	}
	delete(m.roles, name)
	return nil
}

func (m *mockRoleManager) ListRoles() ([]*rbac.Role, error) {
	if m.injectErr != nil {
		return nil, m.injectErr
	}
	roles := make([]*rbac.Role, 0, len(m.roles))
	for _, r := range m.roles {
		roles = append(roles, r)
	}
	return roles, nil
}

func (m *mockRoleManager) AssignRole(subjectID string, roleNames []string, expiresAt *time.Time) error {
	if m.injectErr != nil {
		return m.injectErr
	}
	m.assignments[subjectID] = &rbac.RoleAssignment{SubjectID: subjectID, RoleNames: roleNames, ExpiresAt: expiresAt}
	return nil
}

func (m *mockRoleManager) UnassignRole(subjectID string) error {
	if m.injectErr != nil {
		return m.injectErr
	}
	if _, ok := m.assignments[subjectID]; !ok {
		return rbac.ErrAssignmentNotFound
	}
	delete(m.assignments, subjectID)
	return nil
}

func (m *mockRoleManager) ListAssignments() ([]*rbac.RoleAssignment, error) {
	if m.injectErr != nil {
		return nil, m.injectErr
	}
	out := make([]*rbac.RoleAssignment, 0, len(m.assignments))
	for _, a := range m.assignments {
		out = append(out, a)
	}
	return out, nil
}

func (m *mockRoleManager) GetAssignment(subjectID string) (*rbac.RoleAssignment, error) {
	if m.injectErr != nil {
		return nil, m.injectErr
	}
	a, ok := m.assignments[subjectID]
	if !ok {
		return nil, rbac.ErrAssignmentNotFound
	}
	return a, nil
}

func (m *mockRoleManager) GetEffectivePolicies(subjectID string) ([]string, error) {
	if m.injectErr != nil {
		return nil, m.injectErr
	}
	a, ok := m.assignments[subjectID]
	if !ok {
		return []string{}, nil
	}
	var policies []string
	for _, name := range a.RoleNames {
		if r, exists := m.roles[name]; exists {
			policies = append(policies, r.Policies...)
		}
	}
	return policies, nil
}

func (m *mockRoleManager) Authorize(subjectID string, directPolicies []string, resource, capability string) bool {
	return false
}

func setupRBACApp(mgr rbac.RoleManager) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	log := logger.New(0, "json")
	h := NewRBACHandler(mgr, log)
	h.RegisterRoutes(app)
	return app
}

func TestCreateRole_Success(t *testing.T) {
	app := setupRBACApp(newMockRoleManager())

	body, _ := json.Marshal(rbac.Role{Name: "admin", Policies: []string{"kv:write"}})
	req := httptest.NewRequest(http.MethodPost, "/rbac/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestCreateRole_Conflict(t *testing.T) {
	mgr := newMockRoleManager()
	mgr.roles["admin"] = &rbac.Role{Name: "admin"}
	app := setupRBACApp(mgr)

	body, _ := json.Marshal(rbac.Role{Name: "admin"})
	req := httptest.NewRequest(http.MethodPost, "/rbac/roles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestGetRole_Success(t *testing.T) {
	mgr := newMockRoleManager()
	mgr.roles["admin"] = &rbac.Role{Name: "admin", Policies: []string{"*"}}
	app := setupRBACApp(mgr)

	req := httptest.NewRequest(http.MethodGet, "/rbac/roles/admin", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetRole_NotFound(t *testing.T) {
	app := setupRBACApp(newMockRoleManager())

	req := httptest.NewRequest(http.MethodGet, "/rbac/roles/nonexistent", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteRole_Success(t *testing.T) {
	mgr := newMockRoleManager()
	mgr.roles["admin"] = &rbac.Role{Name: "admin"}
	app := setupRBACApp(mgr)

	req := httptest.NewRequest(http.MethodDelete, "/rbac/roles/admin", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestListRoles_Success(t *testing.T) {
	mgr := newMockRoleManager()
	mgr.roles["admin"] = &rbac.Role{Name: "admin"}
	mgr.roles["readonly"] = &rbac.Role{Name: "readonly"}
	app := setupRBACApp(mgr)

	req := httptest.NewRequest(http.MethodGet, "/rbac/roles", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(2), result["count"])
}

func TestAssignRole_Success(t *testing.T) {
	app := setupRBACApp(newMockRoleManager())

	body, _ := json.Marshal(map[string]interface{}{
		"subject_id": "user1",
		"role_names": []string{"admin"},
	})
	req := httptest.NewRequest(http.MethodPost, "/rbac/assignments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestUnassignRole_Success(t *testing.T) {
	mgr := newMockRoleManager()
	mgr.assignments["user1"] = &rbac.RoleAssignment{SubjectID: "user1", RoleNames: []string{"admin"}}
	app := setupRBACApp(mgr)

	req := httptest.NewRequest(http.MethodDelete, "/rbac/assignments/user1", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestGetEffectivePolicies_Success(t *testing.T) {
	mgr := newMockRoleManager()
	mgr.roles["admin"] = &rbac.Role{Name: "admin", Policies: []string{"kv:write", "svc:read"}}
	mgr.assignments["user1"] = &rbac.RoleAssignment{SubjectID: "user1", RoleNames: []string{"admin"}}
	app := setupRBACApp(mgr)

	req := httptest.NewRequest(http.MethodGet, "/rbac/subjects/user1/policies", http.NoBody)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(2), result["count"])
}
