package namespace

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	badger "github.com/dgraph-io/badger/v4"
)

const badgerPrefix = "_namespace:"

// BadgerNamespaceStore persists namespaces in BadgerDB under "_namespace:<name>" keys.
type BadgerNamespaceStore struct {
	db *badger.DB
}

// NewBadgerStore returns a BadgerNamespaceStore and ensures the default namespace exists.
func NewBadgerStore(db *badger.DB) (*BadgerNamespaceStore, error) {
	s := &BadgerNamespaceStore{db: db}
	exists, err := s.Exists(DefaultNamespace)
	if err != nil {
		return nil, err
	}
	if !exists {
		now := time.Now().UTC()
		if err := s.Create(&Namespace{Name: DefaultNamespace, CreatedAt: now, UpdatedAt: now}); err != nil {
			return nil, fmt.Errorf("seed default namespace: %w", err)
		}
	}
	return s, nil
}

func (s *BadgerNamespaceStore) Create(ns *Namespace) error {
	if err := ValidateName(ns.Name); err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(badgerPrefix + ns.Name)
		if _, err := txn.Get(key); err == nil {
			return fmt.Errorf("%w: %s", ErrNamespaceExists, ns.Name)
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}
		now := time.Now().UTC()
		cp := *ns
		cp.CreatedAt = now
		cp.UpdatedAt = now
		data, err := json.Marshal(&cp)
		if err != nil {
			return err
		}
		return txn.Set(key, data)
	})
}

func (s *BadgerNamespaceStore) Get(name string) (*Namespace, error) {
	var ns Namespace
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(badgerPrefix + name))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return fmt.Errorf("%w: %s", ErrNamespaceNotFound, name)
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &ns)
		})
	})
	if err != nil {
		return nil, err
	}
	return &ns, nil
}

func (s *BadgerNamespaceStore) List() ([]*Namespace, error) {
	var out []*Namespace
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(badgerPrefix)
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			var ns Namespace
			if err := it.Item().Value(func(val []byte) error {
				return json.Unmarshal(val, &ns)
			}); err != nil {
				return err
			}
			cp := ns
			out = append(out, &cp)
		}
		return nil
	})
	return out, err
}

func (s *BadgerNamespaceStore) Delete(name string) error {
	if IsProtected(name) {
		return fmt.Errorf("%w: %s", ErrNamespaceReserved, name)
	}
	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(badgerPrefix + name)
		if _, err := txn.Get(key); errors.Is(err, badger.ErrKeyNotFound) {
			return fmt.Errorf("%w: %s", ErrNamespaceNotFound, name)
		}
		return txn.Delete(key)
	})
}

func (s *BadgerNamespaceStore) Exists(name string) (bool, error) {
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte(badgerPrefix + name))
		return err
	})
	if errors.Is(err, badger.ErrKeyNotFound) {
		return false, nil
	}
	return err == nil, err
}
