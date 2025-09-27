package healthcheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
				DisableKeepAlives:   true,
				MaxIdleConnsPerHost: 1,
			},
		},
	}
}

func (h *HTTPChecker) Check(ctx context.Context, check *Check) (Status, string, error) {
	if check.HTTP == "" {
		return StatusCritical, "HTTP URL not specified", fmt.Errorf("HTTP URL required")
	}

	// Create request with context for timeout
	method := check.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, check.HTTP, nil)
	if err != nil {
		return StatusCritical, fmt.Sprintf("Failed to create request: %v", err), err
	}

	// Add custom headers
	for key, value := range check.Headers {
		req.Header.Set(key, value)
	}

	// Set User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Konsul-Health-Check/1.0")
	}

	// Configure TLS
	if check.TLSSkipVerify {
		transport := h.client.Transport.(*http.Transport)
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	// Set timeout from check
	if check.Timeout > 0 {
		h.client.Timeout = check.Timeout
	}

	start := time.Now()
	resp, err := h.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return StatusCritical, fmt.Sprintf("Request failed after %v: %v", duration, err), err
	}
	defer resp.Body.Close()

	// Check status code
	output := fmt.Sprintf("HTTP %d %s (%.3fs)", resp.StatusCode, resp.Status, duration.Seconds())

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return StatusPassing, output, nil
	} else if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return StatusWarning, output, nil
	} else {
		return StatusCritical, output, fmt.Errorf("HTTP check failed with status %d", resp.StatusCode)
	}
}