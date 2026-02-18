// Package ratelinit represents rate limit implementation
package ratelimit

import (
	"testing"
	"time"
)

func TestAccessList_WhitelistBasic(t *testing.T) {
	al := NewAccessList()

	entry := WhitelistEntry{
		Identifier: "192.168.1.100",
		Type:       "ip",
		Reason:     "Test whitelist",
		AddedBy:    "admin",
	}

	err := al.AddToWhitelist(entry)
	if err != nil {
		t.Fatalf("Failed to add to whitelist: %v", err)
	}

	if !al.IsWhitelisted("192.168.1.100") {
		t.Error("Expected IP to be whitelisted")
	}

	if al.IsWhitelisted("192.168.1.101") {
		t.Error("Expected different IP not to be whitelisted")
	}
}

func TestAccessList_WhitelistWithExpiry(t *testing.T) {
	al := NewAccessList()

	expires := time.Now().Add(50 * time.Millisecond)
	entry := WhitelistEntry{
		Identifier: "test-key",
		Type:       "apikey",
		Reason:     "Temporary access",
		AddedBy:    "admin",
		ExpiresAt:  &expires,
	}

	err := al.AddToWhitelist(entry)
	if err != nil {
		t.Fatalf("Failed to add to whitelist: %v", err)
	}

	// Should be whitelisted initially
	if !al.IsWhitelisted("test-key") {
		t.Error("Expected key to be whitelisted")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Should no longer be whitelisted
	if al.IsWhitelisted("test-key") {
		t.Error("Expected key to be expired from whitelist")
	}
}

func TestAccessList_RemoveFromWhitelist(t *testing.T) {
	al := NewAccessList()

	entry := WhitelistEntry{
		Identifier: "192.168.1.100",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
	}

	if err := al.AddToWhitelist(entry); err != nil {
		t.Fatalf("Failed to add to whitelist: %v", err)
	}

	if !al.IsWhitelisted("192.168.1.100") {
		t.Fatal("Expected IP to be whitelisted")
	}

	removed := al.RemoveFromWhitelist("192.168.1.100")
	if !removed {
		t.Error("Expected removal to return true")
	}

	if al.IsWhitelisted("192.168.1.100") {
		t.Error("Expected IP to no longer be whitelisted")
	}

	// Removing again should return false
	removed = al.RemoveFromWhitelist("192.168.1.100")
	if removed {
		t.Error("Expected removal of non-existent entry to return false")
	}
}

func TestAccessList_BlacklistBasic(t *testing.T) {
	al := NewAccessList()

	entry := BlacklistEntry{
		Identifier: "192.168.1.200",
		Type:       "ip",
		Reason:     "Abuse",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	err := al.AddToBlacklist(entry)
	if err != nil {
		t.Fatalf("Failed to add to blacklist: %v", err)
	}

	if !al.IsBlacklisted("192.168.1.200") {
		t.Error("Expected IP to be blacklisted")
	}

	if al.IsBlacklisted("192.168.1.201") {
		t.Error("Expected different IP not to be blacklisted")
	}
}

func TestAccessList_BlacklistExpiry(t *testing.T) {
	al := NewAccessList()

	entry := BlacklistEntry{
		Identifier: "bad-key",
		Type:       "apikey",
		Reason:     "Temporary block",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(50 * time.Millisecond),
	}

	err := al.AddToBlacklist(entry)
	if err != nil {
		t.Fatalf("Failed to add to blacklist: %v", err)
	}

	// Should be blacklisted initially
	if !al.IsBlacklisted("bad-key") {
		t.Error("Expected key to be blacklisted")
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Should no longer be blacklisted
	if al.IsBlacklisted("bad-key") {
		t.Error("Expected key to be expired from blacklist")
	}
}

func TestAccessList_RemoveFromBlacklist(t *testing.T) {
	al := NewAccessList()

	entry := BlacklistEntry{
		Identifier: "192.168.1.200",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	if err := al.AddToBlacklist(entry); err != nil {
		t.Fatalf("Failed to add to blacklist: %v", err)
	}

	if !al.IsBlacklisted("192.168.1.200") {
		t.Fatal("Expected IP to be blacklisted")
	}

	removed := al.RemoveFromBlacklist("192.168.1.200")
	if !removed {
		t.Error("Expected removal to return true")
	}

	if al.IsBlacklisted("192.168.1.200") {
		t.Error("Expected IP to no longer be blacklisted")
	}
}

func TestAccessList_GetWhitelist(t *testing.T) {
	al := NewAccessList()

	entries := []WhitelistEntry{
		{Identifier: "ip1", Type: "ip", Reason: "Test 1", AddedBy: "admin"},
		{Identifier: "ip2", Type: "ip", Reason: "Test 2", AddedBy: "admin"},
		{Identifier: "key1", Type: "apikey", Reason: "Test 3", AddedBy: "admin"},
	}

	for _, entry := range entries {
		if err := al.AddToWhitelist(entry); err != nil {
			t.Fatalf("Failed to add to whitelist: %v", err)
		}
	}

	list := al.GetWhitelist()

	if len(list) != 3 {
		t.Errorf("Expected 3 whitelist entries, got %d", len(list))
	}
}

func TestAccessList_GetWhitelist_ExcludesExpired(t *testing.T) {
	al := NewAccessList()

	expires := time.Now().Add(50 * time.Millisecond)
	entries := []WhitelistEntry{
		{Identifier: "active", Type: "ip", Reason: "Active", AddedBy: "admin"},
		{Identifier: "expiring", Type: "ip", Reason: "Expiring", AddedBy: "admin", ExpiresAt: &expires},
	}

	for _, entry := range entries {
		if err := al.AddToWhitelist(entry); err != nil {
			t.Fatalf("Failed to add to whitelist: %v", err)
		}
	}

	// Initially should have 2
	list := al.GetWhitelist()
	if len(list) != 2 {
		t.Errorf("Expected 2 entries initially, got %d", len(list))
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Should only have 1 now
	list = al.GetWhitelist()
	if len(list) != 1 {
		t.Errorf("Expected 1 entry after expiry, got %d", len(list))
	}

	if list[0].Identifier != "active" {
		t.Errorf("Expected 'active' entry to remain, got %s", list[0].Identifier)
	}
}

func TestAccessList_GetBlacklist(t *testing.T) {
	al := NewAccessList()

	entries := []BlacklistEntry{
		{Identifier: "bad1", Type: "ip", Reason: "Abuse 1", AddedBy: "admin", ExpiresAt: time.Now().Add(1 * time.Hour)},
		{Identifier: "bad2", Type: "ip", Reason: "Abuse 2", AddedBy: "admin", ExpiresAt: time.Now().Add(1 * time.Hour)},
	}

	for _, entry := range entries {
		if err := al.AddToBlacklist(entry); err != nil {
			t.Fatalf("Failed to add to blacklist: %v", err)
		}
	}

	list := al.GetBlacklist()

	if len(list) != 2 {
		t.Errorf("Expected 2 blacklist entries, got %d", len(list))
	}
}

func TestAccessList_Count(t *testing.T) {
	al := NewAccessList()

	// Add whitelist entries
	_ = al.AddToWhitelist(WhitelistEntry{
		Identifier: "white1",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
	})
	_ = al.AddToWhitelist(WhitelistEntry{
		Identifier: "white2",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
	})

	// Add blacklist entries
	_ = al.AddToBlacklist(BlacklistEntry{
		Identifier: "black1",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	})

	whitelisted, blacklisted := al.Count()

	if whitelisted != 2 {
		t.Errorf("Expected 2 whitelisted, got %d", whitelisted)
	}

	if blacklisted != 1 {
		t.Errorf("Expected 1 blacklisted, got %d", blacklisted)
	}
}

func TestAccessList_Validation(t *testing.T) {
	al := NewAccessList()

	// Test empty identifier
	err := al.AddToWhitelist(WhitelistEntry{
		Identifier: "",
		Type:       "ip",
		Reason:     "Test",
	})
	if err == nil {
		t.Error("Expected error for empty identifier")
	}

	// Test invalid type
	err = al.AddToWhitelist(WhitelistEntry{
		Identifier: "test",
		Type:       "invalid",
		Reason:     "Test",
	})
	if err == nil {
		t.Error("Expected error for invalid type")
	}

	// Test valid types
	validTypes := []string{"ip", "apikey"}
	for _, typ := range validTypes {
		err := al.AddToWhitelist(WhitelistEntry{
			Identifier: "test-" + typ,
			Type:       typ,
			Reason:     "Test",
			AddedBy:    "admin",
		})
		if err != nil {
			t.Errorf("Expected no error for valid type %s: %v", typ, err)
		}
	}
}

func TestAccessList_BlacklistRequiresExpiry(t *testing.T) {
	al := NewAccessList()

	// Test missing expiry (zero time)
	err := al.AddToBlacklist(BlacklistEntry{
		Identifier: "test",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
		ExpiresAt:  time.Time{}, // Zero time
	})
	if err == nil {
		t.Error("Expected error for missing expiry on blacklist entry")
	}
}

func TestAccessList_GetEntry(t *testing.T) {
	al := NewAccessList()

	entry := WhitelistEntry{
		Identifier: "test-id",
		Type:       "ip",
		Reason:     "Testing",
		AddedBy:    "admin",
	}

	if err := al.AddToWhitelist(entry); err != nil {
		t.Fatalf("Failed to add to whitelist: %v", err)
	}

	retrieved := al.GetWhitelistEntry("test-id")
	if retrieved == nil {
		t.Fatal("Expected to retrieve whitelist entry")
	}

	if retrieved.Identifier != "test-id" {
		t.Errorf("Expected identifier test-id, got %s", retrieved.Identifier)
	}

	if retrieved.Reason != "Testing" {
		t.Errorf("Expected reason 'Testing', got %s", retrieved.Reason)
	}

	// Non-existent entry
	notFound := al.GetWhitelistEntry("nonexistent")
	if notFound != nil {
		t.Error("Expected nil for non-existent entry")
	}
}

func TestAccessList_CleanupExpiredEntries(t *testing.T) {
	al := NewAccessList()

	// Add expired whitelist entry
	// Add expired whitelist entry
	pastTime := time.Now().Add(-1 * time.Hour)
	_ = al.AddToWhitelist(WhitelistEntry{
		Identifier: "expired-white",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
		ExpiresAt:  &pastTime,
	})

	// Add expired blacklist entry
	_ = al.AddToBlacklist(BlacklistEntry{
		Identifier: "expired-black",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	})

	// Add non-expired entries
	_ = al.AddToWhitelist(WhitelistEntry{
		Identifier: "active-white",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
	})

	_ = al.AddToBlacklist(BlacklistEntry{
		Identifier: "active-black",
		Type:       "ip",
		Reason:     "Test",
		AddedBy:    "admin",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	})

	// Manually trigger cleanup
	al.cleanup()

	// Check that expired entries are removed
	if al.IsWhitelisted("expired-white") {
		t.Error("Expected expired whitelist entry to be cleaned up")
	}

	if al.IsBlacklisted("expired-black") {
		t.Error("Expected expired blacklist entry to be cleaned up")
	}

	// Check that active entries remain
	if !al.IsWhitelisted("active-white") {
		t.Error("Expected active whitelist entry to remain")
	}

	if !al.IsBlacklisted("active-black") {
		t.Error("Expected active blacklist entry to remain")
	}
}
