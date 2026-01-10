package auth

import (
	"strings"
	"testing"
	"time"
)

func TestNewAPIKeyService(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		wantPrefix string
	}{
		{
			name:       "with prefix",
			prefix:     "test",
			wantPrefix: "test",
		},
		{
			name:       "empty prefix defaults to konsul",
			prefix:     "",
			wantPrefix: "konsul",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAPIKeyService(tt.prefix)
			if service == nil {
				t.Fatal("NewAPIKeyService() returned nil")
			}
			if service.prefix != tt.wantPrefix {
				t.Errorf("NewAPIKeyService() prefix = %v, want %v", service.prefix, tt.wantPrefix)
			}
		})
	}
}

func TestAPIKeyService_GenerateAPIKey(t *testing.T) {
	service := NewAPIKeyService("test")

	tests := []struct {
		name        string
		keyName     string
		permissions []string
		metadata    map[string]string
		expiresAt   *time.Time
		wantErr     bool
	}{
		{
			name:        "valid API key",
			keyName:     "test-key",
			permissions: []string{"read", "write"},
			metadata:    map[string]string{"env": "test"},
			expiresAt:   nil,
			wantErr:     false,
		},
		{
			name:        "with expiration",
			keyName:     "expiring-key",
			permissions: []string{"read"},
			metadata:    nil,
			expiresAt:   timePtr(time.Now().Add(24 * time.Hour)),
			wantErr:     false,
		},
		{
			name:        "empty name",
			keyName:     "",
			permissions: []string{"read"},
			metadata:    nil,
			expiresAt:   nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyString, apiKey, err := service.GenerateAPIKey(tt.keyName, tt.permissions, tt.metadata, tt.expiresAt)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if keyString == "" {
					t.Error("GenerateAPIKey() returned empty key string")
				}
				if !strings.HasPrefix(keyString, service.prefix+"_") {
					t.Errorf("GenerateAPIKey() key doesn't have correct prefix, got %v", keyString)
				}
				if apiKey == nil {
					t.Fatal("GenerateAPIKey() returned nil API key")
				}
				if apiKey.Name != tt.keyName {
					t.Errorf("GenerateAPIKey() name = %v, want %v", apiKey.Name, tt.keyName)
				}
				if !apiKey.Enabled {
					t.Error("GenerateAPIKey() key should be enabled by default")
				}
			}
		})
	}
}

func TestAPIKeyService_ValidateAPIKey(t *testing.T) {
	service := NewAPIKeyService("test")

	// Generate a valid key
	keyString, apiKey, err := service.GenerateAPIKey("test-key", []string{"read"}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Generate an expired key
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredKeyString, _, err := service.GenerateAPIKey("expired-key", []string{"read"}, nil, &expiredTime)
	if err != nil {
		t.Fatalf("Failed to generate expired API key: %v", err)
	}

	// Generate a key and disable it
	disabledKeyString, _, err := service.GenerateAPIKey("disabled-key", []string{"read"}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate disabled API key: %v", err)
	}
	// Find and disable the key
	for _, key := range service.ListAPIKeys() {
		if key.Name == "disabled-key" {
			if err := service.RevokeAPIKey(key.ID); err != nil {
				t.Fatalf("Failed to revoke API key: %v", err)
			}
			break
		}
	}

	tests := []struct {
		name    string
		key     string
		wantErr error
	}{
		{
			name:    "valid key",
			key:     keyString,
			wantErr: nil,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: ErrAPIKeyNotFound,
		},
		{
			name:    "invalid key",
			key:     "test_invalidkey123",
			wantErr: ErrAPIKeyNotFound,
		},
		{
			name:    "expired key",
			key:     expiredKeyString,
			wantErr: ErrAPIKeyExpired,
		},
		{
			name:    "disabled key",
			key:     disabledKeyString,
			wantErr: ErrAPIKeyDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ValidateAPIKey(tt.key)
			if err != tt.wantErr {
				t.Errorf("ValidateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Fatal("ValidateAPIKey() returned nil for valid key")
				}
				if result.ID != apiKey.ID {
					t.Errorf("ValidateAPIKey() ID = %v, want %v", result.ID, apiKey.ID)
				}
				if result.LastUsedAt == nil {
					t.Error("ValidateAPIKey() should update LastUsedAt")
				}
			}
		})
	}
}

func TestAPIKeyService_RevokeAPIKey(t *testing.T) {
	service := NewAPIKeyService("test")

	keyString, apiKey, err := service.GenerateAPIKey("test-key", []string{"read"}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Revoke the key
	err = service.RevokeAPIKey(apiKey.ID)
	if err != nil {
		t.Errorf("RevokeAPIKey() error = %v", err)
	}

	// Try to validate revoked key
	_, err = service.ValidateAPIKey(keyString)
	if err != ErrAPIKeyDisabled {
		t.Errorf("ValidateAPIKey() error = %v, want %v", err, ErrAPIKeyDisabled)
	}

	// Try to revoke non-existent key
	err = service.RevokeAPIKey("non-existent-id")
	if err != ErrAPIKeyNotFound {
		t.Errorf("RevokeAPIKey() error = %v, want %v", err, ErrAPIKeyNotFound)
	}
}

func TestAPIKeyService_DeleteAPIKey(t *testing.T) {
	service := NewAPIKeyService("test")

	keyString, apiKey, err := service.GenerateAPIKey("test-key", []string{"read"}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Delete the key
	err = service.DeleteAPIKey(apiKey.ID)
	if err != nil {
		t.Errorf("DeleteAPIKey() error = %v", err)
	}

	// Try to validate deleted key
	_, err = service.ValidateAPIKey(keyString)
	if err != ErrAPIKeyNotFound {
		t.Errorf("ValidateAPIKey() error = %v, want %v", err, ErrAPIKeyNotFound)
	}

	// Try to delete non-existent key
	err = service.DeleteAPIKey("non-existent-id")
	if err != ErrAPIKeyNotFound {
		t.Errorf("DeleteAPIKey() error = %v, want %v", err, ErrAPIKeyNotFound)
	}
}

func TestAPIKeyService_ListAPIKeys(t *testing.T) {
	service := NewAPIKeyService("test")

	// Initially empty
	keys := service.ListAPIKeys()
	if len(keys) != 0 {
		t.Errorf("ListAPIKeys() initial count = %v, want 0", len(keys))
	}

	// Generate some keys
	_, _, err := service.GenerateAPIKey("key1", []string{"read"}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate key1: %v", err)
	}

	_, _, err = service.GenerateAPIKey("key2", []string{"write"}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to generate key2: %v", err)
	}

	keys = service.ListAPIKeys()
	if len(keys) != 2 {
		t.Errorf("ListAPIKeys() count = %v, want 2", len(keys))
	}

	// Verify sensitive data is not included
	for _, key := range keys {
		if key.KeyHash != "" {
			t.Error("ListAPIKeys() should not include KeyHash")
		}
	}
}

func TestAPIKeyService_GetAPIKey(t *testing.T) {
	service := NewAPIKeyService("test")

	_, apiKey, err := service.GenerateAPIKey("test-key", []string{"read"}, map[string]string{"env": "test"}, nil)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	tests := []struct {
		name    string
		keyID   string
		wantErr error
	}{
		{
			name:    "valid key ID",
			keyID:   apiKey.ID,
			wantErr: nil,
		},
		{
			name:    "invalid key ID",
			keyID:   "non-existent",
			wantErr: ErrAPIKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetAPIKey(tt.keyID)
			if err != tt.wantErr {
				t.Errorf("GetAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Error("GetAPIKey() returned nil")
				}
				if result.KeyHash != "" {
					t.Error("GetAPIKey() should not include KeyHash")
				}
				if result.Name != apiKey.Name {
					t.Errorf("GetAPIKey() name = %v, want %v", result.Name, apiKey.Name)
				}
			}
		})
	}
}

func TestAPIKeyService_UpdateAPIKey(t *testing.T) {
	service := NewAPIKeyService("test")

	_, apiKey, err := service.GenerateAPIKey("test-key", []string{"read"}, map[string]string{"env": "test"}, nil)
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	tests := []struct {
		name        string
		keyID       string
		newName     string
		permissions []string
		metadata    map[string]string
		enabled     *bool
		wantErr     error
	}{
		{
			name:        "update name",
			keyID:       apiKey.ID,
			newName:     "updated-key",
			permissions: nil,
			metadata:    nil,
			enabled:     nil,
			wantErr:     nil,
		},
		{
			name:        "update permissions",
			keyID:       apiKey.ID,
			newName:     "",
			permissions: []string{"read", "write"},
			metadata:    nil,
			enabled:     nil,
			wantErr:     nil,
		},
		{
			name:        "disable key",
			keyID:       apiKey.ID,
			newName:     "",
			permissions: nil,
			metadata:    nil,
			enabled:     boolPtr(false),
			wantErr:     nil,
		},
		{
			name:        "non-existent key",
			keyID:       "non-existent",
			newName:     "test",
			permissions: nil,
			metadata:    nil,
			enabled:     nil,
			wantErr:     ErrAPIKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateAPIKey(tt.keyID, tt.newName, tt.permissions, tt.metadata, tt.enabled)
			if err != tt.wantErr {
				t.Errorf("UpdateAPIKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPIKeyService_HasPermission(t *testing.T) {
	service := NewAPIKeyService("test")

	tests := []struct {
		name        string
		permissions []string
		checkPerm   string
		want        bool
	}{
		{
			name:        "has specific permission",
			permissions: []string{"read", "write"},
			checkPerm:   "read",
			want:        true,
		},
		{
			name:        "does not have permission",
			permissions: []string{"read"},
			checkPerm:   "write",
			want:        false,
		},
		{
			name:        "wildcard permission",
			permissions: []string{"*"},
			checkPerm:   "anything",
			want:        true,
		},
		{
			name:        "nil API key",
			permissions: nil,
			checkPerm:   "read",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiKey *APIKey
			if tt.permissions != nil {
				_, apiKey, _ = service.GenerateAPIKey("test-"+tt.name, tt.permissions, nil, nil)
			}

			got := service.HasPermission(apiKey, tt.checkPerm)
			if got != tt.want {
				t.Errorf("HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func boolPtr(b bool) *bool {
	return &b
}
