// Package store provides a key-value store implementation
package store

import (
	"fmt"
)

// CASConflictError represents a Compare-And-Swap conflict
// when the expected ModifyIndex doesn't match the current value
type CASConflictError struct {
	Key           string
	ExpectedIndex uint64
	CurrentIndex  uint64
	OperationType string
}

func (e *CASConflictError) Error() string {
	return fmt.Sprintf("CAS conflict for %s '%s': expected ModifyIndex %d, but current is %d",
		e.OperationType, e.Key, e.ExpectedIndex, e.CurrentIndex)
}

// IsCASConflict checks if an error is a CAS conflict error
func IsCASConflict(err error) bool {
	_, ok := err.(*CASConflictError)
	return ok
}

// NotFoundError represents an entity not found error
type NotFoundError struct {
	Type string
	Key  string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s '%s' not found", e.Type, e.Key)
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// IndexMismatchError represents a mismatch between expected and actual indices
type IndexMismatchError struct {
	Message string
}

func (e *IndexMismatchError) Error() string {
	return e.Message
}
