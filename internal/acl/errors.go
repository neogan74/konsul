package acl

import "errors"

var (
	// ErrPermissionDenied is returned when access is denied
	ErrPermissionDenied = errors.New("permission denied")

	// ErrInvalidPolicy is returned when a policy is invalid
	ErrInvalidPolicy = errors.New("invalid policy")

	// ErrPolicyNotFound is returned when a policy is not found
	ErrPolicyNotFound = errors.New("policy not found")

	// ErrPolicyExists is returned when a policy already exists
	ErrPolicyExists = errors.New("policy already exists")

	// ErrNoPolicies is returned when no policies are attached to a token
	ErrNoPolicies = errors.New("no policies attached")
)
