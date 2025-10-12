package template

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Executor handles command execution after template rendering
type Executor struct {
	defaultTimeout time.Duration
}

// NewExecutor creates a new command executor
func NewExecutor() *Executor {
	return &Executor{
		defaultTimeout: 30 * time.Second,
	}
}

// Execute runs a command with optional timeout
func (e *Executor) Execute(command string, timeout time.Duration) (string, error) {
	if command == "" {
		return "", nil
	}

	// Use default timeout if not specified
	if timeout == 0 {
		timeout = e.defaultTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Parse command (simple shell invocation)
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if err != nil {
		// Check if context was canceled (timeout)
		if ctx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("command timed out after %v: %s", timeout, output)
		}
		return output, fmt.Errorf("command failed: %w\nOutput: %s", err, output)
	}

	return output, nil
}

// ExecuteWithRetry runs a command with retry logic
func (e *Executor) ExecuteWithRetry(command string, timeout time.Duration, maxRetries int) (string, error) {
	var lastErr error
	var output string

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * time.Second
			time.Sleep(backoff)
		}

		output, lastErr = e.Execute(command, timeout)
		if lastErr == nil {
			return output, nil
		}
	}

	return output, fmt.Errorf("command failed after %d attempts: %w", maxRetries+1, lastErr)
}
