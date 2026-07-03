// Package namespace provides multi-tenancy namespace management for Konsul.
package namespace

import (
	"errors"
	"time"
)

const (
	// DefaultNamespace is the implicit namespace for all pre-namespace data.
	DefaultNamespace = "default"

	// SystemNamespace is reserved for internal Konsul use.
	SystemNamespace = "system"
)

// Namespace represents an isolated partition for services and KV data.
type Namespace struct {
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var (
	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrNamespaceExists   = errors.New("namespace already exists")
	ErrNamespaceReserved = errors.New("namespace name is reserved and cannot be deleted")
	ErrInvalidNamespace  = errors.New("invalid namespace name")
)
