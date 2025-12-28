package agent

import (
	"fmt"
	"time"
)

// Config represents the agent configuration
type Config struct {
	// Agent identity
	ID         string            `json:"id" yaml:"id"`
	NodeName   string            `json:"node_name" yaml:"node_name"`
	NodeIP     string            `json:"node_ip" yaml:"node_ip"`
	Datacenter string            `json:"datacenter" yaml:"datacenter"`
	Metadata   map[string]string `json:"metadata" yaml:"metadata"`

	// Server connection
	ServerAddress string    `json:"server_address" yaml:"server_address"`
	TLS           TLSConfig `json:"tls" yaml:"tls"`

	// API server
	BindAddress string `json:"bind_address" yaml:"bind_address"` // Default: 127.0.0.1:8502

	// Cache configuration
	Cache CacheConfig `json:"cache" yaml:"cache"`

	// Health checks
	HealthChecks HealthCheckConfig `json:"health_checks" yaml:"health_checks"`

	// Sync configuration
	Sync SyncConfig `json:"sync" yaml:"sync"`

	// Resources
	Resources ResourceConfig `json:"resources" yaml:"resources"`

	// Watched prefixes for KV store
	WatchedPrefixes []string `json:"watched_prefixes" yaml:"watched_prefixes"`
}

// TLSConfig represents TLS configuration for server connection
type TLSConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	CACert     string `json:"ca_cert" yaml:"ca_cert"`
	ClientCert string `json:"client_cert" yaml:"client_cert"`
	ClientKey  string `json:"client_key" yaml:"client_key"`
	SkipVerify bool   `json:"skip_verify" yaml:"skip_verify"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	ServiceTTL     time.Duration `json:"service_ttl" yaml:"service_ttl"`         // Default: 60s
	KVTTL          time.Duration `json:"kv_ttl" yaml:"kv_ttl"`                   // Default: 300s (5m)
	HealthTTL      time.Duration `json:"health_ttl" yaml:"health_ttl"`           // Default: 30s
	MaxEntries     int           `json:"max_entries" yaml:"max_entries"`         // Default: 10000
	EvictionPolicy string        `json:"eviction_policy" yaml:"eviction_policy"` // Default: "lru"
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	EnableLocalExecution bool          `json:"enable_local_execution" yaml:"enable_local_execution"` // Default: true
	CheckInterval        time.Duration `json:"check_interval" yaml:"check_interval"`                 // Default: 10s
	ReportOnlyChanges    bool          `json:"report_only_changes" yaml:"report_only_changes"`       // Default: true
	Timeout              time.Duration `json:"timeout" yaml:"timeout"`                               // Default: 5s
}

// SyncConfig represents sync configuration
type SyncConfig struct {
	Interval         time.Duration `json:"interval" yaml:"interval"`                     // Default: 10s
	FullSyncInterval time.Duration `json:"full_sync_interval" yaml:"full_sync_interval"` // Default: 300s (5m)
	BatchSize        int           `json:"batch_size" yaml:"batch_size"`                 // Default: 100
	Compression      bool          `json:"compression" yaml:"compression"`               // Default: true
	RetryAttempts    int           `json:"retry_attempts" yaml:"retry_attempts"`         // Default: 3
	RetryDelay       time.Duration `json:"retry_delay" yaml:"retry_delay"`               // Default: 5s
}

// ResourceConfig represents resource limits
type ResourceConfig struct {
	MemoryLimit string `json:"memory_limit" yaml:"memory_limit"` // e.g., "128Mi"
	CPULimit    string `json:"cpu_limit" yaml:"cpu_limit"`       // e.g., "100m"
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ID:            generateDefaultAgentID(),
		NodeName:      "konsul-agent",
		BindAddress:   "127.0.0.1:8502",
		ServerAddress: "http://localhost:8888",
		TLS: TLSConfig{
			Enabled:    false,
			SkipVerify: false,
		},
		Cache: CacheConfig{
			ServiceTTL:     60 * time.Second,
			KVTTL:          300 * time.Second,
			HealthTTL:      30 * time.Second,
			MaxEntries:     10000,
			EvictionPolicy: "lru",
		},
		HealthChecks: HealthCheckConfig{
			EnableLocalExecution: true,
			CheckInterval:        10 * time.Second,
			ReportOnlyChanges:    true,
			Timeout:              5 * time.Second,
		},
		Sync: SyncConfig{
			Interval:         10 * time.Second,
			FullSyncInterval: 300 * time.Second,
			BatchSize:        100,
			Compression:      true,
			RetryAttempts:    3,
			RetryDelay:       5 * time.Second,
		},
		Resources: ResourceConfig{
			MemoryLimit: "128Mi",
			CPULimit:    "100m",
		},
		WatchedPrefixes: []string{"config/", "feature_flags/"},
		Metadata:        make(map[string]string),
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("agent ID is required")
	}
	if c.NodeName == "" {
		return fmt.Errorf("node name is required")
	}
	if c.ServerAddress == "" {
		return fmt.Errorf("server address is required")
	}
	if c.BindAddress == "" {
		return fmt.Errorf("bind address is required")
	}
	if c.Cache.ServiceTTL <= 0 {
		return fmt.Errorf("cache service TTL must be positive")
	}
	if c.Cache.KVTTL <= 0 {
		return fmt.Errorf("cache KV TTL must be positive")
	}
	if c.Cache.MaxEntries <= 0 {
		return fmt.Errorf("cache max entries must be positive")
	}
	if c.Sync.Interval <= 0 {
		return fmt.Errorf("sync interval must be positive")
	}
	if c.Sync.BatchSize <= 0 {
		return fmt.Errorf("sync batch size must be positive")
	}
	return nil
}

// generateDefaultAgentID generates a default agent ID
func generateDefaultAgentID() string {
	// Will be replaced with a proper implementation using hostname + UUID
	return "agent-default"
}
