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
