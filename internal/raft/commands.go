// Package raft implements Raft consensus for Konsul clustering.
// This enables multi-node deployments with automatic leader election,
// log replication, and strong consistency guarantees.
package raft

import (
	"encoding/json"
	"time"

	"github.com/neogan74/konsul/internal/store"
)

// CommandType represents the type of operation to apply to the FSM.
type CommandType uint8

const (
	// CmdKVSet sets a key-value pair
	CmdKVSet CommandType = iota
	// CmdKVSetWithFlags sets a key-value pair with custom flags
	CmdKVSetWithFlags
	// CmdKVSetCAS sets a key-value pair with CAS
	CmdKVSetCAS
	// CmdKVDelete deletes a key
	CmdKVDelete
	// CmdKVDeleteCAS deletes a key with CAS
	CmdKVDeleteCAS
	// CmdKVBatchSet sets multiple key-value pairs atomically
	CmdKVBatchSet
	// CmdKVBatchSetCAS sets multiple key-value pairs with CAS
	CmdKVBatchSetCAS
	// CmdKVBatchDelete deletes multiple keys atomically
	CmdKVBatchDelete
	// CmdKVBatchDeleteCAS deletes multiple keys with CAS
	CmdKVBatchDeleteCAS

	// CmdServiceRegister registers a service
	CmdServiceRegister
	// CmdServiceRegisterCAS registers a service with CAS
	CmdServiceRegisterCAS
	// CmdServiceDeregister deregisters a service
	CmdServiceDeregister
	// CmdServiceDeregisterCAS deregisters a service with CAS
	CmdServiceDeregisterCAS
	// CmdServiceHeartbeat updates service TTL
	CmdServiceHeartbeat

	// CmdHealthTTLUpdate updates health check TTL
	CmdHealthTTLUpdate
)

// Command represents a Raft log entry command.
// Commands are serialized and replicated across the cluster.
type Command struct {
	Type      CommandType `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Payload   []byte      `json:"payload"`
}

// NewCommand creates a new command with the given type and payload.
func NewCommand(cmdType CommandType, payload interface{}) (*Command, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Command{
		Type:      cmdType,
		Timestamp: time.Now().Unix(),
		Payload:   data,
	}, nil
}

// Marshal serializes the command for Raft log entry.
func (c *Command) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// UnmarshalCommand deserializes a command from Raft log entry.
func UnmarshalCommand(data []byte) (*Command, error) {
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

// --- Payload Definitions ---

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
