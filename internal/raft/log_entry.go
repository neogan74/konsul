package raft

import (
	"encoding/json"
	"time"

	"github.com/neogan74/konsul/internal/store"
)

type EntryType string

const (
	EntryKVSet            EntryType = "kv_set"
	EntryKVSetWithFlags   EntryType = "kv_set_with_flags"
	EntryKVSetCAS         EntryType = "kv_set_cas"
	EntryKVDelete         EntryType = "kv_delete"
	EntryKVDeleteCAS      EntryType = "kv_delete_cas"
	EntryKVBatchSet       EntryType = "kv_batch_set"
	EntryKVBatchSetCAS    EntryType = "kv_batch_set_cas"
	EntryKVBatchDelete    EntryType = "kv_batch_delete"
	EntryKVBatchDeleteCAS EntryType = "kv_batch_delete_cas"

	EntryServiceRegister      EntryType = "service_register"
	EntryServiceRegisterCAS   EntryType = "service_register_cas"
	EntryServiceDeregister    EntryType = "service_deregister"
	EntryServiceDeregisterCAS EntryType = "service_deregister_cas"
	EntryServiceHeartbeat     EntryType = "service_heartbeat"

	EntryHealthTTLUpdate EntryType = "health_ttl_update"
)

type LogEntry struct {
	Type      EntryType       `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func NewLogEntry(entryType EntryType, payload any) (LogEntry, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return LogEntry{}, err
	}
	return LogEntry{
		Type:      entryType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}, nil
}

func (e LogEntry) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type KVSetPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KVSetWithFlagsPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Flags uint64 `json:"flags"`
}

type KVSetCASPayload struct {
	Key           string `json:"key"`
	Value         string `json:"value"`
	ExpectedIndex uint64 `json:"expected_index"`
}

type KVDeletePayload struct {
	Key string `json:"key"`
}

type KVDeleteCASPayload struct {
	Key           string `json:"key"`
	ExpectedIndex uint64 `json:"expected_index"`
}

type KVBatchSetPayload struct {
	Items map[string]string `json:"items"`
}

type KVBatchSetCASPayload struct {
	Items           map[string]string `json:"items"`
	ExpectedIndices map[string]uint64 `json:"expected_indices"`
}

type KVBatchDeletePayload struct {
	Keys []string `json:"keys"`
}

type KVBatchDeleteCASPayload struct {
	Keys            []string          `json:"keys"`
	ExpectedIndices map[string]uint64 `json:"expected_indices"`
}

type ServiceRegisterPayload struct {
	Service store.Service `json:"service"`
}

type ServiceRegisterCASPayload struct {
	Service       store.Service `json:"service"`
	ExpectedIndex uint64        `json:"expected_index"`
}

type ServiceDeregisterPayload struct {
	Name string `json:"name"`
}

type ServiceDeregisterCASPayload struct {
	Name          string `json:"name"`
	ExpectedIndex uint64 `json:"expected_index"`
}

type ServiceHeartbeatPayload struct {
	Name string `json:"name"`
}

type HealthTTLUpdatePayload struct {
	CheckID string `json:"check_id"`
}
