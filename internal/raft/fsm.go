package raft

import (
	"encoding/json"
	"fmt"
	"io"

	hashiraft "github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

type KonsulFSM struct {
	kvStore      *store.KVStore
	serviceStore *store.ServiceStore
	log          logger.Logger
}

func NewFSM(kvStore *store.KVStore, serviceStore *store.ServiceStore, log logger.Logger) *KonsulFSM {
	return &KonsulFSM{
		kvStore:      kvStore,
		serviceStore: serviceStore,
		log:          log,
	}
}

func (f *KonsulFSM) Apply(logEntry *hashiraft.Log) interface{} {
	var entry LogEntry
	if err := json.Unmarshal(logEntry.Data, &entry); err != nil {
		f.log.Error("Raft FSM: failed to decode log entry", logger.Error(err))
		return err
	}

	switch entry.Type {
	case EntryKVSet:
		var payload KVSetPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		f.kvStore.Set(payload.Key, payload.Value)
		return nil
	case EntryKVSetWithFlags:
		var payload KVSetWithFlagsPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		f.kvStore.SetWithFlags(payload.Key, payload.Value, payload.Flags)
		return nil
	case EntryKVSetCAS:
		var payload KVSetCASPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		newIndex, err := f.kvStore.SetCAS(payload.Key, payload.Value, payload.ExpectedIndex)
		if err != nil {
			return err
		}
		return newIndex
	case EntryKVDelete:
		var payload KVDeletePayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		f.kvStore.Delete(payload.Key)
		return nil
	case EntryKVDeleteCAS:
		var payload KVDeleteCASPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.kvStore.DeleteCAS(payload.Key, payload.ExpectedIndex)
	case EntryKVBatchSet:
		var payload KVBatchSetPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.kvStore.BatchSet(payload.Items)
	case EntryKVBatchSetCAS:
		var payload KVBatchSetCASPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		returnValue, err := f.kvStore.BatchSetCAS(payload.Items, payload.ExpectedIndices)
		if err != nil {
			return err
		}
		return returnValue
	case EntryKVBatchDelete:
		var payload KVBatchDeletePayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.kvStore.BatchDelete(payload.Keys)
	case EntryKVBatchDeleteCAS:
		var payload KVBatchDeleteCASPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.kvStore.BatchDeleteCAS(payload.Keys, payload.ExpectedIndices)
	case EntryServiceRegister:
		var payload ServiceRegisterPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.serviceStore.Register(payload.Service)
	case EntryServiceRegisterCAS:
		var payload ServiceRegisterCASPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		newIndex, err := f.serviceStore.RegisterCAS(payload.Service, payload.ExpectedIndex)
		if err != nil {
			return err
		}
		return newIndex
	case EntryServiceDeregister:
		var payload ServiceDeregisterPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		f.serviceStore.Deregister(payload.Name)
		return nil
	case EntryServiceDeregisterCAS:
		var payload ServiceDeregisterCASPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.serviceStore.DeregisterCAS(payload.Name, payload.ExpectedIndex)
	case EntryServiceHeartbeat:
		var payload ServiceHeartbeatPayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.serviceStore.Heartbeat(payload.Name)
	case EntryHealthTTLUpdate:
		var payload HealthTTLUpdatePayload
		if err := json.Unmarshal(entry.Data, &payload); err != nil {
			return err
		}
		return f.serviceStore.UpdateTTLCheck(payload.CheckID)
	default:
		return fmt.Errorf("unknown raft log entry type: %s", entry.Type)
	}
}

func (f *KonsulFSM) Snapshot() (hashiraft.FSMSnapshot, error) {
	kvEntries, kvIndex := f.kvStore.SnapshotState()
	serviceEntries, serviceIndex := f.serviceStore.SnapshotState()

	return &KonsulSnapshot{
		KVEntries:      kvEntries,
		KVIndex:        kvIndex,
		ServiceEntries: serviceEntries,
		ServiceIndex:   serviceIndex,
		log:            f.log,
	}, nil
}

func (f *KonsulFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var snapshot SnapshotState
	if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
		return err
	}

	f.kvStore.RestoreSnapshot(snapshot.KVEntries, snapshot.KVIndex)
	f.serviceStore.RestoreSnapshot(snapshot.ServiceEntries, snapshot.ServiceIndex)
	return nil
}
