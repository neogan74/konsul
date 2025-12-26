package raft

import (
	"encoding/json"
	"io"

	hashiraft "github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

type SnapshotState struct {
	KVEntries      map[string]store.KVEntry      `json:"kv_entries"`
	KVIndex        uint64                        `json:"kv_index"`
	ServiceEntries map[string]store.ServiceEntry `json:"service_entries"`
	ServiceIndex   uint64                        `json:"service_index"`
}

type KonsulSnapshot struct {
	KVEntries      map[string]store.KVEntry
	KVIndex        uint64
	ServiceEntries map[string]store.ServiceEntry
	ServiceIndex   uint64
	log            logger.Logger
}

func (s *KonsulSnapshot) Persist(sink hashiraft.SnapshotSink) error {
	state := SnapshotState{
		KVEntries:      s.KVEntries,
		KVIndex:        s.KVIndex,
		ServiceEntries: s.ServiceEntries,
		ServiceIndex:   s.ServiceIndex,
	}

	if err := writeSnapshot(sink, state); err != nil {
		if err := sink.Cancel(); err != nil {
			s.log.Error("Raft snapshot cancel failed", logger.Error(err))
		}
		return err
	}
	return sink.Close()
}

func (s *KonsulSnapshot) Release() {}

func writeSnapshot(w io.Writer, state SnapshotState) error {
	return json.NewEncoder(w).Encode(state)
}
