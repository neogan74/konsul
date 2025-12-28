package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/persistence"
	"go.uber.org/zap/zapcore"
)

// MockPersistenceEngine is a mock implementation of persistence.Engine for testing
type MockPersistenceEngine struct {
	BackupFunc        func(path string) error
	RestoreFunc       func(path string) error
	CloseFunc         func() error
	GetFunc           func(key string) ([]byte, error)
	SetFunc           func(key string, value []byte) error
	DeleteFunc        func(key string) error
	ListFunc          func(prefix string) ([]string, error)
	GetServiceFunc    func(name string) ([]byte, error)
	SetServiceFunc    func(name string, data []byte, ttl time.Duration) error
	DeleteServiceFunc func(name string) error
	ListServicesFunc  func() ([]string, error)
	BatchSetFunc      func(items map[string][]byte) error
	BatchDeleteFunc   func(keys []string) error
	BeginTxFunc       func() (persistence.Transaction, error)
}

// KV operations
func (m *MockPersistenceEngine) Get(key string) ([]byte, error) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return nil, nil
}

func (m *MockPersistenceEngine) Set(key string, value []byte) error {
	if m.SetFunc != nil {
		return m.SetFunc(key, value)
	}
	return nil
}

func (m *MockPersistenceEngine) Delete(key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(key)
	}
	return nil
}

func (m *MockPersistenceEngine) List(prefix string) ([]string, error) {
	if m.ListFunc != nil {
		return m.ListFunc(prefix)
	}
	return []string{}, nil
}

// Service operations
func (m *MockPersistenceEngine) GetService(name string) ([]byte, error) {
	if m.GetServiceFunc != nil {
		return m.GetServiceFunc(name)
	}
	return nil, nil
}

func (m *MockPersistenceEngine) SetService(name string, data []byte, ttl time.Duration) error {
	if m.SetServiceFunc != nil {
		return m.SetServiceFunc(name, data, ttl)
	}
	return nil
}

func (m *MockPersistenceEngine) DeleteService(name string) error {
	if m.DeleteServiceFunc != nil {
		return m.DeleteServiceFunc(name)
	}
	return nil
}

func (m *MockPersistenceEngine) ListServices() ([]string, error) {
	if m.ListServicesFunc != nil {
		return m.ListServicesFunc()
	}
	return []string{}, nil
}

// Batch operations
func (m *MockPersistenceEngine) BatchSet(items map[string][]byte) error {
	if m.BatchSetFunc != nil {
		return m.BatchSetFunc(items)
	}
	return nil
}

func (m *MockPersistenceEngine) BatchDelete(keys []string) error {
	if m.BatchDeleteFunc != nil {
		return m.BatchDeleteFunc(keys)
	}
	return nil
}

// Management
func (m *MockPersistenceEngine) Backup(path string) error {
	if m.BackupFunc != nil {
		return m.BackupFunc(path)
	}
	return nil
}

func (m *MockPersistenceEngine) Restore(path string) error {
	if m.RestoreFunc != nil {
		return m.RestoreFunc(path)
	}
	return nil
}

func (m *MockPersistenceEngine) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// Transaction support
func (m *MockPersistenceEngine) BeginTx() (persistence.Transaction, error) {
	if m.BeginTxFunc != nil {
		return m.BeginTxFunc()
	}
	return nil, nil
}

func setupBackupHandler(engine *MockPersistenceEngine) (*BackupHandler, *fiber.App) {
	log := logger.New(zapcore.InfoLevel, "json")
	handler := NewBackupHandler(engine, log)

	app := fiber.New()

	app.Post("/backup", handler.CreateBackup)
	app.Post("/restore", handler.RestoreBackup)
	app.Get("/export", handler.ExportData)
	app.Post("/import", handler.ImportData)
	app.Get("/backups", handler.ListBackups)

	return handler, app
}

func TestBackupHandler_CreateBackup_Success(t *testing.T) {
	mockEngine := &MockPersistenceEngine{
		BackupFunc: func(path string) error {
			return nil
		},
	}

	_, app := setupBackupHandler(mockEngine)

	req := httptest.NewRequest(http.MethodPost, "/backup", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := result["backup_path"]; !ok {
		t.Error("expected backup_path in response")
	}
	if _, ok := result["timestamp"]; !ok {
		t.Error("expected timestamp in response")
	}
}

func TestBackupHandler_CreateBackup_EngineError(t *testing.T) {
	mockEngine := &MockPersistenceEngine{
		BackupFunc: func(path string) error {
			return fiber.NewError(fiber.StatusInternalServerError, "backup failed")
		},
	}

	_, app := setupBackupHandler(mockEngine)

	req := httptest.NewRequest(http.MethodPost, "/backup", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_CreateBackup_NilEngine(t *testing.T) {
	_, app := setupBackupHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/backup", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("CreateBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_RestoreBackup_Success(t *testing.T) {
	mockEngine := &MockPersistenceEngine{
		RestoreFunc: func(path string) error {
			return nil
		},
	}

	_, app := setupBackupHandler(mockEngine)

	body := bytes.NewReader([]byte(`{"backup_path": "/path/to/backup.db"}`))
	req := httptest.NewRequest(http.MethodPost, "/restore", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("RestoreBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result["path"] != "/path/to/backup.db" {
		t.Errorf("expected path '/path/to/backup.db', got %v", result["path"])
	}
}

func TestBackupHandler_RestoreBackup_MissingPath(t *testing.T) {
	mockEngine := &MockPersistenceEngine{}
	_, app := setupBackupHandler(mockEngine)

	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPost, "/restore", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("RestoreBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_RestoreBackup_InvalidJSON(t *testing.T) {
	mockEngine := &MockPersistenceEngine{}
	_, app := setupBackupHandler(mockEngine)

	body := bytes.NewReader([]byte(`invalid json`))
	req := httptest.NewRequest(http.MethodPost, "/restore", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("RestoreBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_RestoreBackup_EngineError(t *testing.T) {
	mockEngine := &MockPersistenceEngine{
		RestoreFunc: func(path string) error {
			return fiber.NewError(fiber.StatusInternalServerError, "restore failed")
		},
	}

	_, app := setupBackupHandler(mockEngine)

	body := bytes.NewReader([]byte(`{"backup_path": "/invalid/path.db"}`))
	req := httptest.NewRequest(http.MethodPost, "/restore", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("RestoreBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_RestoreBackup_NilEngine(t *testing.T) {
	_, app := setupBackupHandler(nil)

	body := bytes.NewReader([]byte(`{"backup_path": "/path/to/backup.db"}`))
	req := httptest.NewRequest(http.MethodPost, "/restore", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("RestoreBackup request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_ExportData_Success(t *testing.T) {
	mockEngine := &MockPersistenceEngine{}

	_, app := setupBackupHandler(mockEngine)

	req := httptest.NewRequest(http.MethodGet, "/export", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ExportData request failed: %v", err)
	}

	// Note: The current implementation requires BadgerEngine type assertion
	// Mock engine will return BadRequest since it's not a BadgerEngine
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for non-BadgerEngine, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_ExportData_NilEngine(t *testing.T) {
	_, app := setupBackupHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/export", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ExportData request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_ImportData_Success(t *testing.T) {
	mockEngine := &MockPersistenceEngine{}

	_, app := setupBackupHandler(mockEngine)

	body := bytes.NewReader([]byte(`{"kv": {"key1": "value1"}, "services": []}`))
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ImportData request failed: %v", err)
	}

	// Note: The current implementation requires BadgerEngine type assertion
	// Mock engine will return BadRequest since it's not a BadgerEngine
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for non-BadgerEngine, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_ImportData_InvalidJSON(t *testing.T) {
	mockEngine := &MockPersistenceEngine{}
	_, app := setupBackupHandler(mockEngine)

	body := bytes.NewReader([]byte(`invalid json`))
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ImportData request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_ImportData_NilEngine(t *testing.T) {
	_, app := setupBackupHandler(nil)

	body := bytes.NewReader([]byte(`{"kv": {}}`))
	req := httptest.NewRequest(http.MethodPost, "/import", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ImportData request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupHandler_ListBackups_NotImplemented(t *testing.T) {
	mockEngine := &MockPersistenceEngine{}
	_, app := setupBackupHandler(mockEngine)

	req := httptest.NewRequest(http.MethodGet, "/backups", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ListBackups request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check that it returns a message about not being implemented
	if _, ok := result["message"]; !ok {
		t.Error("expected message in response")
	}
}

func TestBackupHandler_ListBackups_NilEngine(t *testing.T) {
	_, app := setupBackupHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/backups", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("ListBackups request failed: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}
