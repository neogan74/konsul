package store

import (
	"fmt"
	"regexp"
	"strings"
)

// Tag validation limits
const (
	MaxTagsPerService = 64
	MaxTagLength      = 255
)

// Metadata validation limits
const (
	MaxMetadataKeys        = 64
	MaxMetadataKeyLength   = 128
	MaxMetadataValueLength = 512
)

// Reserved metadata key prefixes (for internal use)
var ReservedMetadataKeyPrefixes = []string{
	"konsul_",
	"_",
}

// Tag format regex: allow alphanumeric, -, _, :, ., /
var tagFormatRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_:./]+$`)

// Metadata key format regex: allow alphanumeric, -, _
var metadataKeyFormatRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// ValidateTags validates service tags
func ValidateTags(tags []string) error {
	if len(tags) > MaxTagsPerService {
		return fmt.Errorf("too many tags: %d (max %d)", len(tags), MaxTagsPerService)
	}

	seen := make(map[string]bool)
	for _, tag := range tags {
		if len(tag) == 0 {
			return fmt.Errorf("empty tag not allowed")
		}

		if len(tag) > MaxTagLength {
			return fmt.Errorf("tag too long: %d chars (max %d)", len(tag), MaxTagLength)
		}

		if !tagFormatRegex.MatchString(tag) {
			return fmt.Errorf("invalid tag format: %s (allowed: alphanumeric, -, _, :, ., /)", tag)
		}

		if seen[tag] {
			return fmt.Errorf("duplicate tag: %s", tag)
		}
		seen[tag] = true
	}

	return nil
}

// ValidateMetadata validates service metadata
func ValidateMetadata(meta map[string]string) error {
	if len(meta) > MaxMetadataKeys {
		return fmt.Errorf("too many metadata keys: %d (max %d)", len(meta), MaxMetadataKeys)
	}

	for key, value := range meta {
		if len(key) == 0 {
			return fmt.Errorf("empty metadata key not allowed")
		}

		if len(key) > MaxMetadataKeyLength {
			return fmt.Errorf("metadata key too long: %s (%d chars, max %d)",
				key, len(key), MaxMetadataKeyLength)
		}

		if len(value) > MaxMetadataValueLength {
			return fmt.Errorf("metadata value too long for key %s: %d chars (max %d)",
				key, len(value), MaxMetadataValueLength)
		}

		if !metadataKeyFormatRegex.MatchString(key) {
			return fmt.Errorf("invalid metadata key format: %s (allowed: alphanumeric, -, _)", key)
		}

		if isReservedMetadataKey(key) {
			return fmt.Errorf("reserved metadata key: %s", key)
		}
	}

	return nil
}

// isReservedMetadataKey checks if a metadata key uses a reserved prefix
func isReservedMetadataKey(key string) bool {
	for _, prefix := range ReservedMetadataKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// ValidateService validates a complete service including tags and metadata
func ValidateService(service *Service) error {
	// Basic field validation
	if service.Name == "" {
		return fmt.Errorf("service name is required")
	}

	if service.Address == "" {
		return fmt.Errorf("service address is required")
	}

	if service.Port == 0 {
		return fmt.Errorf("service port is required")
	}

	if service.Port < 1 || service.Port > 65535 {
		return fmt.Errorf("service port must be between 1 and 65535")
	}

	// Tag validation
	if err := ValidateTags(service.Tags); err != nil {
		return fmt.Errorf("tag validation failed: %w", err)
	}

	// Metadata validation
	if err := ValidateMetadata(service.Meta); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	return nil
}
