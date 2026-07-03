package store_test

import (
	"testing"

	"github.com/neogan74/konsul/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newKVPair() (*store.KVStore, *store.NamespacedKV) {
	kv := store.NewKVStore()
	return kv, store.NewNamespacedKV(kv, "team-a")
}

func TestNamespacedKV_SetGet(t *testing.T) {
	_, nkv := newKVPair()
	nkv.Set("foo", "bar")
	v, ok := nkv.Get("foo")
	require.True(t, ok)
	assert.Equal(t, "bar", v)
}

func TestNamespacedKV_Isolation(t *testing.T) {
	kv := store.NewKVStore()
	a := store.NewNamespacedKV(kv, "team-a")
	b := store.NewNamespacedKV(kv, "team-b")

	a.Set("key", "from-a")
	b.Set("key", "from-b")

	va, okA := a.Get("key")
	vb, okB := b.Get("key")
	require.True(t, okA)
	require.True(t, okB)
	assert.Equal(t, "from-a", va)
	assert.Equal(t, "from-b", vb)
}

func TestNamespacedKV_List_StripsPrefix(t *testing.T) {
	kv := store.NewKVStore()
	a := store.NewNamespacedKV(kv, "team-a")
	b := store.NewNamespacedKV(kv, "team-b")

	a.Set("x", "1")
	a.Set("y", "2")
	b.Set("x", "3")

	keys := a.List()
	assert.ElementsMatch(t, []string{"x", "y"}, keys)
}

func TestNamespacedKV_ListEntries_StripsPrefix(t *testing.T) {
	_, nkv := newKVPair()
	nkv.Set("alpha", "v1")
	entries := nkv.ListEntries()
	_, ok := entries["alpha"]
	assert.True(t, ok)
	// raw ns-prefixed key must not appear
	_, badKey := entries["ns:team-a:alpha"]
	assert.False(t, badKey)
}

func TestNamespacedKV_Delete(t *testing.T) {
	_, nkv := newKVPair()
	nkv.Set("del", "v")
	nkv.Delete("del")
	_, ok := nkv.Get("del")
	assert.False(t, ok)
}

func TestNamespacedKV_SetCAS(t *testing.T) {
	_, nkv := newKVPair()
	nkv.Set("cas", "v1")
	entry, ok := nkv.GetEntry("cas")
	require.True(t, ok)

	newIdx, err := nkv.SetCAS("cas", "v2", entry.ModifyIndex)
	require.NoError(t, err)
	assert.Greater(t, newIdx, entry.ModifyIndex)

	v, _ := nkv.Get("cas")
	assert.Equal(t, "v2", v)
}

func TestNamespacedKV_BatchSet(t *testing.T) {
	_, nkv := newKVPair()
	err := nkv.BatchSet(map[string]string{"a": "1", "b": "2"})
	require.NoError(t, err)

	va, _ := nkv.Get("a")
	vb, _ := nkv.Get("b")
	assert.Equal(t, "1", va)
	assert.Equal(t, "2", vb)
}

func TestNamespacedKV_BatchGet_StripsPrefix(t *testing.T) {
	_, nkv := newKVPair()
	nkv.Set("p", "v")
	found, notFound := nkv.BatchGet([]string{"p", "missing"})
	assert.Equal(t, "v", found["p"])
	assert.ElementsMatch(t, []string{"missing"}, notFound)
}

func TestNamespacedKV_PanicOnEmptyNamespace(t *testing.T) {
	kv := store.NewKVStore()
	assert.Panics(t, func() { store.NewNamespacedKV(kv, "") })
}
