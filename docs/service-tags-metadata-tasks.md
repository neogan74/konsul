# Service Tags and Metadata - Implementation Tasks

**Based on**: ADR-0017 (Service Tags and Metadata)
**Created**: 2025-10-28
**Status**: Planning

This document breaks down the Service Tags and Metadata implementation into actionable tasks with clear acceptance criteria, dependencies, and time estimates.

---

## Phase 1: Core Data Model (Week 1)

### 1.1 Data Structure Updates

#### Task 1.1.1: Update Service Struct
**Priority**: P0 (Critical Path)
**Estimated Time**: 2 hours
**Dependencies**: None

**Description**:
Add `Tags` and `Meta` fields to the `Service` struct.

**Acceptance Criteria**:
- [ ] Add `Tags []string` field to `Service` struct
- [ ] Add `Meta map[string]string` field to `Service` struct
- [ ] Add JSON tags with `omitempty`
- [ ] Update struct documentation
- [ ] Ensure backward compatibility (existing services without tags/meta work)

**File**: `internal/store/service.go`

**Code Changes**:
```go
type Service struct {
    Name    string                        `json:"name"`
    Address string                        `json:"address"`
    Port    int                           `json:"port"`
    Tags    []string                      `json:"tags,omitempty"`     // NEW
    Meta    map[string]string             `json:"meta,omitempty"`     // NEW
    Checks  []*healthcheck.CheckDefinition `json:"checks,omitempty"`
}
```

---

#### Task 1.1.2: Add Validation Constants
**Priority**: P0
**Estimated Time**: 1 hour
**Dependencies**: 1.1.1

**Description**:
Define validation constants for tags and metadata limits.

**Acceptance Criteria**:
- [ ] Add constants to a new validation file
- [ ] MaxTagsPerService = 64
- [ ] MaxTagLength = 255
- [ ] MaxMetadataKeys = 64
- [ ] MaxMetadataKeyLength = 128
- [ ] MaxMetadataValueLength = 512
- [ ] Document each constant

**File**: `internal/store/validation.go` (new)

**Code**:
```go
package store

const (
    // Tag validation limits
    MaxTagsPerService = 64
    MaxTagLength      = 255

    // Metadata validation limits
    MaxMetadataKeys         = 64
    MaxMetadataKeyLength    = 128
    MaxMetadataValueLength  = 512
)

// Reserved metadata key prefixes
var ReservedMetadataKeyPrefixes = []string{
    "konsul_",
    "_",
}
```

---

#### Task 1.1.3: Implement Service Validation
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.1.2

**Description**:
Create validation functions for tags and metadata.

**Acceptance Criteria**:
- [ ] Implement `ValidateTags(tags []string) error`
- [ ] Implement `ValidateMetadata(meta map[string]string) error`
- [ ] Implement `ValidateService(service *Service) error`
- [ ] Check tag count limit
- [ ] Check tag length limit
- [ ] Check tag format (alphanumeric, `-`, `_`, `:`, `.`, `/`)
- [ ] Check for duplicate tags
- [ ] Check metadata key count limit
- [ ] Check metadata key/value length limits
- [ ] Check metadata key format
- [ ] Check for reserved metadata keys
- [ ] Return descriptive error messages
- [ ] Write unit tests

**File**: `internal/store/validation.go`

**Example**:
```go
func ValidateTags(tags []string) error {
    if len(tags) > MaxTagsPerService {
        return fmt.Errorf("too many tags: %d (max %d)", len(tags), MaxTagsPerService)
    }

    seen := make(map[string]bool)
    for _, tag := range tags {
        if len(tag) > MaxTagLength {
            return fmt.Errorf("tag too long: %d chars (max %d)", len(tag), MaxTagLength)
        }

        if !isValidTagFormat(tag) {
            return fmt.Errorf("invalid tag format: %s", tag)
        }

        if seen[tag] {
            return fmt.Errorf("duplicate tag: %s", tag)
        }
        seen[tag] = true
    }

    return nil
}

func isValidTagFormat(tag string) bool {
    // Allow alphanumeric, -, _, :, ., /
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_:./]+$`, tag)
    return matched
}

func ValidateMetadata(meta map[string]string) error {
    if len(meta) > MaxMetadataKeys {
        return fmt.Errorf("too many metadata keys: %d (max %d)", len(meta), MaxMetadataKeys)
    }

    for key, value := range meta {
        if len(key) > MaxMetadataKeyLength {
            return fmt.Errorf("metadata key too long: %s (%d chars, max %d)",
                key, len(key), MaxMetadataKeyLength)
        }

        if len(value) > MaxMetadataValueLength {
            return fmt.Errorf("metadata value too long for key %s: %d chars (max %d)",
                key, len(value), MaxMetadataValueLength)
        }

        if !isValidMetadataKey(key) {
            return fmt.Errorf("invalid metadata key format: %s", key)
        }

        if isReservedMetadataKey(key) {
            return fmt.Errorf("reserved metadata key: %s", key)
        }
    }

    return nil
}

func ValidateService(service *Service) error {
    // Existing validation
    if service.Name == "" || service.Address == "" || service.Port == 0 {
        return fmt.Errorf("missing required fields")
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
```

---

#### Task 1.1.4: Write Validation Unit Tests
**Priority**: P0
**Estimated Time**: 3 hours
**Dependencies**: 1.1.3

**Description**:
Comprehensive unit tests for validation logic.

**Acceptance Criteria**:
- [ ] Test valid tags
- [ ] Test invalid tag format
- [ ] Test tag count limit
- [ ] Test tag length limit
- [ ] Test duplicate tags
- [ ] Test valid metadata
- [ ] Test invalid metadata key format
- [ ] Test metadata count limit
- [ ] Test metadata key/value length limits
- [ ] Test reserved metadata keys
- [ ] Test ValidateService integration
- [ ] Achieve >90% code coverage

**File**: `internal/store/validation_test.go`

---

### 1.2 Indexing Infrastructure

#### Task 1.2.1: Add Tag and Metadata Indexes
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.1.1

**Description**:
Add indexing structures to ServiceStore for fast tag/metadata queries.

**Acceptance Criteria**:
- [ ] Add `TagIndex map[string]map[string]bool` to ServiceStore
- [ ] Add `MetaIndex map[string]map[string][]string` to ServiceStore
- [ ] Document index structure
- [ ] Initialize indexes in constructors

**File**: `internal/store/service.go`

**Code Changes**:
```go
type ServiceStore struct {
    Data          map[string]ServiceEntry
    TagIndex      map[string]map[string]bool    // Tag → {ServiceName: true}
    MetaIndex     map[string]map[string][]string // MetaKey → {MetaValue: [ServiceNames]}
    Mutex         sync.RWMutex
    TTL           time.Duration
    engine        persistence.Engine
    log           logger.Logger
    healthManager *healthcheck.Manager
}

func NewServiceStore() *ServiceStore {
    return &ServiceStore{
        Data:          make(map[string]ServiceEntry),
        TagIndex:      make(map[string]map[string]bool),
        MetaIndex:     make(map[string]map[string][]string),
        TTL:           30 * time.Second,
        log:           logger.GetDefault(),
        healthManager: healthcheck.NewManager(logger.GetDefault()),
    }
}
```

---

#### Task 1.2.2: Implement Index Update Functions
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.2.1

**Description**:
Create functions to maintain tag and metadata indexes.

**Acceptance Criteria**:
- [ ] Implement `addToTagIndex(serviceName string, tags []string)`
- [ ] Implement `removeFromTagIndex(serviceName string, tags []string)`
- [ ] Implement `addToMetaIndex(serviceName string, meta map[string]string)`
- [ ] Implement `removeFromMetaIndex(serviceName string, meta map[string]string)`
- [ ] Handle edge cases (empty tags/meta)
- [ ] Thread-safe operations
- [ ] Write unit tests

**File**: `internal/store/service_index.go` (new)

**Example**:
```go
func (s *ServiceStore) addToTagIndex(serviceName string, tags []string) {
    for _, tag := range tags {
        if s.TagIndex[tag] == nil {
            s.TagIndex[tag] = make(map[string]bool)
        }
        s.TagIndex[tag][serviceName] = true
    }
}

func (s *ServiceStore) removeFromTagIndex(serviceName string, tags []string) {
    for _, tag := range tags {
        if services, ok := s.TagIndex[tag]; ok {
            delete(services, serviceName)
            if len(services) == 0 {
                delete(s.TagIndex, tag)
            }
        }
    }
}

func (s *ServiceStore) addToMetaIndex(serviceName string, meta map[string]string) {
    for key, value := range meta {
        if s.MetaIndex[key] == nil {
            s.MetaIndex[key] = make(map[string][]string)
        }
        s.MetaIndex[key][value] = append(s.MetaIndex[key][value], serviceName)
    }
}

func (s *ServiceStore) removeFromMetaIndex(serviceName string, meta map[string]string) {
    for key, value := range meta {
        if values, ok := s.MetaIndex[key]; ok {
            if services, ok := values[value]; ok {
                // Remove serviceName from slice
                for i, name := range services {
                    if name == serviceName {
                        s.MetaIndex[key][value] = append(services[:i], services[i+1:]...)
                        break
                    }
                }
                // Cleanup empty entries
                if len(s.MetaIndex[key][value]) == 0 {
                    delete(s.MetaIndex[key], value)
                }
            }
            if len(s.MetaIndex[key]) == 0 {
                delete(s.MetaIndex, key)
            }
        }
    }
}
```

---

#### Task 1.2.3: Update Register Method
**Priority**: P0
**Estimated Time**: 3 hours
**Dependencies**: 1.2.2, 1.1.3

**Description**:
Update the Register method to validate and index tags/metadata.

**Acceptance Criteria**:
- [ ] Call ValidateService before registration
- [ ] Update indexes on registration
- [ ] Handle service update (remove old indexes, add new)
- [ ] Persist tags/metadata to storage
- [ ] Add logging for tag/metadata operations
- [ ] Update unit tests

**File**: `internal/store/service.go`

**Code Changes**:
```go
func (s *ServiceStore) Register(service Service) error {
    // Validate service including tags and metadata
    if err := ValidateService(&service); err != nil {
        s.log.Error("Service validation failed",
            logger.String("service", service.Name),
            logger.Error(err))
        return err
    }

    s.Mutex.Lock()
    defer s.Mutex.Unlock()

    // Remove old indexes if service exists
    if oldEntry, exists := s.Data[service.Name]; exists {
        s.removeFromTagIndex(service.Name, oldEntry.Service.Tags)
        s.removeFromMetaIndex(service.Name, oldEntry.Service.Meta)
    }

    entry := ServiceEntry{
        Service:   service,
        ExpiresAt: time.Now().Add(s.TTL),
    }
    s.Data[service.Name] = entry

    // Add to indexes
    s.addToTagIndex(service.Name, service.Tags)
    s.addToMetaIndex(service.Name, service.Meta)

    // Register health checks
    for _, checkDef := range service.Checks {
        if checkDef.ServiceID == "" {
            checkDef.ServiceID = service.Name
        }
        if checkDef.Name == "" {
            checkDef.Name = fmt.Sprintf("%s-health", service.Name)
        }
        _, err := s.healthManager.AddCheck(checkDef)
        if err != nil {
            s.log.Error("Failed to add health check",
                logger.String("service", service.Name),
                logger.String("check", checkDef.Name),
                logger.Error(err))
        }
    }

    // Persist to storage
    if s.engine != nil {
        data, err := json.Marshal(entry)
        if err != nil {
            s.log.Error("Failed to marshal service entry",
                logger.String("service", service.Name),
                logger.Error(err))
            return err
        }
        if err := s.engine.SetService(service.Name, data, s.TTL); err != nil {
            s.log.Error("Failed to persist service",
                logger.String("service", service.Name),
                logger.Error(err))
            return err
        }
    }

    s.log.Info("Service registered with tags/metadata",
        logger.String("service", service.Name),
        logger.Int("tags", len(service.Tags)),
        logger.Int("metadata_keys", len(service.Meta)))

    return nil
}
```

---

#### Task 1.2.4: Update Deregister Method
**Priority**: P0
**Estimated Time**: 2 hours
**Dependencies**: 1.2.2

**Description**:
Update Deregister to remove from indexes.

**Acceptance Criteria**:
- [ ] Remove service from tag index
- [ ] Remove service from metadata index
- [ ] Update unit tests

**File**: `internal/store/service.go`

**Code Changes**:
```go
func (s *ServiceStore) Deregister(name string) {
    s.Mutex.Lock()
    defer s.Mutex.Unlock()

    // Remove from indexes
    if entry, exists := s.Data[name]; exists {
        s.removeFromTagIndex(name, entry.Service.Tags)
        s.removeFromMetaIndex(name, entry.Service.Meta)
    }

    delete(s.Data, name)

    // Delete from persistence
    if s.engine != nil {
        if err := s.engine.DeleteService(name); err != nil {
            s.log.Error("Failed to delete service from persistence",
                logger.String("service", name),
                logger.Error(err))
        }
    }
}
```

---

#### Task 1.2.5: Update CleanupExpired Method
**Priority**: P0
**Estimated Time**: 2 hours
**Dependencies**: 1.2.2

**Description**:
Update CleanupExpired to remove expired services from indexes.

**Acceptance Criteria**:
- [ ] Remove expired services from tag index
- [ ] Remove expired services from metadata index
- [ ] Update unit tests

**File**: `internal/store/service.go`

---

### 1.3 Query Functions

#### Task 1.3.1: Implement Query by Tags
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.2.2

**Description**:
Create function to query services by tags.

**Acceptance Criteria**:
- [ ] Implement `QueryByTags(tags []string) []Service`
- [ ] Support multiple tags (AND logic)
- [ ] Use tag index for efficient lookup
- [ ] Filter out expired services
- [ ] Return sorted results
- [ ] Write unit tests with various scenarios

**File**: `internal/store/service_query.go` (new)

**Example**:
```go
// QueryByTags returns services that have ALL specified tags (AND logic)
func (s *ServiceStore) QueryByTags(tags []string) []Service {
    s.Mutex.RLock()
    defer s.Mutex.RUnlock()

    if len(tags) == 0 {
        return s.List()
    }

    // Get services with first tag
    var candidateServices map[string]bool
    if services, ok := s.TagIndex[tags[0]]; ok {
        candidateServices = make(map[string]bool, len(services))
        for name := range services {
            candidateServices[name] = true
        }
    } else {
        return []Service{} // No services with first tag
    }

    // Intersect with other tags (AND logic)
    for _, tag := range tags[1:] {
        if services, ok := s.TagIndex[tag]; ok {
            for name := range candidateServices {
                if !services[name] {
                    delete(candidateServices, name)
                }
            }
        } else {
            return []Service{} // No services with this tag
        }
    }

    // Build result list (filter expired)
    now := time.Now()
    result := make([]Service, 0, len(candidateServices))
    for name := range candidateServices {
        if entry, ok := s.Data[name]; ok && entry.ExpiresAt.After(now) {
            result = append(result, entry.Service)
        }
    }

    return result
}
```

---

#### Task 1.3.2: Implement Query by Metadata
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.2.2

**Description**:
Create function to query services by metadata.

**Acceptance Criteria**:
- [ ] Implement `QueryByMetadata(filters map[string]string) []Service`
- [ ] Support multiple metadata filters (AND logic)
- [ ] Use metadata index for efficient lookup
- [ ] Filter out expired services
- [ ] Return sorted results
- [ ] Write unit tests

**File**: `internal/store/service_query.go`

**Example**:
```go
// QueryByMetadata returns services that match ALL specified metadata filters (AND logic)
func (s *ServiceStore) QueryByMetadata(filters map[string]string) []Service {
    s.Mutex.RLock()
    defer s.Mutex.RUnlock()

    if len(filters) == 0 {
        return s.List()
    }

    // Get candidate services from first filter
    var candidateServices map[string]bool
    firstKey := ""
    var firstValue string
    for k, v := range filters {
        firstKey = k
        firstValue = v
        break
    }

    if values, ok := s.MetaIndex[firstKey]; ok {
        if services, ok := values[firstValue]; ok {
            candidateServices = make(map[string]bool, len(services))
            for _, name := range services {
                candidateServices[name] = true
            }
        } else {
            return []Service{} // No services with this metadata value
        }
    } else {
        return []Service{} // No services with this metadata key
    }

    // Intersect with other filters (AND logic)
    for key, value := range filters {
        if key == firstKey {
            continue
        }

        if values, ok := s.MetaIndex[key]; ok {
            if services, ok := values[value]; ok {
                serviceSet := make(map[string]bool)
                for _, name := range services {
                    serviceSet[name] = true
                }

                for name := range candidateServices {
                    if !serviceSet[name] {
                        delete(candidateServices, name)
                    }
                }
            } else {
                return []Service{} // No services with this metadata value
            }
        } else {
            return []Service{} // No services with this metadata key
        }
    }

    // Build result list (filter expired)
    now := time.Now()
    result := make([]Service, 0, len(candidateServices))
    for name := range candidateServices {
        if entry, ok := s.Data[name]; ok && entry.ExpiresAt.After(now) {
            result = append(result, entry.Service)
        }
    }

    return result
}
```

---

#### Task 1.3.3: Implement Combined Query
**Priority**: P0
**Estimated Time**: 3 hours
**Dependencies**: 1.3.1, 1.3.2

**Description**:
Create function to query by both tags and metadata.

**Acceptance Criteria**:
- [ ] Implement `QueryByTagsAndMetadata(tags []string, meta map[string]string) []Service`
- [ ] Intersect results from tag and metadata queries
- [ ] Efficient set intersection
- [ ] Write unit tests

**File**: `internal/store/service_query.go`

**Example**:
```go
// QueryByTagsAndMetadata returns services that match both tag and metadata filters
func (s *ServiceStore) QueryByTagsAndMetadata(tags []string, meta map[string]string) []Service {
    // Get services matching tags
    tagServices := s.QueryByTags(tags)
    if len(tagServices) == 0 {
        return []Service{}
    }

    // Get services matching metadata
    metaServices := s.QueryByMetadata(meta)
    if len(metaServices) == 0 {
        return []Service{}
    }

    // Intersect results
    tagServiceSet := make(map[string]bool)
    for _, svc := range tagServices {
        tagServiceSet[svc.Name] = true
    }

    result := make([]Service, 0)
    for _, svc := range metaServices {
        if tagServiceSet[svc.Name] {
            result = append(result, svc)
        }
    }

    return result
}
```

---

#### Task 1.3.4: Write Query Unit Tests
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 1.3.3

**Description**:
Comprehensive tests for query functions.

**Acceptance Criteria**:
- [ ] Test QueryByTags with single tag
- [ ] Test QueryByTags with multiple tags (AND)
- [ ] Test QueryByTags with no matching services
- [ ] Test QueryByMetadata with single filter
- [ ] Test QueryByMetadata with multiple filters (AND)
- [ ] Test QueryByMetadata with no matching services
- [ ] Test QueryByTagsAndMetadata
- [ ] Test with expired services (should be filtered)
- [ ] Test with empty tags/metadata
- [ ] Achieve >90% code coverage

**File**: `internal/store/service_query_test.go`

---

## Phase 2: API Endpoints (Week 1-2)

### 2.1 Update Service Handler

#### Task 2.1.1: Update Register Endpoint
**Priority**: P0
**Estimated Time**: 3 hours
**Dependencies**: Phase 1 complete

**Description**:
Update service registration handler to accept tags and metadata.

**Acceptance Criteria**:
- [ ] Parse tags from request body
- [ ] Parse metadata from request body
- [ ] Call validation before registration
- [ ] Return validation errors with 400 status
- [ ] Log tags/metadata in registration
- [ ] Update Prometheus metrics (add tag count label)
- [ ] Update handler tests

**File**: `internal/handlers/service.go`

**Code Changes**:
```go
func (h *ServiceHandler) Register(c *fiber.Ctx) error {
    log := middleware.GetLogger(c)
    var svc store.Service

    if err := c.BodyParser(&svc); err != nil {
        log.Error("Failed to parse service registration body", logger.Error(err))
        return middleware.BadRequest(c, "Invalid JSON body")
    }

    // Validation now includes tags and metadata
    if err := store.ValidateService(&svc); err != nil {
        log.Error("Service validation failed",
            logger.String("service", svc.Name),
            logger.Error(err))
        return middleware.BadRequest(c, err.Error())
    }

    log.Info("Registering service",
        logger.String("service_name", svc.Name),
        logger.String("address", svc.Address),
        logger.Int("port", svc.Port),
        logger.Int("tags", len(svc.Tags)),
        logger.Int("metadata_keys", len(svc.Meta)))

    if err := h.store.Register(svc); err != nil {
        log.Error("Failed to register service",
            logger.String("service", svc.Name),
            logger.Error(err))
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    log.Info("Service registered successfully",
        logger.String("service_name", svc.Name))

    metrics.ServiceOperationsTotal.WithLabelValues("register", "success").Inc()
    metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))

    return c.JSON(fiber.Map{"message": "service registered", "service": svc})
}
```

---

#### Task 2.1.2: Update Get Endpoint
**Priority**: P1
**Estimated Time**: 1 hour
**Dependencies**: 2.1.1

**Description**:
Ensure Get endpoint returns tags and metadata.

**Acceptance Criteria**:
- [ ] Verify tags are included in response
- [ ] Verify metadata is included in response
- [ ] Update tests

**File**: `internal/handlers/service.go`

---

#### Task 2.1.3: Update List Endpoint
**Priority**: P1
**Estimated Time**: 1 hour
**Dependencies**: 2.1.1

**Description**:
Ensure List endpoint returns tags and metadata.

**Acceptance Criteria**:
- [ ] Verify tags are included in response
- [ ] Verify metadata is included in response
- [ ] Update tests

**File**: `internal/handlers/service.go`

---

### 2.2 Create Catalog Handler

#### Task 2.2.1: Create Catalog Handler Structure
**Priority**: P0
**Estimated Time**: 2 hours
**Dependencies**: 2.1.1

**Description**:
Create a new handler for catalog/query operations.

**Acceptance Criteria**:
- [ ] Create `internal/handlers/catalog.go`
- [ ] Add `CatalogHandler` struct with ServiceStore dependency
- [ ] Add constructor `NewCatalogHandler()`
- [ ] Add logging

**File**: `internal/handlers/catalog.go` (new)

**Example**:
```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/neogan74/konsul/internal/logger"
    "github.com/neogan74/konsul/internal/middleware"
    "github.com/neogan74/konsul/internal/store"
    "strings"
)

type CatalogHandler struct {
    store *store.ServiceStore
}

func NewCatalogHandler(serviceStore *store.ServiceStore) *CatalogHandler {
    return &CatalogHandler{store: serviceStore}
}
```

---

#### Task 2.2.2: Implement Query by Tags Endpoint
**Priority**: P0
**Estimated Time**: 4 hours
**Dependencies**: 2.2.1

**Description**:
Implement `GET /v1/catalog/services?tag=<tag>` endpoint.

**Acceptance Criteria**:
- [ ] Parse multiple `tag` query parameters
- [ ] Call `QueryByTags()` from store
- [ ] Return JSON array of services
- [ ] Return empty array if no matches
- [ ] Add logging
- [ ] Add Prometheus metrics (query type, duration)
- [ ] Add ACL protection
- [ ] Write handler tests

**File**: `internal/handlers/catalog.go`

**Example**:
```go
func (h *CatalogHandler) QueryServices(c *fiber.Ctx) error {
    log := middleware.GetLogger(c)

    // Parse tag filters
    tagParams := c.Queries()["tag"]
    var tags []string
    if tagParams != nil {
        tags = tagParams.([]string)
    }

    // Parse metadata filters
    metaParams := c.Queries()["meta"]
    meta := make(map[string]string)
    if metaParams != nil {
        for _, metaFilter := range metaParams.([]string) {
            parts := strings.SplitN(metaFilter, ":", 2)
            if len(parts) == 2 {
                meta[parts[0]] = parts[1]
            }
        }
    }

    log.Debug("Querying services",
        logger.Int("tags", len(tags)),
        logger.Int("meta_filters", len(meta)))

    var services []store.Service

    // Determine query type
    if len(tags) > 0 && len(meta) > 0 {
        services = h.store.QueryByTagsAndMetadata(tags, meta)
    } else if len(tags) > 0 {
        services = h.store.QueryByTags(tags)
    } else if len(meta) > 0 {
        services = h.store.QueryByMetadata(meta)
    } else {
        services = h.store.List()
    }

    log.Info("Services queried",
        logger.Int("result_count", len(services)))

    return c.JSON(services)
}
```

---

#### Task 2.2.3: Add Catalog Routes
**Priority**: P0
**Estimated Time**: 2 hours
**Dependencies**: 2.2.2

**Description**:
Add catalog routes to main application.

**Acceptance Criteria**:
- [ ] Create catalog handler instance
- [ ] Add `GET /v1/catalog/services` route
- [ ] Apply ACL middleware if enabled
- [ ] Apply metrics middleware
- [ ] Apply logging middleware
- [ ] Update integration tests

**File**: `cmd/konsul/main.go`

**Example**:
```go
// Catalog handler
catalogHandler := handlers.NewCatalogHandler(serviceStore)

// Catalog endpoints
if cfg.ACL.Enabled {
    app.Get("/v1/catalog/services",
        middleware.ACLMiddleware(aclEvaluator, acl.ResourceTypeService, acl.CapabilityRead),
        catalogHandler.QueryServices)
} else {
    app.Get("/v1/catalog/services", catalogHandler.QueryServices)
}
```

---

### 2.3 Metrics

#### Task 2.3.1: Add Tag/Metadata Metrics
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 2.2.2

**Description**:
Add Prometheus metrics for tag and metadata queries.

**Acceptance Criteria**:
- [ ] Add `konsul_catalog_queries_total{filter_type}` counter
- [ ] Add `konsul_catalog_query_duration_seconds{filter_type}` histogram
- [ ] Add `konsul_service_tags_total{service}` gauge
- [ ] Add `konsul_service_metadata_keys_total{service}` gauge
- [ ] Update metrics on registration
- [ ] Update metrics on query
- [ ] Document metrics

**File**: `internal/metrics/metrics.go`

**Example**:
```go
var (
    CatalogQueriesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "konsul_catalog_queries_total",
            Help: "Total number of catalog queries",
        },
        []string{"filter_type"}, // tag, meta, combined, none
    )

    CatalogQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "konsul_catalog_query_duration_seconds",
            Help: "Catalog query duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"filter_type"},
    )

    ServiceTagsTotal = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "konsul_service_tags_total",
            Help: "Number of tags per service",
        },
        []string{"service"},
    )

    ServiceMetadataKeysTotal = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "konsul_service_metadata_keys_total",
            Help: "Number of metadata keys per service",
        },
        []string{"service"},
    )
)
```

---

## Phase 3: DNS Integration (Week 2)

### 3.1 DNS Tag Queries

#### Task 3.1.1: Update DNS Handler for Tag Queries
**Priority**: P1
**Estimated Time**: 6 hours
**Dependencies**: Phase 1 complete

**Description**:
Support DNS queries with tags: `<tag>.<service>.service.konsul`

**Acceptance Criteria**:
- [ ] Parse tag from DNS query name
- [ ] Support format: `<tag>.<service>.service.konsul`
- [ ] Query services by tag using store
- [ ] Return A/AAAA records for matching services
- [ ] Support multiple tags (comma-separated or multiple labels)
- [ ] Add logging for tag-based DNS queries
- [ ] Write integration tests

**File**: `internal/dns/handler.go`

**Example**:
```go
// Parse service query with optional tag
// Format: [tag.]service.service.konsul
func parseServiceQuery(name string) (service string, tag string, ok bool) {
    parts := strings.Split(strings.TrimSuffix(name, "."), ".")

    if len(parts) < 3 {
        return "", "", false
    }

    if parts[len(parts)-1] != "konsul" || parts[len(parts)-2] != "service" {
        return "", "", false
    }

    if len(parts) == 3 {
        // Simple service query: service.service.konsul
        return parts[0], "", true
    } else if len(parts) == 4 {
        // Tag query: tag.service.service.konsul
        return parts[1], parts[0], true
    }

    return "", "", false
}

func (h *DNSHandler) handleServiceQuery(w dns.ResponseWriter, r *dns.Msg) {
    name := r.Question[0].Name
    service, tag, ok := parseServiceQuery(name)

    if !ok {
        h.log.Warn("Invalid service query format", logger.String("name", name))
        m := new(dns.Msg)
        m.SetRcode(r, dns.RcodeNameError)
        w.WriteMsg(m)
        return
    }

    var services []store.Service
    if tag != "" {
        // Tag-based query
        h.log.Debug("DNS tag query",
            logger.String("service", service),
            logger.String("tag", tag))
        services = h.store.QueryByTags([]string{tag})
        // Filter by service name
        filtered := make([]store.Service, 0)
        for _, svc := range services {
            if svc.Name == service {
                filtered = append(filtered, svc)
            }
        }
        services = filtered
    } else {
        // Regular service query
        if svc, ok := h.store.Get(service); ok {
            services = []store.Service{svc}
        }
    }

    // Build DNS response with A records
    // ... existing DNS response logic
}
```

---

#### Task 3.1.2: Write DNS Integration Tests
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 3.1.1

**Description**:
Test DNS queries with tags.

**Acceptance Criteria**:
- [ ] Test simple service query
- [ ] Test tag-based service query
- [ ] Test tag query with no matches
- [ ] Test tag query with multiple matches
- [ ] Test invalid query format

**File**: `internal/dns/handler_test.go`

---

## Phase 4: CLI Support (Week 2-3)

### 4.1 konsulctl Updates

#### Task 4.1.1: Update Service Register Command
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: Phase 2 complete

**Description**:
Add flags for tags and metadata to `konsulctl service register`.

**Acceptance Criteria**:
- [ ] Add `--tag` flag (repeatable)
- [ ] Add `--meta` flag (key=value format, repeatable)
- [ ] Parse tags and metadata from flags
- [ ] Include in registration request
- [ ] Add examples to help text
- [ ] Update command tests

**File**: `cmd/konsulctl/service.go`

**Example**:
```go
var serviceRegisterCmd = &cobra.Command{
    Use:   "register <name> --address <addr> --port <port>",
    Short: "Register a service",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]
        address, _ := cmd.Flags().GetString("address")
        port, _ := cmd.Flags().GetInt("port")
        tags, _ := cmd.Flags().GetStringSlice("tag")
        metaFlags, _ := cmd.Flags().GetStringSlice("meta")

        // Parse metadata key=value pairs
        meta := make(map[string]string)
        for _, metaFlag := range metaFlags {
            parts := strings.SplitN(metaFlag, "=", 2)
            if len(parts) == 2 {
                meta[parts[0]] = parts[1]
            } else {
                return fmt.Errorf("invalid metadata format: %s (expected key=value)", metaFlag)
            }
        }

        service := map[string]interface{}{
            "name":    name,
            "address": address,
            "port":    port,
        }

        if len(tags) > 0 {
            service["tags"] = tags
        }

        if len(meta) > 0 {
            service["meta"] = meta
        }

        // Send registration request
        // ... existing logic
    },
}

func init() {
    serviceRegisterCmd.Flags().String("address", "", "Service address")
    serviceRegisterCmd.Flags().Int("port", 0, "Service port")
    serviceRegisterCmd.Flags().StringSlice("tag", []string{}, "Service tag (can be repeated)")
    serviceRegisterCmd.Flags().StringSlice("meta", []string{}, "Service metadata key=value (can be repeated)")
    serviceRegisterCmd.MarkFlagRequired("address")
    serviceRegisterCmd.MarkFlagRequired("port")
}
```

---

#### Task 4.1.2: Add Service List Filters
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: 4.1.1

**Description**:
Add filtering options to `konsulctl service list`.

**Acceptance Criteria**:
- [ ] Add `--tag` flag for filtering
- [ ] Add `--meta` flag for filtering
- [ ] Support multiple filters
- [ ] Call catalog API with filters
- [ ] Format output nicely (table with tags/meta columns)
- [ ] Add color coding
- [ ] Update command tests

**File**: `cmd/konsulctl/service.go`

**Example**:
```go
var serviceListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all services",
    RunE: func(cmd *cobra.Command, args []string) error {
        tags, _ := cmd.Flags().GetStringSlice("tag")
        metaFlags, _ := cmd.Flags().GetStringSlice("meta")

        // Build query parameters
        params := url.Values{}
        for _, tag := range tags {
            params.Add("tag", tag)
        }
        for _, metaFlag := range metaFlags {
            params.Add("meta", metaFlag)
        }

        // Call catalog API
        endpoint := "/v1/catalog/services"
        if len(params) > 0 {
            endpoint += "?" + params.Encode()
        }

        // ... fetch and display results in table format
    },
}

func init() {
    serviceListCmd.Flags().StringSlice("tag", []string{}, "Filter by tag")
    serviceListCmd.Flags().StringSlice("meta", []string{}, "Filter by metadata key:value")
}
```

---

#### Task 4.1.3: Update Service Display Format
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 4.1.2

**Description**:
Improve table output to show tags and metadata.

**Acceptance Criteria**:
- [ ] Add Tags column to table
- [ ] Add Metadata column (abbreviated)
- [ ] Use color for tags (green)
- [ ] Truncate long values
- [ ] Add `--verbose` flag for full metadata
- [ ] Test output formatting

**File**: `cmd/konsulctl/service.go`

---

## Phase 5: Admin UI (Week 3)

### 5.1 UI Components

#### Task 5.1.1: Update Service Registration Form
**Priority**: P1
**Estimated Time**: 4 hours
**Dependencies**: Phase 2 complete

**Description**:
Add tag and metadata inputs to service registration form.

**Acceptance Criteria**:
- [ ] Add multi-select tag input or tag chips
- [ ] Add dynamic key-value input for metadata
- [ ] Add validation for tag format
- [ ] Add validation for metadata keys/values
- [ ] Show character count for limits
- [ ] Update form submission to include tags/meta
- [ ] Update tests

**File**: `ui/src/components/ServiceForm.tsx` (or similar)

---

#### Task 5.1.2: Update Service List Display
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 5.1.1

**Description**:
Display tags and metadata in service list.

**Acceptance Criteria**:
- [ ] Show tags as colored badges/chips
- [ ] Show metadata count (expandable)
- [ ] Add tooltip for full metadata on hover
- [ ] Use consistent styling

**File**: `ui/src/components/ServiceList.tsx`

---

#### Task 5.1.3: Add Service Filters
**Priority**: P1
**Estimated Time**: 5 hours
**Dependencies**: 5.1.2

**Description**:
Add filter UI for tags and metadata.

**Acceptance Criteria**:
- [ ] Add tag filter dropdown
- [ ] Add metadata filter input
- [ ] Support multiple filters
- [ ] Real-time filtering
- [ ] Show active filters with clear option
- [ ] Update query string in URL
- [ ] Update tests

**File**: `ui/src/components/ServiceFilters.tsx` (new)

---

## Phase 6: Documentation (Week 3-4)

### 6.1 User Documentation

#### Task 6.1.1: Create Service Tags and Metadata Guide
**Priority**: P1
**Estimated Time**: 6 hours
**Dependencies**: Phases 1-5 complete

**Description**:
Comprehensive guide for using tags and metadata.

**Acceptance Criteria**:
- [ ] Create `docs/service-tags-metadata.md`
- [ ] Section: Introduction and concepts
- [ ] Section: Tag conventions and best practices
- [ ] Section: Metadata conventions
- [ ] Section: API examples (curl)
- [ ] Section: CLI examples (konsulctl)
- [ ] Section: DNS query examples
- [ ] Section: Common use cases (canary, multi-env, etc.)
- [ ] Section: Performance considerations
- [ ] Section: Validation rules and limits

**File**: `docs/service-tags-metadata.md`

---

#### Task 6.1.2: Update API Reference
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 6.1.1

**Description**:
Update API documentation with tag/metadata endpoints.

**Acceptance Criteria**:
- [ ] Document updated `/v1/agent/service/register` with tags/meta
- [ ] Document `/v1/catalog/services` query parameters
- [ ] Add request/response examples
- [ ] Document validation errors
- [ ] Add OpenAPI/Swagger spec updates

**File**: `docs/api-reference.md`

---

#### Task 6.1.3: Create Migration Guide
**Priority**: P1
**Estimated Time**: 3 hours
**Dependencies**: 6.1.1

**Description**:
Guide for migrating existing services to use tags/metadata.

**Acceptance Criteria**:
- [ ] Create `docs/migration-guides/tags-metadata.md`
- [ ] Backward compatibility notes
- [ ] Step-by-step migration process
- [ ] Example migration scripts
- [ ] Common pitfalls and solutions

**File**: `docs/migration-guides/tags-metadata.md`

---

#### Task 6.1.4: Update Main README
**Priority**: P1
**Estimated Time**: 2 hours
**Dependencies**: 6.1.1

**Description**:
Update README with tags/metadata feature.

**Acceptance Criteria**:
- [ ] Add to feature list
- [ ] Add quick example
- [ ] Link to detailed docs

**File**: `README.md`

---

## Summary

### Total Time Estimate

- **Phase 1**: Core Data Model - 36 hours (~1 week)
- **Phase 2**: API Endpoints - 20 hours (~2-3 days)
- **Phase 3**: DNS Integration - 9 hours (~1 day)
- **Phase 4**: CLI Support - 11 hours (~1-2 days)
- **Phase 5**: Admin UI - 12 hours (~1-2 days)
- **Phase 6**: Documentation - 14 hours (~2 days)

**Total**: ~102 hours (~2.5-3 weeks for a single developer)

### Critical Path

```
Phase 1 (Core) → Phase 2 (API) → Phase 3 (DNS) & Phase 4 (CLI) & Phase 5 (UI) → Phase 6 (Docs)
```

### Priorities

- **P0** (Must Have): Phase 1, Phase 2 core tasks
- **P1** (Should Have): Phase 2 metrics, Phase 3-6
- **P2** (Nice to Have): Advanced filtering, UI enhancements

### Dependencies Graph

```
1.1.1 → 1.1.2 → 1.1.3 → 1.1.4
         ↓
       1.2.1 → 1.2.2 → 1.2.3, 1.2.4, 1.2.5
                  ↓
              1.3.1, 1.3.2 → 1.3.3 → 1.3.4
                                ↓
                            Phase 2
                                ↓
                    Phase 3, Phase 4, Phase 5
                                ↓
                            Phase 6
```

### Quick Start (MVP)

For a minimum viable product, focus on:
1. Tasks 1.1.1 - 1.1.4 (Data structure + validation)
2. Tasks 1.2.1 - 1.2.5 (Indexing)
3. Tasks 1.3.1 - 1.3.4 (Query functions)
4. Tasks 2.1.1 - 2.2.3 (API endpoints)
5. Task 6.1.1 (Basic documentation)

**MVP Time**: ~50 hours (~1.5 weeks)

---

## Testing Checklist

- [ ] Unit tests for validation
- [ ] Unit tests for indexing
- [ ] Unit tests for query functions
- [ ] Integration tests for API endpoints
- [ ] Integration tests for DNS queries
- [ ] CLI command tests
- [ ] UI component tests
- [ ] Performance benchmarks
- [ ] Load tests (10,000 services with tags/meta)

## Performance Targets

- Service registration with tags/meta: <5ms
- Query by single tag: <10ms
- Query by multiple tags: <20ms
- Query by metadata: <20ms
- Combined query: <30ms
- Index memory overhead: <2KB per service

## Next Steps

1. **Review this plan** with the team
2. **Create GitHub issues** for each task
3. **Set up project board** to track progress
4. **Assign Phase 1 tasks** to developers
5. **Start with Task 1.1.1** - Update Service struct
