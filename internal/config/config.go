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
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Host string
	Port int
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

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Host: getEnvString("KONSUL_HOST", ""),
			Port: getEnvInt("KONSUL_PORT", 8888),
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