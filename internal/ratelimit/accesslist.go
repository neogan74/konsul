package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

// AccessList manages whitelisted and blacklisted identifiers
type AccessList struct {
	whitelisted map[string]*WhitelistEntry
	blacklisted map[string]*BlacklistEntry
	mu          sync.RWMutex
}

// WhitelistEntry represents a whitelisted identifier
type WhitelistEntry struct {
	Identifier string     `json:"identifier"`
	Type       string     `json:"type"`       // "ip" or "apikey"
	Reason     string     `json:"reason"`     // Why it's whitelisted
	AddedAt    time.Time  `json:"added_at"`   // When it was added
	ExpiresAt  *time.Time `json:"expires_at"` // Optional expiry (nil = never expires)
	AddedBy    string     `json:"added_by"`   // Who added it
}

// BlacklistEntry represents a blacklisted identifier
type BlacklistEntry struct {
	Identifier string    `json:"identifier"`
	Type       string    `json:"type"`       // "ip" or "apikey"
	Reason     string    `json:"reason"`     // Why it's blacklisted
	AddedAt    time.Time `json:"added_at"`   // When it was added
	ExpiresAt  time.Time `json:"expires_at"` // When it expires
	AddedBy    string    `json:"added_by"`   // Who added it
}

// NewAccessList creates a new access list manager
func NewAccessList() *AccessList {
	al := &AccessList{
		whitelisted: make(map[string]*WhitelistEntry),
		blacklisted: make(map[string]*BlacklistEntry),
	}

	// Start cleanup goroutine for expired entries
	go al.cleanupExpiredEntries()

	return al
}

// IsWhitelisted checks if an identifier is whitelisted
func (a *AccessList) IsWhitelisted(identifier string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, exists := a.whitelisted[identifier]
	if !exists {
		return false
	}

	// Check if expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return false
	}

	return true
}

// IsBlacklisted checks if an identifier is blacklisted
func (a *AccessList) IsBlacklisted(identifier string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, exists := a.blacklisted[identifier]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	return true
}

// AddToWhitelist adds an identifier to the whitelist
func (a *AccessList) AddToWhitelist(entry WhitelistEntry) error {
	if entry.Identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if entry.Type != "ip" && entry.Type != "apikey" {
		return fmt.Errorf("type must be 'ip' or 'apikey'")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Set defaults
	if entry.AddedAt.IsZero() {
		entry.AddedAt = time.Now()
	}

	a.whitelisted[entry.Identifier] = &entry
	return nil
}

// AddToBlacklist adds an identifier to the blacklist
func (a *AccessList) AddToBlacklist(entry BlacklistEntry) error {
	if entry.Identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if entry.Type != "ip" && entry.Type != "apikey" {
		return fmt.Errorf("type must be 'ip' or 'apikey'")
	}
	if entry.ExpiresAt.IsZero() {
		return fmt.Errorf("expiresAt is required for blacklist entries")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Set defaults
	if entry.AddedAt.IsZero() {
		entry.AddedAt = time.Now()
	}

	a.blacklisted[entry.Identifier] = &entry
	return nil
}

// RemoveFromWhitelist removes an identifier from the whitelist
func (a *AccessList) RemoveFromWhitelist(identifier string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.whitelisted[identifier]; exists {
		delete(a.whitelisted, identifier)
		return true
	}
	return false
}

// RemoveFromBlacklist removes an identifier from the blacklist
func (a *AccessList) RemoveFromBlacklist(identifier string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.blacklisted[identifier]; exists {
		delete(a.blacklisted, identifier)
		return true
	}
	return false
}

// GetWhitelist returns all whitelisted identifiers
func (a *AccessList) GetWhitelist() []WhitelistEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entries := make([]WhitelistEntry, 0, len(a.whitelisted))
	now := time.Now()

	for _, entry := range a.whitelisted {
		// Skip expired entries
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			continue
		}
		entries = append(entries, *entry)
	}

	return entries
}

// GetBlacklist returns all blacklisted identifiers
func (a *AccessList) GetBlacklist() []BlacklistEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entries := make([]BlacklistEntry, 0, len(a.blacklisted))
	now := time.Now()

	for _, entry := range a.blacklisted {
		// Skip expired entries
		if now.After(entry.ExpiresAt) {
			continue
		}
		entries = append(entries, *entry)
	}

	return entries
}

// GetWhitelistEntry returns a specific whitelist entry
func (a *AccessList) GetWhitelistEntry(identifier string) *WhitelistEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, exists := a.whitelisted[identifier]
	if !exists {
		return nil
	}

	// Check if expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return nil
	}

	// Return a copy
	entryCopy := *entry
	return &entryCopy
}

// GetBlacklistEntry returns a specific blacklist entry
func (a *AccessList) GetBlacklistEntry(identifier string) *BlacklistEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, exists := a.blacklisted[identifier]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	// Return a copy
	entryCopy := *entry
	return &entryCopy
}

// Count returns the number of active whitelist and blacklist entries
func (a *AccessList) Count() (whitelisted int, blacklisted int) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := time.Now()

	// Count non-expired whitelist entries
	for _, entry := range a.whitelisted {
		if entry.ExpiresAt == nil || now.Before(*entry.ExpiresAt) {
			whitelisted++
		}
	}

	// Count non-expired blacklist entries
	for _, entry := range a.blacklisted {
		if now.Before(entry.ExpiresAt) {
			blacklisted++
		}
	}

	return
}

// cleanupExpiredEntries periodically removes expired entries
func (a *AccessList) cleanupExpiredEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		a.cleanup()
	}
}

// cleanup removes expired entries from both lists
func (a *AccessList) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()

	// Clean expired whitelist entries
	for key, entry := range a.whitelisted {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			delete(a.whitelisted, key)
		}
	}

	// Clean expired blacklist entries
	for key, entry := range a.blacklisted {
		if now.After(entry.ExpiresAt) {
			delete(a.blacklisted, key)
		}
	}
}
