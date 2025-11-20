package store

import (
	"fmt"
	"sync"
	"testing"
)

func TestKVStore_SetCAS_CreateOnly(t *testing.T) {
	store := NewKVStore()

	// Test create-only with expectedIndex=0
	newIndex, err := store.SetCAS("test-key", "value1", 0)
	if err != nil {
		t.Fatalf("Expected create to succeed, got error: %v", err)
	}
	if newIndex == 0 {
		t.Fatalf("Expected non-zero index, got 0")
	}

	// Test create-only fails when key exists
	_, err = store.SetCAS("test-key", "value2", 0)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify value wasn't changed
	value, ok := store.Get("test-key")
	if !ok || value != "value1" {
		t.Fatalf("Expected value1, got %v", value)
	}
}

func TestKVStore_SetCAS_Update(t *testing.T) {
	store := NewKVStore()

	// Create initial key
	store.Set("test-key", "value1")
	entry, ok := store.GetEntry("test-key")
	if !ok {
		t.Fatal("Key should exist")
	}
	initialIndex := entry.ModifyIndex

	// Test successful CAS update
	newIndex, err := store.SetCAS("test-key", "value2", initialIndex)
	if err != nil {
		t.Fatalf("Expected update to succeed, got error: %v", err)
	}
	if newIndex <= initialIndex {
		t.Fatalf("Expected new index (%d) > initial index (%d)", newIndex, initialIndex)
	}

	// Verify value was changed
	value, ok := store.Get("test-key")
	if !ok || value != "value2" {
		t.Fatalf("Expected value2, got %v", value)
	}

	// Test CAS fails with old index
	_, err = store.SetCAS("test-key", "value3", initialIndex)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify value wasn't changed
	value, ok = store.Get("test-key")
	if !ok || value != "value2" {
		t.Fatalf("Expected value2, got %v", value)
	}
}

func TestKVStore_SetCAS_NotFound(t *testing.T) {
	store := NewKVStore()

	// Test CAS fails for non-existent key with non-zero index
	_, err := store.SetCAS("missing-key", "value", 123)
	if err == nil {
		t.Fatal("Expected NotFoundError, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got %T: %v", err, err)
	}
}

func TestKVStore_DeleteCAS(t *testing.T) {
	store := NewKVStore()

	// Create a key
	store.Set("test-key", "value1")
	entry, _ := store.GetEntry("test-key")
	correctIndex := entry.ModifyIndex

	// Test delete with wrong index fails
	err := store.DeleteCAS("test-key", correctIndex+1)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify key still exists
	_, ok := store.Get("test-key")
	if !ok {
		t.Fatal("Key should still exist")
	}

	// Test delete with correct index succeeds
	err = store.DeleteCAS("test-key", correctIndex)
	if err != nil {
		t.Fatalf("Expected delete to succeed, got error: %v", err)
	}

	// Verify key is deleted
	_, ok = store.Get("test-key")
	if ok {
		t.Fatal("Key should be deleted")
	}

	// Test delete of non-existent key fails
	err = store.DeleteCAS("test-key", correctIndex)
	if err == nil {
		t.Fatal("Expected NotFoundError, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got %T: %v", err, err)
	}
}

func TestKVStore_BatchSetCAS(t *testing.T) {
	store := NewKVStore()

	// Create initial keys
	store.Set("key1", "value1")
	store.Set("key2", "value2")
	entry1, _ := store.GetEntry("key1")
	entry2, _ := store.GetEntry("key2")

	// Test successful batch CAS
	items := map[string]string{
		"key1": "new-value1",
		"key2": "new-value2",
		"key3": "value3", // New key
	}
	expectedIndices := map[string]uint64{
		"key1": entry1.ModifyIndex,
		"key2": entry2.ModifyIndex,
		"key3": 0, // Create only
	}

	newIndices, err := store.BatchSetCAS(items, expectedIndices)
	if err != nil {
		t.Fatalf("Expected batch CAS to succeed, got error: %v", err)
	}
	if len(newIndices) != 3 {
		t.Fatalf("Expected 3 new indices, got %d", len(newIndices))
	}

	// Verify all values were updated
	for key, expectedValue := range items {
		value, ok := store.Get(key)
		if !ok || value != expectedValue {
			t.Fatalf("Expected %s for key %s, got %s", expectedValue, key, value)
		}
	}

	// Test batch CAS fails when one key has wrong index
	items2 := map[string]string{
		"key1": "another-value1",
		"key2": "another-value2",
	}
	expectedIndices2 := map[string]uint64{
		"key1": newIndices["key1"],
		"key2": entry2.ModifyIndex, // Wrong index (old one)
	}

	_, err = store.BatchSetCAS(items2, expectedIndices2)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify NO values were changed (atomic operation)
	value1, _ := store.Get("key1")
	value2, _ := store.Get("key2")
	if value1 != "new-value1" || value2 != "new-value2" {
		t.Fatal("Batch CAS should be atomic - no values should change on conflict")
	}
}

func TestKVStore_BatchDeleteCAS(t *testing.T) {
	store := NewKVStore()

	// Create initial keys
	store.Set("key1", "value1")
	store.Set("key2", "value2")
	store.Set("key3", "value3")
	entry1, _ := store.GetEntry("key1")
	entry2, _ := store.GetEntry("key2")
	entry3, _ := store.GetEntry("key3")

	// Test batch delete CAS with one wrong index fails
	keys := []string{"key1", "key2", "key3"}
	expectedIndices := map[string]uint64{
		"key1": entry1.ModifyIndex,
		"key2": entry2.ModifyIndex + 999, // Wrong index
		"key3": entry3.ModifyIndex,
	}

	err := store.BatchDeleteCAS(keys, expectedIndices)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify NO keys were deleted (atomic operation)
	for _, key := range keys {
		if _, ok := store.Get(key); !ok {
			t.Fatalf("Key %s should still exist", key)
		}
	}

	// Test successful batch delete CAS
	expectedIndices["key2"] = entry2.ModifyIndex // Fix the index
	err = store.BatchDeleteCAS(keys, expectedIndices)
	if err != nil {
		t.Fatalf("Expected batch delete CAS to succeed, got error: %v", err)
	}

	// Verify all keys were deleted
	for _, key := range keys {
		if _, ok := store.Get(key); ok {
			t.Fatalf("Key %s should be deleted", key)
		}
	}
}

func TestKVStore_CAS_Concurrency(t *testing.T) {
	store := NewKVStore()

	// Create initial key
	store.Set("counter", "0")

	// Try to concurrently update the same key with CAS
	// Only one goroutine should succeed per iteration
	iterations := 10
	goroutines := 5
	successCount := 0
	conflictCount := 0
	var mu sync.Mutex

	for i := 0; i < iterations; i++ {
		entry, _ := store.GetEntry("counter")
		currentIndex := entry.ModifyIndex
		currentValue := entry.Value

		var wg sync.WaitGroup
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				newValue := fmt.Sprintf("%s-%d", currentValue, goroutineID)
				_, err := store.SetCAS("counter", newValue, currentIndex)
				mu.Lock()
				if err == nil {
					successCount++
				} else if IsCASConflict(err) {
					conflictCount++
				}
				mu.Unlock()
			}(g)
		}
		wg.Wait()
	}

	// Exactly one goroutine should succeed per iteration
	if successCount != iterations {
		t.Fatalf("Expected %d successful updates, got %d", iterations, successCount)
	}
	expectedConflicts := (goroutines * iterations) - iterations
	if conflictCount != expectedConflicts {
		t.Fatalf("Expected %d conflicts, got %d", expectedConflicts, conflictCount)
	}

	t.Logf("Concurrency test passed: %d successes, %d conflicts", successCount, conflictCount)
}

func TestKVStore_CAS_IndexMonotonicity(t *testing.T) {
	store := NewKVStore()

	// Verify indices are monotonically increasing
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	var lastIndex uint64 = 0

	for _, key := range keys {
		store.Set(key, "value")
		entry, _ := store.GetEntry(key)
		if entry.ModifyIndex <= lastIndex {
			t.Fatalf("Expected monotonically increasing indices, got %d after %d",
				entry.ModifyIndex, lastIndex)
		}
		lastIndex = entry.ModifyIndex
	}

	// Update a key and verify index increases
	entry, _ := store.GetEntry("key2")
	oldIndex := entry.ModifyIndex
	store.Set("key2", "new-value")
	entry, _ = store.GetEntry("key2")
	if entry.ModifyIndex <= oldIndex {
		t.Fatalf("Expected index to increase on update, got %d (was %d)",
			entry.ModifyIndex, oldIndex)
	}
	if entry.ModifyIndex <= lastIndex {
		t.Fatalf("Expected index to be greater than all previous indices")
	}
}

func TestKVStore_CAS_PreservesCreateIndex(t *testing.T) {
	store := NewKVStore()

	// Create a key
	store.Set("test-key", "value1")
	entry, _ := store.GetEntry("test-key")
	originalCreateIndex := entry.CreateIndex
	originalModifyIndex := entry.ModifyIndex

	if originalCreateIndex != originalModifyIndex {
		t.Fatalf("Expected CreateIndex == ModifyIndex on creation, got %d != %d",
			originalCreateIndex, originalModifyIndex)
	}

	// Update the key multiple times
	for i := 0; i < 3; i++ {
		newIndex, err := store.SetCAS("test-key", fmt.Sprintf("value%d", i+2), entry.ModifyIndex)
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}
		entry, _ = store.GetEntry("test-key")
		if entry.CreateIndex != originalCreateIndex {
			t.Fatalf("CreateIndex changed from %d to %d on update %d",
				originalCreateIndex, entry.CreateIndex, i)
		}
		if entry.ModifyIndex != newIndex {
			t.Fatalf("ModifyIndex mismatch: expected %d, got %d", newIndex, entry.ModifyIndex)
		}
	}
}

func TestKVStore_CAS_PreservesFlags(t *testing.T) {
	store := NewKVStore()

	// Create a key with flags
	store.SetWithFlags("test-key", "value1", 42)
	entry, _ := store.GetEntry("test-key")
	if entry.Flags != 42 {
		t.Fatalf("Expected flags=42, got %d", entry.Flags)
	}

	// Update with CAS and verify flags are preserved
	_, err := store.SetCAS("test-key", "value2", entry.ModifyIndex)
	if err != nil {
		t.Fatalf("CAS update failed: %v", err)
	}

	entry, _ = store.GetEntry("test-key")
	if entry.Flags != 42 {
		t.Fatalf("Expected flags=42 after update, got %d", entry.Flags)
	}
	if entry.Value != "value2" {
		t.Fatalf("Expected value2, got %s", entry.Value)
	}
}
