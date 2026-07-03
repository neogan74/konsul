package store_test

import (
	"testing"

	"github.com/neogan74/konsul/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespacedService_RegisterGet(t *testing.T) {
	ss := store.NewServiceStore()
	ns := store.NewNamespacedService(ss, "team-a")

	err := ns.Register(store.Service{Name: "web", Address: "127.0.0.1", Port: 80})
	require.NoError(t, err)

	svc, ok := ns.Get("web")
	require.True(t, ok)
	assert.Equal(t, "web", svc.Name)
	assert.Equal(t, "team-a", svc.Namespace)
}

func TestNamespacedService_Isolation(t *testing.T) {
	ss := store.NewServiceStore()
	a := store.NewNamespacedService(ss, "team-a")
	b := store.NewNamespacedService(ss, "team-b")

	require.NoError(t, a.Register(store.Service{Name: "api", Address: "1.1.1.1", Port: 8080}))
	require.NoError(t, b.Register(store.Service{Name: "api", Address: "2.2.2.2", Port: 9090}))

	svcA, okA := a.Get("api")
	svcB, okB := b.Get("api")
	require.True(t, okA)
	require.True(t, okB)
	assert.Equal(t, "1.1.1.1", svcA.Address)
	assert.Equal(t, "2.2.2.2", svcB.Address)
}

func TestNamespacedService_List(t *testing.T) {
	ss := store.NewServiceStore()
	a := store.NewNamespacedService(ss, "team-a")
	b := store.NewNamespacedService(ss, "team-b")

	require.NoError(t, a.Register(store.Service{Name: "svc1", Address: "1.1.1.1", Port: 1}))
	require.NoError(t, a.Register(store.Service{Name: "svc2", Address: "1.1.1.2", Port: 2}))
	require.NoError(t, b.Register(store.Service{Name: "svc1", Address: "2.2.2.1", Port: 3}))

	listA := a.List()
	assert.Len(t, listA, 2)
	for _, s := range listA {
		assert.NotContains(t, s.Name, "ns:")
	}

	listB := b.List()
	assert.Len(t, listB, 1)
}

func TestNamespacedService_Deregister(t *testing.T) {
	ss := store.NewServiceStore()
	ns := store.NewNamespacedService(ss, "ns1")

	require.NoError(t, ns.Register(store.Service{Name: "gone", Address: "x", Port: 1}))
	ns.Deregister("gone")
	_, ok := ns.Get("gone")
	assert.False(t, ok)
}

func TestNamespacedService_PanicOnEmpty(t *testing.T) {
	ss := store.NewServiceStore()
	assert.Panics(t, func() { store.NewNamespacedService(ss, "") })
}
