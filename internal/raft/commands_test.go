package raft

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmdType     CommandType
		payload     interface{}
		expectError bool
	}{
		{
			name:    "KV Set",
			cmdType: CmdKVSet,
			payload: KVSetPayload{Key: "foo", Value: "bar"},
		},
		{
			name:    "KV Set With Flags",
			cmdType: CmdKVSetWithFlags,
			payload: KVSetWithFlagsPayload{Key: "foo", Value: "bar", Flags: 123},
		},
		{
			name:    "KV Delete",
			cmdType: CmdKVDelete,
			payload: KVDeletePayload{Key: "foo"},
		},
		{
			name:    "KV Batch Set",
			cmdType: CmdKVBatchSet,
			payload: KVBatchSetPayload{Items: map[string]string{"a": "1", "b": "2"}},
		},
		{
			name:    "KV Batch Delete",
			cmdType: CmdKVBatchDelete,
			payload: KVBatchDeletePayload{Keys: []string{"a", "b", "c"}},
		},
		{
			name:    "Service Register",
			cmdType: CmdServiceRegister,
			payload: ServiceRegisterPayload{
				Name:    "web",
				Address: "10.0.0.1",
				Port:    8080,
				Tags:    []string{"primary"},
				Meta:    map[string]string{"version": "1.0"},
			},
		},
		{
			name:    "Service Deregister",
			cmdType: CmdServiceDeregister,
			payload: ServiceDeregisterPayload{Name: "web"},
		},
		{
			name:    "Service Heartbeat",
			cmdType: CmdServiceHeartbeat,
			payload: ServiceHeartbeatPayload{Name: "web"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := NewCommand(tt.cmdType, tt.payload)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.cmdType, cmd.Type)
			assert.NotEmpty(t, cmd.Payload)
		})
	}
}

func TestCommand_Marshal_Unmarshal(t *testing.T) {
	// Create a command
	original, err := NewCommand(CmdKVSet, KVSetPayload{Key: "test", Value: "value"})
	require.NoError(t, err)

	// Marshal
	data, err := original.Marshal()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal
	restored, err := UnmarshalCommand(data)
	require.NoError(t, err)
	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, original.Payload, restored.Payload)
}

func TestUnmarshalCommand_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "invalid json",
			data: []byte("not json"),
		},
		{
			name: "incomplete json",
			data: []byte("{"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalCommand(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestCommandTypeName(t *testing.T) {
	tests := []struct {
		cmdType  CommandType
		expected string
	}{
		{CmdKVSet, "kv_set"},
		{CmdKVSetWithFlags, "kv_set_flags"},
		{CmdKVDelete, "kv_delete"},
		{CmdKVBatchSet, "kv_batch_set"},
		{CmdKVBatchDelete, "kv_batch_delete"},
		{CmdServiceRegister, "service_register"},
		{CmdServiceDeregister, "service_deregister"},
		{CmdServiceHeartbeat, "service_heartbeat"},
		{CommandType(255), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, CommandTypeName(tt.cmdType))
		})
	}
}
