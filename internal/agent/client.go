package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/neogan74/konsul/internal/store"
)

// ServerClient handles communication with the Konsul server
type ServerClient struct {
	serverURL  string
	httpClient *http.Client
	agentID    string
}

// NewServerClient creates a new server client
func NewServerClient(serverURL string, tlsConfig TLSConfig, agentID string) (*ServerClient, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Configure TLS if enabled
	if tlsConfig.Enabled {
		transport := client.Transport.(*http.Transport)
		tlsCfg, err := buildTLSConfig(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}
		transport.TLSClientConfig = tlsCfg
	}

	return &ServerClient{
		serverURL:  serverURL,
		httpClient: client,
		agentID:    agentID,
	}, nil
}

// buildTLSConfig builds TLS configuration from config
func buildTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.SkipVerify,
	}

	// Load CA cert if provided
	if cfg.CACert != "" {
		caCert, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client cert if provided (mTLS)
	if cfg.ClientCert != "" && cfg.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// RegisterAgent registers the agent with the server
func (c *ServerClient) RegisterAgent(ctx context.Context, info AgentInfo) error {
	url := fmt.Sprintf("%s/v1/agent/register", c.serverURL)
	return c.doPost(ctx, url, info, nil)
}

// Sync performs a sync request to get updates from the server
func (c *ServerClient) Sync(ctx context.Context, req SyncRequest) (*SyncResponse, error) {
	url := fmt.Sprintf("%s/v1/agent/sync", c.serverURL)

	var resp SyncResponse
	if err := c.doPost(ctx, url, req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// BatchUpdate sends batched service updates to the server
func (c *ServerClient) BatchUpdate(ctx context.Context, updates []ServiceUpdate) error {
	url := fmt.Sprintf("%s/v1/agent/batch-update", c.serverURL)

	req := map[string]interface{}{
		"agent_id": c.agentID,
		"updates":  updates,
	}

	return c.doPost(ctx, url, req, nil)
}

// RegisterService registers a service with the server
func (c *ServerClient) RegisterService(ctx context.Context, svc store.Service) error {
	url := fmt.Sprintf("%s/register", c.serverURL)
	return c.doPost(ctx, url, svc, nil)
}

// DeregisterService deregisters a service from the server
func (c *ServerClient) DeregisterService(ctx context.Context, name string) error {
	url := fmt.Sprintf("%s/deregister/%s", c.serverURL, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deregister failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// GetService retrieves service entries from the server
func (c *ServerClient) GetService(ctx context.Context, name string) ([]*store.ServiceEntry, error) {
	url := fmt.Sprintf("%s/services/%s", c.serverURL, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get service failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var entries []*store.ServiceEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return entries, nil
}

// GetKV retrieves a KV entry from the server
func (c *ServerClient) GetKV(ctx context.Context, key string) (*store.KVEntry, error) {
	url := fmt.Sprintf("%s/kv/%s", c.serverURL, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get KV failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var entry store.KVEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &entry, nil
}

// SetKV sets a KV entry on the server
func (c *ServerClient) SetKV(ctx context.Context, key string, entry *store.KVEntry) error {
	url := fmt.Sprintf("%s/kv/%s", c.serverURL, key)
	return c.doPut(ctx, url, entry, nil)
}

// DeleteKV deletes a KV entry from the server
func (c *ServerClient) DeleteKV(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/kv/%s", c.serverURL, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete KV failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// ReportHealthCheck reports a health check status change to the server
func (c *ServerClient) ReportHealthCheck(ctx context.Context, update HealthUpdate) error {
	url := fmt.Sprintf("%s/v1/agent/health-update", c.serverURL)
	return c.doPost(ctx, url, update, nil)
}

// Helper methods

func (c *ServerClient) doPost(ctx context.Context, url string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPost, url, body, result)
}

func (c *ServerClient) doPut(ctx context.Context, url string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, http.MethodPut, url, body, result)
}

func (c *ServerClient) doRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Agent-ID", c.agentID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed: %s (status: %d)", string(bodyBytes), resp.StatusCode)
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Close closes the client and releases resources
func (c *ServerClient) Close() {
	c.httpClient.CloseIdleConnections()
}
