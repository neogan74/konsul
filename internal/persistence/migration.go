package persistence

import (
	"errors"

	badger "github.com/dgraph-io/badger/v4"
)

const (
	// MigrationFlagKey is written to BadgerDB once the v1 namespace migration completes.
	MigrationFlagKey = "_migrated:v1"
	nsDefaultPrefix  = "ns:default:"
)

// IsMigrated reports whether the one-time namespace key migration has already run.
func IsMigrated(db *badger.DB) (bool, error) {
	err := db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(MigrationFlagKey))
		return err
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return false, nil
	}
	return err == nil, err
}

// MigrateToNamespacedKeys rewrites all bare keys (those not starting with "ns:" or "_")
// to "ns:default:<key>" and marks the migration complete with a flag key.
// Safe to call multiple times — subsequent calls are no-ops.
// Writes the new key BEFORE deleting the old one so a mid-run crash leaves data intact.
func MigrateToNamespacedKeys(db *badger.DB) error {
	done, err := IsMigrated(db)
	if err != nil || done {
		return err
	}

	// Collect bare keys in a read-only scan.
	var keys [][]byte
	if err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			k := it.Item().KeyCopy(nil)
			s := string(k)
			if len(s) > 0 && s[0] != '_' && (len(s) < 3 || s[:3] != "ns:") {
				keys = append(keys, k)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Rewrite in batches of 100.
	const batchSize = 100
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]
		if err := db.Update(func(txn *badger.Txn) error {
			for _, k := range batch {
				item, err := txn.Get(k)
				if errors.Is(err, badger.ErrKeyNotFound) {
					continue // already migrated in a previous partial run
				}
				if err != nil {
					return err
				}
				val, err := item.ValueCopy(nil)
				if err != nil {
					return err
				}
				newKey := []byte(nsDefaultPrefix + string(k))
				if err := txn.Set(newKey, val); err != nil {
					return err
				}
				if err := txn.Delete(k); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// Write migration flag last — only set when fully complete.
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(MigrationFlagKey), []byte("1"))
	})
}
