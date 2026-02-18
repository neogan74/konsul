package persistence

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/logger"
)

func TestBadgerEngine_Basic(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	log := logger.NewFromConfig("info", "text")

	engine, err := NewBadgerEngine(tempDir, true, log)
	if err != nil {
		t.Fatalf("Failed to create BadgerEngine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	// Test KV operations
	key := "test-key"
	value := []byte("test-value")

	// Test Set
	err = engine.Set(key, value)
	if err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	// Test Get
	retrievedValue, err := engine.Get(key)
	if err != nil {
		t.Fatalf("Failed to get key: %v", err)
	}

	if string(retrievedValue) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrievedValue)
	}

	// Test List
	keys, err := engine.List("")
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(keys) != 1 || keys[0] != key {
		t.Errorf("Expected [%s], got %v", key, keys)
	}

	// Test Delete
	err = engine.Delete(key)
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	// Verify deletion
	_, err = engine.Get(key)
	if err == nil {
		t.Error("Expected error when getting deleted key")
	}
}

func TestBadgerEngine_Services(t *testing.T) {
	tempDir := t.TempDir()
	log := logger.NewFromConfig("info", "text")

	engine, err := NewBadgerEngine(tempDir, true, log)
	if err != nil {
		t.Fatalf("Failed to create BadgerEngine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	serviceName := "test-service"
	serviceData := []byte(`{"name":"test-service","address":"localhost","port":8080}`)
	ttl := 5 * time.Second

	// Test SetService
	err = engine.SetService(serviceName, serviceData, ttl)
	if err != nil {
		t.Fatalf("Failed to set service: %v", err)
	}

	// Test GetService
	retrievedData, err := engine.GetService(serviceName)
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if string(retrievedData) != string(serviceData) {
		t.Errorf("Expected %s, got %s", serviceData, retrievedData)
	}

	// Test ListServices
	services, err := engine.ListServices()
	if err != nil {
		t.Fatalf("Failed to list services: %v", err)
	}

	if len(services) != 1 || services[0] != serviceName {
		t.Errorf("Expected [%s], got %v", serviceName, services)
	}

	// Test service expiration (wait a bit longer than TTL)
	time.Sleep(6 * time.Second)

	// Service should be expired
	_, err = engine.GetService(serviceName)
	if err == nil {
		t.Error("Expected error when getting expired service")
	}

	// Should not appear in list
	services, err = engine.ListServices()
	if err != nil {
		t.Fatalf("Failed to list services: %v", err)
	}

	if len(services) != 0 {
		t.Errorf("Expected empty list, got %v", services)
	}
}

func TestBadgerEngine_BatchOperations(t *testing.T) {
	tempDir := t.TempDir()
	log := logger.NewFromConfig("info", "text")

	engine, err := NewBadgerEngine(tempDir, true, log)
	if err != nil {
		t.Fatalf("Failed to create BadgerEngine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	// Test BatchSet
	items := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	err = engine.BatchSet(items)
	if err != nil {
		t.Fatalf("Failed to batch set: %v", err)
	}

	// Verify all keys were set
	for key, expectedValue := range items {
		value, err := engine.Get(key)
		if err != nil {
			t.Fatalf("Failed to get key %s: %v", key, err)
		}
		if string(value) != string(expectedValue) {
			t.Errorf("Key %s: expected %s, got %s", key, expectedValue, value)
		}
	}

	// Test BatchDelete
	keys := []string{"key1", "key2"}
	err = engine.BatchDelete(keys)
	if err != nil {
		t.Fatalf("Failed to batch delete: %v", err)
	}

	// Verify keys were deleted
	for _, key := range keys {
		_, err := engine.Get(key)
		if err == nil {
			t.Errorf("Key %s should have been deleted", key)
		}
	}

	// Verify key3 still exists
	_, err = engine.Get("key3")
	if err != nil {
		t.Error("Key3 should still exist")
	}
}

func TestBadgerEngine_BackupRestore(t *testing.T) {
	tempDir := t.TempDir()
	backupPath := filepath.Join(tempDir, "backup.db")
	log := logger.NewFromConfig("info", "text")

	// Create engine and add some data
	engine, err := NewBadgerEngine(tempDir, true, log)
	if err != nil {
		t.Fatalf("Failed to create BadgerEngine: %v", err)
	}

	// Add test data
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	for key, value := range testData {
		err = engine.Set(key, value)
		if err != nil {
			t.Fatalf("Failed to set key %s: %v", key, err)
		}
	}

	// Create backup
	err = engine.Backup(backupPath)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("Backup file was not created")
	}

	// Close original engine
	defer func() { _ = engine.Close() }()

	// Create new engine with different directory
	restoreDir := filepath.Join(tempDir, "restore")
	newEngine, err := NewBadgerEngine(restoreDir, true, log)
	if err != nil {
		t.Fatalf("Failed to create new BadgerEngine: %v", err)
	}
	defer func() { _ = newEngine.Close() }()

	// Restore backup
	err = newEngine.Restore(backupPath)
	if err != nil {
		t.Fatalf("Failed to restore backup: %v", err)
	}

	// Verify restored data
	for key, expectedValue := range testData {
		value, err := newEngine.Get(key)
		if err != nil {
			t.Fatalf("Failed to get restored key %s: %v", key, err)
		}
		if string(value) != string(expectedValue) {
			t.Errorf("Restored key %s: expected %s, got %s", key, expectedValue, value)
		}
	}
}

func TestBadgerEngine_Transactions(t *testing.T) {
	tempDir := t.TempDir()
	log := logger.NewFromConfig("info", "text")

	engine, err := NewBadgerEngine(tempDir, true, log)
	if err != nil {
		t.Fatalf("Failed to create BadgerEngine: %v", err)
	}
	defer func() { _ = engine.Close() }()

	// Test successful transaction
	tx, err := engine.BeginTx()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	err = tx.Set("tx-key1", []byte("tx-value1"))
	if err != nil {
		t.Fatalf("Failed to set in transaction: %v", err)
	}

	err = tx.Set("tx-key2", []byte("tx-value2"))
	if err != nil {
		t.Fatalf("Failed to set in transaction: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify transaction results
	value, err := engine.Get("tx-key1")
	if err != nil {
		t.Fatalf("Failed to get tx-key1: %v", err)
	}
	if string(value) != "tx-value1" {
		t.Errorf("Expected tx-value1, got %s", value)
	}

	// Test rollback transaction
	tx2, err := engine.BeginTx()
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}

	err = tx2.Set("rollback-key", []byte("rollback-value"))
	if err != nil {
		t.Fatalf("Failed to set in transaction: %v", err)
	}

	err = tx2.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify rollback - key should not exist
	_, err = engine.Get("rollback-key")
	if err == nil {
		t.Error("rollback-key should not exist after rollback")
	}
}
