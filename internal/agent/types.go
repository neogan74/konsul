package agent

import (
	"time"

	"github.com/neogan74/konsul/internal/healthcheck"
	"github.com/neogan74/konsul/internal/store"
)

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthStatusPassing  HealthStatus = "passing"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// UpdateType represents the type of update (add, update, delete)
type UpdateType string

const (
	UpdateTypeAdd    UpdateType = "add"
	UpdateTypeUpdate UpdateType = "update"
	UpdateTypeDelete UpdateType = "delete"
)

// ServiceUpdate represents a service update in the sync protocol
type ServiceUpdate struct {
	Type        UpdateType          `json:"type"`
	ServiceName string              `json:"service_name"`
	Service     *store.Service      `json:"service,omitempty"`
	Entry       *store.ServiceEntry `json:"entry,omitempty"`
}

// KVUpdate represents a KV update in the sync protocol
type KVUpdate struct {
	Type  UpdateType     `json:"type"`
	Key   string         `json:"key"`
	Entry *store.KVEntry `json:"entry,omitempty"`
}

// HealthUpdate represents a health check status update
type HealthUpdate struct {
	ServiceID string                       `json:"service_id"`
	CheckID   string                       `json:"check_id"`
	Status    HealthStatus                 `json:"status"`
	Output    string                       `json:"output,omitempty"`
	Check     *healthcheck.CheckDefinition `json:"check,omitempty"`
}

// SyncRequest represents a sync request from agent to server
type SyncRequest struct {
	AgentID         string   `json:"agent_id"`
	LastSyncIndex   int64    `json:"last_sync_index"`
	WatchedPrefixes []string `json:"watched_prefixes,omitempty"`
	FullSync        bool     `json:"full_sync"`
}

// SyncResponse represents a sync response from server to agent
type SyncResponse struct {
	CurrentIndex   int64           `json:"current_index"`
	ServiceUpdates []ServiceUpdate `json:"service_updates,omitempty"`
	KVUpdates      []KVUpdate      `json:"kv_updates,omitempty"`
	HealthUpdates  []HealthUpdate  `json:"health_updates,omitempty"`
}

// BatchRegisterRequest represents a batch registration request
type BatchRegisterRequest struct {
	AgentID        string          `json:"agent_id"`
	Services       []store.Service `json:"services"`
	SequenceNumber int64           `json:"sequence_number"`
}

// AgentInfo represents agent identification and metadata
type AgentInfo struct {
	ID         string            `json:"id"`
	NodeName   string            `json:"node_name"`
	NodeIP     string            `json:"node_ip"`
	Datacenter string            `json:"datacenter,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	StartedAt  time.Time         `json:"started_at"`
	Version    string            `json:"version,omitempty"`
}

// AgentStats represents agent runtime statistics
type AgentStats struct {
	CacheHitRate    float64   `json:"cache_hit_rate"`
	CacheEntries    int       `json:"cache_entries"`
	LocalServices   int       `json:"local_services"`
	LastSyncTime    time.Time `json:"last_sync_time"`
	SyncErrorsTotal int64     `json:"sync_errors_total"`
	Uptime          string    `json:"uptime"`
}
