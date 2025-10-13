package template

import (
	"os"
	"testing"
)

// MockKVStore implements KVStoreReader for testing
type MockKVStore struct {
	data map[string]string
}

func NewMockKVStore() *MockKVStore {
	return &MockKVStore{
		data: make(map[string]string),
	}
}

func (m *MockKVStore) Get(key string) (string, bool) {
	val, ok := m.data[key]
	return val, ok
}

func (m *MockKVStore) List() []string {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys
}

func (m *MockKVStore) Set(key, value string) {
	m.data[key] = value
}

// MockServiceStore implements ServiceStoreReader for testing
type MockServiceStore struct {
	services map[string]Service
}

func NewMockServiceStore() *MockServiceStore {
	return &MockServiceStore{
		services: make(map[string]Service),
	}
}

func (m *MockServiceStore) List() []Service {
	services := make([]Service, 0, len(m.services))
	for _, svc := range m.services {
		services = append(services, svc)
	}
	return services
}

func (m *MockServiceStore) Get(name string) (Service, bool) {
	svc, ok := m.services[name]
	return svc, ok
}

func (m *MockServiceStore) Register(svc Service) {
	m.services[svc.Name] = svc
}

func TestKVFunction(t *testing.T) {
	kvStore := NewMockKVStore()
	kvStore.Set("config/database/host", "localhost")
	kvStore.Set("config/database/port", "5432")

	ctx := &RenderContext{
		KVStore: kvStore,
	}

	tests := []struct {
		name      string
		key       string
		want      string
		expectErr bool
	}{
		{
			name:      "existing key",
			key:       "config/database/host",
			want:      "localhost",
			expectErr: false,
		},
		{
			name:      "non-existing key",
			key:       "config/nonexistent",
			want:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ctx.kv(tt.key)
			if (err != nil) != tt.expectErr {
				t.Errorf("kv() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if got != tt.want {
				t.Errorf("kv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKVTreeFunction(t *testing.T) {
	kvStore := NewMockKVStore()
	kvStore.Set("config/database/host", "localhost")
	kvStore.Set("config/database/port", "5432")
	kvStore.Set("config/redis/host", "redis.local")
	kvStore.Set("other/key", "value")

	ctx := &RenderContext{
		KVStore: kvStore,
	}

	pairs, err := ctx.kvTree("config/")
	if err != nil {
		t.Fatalf("kvTree() error = %v", err)
	}

	if len(pairs) != 3 {
		t.Errorf("kvTree() returned %d pairs, want 3", len(pairs))
	}

	// Check that all returned pairs have the correct prefix
	for _, pair := range pairs {
		if pair.Key[:7] != "config/" {
			t.Errorf("kvTree() returned key %s without prefix 'config/'", pair.Key)
		}
	}
}

func TestServiceFunction(t *testing.T) {
	serviceStore := NewMockServiceStore()
	serviceStore.Register(Service{
		Name:    "web",
		Address: "10.0.0.1",
		Port:    8080,
	})
	serviceStore.Register(Service{
		Name:    "api",
		Address: "10.0.0.2",
		Port:    9000,
	})

	ctx := &RenderContext{
		ServiceStore: serviceStore,
	}

	tests := []struct {
		name      string
		svcName   string
		wantCount int
		expectErr bool
	}{
		{
			name:      "existing service",
			svcName:   "web",
			wantCount: 1,
			expectErr: false,
		},
		{
			name:      "non-existing service",
			svcName:   "nonexistent",
			wantCount: 0,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ctx.service(tt.svcName)
			if (err != nil) != tt.expectErr {
				t.Errorf("service() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("service() returned %d services, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestServicesFunction(t *testing.T) {
	serviceStore := NewMockServiceStore()
	serviceStore.Register(Service{Name: "web", Address: "10.0.0.1", Port: 8080})
	serviceStore.Register(Service{Name: "api", Address: "10.0.0.2", Port: 9000})
	serviceStore.Register(Service{Name: "db", Address: "10.0.0.3", Port: 5432})

	ctx := &RenderContext{
		ServiceStore: serviceStore,
	}

	services, err := ctx.services()
	if err != nil {
		t.Fatalf("services() error = %v", err)
	}

	if len(services) != 3 {
		t.Errorf("services() returned %d services, want 3", len(services))
	}
}

func TestEnvFunction(t *testing.T) {
	// Set test environment variable
	key := "TEST_KONSUL_VAR"
	value := "test-value"
	os.Setenv(key, value)
	defer os.Unsetenv(key)

	got := env(key)
	if got != value {
		t.Errorf("env() = %v, want %v", got, value)
	}

	// Test non-existent variable
	got = env("NONEXISTENT_VAR_XYZABC")
	if got != "" {
		t.Errorf("env() = %v, want empty string", got)
	}
}

func TestFileFunction(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "konsul-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := "test file content\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test reading the file
	got, err := file(tmpFile.Name())
	if err != nil {
		t.Errorf("file() error = %v", err)
	}
	if got != content {
		t.Errorf("file() = %q, want %q", got, content)
	}

	// Test non-existent file
	_, err = file("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Errorf("file() expected error for non-existent file, got nil")
	}
}
