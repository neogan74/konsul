package healthcheck

import (
	"context"
	"time"
)

type CheckType string

const (
	CheckTypeTTL  CheckType = "ttl"
	CheckTypeHTTP CheckType = "http"
	CheckTypeTCP  CheckType = "tcp"
	CheckTypeGRPC CheckType = "grpc"
)

type Status string

const (
	StatusPassing Status = "passing"
	StatusWarning Status = "warning"
	StatusCritical Status = "critical"
)

type Check struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	ServiceID    string            `json:"service_id"`
	Type         CheckType         `json:"type"`
	Status       Status            `json:"status"`
	Output       string            `json:"output"`
	LastCheck    time.Time         `json:"last_check"`

	// Common fields
	Interval     time.Duration     `json:"interval"`
	Timeout      time.Duration     `json:"timeout"`

	// HTTP specific
	HTTP         string            `json:"http,omitempty"`
	Method       string            `json:"method,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	TLSSkipVerify bool             `json:"tls_skip_verify,omitempty"`

	// TCP specific
	TCP          string            `json:"tcp,omitempty"`

	// gRPC specific
	GRPC         string            `json:"grpc,omitempty"`
	GRPCUseTLS   bool              `json:"grpc_use_tls,omitempty"`

	// TTL specific
	TTL          time.Duration     `json:"ttl,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at,omitempty"`
}

type CheckDefinition struct {
	ID           string            `json:"id,omitempty"`
	Name         string            `json:"name"`
	ServiceID    string            `json:"service_id,omitempty"`

	// Check type and common settings
	HTTP         string            `json:"http,omitempty"`
	TCP          string            `json:"tcp,omitempty"`
	GRPC         string            `json:"grpc,omitempty"`
	TTL          string            `json:"ttl,omitempty"`

	Interval     string            `json:"interval,omitempty"`
	Timeout      string            `json:"timeout,omitempty"`

	// HTTP specific
	Method       string            `json:"method,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	TLSSkipVerify bool             `json:"tls_skip_verify,omitempty"`

	// gRPC specific
	GRPCUseTLS   bool              `json:"grpc_use_tls,omitempty"`
}

type Checker interface {
	Check(ctx context.Context, check *Check) (Status, string, error)
}

type CheckResult struct {
	Status Status
	Output string
	Error  error
}