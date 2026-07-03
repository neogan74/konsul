package namespace_test

import (
	"errors"
	"testing"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/neogan74/konsul/internal/namespace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storeFactory lets us run the same suite against both backends.
type storeFactory func(t *testing.T) namespace.NamespaceStore

func testSuite(t *testing.T, factory storeFactory) {
	t.Helper()

	t.Run("default namespace pre-exists", func(t *testing.T) {
		s := factory(t)
		ok, err := s.Exists(namespace.DefaultNamespace)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("create and get", func(t *testing.T) {
		s := factory(t)
		err := s.Create(&namespace.Namespace{Name: "team-a", Description: "Team A"})
		require.NoError(t, err)
		ns, err := s.Get("team-a")
		require.NoError(t, err)
		assert.Equal(t, "team-a", ns.Name)
		assert.Equal(t, "Team A", ns.Description)
		assert.False(t, ns.CreatedAt.IsZero())
	})

	t.Run("duplicate create returns ErrNamespaceExists", func(t *testing.T) {
		s := factory(t)
		require.NoError(t, s.Create(&namespace.Namespace{Name: "dup"}))
		err := s.Create(&namespace.Namespace{Name: "dup"})
		assert.True(t, errors.Is(err, namespace.ErrNamespaceExists))
	})

	t.Run("get missing returns ErrNamespaceNotFound", func(t *testing.T) {
		s := factory(t)
		_, err := s.Get("no-such-ns")
		assert.True(t, errors.Is(err, namespace.ErrNamespaceNotFound))
	})

	t.Run("list includes default and created", func(t *testing.T) {
		s := factory(t)
		require.NoError(t, s.Create(&namespace.Namespace{Name: "listed"}))
		all, err := s.List()
		require.NoError(t, err)
		names := make(map[string]bool, len(all))
		for _, ns := range all {
			names[ns.Name] = true
		}
		assert.True(t, names[namespace.DefaultNamespace])
		assert.True(t, names["listed"])
	})

	t.Run("delete removes namespace", func(t *testing.T) {
		s := factory(t)
		require.NoError(t, s.Create(&namespace.Namespace{Name: "to-delete"}))
		require.NoError(t, s.Delete("to-delete"))
		ok, err := s.Exists("to-delete")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("delete default returns ErrNamespaceReserved", func(t *testing.T) {
		s := factory(t)
		err := s.Delete(namespace.DefaultNamespace)
		assert.True(t, errors.Is(err, namespace.ErrNamespaceReserved))
	})

	t.Run("delete system returns ErrNamespaceReserved", func(t *testing.T) {
		s := factory(t)
		err := s.Delete(namespace.SystemNamespace)
		assert.True(t, errors.Is(err, namespace.ErrNamespaceReserved))
	})

	t.Run("delete missing returns ErrNamespaceNotFound", func(t *testing.T) {
		s := factory(t)
		err := s.Delete("ghost")
		assert.True(t, errors.Is(err, namespace.ErrNamespaceNotFound))
	})

	t.Run("exists returns false for unknown", func(t *testing.T) {
		s := factory(t)
		ok, err := s.Exists("nope")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestMemoryStore(t *testing.T) {
	testSuite(t, func(t *testing.T) namespace.NamespaceStore {
		return namespace.NewMemoryStore()
	})
}

func TestBadgerStore(t *testing.T) {
	testSuite(t, func(t *testing.T) namespace.NamespaceStore {
		opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil)
		db, err := badger.Open(opts)
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		store, err := namespace.NewBadgerStore(db)
		require.NoError(t, err)
		return store
	})
}

func TestValidateName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"default", false},
		{"team-a", false},
		{"a", false},
		{"ab12", false},
		{"", true},
		{"-leading", true},
		{"UPPER", true},
		{"has space", true},
		{"has_underscore", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := namespace.ValidateName(tc.name)
			if tc.wantErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, namespace.ErrInvalidNamespace))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsProtected(t *testing.T) {
	assert.True(t, namespace.IsProtected(namespace.DefaultNamespace))
	assert.True(t, namespace.IsProtected(namespace.SystemNamespace))
	assert.False(t, namespace.IsProtected("team-a"))
}
