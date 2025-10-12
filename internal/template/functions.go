package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// FuncMap returns the template function map
func (ctx *RenderContext) FuncMap() template.FuncMap {
	return template.FuncMap{
		// KV Store functions
		"kv":     ctx.kv,
		"kvTree": ctx.kvTree,
		"kvList": ctx.kvList,

		// Service discovery functions
		"service":  ctx.service,
		"services": ctx.services,

		// Utility functions
		"env":  env,
		"file": file,

		// String manipulation
		"toLower":   strings.ToLower,
		"toUpper":   strings.ToUpper,
		"trim":      strings.TrimSpace,
		"split":     strings.Split,
		"join":      strings.Join,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
	}
}

// kv retrieves a value from the KV store
// Usage: {{ kv "config/database/host" }}
func (ctx *RenderContext) kv(key string) (string, error) {
	if ctx.KVStore == nil {
		return "", fmt.Errorf("KV store not available")
	}

	value, ok := ctx.KVStore.Get(key)
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return value, nil
}

// kvTree retrieves all key-value pairs under a prefix
// Usage: {{ range kvTree "config/" }}{{ .Key }}: {{ .Value }}{{ end }}
func (ctx *RenderContext) kvTree(prefix string) ([]KVPair, error) {
	if ctx.KVStore == nil {
		return nil, fmt.Errorf("KV store not available")
	}

	keys := ctx.KVStore.List()
	var pairs []KVPair

	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			if value, ok := ctx.KVStore.Get(key); ok {
				pairs = append(pairs, KVPair{
					Key:   key,
					Value: value,
				})
			}
		}
	}

	return pairs, nil
}

// kvList returns all keys under a prefix
// Usage: {{ range kvList "config/" }}{{ . }}{{ end }}
func (ctx *RenderContext) kvList(prefix string) ([]string, error) {
	if ctx.KVStore == nil {
		return nil, fmt.Errorf("KV store not available")
	}

	keys := ctx.KVStore.List()
	var filtered []string

	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			filtered = append(filtered, key)
		}
	}

	return filtered, nil
}

// service retrieves all instances of a service
// Usage: {{ range service "web" }}{{ .Address }}:{{ .Port }}{{ end }}
func (ctx *RenderContext) service(name string) ([]Service, error) {
	if ctx.ServiceStore == nil {
		return nil, fmt.Errorf("service store not available")
	}

	// Get specific service
	svc, ok := ctx.ServiceStore.Get(name)
	if !ok {
		return []Service{}, nil // Return empty slice if service not found
	}

	return []Service{svc}, nil
}

// services retrieves all registered services
// Usage: {{ range services }}{{ .Name }}: {{ .Address }}:{{ .Port }}{{ end }}
func (ctx *RenderContext) services() ([]Service, error) {
	if ctx.ServiceStore == nil {
		return nil, fmt.Errorf("service store not available")
	}

	return ctx.ServiceStore.List(), nil
}

// env retrieves an environment variable
// Usage: {{ env "HOME" }}
func env(key string) string {
	return os.Getenv(key)
}

// file reads a file and returns its contents
// Usage: {{ file "/etc/hostname" }}
func file(path string) (string, error) {
	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %s: %w", path, err)
	}

	// Read file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", absPath, err)
	}

	return string(content), nil
}

// KVPair represents a key-value pair
type KVPair struct {
	Key   string
	Value string
}
