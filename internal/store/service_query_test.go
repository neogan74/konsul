package store

import (
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestQueryByTags(t *testing.T) {
	store := NewServiceStore()

	// Register test services
	services := []Service{
		{
			Name:    "api-prod-v1",
			Address: "10.0.1.1",
			Port:    8080,
			Tags:    []string{"env:production", "version:v1", "http"},
		},
		{
			Name:    "api-prod-v2",
			Address: "10.0.1.2",
			Port:    8080,
			Tags:    []string{"env:production", "version:v2", "http"},
		},
		{
			Name:    "api-staging-v1",
			Address: "10.0.2.1",
			Port:    8080,
			Tags:    []string{"env:staging", "version:v1", "http"},
		},
		{
			Name:    "db-prod",
			Address: "10.0.1.10",
			Port:    5432,
			Tags:    []string{"env:production", "database", "postgres"},
		},
		{
			Name:    "web-prod",
			Address: "10.0.1.20",
			Port:    80,
			Tags:    []string{"env:production", "web"},
		},
	}

	for _, svc := range services {
		if err := store.Register(svc); err != nil {
			t.Fatalf("Failed to register service %s: %v", svc.Name, err)
		}
	}

	tests := []struct {
		name         string
		tags         []string
		wantServices []string
	}{
		{
			name:         "single tag - production",
			tags:         []string{"env:production"},
			wantServices: []string{"api-prod-v1", "api-prod-v2", "db-prod", "web-prod"},
		},
		{
			name:         "single tag - staging",
			tags:         []string{"env:staging"},
			wantServices: []string{"api-staging-v1"},
		},
		{
			name:         "single tag - http",
			tags:         []string{"http"},
			wantServices: []string{"api-prod-v1", "api-prod-v2", "api-staging-v1"},
		},
		{
			name:         "multiple tags - production AND http",
			tags:         []string{"env:production", "http"},
			wantServices: []string{"api-prod-v1", "api-prod-v2"},
		},
		{
			name:         "multiple tags - production AND version:v1",
			tags:         []string{"env:production", "version:v1"},
			wantServices: []string{"api-prod-v1"},
		},
		{
			name:         "multiple tags - production AND database",
			tags:         []string{"env:production", "database"},
			wantServices: []string{"db-prod"},
		},
		{
			name:         "tag with no matches",
			tags:         []string{"nonexistent"},
			wantServices: []string{},
		},
		{
			name:         "multiple tags with no matches",
			tags:         []string{"env:production", "nonexistent"},
			wantServices: []string{},
		},
		{
			name:         "empty tags - returns all",
			tags:         []string{},
			wantServices: []string{"api-prod-v1", "api-prod-v2", "api-staging-v1", "db-prod", "web-prod"},
		},
		{
			name:         "nil tags - returns all",
			tags:         nil,
			wantServices: []string{"api-prod-v1", "api-prod-v2", "api-staging-v1", "db-prod", "web-prod"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.QueryByTags(tt.tags)
			gotNames := make([]string, len(got))
			for i, svc := range got {
				gotNames[i] = svc.Name
			}
			sort.Strings(gotNames)
			sort.Strings(tt.wantServices)

			if !reflect.DeepEqual(gotNames, tt.wantServices) {
				t.Errorf("QueryByTags() = %v, want %v", gotNames, tt.wantServices)
			}
		})
	}
}

func TestQueryByMetadata(t *testing.T) {
	store := NewServiceStore()

	// Register test services
	services := []Service{
		{
			Name:    "api-platform",
			Address: "10.0.1.1",
			Port:    8080,
			Meta: map[string]string{
				"team":        "platform",
				"owner":       "alice@example.com",
				"cost-center": "engineering",
			},
		},
		{
			Name:    "api-data",
			Address: "10.0.1.2",
			Port:    8080,
			Meta: map[string]string{
				"team":        "data",
				"owner":       "bob@example.com",
				"cost-center": "engineering",
			},
		},
		{
			Name:    "web-platform",
			Address: "10.0.1.3",
			Port:    80,
			Meta: map[string]string{
				"team":        "platform",
				"owner":       "alice@example.com",
				"cost-center": "product",
			},
		},
		{
			Name:    "db-platform",
			Address: "10.0.1.4",
			Port:    5432,
			Meta: map[string]string{
				"team":  "platform",
				"owner": "charlie@example.com",
			},
		},
	}

	for _, svc := range services {
		if err := store.Register(svc); err != nil {
			t.Fatalf("Failed to register service %s: %v", svc.Name, err)
		}
	}

	tests := []struct {
		name         string
		filters      map[string]string
		wantServices []string
	}{
		{
			name:         "single filter - team:platform",
			filters:      map[string]string{"team": "platform"},
			wantServices: []string{"api-platform", "web-platform", "db-platform"},
		},
		{
			name:         "single filter - team:data",
			filters:      map[string]string{"team": "data"},
			wantServices: []string{"api-data"},
		},
		{
			name:         "single filter - owner:alice",
			filters:      map[string]string{"owner": "alice@example.com"},
			wantServices: []string{"api-platform", "web-platform"},
		},
		{
			name:         "multiple filters - team:platform AND owner:alice",
			filters:      map[string]string{"team": "platform", "owner": "alice@example.com"},
			wantServices: []string{"api-platform", "web-platform"},
		},
		{
			name:         "multiple filters - team:platform AND cost-center:engineering",
			filters:      map[string]string{"team": "platform", "cost-center": "engineering"},
			wantServices: []string{"api-platform"},
		},
		{
			name:         "filter with no matches",
			filters:      map[string]string{"team": "nonexistent"},
			wantServices: []string{},
		},
		{
			name:         "multiple filters with no matches",
			filters:      map[string]string{"team": "platform", "owner": "nonexistent@example.com"},
			wantServices: []string{},
		},
		{
			name:         "empty filters - returns all",
			filters:      map[string]string{},
			wantServices: []string{"api-platform", "api-data", "web-platform", "db-platform"},
		},
		{
			name:         "nil filters - returns all",
			filters:      nil,
			wantServices: []string{"api-platform", "api-data", "web-platform", "db-platform"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.QueryByMetadata(tt.filters)
			gotNames := make([]string, len(got))
			for i, svc := range got {
				gotNames[i] = svc.Name
			}
			sort.Strings(gotNames)
			sort.Strings(tt.wantServices)

			if !reflect.DeepEqual(gotNames, tt.wantServices) {
				t.Errorf("QueryByMetadata() = %v, want %v", gotNames, tt.wantServices)
			}
		})
	}
}

func TestQueryByTagsAndMetadata(t *testing.T) {
	store := NewServiceStore()

	// Register test services
	services := []Service{
		{
			Name:    "api-prod-platform",
			Address: "10.0.1.1",
			Port:    8080,
			Tags:    []string{"env:production", "http"},
			Meta: map[string]string{
				"team":  "platform",
				"owner": "alice@example.com",
			},
		},
		{
			Name:    "api-prod-data",
			Address: "10.0.1.2",
			Port:    8080,
			Tags:    []string{"env:production", "http"},
			Meta: map[string]string{
				"team":  "data",
				"owner": "bob@example.com",
			},
		},
		{
			Name:    "api-staging-platform",
			Address: "10.0.2.1",
			Port:    8080,
			Tags:    []string{"env:staging", "http"},
			Meta: map[string]string{
				"team":  "platform",
				"owner": "alice@example.com",
			},
		},
		{
			Name:    "db-prod-platform",
			Address: "10.0.1.10",
			Port:    5432,
			Tags:    []string{"env:production", "database"},
			Meta: map[string]string{
				"team":  "platform",
				"owner": "charlie@example.com",
			},
		},
	}

	for _, svc := range services {
		if err := store.Register(svc); err != nil {
			t.Fatalf("Failed to register service %s: %v", svc.Name, err)
		}
	}

	tests := []struct {
		name         string
		tags         []string
		meta         map[string]string
		wantServices []string
	}{
		{
			name:         "tags only - production",
			tags:         []string{"env:production"},
			meta:         nil,
			wantServices: []string{"api-prod-platform", "api-prod-data", "db-prod-platform"},
		},
		{
			name:         "metadata only - team:platform",
			tags:         nil,
			meta:         map[string]string{"team": "platform"},
			wantServices: []string{"api-prod-platform", "api-staging-platform", "db-prod-platform"},
		},
		{
			name:         "tags AND metadata - production AND platform",
			tags:         []string{"env:production"},
			meta:         map[string]string{"team": "platform"},
			wantServices: []string{"api-prod-platform", "db-prod-platform"},
		},
		{
			name:         "tags AND metadata - production AND http AND platform",
			tags:         []string{"env:production", "http"},
			meta:         map[string]string{"team": "platform"},
			wantServices: []string{"api-prod-platform"},
		},
		{
			name:         "tags AND metadata - production AND http AND platform AND alice",
			tags:         []string{"env:production", "http"},
			meta:         map[string]string{"team": "platform", "owner": "alice@example.com"},
			wantServices: []string{"api-prod-platform"},
		},
		{
			name:         "no match - tags match but metadata doesn't",
			tags:         []string{"env:production"},
			meta:         map[string]string{"team": "nonexistent"},
			wantServices: []string{},
		},
		{
			name:         "no match - metadata matches but tags don't",
			tags:         []string{"nonexistent"},
			meta:         map[string]string{"team": "platform"},
			wantServices: []string{},
		},
		{
			name:         "empty filters - returns all",
			tags:         []string{},
			meta:         map[string]string{},
			wantServices: []string{"api-prod-platform", "api-prod-data", "api-staging-platform", "db-prod-platform"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := store.QueryByTagsAndMetadata(tt.tags, tt.meta)
			gotNames := make([]string, len(got))
			for i, svc := range got {
				gotNames[i] = svc.Name
			}
			sort.Strings(gotNames)
			sort.Strings(tt.wantServices)

			if !reflect.DeepEqual(gotNames, tt.wantServices) {
				t.Errorf("QueryByTagsAndMetadata() = %v, want %v", gotNames, tt.wantServices)
			}
		})
	}
}

func TestQueryByTags_ExpiredServices(t *testing.T) {
	store := NewServiceStoreWithTTL(100 * time.Millisecond) // Very short TTL

	// Register test service
	service := Service{
		Name:    "api-service",
		Address: "10.0.1.1",
		Port:    8080,
		Tags:    []string{"env:production"},
	}

	if err := store.Register(service); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Should find the service initially
	got := store.QueryByTags([]string{"env:production"})
	if len(got) != 1 {
		t.Errorf("QueryByTags() before expiration = %d services, want 1", len(got))
	}

	// Wait for service to expire
	time.Sleep(150 * time.Millisecond)

	// Should not find expired service
	got = store.QueryByTags([]string{"env:production"})
	if len(got) != 0 {
		t.Errorf("QueryByTags() after expiration = %d services, want 0", len(got))
	}
}

func TestQueryByMetadata_ExpiredServices(t *testing.T) {
	store := NewServiceStoreWithTTL(100 * time.Millisecond) // Very short TTL

	// Register test service
	service := Service{
		Name:    "api-service",
		Address: "10.0.1.1",
		Port:    8080,
		Meta: map[string]string{
			"team": "platform",
		},
	}

	if err := store.Register(service); err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Should find the service initially
	got := store.QueryByMetadata(map[string]string{"team": "platform"})
	if len(got) != 1 {
		t.Errorf("QueryByMetadata() before expiration = %d services, want 1", len(got))
	}

	// Wait for service to expire
	time.Sleep(150 * time.Millisecond)

	// Should not find expired service
	got = store.QueryByMetadata(map[string]string{"team": "platform"})
	if len(got) != 0 {
		t.Errorf("QueryByMetadata() after expiration = %d services, want 0", len(got))
	}
}
