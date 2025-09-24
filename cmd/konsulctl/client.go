package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type KonsulClient struct {
	BaseURL    string
	HTTPClient *http.Client
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

func NewKonsulClient(baseURL string) *KonsulClient {
	return &KonsulClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
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