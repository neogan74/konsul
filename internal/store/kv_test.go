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