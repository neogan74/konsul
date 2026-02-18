package config

import (
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check default values
	if cfg.Server.Host != "" {
		t.Errorf("expected empty host, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Server.Port)
	}
	if cfg.Service.TTL != 30*time.Second {
		t.Errorf("expected TTL 30s, got %v", cfg.Service.TTL)
	}
	if cfg.Service.CleanupInterval != 60*time.Second {
		t.Errorf("expected cleanup interval 60s, got %v", cfg.Service.CleanupInterval)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected log level 'info', got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("expected log format 'text', got %q", cfg.Log.Format)
	}
	if cfg.Audit.Enabled {
		t.Errorf("expected audit logging disabled by default")
	}
	if cfg.Audit.Sink != "file" {
		t.Errorf("expected audit sink 'file', got %q", cfg.Audit.Sink)
	}
	if cfg.Audit.FilePath != "./logs/audit.log" {
		t.Errorf("unexpected audit file path: %q", cfg.Audit.FilePath)
	}
	if cfg.Audit.BufferSize != 1024 {
		t.Errorf("expected audit buffer size 1024, got %d", cfg.Audit.BufferSize)
	}
	if cfg.Audit.FlushInterval != time.Second {
		t.Errorf("expected audit flush interval 1s, got %v", cfg.Audit.FlushInterval)
	}
	if cfg.Audit.DropPolicy != "drop" {
		t.Errorf("expected audit drop policy 'drop', got %q", cfg.Audit.DropPolicy)
	}
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Clear environment variables first
	clearEnvVars(t)

	// Set environment variables
	t.Setenv("KONSUL_HOST", "localhost")
	t.Setenv("KONSUL_PORT", "9999")
	t.Setenv("KONSUL_SERVICE_TTL", "45s")
	t.Setenv("KONSUL_CLEANUP_INTERVAL", "2m")
	t.Setenv("KONSUL_LOG_LEVEL", "debug")
	t.Setenv("KONSUL_LOG_FORMAT", "json")
	t.Setenv("KONSUL_AUDIT_ENABLED", "true")
	t.Setenv("KONSUL_AUDIT_SINK", "stdout")
	t.Setenv("KONSUL_AUDIT_BUFFER_SIZE", "2048")
	t.Setenv("KONSUL_AUDIT_FILE_PATH", "/var/log/konsul/audit.log")
	t.Setenv("KONSUL_AUDIT_FLUSH_INTERVAL", "2s")
	t.Setenv("KONSUL_AUDIT_DROP_POLICY", "block")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check environment variable values
	if cfg.Server.Host != "localhost" {
		t.Errorf("expected host 'localhost', got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Server.Port)
	}
	if cfg.Service.TTL != 45*time.Second {
		t.Errorf("expected TTL 45s, got %v", cfg.Service.TTL)
	}
	if cfg.Service.CleanupInterval != 2*time.Minute {
		t.Errorf("expected cleanup interval 2m, got %v", cfg.Service.CleanupInterval)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level 'debug', got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("expected log format 'json', got %q", cfg.Log.Format)
	}
	if !cfg.Audit.Enabled {
		t.Errorf("expected audit enabled via env")
	}
	if cfg.Audit.Sink != "stdout" {
		t.Errorf("expected audit sink stdout, got %q", cfg.Audit.Sink)
	}
	if cfg.Audit.BufferSize != 2048 {
		t.Errorf("expected audit buffer size 2048, got %d", cfg.Audit.BufferSize)
	}
	if cfg.Audit.FilePath != "/var/log/konsul/audit.log" {
		t.Errorf("unexpected audit file path: %s", cfg.Audit.FilePath)
	}
	if cfg.Audit.FlushInterval != 2*time.Second {
		t.Errorf("unexpected audit flush interval: %v", cfg.Audit.FlushInterval)
	}
	if cfg.Audit.DropPolicy != "block" {
		t.Errorf("expected audit drop policy block, got %q", cfg.Audit.DropPolicy)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() failed for valid config: %v", err)
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	testCases := []int{0, -1, 65536, 100000}

	for _, port := range testCases {
		cfg := &Config{
			Server: ServerConfig{Port: port},
			Service: ServiceConfig{
				TTL:             30 * time.Second,
				CleanupInterval: 60 * time.Second,
			},
			Log: LogConfig{Level: "info", Format: "text"},
		}

		if err := cfg.Validate(); err == nil {
			t.Errorf("expected validation error for port %d", port)
		}
	}
}

func TestValidate_InvalidTTL(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             -1 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for negative TTL")
	}
}

func TestValidate_InvalidCleanupInterval(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 0,
		},
		Log: LogConfig{Level: "info", Format: "text"},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for zero cleanup interval")
	}
}

func TestValidate_InvalidAuditConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Audit: AuditConfig{
			Enabled:       true,
			Sink:          "invalid",
			FilePath:      "./audit.log",
			BufferSize:    10,
			FlushInterval: time.Second,
			DropPolicy:    "drop",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for invalid audit sink")
	}
}

func TestValidate_InvalidAuditDropPolicy(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Audit: AuditConfig{
			Enabled:       true,
			Sink:          "file",
			FilePath:      "./audit.log",
			BufferSize:    10,
			FlushInterval: time.Second,
			DropPolicy:    "invalid",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for audit drop policy")
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "invalid", Format: "text"},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for invalid log level")
	}
}

func TestValidate_InvalidLogFormat(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "invalid"},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for invalid log format")
	}
}

func TestAddress(t *testing.T) {
	testCases := []struct {
		host     string
		port     int
		expected string
	}{
		{"", 8080, ":8080"},
		{"localhost", 8080, "localhost:8080"},
		{"127.0.0.1", 9999, "127.0.0.1:9999"},
		{"0.0.0.0", 80, "0.0.0.0:80"},
	}

	for _, tc := range testCases {
		cfg := &Config{
			Server: ServerConfig{
				Host: tc.host,
				Port: tc.port,
			},
		}

		address := cfg.Address()
		if address != tc.expected {
			t.Errorf("Address() = %q, expected %q", address, tc.expected)
		}
	}
}

func TestLoad_InvalidEnvironmentValues(t *testing.T) {
	clearEnvVars(t)

	// Test invalid port
	t.Setenv("KONSUL_PORT", "invalid")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	// Should fall back to default
	if cfg.Server.Port != 8888 {
		t.Errorf("expected default port 8888 for invalid env value, got %d", cfg.Server.Port)
	}

	// Test invalid duration
	t.Setenv("KONSUL_SERVICE_TTL", "invalid")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	// Should fall back to default
	if cfg.Service.TTL != 30*time.Second {
		t.Errorf("expected default TTL 30s for invalid env value, got %v", cfg.Service.TTL)
	}

	clearEnvVars(t)
}

func TestLoad_InvalidConfigValidation(t *testing.T) {
	clearEnvVars(t)

	// Set invalid port that will fail validation
	t.Setenv("KONSUL_PORT", "0")
	defer clearEnvVars(t)

	_, err := Load()
	if err == nil {
		t.Error("expected Load() to fail validation with invalid port")
	}
}

func TestDNS_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check DNS default values
	if cfg.DNS.Enabled != true {
		t.Errorf("expected DNS enabled true by default, got %t", cfg.DNS.Enabled)
	}
	if cfg.DNS.Host != "" {
		t.Errorf("expected DNS host empty by default, got %q", cfg.DNS.Host)
	}
	if cfg.DNS.Port != 8600 {
		t.Errorf("expected DNS port 8600 by default, got %d", cfg.DNS.Port)
	}
	if cfg.DNS.Domain != "consul" {
		t.Errorf("expected DNS domain 'consul' by default, got %q", cfg.DNS.Domain)
	}
}

func TestDNS_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	// Set DNS environment variables
	t.Setenv("KONSUL_DNS_ENABLED", "false")
	t.Setenv("KONSUL_DNS_HOST", "127.0.0.1")
	t.Setenv("KONSUL_DNS_PORT", "5353")
	t.Setenv("KONSUL_DNS_DOMAIN", "local")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check DNS environment variable values
	if cfg.DNS.Enabled != false {
		t.Errorf("expected DNS enabled false, got %t", cfg.DNS.Enabled)
	}
	if cfg.DNS.Host != "127.0.0.1" {
		t.Errorf("expected DNS host '127.0.0.1', got %q", cfg.DNS.Host)
	}
	if cfg.DNS.Port != 5353 {
		t.Errorf("expected DNS port 5353, got %d", cfg.DNS.Port)
	}
	if cfg.DNS.Domain != "local" {
		t.Errorf("expected DNS domain 'local', got %q", cfg.DNS.Domain)
	}
}

func TestValidate_DNSInvalidPort(t *testing.T) {
	clearEnvVars(t)

	// Set invalid DNS port
	t.Setenv("KONSUL_DNS_PORT", "0")
	defer clearEnvVars(t)

	_, err := Load()
	if err == nil {
		t.Error("expected Load() to fail validation with invalid DNS port")
	}
}

func TestValidate_DNSInvalidDomain(t *testing.T) {
	clearEnvVars(t)

	// Test that empty domain env var falls back to default and passes validation
	t.Setenv("KONSUL_DNS_ENABLED", "true")
	t.Setenv("KONSUL_DNS_DOMAIN", "")
	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not fail with empty domain env var: %v", err)
	}

	// Should fall back to default domain
	if cfg.DNS.Domain != "consul" {
		t.Errorf("expected DNS domain to fall back to 'consul', got %q", cfg.DNS.Domain)
	}
}

func TestValidate_DNSValidConfig(t *testing.T) {
	clearEnvVars(t)

	// Set valid DNS configuration
	t.Setenv("KONSUL_DNS_ENABLED", "true")
	t.Setenv("KONSUL_DNS_HOST", "localhost")
	t.Setenv("KONSUL_DNS_PORT", "53")
	t.Setenv("KONSUL_DNS_DOMAIN", "internal")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed with valid DNS config: %v", err)
	}

	// Verify all values are set correctly
	if !cfg.DNS.Enabled {
		t.Error("expected DNS to be enabled")
	}
	if cfg.DNS.Host != "localhost" {
		t.Errorf("expected DNS host 'localhost', got %q", cfg.DNS.Host)
	}
	if cfg.DNS.Port != 53 {
		t.Errorf("expected DNS port 53, got %d", cfg.DNS.Port)
	}
	if cfg.DNS.Domain != "internal" {
		t.Errorf("expected DNS domain 'internal', got %q", cfg.DNS.Domain)
	}
}

// TLS Configuration Tests
func TestTLS_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.TLS.Enabled {
		t.Error("expected TLS disabled by default")
	}
	if cfg.Server.TLS.CertFile != "" {
		t.Errorf("expected empty cert file by default, got %q", cfg.Server.TLS.CertFile)
	}
	if cfg.Server.TLS.KeyFile != "" {
		t.Errorf("expected empty key file by default, got %q", cfg.Server.TLS.KeyFile)
	}
	if cfg.Server.TLS.AutoCert {
		t.Error("expected AutoCert disabled by default")
	}
}

func TestTLS_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	t.Setenv("KONSUL_TLS_ENABLED", "true")
	t.Setenv("KONSUL_TLS_CERT_FILE", "/path/to/cert.pem")
	t.Setenv("KONSUL_TLS_KEY_FILE", "/path/to/key.pem")
	t.Setenv("KONSUL_TLS_AUTO_CERT", "false")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.Server.TLS.Enabled {
		t.Error("expected TLS enabled")
	}
	if cfg.Server.TLS.CertFile != "/path/to/cert.pem" {
		t.Errorf("expected cert file '/path/to/cert.pem', got %q", cfg.Server.TLS.CertFile)
	}
	if cfg.Server.TLS.KeyFile != "/path/to/key.pem" {
		t.Errorf("expected key file '/path/to/key.pem', got %q", cfg.Server.TLS.KeyFile)
	}
	if cfg.Server.TLS.AutoCert {
		t.Error("expected AutoCert disabled")
	}
}

func TestValidate_TLSMissingCertFile(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			TLS: TLSConfig{
				Enabled:  true,
				AutoCert: false,
				CertFile: "",
				KeyFile:  "/path/to/key.pem",
			},
		},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing TLS cert file")
	}
}

func TestValidate_TLSMissingKeyFile(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			TLS: TLSConfig{
				Enabled:  true,
				AutoCert: false,
				CertFile: "/path/to/cert.pem",
				KeyFile:  "",
			},
		},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing TLS key file")
	}
}

func TestValidate_TLSAutoCert(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 8080,
			TLS: TLSConfig{
				Enabled:  true,
				AutoCert: true,
			},
		},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no validation error with AutoCert enabled: %v", err)
	}
}

// Persistence Configuration Tests
func TestPersistence_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Persistence.Enabled {
		t.Error("expected persistence disabled by default")
	}
	if cfg.Persistence.Type != "badger" {
		t.Errorf("expected persistence type 'badger' by default, got %q", cfg.Persistence.Type)
	}
	if cfg.Persistence.DataDir != "./data" {
		t.Errorf("expected data dir './data' by default, got %q", cfg.Persistence.DataDir)
	}
	if cfg.Persistence.BackupDir != "./backups" {
		t.Errorf("expected backup dir './backups' by default, got %q", cfg.Persistence.BackupDir)
	}
	if !cfg.Persistence.SyncWrites {
		t.Error("expected SyncWrites enabled by default")
	}
	if !cfg.Persistence.WALEnabled {
		t.Error("expected WALEnabled true by default")
	}
}

func TestPersistence_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	t.Setenv("KONSUL_PERSISTENCE_ENABLED", "true")
	t.Setenv("KONSUL_PERSISTENCE_TYPE", "memory")
	t.Setenv("KONSUL_DATA_DIR", "/var/lib/konsul")
	t.Setenv("KONSUL_BACKUP_DIR", "/var/backups/konsul")
	t.Setenv("KONSUL_SYNC_WRITES", "false")
	t.Setenv("KONSUL_WAL_ENABLED", "false")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.Persistence.Enabled {
		t.Error("expected persistence enabled")
	}
	if cfg.Persistence.Type != "memory" {
		t.Errorf("expected persistence type 'memory', got %q", cfg.Persistence.Type)
	}
	if cfg.Persistence.DataDir != "/var/lib/konsul" {
		t.Errorf("expected data dir '/var/lib/konsul', got %q", cfg.Persistence.DataDir)
	}
	if cfg.Persistence.BackupDir != "/var/backups/konsul" {
		t.Errorf("expected backup dir '/var/backups/konsul', got %q", cfg.Persistence.BackupDir)
	}
	if cfg.Persistence.SyncWrites {
		t.Error("expected SyncWrites disabled")
	}
	if cfg.Persistence.WALEnabled {
		t.Error("expected WALEnabled false")
	}
}

func TestValidate_PersistenceInvalidType(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Persistence: PersistenceConfig{
			Enabled: true,
			Type:    "invalid",
			DataDir: "/tmp/data",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for invalid persistence type")
	}
}

func TestValidate_PersistenceMissingDataDir(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Persistence: PersistenceConfig{
			Enabled: true,
			Type:    "badger",
			DataDir: "",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing data directory")
	}
}

// RateLimit Configuration Tests
func TestRateLimit_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.RateLimit.Enabled {
		t.Error("expected rate limiting disabled by default")
	}
	if cfg.RateLimit.RequestsPerSec != 100.0 {
		t.Errorf("expected 100 requests per second by default, got %f", cfg.RateLimit.RequestsPerSec)
	}
	if cfg.RateLimit.Burst != 20 {
		t.Errorf("expected burst 20 by default, got %d", cfg.RateLimit.Burst)
	}
	if !cfg.RateLimit.ByIP {
		t.Error("expected ByIP enabled by default")
	}
	if cfg.RateLimit.ByAPIKey {
		t.Error("expected ByAPIKey disabled by default")
	}
	if cfg.RateLimit.CleanupInterval != 5*time.Minute {
		t.Errorf("expected cleanup interval 5m by default, got %v", cfg.RateLimit.CleanupInterval)
	}
}

func TestRateLimit_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	t.Setenv("KONSUL_RATE_LIMIT_ENABLED", "true")
	t.Setenv("KONSUL_RATE_LIMIT_REQUESTS_PER_SEC", "50.5")
	t.Setenv("KONSUL_RATE_LIMIT_BURST", "10")
	t.Setenv("KONSUL_RATE_LIMIT_BY_IP", "false")
	t.Setenv("KONSUL_RATE_LIMIT_BY_APIKEY", "true")
	t.Setenv("KONSUL_RATE_LIMIT_CLEANUP", "10m")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.RateLimit.Enabled {
		t.Error("expected rate limiting enabled")
	}
	if cfg.RateLimit.RequestsPerSec != 50.5 {
		t.Errorf("expected 50.5 requests per second, got %f", cfg.RateLimit.RequestsPerSec)
	}
	if cfg.RateLimit.Burst != 10 {
		t.Errorf("expected burst 10, got %d", cfg.RateLimit.Burst)
	}
	if cfg.RateLimit.ByIP {
		t.Error("expected ByIP disabled")
	}
	if !cfg.RateLimit.ByAPIKey {
		t.Error("expected ByAPIKey enabled")
	}
	if cfg.RateLimit.CleanupInterval != 10*time.Minute {
		t.Errorf("expected cleanup interval 10m, got %v", cfg.RateLimit.CleanupInterval)
	}
}

func TestValidate_RateLimitInvalidRequestsPerSec(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerSec: 0,
			Burst:          10,
			ByIP:           true,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for zero requests per second")
	}
}

func TestValidate_RateLimitInvalidBurst(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerSec: 100,
			Burst:          0,
			ByIP:           true,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for zero burst")
	}
}

func TestValidate_RateLimitNoStrategyEnabled(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerSec: 100,
			Burst:          10,
			ByIP:           false,
			ByAPIKey:       false,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error when neither ByIP nor ByAPIKey is enabled")
	}
}

// Auth Configuration Tests
func TestAuth_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Auth.Enabled {
		t.Error("expected auth disabled by default")
	}
	if cfg.Auth.JWTSecret != "" {
		t.Errorf("expected empty JWT secret by default, got %q", cfg.Auth.JWTSecret)
	}
	if cfg.Auth.JWTExpiry != 15*time.Minute {
		t.Errorf("expected JWT expiry 15m by default, got %v", cfg.Auth.JWTExpiry)
	}
	if cfg.Auth.RefreshExpiry != 7*24*time.Hour {
		t.Errorf("expected refresh expiry 7d by default, got %v", cfg.Auth.RefreshExpiry)
	}
	if cfg.Auth.Issuer != "konsul" {
		t.Errorf("expected issuer 'konsul' by default, got %q", cfg.Auth.Issuer)
	}
	if cfg.Auth.APIKeyPrefix != "konsul" {
		t.Errorf("expected API key prefix 'konsul' by default, got %q", cfg.Auth.APIKeyPrefix)
	}
	if cfg.Auth.RequireAuth {
		t.Error("expected RequireAuth disabled by default")
	}
}

func TestAuth_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	t.Setenv("KONSUL_AUTH_ENABLED", "true")
	t.Setenv("KONSUL_JWT_SECRET", "my-secret")
	t.Setenv("KONSUL_JWT_EXPIRY", "30m")
	t.Setenv("KONSUL_REFRESH_EXPIRY", "24h")
	t.Setenv("KONSUL_JWT_ISSUER", "custom-issuer")
	t.Setenv("KONSUL_APIKEY_PREFIX", "custom")
	t.Setenv("KONSUL_REQUIRE_AUTH", "true")
	t.Setenv("KONSUL_PUBLIC_PATHS", "/public,/api/v1/health")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.Auth.Enabled {
		t.Error("expected auth enabled")
	}
	if cfg.Auth.JWTSecret != "my-secret" {
		t.Errorf("expected JWT secret 'my-secret', got %q", cfg.Auth.JWTSecret)
	}
	if cfg.Auth.JWTExpiry != 30*time.Minute {
		t.Errorf("expected JWT expiry 30m, got %v", cfg.Auth.JWTExpiry)
	}
	if cfg.Auth.RefreshExpiry != 24*time.Hour {
		t.Errorf("expected refresh expiry 24h, got %v", cfg.Auth.RefreshExpiry)
	}
	if cfg.Auth.Issuer != "custom-issuer" {
		t.Errorf("expected issuer 'custom-issuer', got %q", cfg.Auth.Issuer)
	}
	if cfg.Auth.APIKeyPrefix != "custom" {
		t.Errorf("expected API key prefix 'custom', got %q", cfg.Auth.APIKeyPrefix)
	}
	if !cfg.Auth.RequireAuth {
		t.Error("expected RequireAuth enabled")
	}
	if len(cfg.Auth.PublicPaths) != 2 {
		t.Errorf("expected 2 public paths, got %d", len(cfg.Auth.PublicPaths))
	}
}

func TestValidate_AuthMissingJWTSecret(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Auth: AuthConfig{
			Enabled:       true,
			JWTSecret:     "",
			JWTExpiry:     15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "konsul",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing JWT secret")
	}
}

func TestValidate_AuthInvalidJWTExpiry(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Auth: AuthConfig{
			Enabled:       true,
			JWTSecret:     "secret",
			JWTExpiry:     0,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "konsul",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for zero JWT expiry")
	}
}

func TestValidate_AuthInvalidRefreshExpiry(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Auth: AuthConfig{
			Enabled:       true,
			JWTSecret:     "secret",
			JWTExpiry:     15 * time.Minute,
			RefreshExpiry: 0,
			Issuer:        "konsul",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for zero refresh expiry")
	}
}

func TestValidate_AuthMissingIssuer(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Auth: AuthConfig{
			Enabled:       true,
			JWTSecret:     "secret",
			JWTExpiry:     15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing issuer")
	}
}

// ACL Configuration Tests
func TestACL_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.ACL.Enabled {
		t.Error("expected ACL disabled by default")
	}
	if cfg.ACL.DefaultPolicy != "deny" {
		t.Errorf("expected default policy 'deny' by default, got %q", cfg.ACL.DefaultPolicy)
	}
	if cfg.ACL.PolicyDir != "./policies" {
		t.Errorf("expected policy dir './policies' by default, got %q", cfg.ACL.PolicyDir)
	}
}

func TestACL_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	t.Setenv("KONSUL_ACL_ENABLED", "true")
	t.Setenv("KONSUL_ACL_DEFAULT_POLICY", "allow")
	t.Setenv("KONSUL_ACL_POLICY_DIR", "/etc/konsul/policies")
	t.Setenv("KONSUL_AUTH_ENABLED", "true")
	t.Setenv("KONSUL_JWT_SECRET", "secret")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.ACL.Enabled {
		t.Error("expected ACL enabled")
	}
	if cfg.ACL.DefaultPolicy != "allow" {
		t.Errorf("expected default policy 'allow', got %q", cfg.ACL.DefaultPolicy)
	}
	if cfg.ACL.PolicyDir != "/etc/konsul/policies" {
		t.Errorf("expected policy dir '/etc/konsul/policies', got %q", cfg.ACL.PolicyDir)
	}
}

func TestValidate_ACLInvalidDefaultPolicy(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		Auth: AuthConfig{
			Enabled:       true,
			JWTSecret:     "secret",
			JWTExpiry:     15 * time.Minute,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "konsul",
		},
		ACL: ACLConfig{
			Enabled:       true,
			DefaultPolicy: "invalid",
			PolicyDir:     "./policies",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for invalid ACL default policy")
	}
}

func TestValidate_ACLRequiresAuth(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080},
		Service: ServiceConfig{
			TTL:             30 * time.Second,
			CleanupInterval: 60 * time.Second,
		},
		Log: LogConfig{Level: "info", Format: "text"},
		ACL: ACLConfig{
			Enabled:       true,
			DefaultPolicy: "deny",
			PolicyDir:     "./policies",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error when ACL enabled but auth disabled")
	}
}

// GraphQL Configuration Tests
func TestGraphQL_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.GraphQL.Enabled {
		t.Error("expected GraphQL disabled by default")
	}
	if !cfg.GraphQL.PlaygroundEnabled {
		t.Error("expected GraphQL playground enabled by default")
	}
}

func TestGraphQL_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	t.Setenv("KONSUL_GRAPHQL_ENABLED", "true")
	t.Setenv("KONSUL_GRAPHQL_PLAYGROUND_ENABLED", "false")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.GraphQL.Enabled {
		t.Error("expected GraphQL enabled")
	}
	if cfg.GraphQL.PlaygroundEnabled {
		t.Error("expected GraphQL playground disabled")
	}
}

// Tracing Configuration Tests
func TestTracing_DefaultValues(t *testing.T) {
	clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Tracing.Enabled {
		t.Error("expected tracing disabled by default")
	}
	if cfg.Tracing.Endpoint != "otel-collector:4318" {
		t.Errorf("expected endpoint 'otel-collector:4318' by default, got %q", cfg.Tracing.Endpoint)
	}
	if cfg.Tracing.ServiceName != "konsul" {
		t.Errorf("expected service name 'konsul' by default, got %q", cfg.Tracing.ServiceName)
	}
	if cfg.Tracing.ServiceVersion != "1.0.0" {
		t.Errorf("expected service version '1.0.0' by default, got %q", cfg.Tracing.ServiceVersion)
	}
	if cfg.Tracing.Environment != "development" {
		t.Errorf("expected environment 'development' by default, got %q", cfg.Tracing.Environment)
	}
	if cfg.Tracing.SamplingRatio != 1.0 {
		t.Errorf("expected sampling ratio 1.0 by default, got %f", cfg.Tracing.SamplingRatio)
	}
	if !cfg.Tracing.InsecureConn {
		t.Error("expected insecure connection enabled by default")
	}
}

func TestTracing_EnvironmentVariables(t *testing.T) {
	clearEnvVars(t)

	t.Setenv("KONSUL_TRACING_ENABLED", "true")
	t.Setenv("KONSUL_TRACING_ENDPOINT", "localhost:4317")
	t.Setenv("KONSUL_TRACING_SERVICE_NAME", "my-service")
	t.Setenv("KONSUL_TRACING_SERVICE_VERSION", "2.0.0")
	t.Setenv("KONSUL_TRACING_ENVIRONMENT", "production")
	t.Setenv("KONSUL_TRACING_SAMPLING_RATIO", "0.5")
	t.Setenv("KONSUL_TRACING_INSECURE", "false")

	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if !cfg.Tracing.Enabled {
		t.Error("expected tracing enabled")
	}
	if cfg.Tracing.Endpoint != "localhost:4317" {
		t.Errorf("expected endpoint 'localhost:4317', got %q", cfg.Tracing.Endpoint)
	}
	if cfg.Tracing.ServiceName != "my-service" {
		t.Errorf("expected service name 'my-service', got %q", cfg.Tracing.ServiceName)
	}
	if cfg.Tracing.ServiceVersion != "2.0.0" {
		t.Errorf("expected service version '2.0.0', got %q", cfg.Tracing.ServiceVersion)
	}
	if cfg.Tracing.Environment != "production" {
		t.Errorf("expected environment 'production', got %q", cfg.Tracing.Environment)
	}
	if cfg.Tracing.SamplingRatio != 0.5 {
		t.Errorf("expected sampling ratio 0.5, got %f", cfg.Tracing.SamplingRatio)
	}
	if cfg.Tracing.InsecureConn {
		t.Error("expected insecure connection disabled")
	}
}

// Helper Function Tests
func TestGetEnvStringSlice(t *testing.T) {
	clearEnvVars(t)

	// Test with valid comma-separated values
	t.Setenv("KONSUL_PUBLIC_PATHS", "/health,/metrics,/api/v1/public")
	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	expectedPaths := []string{"/health", "/metrics", "/api/v1/public"}
	if len(cfg.Auth.PublicPaths) != len(expectedPaths) {
		t.Errorf("expected %d public paths, got %d", len(expectedPaths), len(cfg.Auth.PublicPaths))
	}
	for i, expected := range expectedPaths {
		if i >= len(cfg.Auth.PublicPaths) || cfg.Auth.PublicPaths[i] != expected {
			t.Errorf("expected path[%d] = %q, got %q", i, expected, cfg.Auth.PublicPaths[i])
		}
	}
}

func TestGetEnvStringSlice_WithSpaces(t *testing.T) {
	clearEnvVars(t)

	// Test with spaces around values
	t.Setenv("KONSUL_PUBLIC_PATHS", " /health , /metrics , /api/v1/public ")
	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	expectedPaths := []string{"/health", "/metrics", "/api/v1/public"}
	if len(cfg.Auth.PublicPaths) != len(expectedPaths) {
		t.Errorf("expected %d public paths, got %d", len(expectedPaths), len(cfg.Auth.PublicPaths))
	}
	for i, expected := range expectedPaths {
		if i >= len(cfg.Auth.PublicPaths) || cfg.Auth.PublicPaths[i] != expected {
			t.Errorf("expected path[%d] = %q, got %q", i, expected, cfg.Auth.PublicPaths[i])
		}
	}
}

func TestGetEnvStringSlice_Empty(t *testing.T) {
	clearEnvVars(t)

	// Test with empty string
	t.Setenv("KONSUL_PUBLIC_PATHS", "")
	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should fall back to default
	if len(cfg.Auth.PublicPaths) != 7 {
		t.Errorf("expected 7 default public paths, got %d", len(cfg.Auth.PublicPaths))
	}
}

func TestSplitAndTrim(t *testing.T) {
	testCases := []struct {
		input     string
		delimiter string
		expected  []string
	}{
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{" a , b , c ", ",", []string{"a", "b", "c"}},
		{"one", ",", []string{"one"}},
		{"a::b::c", "::", []string{"a", "b", "c"}},
	}

	for _, tc := range testCases {
		result := splitAndTrim(tc.input, tc.delimiter)
		if len(result) != len(tc.expected) {
			t.Errorf("splitAndTrim(%q, %q) = %v, expected %v", tc.input, tc.delimiter, result, tc.expected)
			continue
		}
		for i, expected := range tc.expected {
			if result[i] != expected {
				t.Errorf("splitAndTrim(%q, %q)[%d] = %q, expected %q", tc.input, tc.delimiter, i, result[i], expected)
			}
		}
	}

	// Test empty string separately as splitString returns empty slice for empty input
	emptyResult := splitAndTrim("", ",")
	if len(emptyResult) != 0 {
		t.Errorf("splitAndTrim(\"\", \",\") = %v, expected empty slice", emptyResult)
	}
}

func TestTrimSpace(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"hello", "hello"},
		{"\t\nhello\r\n", "hello"},
		{"  ", ""},
		{"", ""},
		{"no spaces", "no spaces"},
	}

	for _, tc := range testCases {
		result := trimSpace(tc.input)
		if result != tc.expected {
			t.Errorf("trimSpace(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestGetEnvBool_InvalidValue(t *testing.T) {
	clearEnvVars(t)

	// Test invalid boolean value
	t.Setenv("KONSUL_DNS_ENABLED", "invalid")
	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should fall back to default (true for DNS)
	if !cfg.DNS.Enabled {
		t.Error("expected DNS enabled as default for invalid boolean value")
	}
}

func TestGetEnvFloat_InvalidValue(t *testing.T) {
	clearEnvVars(t)

	// Test invalid float value
	t.Setenv("KONSUL_RATE_LIMIT_REQUESTS_PER_SEC", "invalid")
	defer clearEnvVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should fall back to default (100.0)
	if cfg.RateLimit.RequestsPerSec != 100.0 {
		t.Errorf("expected default 100.0 for invalid float value, got %f", cfg.RateLimit.RequestsPerSec)
	}
}

// clearEnvVars clears all KONSUL environment variables
func clearEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("KONSUL_HOST", "")
	t.Setenv("KONSUL_PORT", "")
	t.Setenv("KONSUL_SERVICE_TTL", "")
	t.Setenv("KONSUL_CLEANUP_INTERVAL", "")
	t.Setenv("KONSUL_LOG_LEVEL", "")
	t.Setenv("KONSUL_LOG_FORMAT", "")
	t.Setenv("KONSUL_PERSISTENCE_ENABLED", "")
	t.Setenv("KONSUL_PERSISTENCE_TYPE", "")
	t.Setenv("KONSUL_DATA_DIR", "")
	t.Setenv("KONSUL_BACKUP_DIR", "")
	t.Setenv("KONSUL_SYNC_WRITES", "")
	t.Setenv("KONSUL_WAL_ENABLED", "")
	t.Setenv("KONSUL_DNS_ENABLED", "")
	t.Setenv("KONSUL_DNS_HOST", "")
	t.Setenv("KONSUL_DNS_PORT", "")
	t.Setenv("KONSUL_DNS_DOMAIN", "")
	t.Setenv("KONSUL_TLS_ENABLED", "")
	t.Setenv("KONSUL_TLS_CERT_FILE", "")
	t.Setenv("KONSUL_TLS_KEY_FILE", "")
	t.Setenv("KONSUL_TLS_AUTO_CERT", "")
	t.Setenv("KONSUL_RATE_LIMIT_ENABLED", "")
	t.Setenv("KONSUL_RATE_LIMIT_REQUESTS_PER_SEC", "")
	t.Setenv("KONSUL_RATE_LIMIT_BURST", "")
	t.Setenv("KONSUL_RATE_LIMIT_BY_IP", "")
	t.Setenv("KONSUL_RATE_LIMIT_BY_APIKEY", "")
	t.Setenv("KONSUL_RATE_LIMIT_CLEANUP", "")
	t.Setenv("KONSUL_AUTH_ENABLED", "")
	t.Setenv("KONSUL_JWT_SECRET", "")
	t.Setenv("KONSUL_JWT_EXPIRY", "")
	t.Setenv("KONSUL_REFRESH_EXPIRY", "")
	t.Setenv("KONSUL_JWT_ISSUER", "")
	t.Setenv("KONSUL_APIKEY_PREFIX", "")
	t.Setenv("KONSUL_REQUIRE_AUTH", "")
	t.Setenv("KONSUL_PUBLIC_PATHS", "")
	t.Setenv("KONSUL_ACL_ENABLED", "")
	t.Setenv("KONSUL_ACL_DEFAULT_POLICY", "")
	t.Setenv("KONSUL_ACL_POLICY_DIR", "")
	t.Setenv("KONSUL_GRAPHQL_ENABLED", "")
	t.Setenv("KONSUL_GRAPHQL_PLAYGROUND_ENABLED", "")
	t.Setenv("KONSUL_TRACING_ENABLED", "")
	t.Setenv("KONSUL_TRACING_ENDPOINT", "")
	t.Setenv("KONSUL_TRACING_SERVICE_NAME", "")
	t.Setenv("KONSUL_TRACING_SERVICE_VERSION", "")
	t.Setenv("KONSUL_TRACING_ENVIRONMENT", "")
	t.Setenv("KONSUL_TRACING_SAMPLING_RATIO", "")
	t.Setenv("KONSUL_TRACING_INSECURE", "")
	t.Setenv("KONSUL_ADMIN_UI_ENABLED", "")
	t.Setenv("KONSUL_ADMIN_UI_PATH", "")
	t.Setenv("KONSUL_WATCH_ENABLED", "")
	t.Setenv("KONSUL_WATCH_BUFFER_SIZE", "")
	t.Setenv("KONSUL_WATCH_MAX_PER_CLIENT", "")
	t.Setenv("KONSUL_AUDIT_ENABLED", "")
	t.Setenv("KONSUL_AUDIT_SINK", "")
	t.Setenv("KONSUL_AUDIT_FILE_PATH", "")
	t.Setenv("KONSUL_AUDIT_BUFFER_SIZE", "")
	t.Setenv("KONSUL_AUDIT_FLUSH_INTERVAL", "")
	t.Setenv("KONSUL_AUDIT_DROP_POLICY", "")
}
