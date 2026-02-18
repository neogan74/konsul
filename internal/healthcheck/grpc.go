package healthcheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type GRPCChecker struct{}

func NewGRPCChecker() *GRPCChecker {
	return &GRPCChecker{}
}

func (g *GRPCChecker) Check(ctx context.Context, check *Check) (Status, string, error) {
	if check.GRPC == "" {
		return StatusCritical, "gRPC address not specified", fmt.Errorf("gRPC address required")
	}

	timeout := check.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Configure credentials
	var creds credentials.TransportCredentials
	if check.GRPCUseTLS {
		creds = credentials.NewTLS(&tls.Config{})
	} else {
		creds = insecure.NewCredentials()
	}

	// Dial options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	start := time.Now()
	conn, err := grpc.NewClient(check.GRPC, opts...)
	if err != nil {
		duration := time.Since(start)
		output := fmt.Sprintf("gRPC connection to %s failed after %v: %v", check.GRPC, duration, err)
		return StatusCritical, output, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			// Best-effort close; connection already used for health check.
		}
	}()

	// Create health client
	client := grpc_health_v1.NewHealthClient(conn)

	// Check health
	resp, err := client.Check(ctxWithTimeout, &grpc_health_v1.HealthCheckRequest{
		Service: "", // Empty service name checks overall server health
	})

	duration := time.Since(start)

	if err != nil {
		output := fmt.Sprintf("gRPC health check to %s failed after %v: %v", check.GRPC, duration, err)
		return StatusCritical, output, err
	}

	// Evaluate response
	switch resp.Status {
	case grpc_health_v1.HealthCheckResponse_SERVING:
		output := fmt.Sprintf("gRPC health check to %s successful (%.3fs)", check.GRPC, duration.Seconds())
		return StatusPassing, output, nil
	case grpc_health_v1.HealthCheckResponse_NOT_SERVING:
		output := fmt.Sprintf("gRPC service at %s is not serving (%.3fs)", check.GRPC, duration.Seconds())
		return StatusCritical, output, fmt.Errorf("service not serving")
	case grpc_health_v1.HealthCheckResponse_UNKNOWN:
		output := fmt.Sprintf("gRPC service at %s status unknown (%.3fs)", check.GRPC, duration.Seconds())
		return StatusWarning, output, fmt.Errorf("service status unknown")
	default:
		output := fmt.Sprintf("gRPC service at %s returned unknown status %v (%.3fs)", check.GRPC, resp.Status, duration.Seconds())
		return StatusCritical, output, fmt.Errorf("unknown health status: %v", resp.Status)
	}
}
