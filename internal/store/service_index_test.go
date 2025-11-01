package store

import (
	"testing"
)

func TestAddToTagIndex(t *testing.T) {
	store := NewServiceStore()

	tests := []struct {
		name        string
		serviceName string
		tags        []string
	}{
		{
			name:        "add single tag",
			serviceName: "service1",
			tags:        []string{"env:production"},
		},
		{
			name:        "add multiple tags",
			serviceName: "service2",
			tags:        []string{"env:production", "http", "version:v1"},
		},
		{
			name:        "add empty tags",
			serviceName: "service3",
			tags:        []string{},
		},
		{
			name:        "add nil tags",
			serviceName: "service4",
			tags:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.addToTagIndex(tt.serviceName, tt.tags)

			// Verify each tag has the service in its index
			for _, tag := range tt.tags {
				if services, ok := store.TagIndex[tag]; !ok {
					t.Errorf("Tag %q not found in TagIndex", tag)
				} else if !services[tt.serviceName] {
					t.Errorf("Service %q not found in TagIndex[%q]", tt.serviceName, tag)
				}
			}
		})
	}
}

func TestRemoveFromTagIndex(t *testing.T) {
	store := NewServiceStore()

	// Setup: Add services to index
	store.addToTagIndex("service1", []string{"env:production", "http"})
	store.addToTagIndex("service2", []string{"env:production", "grpc"})
	store.addToTagIndex("service3", []string{"env:staging", "http"})

	tests := []struct {
		name        string
		serviceName string
		tags        []string
	}{
		{
			name:        "remove single tag",
			serviceName: "service1",
			tags:        []string{"http"},
		},
		{
			name:        "remove all tags for service",
			serviceName: "service2",
			tags:        []string{"env:production", "grpc"},
		},
		{
			name:        "remove empty tags",
			serviceName: "service3",
			tags:        []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.removeFromTagIndex(tt.serviceName, tt.tags)

			// Verify each tag no longer has the service
			for _, tag := range tt.tags {
				if services, ok := store.TagIndex[tag]; ok {
					if services[tt.serviceName] {
						t.Errorf("Service %q still found in TagIndex[%q] after removal", tt.serviceName, tag)
					}
				}
			}
		})
	}

	// Verify cleanup: tags with no services should be removed
	if _, ok := store.TagIndex["grpc"]; ok {
		t.Error("Tag 'grpc' should be removed from index when last service is removed")
	}
}

func TestAddToMetaIndex(t *testing.T) {
	store := NewServiceStore()

	tests := []struct {
		name        string
		serviceName string
		meta        map[string]string
	}{
		{
			name:        "add single metadata",
			serviceName: "service1",
			meta: map[string]string{
				"team": "platform",
			},
		},
		{
			name:        "add multiple metadata",
			serviceName: "service2",
			meta: map[string]string{
				"team":  "platform",
				"owner": "alice@example.com",
			},
		},
		{
			name:        "add empty metadata",
			serviceName: "service3",
			meta:        map[string]string{},
		},
		{
			name:        "add nil metadata",
			serviceName: "service4",
			meta:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.addToMetaIndex(tt.serviceName, tt.meta)

			// Verify each metadata key-value has the service in its index
			for key, value := range tt.meta {
				if values, ok := store.MetaIndex[key]; !ok {
					t.Errorf("Metadata key %q not found in MetaIndex", key)
				} else if services, ok := values[value]; !ok {
					t.Errorf("Metadata value %q not found in MetaIndex[%q]", value, key)
				} else {
					found := false
					for _, svc := range services {
						if svc == tt.serviceName {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Service %q not found in MetaIndex[%q][%q]", tt.serviceName, key, value)
					}
				}
			}
		})
	}
}

func TestRemoveFromMetaIndex(t *testing.T) {
	store := NewServiceStore()

	// Setup: Add services to metadata index
	store.addToMetaIndex("service1", map[string]string{"team": "platform", "owner": "alice@example.com"})
	store.addToMetaIndex("service2", map[string]string{"team": "platform", "owner": "bob@example.com"})
	store.addToMetaIndex("service3", map[string]string{"team": "data", "owner": "charlie@example.com"})

	tests := []struct {
		name        string
		serviceName string
		meta        map[string]string
	}{
		{
			name:        "remove single metadata",
			serviceName: "service1",
			meta: map[string]string{
				"owner": "alice@example.com",
			},
		},
		{
			name:        "remove all metadata for service",
			serviceName: "service3",
			meta: map[string]string{
				"team":  "data",
				"owner": "charlie@example.com",
			},
		},
		{
			name:        "remove empty metadata",
			serviceName: "service2",
			meta:        map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.removeFromMetaIndex(tt.serviceName, tt.meta)

			// Verify each metadata key-value no longer has the service
			for key, value := range tt.meta {
				if values, ok := store.MetaIndex[key]; ok {
					if services, ok := values[value]; ok {
						for _, svc := range services {
							if svc == tt.serviceName {
								t.Errorf("Service %q still found in MetaIndex[%q][%q] after removal",
									tt.serviceName, key, value)
							}
						}
					}
				}
			}
		})
	}

	// Verify cleanup: metadata entries with no services should be removed
	if values, ok := store.MetaIndex["team"]; ok {
		if _, ok := values["data"]; ok {
			t.Error("Metadata value 'data' for key 'team' should be removed when last service is removed")
		}
	}

	if values, ok := store.MetaIndex["owner"]; ok {
		if _, ok := values["charlie@example.com"]; ok {
			t.Error("Metadata value 'charlie@example.com' should be removed when last service is removed")
		}
	}
}

func TestRegisterWithIndexing(t *testing.T) {
	store := NewServiceStore()

	service := Service{
		Name:    "api-service",
		Address: "10.0.1.1",
		Port:    8080,
		Tags:    []string{"env:production", "http"},
		Meta: map[string]string{
			"team":  "platform",
			"owner": "alice@example.com",
		},
	}

	// Register service
	if err := store.Register(service); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Verify tag index
	for _, tag := range service.Tags {
		if services, ok := store.TagIndex[tag]; !ok {
			t.Errorf("Tag %q not found in TagIndex after registration", tag)
		} else if !services[service.Name] {
			t.Errorf("Service %q not found in TagIndex[%q] after registration", service.Name, tag)
		}
	}

	// Verify metadata index
	for key, value := range service.Meta {
		if values, ok := store.MetaIndex[key]; !ok {
			t.Errorf("Metadata key %q not found in MetaIndex after registration", key)
		} else if services, ok := values[value]; !ok {
			t.Errorf("Metadata value %q not found in MetaIndex[%q] after registration", value, key)
		} else {
			found := false
			for _, svc := range services {
				if svc == service.Name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Service %q not found in MetaIndex[%q][%q] after registration",
					service.Name, key, value)
			}
		}
	}
}

func TestReregisterWithIndexUpdate(t *testing.T) {
	store := NewServiceStore()

	// Register service with initial tags/metadata
	service := Service{
		Name:    "api-service",
		Address: "10.0.1.1",
		Port:    8080,
		Tags:    []string{"env:production", "http"},
		Meta: map[string]string{
			"team": "platform",
		},
	}

	if err := store.Register(service); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Re-register with updated tags/metadata
	updatedService := Service{
		Name:    "api-service",
		Address: "10.0.1.1",
		Port:    8080,
		Tags:    []string{"env:staging", "grpc"}, // Different tags
		Meta: map[string]string{
			"team": "data", // Different metadata
		},
	}

	if err := store.Register(updatedService); err != nil {
		t.Fatalf("Failed to re-register service: %v", err)
	}

	// Verify old tags are removed
	if services, ok := store.TagIndex["env:production"]; ok {
		if services[service.Name] {
			t.Error("Service still has old tag 'env:production' after re-registration")
		}
	}
	if services, ok := store.TagIndex["http"]; ok {
		if services[service.Name] {
			t.Error("Service still has old tag 'http' after re-registration")
		}
	}

	// Verify new tags are added
	if services, ok := store.TagIndex["env:staging"]; !ok {
		t.Error("New tag 'env:staging' not found in index")
	} else if !services[service.Name] {
		t.Error("Service not found in new tag 'env:staging' index")
	}

	// Verify old metadata is removed
	if values, ok := store.MetaIndex["team"]; ok {
		if services, ok := values["platform"]; ok {
			for _, svc := range services {
				if svc == service.Name {
					t.Error("Service still has old metadata team:platform after re-registration")
				}
			}
		}
	}

	// Verify new metadata is added
	if values, ok := store.MetaIndex["team"]; !ok {
		t.Error("Metadata key 'team' not found in index")
	} else if services, ok := values["data"]; !ok {
		t.Error("New metadata value 'data' not found for key 'team'")
	} else {
		found := false
		for _, svc := range services {
			if svc == service.Name {
				found = true
				break
			}
		}
		if !found {
			t.Error("Service not found in new metadata team:data index")
		}
	}
}

func TestDeregisterRemovesFromIndexes(t *testing.T) {
	store := NewServiceStore()

	service := Service{
		Name:    "api-service",
		Address: "10.0.1.1",
		Port:    8080,
		Tags:    []string{"env:production", "http"},
		Meta: map[string]string{
			"team": "platform",
		},
	}

	// Register and then deregister
	if err := store.Register(service); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	store.Deregister(service.Name)

	// Verify tags are removed
	for _, tag := range service.Tags {
		if services, ok := store.TagIndex[tag]; ok {
			if services[service.Name] {
				t.Errorf("Service still found in TagIndex[%q] after deregistration", tag)
			}
		}
	}

	// Verify metadata is removed
	for key, value := range service.Meta {
		if values, ok := store.MetaIndex[key]; ok {
			if services, ok := values[value]; ok {
				for _, svc := range services {
					if svc == service.Name {
						t.Errorf("Service still found in MetaIndex[%q][%q] after deregistration", key, value)
					}
				}
			}
		}
	}
}

func TestCleanupExpiredRemovesFromIndexes(t *testing.T) {
	store := NewServiceStoreWithTTL(50 * time.Millisecond)

	service := Service{
		Name:    "api-service",
		Address: "10.0.1.1",
		Port:    8080,
		Tags:    []string{"env:production"},
		Meta: map[string]string{
			"team": "platform",
		},
	}

	// Register service
	if err := store.Register(service); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Cleanup expired services
	count := store.CleanupExpired()
	if count != 1 {
		t.Errorf("CleanupExpired() removed %d services, want 1", count)
	}

	// Verify tags are removed
	for _, tag := range service.Tags {
		if services, ok := store.TagIndex[tag]; ok {
			if services[service.Name] {
				t.Errorf("Expired service still found in TagIndex[%q] after cleanup", tag)
			}
		}
	}

	// Verify metadata is removed
	for key, value := range service.Meta {
		if values, ok := store.MetaIndex[key]; ok {
			if services, ok := values[value]; ok {
				for _, svc := range services {
					if svc == service.Name {
						t.Errorf("Expired service still found in MetaIndex[%q][%q] after cleanup", key, value)
					}
				}
			}
		}
	}
}
