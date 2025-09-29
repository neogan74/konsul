package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables
	clearEnvVars()

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
}

func TestLoad_EnvironmentVariables(t *testing.T) {
	// Clear environment variables first
	clearEnvVars()

	// Set environment variables
	os.Setenv("KONSUL_HOST", "localhost")
	os.Setenv("KONSUL_PORT", "9999")
	os.Setenv("KONSUL_SERVICE_TTL", "45s")
	os.Setenv("KONSUL_CLEANUP_INTERVAL", "2m")
	os.Setenv("KONSUL_LOG_LEVEL", "debug")
	os.Setenv("KONSUL_LOG_FORMAT", "json")

	defer clearEnvVars()

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
	clearEnvVars()

	// Test invalid port
	os.Setenv("KONSUL_PORT", "invalid")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	// Should fall back to default
	if cfg.Server.Port != 8888 {
		t.Errorf("expected default port 8888 for invalid env value, got %d", cfg.Server.Port)
	}

	// Test invalid duration
	os.Setenv("KONSUL_SERVICE_TTL", "invalid")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	// Should fall back to default
	if cfg.Service.TTL != 30*time.Second {
		t.Errorf("expected default TTL 30s for invalid env value, got %v", cfg.Service.TTL)
	}

	clearEnvVars()
}

func TestLoad_InvalidConfigValidation(t *testing.T) {
	clearEnvVars()

	// Set invalid port that will fail validation
	os.Setenv("KONSUL_PORT", "0")
	defer clearEnvVars()

	_, err := Load()
	if err == nil {
		t.Error("expected Load() to fail validation with invalid port")
	}
}

func TestDNS_DefaultValues(t *testing.T) {
	clearEnvVars()

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
	clearEnvVars()

	// Set DNS environment variables
	os.Setenv("KONSUL_DNS_ENABLED", "false")
	os.Setenv("KONSUL_DNS_HOST", "127.0.0.1")
	os.Setenv("KONSUL_DNS_PORT", "5353")
	os.Setenv("KONSUL_DNS_DOMAIN", "local")

	defer clearEnvVars()

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
	clearEnvVars()

	// Set invalid DNS port
	os.Setenv("KONSUL_DNS_PORT", "0")
	defer clearEnvVars()

	_, err := Load()
	if err == nil {
		t.Error("expected Load() to fail validation with invalid DNS port")
	}
}

func TestValidate_DNSInvalidDomain(t *testing.T) {
	clearEnvVars()

	// Set empty DNS domain
	os.Setenv("KONSUL_DNS_DOMAIN", "")
	defer clearEnvVars()

	_, err := Load()
	if err == nil {
		t.Error("expected Load() to fail validation with empty DNS domain")
	}
}

func TestValidate_DNSValidConfig(t *testing.T) {
	clearEnvVars()

	// Set valid DNS configuration
	os.Setenv("KONSUL_DNS_ENABLED", "true")
	os.Setenv("KONSUL_DNS_HOST", "localhost")
	os.Setenv("KONSUL_DNS_PORT", "53")
	os.Setenv("KONSUL_DNS_DOMAIN", "internal")

	defer clearEnvVars()

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

// clearEnvVars clears all KONSUL environment variables
func clearEnvVars() {
	os.Unsetenv("KONSUL_HOST")
	os.Unsetenv("KONSUL_PORT")
	os.Unsetenv("KONSUL_SERVICE_TTL")
	os.Unsetenv("KONSUL_CLEANUP_INTERVAL")
	os.Unsetenv("KONSUL_LOG_LEVEL")
	os.Unsetenv("KONSUL_LOG_FORMAT")
	os.Unsetenv("KONSUL_PERSISTENCE_ENABLED")
	os.Unsetenv("KONSUL_PERSISTENCE_TYPE")
	os.Unsetenv("KONSUL_DATA_DIR")
	os.Unsetenv("KONSUL_BACKUP_DIR")
	os.Unsetenv("KONSUL_SYNC_WRITES")
	os.Unsetenv("KONSUL_WAL_ENABLED")
	os.Unsetenv("KONSUL_DNS_ENABLED")
	os.Unsetenv("KONSUL_DNS_HOST")
	os.Unsetenv("KONSUL_DNS_PORT")
	os.Unsetenv("KONSUL_DNS_DOMAIN")
}