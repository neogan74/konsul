package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRendererBasic(t *testing.T) {
	// Setup mock stores
	kvStore := NewMockKVStore()
	kvStore.Set("app/name", "konsul")
	kvStore.Set("app/version", "1.0.0")

	serviceStore := NewMockServiceStore()
	serviceStore.Register(Service{
		Name:    "web",
		Address: "10.0.0.1",
		Port:    8080,
	})

	ctx := &RenderContext{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
		DryRun:       true,
	}

	renderer := NewRenderer(ctx)

	// Create a temporary template file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tpl")
	templateContent := `Application: {{ kv "app/name" }}
Version: {{ kv "app/version" }}
Services:
{{- range services }}
  - {{ .Name }}: {{ .Address }}:{{ .Port }}
{{- end }}
`

	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	// Render the template
	config := Config{
		Source:      templatePath,
		Destination: filepath.Join(tmpDir, "output.txt"),
	}

	result, err := renderer.Render(config)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expectedContent := `Application: konsul
Version: 1.0.0
Services:
  - web: 10.0.0.1:8080
`

	if result.Content != expectedContent {
		t.Errorf("Render() content mismatch\nGot:\n%s\nWant:\n%s", result.Content, expectedContent)
	}
}

func TestRendererFileWrite(t *testing.T) {
	kvStore := NewMockKVStore()
	kvStore.Set("greeting", "Hello, World!")

	ctx := &RenderContext{
		KVStore: kvStore,
		DryRun:  false, // Actually write files
	}

	renderer := NewRenderer(ctx)

	// Create template
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tpl")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := `{{ kv "greeting" }}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	config := Config{
		Source:      templatePath,
		Destination: outputPath,
		Perms:       0644,
	}

	result, err := renderer.Render(config)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !result.Written {
		t.Errorf("Render() file was not written")
	}

	// Verify file was written
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Output file content = %q, want %q", string(content), "Hello, World!")
	}
}

func TestRendererBackup(t *testing.T) {
	kvStore := NewMockKVStore()
	kvStore.Set("value", "new value")

	ctx := &RenderContext{
		KVStore: kvStore,
		DryRun:  false,
	}

	renderer := NewRenderer(ctx)

	// Create template and existing output file
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tpl")
	outputPath := filepath.Join(tmpDir, "output.txt")
	backupPath := outputPath + ".bak"

	// Write initial content
	initialContent := "old value"
	if err := os.WriteFile(outputPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	// Write template
	templateContent := `{{ kv "value" }}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	config := Config{
		Source:      templatePath,
		Destination: outputPath,
		Backup:      true,
		Perms:       0644,
	}

	_, err := renderer.Render(config)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Verify backup was created
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(backupContent) != initialContent {
		t.Errorf("Backup content = %q, want %q", string(backupContent), initialContent)
	}

	// Verify new content was written
	newContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(newContent) != "new value" {
		t.Errorf("Output content = %q, want %q", string(newContent), "new value")
	}
}

func TestRendererDryRun(t *testing.T) {
	kvStore := NewMockKVStore()
	kvStore.Set("test", "value")

	ctx := &RenderContext{
		KVStore: kvStore,
		DryRun:  true,
	}

	renderer := NewRenderer(ctx)

	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test.tpl")
	outputPath := filepath.Join(tmpDir, "output.txt")

	templateContent := `{{ kv "test" }}`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}

	config := Config{
		Source:      templatePath,
		Destination: outputPath,
	}

	result, err := renderer.Render(config)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if result.Written {
		t.Errorf("Render() file was written in dry-run mode")
	}

	// Verify file was NOT created
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Errorf("Output file should not exist in dry-run mode")
	}

	// But content should be rendered
	if result.Content != "value" {
		t.Errorf("Render() content = %q, want %q", result.Content, "value")
	}
}
