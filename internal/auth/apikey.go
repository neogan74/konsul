package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAPIKeyNotFound = errors.New("API key not found")
	ErrAPIKeyExpired  = errors.New("API key has expired")
	ErrAPIKeyDisabled = errors.New("API key is disabled")
)

type APIKey struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	KeyHash     string            `json:"key_hash"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time        `json:"last_used_at,omitempty"`
	Enabled     bool              `json:"enabled"`
}

type APIKeyService struct {
	keys   map[string]*APIKey // key hash -> APIKey
	mu     sync.RWMutex
	prefix string
}

func NewAPIKeyService(prefix string) *APIKeyService {
	if prefix == "" {
		prefix = "konsul"
	}
	return &APIKeyService{
		keys:   make(map[string]*APIKey),
		prefix: prefix,
	}
}

func (a *APIKeyService) GenerateAPIKey(name string, permissions []string, metadata map[string]string, expiresAt *time.Time) (string, *APIKey, error) {
	if name == "" {
		return "", nil, errors.New("API key name cannot be empty")
	}

	// Generate random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	// Create the full key with prefix
	keyString := fmt.Sprintf("%s_%s", a.prefix, hex.EncodeToString(keyBytes))

	// Hash the key for storage
	hasher := sha256.New()
	hasher.Write([]byte(keyString))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	apiKey := &APIKey{
		ID:          uuid.New().String(),
		Name:        name,
		KeyHash:     keyHash,
		Permissions: permissions,
		Metadata:    metadata,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Enabled:     true,
	}

	a.mu.Lock()
	a.keys[keyHash] = apiKey
	a.mu.Unlock()

	return keyString, apiKey, nil
}

func (a *APIKeyService) ValidateAPIKey(keyString string) (*APIKey, error) {
	if keyString == "" {
		return nil, ErrAPIKeyNotFound
	}

	// Hash the provided key
	hasher := sha256.New()
	hasher.Write([]byte(keyString))
	keyHash := hex.EncodeToString(hasher.Sum(nil))

	a.mu.RLock()
	apiKey, exists := a.keys[keyHash]
	a.mu.RUnlock()

	if !exists {
		return nil, ErrAPIKeyNotFound
	}

	if !apiKey.Enabled {
		return nil, ErrAPIKeyDisabled
	}

	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}

	// Update last used time
	a.mu.Lock()
	now := time.Now()
	apiKey.LastUsedAt = &now
	a.mu.Unlock()

	return apiKey, nil
}

func (a *APIKeyService) RevokeAPIKey(keyID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, apiKey := range a.keys {
		if apiKey.ID == keyID {
			apiKey.Enabled = false
			return nil
		}
	}

	return ErrAPIKeyNotFound
}

func (a *APIKeyService) DeleteAPIKey(keyID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for keyHash, apiKey := range a.keys {
		if apiKey.ID == keyID {
			delete(a.keys, keyHash)
			return nil
		}
	}

	return ErrAPIKeyNotFound
}

func (a *APIKeyService) ListAPIKeys() []*APIKey {
	a.mu.RLock()
	defer a.mu.RUnlock()

	keys := make([]*APIKey, 0, len(a.keys))
	for _, apiKey := range a.keys {
		// Create a copy without sensitive data
		keyCopy := &APIKey{
			ID:          apiKey.ID,
			Name:        apiKey.Name,
			Permissions: apiKey.Permissions,
			Metadata:    apiKey.Metadata,
			CreatedAt:   apiKey.CreatedAt,
			ExpiresAt:   apiKey.ExpiresAt,
			LastUsedAt:  apiKey.LastUsedAt,
			Enabled:     apiKey.Enabled,
		}
		keys = append(keys, keyCopy)
	}

	return keys
}

func (a *APIKeyService) GetAPIKey(keyID string) (*APIKey, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, apiKey := range a.keys {
		if apiKey.ID == keyID {
			// Create a copy without sensitive data
			keyCopy := &APIKey{
				ID:          apiKey.ID,
				Name:        apiKey.Name,
				Permissions: apiKey.Permissions,
				Metadata:    apiKey.Metadata,
				CreatedAt:   apiKey.CreatedAt,
				ExpiresAt:   apiKey.ExpiresAt,
				LastUsedAt:  apiKey.LastUsedAt,
				Enabled:     apiKey.Enabled,
			}
			return keyCopy, nil
		}
	}

	return nil, ErrAPIKeyNotFound
}

func (a *APIKeyService) UpdateAPIKey(keyID string, name string, permissions []string, metadata map[string]string, enabled *bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, apiKey := range a.keys {
		if apiKey.ID == keyID {
			if name != "" {
				apiKey.Name = name
			}
			if permissions != nil {
				apiKey.Permissions = permissions
			}
			if metadata != nil {
				apiKey.Metadata = metadata
			}
			if enabled != nil {
				apiKey.Enabled = *enabled
			}
			return nil
		}
	}

	return ErrAPIKeyNotFound
}

func (a *APIKeyService) HasPermission(apiKey *APIKey, permission string) bool {
	if apiKey == nil {
		return false
	}

	for _, perm := range apiKey.Permissions {
		if perm == permission || perm == "*" {
			return true
		}
	}

	return false
}