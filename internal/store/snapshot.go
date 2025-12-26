package store

import (
	"fmt"
	"time"

	"github.com/neogan74/konsul/internal/healthcheck"
	"github.com/neogan74/konsul/internal/logger"
)

// SnapshotState returns a copy of KV entries and current global index.
func (kv *KVStore) SnapshotState() (map[string]KVEntry, uint64) {
	kv.Mutex.RLock()
	defer kv.Mutex.RUnlock()

	entries := make(map[string]KVEntry, len(kv.Data))
	for key, entry := range kv.Data {
		entries[key] = entry
	}
	return entries, kv.globalIndex
}

// RestoreSnapshot replaces KV data with snapshot entries and resets the global index.
func (kv *KVStore) RestoreSnapshot(entries map[string]KVEntry, index uint64) {
	kv.Mutex.Lock()
	defer kv.Mutex.Unlock()

	kv.Data = make(map[string]KVEntry, len(entries))
	for key, entry := range entries {
		kv.Data[key] = entry
	}
	kv.globalIndex = index
}

// SnapshotState returns a copy of service entries and current global index.
func (s *ServiceStore) SnapshotState() (map[string]ServiceEntry, uint64) {
	s.Mutex.RLock()
	defer s.Mutex.RUnlock()

	entries := make(map[string]ServiceEntry, len(s.Data))
	for name, entry := range s.Data {
		entries[name] = entry
	}
	return entries, s.globalIndex
}

// RestoreSnapshot replaces service data with snapshot entries and resets indices.
func (s *ServiceStore) RestoreSnapshot(entries map[string]ServiceEntry, index uint64) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	s.Data = make(map[string]ServiceEntry, len(entries))
	s.TagIndex = make(map[string]map[string]bool)
	s.MetaIndex = make(map[string]map[string][]string)

	for name, entry := range entries {
		s.Data[name] = entry
		s.addToTagIndex(name, entry.Service.Tags)
		s.addToMetaIndex(name, entry.Service.Meta)
	}
	s.globalIndex = index

	s.healthManager.Stop()
	s.healthManager = healthcheck.NewManager(s.log)

	for name, entry := range entries {
		for _, checkDef := range entry.Service.Checks {
			if checkDef.ServiceID == "" {
				checkDef.ServiceID = name
			}
			if checkDef.Name == "" {
				checkDef.Name = fmt.Sprintf("%s-health", name)
			}
			if checkDef.TTL != "" {
				if _, err := time.ParseDuration(checkDef.TTL); err != nil {
					s.log.Warn("Invalid TTL in snapshot check",
						logger.String("service", name),
						logger.String("check", checkDef.Name),
						logger.Error(err))
					continue
				}
			}
			if _, err := s.healthManager.AddCheck(checkDef); err != nil {
				s.log.Error("Failed to restore health check",
					logger.String("service", name),
					logger.String("check", checkDef.Name),
					logger.Error(err))
			}
		}
	}
}
