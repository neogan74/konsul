package store

import (
	"sync"
	"testing"
	"time"
)

func TestServiceStore_RegisterCAS_CreateOnly(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Tags:    []string{"tag1"},
	}

	// Test create-only with expectedIndex=0
	newIndex, err := store.RegisterCAS(service, 0)
	if err != nil {
		t.Fatalf("Expected create to succeed, got error: %v", err)
	}
	if newIndex == 0 {
		t.Fatalf("Expected non-zero index, got 0")
	}

	// Test create-only fails when service exists
	_, err = store.RegisterCAS(service, 0)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify service wasn't changed
	svc, ok := store.Get("test-service")
	if !ok {
		t.Fatal("Service should exist")
	}
	if len(svc.Tags) != 1 || svc.Tags[0] != "tag1" {
		t.Fatalf("Service tags should not have changed")
	}
}

func TestServiceStore_RegisterCAS_Update(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service1 := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Tags:    []string{"v1"},
	}

	// Register initial service
	err := store.Register(service1)
	if err != nil {
		t.Fatalf("Initial registration failed: %v", err)
	}

	entry, ok := store.GetEntry("test-service")
	if !ok {
		t.Fatal("Service should exist")
	}
	initialIndex := entry.ModifyIndex

	// Update service with CAS
	service2 := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    9090,
		Tags:    []string{"v2"},
	}

	newIndex, err := store.RegisterCAS(service2, initialIndex)
	if err != nil {
		t.Fatalf("Expected update to succeed, got error: %v", err)
	}
	if newIndex <= initialIndex {
		t.Fatalf("Expected new index (%d) > initial index (%d)", newIndex, initialIndex)
	}

	// Verify service was updated
	svc, ok := store.Get("test-service")
	if !ok {
		t.Fatal("Service should exist")
	}
	if svc.Port != 9090 {
		t.Fatalf("Expected port 9090, got %d", svc.Port)
	}
	if len(svc.Tags) != 1 || svc.Tags[0] != "v2" {
		t.Fatalf("Expected tags [v2], got %v", svc.Tags)
	}

	// Test CAS fails with old index
	service3 := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    7070,
	}
	_, err = store.RegisterCAS(service3, initialIndex)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify service wasn't changed
	svc, ok = store.Get("test-service")
	if !ok || svc.Port != 9090 {
		t.Fatal("Service should not have been updated")
	}
}

func TestServiceStore_RegisterCAS_NotFound(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service := Service{
		Name:    "missing-service",
		Address: "localhost",
		Port:    8080,
	}

	// Test CAS fails for non-existent service with non-zero index
	_, err := store.RegisterCAS(service, 123)
	if err == nil {
		t.Fatal("Expected NotFoundError, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got %T: %v", err, err)
	}
}

func TestServiceStore_DeregisterCAS(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
	}

	// Register service
	err := store.Register(service)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	entry, _ := store.GetEntry("test-service")
	correctIndex := entry.ModifyIndex

	// Test deregister with wrong index fails
	err = store.DeregisterCAS("test-service", correctIndex+1)
	if err == nil {
		t.Fatal("Expected CAS conflict, got nil")
	}
	if !IsCASConflict(err) {
		t.Fatalf("Expected CASConflictError, got %T: %v", err, err)
	}

	// Verify service still exists
	_, ok := store.Get("test-service")
	if !ok {
		t.Fatal("Service should still exist")
	}

	// Test deregister with correct index succeeds
	err = store.DeregisterCAS("test-service", correctIndex)
	if err != nil {
		t.Fatalf("Expected deregister to succeed, got error: %v", err)
	}

	// Verify service is deleted
	_, ok = store.Get("test-service")
	if ok {
		t.Fatal("Service should be deleted")
	}

	// Test deregister of non-existent service fails
	err = store.DeregisterCAS("test-service", correctIndex)
	if err == nil {
		t.Fatal("Expected NotFoundError, got nil")
	}
	if !IsNotFound(err) {
		t.Fatalf("Expected NotFoundError, got %T: %v", err, err)
	}
}

func TestServiceStore_CAS_Concurrency(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service := Service{
		Name:    "concurrent-service",
		Address: "localhost",
		Port:    8080,
	}

	// Register initial service
	err := store.Register(service)
	if err != nil {
		t.Fatalf("Initial registration failed: %v", err)
	}

	// Try to concurrently update the same service with CAS
	// Only one goroutine should succeed per iteration
	iterations := 10
	goroutines := 5
	successCount := 0
	conflictCount := 0
	var mu sync.Mutex

	for i := 0; i < iterations; i++ {
		entry, _ := store.GetEntry("concurrent-service")
		currentIndex := entry.ModifyIndex

		var wg sync.WaitGroup
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				svc := Service{
					Name:    "concurrent-service",
					Address: "localhost",
					Port:    8080 + goroutineID,
				}
				_, err := store.RegisterCAS(svc, currentIndex)
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

func TestServiceStore_CAS_IndexMonotonicity(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	// Verify indices are monotonically increasing
	services := []Service{
		{Name: "svc1", Address: "localhost", Port: 8081},
		{Name: "svc2", Address: "localhost", Port: 8082},
		{Name: "svc3", Address: "localhost", Port: 8083},
		{Name: "svc4", Address: "localhost", Port: 8084},
		{Name: "svc5", Address: "localhost", Port: 8085},
	}

	var lastIndex uint64 = 0

	for _, svc := range services {
		err := store.Register(svc)
		if err != nil {
			t.Fatalf("Registration of %s failed: %v", svc.Name, err)
		}
		entry, _ := store.GetEntry(svc.Name)
		if entry.ModifyIndex <= lastIndex {
			t.Fatalf("Expected monotonically increasing indices, got %d after %d",
				entry.ModifyIndex, lastIndex)
		}
		lastIndex = entry.ModifyIndex
	}

	// Update a service and verify index increases
	entry, _ := store.GetEntry("svc2")
	oldIndex := entry.ModifyIndex

	updatedSvc := Service{Name: "svc2", Address: "localhost", Port: 9999}
	err := store.Register(updatedSvc)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	entry, _ = store.GetEntry("svc2")
	if entry.ModifyIndex <= oldIndex {
		t.Fatalf("Expected index to increase on update, got %d (was %d)",
			entry.ModifyIndex, oldIndex)
	}
	if entry.ModifyIndex <= lastIndex {
		t.Fatalf("Expected index to be greater than all previous indices")
	}
}

func TestServiceStore_CAS_PreservesCreateIndex(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
	}

	// Register initial service
	err := store.Register(service)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	entry, _ := store.GetEntry("test-service")
	originalCreateIndex := entry.CreateIndex
	originalModifyIndex := entry.ModifyIndex

	if originalCreateIndex != originalModifyIndex {
		t.Fatalf("Expected CreateIndex == ModifyIndex on creation, got %d != %d",
			originalCreateIndex, originalModifyIndex)
	}

	// Update the service multiple times
	for i := 0; i < 3; i++ {
		updatedService := Service{
			Name:    "test-service",
			Address: "localhost",
			Port:    8080 + i + 1,
		}
		newIndex, err := store.RegisterCAS(updatedService, entry.ModifyIndex)
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}
		entry, _ = store.GetEntry("test-service")
		if entry.CreateIndex != originalCreateIndex {
			t.Fatalf("CreateIndex changed from %d to %d on update %d",
				originalCreateIndex, entry.CreateIndex, i)
		}
		if entry.ModifyIndex != newIndex {
			t.Fatalf("ModifyIndex mismatch: expected %d, got %d", newIndex, entry.ModifyIndex)
		}
	}
}

func TestServiceStore_CAS_HeartbeatPreservesIndex(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
	}

	// Register service
	err := store.Register(service)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	entry, _ := store.GetEntry("test-service")
	initialModifyIndex := entry.ModifyIndex
	initialCreateIndex := entry.CreateIndex

	// Send heartbeat
	success := store.Heartbeat("test-service")
	if !success {
		t.Fatal("Heartbeat should succeed")
	}

	// Verify indices are preserved
	entry, _ = store.GetEntry("test-service")
	if entry.ModifyIndex != initialModifyIndex {
		t.Fatalf("Heartbeat should not change ModifyIndex, expected %d, got %d",
			initialModifyIndex, entry.ModifyIndex)
	}
	if entry.CreateIndex != initialCreateIndex {
		t.Fatalf("Heartbeat should not change CreateIndex, expected %d, got %d",
			initialCreateIndex, entry.CreateIndex)
	}
}

func TestServiceStore_CAS_TagAndMetaIndexes(t *testing.T) {
	store := NewServiceStoreWithTTL(30 * time.Second)

	service1 := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Tags:    []string{"tag1", "tag2"},
		Meta:    map[string]string{"env": "dev"},
	}

	// Register with tags and metadata
	err := store.Register(service1)
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	entry, _ := store.GetEntry("test-service")

	// Update with different tags using CAS
	service2 := Service{
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Tags:    []string{"tag3"},
		Meta:    map[string]string{"env": "prod"},
	}

	newIndex, err := store.RegisterCAS(service2, entry.ModifyIndex)
	if err != nil {
		t.Fatalf("CAS update failed: %v", err)
	}

	// Verify service was updated
	svc, ok := store.Get("test-service")
	if !ok {
		t.Fatal("Service should exist")
	}
	if len(svc.Tags) != 1 || svc.Tags[0] != "tag3" {
		t.Fatalf("Expected tags [tag3], got %v", svc.Tags)
	}
	if svc.Meta["env"] != "prod" {
		t.Fatalf("Expected meta env=prod, got %v", svc.Meta["env"])
	}

	// Verify ModifyIndex was updated
	entry, _ = store.GetEntry("test-service")
	if entry.ModifyIndex != newIndex {
		t.Fatalf("Expected ModifyIndex %d, got %d", newIndex, entry.ModifyIndex)
	}
}
