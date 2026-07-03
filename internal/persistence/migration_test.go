package persistence_test

import (
	"testing"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/neogan74/konsul/internal/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openInMemDB(t *testing.T) *badger.DB {
	t.Helper()
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
	db, err := badger.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func setKey(t *testing.T, db *badger.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(value))
	}))
}

func getKey(t *testing.T, db *badger.DB, key string) (string, bool) {
	t.Helper()
	var val []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return "", false
	}
	return string(val), true
}

func TestMigrateToNamespacedKeys(t *testing.T) {
	db := openInMemDB(t)

	setKey(t, db, "mykey", "myval")
	setKey(t, db, "another", "val2")

	require.NoError(t, persistence.MigrateToNamespacedKeys(db))

	// Old keys gone.
	_, ok := getKey(t, db, "mykey")
	assert.False(t, ok, "bare key should be removed")

	// New keys present.
	v, ok := getKey(t, db, "ns:default:mykey")
	assert.True(t, ok)
	assert.Equal(t, "myval", v)

	v2, ok := getKey(t, db, "ns:default:another")
	assert.True(t, ok)
	assert.Equal(t, "val2", v2)

	// Flag written.
	_, flagOK := getKey(t, db, persistence.MigrationFlagKey)
	assert.True(t, flagOK)
}

func TestMigrateToNamespacedKeys_Idempotent(t *testing.T) {
	db := openInMemDB(t)
	setKey(t, db, "k", "v")

	require.NoError(t, persistence.MigrateToNamespacedKeys(db))
	require.NoError(t, persistence.MigrateToNamespacedKeys(db)) // second call is no-op

	v, ok := getKey(t, db, "ns:default:k")
	assert.True(t, ok)
	assert.Equal(t, "v", v)
}

func TestMigrateToNamespacedKeys_SkipsInternalKeys(t *testing.T) {
	db := openInMemDB(t)
	setKey(t, db, "_internal", "x")
	setKey(t, db, "ns:already:here", "y")

	require.NoError(t, persistence.MigrateToNamespacedKeys(db))

	// Internal keys untouched.
	v, ok := getKey(t, db, "_internal")
	assert.True(t, ok)
	assert.Equal(t, "x", v)

	// Already-namespaced keys untouched.
	v2, ok := getKey(t, db, "ns:already:here")
	assert.True(t, ok)
	assert.Equal(t, "y", v2)
}

func TestIsMigrated(t *testing.T) {
	db := openInMemDB(t)

	ok, err := persistence.IsMigrated(db)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, persistence.MigrateToNamespacedKeys(db))

	ok, err = persistence.IsMigrated(db)
	require.NoError(t, err)
	assert.True(t, ok)
}
