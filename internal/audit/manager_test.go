package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/logger"
)

func TestManagerDisabledIsNoop(t *testing.T) {
	mgr, err := NewManager(Config{Enabled: false}, logger.GetDefault())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := mgr.Record(context.Background(), &Event{
		Action:   "kv.set",
		Result:   "success",
		Resource: Resource{Type: "kv", ID: "foo"},
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManagerFileSinkWritesEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	cfg := Config{
		Enabled:       true,
		Sink:          "file",
		FilePath:      path,
		BufferSize:    8,
		FlushInterval: 5 * time.Millisecond,
		DropPolicy:    DropPolicyBlock,
	}

	mgr, err := NewManager(cfg, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if _, err := mgr.Record(context.Background(), &Event{
		Action:   "kv.set",
		Result:   "success",
		Resource: Resource{Type: "kv", ID: "foo"},
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	if !strings.Contains(string(data), "\"action\":\"kv.set\"") {
		t.Fatalf("audit log missing action, got: %s", string(data))
	}
}

func TestManagerRejectsWritesAfterShutdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")

	cfg := Config{
		Enabled:       true,
		Sink:          "file",
		FilePath:      path,
		BufferSize:    1,
		FlushInterval: 5 * time.Millisecond,
		DropPolicy:    DropPolicyBlock,
	}

	mgr, err := NewManager(cfg, logger.GetDefault())
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := mgr.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if _, err := mgr.Record(context.Background(), &Event{
		Action:   "kv.set",
		Result:   "success",
		Resource: Resource{Type: "kv", ID: "bar"},
	}); err == nil {
		t.Fatalf("expected error when recording after shutdown")
	}
}
