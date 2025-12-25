package raft

import (
	"encoding/json"
	"testing"

	hashiraft "github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

func TestFSMApplyKVSet(t *testing.T) {
	kvStore := store.NewKVStore()
	svcStore := store.NewServiceStore()
	fsm := NewFSM(kvStore, svcStore, logger.GetDefault())

	entry, err := NewLogEntry(EntryKVSet, KVSetPayload{
		Key:   "test-key",
		Value: "value",
	})
	if err != nil {
		t.Fatalf("NewLogEntry failed: %v", err)
	}

	data, err := entry.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	resp := fsm.Apply(&hashiraft.Log{Data: data})
	if resp != nil {
		if applyErr, ok := resp.(error); ok {
			t.Fatalf("Apply returned error: %v", applyErr)
		}
	}

	if val, ok := kvStore.Get("test-key"); !ok || val != "value" {
		t.Fatalf("expected KV to be set, got ok=%v value=%q", ok, val)
	}
}

func TestFSMApplyServiceRegister(t *testing.T) {
	kvStore := store.NewKVStore()
	svcStore := store.NewServiceStore()
	fsm := NewFSM(kvStore, svcStore, logger.GetDefault())

	service := store.Service{
		Name:    "api",
		Address: "127.0.0.1",
		Port:    8080,
	}

	payload := ServiceRegisterPayload{Service: service}
	entry := LogEntry{
		Type: EntryServiceRegister,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("payload marshal failed: %v", err)
	}
	entry.Data = data

	raw, err := entry.Marshal()
	if err != nil {
		t.Fatalf("entry marshal failed: %v", err)
	}

	resp := fsm.Apply(&hashiraft.Log{Data: raw})
	if resp != nil {
		if applyErr, ok := resp.(error); ok {
			t.Fatalf("Apply returned error: %v", applyErr)
		}
	}

	if _, ok := svcStore.Get("api"); !ok {
		t.Fatalf("expected service to be registered")
	}
}
