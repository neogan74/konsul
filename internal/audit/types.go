package audit

import "time"

// Actor describes who initiated an operation.
type Actor struct {
	ID      string   `json:"id,omitempty"`
	Type    string   `json:"type,omitempty"` // user, service, api_key, cli
	Name    string   `json:"name,omitempty"`
	Roles   []string `json:"roles,omitempty"`
	TokenID string   `json:"token_id,omitempty"`
}

// Resource represents the target of an operation.
type Resource struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// Event captures a single auditable action.
type Event struct {
	ID          string            `json:"event_id"`
	Timestamp   time.Time         `json:"timestamp"`
	Action      string            `json:"action"`
	Result      string            `json:"result"`
	Resource    Resource          `json:"resource"`
	Actor       Actor             `json:"actor"`
	SourceIP    string            `json:"source_ip,omitempty"`
	AuthMethod  string            `json:"auth_method,omitempty"`
	HTTPMethod  string            `json:"http_method,omitempty"`
	HTTPPath    string            `json:"http_path,omitempty"`
	HTTPStatus  int               `json:"http_status,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
	SpanID      string            `json:"span_id,omitempty"`
	RequestHash string            `json:"request_hash,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
