package persistence

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/neogan74/konsul/internal/logger"
)

const (
	kvPrefix      = "kv:"
	servicePrefix = "svc:"
)

// BadgerEngine implements Engine using BadgerDB
type BadgerEngine struct {
	db  *badger.DB
	log logger.Logger
}

// NewBadgerEngine creates a new BadgerDB persistence engine
func NewBadgerEngine(dataDir string, syncWrites bool, log logger.Logger) (*BadgerEngine, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	opts := badger.DefaultOptions(dataDir)
	opts.SyncWrites = syncWrites
	opts.Logger = nil // Disable BadgerDB internal logging or wrap it

	// WAL configuration for crash recovery
	opts.ValueLogFileSize = 64 << 20 // 64MB value log files
	opts.MemTableSize = 64 << 20     // 64MB memtable
	opts.NumMemtables = 5            // Keep 5 memtables in memory
	opts.NumLevelZeroTables = 5      // Maximum L0 tables before compaction
	opts.NumLevelZeroTablesStall = 10 // Stall writes when this many L0 tables

	// Enable compression for better storage efficiency
	opts.Compression = 1 // Snappy compression

	// Configure for durability
	if syncWrites {
		opts.SyncWrites = true
		log.Info("WAL enabled with synchronous writes for maximum durability")
	} else {
		opts.SyncWrites = false
		log.Info("WAL enabled with asynchronous writes for better performance")
	}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}

	engine := &BadgerEngine{
		db:  db,
		log: log,
	}

	// Start garbage collection routine
	go engine.runGarbageCollection()

	log.Info("BadgerDB persistence engine initialized with WAL support",
		logger.String("data_dir", dataDir),
		logger.String("sync_writes", fmt.Sprintf("%t", syncWrites)))

	return engine, nil
}

func (b *BadgerEngine) runGarbageCollection() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err := b.db.RunValueLogGC(0.5)
		if err != nil && !errors.Is(err, badger.ErrNoRewrite) {
			b.log.Warn("BadgerDB garbage collection failed", logger.Error(err))
		}
	}
}

func (b *BadgerEngine) Get(key string) ([]byte, error) {
	var value []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(kvPrefix + key))
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, errors.New("key not found")
	}
	return value, err
}

func (b *BadgerEngine) Set(key string, value []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(kvPrefix+key), value)
	})
}

func (b *BadgerEngine) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(kvPrefix + key))
	})
}

func (b *BadgerEngine) List(prefix string) ([]string, error) {
	var keys []string
	searchPrefix := kvPrefix + prefix

	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(searchPrefix)
		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			key := string(item.Key())
			// Remove the kvPrefix from the key
			keys = append(keys, strings.TrimPrefix(key, kvPrefix))
		}
		return nil
	})
	return keys, err
}

func (b *BadgerEngine) GetService(name string) ([]byte, error) {
	var data []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(servicePrefix + name))
		if err != nil {
			return err
		}

		// Check if expired
		if item.ExpiresAt() != 0 && time.Now().Unix() > int64(item.ExpiresAt()) {
			return errors.New("service expired")
		}

		data, err = item.ValueCopy(nil)
		return err
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, errors.New("service not found")
	}
	return data, err
}

func (b *BadgerEngine) SetService(name string, data []byte, ttl time.Duration) error {
	return b.db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry([]byte(servicePrefix+name), data).WithTTL(ttl)
		return txn.SetEntry(e)
	})
}

func (b *BadgerEngine) DeleteService(name string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(servicePrefix + name))
	})
}

func (b *BadgerEngine) ListServices() ([]string, error) {
	var names []string
	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefixBytes := []byte(servicePrefix)
		now := time.Now().Unix()

		for it.Seek(prefixBytes); it.ValidForPrefix(prefixBytes); it.Next() {
			item := it.Item()
			// Skip expired items
			if item.ExpiresAt() != 0 && now > int64(item.ExpiresAt()) {
				continue
			}
			key := string(item.Key())
			names = append(names, strings.TrimPrefix(key, servicePrefix))
		}
		return nil
	})
	return names, err
}

func (b *BadgerEngine) BatchSet(items map[string][]byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		for key, value := range items {
			if err := txn.Set([]byte(kvPrefix+key), value); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BadgerEngine) BatchDelete(keys []string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		for _, key := range keys {
			if err := txn.Delete([]byte(kvPrefix + key)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BadgerEngine) Close() error {
	return b.db.Close()
}

func (b *BadgerEngine) Backup(path string) error {
	// Ensure backup directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	_, err = b.db.Backup(file, 0)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	b.log.Info("Backup completed successfully", logger.String("path", path))
	return nil
}

func (b *BadgerEngine) Restore(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	err = b.db.Load(file, 256)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	b.log.Info("Restore completed successfully", logger.String("path", path))
	return nil
}

func (b *BadgerEngine) BeginTx() (Transaction, error) {
	txn := b.db.NewTransaction(true)
	return &badgerTx{txn: txn}, nil
}

// badgerTx implements Transaction for BadgerDB
type badgerTx struct {
	txn *badger.Txn
}

func (tx *badgerTx) Set(key string, value []byte) error {
	return tx.txn.Set([]byte(kvPrefix+key), value)
}

func (tx *badgerTx) Delete(key string) error {
	return tx.txn.Delete([]byte(kvPrefix + key))
}

func (tx *badgerTx) Commit() error {
	return tx.txn.Commit()
}

func (tx *badgerTx) Rollback() error {
	tx.txn.Discard()
	return nil
}

// ExportData exports all data to JSON (useful for debugging)
func (b *BadgerEngine) ExportData() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	kvData := make(map[string]string)
	svcData := make(map[string]interface{})

	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			var value []byte
			err := item.Value(func(val []byte) error {
				value = append([]byte{}, val...)
				return nil
			})
			if err != nil {
				return err
			}

			if strings.HasPrefix(key, kvPrefix) {
				kvData[strings.TrimPrefix(key, kvPrefix)] = string(value)
			} else if strings.HasPrefix(key, servicePrefix) {
				var svcInfo map[string]interface{}
				if err := json.Unmarshal(value, &svcInfo); err == nil {
					svcData[strings.TrimPrefix(key, servicePrefix)] = svcInfo
				}
			}
		}
		return nil
	})

	result["kv"] = kvData
	result["services"] = svcData
	return result, err
}

// ImportData imports data from JSON
func (b *BadgerEngine) ImportData(data map[string]interface{}) error {
	return b.db.Update(func(txn *badger.Txn) error {
		// Import KV data
		if kvData, ok := data["kv"].(map[string]interface{}); ok {
			for key, value := range kvData {
				if strVal, ok := value.(string); ok {
					if err := txn.Set([]byte(kvPrefix+key), []byte(strVal)); err != nil {
						return err
					}
				}
			}
		}

		// Import service data
		if svcData, ok := data["services"].(map[string]interface{}); ok {
			for name, svcInfo := range svcData {
				jsonData, err := json.Marshal(svcInfo)
				if err != nil {
					return err
				}
				if err := txn.Set([]byte(servicePrefix+name), jsonData); err != nil {
					return err
				}
			}
		}

		return nil
	})
}