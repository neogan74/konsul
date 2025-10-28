package store

import (
	"fmt"
	"sync"
	"testing"
)

func TestKVStore_NewKVStore(t *testing.T) {
	kv := NewKVStore()
	if kv == nil {
		t.Fatal("expected NewKVStore to return non-nil store")
	}
	if kv.Data == nil {
		t.Error("expected Data map to be initialized")
	}
	if len(kv.Data) != 0 {
		t.Error("expected new store to be empty")
	}
}

func TestKVStore_SetAndGet(t *testing.T) {
	kv := NewKVStore()
	key := "test-key"
	value := "test-value"

	// Test getting non-existent key
	_, ok := kv.Get(key)
	if ok {
		t.Error("expected Get to return false for non-existent key")
	}

	// Set key-value pair
	kv.Set(key, value)

	// Get the value back
	got, ok := kv.Get(key)
	if !ok {
		t.Fatal("expected Get to return true for existing key")
	}
	if got != value {
		t.Errorf("expected %q, got %q", value, got)
	}
}

func TestKVStore_SetOverwrite(t *testing.T) {
	kv := NewKVStore()
	key := "test-key"
	value1 := "first-value"
	value2 := "second-value"

	// Set initial value
	kv.Set(key, value1)
	got, ok := kv.Get(key)
	if !ok || got != value1 {
		t.Fatalf("expected initial value %q, got %q", value1, got)
	}

	// Overwrite with new value
	kv.Set(key, value2)
	got, ok = kv.Get(key)
	if !ok || got != value2 {
		t.Errorf("expected overwritten value %q, got %q", value2, got)
	}
}

func TestKVStore_Delete(t *testing.T) {
	kv := NewKVStore()
	key := "test-key"
	value := "test-value"

	// Set a key-value pair
	kv.Set(key, value)
	_, ok := kv.Get(key)
	if !ok {
		t.Fatal("expected key to exist after Set")
	}

	// Delete the key
	kv.Delete(key)
	_, ok = kv.Get(key)
	if ok {
		t.Error("expected key to not exist after Delete")
	}
}

func TestKVStore_DeleteNonExistent(t *testing.T) {
	kv := NewKVStore()

	// Delete non-existent key should not panic
	kv.Delete("non-existent")

	// Should still be able to use the store
	kv.Set("test", "value")
	got, ok := kv.Get("test")
	if !ok || got != "value" {
		t.Error("store should still work after deleting non-existent key")
	}
}

func TestKVStore_MultipleKeys(t *testing.T) {
	kv := NewKVStore()
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
	}

	// Set multiple key-value pairs
	for key, value := range testData {
		kv.Set(key, value)
	}

	// Verify all keys exist with correct values
	for key, expectedValue := range testData {
		got, ok := kv.Get(key)
		if !ok {
			t.Errorf("expected key %q to exist", key)
			continue
		}
		if got != expectedValue {
			t.Errorf("for key %q: expected %q, got %q", key, expectedValue, got)
		}
	}

	// Delete one key
	delete(testData, "key2")
	kv.Delete("key2")

	// Verify deleted key doesn't exist
	_, ok := kv.Get("key2")
	if ok {
		t.Error("expected key2 to not exist after deletion")
	}

	// Verify other keys still exist
	for key, expectedValue := range testData {
		got, ok := kv.Get(key)
		if !ok {
			t.Errorf("expected key %q to still exist after deleting key2", key)
			continue
		}
		if got != expectedValue {
			t.Errorf("for key %q: expected %q, got %q", key, expectedValue, got)
		}
	}
}

func TestKVStore_EmptyValues(t *testing.T) {
	kv := NewKVStore()

	// Test empty string value
	key := "empty-key"
	kv.Set(key, "")
	got, ok := kv.Get(key)
	if !ok {
		t.Error("expected empty value to be stored and retrievable")
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestKVStore_SpecialCharacters(t *testing.T) {
	kv := NewKVStore()

	testCases := []struct {
		key   string
		value string
	}{
		{"key with spaces", "value with spaces"},
		{"key/with/slashes", "value/with/slashes"},
		{"key-with-dashes", "value-with-dashes"},
		{"key_with_underscores", "value_with_underscores"},
		{"key.with.dots", "value.with.dots"},
		{"key@with@symbols", "value@with@symbols"},
		{"unicode-key-🔑", "unicode-value-🎯"},
		{"", "empty-key-test"},
	}

	for _, tc := range testCases {
		kv.Set(tc.key, tc.value)
		got, ok := kv.Get(tc.key)
		if !ok {
			t.Errorf("expected key %q to exist", tc.key)
			continue
		}
		if got != tc.value {
			t.Errorf("for key %q: expected %q, got %q", tc.key, tc.value, got)
		}
	}
}

func TestKVStore_ConcurrentAccess(t *testing.T) {
	kv := NewKVStore()
	const numGoroutines = 100
	const numOperations = 100

	var wg sync.WaitGroup

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				kv.Set(key, value)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				expectedValue := fmt.Sprintf("value-%d-%d", id, j)
				got, ok := kv.Get(key)
				if !ok {
					t.Errorf("expected key %q to exist", key)
					return
				}
				if got != expectedValue {
					t.Errorf("for key %q: expected %q, got %q", key, expectedValue, got)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	// Mixed concurrent operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/10; j++ {
				key := fmt.Sprintf("mixed-key-%d-%d", id, j)
				value := fmt.Sprintf("mixed-value-%d-%d", id, j)

				// Set
				kv.Set(key, value)

				// Get
				got, ok := kv.Get(key)
				if ok && got != value {
					t.Errorf("concurrent operation failed: expected %q, got %q", value, got)
				}

				// Delete
				kv.Delete(key)

				// Verify deleted
				_, ok = kv.Get(key)
				if ok {
					t.Error("key should not exist after deletion")
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestKVStore_List(t *testing.T) {
	kv := NewKVStore()

	// Test empty list
	keys := kv.List()
	if len(keys) != 0 {
		t.Errorf("expected empty list, got %d keys", len(keys))
	}

	// Add some keys
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range testData {
		kv.Set(key, value)
	}

	// List should return all keys
	keys = kv.List()
	if len(keys) != len(testData) {
		t.Errorf("expected %d keys, got %d", len(testData), len(keys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	for key := range testData {
		if !keyMap[key] {
			t.Errorf("expected key %q to be in list", key)
		}
	}
}

func TestKVStore_BatchSet(t *testing.T) {
	kv := NewKVStore()

	// Test batch set
	items := map[string]string{
		"batch1": "value1",
		"batch2": "value2",
		"batch3": "value3",
		"batch4": "value4",
	}

	err := kv.BatchSet(items)
	if err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	// Verify all items were set
	for key, expectedValue := range items {
		got, ok := kv.Get(key)
		if !ok {
			t.Errorf("expected key %q to exist", key)
			continue
		}
		if got != expectedValue {
			t.Errorf("for key %q: expected %q, got %q", key, expectedValue, got)
		}
	}
}

func TestKVStore_BatchSetEmpty(t *testing.T) {
	kv := NewKVStore()

	// Test batch set with empty map
	err := kv.BatchSet(map[string]string{})
	if err != nil {
		t.Fatalf("BatchSet with empty map failed: %v", err)
	}

	// Store should still be empty
	keys := kv.List()
	if len(keys) != 0 {
		t.Errorf("expected empty store, got %d keys", len(keys))
	}
}

func TestKVStore_BatchSetOverwrite(t *testing.T) {
	kv := NewKVStore()

	// Set initial values
	kv.Set("key1", "old-value1")
	kv.Set("key2", "old-value2")

	// Batch set should overwrite
	items := map[string]string{
		"key1": "new-value1",
		"key2": "new-value2",
		"key3": "new-value3",
	}

	err := kv.BatchSet(items)
	if err != nil {
		t.Fatalf("BatchSet failed: %v", err)
	}

	// Verify all values were updated
	for key, expectedValue := range items {
		got, ok := kv.Get(key)
		if !ok {
			t.Errorf("expected key %q to exist", key)
			continue
		}
		if got != expectedValue {
			t.Errorf("for key %q: expected %q, got %q", key, expectedValue, got)
		}
	}
}

func TestKVStore_BatchDelete(t *testing.T) {
	kv := NewKVStore()

	// Set up test data
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	for key, value := range testData {
		kv.Set(key, value)
	}

	// Delete some keys in batch
	keysToDelete := []string{"key1", "key3", "key5"}
	err := kv.BatchDelete(keysToDelete)
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	// Verify deleted keys don't exist
	for _, key := range keysToDelete {
		_, ok := kv.Get(key)
		if ok {
			t.Errorf("expected key %q to be deleted", key)
		}
	}

	// Verify remaining keys still exist
	remainingKeys := []string{"key2", "key4"}
	for _, key := range remainingKeys {
		got, ok := kv.Get(key)
		if !ok {
			t.Errorf("expected key %q to still exist", key)
			continue
		}
		if got != testData[key] {
			t.Errorf("for key %q: expected %q, got %q", key, testData[key], got)
		}
	}
}

func TestKVStore_BatchDeleteEmpty(t *testing.T) {
	kv := NewKVStore()

	// Test batch delete with empty slice
	err := kv.BatchDelete([]string{})
	if err != nil {
		t.Fatalf("BatchDelete with empty slice failed: %v", err)
	}
}

func TestKVStore_BatchDeleteNonExistent(t *testing.T) {
	kv := NewKVStore()

	// Set some data
	kv.Set("key1", "value1")
	kv.Set("key2", "value2")

	// Delete non-existent keys (should not error)
	err := kv.BatchDelete([]string{"nonexistent1", "nonexistent2"})
	if err != nil {
		t.Fatalf("BatchDelete of non-existent keys failed: %v", err)
	}

	// Verify existing keys are still there
	got, ok := kv.Get("key1")
	if !ok || got != "value1" {
		t.Error("existing keys should not be affected")
	}
}

func TestKVStore_Close(t *testing.T) {
	kv := NewKVStore()

	// Close should not error even without persistence
	err := kv.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Store should still be usable after close (no persistence)
	kv.Set("test", "value")
	got, ok := kv.Get("test")
	if !ok || got != "value" {
		t.Error("store should still work after close when no persistence")
	}
}

func TestKVStore_ConcurrentBatchOperations(t *testing.T) {
	kv := NewKVStore()
	const numGoroutines = 50

	var wg sync.WaitGroup

	// Concurrent batch sets
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			items := make(map[string]string)
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("batch-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				items[key] = value
			}
			err := kv.BatchSet(items)
			if err != nil {
				t.Errorf("BatchSet failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all keys were set
	keys := kv.List()
	expectedCount := numGoroutines * 10
	if len(keys) != expectedCount {
		t.Errorf("expected %d keys, got %d", expectedCount, len(keys))
	}

	// Concurrent batch deletes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			keysToDelete := make([]string, 0, 10)
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("batch-%d-%d", id, j)
				keysToDelete = append(keysToDelete, key)
			}
			err := kv.BatchDelete(keysToDelete)
			if err != nil {
				t.Errorf("BatchDelete failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// All keys should be deleted
	keys = kv.List()
	if len(keys) != 0 {
		t.Errorf("expected 0 keys after batch delete, got %d", len(keys))
	}
}

func TestKVStore_LargeValues(t *testing.T) {
	kv := NewKVStore()

	// Test with large values
	largeValue := string(make([]byte, 1024*1024)) // 1MB
	kv.Set("large-key", largeValue)

	got, ok := kv.Get("large-key")
	if !ok {
		t.Fatal("expected large value to be stored")
	}
	if len(got) != len(largeValue) {
		t.Errorf("expected value length %d, got %d", len(largeValue), len(got))
	}
}

func TestKVStore_ManyKeys(t *testing.T) {
	kv := NewKVStore()

	// Test with many keys
	const numKeys = 10000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		kv.Set(key, value)
	}

	// Verify all keys exist
	keys := kv.List()
	if len(keys) != numKeys {
		t.Errorf("expected %d keys, got %d", numKeys, len(keys))
	}

	// Spot check some values
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i*100)
		expectedValue := fmt.Sprintf("value-%d", i*100)
		got, ok := kv.Get(key)
		if !ok {
			t.Errorf("expected key %q to exist", key)
			continue
		}
		if got != expectedValue {
			t.Errorf("for key %q: expected %q, got %q", key, expectedValue, got)
		}
	}
}
