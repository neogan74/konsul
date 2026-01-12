package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	// Create limiter: 10 requests per second, burst of 5
	limiter := NewLimiter(10.0, 5)

	// Should allow burst requests
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed (within burst)", i)
		}
	}

	// 6th request should be denied (burst exhausted)
	if limiter.Allow() {
		t.Error("Request after burst should be denied")
	}

	// Wait for tokens to replenish (100ms = 1 token at 10/sec)
	time.Sleep(150 * time.Millisecond)

	// Should allow one more request
	if !limiter.Allow() {
		t.Error("Request after wait should be allowed")
	}
}

func TestLimiter_Reset(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	// Exhaust all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	// Should be denied
	if limiter.Allow() {
		t.Error("Request should be denied after exhausting tokens")
	}

	// Reset
	limiter.Reset()

	// Should be allowed after reset
	if !limiter.Allow() {
		t.Error("Request should be allowed after reset")
	}
}

func TestLimiter_Tokens(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	initialTokens := limiter.Tokens()
	if initialTokens != 5.0 {
		t.Errorf("Initial tokens = %v, want 5.0", initialTokens)
	}

	limiter.Allow()
	afterOne := limiter.Tokens()
	if afterOne != 4.0 {
		t.Errorf("Tokens after one request = %v, want 4.0", afterOne)
	}
}

func TestStore_GetLimiter(t *testing.T) {
	store := NewStore(10.0, 5, 0)

	limiter1 := store.GetLimiter("client1")
	limiter2 := store.GetLimiter("client1")

	if limiter1 != limiter2 {
		t.Error("GetLimiter should return same limiter for same key")
	}

	limiter3 := store.GetLimiter("client2")
	if limiter1 == limiter3 {
		t.Error("GetLimiter should return different limiter for different key")
	}
}

func TestStore_Allow(t *testing.T) {
	store := NewStore(10.0, 2, 0)

	// Client1 should get 2 requests (burst)
	if !store.Allow("client1") {
		t.Error("First request for client1 should be allowed")
	}
	if !store.Allow("client1") {
		t.Error("Second request for client1 should be allowed")
	}
	if store.Allow("client1") {
		t.Error("Third request for client1 should be denied")
	}

	// Client2 should have independent limit
	if !store.Allow("client2") {
		t.Error("First request for client2 should be allowed")
	}
	if !store.Allow("client2") {
		t.Error("Second request for client2 should be allowed")
	}
}

func TestStore_Reset(t *testing.T) {
	store := NewStore(10.0, 2, 0)

	// Exhaust client's tokens
	store.Allow("client1")
	store.Allow("client1")

	// Should be denied
	if store.Allow("client1") {
		t.Error("Request should be denied")
	}

	// Reset client
	store.Reset("client1")

	// Should be allowed after reset
	if !store.Allow("client1") {
		t.Error("Request should be allowed after reset")
	}
}

func TestStore_ResetAll(t *testing.T) {
	store := NewStore(10.0, 2, 0)

	// Create limiters for multiple clients
	store.Allow("client1")
	store.Allow("client2")
	store.Allow("client3")

	if store.Count() != 3 {
		t.Errorf("Count = %d, want 3", store.Count())
	}

	store.ResetAll()

	if store.Count() != 0 {
		t.Errorf("Count after ResetAll = %d, want 0", store.Count())
	}
}

func TestStore_Count(t *testing.T) {
	store := NewStore(10.0, 5, 0)

	if store.Count() != 0 {
		t.Errorf("Initial count = %d, want 0", store.Count())
	}

	store.GetLimiter("client1")
	store.GetLimiter("client2")
	store.GetLimiter("client3")

	if store.Count() != 3 {
		t.Errorf("Count = %d, want 3", store.Count())
	}
}

func TestService_AllowIP(t *testing.T) {
	config := Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           2,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 0,
	}

	service := NewService(config)

	// Should allow first 2 requests
	if !service.AllowIP("192.168.1.1") {
		t.Error("First IP request should be allowed")
	}
	if !service.AllowIP("192.168.1.1") {
		t.Error("Second IP request should be allowed")
	}

	// Third should be denied
	if service.AllowIP("192.168.1.1") {
		t.Error("Third IP request should be denied")
	}

	// Different IP should have independent limit
	if !service.AllowIP("192.168.1.2") {
		t.Error("First request from different IP should be allowed")
	}
}

func TestService_AllowAPIKey(t *testing.T) {
	config := Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           2,
		ByIP:            false,
		ByAPIKey:        true,
		CleanupInterval: 0,
	}

	service := NewService(config)

	// Should allow first 2 requests
	if !service.AllowAPIKey("key123") {
		t.Error("First API key request should be allowed")
	}
	if !service.AllowAPIKey("key123") {
		t.Error("Second API key request should be allowed")
	}

	// Third should be denied
	if service.AllowAPIKey("key123") {
		t.Error("Third API key request should be denied")
	}

	// Different key should have independent limit
	if !service.AllowAPIKey("key456") {
		t.Error("First request from different API key should be allowed")
	}
}

func TestService_Disabled(t *testing.T) {
	config := Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           1,
		ByIP:            false, // Disabled
		ByAPIKey:        false, // Disabled
		CleanupInterval: 0,
	}

	service := NewService(config)

	// When disabled, should always allow
	for i := 0; i < 10; i++ {
		if !service.AllowIP("192.168.1.1") {
			t.Error("IP rate limiting should allow when disabled")
		}
		if !service.AllowAPIKey("key123") {
			t.Error("API key rate limiting should allow when disabled")
		}
	}
}

func TestService_Reset(t *testing.T) {
	config := Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           1,
		ByIP:            true,
		ByAPIKey:        true,
		CleanupInterval: 0,
	}

	service := NewService(config)

	// Exhaust limits
	service.AllowIP("192.168.1.1")
	service.AllowAPIKey("key123")

	// Should be denied
	if service.AllowIP("192.168.1.1") {
		t.Error("Should be rate limited")
	}
	if service.AllowAPIKey("key123") {
		t.Error("Should be rate limited")
	}

	// Reset
	service.ResetIP("192.168.1.1")
	service.ResetAPIKey("key123")

	// Should be allowed after reset
	if !service.AllowIP("192.168.1.1") {
		t.Error("Should be allowed after reset")
	}
	if !service.AllowAPIKey("key123") {
		t.Error("Should be allowed after reset")
	}
}

func TestService_Stats(t *testing.T) {
	config := Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           5,
		ByIP:            true,
		ByAPIKey:        true,
		CleanupInterval: 0,
	}

	service := NewService(config)

	// Create some limiters
	service.AllowIP("192.168.1.1")
	service.AllowIP("192.168.1.2")
	service.AllowAPIKey("key1")
	service.AllowAPIKey("key2")
	service.AllowAPIKey("key3")

	stats := service.Stats()

	if stats["ip_limiters"] != 2 {
		t.Errorf("IP limiters = %v, want 2", stats["ip_limiters"])
	}

	if stats["apikey_limiters"] != 3 {
		t.Errorf("API key limiters = %v, want 3", stats["apikey_limiters"])
	}
}

func BenchmarkLimiter_Allow(b *testing.B) {
	limiter := NewLimiter(1000.0, 100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		limiter.Allow()
	}
}

func BenchmarkStore_Allow(b *testing.B) {
	store := NewStore(1000.0, 100, 0)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.Allow("client1")
	}
}

func BenchmarkStore_AllowParallel(b *testing.B) {
	store := NewStore(1000.0, 100, 0)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		clientID := 0
		for pb.Next() {
			store.Allow(string(rune(clientID % 10)))
			clientID++
		}
	})
}

func TestService_IsWhitelisted(t *testing.T) {
	service := NewService(Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           2,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 0,
	})

	// Add to whitelist
	if err := service.GetAccessList().AddToWhitelist(WhitelistEntry{
		Identifier: "192.168.1.10",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
	}); err != nil {
		t.Fatalf("Failed to add to whitelist: %v", err)
	}

	if !service.IsWhitelisted("192.168.1.10") {
		t.Error("Expected IP to be whitelisted")
	}

	if service.IsWhitelisted("192.168.1.11") {
		t.Error("Expected different IP not to be whitelisted")
	}
}

func TestService_IsBlacklisted(t *testing.T) {
	service := NewService(Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           2,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 0,
	})

	// Add to blacklist
	if err := service.GetAccessList().AddToBlacklist(BlacklistEntry{
		Identifier: "192.168.1.20",
		Type:       "ip",
		Reason:     "Abuse",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}); err != nil {
		t.Fatalf("Failed to add to blacklist: %v", err)
	}

	if !service.IsBlacklisted("192.168.1.20") {
		t.Error("Expected IP to be blacklisted")
	}

	if service.IsBlacklisted("192.168.1.21") {
		t.Error("Expected different IP not to be blacklisted")
	}
}

func TestService_GetActiveClients(t *testing.T) {
	service := NewService(Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           5,
		ByIP:            true,
		ByAPIKey:        true,
		CleanupInterval: 0,
	})

	// Create some clients
	service.AllowIP("192.168.1.1")
	service.AllowIP("192.168.1.2")
	service.AllowAPIKey("key1")
	service.AllowAPIKey("key2")

	// Test get all
	allClients := service.GetActiveClients("all")
	if len(allClients) != 4 {
		t.Errorf("Expected 4 total clients, got %d", len(allClients))
	}

	// Test get IP only
	ipClients := service.GetActiveClients("ip")
	if len(ipClients) != 2 {
		t.Errorf("Expected 2 IP clients, got %d", len(ipClients))
	}

	// Test get API key only
	keyClients := service.GetActiveClients("apikey")
	if len(keyClients) != 2 {
		t.Errorf("Expected 2 API key clients, got %d", len(keyClients))
	}
}

func TestService_GetClientStatus(t *testing.T) {
	service := NewService(Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           5,
		ByIP:            true,
		ByAPIKey:        true,
		CleanupInterval: 0,
	})

	// Create a client
	testIP := "192.168.1.100"
	service.AllowIP(testIP)

	// Get status
	status := service.GetClientStatus(testIP)
	if status == nil {
		t.Fatal("Expected client status to be found")
	}

	if status.Identifier != testIP {
		t.Errorf("Expected identifier %s, got %s", testIP, status.Identifier)
	}

	if status.Type != "ip" {
		t.Errorf("Expected type 'ip', got %s", status.Type)
	}

	// Non-existent client
	notFound := service.GetClientStatus("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for non-existent client")
	}
}

func TestService_UpdateConfig(t *testing.T) {
	service := NewService(Config{
		Enabled:         true,
		RequestsPerSec:  10.0,
		Burst:           5,
		ByIP:            true,
		ByAPIKey:        false,
		CleanupInterval: 0,
	})

	newRate := 20.0
	newBurst := 10

	// Update config
	changed := service.UpdateConfig(&newRate, &newBurst)
	if !changed {
		t.Error("Expected config to be changed")
	}

	// Verify changes
	config := service.GetConfig()
	if config.RequestsPerSec != 20.0 {
		t.Errorf("Expected RequestsPerSec 20.0, got %.1f", config.RequestsPerSec)
	}
	if config.Burst != 10 {
		t.Errorf("Expected Burst 10, got %d", config.Burst)
	}

	// Update with same values (no change)
	changed = service.UpdateConfig(&newRate, &newBurst)
	if changed {
		t.Error("Expected no change when updating with same values")
	}
}

func TestStore_GetClients(t *testing.T) {
	store := NewStore(10.0, 5, 0)

	// Create some clients
	store.Allow("client1")
	store.Allow("client2")
	store.Allow("client3")

	clients := store.GetClients("test-type")
	if len(clients) != 3 {
		t.Errorf("Expected 3 clients, got %d", len(clients))
	}

	// Verify client info structure
	for _, client := range clients {
		if client.Type != "test-type" {
			t.Errorf("Expected type 'test-type', got %s", client.Type)
		}
		if client.Rate != 10.0 {
			t.Errorf("Expected rate 10.0, got %.1f", client.Rate)
		}
		if client.MaxTokens != 5 {
			t.Errorf("Expected max tokens 5, got %d", client.MaxTokens)
		}
	}
}

func TestStore_GetClientStatus(t *testing.T) {
	store := NewStore(10.0, 5, 0)

	// Create a client
	identifier := "test-client"
	store.Allow(identifier)

	// Get status
	status := store.GetClientStatus(identifier, "test-type")
	if status == nil {
		t.Fatal("Expected client status to be found")
	}

	if status.Identifier != identifier {
		t.Errorf("Expected identifier %s, got %s", identifier, status.Identifier)
	}

	// Non-existent client
	notFound := store.GetClientStatus("nonexistent", "test-type")
	if notFound != nil {
		t.Error("Expected nil for non-existent client")
	}
}
