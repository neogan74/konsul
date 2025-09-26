package healthcheck

import (
	"context"
	"fmt"
	"net"
	"time"
)

type TCPChecker struct{}

func NewTCPChecker() *TCPChecker {
	return &TCPChecker{}
}

func (t *TCPChecker) Check(ctx context.Context, check *Check) (Status, string, error) {
	if check.TCP == "" {
		return StatusCritical, "TCP address not specified", fmt.Errorf("TCP address required")
	}

	timeout := check.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Use context with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", check.TCP)
	duration := time.Since(start)

	if err != nil {
		output := fmt.Sprintf("TCP connection to %s failed after %v: %v", check.TCP, duration, err)
		return StatusCritical, output, err
	}

	// Immediately close the connection since we only need to test connectivity
	conn.Close()

	output := fmt.Sprintf("TCP connection to %s successful (%.3fs)", check.TCP, duration.Seconds())
	return StatusPassing, output, nil
}