package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type KonsulClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// TLSConfig holds TLS configuration for the client
type TLSConfig struct {
	Enabled        bool
	SkipVerify     bool
	CACertFile     string
	ClientCertFile string
	ClientKeyFile  string
}

type KVRequest struct {
	Value string `json:"value"`
}

type KVResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type Service struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type CheckDefinition struct {
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name"`
	ServiceID     string            `json:"service_id,omitempty"`
	HTTP          string            `json:"http,omitempty"`
	TCP           string            `json:"tcp,omitempty"`
	GRPC          string            `json:"grpc,omitempty"`
	TTL           string            `json:"ttl,omitempty"`
	Interval      string            `json:"interval,omitempty"`
	Timeout       string            `json:"timeout,omitempty"`
	Method        string            `json:"method,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	TLSSkipVerify bool              `json:"tls_skip_verify,omitempty"`
	GRPCUseTLS    bool              `json:"grpc_use_tls,omitempty"`
}

type ServiceRegisterRequest struct {
	Name    string             `json:"name"`
	Address string             `json:"address"`
	Port    int                `json:"port"`
	Checks  []*CheckDefinition `json:"checks,omitempty"`
}

type BackupResponse struct {
	Message    string `json:"message"`
	BackupFile string `json:"backup_file"`
}

type RestoreRequest struct {
	BackupPath string `json:"backup_path"`
}

// Rate limit types
type RateLimitStats struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
}

type RateLimitConfig struct {
	Success bool `json:"success"`
	Config  struct {
		Enabled         bool    `json:"enabled"`
		RequestsPerSec  float64 `json:"requests_per_sec"`
		Burst           int     `json:"burst"`
		ByIP            bool    `json:"by_ip"`
		ByAPIKey        bool    `json:"by_apikey"`
		CleanupInterval string  `json:"cleanup_interval"`
	} `json:"config"`
}

type RateLimitClient struct {
	Identifier string  `json:"identifier"`
	Type       string  `json:"type"`
	Tokens     float64 `json:"tokens"`
	MaxTokens  int     `json:"max_tokens"`
	Rate       float64 `json:"rate"`
	LastUpdate string  `json:"last_update"`
}

type RateLimitClientsResponse struct {
	Success bool               `json:"success"`
	Count   int                `json:"count"`
	Clients []*RateLimitClient `json:"clients"`
}

type RateLimitClientStatus struct {
	Success bool             `json:"success"`
	Client  *RateLimitClient `json:"client"`
}

type RateLimitResetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	IP      string `json:"ip,omitempty"`
	KeyID   string `json:"key_id,omitempty"`
	Type    string `json:"type,omitempty"`
}

type RateLimitConfigUpdate struct {
	RequestsPerSec *float64 `json:"requests_per_sec,omitempty"`
	Burst          *int     `json:"burst,omitempty"`
}

type RateLimitConfigUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Config  struct {
		RequestsPerSec float64 `json:"requests_per_sec"`
		Burst          int     `json:"burst"`
	} `json:"config,omitempty"`
}

type RateLimitAdjustResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RateLimitGenericResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type RateLimitWhitelistEntry struct {
	Identifier string  `json:"identifier"`
	Type       string  `json:"type"`
	Reason     string  `json:"reason"`
	AddedBy    string  `json:"added_by"`
	AddedAt    string  `json:"added_at"`
	ExpiresAt  *string `json:"expires_at"`
}

type RateLimitWhitelistResponse struct {
	Success bool                       `json:"success"`
	Count   int                        `json:"count"`
	Entries []*RateLimitWhitelistEntry `json:"entries"`
}

type RateLimitBlacklistEntry struct {
	Identifier string `json:"identifier"`
	Type       string `json:"type"`
	Reason     string `json:"reason"`
	AddedBy    string `json:"added_by"`
	AddedAt    string `json:"added_at"`
	ExpiresAt  string `json:"expires_at"`
}

type RateLimitBlacklistResponse struct {
	Success bool                       `json:"success"`
	Count   int                        `json:"count"`
	Entries []*RateLimitBlacklistEntry `json:"entries"`
}

func NewKonsulClient(baseURL string) *KonsulClient {
	return NewKonsulClientWithTLS(baseURL, nil)
}

func NewKonsulClientWithTLS(baseURL string, tlsConfig *TLSConfig) *KonsulClient {
	transport := &http.Transport{}

	// Configure TLS if provided
	if tlsConfig != nil && (tlsConfig.Enabled || strings.HasPrefix(baseURL, "https://")) {
		tlsClientConfig := &tls.Config{
			InsecureSkipVerify: tlsConfig.SkipVerify,
		}

		// Load CA certificate if provided
		if tlsConfig.CACertFile != "" {
			caCert, err := os.ReadFile(tlsConfig.CACertFile)
			if err == nil {
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				tlsClientConfig.RootCAs = caCertPool
			}
		}

		// Load client certificate if provided
		if tlsConfig.ClientCertFile != "" && tlsConfig.ClientKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(tlsConfig.ClientCertFile, tlsConfig.ClientKeyFile)
			if err == nil {
				tlsClientConfig.Certificates = []tls.Certificate{cert}
			}
		}

		transport.TLSClientConfig = tlsClientConfig
	}

	return &KonsulClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

func (c *KonsulClient) GetKV(key string) (string, error) {
	url := fmt.Sprintf("%s/kv/%s", c.BaseURL, key)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("key not found")
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return "", fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var kvResp KVResponse
	if err := json.Unmarshal(body, &kvResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return kvResp.Value, nil
}

func (c *KonsulClient) SetKV(key, value string) error {
	url := fmt.Sprintf("%s/kv/%s", c.BaseURL, key)

	reqBody := KVRequest{Value: value}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) DeleteKV(key string) error {
	url := fmt.Sprintf("%s/kv/%s", c.BaseURL, key)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("key not found")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) ListKV() ([]string, error) {
	url := fmt.Sprintf("%s/kv/", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var keys []string
	if err := json.Unmarshal(body, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return keys, nil
}

func (c *KonsulClient) RegisterService(name, address, port string) error {
	url := fmt.Sprintf("%s/register", c.BaseURL)

	// Convert port to int
	portInt := 0
	if _, err := fmt.Sscanf(port, "%d", &portInt); err != nil {
		return fmt.Errorf("invalid port: %s", port)
	}

	reqBody := ServiceRegisterRequest{
		Name:    name,
		Address: address,
		Port:    portInt,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) RegisterServiceWithChecks(name, address, port string, checks []*CheckDefinition) error {
	url := fmt.Sprintf("%s/register", c.BaseURL)

	// Convert port to int
	portInt := 0
	if _, err := fmt.Sscanf(port, "%d", &portInt); err != nil {
		return fmt.Errorf("invalid port: %s", port)
	}

	reqBody := ServiceRegisterRequest{
		Name:    name,
		Address: address,
		Port:    portInt,
		Checks:  checks,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) ListServices() ([]Service, error) {
	url := fmt.Sprintf("%s/services/", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var services []Service
	if err := json.Unmarshal(body, &services); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return services, nil
}

func (c *KonsulClient) DeregisterService(name string) error {
	url := fmt.Sprintf("%s/deregister/%s", c.BaseURL, name)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("service not found")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) ServiceHeartbeat(name string) error {
	url := fmt.Sprintf("%s/heartbeat/%s", c.BaseURL, name)

	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("service not found")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) CreateBackup() (string, error) {
	url := fmt.Sprintf("%s/backup", c.BaseURL)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return "", fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var backupResp BackupResponse
	if err := json.Unmarshal(body, &backupResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return backupResp.BackupFile, nil
}

func (c *KonsulClient) RestoreBackup(backupPath string) error {
	url := fmt.Sprintf("%s/restore", c.BaseURL)

	reqBody := RestoreRequest{BackupPath: backupPath}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) ListBackups() ([]string, error) {
	url := fmt.Sprintf("%s/backups", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var backups []string
	if err := json.Unmarshal(body, &backups); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return backups, nil
}

func (c *KonsulClient) ExportData() (string, error) {
	url := fmt.Sprintf("%s/export", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return "", fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return string(body), nil
}

// Rate limit admin methods

func (c *KonsulClient) GetRateLimitStats() (*RateLimitStats, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/stats", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var stats RateLimitStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &stats, nil
}

func (c *KonsulClient) GetRateLimitConfig() (*RateLimitConfig, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/config", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var config RateLimitConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &config, nil
}

func (c *KonsulClient) GetRateLimitClients(limiterType string) (*RateLimitClientsResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/clients?type=%s", c.BaseURL, limiterType)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var clients RateLimitClientsResponse
	if err := json.Unmarshal(body, &clients); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &clients, nil
}

func (c *KonsulClient) GetRateLimitClientStatus(identifier string) (*RateLimitClient, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/client/%s", c.BaseURL, identifier)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("client not found")
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status RateLimitClientStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return status.Client, nil
}

func (c *KonsulClient) ResetRateLimitIP(ip string) error {
	url := fmt.Sprintf("%s/admin/ratelimit/reset/ip/%s", c.BaseURL, ip)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) ResetRateLimitAPIKey(keyID string) error {
	url := fmt.Sprintf("%s/admin/ratelimit/reset/apikey/%s", c.BaseURL, keyID)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) ResetRateLimitAll(limiterType string) error {
	url := fmt.Sprintf("%s/admin/ratelimit/reset/all?type=%s", c.BaseURL, limiterType)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *KonsulClient) UpdateRateLimitConfig(requestsPerSec *float64, burst *int) (*RateLimitConfigUpdateResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/config", c.BaseURL)

	update := RateLimitConfigUpdate{
		RequestsPerSec: requestsPerSec,
		Burst:          burst,
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("server error: %s - %s", errResp.Error, errResp.Message)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var updateResp RateLimitConfigUpdateResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &updateResp, nil
}

// AdjustClientLimit temporarily adjusts rate limit for a specific client
func (c *KonsulClient) AdjustClientLimit(clientType, identifier string, rate float64, burst int, duration string) (*RateLimitAdjustResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/client/%s/%s", c.BaseURL, clientType, identifier)

	adjust := map[string]interface{}{
		"rate":     rate,
		"burst":    burst,
		"duration": duration,
	}

	body, err := json.Marshal(adjust)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to adjust client limit: %s", string(respBody))
	}

	var adjustResp RateLimitAdjustResponse
	if err := json.NewDecoder(resp.Body).Decode(&adjustResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &adjustResp, nil
}

// GetWhitelist returns all whitelisted entries
func (c *KonsulClient) GetWhitelist() (*RateLimitWhitelistResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/whitelist", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get whitelist: %s", string(body))
	}

	var whitelist RateLimitWhitelistResponse
	if err := json.NewDecoder(resp.Body).Decode(&whitelist); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &whitelist, nil
}

// AddToWhitelist adds an identifier to the whitelist
func (c *KonsulClient) AddToWhitelist(identifier, clientType, reason, duration string) (*RateLimitGenericResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/whitelist", c.BaseURL)

	req := map[string]interface{}{
		"identifier": identifier,
		"type":       clientType,
		"reason":     reason,
	}
	if duration != "" {
		req["duration"] = duration
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add to whitelist: %s", string(respBody))
	}

	var result RateLimitGenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// RemoveFromWhitelist removes an identifier from the whitelist
func (c *KonsulClient) RemoveFromWhitelist(identifier string) (*RateLimitGenericResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/whitelist/%s", c.BaseURL, identifier)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to remove from whitelist: %s", string(body))
	}

	var result RateLimitGenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetBlacklist returns all blacklisted entries
func (c *KonsulClient) GetBlacklist() (*RateLimitBlacklistResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/blacklist", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get blacklist: %s", string(body))
	}

	var blacklist RateLimitBlacklistResponse
	if err := json.NewDecoder(resp.Body).Decode(&blacklist); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &blacklist, nil
}

// AddToBlacklist adds an identifier to the blacklist
func (c *KonsulClient) AddToBlacklist(identifier, clientType, reason, duration string) (*RateLimitGenericResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/blacklist", c.BaseURL)

	req := map[string]interface{}{
		"identifier": identifier,
		"type":       clientType,
		"reason":     reason,
		"duration":   duration,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add to blacklist: %s", string(respBody))
	}

	var result RateLimitGenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// RemoveFromBlacklist removes an identifier from the blacklist
func (c *KonsulClient) RemoveFromBlacklist(identifier string) (*RateLimitGenericResponse, error) {
	url := fmt.Sprintf("%s/admin/ratelimit/blacklist/%s", c.BaseURL, identifier)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to remove from blacklist: %s", string(body))
	}

	var result RateLimitGenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ACL Policy Methods

// ListACLPolicies lists all ACL policies
func (c *KonsulClient) ListACLPolicies() (*ACLPoliciesResponse, error) {
	url := fmt.Sprintf("%s/acl/policies", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result ACLPoliciesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetACLPolicy retrieves a specific ACL policy
func (c *KonsulClient) GetACLPolicy(name string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/acl/policies/%s", c.BaseURL, name)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// CreateACLPolicy creates a new ACL policy
func (c *KonsulClient) CreateACLPolicy(policy map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/acl/policies", c.BaseURL)

	jsonData, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateACLPolicy updates an existing ACL policy
func (c *KonsulClient) UpdateACLPolicy(name string, policy map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/acl/policies/%s", c.BaseURL, name)

	jsonData, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// DeleteACLPolicy deletes an ACL policy
func (c *KonsulClient) DeleteACLPolicy(name string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/acl/policies/%s", c.BaseURL, name)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// TestACLPolicy tests ACL permissions
func (c *KonsulClient) TestACLPolicy(policies []string, resource, path, capability string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/acl/test", c.BaseURL)

	request := map[string]interface{}{
		"policies":   policies,
		"resource":   resource,
		"path":       path,
		"capability": capability,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// ACL Response types

// ACLPoliciesResponse represents the response from listing policies
type ACLPoliciesResponse struct {
	Policies []string `json:"policies"`
	Count    int      `json:"count"`
}
