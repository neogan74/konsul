package healthcheck

import (
	"testing"
	"time"

	"github.com/neogan74/konsul/internal/logger"
)

func TestManager_AddCheck(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		Name:      "test-http",
		ServiceID: "web-service",
		HTTP:      "http://localhost:8080/health",
		Interval:  "10s",
		Timeout:   "5s",
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.Name != "test-http" {
		t.Errorf("expected name 'test-http', got '%s'", check.Name)
	}
	if check.Type != CheckTypeHTTP {
		t.Errorf("expected type HTTP, got %s", check.Type)
	}
	if check.Status != StatusCritical {
		t.Errorf("expected initial status Critical, got %s", check.Status)
	}
	if check.Interval != 10*time.Second {
		t.Errorf("expected interval 10s, got %v", check.Interval)
	}
	if check.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", check.Timeout)
	}
}

func TestManager_AddCheck_WithID(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		ID:       "custom-id",
		Name:     "test-check",
		HTTP:     "http://localhost:8080",
		Interval: "30s",
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.ID != "custom-id" {
		t.Errorf("expected ID 'custom-id', got '%s'", check.ID)
	}
}

func TestManager_AddCheck_GeneratesID(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		Name: "test-check",
		HTTP: "http://localhost:8080",
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.ID == "" {
		t.Error("expected ID to be generated")
	}
}

func TestManager_AddCheck_DefaultValues(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		Name: "test-check",
		HTTP: "http://localhost:8080",
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.Interval != 30*time.Second {
		t.Errorf("expected default interval 30s, got %v", check.Interval)
	}
	if check.Timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", check.Timeout)
	}
}

func TestManager_GetCheck(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		ID:   "test-id",
		Name: "test-check",
		HTTP: "http://localhost:8080",
	}

	added, _ := manager.AddCheck(def)

	// Get the check
	retrieved, exists := manager.GetCheck("test-id")
	if !exists {
		t.Fatal("expected check to exist")
	}

	if retrieved.ID != added.ID {
		t.Errorf("expected ID %s, got %s", added.ID, retrieved.ID)
	}
	if retrieved.Name != added.Name {
		t.Errorf("expected name %s, got %s", added.Name, retrieved.Name)
	}
}

func TestManager_GetCheck_NotFound(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	_, exists := manager.GetCheck("nonexistent")
	if exists {
		t.Error("expected check not to exist")
	}
}

func TestManager_ListChecks(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	// Add multiple checks
	def1 := &CheckDefinition{Name: "check1", HTTP: "http://localhost:8081"}
	def2 := &CheckDefinition{Name: "check2", TCP: "localhost:8082"}
	def3 := &CheckDefinition{Name: "check3", TTL: "60s"}

	manager.AddCheck(def1)
	manager.AddCheck(def2)
	manager.AddCheck(def3)

	checks := manager.ListChecks()
	if len(checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(checks))
	}
}

func TestManager_ListChecks_Empty(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	checks := manager.ListChecks()
	if len(checks) != 0 {
		t.Errorf("expected 0 checks, got %d", len(checks))
	}
}

func TestManager_RemoveCheck(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		ID:   "test-id",
		Name: "test-check",
		HTTP: "http://localhost:8080",
	}

	manager.AddCheck(def)

	// Verify it exists
	_, exists := manager.GetCheck("test-id")
	if !exists {
		t.Fatal("check should exist before removal")
	}

	// Remove it
	err := manager.RemoveCheck("test-id")
	if err != nil {
		t.Fatalf("RemoveCheck failed: %v", err)
	}

	// Verify it's gone
	_, exists = manager.GetCheck("test-id")
	if exists {
		t.Error("check should not exist after removal")
	}
}

func TestManager_RemoveCheck_NotFound(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	err := manager.RemoveCheck("nonexistent")
	if err == nil {
		t.Error("expected error when removing nonexistent check")
	}
}

func TestManager_AddCheck_HTTPType(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		Name:          "http-check",
		HTTP:          "http://localhost:8080/health",
		Method:        "POST",
		Headers:       map[string]string{"Authorization": "Bearer token"},
		TLSSkipVerify: true,
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.Type != CheckTypeHTTP {
		t.Errorf("expected type HTTP, got %s", check.Type)
	}
	if check.HTTP != "http://localhost:8080/health" {
		t.Errorf("expected HTTP URL, got %s", check.HTTP)
	}
	if check.Method != "POST" {
		t.Errorf("expected method POST, got %s", check.Method)
	}
	if !check.TLSSkipVerify {
		t.Error("expected TLSSkipVerify to be true")
	}
	if check.Headers["Authorization"] != "Bearer token" {
		t.Error("expected Authorization header")
	}
}

func TestManager_AddCheck_TCPType(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		Name: "tcp-check",
		TCP:  "localhost:5432",
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.Type != CheckTypeTCP {
		t.Errorf("expected type TCP, got %s", check.Type)
	}
	if check.TCP != "localhost:5432" {
		t.Errorf("expected TCP address, got %s", check.TCP)
	}
}

func TestManager_AddCheck_GRPCType(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		Name:       "grpc-check",
		GRPC:       "localhost:50051",
		GRPCUseTLS: true,
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.Type != CheckTypeGRPC {
		t.Errorf("expected type GRPC, got %s", check.Type)
	}
	if check.GRPC != "localhost:50051" {
		t.Errorf("expected GRPC address, got %s", check.GRPC)
	}
	if !check.GRPCUseTLS {
		t.Error("expected GRPCUseTLS to be true")
	}
}

func TestManager_AddCheck_TTLType(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		Name: "ttl-check",
		TTL:  "60s",
	}

	check, err := manager.AddCheck(def)
	if err != nil {
		t.Fatalf("AddCheck failed: %v", err)
	}

	if check.Type != CheckTypeTTL {
		t.Errorf("expected type TTL, got %s", check.Type)
	}
	if check.TTL != 60*time.Second {
		t.Errorf("expected TTL 60s, got %v", check.TTL)
	}
	if check.ExpiresAt.IsZero() {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestManager_UpdateTTLCheck(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		ID:   "ttl-check",
		Name: "ttl-check",
		TTL:  "60s",
	}

	check, _ := manager.AddCheck(def)

	// Initial status should be critical
	if check.Status != StatusCritical {
		t.Errorf("expected initial status Critical, got %s", check.Status)
	}

	// Update TTL
	err := manager.UpdateTTLCheck("ttl-check")
	if err != nil {
		t.Fatalf("UpdateTTLCheck failed: %v", err)
	}

	// Verify status changed to passing
	updated, _ := manager.GetCheck("ttl-check")
	if updated.Status != StatusPassing {
		t.Errorf("expected status Passing after update, got %s", updated.Status)
	}
	if updated.Output != "TTL check passed" {
		t.Errorf("expected output 'TTL check passed', got '%s'", updated.Output)
	}
}

func TestManager_UpdateTTLCheck_NotFound(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	err := manager.UpdateTTLCheck("nonexistent")
	if err == nil {
		t.Error("expected error when updating nonexistent check")
	}
}

func TestManager_UpdateTTLCheck_NotTTLType(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	def := &CheckDefinition{
		ID:   "http-check",
		Name: "http-check",
		HTTP: "http://localhost:8080",
	}

	manager.AddCheck(def)

	err := manager.UpdateTTLCheck("http-check")
	if err == nil {
		t.Error("expected error when updating non-TTL check")
	}
	if err.Error() != "check is not a TTL check" {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestManager_TTLCheckExpiration(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	// Create TTL check with very short duration
	def := &CheckDefinition{
		ID:   "ttl-check",
		Name: "ttl-check",
		TTL:  "100ms",
	}

	manager.AddCheck(def)

	// Update to make it passing
	manager.UpdateTTLCheck("ttl-check")

	// Get immediately - should be passing
	check, _ := manager.GetCheck("ttl-check")
	if check.Status != StatusPassing {
		t.Errorf("expected status Passing, got %s", check.Status)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Get again - should be critical
	check, _ = manager.GetCheck("ttl-check")
	if check.Status != StatusCritical {
		t.Errorf("expected status Critical after TTL expiration, got %s", check.Status)
	}
	if check.Output != "TTL expired" {
		t.Errorf("expected output 'TTL expired', got '%s'", check.Output)
	}
}

func TestManager_Stop(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)

	def := &CheckDefinition{
		Name: "test-check",
		HTTP: "http://localhost:8080",
	}

	manager.AddCheck(def)

	// Stop should not panic
	manager.Stop()

	// Verify context is cancelled
	select {
	case <-manager.ctx.Done():
		// Success - context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("expected context to be cancelled after Stop()")
	}
}

func TestManager_CheckTypePrecedence(t *testing.T) {
	log := logger.GetDefault()
	manager := NewManager(log)
	defer manager.Stop()

	tests := []struct {
		name     string
		def      *CheckDefinition
		expected CheckType
	}{
		{
			name:     "HTTP takes precedence",
			def:      &CheckDefinition{Name: "test", HTTP: "http://localhost", TCP: "localhost:80", TTL: "60s"},
			expected: CheckTypeHTTP,
		},
		{
			name:     "TCP takes precedence over TTL",
			def:      &CheckDefinition{Name: "test", TCP: "localhost:80", TTL: "60s"},
			expected: CheckTypeTCP,
		},
		{
			name:     "GRPC takes precedence over TTL",
			def:      &CheckDefinition{Name: "test", GRPC: "localhost:50051", TTL: "60s"},
			expected: CheckTypeGRPC,
		},
		{
			name:     "TTL by default",
			def:      &CheckDefinition{Name: "test", TTL: "60s"},
			expected: CheckTypeTTL,
		},
		{
			name:     "TTL when nothing specified",
			def:      &CheckDefinition{Name: "test"},
			expected: CheckTypeTTL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check, err := manager.AddCheck(tt.def)
			if err != nil {
				t.Fatalf("AddCheck failed: %v", err)
			}

			if check.Type != tt.expected {
				t.Errorf("expected type %s, got %s", tt.expected, check.Type)
			}

			// Clean up
			manager.RemoveCheck(check.ID)
		})
	}
}
