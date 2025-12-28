// Package raft implements Raft consensus for Konsul clustering.
// This enables multi-node deployments with automatic leader election,
// log replication, and strong consistency guarantees.
package raft

import "encoding/json"

// CommandType represents the type of operation to apply to the FSM.
type CommandType uint8

const (
	// CmdKVSet sets a key-value pair
	CmdKVSet CommandType = iota
	// CmdKVSetWithFlags sets a key-value pair with custom flags
	CmdKVSetWithFlags
	// CmdKVDelete deletes a key
	CmdKVDelete
	// CmdKVBatchSet sets multiple key-value pairs atomically
	CmdKVBatchSet
	// CmdKVBatchDelete deletes multiple keys atomically
	CmdKVBatchDelete
	// CmdServiceRegister registers a service
	CmdServiceRegister
	// CmdServiceDeregister deregisters a service
	CmdServiceDeregister
	// CmdServiceHeartbeat updates service TTL
	CmdServiceHeartbeat
)

// Command represents a Raft log entry command.
// Commands are serialized and replicated across the cluster.
type Command struct {
	Type    CommandType `json:"type"`
	Payload []byte      `json:"payload"`
}

// NewCommand creates a new command with the given type and payload.
func NewCommand(cmdType CommandType, payload interface{}) (*Command, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Command{
		Type:    cmdType,
		Payload: data,
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

// --- KV Payloads ---

// KVSetPayload is the payload for CmdKVSet.
type KVSetPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// KVSetWithFlagsPayload is the payload for CmdKVSetWithFlags.
type KVSetWithFlagsPayload struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Flags uint64 `json:"flags"`
}

// KVDeletePayload is the payload for CmdKVDelete.
type KVDeletePayload struct {
	Key string `json:"key"`
}

// KVBatchSetPayload is the payload for CmdKVBatchSet.
type KVBatchSetPayload struct {
	Items map[string]string `json:"items"`
}

// KVBatchDeletePayload is the payload for CmdKVBatchDelete.
type KVBatchDeletePayload struct {
	Keys []string `json:"keys"`
}

// --- Service Payloads ---

// ServiceRegisterPayload is the payload for CmdServiceRegister.
type ServiceRegisterPayload struct {
	Name    string            `json:"name"`
	Address string            `json:"address"`
	Port    int               `json:"port"`
	Tags    []string          `json:"tags,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
	// Note: Health checks are handled separately - they run locally on each node
}

// ServiceDeregisterPayload is the payload for CmdServiceDeregister.
type ServiceDeregisterPayload struct {
	Name string `json:"name"`
}

// ServiceHeartbeatPayload is the payload for CmdServiceHeartbeat.
type ServiceHeartbeatPayload struct {
	Name string `json:"name"`
}
