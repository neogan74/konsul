package acl

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"
)

// Capability represents an ACL permission
type Capability string

const (
	// KV capabilities
	CapabilityRead   Capability = "read"
	CapabilityWrite  Capability = "write"
	CapabilityList   Capability = "list"
	CapabilityDelete Capability = "delete"
	CapabilityDeny   Capability = "deny"

	// Service capabilities
	CapabilityRegister   Capability = "register"
	CapabilityDeregister Capability = "deregister"

	// Backup capabilities
	CapabilityCreate  Capability = "create"
	CapabilityRestore Capability = "restore"
	CapabilityExport  Capability = "export"
	CapabilityImport  Capability = "import"

	// Admin capabilities
	CapabilityAdmin Capability = "admin"
)

// ResourceType represents the type of resource being accessed
type ResourceType string

const (
	ResourceTypeKV      ResourceType = "kv"
	ResourceTypeService ResourceType = "service"
	ResourceTypeHealth  ResourceType = "health"
	ResourceTypeBackup  ResourceType = "backup"
	ResourceTypeAdmin   ResourceType = "admin"
)

// Policy defines access control rules
type Policy struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	KV          []KVRule      `json:"kv,omitempty"`
	Service     []ServiceRule `json:"service,omitempty"`
	Health      []HealthRule  `json:"health,omitempty"`
	Backup      []BackupRule  `json:"backup,omitempty"`
	Admin       []AdminRule   `json:"admin,omitempty"`
}

// KVRule defines rules for KV store access
type KVRule struct {
	Path         string       `json:"path"`         // Path pattern (supports * wildcard)
	Capabilities []Capability `json:"capabilities"` // List of allowed capabilities
	compiled     *regexp.Regexp
}

// ServiceRule defines rules for service access
type ServiceRule struct {
	Name         string       `json:"name"`         // Service name pattern (supports * wildcard)
	Capabilities []Capability `json:"capabilities"` // List of allowed capabilities
	compiled     *regexp.Regexp
}

// HealthRule defines rules for health check access
type HealthRule struct {
	Capabilities []Capability `json:"capabilities"` // List of allowed capabilities
}

// BackupRule defines rules for backup operations
type BackupRule struct {
	Capabilities []Capability `json:"capabilities"` // List of allowed capabilities
}

// AdminRule defines rules for admin operations
type AdminRule struct {
	Capabilities []Capability `json:"capabilities"` // List of allowed capabilities
}

// Matches checks if a KV rule matches the given path
func (r *KVRule) Matches(kvPath string) bool {
	if r.compiled == nil {
		r.Compile()
	}
	return r.compiled.MatchString(kvPath)
}

// Compile converts the path pattern to a regex
func (r *KVRule) Compile() {
	pattern := r.Path

	// Escape special regex characters except *
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "+", "\\+")
	pattern = strings.ReplaceAll(pattern, "?", "\\?")
	pattern = strings.ReplaceAll(pattern, "[", "\\[")
	pattern = strings.ReplaceAll(pattern, "]", "\\]")
	pattern = strings.ReplaceAll(pattern, "(", "\\(")
	pattern = strings.ReplaceAll(pattern, ")", "\\)")
	pattern = strings.ReplaceAll(pattern, "{", "\\{")
	pattern = strings.ReplaceAll(pattern, "}", "\\}")

	// Convert wildcards to regex
	// * matches any characters within a path segment
	// ** matches any characters including path separators
	pattern = strings.ReplaceAll(pattern, "**", "§DOUBLESTAR§")
	pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
	pattern = strings.ReplaceAll(pattern, "§DOUBLESTAR§", ".*")

	// Anchor the pattern
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "$"
	}

	r.compiled = regexp.MustCompile(pattern)
}

// HasCapability checks if the rule has a specific capability
func (r *KVRule) HasCapability(cap Capability) bool {
	for _, c := range r.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Matches checks if a service rule matches the given service name
func (r *ServiceRule) Matches(serviceName string) bool {
	if r.compiled == nil {
		r.Compile()
	}
	return r.compiled.MatchString(serviceName)
}

// Compile converts the name pattern to a regex
func (r *ServiceRule) Compile() {
	pattern := r.Name

	// Escape special regex characters except *
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "+", "\\+")
	pattern = strings.ReplaceAll(pattern, "?", "\\?")
	pattern = strings.ReplaceAll(pattern, "[", "\\[")
	pattern = strings.ReplaceAll(pattern, "]", "\\]")
	pattern = strings.ReplaceAll(pattern, "(", "\\(")
	pattern = strings.ReplaceAll(pattern, ")", "\\)")
	pattern = strings.ReplaceAll(pattern, "{", "\\{")
	pattern = strings.ReplaceAll(pattern, "}", "\\}")

	// Convert wildcards to regex
	pattern = strings.ReplaceAll(pattern, "*", ".*")

	// Anchor the pattern
	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "$"
	}

	r.compiled = regexp.MustCompile(pattern)
}

// HasCapability checks if the rule has a specific capability
func (r *ServiceRule) HasCapability(cap Capability) bool {
	for _, c := range r.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// HasCapability checks if the rule has a specific capability
func (r *HealthRule) HasCapability(cap Capability) bool {
	for _, c := range r.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// HasCapability checks if the rule has a specific capability
func (r *BackupRule) HasCapability(cap Capability) bool {
	for _, c := range r.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// HasCapability checks if the rule has a specific capability
func (r *AdminRule) HasCapability(cap Capability) bool {
	for _, c := range r.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// Resource represents a resource being accessed
type Resource struct {
	Type ResourceType
	Path string // For KV: key path, For Service: service name
}

// NewKVResource creates a new KV resource
func NewKVResource(keyPath string) Resource {
	return Resource{
		Type: ResourceTypeKV,
		Path: path.Clean(keyPath),
	}
}

// NewServiceResource creates a new service resource
func NewServiceResource(serviceName string) Resource {
	return Resource{
		Type: ResourceTypeService,
		Path: serviceName,
	}
}

// NewHealthResource creates a new health resource
func NewHealthResource() Resource {
	return Resource{
		Type: ResourceTypeHealth,
	}
}

// NewBackupResource creates a new backup resource
func NewBackupResource() Resource {
	return Resource{
		Type: ResourceTypeBackup,
	}
}

// NewAdminResource creates a new admin resource
func NewAdminResource() Resource {
	return Resource{
		Type: ResourceTypeAdmin,
	}
}

// Validate checks if the policy is valid
func (p *Policy) Validate() error {
	if p.Name == "" {
		return ErrInvalidPolicy
	}

	// Compile all patterns
	for i := range p.KV {
		p.KV[i].Compile()
	}
	for i := range p.Service {
		p.Service[i].Compile()
	}

	return nil
}

// ToJSON converts the policy to JSON
func (p *Policy) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// FromJSON parses a policy from JSON
func FromJSON(data []byte) (*Policy, error) {
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, err
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	return &policy, nil
}
