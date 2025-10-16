package acl

import (
	"testing"

	"github.com/neogan74/konsul/internal/logger"
)

func TestEvaluator_KVAccess(t *testing.T) {
	log := logger.GetDefault()
	eval := NewEvaluator(log)

	// Create a policy with KV rules
	policy := &Policy{
		Name:        "test-policy",
		Description: "Test policy for KV access",
		KV: []KVRule{
			{
				Path:         "app/config/*",
				Capabilities: []Capability{CapabilityRead, CapabilityList},
			},
			{
				Path:         "app/secrets/*",
				Capabilities: []Capability{CapabilityDeny},
			},
			{
				Path:         "app/data/public",
				Capabilities: []Capability{CapabilityRead, CapabilityWrite},
			},
		},
	}

	if err := eval.AddPolicy(policy); err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		capability Capability
		expected   bool
	}{
		// Allow - matches app/config/* with read
		{
			name:       "read config allowed",
			path:       "app/config/database",
			capability: CapabilityRead,
			expected:   true,
		},
		// Allow - matches app/config/* with list
		{
			name:       "list config allowed",
			path:       "app/config/settings",
			capability: CapabilityList,
			expected:   true,
		},
		// Deny - matches app/config/* but write not allowed
		{
			name:       "write config denied",
			path:       "app/config/database",
			capability: CapabilityWrite,
			expected:   false,
		},
		// Deny - explicit deny rule for app/secrets/*
		{
			name:       "read secrets denied",
			path:       "app/secrets/password",
			capability: CapabilityRead,
			expected:   false,
		},
		// Allow - exact match on app/data/public
		{
			name:       "read public data allowed",
			path:       "app/data/public",
			capability: CapabilityRead,
			expected:   true,
		},
		// Allow - exact match on app/data/public with write
		{
			name:       "write public data allowed",
			path:       "app/data/public",
			capability: CapabilityWrite,
			expected:   true,
		},
		// Deny - no matching rule
		{
			name:       "read unknown path denied",
			path:       "app/other/data",
			capability: CapabilityRead,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewKVResource(tt.path)
			allowed := eval.Evaluate([]string{"test-policy"}, resource, tt.capability)
			if allowed != tt.expected {
				t.Errorf("Expected %v, got %v for path=%s, capability=%s",
					tt.expected, allowed, tt.path, tt.capability)
			}
		})
	}
}

func TestEvaluator_ServiceAccess(t *testing.T) {
	log := logger.GetDefault()
	eval := NewEvaluator(log)

	policy := &Policy{
		Name:        "service-policy",
		Description: "Test policy for service access",
		Service: []ServiceRule{
			{
				Name:         "web-*",
				Capabilities: []Capability{CapabilityRead, CapabilityRegister},
			},
			{
				Name:         "database",
				Capabilities: []Capability{CapabilityRead},
			},
		},
	}

	if err := eval.AddPolicy(policy); err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	tests := []struct {
		name        string
		serviceName string
		capability  Capability
		expected    bool
	}{
		{
			name:        "read web service allowed",
			serviceName: "web-frontend",
			capability:  CapabilityRead,
			expected:    true,
		},
		{
			name:        "register web service allowed",
			serviceName: "web-api",
			capability:  CapabilityRegister,
			expected:    true,
		},
		{
			name:        "deregister web service denied",
			serviceName: "web-frontend",
			capability:  CapabilityDeregister,
			expected:    false,
		},
		{
			name:        "read database service allowed",
			serviceName: "database",
			capability:  CapabilityRead,
			expected:    true,
		},
		{
			name:        "register database service denied",
			serviceName: "database",
			capability:  CapabilityRegister,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewServiceResource(tt.serviceName)
			allowed := eval.Evaluate([]string{"service-policy"}, resource, tt.capability)
			if allowed != tt.expected {
				t.Errorf("Expected %v, got %v for service=%s, capability=%s",
					tt.expected, allowed, tt.serviceName, tt.capability)
			}
		})
	}
}

func TestEvaluator_MultiplePolicies(t *testing.T) {
	log := logger.GetDefault()
	eval := NewEvaluator(log)

	// First policy: read-only for app/config/*
	policy1 := &Policy{
		Name:        "readonly",
		Description: "Read-only access",
		KV: []KVRule{
			{
				Path:         "app/config/*",
				Capabilities: []Capability{CapabilityRead},
			},
		},
	}

	// Second policy: write access for app/config/public
	policy2 := &Policy{
		Name:        "writer",
		Description: "Write access to public config",
		KV: []KVRule{
			{
				Path:         "app/config/public",
				Capabilities: []Capability{CapabilityWrite},
			},
		},
	}

	if err := eval.AddPolicy(policy1); err != nil {
		t.Fatalf("Failed to add policy1: %v", err)
	}
	if err := eval.AddPolicy(policy2); err != nil {
		t.Fatalf("Failed to add policy2: %v", err)
	}

	// Test with multiple policies attached
	resource := NewKVResource("app/config/public")

	// Should be able to read (from policy1)
	if !eval.Evaluate([]string{"readonly", "writer"}, resource, CapabilityRead) {
		t.Error("Expected read to be allowed")
	}

	// Should be able to write (from policy2)
	if !eval.Evaluate([]string{"readonly", "writer"}, resource, CapabilityWrite) {
		t.Error("Expected write to be allowed")
	}

	// With only readonly policy, write should be denied
	if eval.Evaluate([]string{"readonly"}, resource, CapabilityWrite) {
		t.Error("Expected write to be denied with readonly policy only")
	}
}

func TestEvaluator_NoPolicies(t *testing.T) {
	log := logger.GetDefault()
	eval := NewEvaluator(log)

	resource := NewKVResource("app/config/test")

	// No policies attached - should deny
	if eval.Evaluate([]string{}, resource, CapabilityRead) {
		t.Error("Expected access to be denied with no policies")
	}

	// Non-existent policy - should deny
	if eval.Evaluate([]string{"non-existent"}, resource, CapabilityRead) {
		t.Error("Expected access to be denied with non-existent policy")
	}
}

func TestKVRule_PathMatching(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		testPath string
		expected bool
	}{
		{
			name:     "exact match",
			pattern:  "app/config",
			testPath: "app/config",
			expected: true,
		},
		{
			name:     "wildcard match",
			pattern:  "app/*",
			testPath: "app/config",
			expected: true,
		},
		{
			name:     "wildcard no match",
			pattern:  "app/*",
			testPath: "app/config/nested",
			expected: false,
		},
		{
			name:     "double wildcard match",
			pattern:  "app/**",
			testPath: "app/config/nested/deep",
			expected: true,
		},
		{
			name:     "prefix match",
			pattern:  "app/config/*",
			testPath: "app/config/database",
			expected: true,
		},
		{
			name:     "no match different path",
			pattern:  "app/config/*",
			testPath: "other/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := KVRule{
				Path:         tt.pattern,
				Capabilities: []Capability{CapabilityRead},
			}
			rule.Compile()

			if rule.Matches(tt.testPath) != tt.expected {
				t.Errorf("Pattern %s matching %s: expected %v, got %v",
					tt.pattern, tt.testPath, tt.expected, rule.Matches(tt.testPath))
			}
		})
	}
}

func TestPolicy_Validate(t *testing.T) {
	tests := []struct {
		name      string
		policy    *Policy
		shouldErr bool
	}{
		{
			name: "valid policy",
			policy: &Policy{
				Name:        "test",
				Description: "Test policy",
				KV: []KVRule{
					{Path: "app/*", Capabilities: []Capability{CapabilityRead}},
				},
			},
			shouldErr: false,
		},
		{
			name: "invalid - no name",
			policy: &Policy{
				Description: "Test policy",
			},
			shouldErr: true,
		},
		{
			name: "valid - empty rules",
			policy: &Policy{
				Name:        "empty",
				Description: "Empty policy",
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.shouldErr {
				t.Errorf("Expected error=%v, got error=%v", tt.shouldErr, err != nil)
			}
		})
	}
}

func TestEvaluator_PolicyManagement(t *testing.T) {
	log := logger.GetDefault()
	eval := NewEvaluator(log)

	policy := &Policy{
		Name:        "test",
		Description: "Test policy",
	}

	// Add policy
	if err := eval.AddPolicy(policy); err != nil {
		t.Fatalf("Failed to add policy: %v", err)
	}

	// Try to add duplicate
	if err := eval.AddPolicy(policy); err != ErrPolicyExists {
		t.Errorf("Expected ErrPolicyExists, got %v", err)
	}

	// Get policy
	retrieved, err := eval.GetPolicy("test")
	if err != nil {
		t.Fatalf("Failed to get policy: %v", err)
	}
	if retrieved.Name != "test" {
		t.Errorf("Expected policy name 'test', got '%s'", retrieved.Name)
	}

	// List policies
	policies := eval.ListPolicies()
	if len(policies) != 1 {
		t.Errorf("Expected 1 policy, got %d", len(policies))
	}

	// Update policy
	policy.Description = "Updated description"
	if err := eval.UpdatePolicy(policy); err != nil {
		t.Fatalf("Failed to update policy: %v", err)
	}

	// Delete policy
	if err := eval.DeletePolicy("test"); err != nil {
		t.Fatalf("Failed to delete policy: %v", err)
	}

	// Verify deleted
	if _, err := eval.GetPolicy("test"); err != ErrPolicyNotFound {
		t.Errorf("Expected ErrPolicyNotFound after delete, got %v", err)
	}
}
