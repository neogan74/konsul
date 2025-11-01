package store

import (
	"strings"
	"testing"
)

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name    string
		tags    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid tags",
			tags:    []string{"env:production", "version:v1.2.3", "http", "region:us-east-1"},
			wantErr: false,
		},
		{
			name:    "empty tags list",
			tags:    []string{},
			wantErr: false,
		},
		{
			name:    "nil tags",
			tags:    nil,
			wantErr: false,
		},
		{
			name:    "single tag",
			tags:    []string{"production"},
			wantErr: false,
		},
		{
			name:    "tags with special characters",
			tags:    []string{"env:prod", "version:v1.2.3", "proto:http/1.1", "path:/api/v1"},
			wantErr: false,
		},
		{
			name:    "empty tag",
			tags:    []string{"env:production", ""},
			wantErr: true,
			errMsg:  "empty tag not allowed",
		},
		{
			name:    "duplicate tags",
			tags:    []string{"env:production", "http", "env:production"},
			wantErr: true,
			errMsg:  "duplicate tag",
		},
		{
			name:    "tag too long",
			tags:    []string{strings.Repeat("a", 256)},
			wantErr: true,
			errMsg:  "tag too long",
		},
		{
			name:    "invalid tag format - spaces",
			tags:    []string{"env production"},
			wantErr: true,
			errMsg:  "invalid tag format",
		},
		{
			name:    "invalid tag format - special chars",
			tags:    []string{"env@production"},
			wantErr: true,
			errMsg:  "invalid tag format",
		},
		{
			name:    "too many tags",
			tags:    make([]string, 65),
			wantErr: true,
			errMsg:  "too many tags",
		},
		{
			name:    "exactly max tags",
			tags:    make([]string, 64),
			wantErr: false,
		},
	}

	// Fill in the "too many tags" and "exactly max tags" test cases
	for i := 0; i < 65; i++ {
		if i < 64 {
			tests[len(tests)-1].tags[i] = "tag" + string(rune('0'+i%10))
		}
		tests[len(tests)-2].tags[i] = "tag" + string(rune('0'+i%10))
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTags(tt.tags)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTags() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateTags() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTags() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		name    string
		meta    map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid metadata",
			meta: map[string]string{
				"team":        "platform",
				"owner":       "alice@example.com",
				"cost-center": "engineering",
			},
			wantErr: false,
		},
		{
			name:    "empty metadata",
			meta:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "nil metadata",
			meta:    nil,
			wantErr: false,
		},
		{
			name: "single metadata entry",
			meta: map[string]string{
				"team": "platform",
			},
			wantErr: false,
		},
		{
			name: "metadata with long value",
			meta: map[string]string{
				"description": strings.Repeat("a", 512),
			},
			wantErr: false,
		},
		{
			name: "empty metadata key",
			meta: map[string]string{
				"": "value",
			},
			wantErr: true,
			errMsg:  "empty metadata key",
		},
		{
			name: "metadata key too long",
			meta: map[string]string{
				strings.Repeat("a", 129): "value",
			},
			wantErr: true,
			errMsg:  "metadata key too long",
		},
		{
			name: "metadata value too long",
			meta: map[string]string{
				"description": strings.Repeat("a", 513),
			},
			wantErr: true,
			errMsg:  "metadata value too long",
		},
		{
			name: "invalid metadata key format - special chars",
			meta: map[string]string{
				"team@name": "platform",
			},
			wantErr: true,
			errMsg:  "invalid metadata key format",
		},
		{
			name: "invalid metadata key format - spaces",
			meta: map[string]string{
				"team name": "platform",
			},
			wantErr: true,
			errMsg:  "invalid metadata key format",
		},
		{
			name: "reserved metadata key - konsul_ prefix",
			meta: map[string]string{
				"konsul_internal": "value",
			},
			wantErr: true,
			errMsg:  "reserved metadata key",
		},
		{
			name: "reserved metadata key - underscore prefix",
			meta: map[string]string{
				"_internal": "value",
			},
			wantErr: true,
			errMsg:  "reserved metadata key",
		},
	}

	// Add test case for too many metadata keys
	tooManyMeta := make(map[string]string)
	for i := 0; i < 65; i++ {
		tooManyMeta["key"+string(rune('0'+i%10))+string(rune('a'+i/10))] = "value"
	}
	tests = append(tests, struct {
		name    string
		meta    map[string]string
		wantErr bool
		errMsg  string
	}{
		name:    "too many metadata keys",
		meta:    tooManyMeta,
		wantErr: true,
		errMsg:  "too many metadata keys",
	})

	// Add test case for exactly max metadata keys
	exactlyMaxMeta := make(map[string]string)
	for i := 0; i < 64; i++ {
		exactlyMaxMeta["key"+string(rune('0'+i%10))+string(rune('a'+i/10))] = "value"
	}
	tests = append(tests, struct {
		name    string
		meta    map[string]string
		wantErr bool
		errMsg  string
	}{
		name:    "exactly max metadata keys",
		meta:    exactlyMaxMeta,
		wantErr: false,
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetadata(tt.meta)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateMetadata() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateMetadata() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateMetadata() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateService(t *testing.T) {
	tests := []struct {
		name    string
		service *Service
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service with tags and metadata",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    8080,
				Tags:    []string{"env:production", "http"},
				Meta: map[string]string{
					"team":  "platform",
					"owner": "alice@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "valid service without tags and metadata",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    8080,
			},
			wantErr: false,
		},
		{
			name: "missing service name",
			service: &Service{
				Address: "10.0.1.50",
				Port:    8080,
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "missing service address",
			service: &Service{
				Name: "api-service",
				Port: 8080,
			},
			wantErr: true,
			errMsg:  "service address is required",
		},
		{
			name: "missing service port",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
			},
			wantErr: true,
			errMsg:  "service port is required",
		},
		{
			name: "invalid port - too low",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    0,
			},
			wantErr: true,
			errMsg:  "service port is required",
		},
		{
			name: "invalid port - too high",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    65536,
			},
			wantErr: true,
			errMsg:  "service port must be between",
		},
		{
			name: "invalid tags",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    8080,
				Tags:    []string{"invalid tag with spaces"},
			},
			wantErr: true,
			errMsg:  "tag validation failed",
		},
		{
			name: "invalid metadata",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    8080,
				Meta: map[string]string{
					"konsul_reserved": "value",
				},
			},
			wantErr: true,
			errMsg:  "metadata validation failed",
		},
		{
			name: "valid port boundaries - minimum",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    1,
			},
			wantErr: false,
		},
		{
			name: "valid port boundaries - maximum",
			service: &Service{
				Name:    "api-service",
				Address: "10.0.1.50",
				Port:    65535,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateService(tt.service)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateService() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateService() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateService() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestIsReservedMetadataKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"konsul prefix", "konsul_internal", true},
		{"underscore prefix", "_internal", true},
		{"valid key", "team", false},
		{"valid key with dash", "cost-center", false},
		{"valid key with underscore in middle", "team_name", false},
		{"konsul in middle", "my_konsul_key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReservedMetadataKey(tt.key)
			if got != tt.want {
				t.Errorf("isReservedMetadataKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
