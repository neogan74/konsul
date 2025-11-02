package model

import (
	"time"

	"github.com/neogan74/konsul/internal/graphql/scalar"
	"github.com/neogan74/konsul/internal/healthcheck"
	"github.com/neogan74/konsul/internal/store"
)

// MapKVPairFromStore converts store data to GraphQL KVPair model
func MapKVPairFromStore(key, value string) *KVPair {
	now := scalar.FromTime(time.Now())
	return &KVPair{
		Key:       key,
		Value:     value,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
}

// MapServiceFromStore converts store.Service to GraphQL Service model
func MapServiceFromStore(svc store.Service, entry store.ServiceEntry) *Service {
	status := ServiceStatusActive
	if entry.ExpiresAt.Before(time.Now()) {
		status = ServiceStatusExpired
	}

	expiresAt := scalar.FromTime(entry.ExpiresAt)

	// Convert tags (handle nil)
	tags := svc.Tags
	if tags == nil {
		tags = []string{}
	}

	// Convert metadata map to array of MetadataEntry
	metadata := make([]*MetadataEntry, 0, len(svc.Meta))
	for key, value := range svc.Meta {
		metadata = append(metadata, &MetadataEntry{
			Key:   key,
			Value: value,
		})
	}

	return &Service{
		Name:      svc.Name,
		Address:   svc.Address,
		Port:      svc.Port,
		Status:    status,
		ExpiresAt: expiresAt,
		Tags:      tags,
		Metadata:  metadata,
		Checks:    []*HealthCheck{}, // Will be populated by resolver
	}
}

// MapHealthCheckFromStore converts healthcheck.Check to GraphQL HealthCheck model
func MapHealthCheckFromStore(check *healthcheck.Check) *HealthCheck {
	checkType := mapCheckType(check.Type)
	checkStatus := mapCheckStatus(check.Status)

	var lastChecked *scalar.Time
	if !check.LastCheck.IsZero() {
		t := scalar.FromTime(check.LastCheck)
		lastChecked = &t
	}

	var interval, timeout *scalar.Duration
	if check.Interval > 0 {
		i := scalar.FromDuration(check.Interval)
		interval = &i
	}
	if check.Timeout > 0 {
		t := scalar.FromDuration(check.Timeout)
		timeout = &t
	}

	return &HealthCheck{
		ID:          check.ID,
		ServiceID:   check.ServiceID,
		Name:        check.Name,
		Type:        checkType,
		Status:      checkStatus,
		Output:      &check.Output,
		Interval:    interval,
		Timeout:     timeout,
		LastChecked: lastChecked,
	}
}

// mapCheckType converts healthcheck.CheckType to GraphQL HealthCheckType
func mapCheckType(checkType healthcheck.CheckType) HealthCheckType {
	switch checkType {
	case healthcheck.CheckTypeHTTP:
		return HealthCheckTypeHTTP
	case healthcheck.CheckTypeTCP:
		return HealthCheckTypeTCP
	case healthcheck.CheckTypeGRPC:
		return HealthCheckTypeGrpc
	case healthcheck.CheckTypeTTL:
		return HealthCheckTypeTTL
	default:
		return HealthCheckTypeHTTP // default
	}
}

// mapCheckStatus converts healthcheck.Status to GraphQL HealthCheckStatus
func mapCheckStatus(status healthcheck.Status) HealthCheckStatus {
	switch status {
	case healthcheck.StatusPassing:
		return HealthCheckStatusPassing
	case healthcheck.StatusWarning:
		return HealthCheckStatusWarning
	case healthcheck.StatusCritical:
		return HealthCheckStatusCritical
	default:
		return HealthCheckStatusCritical // default to critical for safety
	}
}
