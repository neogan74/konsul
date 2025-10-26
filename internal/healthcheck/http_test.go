package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPChecker_Check_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    server.URL,
		Method:  "GET",
		Timeout: 5 * time.Second,
	}

	status, output, err := checker.Check(context.Background(), check)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != StatusPassing {
		t.Errorf("expected status Passing, got %s", status)
	}
	if !contains(output, "HTTP 200") {
		t.Errorf("expected output to contain 'HTTP 200', got: %s", output)
	}
}

func TestHTTPChecker_Check_NoURL(t *testing.T) {
	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    "",
		Timeout: 5 * time.Second,
	}

	status, output, err := checker.Check(context.Background(), check)

	if err == nil {
		t.Error("expected error when HTTP URL not specified")
	}
	if status != StatusCritical {
		t.Errorf("expected status Critical, got %s", status)
	}
	if output != "HTTP URL not specified" {
		t.Errorf("expected specific output, got: %s", output)
	}
}

func TestHTTPChecker_Check_CustomMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    server.URL,
		Method:  "POST",
		Timeout: 5 * time.Second,
	}

	status, _, err := checker.Check(context.Background(), check)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != StatusPassing {
		t.Errorf("expected status Passing, got %s", status)
	}
}

func TestHTTPChecker_Check_DefaultMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected method GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    server.URL,
		Method:  "", // Empty should default to GET
		Timeout: 5 * time.Second,
	}

	status, _, err := checker.Check(context.Background(), check)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != StatusPassing {
		t.Errorf("expected status Passing, got %s", status)
	}
}

func TestHTTPChecker_Check_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Authorization header")
		}
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Error("expected X-Custom header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker()
	check := &Check{
		HTTP: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
			"X-Custom":      "custom-value",
		},
		Timeout: 5 * time.Second,
	}

	status, _, err := checker.Check(context.Background(), check)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != StatusPassing {
		t.Errorf("expected status Passing, got %s", status)
	}
}

func TestHTTPChecker_Check_UserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "Konsul-Health-Check/1.0" {
			t.Errorf("expected User-Agent 'Konsul-Health-Check/1.0', got '%s'", ua)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    server.URL,
		Timeout: 5 * time.Second,
	}

	status, _, err := checker.Check(context.Background(), check)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status != StatusPassing {
		t.Errorf("expected status Passing, got %s", status)
	}
}

func TestHTTPChecker_Check_StatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus Status
		shouldError    bool
	}{
		{"200 OK", 200, StatusPassing, false},
		{"201 Created", 201, StatusPassing, false},
		{"204 No Content", 204, StatusPassing, false},
		{"300 Multiple Choices", 300, StatusWarning, false},
		{"301 Moved Permanently", 301, StatusWarning, false},
		{"302 Found", 302, StatusWarning, false},
		{"400 Bad Request", 400, StatusCritical, true},
		{"404 Not Found", 404, StatusCritical, true},
		{"500 Internal Server Error", 500, StatusCritical, true},
		{"503 Service Unavailable", 503, StatusCritical, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			checker := NewHTTPChecker()
			check := &Check{
				HTTP:    server.URL,
				Timeout: 5 * time.Second,
			}

			status, output, err := checker.Check(context.Background(), check)

			if tt.shouldError && err == nil {
				t.Error("expected error for error status code")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, status)
			}
			if !contains(output, http.StatusText(tt.statusCode)) {
				t.Errorf("expected output to contain status text, got: %s", output)
			}
		})
	}
}

func TestHTTPChecker_Check_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    server.URL,
		Timeout: 50 * time.Millisecond, // Short timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	status, output, err := checker.Check(ctx, check)

	if err == nil {
		t.Error("expected timeout error")
	}
	if status != StatusCritical {
		t.Errorf("expected status Critical on timeout, got %s", status)
	}
	if !contains(output, "failed") {
		t.Errorf("expected output to indicate failure, got: %s", output)
	}
}

func TestHTTPChecker_Check_InvalidURL(t *testing.T) {
	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    "not a valid url :// invalid",
		Timeout: 5 * time.Second,
	}

	status, _, err := checker.Check(context.Background(), check)

	if err == nil {
		t.Error("expected error for invalid URL")
	}
	if status != StatusCritical {
		t.Errorf("expected status Critical, got %s", status)
	}
}

func TestHTTPChecker_Check_ConnectionRefused(t *testing.T) {
	checker := NewHTTPChecker()
	check := &Check{
		HTTP:    "http://localhost:1", // Port 1 should be refused
		Timeout: 1 * time.Second,
	}

	status, output, err := checker.Check(context.Background(), check)

	if err == nil {
		t.Error("expected connection error")
	}
	if status != StatusCritical {
		t.Errorf("expected status Critical, got %s", status)
	}
	if !contains(output, "failed") {
		t.Errorf("expected output to indicate failure, got: %s", output)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
