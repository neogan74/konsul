package raft

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- staticDiscoverer ---

func TestStaticDiscoverer_ReturnsConfiguredSeeds(t *testing.T) {
	d := &staticDiscoverer{seeds: []string{"10.0.0.1:8888", "10.0.0.2:8888"}}
	peers, err := d.Discover(context.Background())
	require.NoError(t, err)
	require.Len(t, peers, 2)
	assert.Equal(t, "10.0.0.1:8888", peers[0].Address)
	assert.Equal(t, "10.0.0.2:8888", peers[1].Address)
}

func TestStaticDiscoverer_FiltersEmptySeeds(t *testing.T) {
	d := &staticDiscoverer{seeds: []string{"10.0.0.1:8888", "", "10.0.0.2:8888"}}
	peers, err := d.Discover(context.Background())
	require.NoError(t, err)
	assert.Len(t, peers, 2)
}

// --- NewDiscoverer ---

func TestNewDiscoverer_None(t *testing.T) {
	cfg := &DiscoveryConfig{Method: DiscoveryMethodNone}
	d, err := NewDiscoverer(cfg)
	require.NoError(t, err)
	assert.Nil(t, d)
}

func TestNewDiscoverer_Static(t *testing.T) {
	cfg := &DiscoveryConfig{
		Method: DiscoveryMethodStatic,
		Seeds:  []string{"10.0.0.1:8888"},
	}
	d, err := NewDiscoverer(cfg)
	require.NoError(t, err)
	require.NotNil(t, d)
	_, ok := d.(*staticDiscoverer)
	assert.True(t, ok)
}

func TestNewDiscoverer_StaticWithNoSeeds_ReturnsError(t *testing.T) {
	cfg := &DiscoveryConfig{Method: DiscoveryMethodStatic, Seeds: nil}
	_, err := NewDiscoverer(cfg)
	assert.ErrorContains(t, err, "at least one seed")
}

func TestNewDiscoverer_DNS(t *testing.T) {
	cfg := &DiscoveryConfig{
		Method:    DiscoveryMethodDNS,
		DNSDomain: "konsul.example.com",
		DNSPort:   8888,
	}
	d, err := NewDiscoverer(cfg)
	require.NoError(t, err)
	require.NotNil(t, d)
	_, ok := d.(*dnsDiscoverer)
	assert.True(t, ok)
}

func TestNewDiscoverer_DNSWithNoDomain_ReturnsError(t *testing.T) {
	cfg := &DiscoveryConfig{Method: DiscoveryMethodDNS}
	_, err := NewDiscoverer(cfg)
	assert.ErrorContains(t, err, "domain name")
}

func TestNewDiscoverer_UnknownMethod_ReturnsError(t *testing.T) {
	cfg := &DiscoveryConfig{Method: "cloud"}
	_, err := NewDiscoverer(cfg)
	assert.ErrorContains(t, err, "unknown discovery method")
}

// --- tryJoinPeer ---

func TestTryJoinPeer_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/cluster/join", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var req joinRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "node2", req.NodeID)
		assert.Equal(t, "10.0.0.2:7000", req.Address)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(joinResponse{Status: "ok"})
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	err := tryJoinPeer(context.Background(), addr, "node2", "10.0.0.2:7000", server.Client())
	assert.NoError(t, err)
}

func TestTryJoinPeer_NotLeaderRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTemporaryRedirect)
		json.NewEncoder(w).Encode(joinResponse{
			Error:      "not leader",
			LeaderAddr: "10.0.0.1:7000",
		})
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	err := tryJoinPeer(context.Background(), addr, "node2", "10.0.0.2:7000", server.Client())
	assert.ErrorContains(t, err, "not the leader")
}

func TestTryJoinPeer_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(joinResponse{Error: "raft apply failed"})
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	err := tryJoinPeer(context.Background(), addr, "node2", "10.0.0.2:7000", server.Client())
	assert.ErrorContains(t, err, "raft apply failed")
}

func TestTryJoinPeer_ConnectionRefused(t *testing.T) {
	err := tryJoinPeer(context.Background(), "127.0.0.1:1", "node2", "10.0.0.2:7000", &http.Client{Timeout: time.Second})
	assert.Error(t, err)
}

// --- tryJoinPeers ---

func TestTryJoinPeers_FirstPeerSucceeds(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(joinResponse{Status: "ok"})
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	peers := []Peer{{Address: addr}, {Address: addr}}
	err := tryJoinPeers(context.Background(), peers, "node2", "10.0.0.2:7000", server.Client())
	assert.NoError(t, err)
	// Should stop after first success
	assert.Equal(t, 1, callCount)
}

func TestTryJoinPeers_FallsBackToSecondPeer(t *testing.T) {
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(joinResponse{Error: "not ready"})
	}))
	defer failServer.Close()

	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(joinResponse{Status: "ok"})
	}))
	defer successServer.Close()

	// Use success server's client (works for both since both are plain HTTP)
	client := &http.Client{}
	peers := []Peer{
		{Address: failServer.Listener.Addr().String()},
		{Address: successServer.Listener.Addr().String()},
	}

	// Run multiple times since peers are shuffled; at least one attempt should succeed.
	var lastErr error
	for i := 0; i < 10; i++ {
		lastErr = tryJoinPeers(context.Background(), peers, "node2", "10.0.0.2:7000", client)
		if lastErr == nil {
			break
		}
	}
	assert.NoError(t, lastErr)
}

func TestTryJoinPeers_NoPeers(t *testing.T) {
	err := tryJoinPeers(context.Background(), nil, "node2", "10.0.0.2:7000", &http.Client{})
	assert.ErrorContains(t, err, "no peers")
}

// --- hasExistingRaftState ---

func TestHasExistingRaftState_NoFiles(t *testing.T) {
	assert.False(t, hasExistingRaftState(t.TempDir()))
}

func TestHasExistingRaftState_WithLogDB(t *testing.T) {
	dir := t.TempDir()
	f, err := createFile(dir + "/raft-log.db")
	require.NoError(t, err)
	f.Close()
	assert.True(t, hasExistingRaftState(dir))
}

func TestHasExistingRaftState_WithStableDB(t *testing.T) {
	dir := t.TempDir()
	f, err := createFile(dir + "/raft-stable.db")
	require.NoError(t, err)
	f.Close()
	assert.True(t, hasExistingRaftState(dir))
}

func createFile(path string) (*os.File, error) {
	return os.Create(path)
}
