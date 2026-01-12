package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Renderer handles template rendering
type Renderer struct {
	ctx      *RenderContext
	executor *Executor
}

// NewRenderer creates a new template renderer
func NewRenderer(ctx *RenderContext) *Renderer {
	return &Renderer{
		ctx:      ctx,
		executor: NewExecutor(),
	}
}

// Render renders a template with the given configuration
func (r *Renderer) Render(config Config) (*RenderResult, error) {
	start := time.Now()
	result := &RenderResult{
		Template: config,
	}

	// Read template source
	templateContent, err := os.ReadFile(config.Source)
	if err != nil {
		result.Error = fmt.Errorf("failed to read template source %s: %w", config.Source, err)
		result.Duration = time.Since(start)
		return result, result.Error
	}

	// Parse and execute template
	tmpl, err := template.New(filepath.Base(config.Source)).
		Funcs(r.ctx.FuncMap()).
		Parse(string(templateContent))
	if err != nil {
		result.Error = fmt.Errorf("failed to parse template %s: %w", config.Source, err)
		result.Duration = time.Since(start)
		return result, result.Error
	}

	// Execute template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		result.Error = fmt.Errorf("failed to execute template %s: %w", config.Source, err)
		result.Duration = time.Since(start)
		return result, result.Error
	}

	result.Content = buf.String()
	result.Duration = time.Since(start)

	// In dry-run mode, don't write or execute
	if r.ctx.DryRun {
		return result, nil
	}

	// Write to destination
	if config.Destination != "" {
		if err := r.writeFile(config); err != nil {
			result.Error = err
			return result, err
		}
		result.Written = true
	}

	// Execute command if specified
	if config.Command != "" {
		output, err := r.executeCommand(config)
		result.CommandOutput = output
		if err != nil {
			result.Error = err
			return result, err
		}
		result.CommandExecuted = true
	}

	return result, nil
}

// writeFile writes the rendered content to the destination file
func (r *Renderer) writeFile(config Config) error {
	// Ensure destination directory exists
	destDir := filepath.Dir(config.Destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destDir, err)
	}

	// Create backup if requested
	if config.Backup {
		if _, err := os.Stat(config.Destination); err == nil {
			backupPath := config.Destination + ".bak"
			if err := copyFile(config.Destination, backupPath); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}
	}

	// Write to temporary file first (atomic write)
	tempPath := config.Destination + ".tmp"
	perms := os.FileMode(config.Perms)
	if perms == 0 {
		perms = 0644 // Default permissions
	}

	if err := os.WriteFile(tempPath, []byte(r.getRenderedContent(config)), perms); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, config.Destination); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// getRenderedContent retrieves the rendered content for a config
// This is a placeholder - in practice, the content would be passed through
func (r *Renderer) getRenderedContent(config Config) string {
	// This will be replaced with proper content passing
	// For now, re-render (not ideal, but works for initial implementation)
	templateContent, err := os.ReadFile(config.Source)
	if err != nil {
		return ""
	}

	tmpl, err := template.New(filepath.Base(config.Source)).
		Funcs(r.ctx.FuncMap()).
		Parse(string(templateContent))
	if err != nil {
		return ""
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		return ""
	}

	return buf.String()
}

// executeCommand executes the post-render command
func (r *Renderer) executeCommand(config Config) (string, error) {
	timeout := config.CommandTimeout
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	return r.executor.Execute(config.Command, timeout)
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, info.Mode())
}
