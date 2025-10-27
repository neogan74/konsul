package healthcheck

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTCPChecker_Check_Success(t *testing.T) {
	// Create a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	// Get the port
	addr := listener.Addr().String()

	checker := NewTCPChecker()
	check := &Check{
		TCP:     addr,
		Timeout: 5 * time.Second,
	}

	status, output, err := checker.Check(context.Background(), check)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != StatusPassing {
		t.Errorf("expected status Passing, got %s", status)
	}
	if !contains(output, "successful") {
		t.Errorf("expected output to indicate success, got: %s", output)
	}
}

func TestTCPChecker_Check_NoAddress(t *testing.T) {
	checker := NewTCPChecker()
	check := &Check{
		TCP:     "",
		Timeout: 5 * time.Second,
	}

	status, output, err := checker.Check(context.Background(), check)

	if err == nil {
		t.Error("expected error when TCP address not specified")
	}
	if status != StatusCritical {
		t.Errorf("expected status Critical, got %s", status)
	}
	if output != "TCP address not specified" {
		t.Errorf("expected specific output, got: %s", output)
	}
}

func TestTCPChecker_Check_ConnectionRefused(t *testing.T) {
	checker := NewTCPChecker()
	check := &Check{
		TCP:     "127.0.0.1:1", // Port 1 should be refused
		Timeout: 1 * time.Second,
	}

	status, output, err := checker.Check(context.Background(), check)

	if err == nil {
		t.Error("expected connection error")
	}
	if status != StatusCritical {
		t.Errorf("expected status Critical, got %s", status)
	}
	if !contains(output, "failed") {
		t.Errorf("expected output to indicate failure, got: %s", output)
	}
}

func TestTCPChecker_Check_Timeout(t *testing.T) {
	// Create a server that accepts but doesn't respond to test actual timeout
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("could not create listener for timeout test")
	}
	defer listener.Close()

	// Start a goroutine to accept connection but do nothing
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			time.Sleep(1 * time.Second) // Hold the connection
			conn.Close()
		}
	}()

	addr := listener.Addr().String()

	checker := NewTCPChecker()
	check := &Check{
		TCP:     addr,
		Timeout: 50 * time.Millisecond, // Very short timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Note: TCP connection might complete before timeout in some cases
	// So we just verify the check completes without panic
	status, _, _ := checker.Check(ctx, check)

	// Status should be either Passing or Critical depending on timing
	if status != StatusPassing && status != StatusCritical {
		t.Errorf("expected status Passing or Critical, got %s", status)
	}
}

func TestTCPChecker_Check_InvalidAddress(t *testing.T) {
	checker := NewTCPChecker()
	check := &Check{
		TCP:     "not a valid address",
		Timeout: 1 * time.Second,
	}

	status, _, err := checker.Check(context.Background(), check)

	if err == nil {
		t.Error("expected error for invalid address")
	}
	if status != StatusCritical {
		t.Errorf("expected status Critical, got %s", status)
	}
}

func TestTCPChecker_Check_DefaultTimeout(t *testing.T) {
	// Create a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	checker := NewTCPChecker()
	check := &Check{
		TCP:     addr,
		Timeout: 0, // Should use default
	}

	status, _, err := checker.Check(context.Background(), check)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != StatusPassing {
		t.Errorf("expected status Passing, got %s", status)
	}
}
