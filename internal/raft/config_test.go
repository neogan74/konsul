package raft

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 1000*time.Millisecond, cfg.HeartbeatTimeout)
	assert.Equal(t, 1000*time.Millisecond, cfg.ElectionTimeout)
	assert.Equal(t, 500*time.Millisecond, cfg.LeaderLeaseTimeout)
	assert.Equal(t, 50*time.Millisecond, cfg.CommitTimeout)
	assert.Equal(t, 120*time.Second, cfg.SnapshotInterval)
	assert.Equal(t, uint64(8192), cfg.SnapshotThreshold)
	assert.Equal(t, 2, cfg.SnapshotRetention)
	assert.Equal(t, 64, cfg.MaxAppendEntries)
	assert.Equal(t, uint64(10240), cfg.TrailingLogs)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				NodeID:            "node1",
				BindAddr:          "0.0.0.0:7000",
				DataDir:           "/tmp/raft",
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   1 * time.Second,
				SnapshotThreshold: 1000,
			},
			expectError: false,
		},
		{
			name: "missing node id",
			config: &Config{
				BindAddr:          "0.0.0.0:7000",
				DataDir:           "/tmp/raft",
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   1 * time.Second,
				SnapshotThreshold: 1000,
			},
			expectError: true,
			errorMsg:    "NodeID is required",
		},
		{
			name: "missing bind addr",
			config: &Config{
				NodeID:            "node1",
				DataDir:           "/tmp/raft",
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   1 * time.Second,
				SnapshotThreshold: 1000,
			},
			expectError: true,
			errorMsg:    "BindAddr is required",
		},
		{
			name: "missing data dir",
			config: &Config{
				NodeID:            "node1",
				BindAddr:          "0.0.0.0:7000",
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   1 * time.Second,
				SnapshotThreshold: 1000,
			},
			expectError: true,
			errorMsg:    "DataDir is required",
		},
		{
			name: "zero heartbeat timeout",
			config: &Config{
				NodeID:            "node1",
				BindAddr:          "0.0.0.0:7000",
				DataDir:           "/tmp/raft",
				HeartbeatTimeout:  0,
				ElectionTimeout:   1 * time.Second,
				SnapshotThreshold: 1000,
			},
			expectError: true,
			errorMsg:    "HeartbeatTimeout must be positive",
		},
		{
			name: "zero election timeout",
			config: &Config{
				NodeID:            "node1",
				BindAddr:          "0.0.0.0:7000",
				DataDir:           "/tmp/raft",
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   0,
				SnapshotThreshold: 1000,
			},
			expectError: true,
			errorMsg:    "ElectionTimeout must be positive",
		},
		{
			name: "election timeout less than heartbeat",
			config: &Config{
				NodeID:            "node1",
				BindAddr:          "0.0.0.0:7000",
				DataDir:           "/tmp/raft",
				HeartbeatTimeout:  2 * time.Second,
				ElectionTimeout:   1 * time.Second,
				SnapshotThreshold: 1000,
			},
			expectError: true,
			errorMsg:    "ElectionTimeout should be >= HeartbeatTimeout",
		},
		{
			name: "zero snapshot threshold",
			config: &Config{
				NodeID:            "node1",
				BindAddr:          "0.0.0.0:7000",
				DataDir:           "/tmp/raft",
				HeartbeatTimeout:  1 * time.Second,
				ElectionTimeout:   1 * time.Second,
				SnapshotThreshold: 0,
			},
			expectError: true,
			errorMsg:    "SnapshotThreshold must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_GetAdvertiseAddr(t *testing.T) {
	tests := []struct {
		name          string
		bindAddr      string
		advertiseAddr string
		expected      string
	}{
		{
			name:          "advertise addr set",
			bindAddr:      "0.0.0.0:7000",
			advertiseAddr: "192.168.1.10:7000",
			expected:      "192.168.1.10:7000",
		},
		{
			name:          "advertise addr empty - use bind addr",
			bindAddr:      "10.0.0.1:7000",
			advertiseAddr: "",
			expected:      "10.0.0.1:7000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				BindAddr:      tt.bindAddr,
				AdvertiseAddr: tt.advertiseAddr,
			}
			assert.Equal(t, tt.expected, cfg.GetAdvertiseAddr())
		})
	}
}
