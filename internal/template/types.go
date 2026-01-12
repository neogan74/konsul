package template

import (
	"time"
)

// ConfigEngine holds the configuration for the template engine
type ConfigEngine struct {
	// Templates is the list of templates to render
	Templates []Config `json:"templates"`

	// KonsulAddr is the address of the Konsul server
	KonsulAddr string `json:"konsul_addr"`

	// Token is the authentication token for Konsul
	Token string `json:"token,omitempty"`

	// Once determines if the engine should run once and exit
	Once bool `json:"once"`

	// DryRun will render templates but not write files or execute commands
	DryRun bool `json:"dry_run"`

	// Wait is the minimum and maximum time to wait before rendering
	Wait *WaitConfig `json:"wait,omitempty"`
}

// Config defines a single template configuration
type Config struct {
	// Source is the path to the template file
	Source string `json:"source"`

	// Destination is the path where the rendered template will be written
	Destination string `json:"destination"`

	// Command to execute after successful rendering
	Command string `json:"command,omitempty"`

	// CommandTimeout is the maximum time to wait for command execution
	CommandTimeout time.Duration `json:"command_timeout,omitempty"`

	// Perms are the file permissions to set on the destination file (e.g., 0644)
	Perms uint32 `json:"perms,omitempty"`

	// Backup determines if a backup should be created before overwriting
	Backup bool `json:"backup"`

	// Wait overrides the global wait configuration for this template
	Wait *WaitConfig `json:"wait,omitempty"`
}

// WaitConfig defines wait timing for de-duplication
type WaitConfig struct {
	// Min is the minimum time to wait before rendering
	Min time.Duration `json:"min"`

	// Max is the maximum time to wait before rendering
	Max time.Duration `json:"max"`
}

// RenderContext holds the context for template rendering
type RenderContext struct {
	// KVStore provides access to KV operations
	KVStore KVStoreReader

	// ServiceStore provides access to service operations
	ServiceStore ServiceStoreReader

	// DryRun indicates if this is a dry-run
	DryRun bool
}

// KVStoreReader interface for reading KV data
type KVStoreReader interface {
	Get(key string) (string, bool)
	List() []string
}

// ServiceStoreReader interface for reading service data
type ServiceStoreReader interface {
	List() []Service
	Get(name string) (Service, bool)
}

// Service represents a registered service (matching store.Service)
type Service struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

// Data holds the data available in templates
type Data struct {
}

// RenderResult contains the result of template rendering
type RenderResult struct {
	// Template is the template configuration that was rendered
	Template Config

	// Content is the rendered content
	Content string

	// Written indicates if the file was written
	Written bool

	// CommandExecuted indicates if the command was executed
	CommandExecuted bool

	// CommandOutput is the output from the command
	CommandOutput string

	// Error is any error that occurred
	Error error

	// Duration is how long the render took
	Duration time.Duration
}
