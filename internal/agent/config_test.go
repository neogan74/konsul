package agent

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BindAddress != "127.0.0.1:8502" {
		t.Errorf("Expected default bind address '127.0.0.1:8502', got '%s'", cfg.BindAddress)
	}

	if cfg.ServerAddress != "http://localhost:8888" {
		t.Errorf("Expected default server address 'http://localhost:8888', got '%s'", cfg.ServerAddress)
	}

	if cfg.Cache.ServiceTTL != 60*time.Second {
		t.Errorf("Expected service TTL 60s, got %v", cfg.Cache.ServiceTTL)
	}

	if cfg.Cache.KVTTL != 300*time.Second {
		t.Errorf("Expected KV TTL 300s, got %v", cfg.Cache.KVTTL)
	}

	if cfg.Cache.MaxEntries != 10000 {
		t.Errorf("Expected max entries 10000, got %d", cfg.Cache.MaxEntries)
	}

	if cfg.Sync.Interval != 10*time.Second {
		t.Errorf("Expected sync interval 10s, got %v", cfg.Sync.Interval)
	}

	if cfg.Sync.BatchSize != 100 {
		t.Errorf("Expected batch size 100, got %d", cfg.Sync.BatchSize)
	}

	if !cfg.Sync.Compression {
		t.Error("Expected compression enabled by default")
	}

	if !cfg.HealthChecks.EnableLocalExecution {
		t.Error("Expected local health check execution enabled by default")
	}

	if !cfg.HealthChecks.ReportOnlyChanges {
		t.Error("Expected report only changes enabled by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  func() *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: func() *Config {
				return &Config{
					ID:            "test-agent",
					NodeName:      "test-node",
					ServerAddress: "http://localhost:8888",
					BindAddress:   "127.0.0.1:8502",
					Cache: CacheConfig{
						ServiceTTL: 60 * time.Second,
						KVTTL:      300 * time.Second,
						MaxEntries: 10000,
					},
					Sync: SyncConfig{
						Interval:  10 * time.Second,
						BatchSize: 100,
					},
				}
			},
			wantErr: false,
		},
		{
			name: "missing agent ID",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ID = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "agent ID is required",
		},
		{
			name: "missing node name",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.NodeName = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "node name is required",
		},
		{
			name: "missing server address",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.ServerAddress = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "server address is required",
		},
		{
			name: "missing bind address",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.BindAddress = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "bind address is required",
		},
		{
			name: "invalid service TTL",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Cache.ServiceTTL = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "cache service TTL must be positive",
		},
		{
			name: "invalid KV TTL",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Cache.KVTTL = -1 * time.Second
				return cfg
			},
			wantErr: true,
			errMsg:  "cache KV TTL must be positive",
		},
		{
			name: "invalid max entries",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Cache.MaxEntries = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "cache max entries must be positive",
		},
		{
			name: "invalid sync interval",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Sync.Interval = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "sync interval must be positive",
		},
		{
			name: "invalid batch size",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Sync.BatchSize = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "sync batch size must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config()
			err := cfg.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestConfig_TLSConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TLS.Enabled = true
	cfg.TLS.SkipVerify = true

	if !cfg.TLS.Enabled {
		t.Error("Expected TLS enabled")
	}

	if !cfg.TLS.SkipVerify {
		t.Error("Expected TLS skip verify")
	}
}

func TestConfig_WatchedPrefixes(t *testing.T) {
	cfg := DefaultConfig()

	expectedPrefixes := []string{"config/", "feature_flags/"}
	if len(cfg.WatchedPrefixes) != len(expectedPrefixes) {
		t.Errorf("Expected %d watched prefixes, got %d", len(expectedPrefixes), len(cfg.WatchedPrefixes))
	}

	for i, prefix := range expectedPrefixes {
		if cfg.WatchedPrefixes[i] != prefix {
			t.Errorf("Expected prefix '%s', got '%s'", prefix, cfg.WatchedPrefixes[i])
		}
	}
}

func TestConfig_Metadata(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Metadata == nil {
		t.Error("Expected metadata map to be initialized")
	}

	// Test adding metadata
	cfg.Metadata["region"] = "us-east-1"
	cfg.Metadata["env"] = "production"

	if cfg.Metadata["region"] != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", cfg.Metadata["region"])
	}

	if cfg.Metadata["env"] != "production" {
		t.Errorf("Expected env 'production', got '%s'", cfg.Metadata["env"])
	}
}
