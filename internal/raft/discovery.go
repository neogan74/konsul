package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DiscoveryMethod defines how peers are discovered.
type DiscoveryMethod string

const (
	// DiscoveryMethodNone disables automatic discovery.
	DiscoveryMethodNone DiscoveryMethod = "none"

	// DiscoveryMethodStatic uses a fixed list of peer HTTP addresses.
	DiscoveryMethodStatic DiscoveryMethod = "static"

	// DiscoveryMethodDNS resolves peers via DNS SRV records.
	DiscoveryMethodDNS DiscoveryMethod = "dns"
)

// DiscoveryConfig configures automatic cluster discovery.
type DiscoveryConfig struct {
	// Method selects the discovery strategy: "none", "static", or "dns".
	Method DiscoveryMethod

	// Seeds is the list of peer HTTP addresses for static discovery.
	// Format: "host:port" (HTTP API port, not Raft port).
	Seeds []string

	// DNSDomain is the DNS domain to query SRV records for DNS discovery.
	// SRV records are looked up as: _konsul._tcp.<DNSDomain>
	DNSDomain string

	// DNSPort is the fallback HTTP port when the SRV record does not include one.
	// Default: 8888.
	DNSPort int

	// RetryInterval is the initial wait between discovery attempts.
	// Doubles on each failure up to RetryMaxInterval.
	// Default: 5s.
	RetryInterval time.Duration

	// RetryMaxInterval caps the exponential backoff.
	// Default: 60s.
	RetryMaxInterval time.Duration

	// RetryMax is the maximum number of join attempts before giving up.
	// 0 means retry indefinitely until the context is cancelled.
	RetryMax int
}

// DefaultDiscoveryConfig returns a DiscoveryConfig with sensible defaults.
func DefaultDiscoveryConfig() *DiscoveryConfig {
	return &DiscoveryConfig{
		Method:           DiscoveryMethodNone,
		DNSPort:          8888,
		RetryInterval:    5 * time.Second,
		RetryMaxInterval: 60 * time.Second,
	}
}

// Peer represents a discovered peer's HTTP API address.
type Peer struct {
	Address string // host:port of the HTTP API
}

// Discoverer discovers HTTP addresses of cluster peers.
type Discoverer interface {
	Discover(ctx context.Context) ([]Peer, error)
}

// NewDiscoverer creates the appropriate Discoverer from the config.
// Returns nil when discovery is disabled.
func NewDiscoverer(cfg *DiscoveryConfig) (Discoverer, error) {
	switch cfg.Method {
	case DiscoveryMethodNone, "":
		return nil, nil
	case DiscoveryMethodStatic:
		if len(cfg.Seeds) == 0 {
			return nil, fmt.Errorf("static discovery requires at least one seed address")
		}
		return &staticDiscoverer{seeds: cfg.Seeds}, nil
	case DiscoveryMethodDNS:
		if cfg.DNSDomain == "" {
			return nil, fmt.Errorf("dns discovery requires a domain name")
		}
		port := cfg.DNSPort
		if port == 0 {
			port = 8888
		}
		return &dnsDiscoverer{domain: cfg.DNSDomain, defaultPort: port}, nil
	default:
		return nil, fmt.Errorf("unknown discovery method: %q", cfg.Method)
	}
}

// staticDiscoverer returns a fixed list of peer addresses.
type staticDiscoverer struct {
	seeds []string
}

// Discover returns the configured seed addresses.
func (d *staticDiscoverer) Discover(_ context.Context) ([]Peer, error) {
	peers := make([]Peer, 0, len(d.seeds))
	for _, s := range d.seeds {
		if s != "" {
			peers = append(peers, Peer{Address: s})
		}
	}
	return peers, nil
}

// dnsDiscoverer resolves peers via DNS SRV records.
// It looks up _konsul._tcp.<domain> and returns the HTTP addresses.
type dnsDiscoverer struct {
	domain      string
	defaultPort int
}

// Discover performs a DNS SRV lookup and returns matching peers.
func (d *dnsDiscoverer) Discover(ctx context.Context) ([]Peer, error) {
	_, addrs, err := net.DefaultResolver.LookupSRV(ctx, "konsul", "tcp", d.domain)
	if err != nil {
		return nil, fmt.Errorf("dns SRV lookup for _konsul._tcp.%s: %w", d.domain, err)
	}

	peers := make([]Peer, 0, len(addrs))
	for _, addr := range addrs {
		host := addr.Target
		// Strip trailing dot from DNS names
		if len(host) > 0 && host[len(host)-1] == '.' {
			host = host[:len(host)-1]
		}
		port := int(addr.Port)
		if port == 0 {
			port = d.defaultPort
		}
		peers = append(peers, Peer{Address: fmt.Sprintf("%s:%d", host, port)})
	}
	return peers, nil
}

// joinRequest is the body sent to POST /cluster/join.
type joinRequest struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

// joinResponse is parsed from the join endpoint.
type joinResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	Error      string `json:"error"`
	LeaderAddr string `json:"leader_addr"`
}

// tryJoinPeer attempts to join the cluster by calling POST /cluster/join on one peer.
// Returns nil on success, or an error describing why it failed.
func tryJoinPeer(ctx context.Context, peerHTTPAddr, selfNodeID, selfRaftAddr string, client *http.Client) error {
	body, err := json.Marshal(joinRequest{NodeID: selfNodeID, Address: selfRaftAddr})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/cluster/join", peerHTTPAddr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result joinResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusTemporaryRedirect:
		return fmt.Errorf("peer %s is not the leader (leader: %s)", peerHTTPAddr, result.LeaderAddr)
	default:
		msg := result.Error
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("peer %s returned error: %s", peerHTTPAddr, msg)
	}
}

// tryJoinPeers tries to join via each discovered peer, returning nil on the first success.
func tryJoinPeers(ctx context.Context, peers []Peer, selfNodeID, selfRaftAddr string, client *http.Client) error {
	// Shuffle so all nodes don't hammer the same peer simultaneously.
	shuffled := make([]Peer, len(peers))
	copy(shuffled, peers)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	var lastErr error
	for _, peer := range shuffled {
		if err := tryJoinPeer(ctx, peer.Address, selfNodeID, selfRaftAddr, client); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no peers discovered")
}

// hasExistingRaftState reports whether the node has existing Raft state on disk,
// meaning it was previously part of a cluster and will rejoin automatically.
func hasExistingRaftState(dataDir string) bool {
	logDB := filepath.Join(dataDir, "raft-log.db")
	if _, err := os.Stat(logDB); err == nil {
		return true
	}
	stableDB := filepath.Join(dataDir, "raft-stable.db")
	_, err := os.Stat(stableDB)
	return err == nil
}

// RunAutoJoin discovers peers and attempts to join the cluster.
// It is a no-op if:
//   - discovery is disabled (method == none)
//   - the node is configured to bootstrap (it creates the cluster)
//   - the node has existing Raft state (it will rejoin automatically)
//
// Otherwise it retries with exponential backoff until ctx is cancelled,
// a join succeeds, or RetryMax attempts are exhausted.
// This method blocks; callers typically run it in a goroutine.
func (n *Node) RunAutoJoin(ctx context.Context, discoverer Discoverer, cfg *DiscoveryConfig) {
	if discoverer == nil || cfg == nil || cfg.Method == DiscoveryMethodNone {
		return
	}

	// Bootstrap nodes create the cluster; they never join an existing one.
	if n.config.Bootstrap {
		n.logger.Info("auto-join skipped: this node is bootstrapping the cluster")
		return
	}

	// Nodes with existing Raft state reconnect automatically via stored peer list.
	if hasExistingRaftState(n.config.DataDir) {
		n.logger.Info("auto-join skipped: existing Raft state found, node will reconnect automatically")
		return
	}

	selfNodeID := n.config.NodeID
	selfRaftAddr := n.config.GetAdvertiseAddr()

	httpClient := &http.Client{Timeout: 10 * time.Second}

	retryInterval := cfg.RetryInterval
	if retryInterval <= 0 {
		retryInterval = 5 * time.Second
	}
	maxInterval := cfg.RetryMaxInterval
	if maxInterval <= 0 {
		maxInterval = 60 * time.Second
	}

	n.logger.Info("starting auto-join",
		"method", string(cfg.Method),
		"self_node_id", selfNodeID,
		"self_raft_addr", selfRaftAddr,
	)

	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			n.logger.Info("auto-join cancelled", "reason", err)
			return
		}

		peers, err := discoverer.Discover(ctx)
		if err != nil {
			n.logger.Warn("discovery failed", "error", err, "attempt", attempt+1)
		} else if len(peers) == 0 {
			n.logger.Warn("no peers discovered", "attempt", attempt+1)
		} else {
			if joinErr := tryJoinPeers(ctx, peers, selfNodeID, selfRaftAddr, httpClient); joinErr == nil {
				n.logger.Info("auto-join successful", "attempts", attempt+1)
				return
			} else {
				n.logger.Warn("auto-join attempt failed", "error", joinErr, "attempt", attempt+1)
			}
		}

		attempt++
		if cfg.RetryMax > 0 && attempt >= cfg.RetryMax {
			n.logger.Error("auto-join failed after max attempts", "max", cfg.RetryMax)
			return
		}

		select {
		case <-ctx.Done():
			n.logger.Info("auto-join cancelled", "reason", ctx.Err())
			return
		case <-time.After(retryInterval):
		}

		retryInterval *= 2
		if retryInterval > maxInterval {
			retryInterval = maxInterval
		}
	}
}
