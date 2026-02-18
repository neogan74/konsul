package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter_SetCustomConfig(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	// Set custom config
	limiter.SetCustomConfig(20.0, 10, 1*time.Hour)

	// Verify custom config is active
	rate, burst := limiter.getEffectiveConfig()
	if rate != 20.0 {
		t.Errorf("Expected custom rate 20.0, got %.1f", rate)
	}
	if burst != 10 {
		t.Errorf("Expected custom burst 10, got %d", burst)
	}
}

func TestLimiter_CustomConfigExpiry(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	// Set custom config with very short duration
	limiter.SetCustomConfig(20.0, 10, 10*time.Millisecond)

	// Verify custom config is active
	rate, _ := limiter.getEffectiveConfig()
	if rate != 20.0 {
		t.Errorf("Expected custom rate 20.0, got %.1f", rate)
	}

	// Wait for expiry
	time.Sleep(50 * time.Millisecond)

	// Verify reverted to default config
	rate, burst := limiter.getEffectiveConfig()
	if rate != 10.0 {
		t.Errorf("Expected default rate 10.0 after expiry, got %.1f", rate)
	}
	if burst != 5 {
		t.Errorf("Expected default burst 5 after expiry, got %d", burst)
	}
}

func TestLimiter_ClearCustomConfig(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	// Set custom config
	limiter.SetCustomConfig(20.0, 10, 1*time.Hour)

	// Clear it
	limiter.ClearCustomConfig()

	// Verify reverted to default
	rate, burst := limiter.getEffectiveConfig()
	if rate != 10.0 {
		t.Errorf("Expected default rate 10.0 after clear, got %.1f", rate)
	}
	if burst != 5 {
		t.Errorf("Expected default burst 5 after clear, got %d", burst)
	}
}

func TestLimiter_CustomConfigWithLowerBurst(t *testing.T) {
	limiter := NewLimiter(10.0, 10)

	// Fill bucket
	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Fatal("Expected token to be available")
		}
	}

	// Set custom config with lower burst (should adjust tokens)
	limiter.SetCustomConfig(10.0, 5, 1*time.Hour)

	// Tokens should be capped at new burst
	limit, remaining, _ := limiter.GetHeaders()
	if limit != 5 {
		t.Errorf("Expected limit 5, got %d", limit)
	}
	if remaining > 5 {
		t.Errorf("Expected remaining <= 5 after burst reduction, got %d", remaining)
	}
}

func TestLimiter_GetStats(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	// Allow some requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Fatal("Expected request to be allowed")
		}
	}

	// Deny some requests
	for i := 0; i < 5; i++ {
		limiter.Allow() // Will eventually deny
	}

	allowed, denied, violations := limiter.GetStats()

	if allowed < 3 {
		t.Errorf("Expected at least 3 allowed requests, got %d", allowed)
	}

	if denied == 0 {
		t.Errorf("Expected some denied requests, got %d", denied)
	}

	if len(violations) != int(denied) {
		t.Errorf("Expected %d violations, got %d", denied, len(violations))
	}
}

func TestLimiter_ViolationTracking(t *testing.T) {
	limiter := NewLimiter(1.0, 2)

	// Exhaust tokens
	limiter.Allow()
	limiter.Allow()

	// Record violations
	for i := 0; i < 5; i++ {
		limiter.AllowWithEndpoint("/api/test")
	}

	_, _, violations := limiter.GetStats()

	if len(violations) != 5 {
		t.Errorf("Expected 5 violations, got %d", len(violations))
	}

	// Check violation details
	for _, v := range violations {
		if v.Endpoint != "/api/test" {
			t.Errorf("Expected endpoint /api/test, got %s", v.Endpoint)
		}
		if v.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	}
}

func TestLimiter_ViolationHistoryLimit(t *testing.T) {
	limiter := NewLimiter(1.0, 1)

	// Exhaust token
	limiter.Allow()

	// Create more than 100 violations
	for i := 0; i < 150; i++ {
		limiter.AllowWithEndpoint("/test")
	}

	_, _, violations := limiter.GetStats()

	// Should be capped at 100
	if len(violations) != 100 {
		t.Errorf("Expected violation history limited to 100, got %d", len(violations))
	}
}

func TestLimiter_GetTimestamps(t *testing.T) {
	beforeCreation := time.Now()
	time.Sleep(1 * time.Millisecond)

	limiter := NewLimiter(10.0, 5)

	time.Sleep(1 * time.Millisecond)
	afterCreation := time.Now()

	firstSeen, lastRequest := limiter.GetTimestamps()

	if firstSeen.Before(beforeCreation) || firstSeen.After(afterCreation) {
		t.Errorf("firstSeen timestamp out of expected range")
	}

	if lastRequest.Before(beforeCreation) || lastRequest.After(afterCreation) {
		t.Errorf("lastRequest timestamp out of expected range")
	}
}

func TestLimiter_GetHeaders(t *testing.T) {
	limiter := NewLimiter(10.0, 5)

	limit, remaining, resetAt := limiter.GetHeaders()

	// Check limit matches burst
	if limit != 5 {
		t.Errorf("Expected limit 5, got %d", limit)
	}

	// Check remaining is positive
	if remaining < 0 || remaining > 5 {
		t.Errorf("Expected remaining 0-5, got %d", remaining)
	}

	// Check reset timestamp is valid
	now := time.Now().Unix()
	if resetAt < now || resetAt > now+10 {
		t.Errorf("Reset timestamp seems invalid: %d (now: %d)", resetAt, now)
	}
}

func TestLimiter_GetHeaders_AfterExhaustion(t *testing.T) {
	limiter := NewLimiter(1.0, 2)

	// Exhaust tokens
	limiter.Allow()
	limiter.Allow()
	limiter.Allow() // This will fail

	limit, remaining, resetAt := limiter.GetHeaders()

	if limit != 2 {
		t.Errorf("Expected limit 2, got %d", limit)
	}

	if remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", remaining)
	}

	// Reset time should be in the future
	now := time.Now().Unix()
	if resetAt <= now {
		t.Errorf("Expected reset time in future, got %d (now: %d)", resetAt, now)
	}

	// Should be within reasonable bounds (max 2 seconds for 1.0 rate)
	if resetAt > now+2 {
		t.Errorf("Reset time too far in future: %d (now: %d)", resetAt, now)
	}
}
