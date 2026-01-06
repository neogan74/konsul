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
