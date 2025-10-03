package ratelimit

import (
	"sync"
	"time"
)

// Limiter represents a token bucket rate limiter
type Limiter struct {
	rate       float64   // tokens per second
	burst      int       // maximum burst size
	tokens     float64   // current tokens
	lastUpdate time.Time // last token update time
	mu         sync.Mutex
}

// NewLimiter creates a new rate limiter with the given rate and burst
func NewLimiter(rate float64, burst int) *Limiter {
	return &Limiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow checks if a request is allowed based on the rate limit
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastUpdate).Seconds()

	// Add tokens based on elapsed time
	l.tokens += elapsed * l.rate
	if l.tokens > float64(l.burst) {
		l.tokens = float64(l.burst)
	}

	l.lastUpdate = now

	// Check if we have at least one token
	if l.tokens >= 1.0 {
		l.tokens -= 1.0
		return true
	}

	return false
}

// Tokens returns the current number of tokens (for testing/debugging)
func (l *Limiter) Tokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.tokens
}

// Reset resets the limiter to full capacity
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = float64(l.burst)
	l.lastUpdate = time.Now()
}

// Store manages rate limiters for multiple clients
type Store struct {
	limiters map[string]*Limiter
	rate     float64
	burst    int
	mu       sync.RWMutex
	cleanup  time.Duration
}

// NewStore creates a new rate limiter store
func NewStore(rate float64, burst int, cleanupInterval time.Duration) *Store {
	store := &Store{
		limiters: make(map[string]*Limiter),
		rate:     rate,
		burst:    burst,
		cleanup:  cleanupInterval,
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store
}

// GetLimiter gets or creates a limiter for the given key
func (s *Store) GetLimiter(key string) *Limiter {
	s.mu.RLock()
	limiter, exists := s.limiters[key]
	s.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check in case another goroutine created it
	if limiter, exists := s.limiters[key]; exists {
		return limiter
	}

	limiter = NewLimiter(s.rate, s.burst)
	s.limiters[key] = limiter
	return limiter
}

// Allow checks if a request from the given key is allowed
func (s *Store) Allow(key string) bool {
	limiter := s.GetLimiter(key)
	return limiter.Allow()
}

// Reset resets the limiter for the given key
func (s *Store) Reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.limiters, key)
}

// ResetAll resets all limiters
func (s *Store) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limiters = make(map[string]*Limiter)
}

// Count returns the number of tracked limiters
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.limiters)
}

// cleanupLoop periodically removes unused limiters
func (s *Store) cleanupLoop() {
	if s.cleanup == 0 {
		return
	}

	ticker := time.NewTicker(s.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupExpired()
	}
}

// cleanup removes limiters that haven't been used recently
func (s *Store) cleanupExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	threshold := 5 * time.Minute // Remove limiters idle for 5 minutes

	for key, limiter := range s.limiters {
		limiter.mu.Lock()
		idle := now.Sub(limiter.lastUpdate)
		limiter.mu.Unlock()

		if idle > threshold {
			delete(s.limiters, key)
		}
	}
}

// Config represents rate limiter configuration
type Config struct {
	Enabled         bool
	RequestsPerSec  float64
	Burst           int
	ByIP            bool
	ByAPIKey        bool
	CleanupInterval time.Duration
}

// Service manages rate limiting with different strategies
type Service struct {
	config   Config
	ipStore  *Store
	keyStore *Store
}

// NewService creates a new rate limiting service
func NewService(config Config) *Service {
	var ipStore, keyStore *Store

	if config.ByIP {
		ipStore = NewStore(config.RequestsPerSec, config.Burst, config.CleanupInterval)
	}

	if config.ByAPIKey {
		keyStore = NewStore(config.RequestsPerSec, config.Burst, config.CleanupInterval)
	}

	return &Service{
		config:   config,
		ipStore:  ipStore,
		keyStore: keyStore,
	}
}

// AllowIP checks if a request from the given IP is allowed
func (s *Service) AllowIP(ip string) bool {
	if !s.config.ByIP || s.ipStore == nil {
		return true
	}
	return s.ipStore.Allow(ip)
}

// AllowAPIKey checks if a request with the given API key is allowed
func (s *Service) AllowAPIKey(apiKey string) bool {
	if !s.config.ByAPIKey || s.keyStore == nil {
		return true
	}
	return s.keyStore.Allow(apiKey)
}

// ResetIP resets the rate limit for a specific IP
func (s *Service) ResetIP(ip string) {
	if s.ipStore != nil {
		s.ipStore.Reset(ip)
	}
}

// ResetAPIKey resets the rate limit for a specific API key
func (s *Service) ResetAPIKey(apiKey string) {
	if s.keyStore != nil {
		s.keyStore.Reset(apiKey)
	}
}

// Stats returns rate limiting statistics
func (s *Service) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	if s.ipStore != nil {
		stats["ip_limiters"] = s.ipStore.Count()
	}

	if s.keyStore != nil {
		stats["apikey_limiters"] = s.keyStore.Count()
	}

	return stats
}
