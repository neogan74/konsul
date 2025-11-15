package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func newWriter(cfg Config) (Writer, error) {
	switch cfg.Sink {
	case "stdout":
		return &stdoutWriter{
			encoder: json.NewEncoder(os.Stdout),
		}, nil
	case "file":
		return newFileWriter(cfg.FilePath)
	default:
		return nil, fmt.Errorf("unsupported audit sink: %s", cfg.Sink)
	}
}

type stdoutWriter struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func (w *stdoutWriter) Write(event *Event) error {
	if event == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.encoder.Encode(event)
}

func (w *stdoutWriter) Flush() error {
	return nil
}

func (w *stdoutWriter) Close(context.Context) error {
	return nil
}

type fileWriter struct {
	mu     sync.Mutex
	writer *bufio.Writer
	file   *os.File
}

func newFileWriter(path string) (*fileWriter, error) {
	if path == "" {
		return nil, fmt.Errorf("audit file path cannot be empty")
	}

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create audit log directory: %w", err)
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	return &fileWriter{
		writer: bufio.NewWriter(file),
		file:   file,
	}, nil
}

func (w *fileWriter) Write(event *Event) error {
	if event == nil {
		return nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.writer.Write(payload); err != nil {
		return err
	}
	if err := w.writer.WriteByte('\n'); err != nil {
		return err
	}
	return nil
}

func (w *fileWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Flush()
}

func (w *fileWriter) Close(ctx context.Context) error {
	done := make(chan struct{})
	var flushErr error

	go func() {
		flushErr = w.Flush()
		if err := w.file.Close(); err != nil && flushErr == nil {
			flushErr = err
		}
		close(done)
	}()

	select {
	case <-done:
		return flushErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
