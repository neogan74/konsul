package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig
	Service     ServiceConfig
	Log         LogConfig
	Persistence PersistenceConfig
	DNS         DNSConfig
	RateLimit   RateLimitConfig
	Auth        AuthConfig
	Tracing     TracingConfig
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host string
	Port int
	TLS  TLSConfig
}

// TLSConfig contains TLS/SSL configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
	AutoCert bool // Auto-generate self-signed cert for development
}

// ServiceConfig contains service discovery configuration
type ServiceConfig struct {
	TTL             time.Duration
	CleanupInterval time.Duration
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level  string
	Format string
}

// PersistenceConfig contains persistence configuration
type PersistenceConfig struct {
	Enabled    bool
	Type       string // "memory", "badger"
	DataDir    string
	BackupDir  string
	SyncWrites bool
	WALEnabled bool
}

// DNSConfig contains DNS server configuration
type DNSConfig struct {
	Enabled bool
	Host    string
	Port    int
	Domain  string
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool
	RequestsPerSec  float64
	Burst           int
	ByIP            bool
	ByAPIKey        bool
	CleanupInterval time.Duration
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled       bool
	JWTSecret     string
	JWTExpiry     time.Duration
	RefreshExpiry time.Duration
	Issuer        string
	APIKeyPrefix  string
	RequireAuth   bool
	PublicPaths   []string
}

// TracingConfig contains OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled        bool
	Endpoint       string
	ServiceName    string
	ServiceVersion string
	Environment    string
	SamplingRatio  float64
	InsecureConn   bool
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Host: getEnvString("KONSUL_HOST", ""),
			Port: getEnvInt("KONSUL_PORT", 8888),
			TLS: TLSConfig{
				Enabled:  getEnvBool("KONSUL_TLS_ENABLED", false),
				CertFile: getEnvString("KONSUL_TLS_CERT_FILE", ""),
				KeyFile:  getEnvString("KONSUL_TLS_KEY_FILE", ""),
				AutoCert: getEnvBool("KONSUL_TLS_AUTO_CERT", false),
			},
		},
		Service: ServiceConfig{
			TTL:             getEnvDuration("KONSUL_SERVICE_TTL", 30*time.Second),
			CleanupInterval: getEnvDuration("KONSUL_CLEANUP_INTERVAL", 60*time.Second),
		},
		Log: LogConfig{
			Level:  getEnvString("KONSUL_LOG_LEVEL", "info"),
			Format: getEnvString("KONSUL_LOG_FORMAT", "text"),
		},
		Persistence: PersistenceConfig{
			Enabled:    getEnvBool("KONSUL_PERSISTENCE_ENABLED", false),
			Type:       getEnvString("KONSUL_PERSISTENCE_TYPE", "badger"),
			DataDir:    getEnvString("KONSUL_DATA_DIR", "./data"),
			BackupDir:  getEnvString("KONSUL_BACKUP_DIR", "./backups"),
			SyncWrites: getEnvBool("KONSUL_SYNC_WRITES", true),
			WALEnabled: getEnvBool("KONSUL_WAL_ENABLED", true),
		},
		DNS: DNSConfig{
			Enabled: getEnvBool("KONSUL_DNS_ENABLED", true),
			Host:    getEnvString("KONSUL_DNS_HOST", ""),
			Port:    getEnvInt("KONSUL_DNS_PORT", 8600),
			Domain:  getEnvString("KONSUL_DNS_DOMAIN", "consul"),
		},
		RateLimit: RateLimitConfig{
			Enabled:         getEnvBool("KONSUL_RATE_LIMIT_ENABLED", false),
			RequestsPerSec:  getEnvFloat("KONSUL_RATE_LIMIT_REQUESTS_PER_SEC", 100.0),
			Burst:           getEnvInt("KONSUL_RATE_LIMIT_BURST", 20),
			ByIP:            getEnvBool("KONSUL_RATE_LIMIT_BY_IP", true),
			ByAPIKey:        getEnvBool("KONSUL_RATE_LIMIT_BY_APIKEY", false),
			CleanupInterval: getEnvDuration("KONSUL_RATE_LIMIT_CLEANUP", 5*time.Minute),
		},
		Auth: AuthConfig{
			Enabled:       getEnvBool("KONSUL_AUTH_ENABLED", false),
			JWTSecret:     getEnvString("KONSUL_JWT_SECRET", ""),
			JWTExpiry:     getEnvDuration("KONSUL_JWT_EXPIRY", 15*time.Minute),
			RefreshExpiry: getEnvDuration("KONSUL_REFRESH_EXPIRY", 7*24*time.Hour),
			Issuer:        getEnvString("KONSUL_JWT_ISSUER", "konsul"),
			APIKeyPrefix:  getEnvString("KONSUL_APIKEY_PREFIX", "konsul"),
			RequireAuth:   getEnvBool("KONSUL_REQUIRE_AUTH", false),
			PublicPaths:   getEnvStringSlice("KONSUL_PUBLIC_PATHS", []string{"/health", "/health/live", "/health/ready", "/metrics"}),
		},
		Tracing: TracingConfig{
			Enabled:        getEnvBool("KONSUL_TRACING_ENABLED", false),
			Endpoint:       getEnvString("KONSUL_TRACING_ENDPOINT", "otel-collector:4318"),
			ServiceName:    getEnvString("KONSUL_TRACING_SERVICE_NAME", "konsul"),
			ServiceVersion: getEnvString("KONSUL_TRACING_SERVICE_VERSION", "1.0.0"),
			Environment:    getEnvString("KONSUL_TRACING_ENVIRONMENT", "development"),
			SamplingRatio:  getEnvFloat("KONSUL_TRACING_SAMPLING_RATIO", 1.0),
			InsecureConn:   getEnvBool("KONSUL_TRACING_INSECURE", true),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Server.Port)
	}

	// Validate TLS configuration if enabled
	if c.Server.TLS.Enabled {
		if !c.Server.TLS.AutoCert {
			if c.Server.TLS.CertFile == "" {
				return fmt.Errorf("TLS cert file must be specified when TLS is enabled")
			}
			if c.Server.TLS.KeyFile == "" {
				return fmt.Errorf("TLS key file must be specified when TLS is enabled")
			}
		}
	}

	if c.Service.TTL <= 0 {
		return fmt.Errorf("invalid service TTL: %v (must be positive)", c.Service.TTL)
	}

	if c.Service.CleanupInterval <= 0 {
		return fmt.Errorf("invalid cleanup interval: %v (must be positive)", c.Service.CleanupInterval)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Log.Level)
	}

	validLogFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validLogFormats[c.Log.Format] {
		return fmt.Errorf("invalid log format: %s (must be text or json)", c.Log.Format)
	}

	// Validate persistence configuration if enabled
	if c.Persistence.Enabled {
		validPersistenceTypes := map[string]bool{
			"memory": true,
			"badger": true,
		}
		if !validPersistenceTypes[c.Persistence.Type] {
			return fmt.Errorf("invalid persistence type: %s (must be memory or badger)", c.Persistence.Type)
		}

		if c.Persistence.DataDir == "" {
			return fmt.Errorf("data directory must be specified when persistence is enabled")
		}
	}

	// Validate DNS configuration if enabled
	if c.DNS.Enabled {
		if c.DNS.Port <= 0 || c.DNS.Port > 65535 {
			return fmt.Errorf("invalid DNS port: %d (must be 1-65535)", c.DNS.Port)
		}

		if c.DNS.Domain == "" {
			return fmt.Errorf("DNS domain must be specified when DNS is enabled")
		}
	}

	// Validate rate limit configuration if enabled
	if c.RateLimit.Enabled {
		if c.RateLimit.RequestsPerSec <= 0 {
			return fmt.Errorf("rate limit requests per second must be positive")
		}

		if c.RateLimit.Burst <= 0 {
			return fmt.Errorf("rate limit burst must be positive")
		}

		if !c.RateLimit.ByIP && !c.RateLimit.ByAPIKey {
			return fmt.Errorf("rate limiting must be enabled for at least IP or API key")
		}
	}

	// Validate auth configuration if enabled
	if c.Auth.Enabled || c.Auth.RequireAuth {
		if c.Auth.JWTSecret == "" {
			return fmt.Errorf("JWT secret must be specified when auth is enabled")
		}

		if c.Auth.JWTExpiry <= 0 {
			return fmt.Errorf("JWT expiry must be positive")
		}

		if c.Auth.RefreshExpiry <= 0 {
			return fmt.Errorf("refresh expiry must be positive")
		}

		if c.Auth.Issuer == "" {
			return fmt.Errorf("JWT issuer must be specified when auth is enabled")
		}
	}

	return nil
}

// Address returns the server address in host:port format
func (c *Config) Address() string {
	if c.Server.Host == "" {
		return fmt.Sprintf(":%d", c.Server.Port)
	}
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// getEnvString gets a string environment variable with a default value
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration gets a duration environment variable with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvFloat gets a float environment variable with a default value
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvStringSlice gets a comma-separated string environment variable as a slice with a default value
func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		result := []string{}
		for _, v := range splitAndTrim(value, ",") {
			if v != "" {
				result = append(result, v)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// splitAndTrim splits a string by delimiter and trims spaces from each element
func splitAndTrim(s, delimiter string) []string {
	parts := []string{}
	for _, part := range splitString(s, delimiter) {
		trimmed := trimSpace(part)
		parts = append(parts, trimmed)
	}
	return parts
}

// splitString splits a string by delimiter
func splitString(s, delimiter string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	current := ""
	for i := 0; i < len(s); i++ {
		if i+len(delimiter) <= len(s) && s[i:i+len(delimiter)] == delimiter {
			result = append(result, current)
			current = ""
			i += len(delimiter) - 1
		} else {
			current += string(s[i])
		}
	}
	result = append(result, current)
	return result
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
