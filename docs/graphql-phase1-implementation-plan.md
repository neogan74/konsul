# GraphQL API - Phase 1 Implementation Plan

**Phase**: Core GraphQL Server with Query Resolvers (MVP)
**Timeline**: 2-3 weeks
**Status**: Planning
**Related ADR**: [ADR-0016: GraphQL API Interface](adr/0016-graphql-api-interface.md)

## Overview

Phase 1 focuses on establishing the foundational GraphQL infrastructure for Konsul. This phase implements read-only query operations for KV store and Service Discovery resources, providing a solid base for future mutations and subscriptions.

## Goals

- ✅ Setup GraphQL server using gqlgen
- ✅ Implement KV store query resolvers
- ✅ Implement Service Discovery query resolvers
- ✅ Add authentication integration (JWT/API key)
- ✅ Include GraphiQL playground for development
- ✅ Basic error handling and logging
- ✅ Integration with existing REST API

## Non-Goals (Deferred to Later Phases)

- ❌ Mutations (write operations) - Phase 2
- ❌ Subscriptions (real-time updates) - Phase 2
- ❌ DataLoaders and N+1 optimization - Phase 3
- ❌ Query complexity limits - Phase 3
- ❌ ACL/Health Check schemas - Phase 4

---

## Week 1: Setup & Foundation

### Day 1-2: Project Setup & Dependencies

#### Tasks

**1.1 Install gqlgen and dependencies**
```bash
# Add gqlgen to project
go get github.com/99designs/gqlgen@latest
go get github.com/99designs/gqlgen/graphql@latest
go get github.com/99designs/gqlgen/graphql/handler@latest
go get github.com/99designs/gqlgen/graphql/playground@latest

# Initialize gqlgen
go run github.com/99designs/gqlgen init
```

**1.2 Create directory structure**
```bash
mkdir -p internal/graphql/schema
mkdir -p internal/graphql/resolver
mkdir -p internal/graphql/middleware
mkdir -p internal/graphql/model
mkdir -p internal/graphql/generated
```

**1.3 Configure gqlgen**

Create `gqlgen.yml` in project root:

```yaml
# gqlgen.yml
schema:
  - internal/graphql/schema/*.graphql

exec:
  filename: internal/graphql/generated/generated.go
  package: generated

model:
  filename: internal/graphql/model/models_gen.go
  package: model

resolver:
  layout: follow-schema
  dir: internal/graphql/resolver
  package: resolver
  filename_template: "{name}.resolvers.go"

# Optional custom models
models:
  Time:
    model: time.Time
  Duration:
    model: time.Duration

# Disable some built-in types we'll define ourselves
autobind:
  - github.com/neogan74/konsul/internal/graphql/model
```

**Deliverables:**
- [ ] gqlgen installed and configured
- [ ] Directory structure created
- [ ] gqlgen.yml configured
- [ ] Dependencies added to go.mod

---

### Day 3-4: Schema Definition

#### Tasks

**2.1 Create base schema**

`internal/graphql/schema/schema.graphql`:

```graphql
# Root types
schema {
  query: Query
}

type Query {
  # System information
  health: SystemHealth!

  # KV Store queries
  kv(key: String!): KVPair
  kvList(prefix: String, limit: Int, offset: Int): KVListResponse!

  # Service Discovery queries
  service(name: String!): Service
  services(limit: Int, offset: Int): [Service!]!
  servicesCount: Int!
}
```

**2.2 Define KV types**

`internal/graphql/schema/kv.graphql`:

```graphql
"""
KVPair represents a key-value pair in the KV store
"""
type KVPair {
  """The key"""
  key: String!

  """The value"""
  value: String!

  """Creation timestamp"""
  createdAt: Time

  """Last modification timestamp"""
  updatedAt: Time
}

"""
Response type for listing KV pairs
"""
type KVListResponse {
  """List of key-value pairs"""
  items: [KVPair!]!

  """Total count of items (useful for pagination)"""
  total: Int!

  """Whether there are more items available"""
  hasMore: Boolean!
}
```

**2.3 Define Service types**

`internal/graphql/schema/service.graphql`:

```graphql
"""
Service represents a registered service in the service registry
"""
type Service {
  """Service name (unique identifier)"""
  name: String!

  """Service IP address or hostname"""
  address: String!

  """Service port number"""
  port: Int!

  """Service status"""
  status: ServiceStatus!

  """Expiration timestamp"""
  expiresAt: Time!

  """Health checks associated with this service"""
  checks: [HealthCheck!]!
}

"""
Service status enum
"""
enum ServiceStatus {
  """Service is active and not expired"""
  ACTIVE

  """Service has expired"""
  EXPIRED
}

"""
Health check definition
"""
type HealthCheck {
  """Check ID"""
  id: String!

  """Service ID this check belongs to"""
  serviceId: String!

  """Check name"""
  name: String!

  """Check type (http, tcp, grpc, ttl)"""
  type: HealthCheckType!

  """Current status"""
  status: HealthCheckStatus!

  """Status output/message"""
  output: String

  """Check interval"""
  interval: Duration

  """Check timeout"""
  timeout: Duration

  """Last check time"""
  lastChecked: Time
}

"""
Health check type
"""
enum HealthCheckType {
  HTTP
  TCP
  GRPC
  TTL
}

"""
Health check status
"""
enum HealthCheckStatus {
  PASSING
  WARNING
  CRITICAL
}
```

**2.4 Define common types**

`internal/graphql/schema/common.graphql`:

```graphql
"""
System health information
"""
type SystemHealth {
  """Overall system status"""
  status: String!

  """Konsul version"""
  version: String!

  """System uptime"""
  uptime: String!

  """Current timestamp"""
  timestamp: Time!

  """Service statistics"""
  services: ServiceStats!

  """KV store statistics"""
  kvStore: KVStats!
}

"""
Service statistics
"""
type ServiceStats {
  """Total registered services"""
  total: Int!

  """Active (non-expired) services"""
  active: Int!

  """Expired services"""
  expired: Int!
}

"""
KV store statistics
"""
type KVStats {
  """Total number of keys"""
  totalKeys: Int!
}

"""
Custom scalar for timestamps
"""
scalar Time

"""
Custom scalar for durations (e.g., "30s", "5m", "2h")
"""
scalar Duration
```

**2.5 Generate code**

```bash
# Generate GraphQL code
go run github.com/99designs/gqlgen generate
```

**Deliverables:**
- [ ] All schema files created
- [ ] Schema validated (no syntax errors)
- [ ] Generated code compiled successfully
- [ ] Custom scalars defined

---

### Day 5: Custom Scalars & Models

#### Tasks

**3.1 Implement Time scalar**

`internal/graphql/scalar/time.go`:

```go
package scalar

import (
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalTime marshals time.Time to RFC3339 string
func MarshalTime(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, fmt.Sprintf(`"%s"`, t.Format(time.RFC3339)))
	})
}

// UnmarshalTime unmarshals RFC3339 string to time.Time
func UnmarshalTime(v interface{}) (time.Time, error) {
	if tmpStr, ok := v.(string); ok {
		return time.Parse(time.RFC3339, tmpStr)
	}
	return time.Time{}, fmt.Errorf("time must be a string")
}
```

**3.2 Implement Duration scalar**

`internal/graphql/scalar/duration.go`:

```go
package scalar

import (
	"fmt"
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalDuration marshals time.Duration to string (e.g., "30s")
func MarshalDuration(d time.Duration) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		io.WriteString(w, fmt.Sprintf(`"%s"`, d.String()))
	})
}

// UnmarshalDuration unmarshals string to time.Duration
func UnmarshalDuration(v interface{}) (time.Duration, error) {
	if tmpStr, ok := v.(string); ok {
		return time.ParseDuration(tmpStr)
	}
	return 0, fmt.Errorf("duration must be a string")
}
```

**3.3 Update gqlgen.yml for custom scalars**

```yaml
# Add to gqlgen.yml under models:
models:
  Time:
    model: time.Time
  Duration:
    model: time.Duration
```

**3.4 Create custom model mappers**

`internal/graphql/model/mappers.go`:

```go
package model

import (
	"time"

	"github.com/neogan74/konsul/internal/store"
)

// MapKVPairFromStore converts store data to GraphQL model
func MapKVPairFromStore(key, value string) *KVPair {
	now := time.Now()
	return &KVPair{
		Key:       key,
		Value:     value,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
}

// MapServiceFromStore converts store.Service to GraphQL model
func MapServiceFromStore(svc store.Service, entry store.ServiceEntry) *Service {
	status := ServiceStatusActive
	if entry.ExpiresAt.Before(time.Now()) {
		status = ServiceStatusExpired
	}

	return &Service{
		Name:      svc.Name,
		Address:   svc.Address,
		Port:      svc.Port,
		Status:    status,
		ExpiresAt: entry.ExpiresAt,
		Checks:    []*HealthCheck{}, // Will populate in resolver
	}
}

// MapHealthCheckFromStore converts healthcheck.Check to GraphQL model
func MapHealthCheckFromStore(check interface{}) *HealthCheck {
	// Implementation depends on internal healthcheck structure
	// This is a placeholder
	return &HealthCheck{
		ID:        "check-1",
		ServiceID: "service-1",
		Name:      "health",
		Type:      HealthCheckTypeHTTP,
		Status:    HealthCheckStatusPassing,
	}
}
```

**Deliverables:**
- [ ] Time scalar implemented
- [ ] Duration scalar implemented
- [ ] Model mappers created
- [ ] Code generation updated and successful

---

## Week 2: Resolver Implementation


#### Tasks

**4.1 Create resolver scaffold**

`internal/graphql/resolver/resolver.go`:

```go
package resolver

import (
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/auth"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

// Resolver is the root resolver
type Resolver struct {
	kvStore      *store.KVStore
	serviceStore *store.ServiceStore
	aclEvaluator *acl.Evaluator
	jwtService   *auth.JWTService
	logger       logger.Logger
	version      string
	startTime    time.Time
}

// NewResolver creates a new resolver
func NewResolver(deps ResolverDependencies) *Resolver {
	return &Resolver{
		kvStore:      deps.KVStore,
		serviceStore: deps.ServiceStore,
		aclEvaluator: deps.ACLEvaluator,
		jwtService:   deps.JWTService,
		logger:       deps.Logger,
		version:      deps.Version,
		startTime:    time.Now(),
	}
}

// ResolverDependencies holds all dependencies for resolvers
type ResolverDependencies struct {
	KVStore      *store.KVStore
	ServiceStore *store.ServiceStore
	ACLEvaluator *acl.Evaluator
	JWTService   *auth.JWTService
	Logger       logger.Logger
	Version      string
}
```

**4.2 Implement KV resolvers**

`internal/graphql/resolver/kv.resolvers.go`:

```go
package resolver

import (
	"context"
	"strings"

	"github.com/neogan74/konsul/internal/graphql/model"
	"github.com/neogan74/konsul/internal/logger"
)

// Kv resolves a single key-value pair
func (r *queryResolver) Kv(ctx context.Context, key string) (*model.KVPair, error) {
	// Check authentication if required
	if err := r.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Check ACL permissions if enabled
	if r.aclEvaluator != nil {
		if err := r.checkACL(ctx, "kv", key, "read"); err != nil {
			return nil, err
		}
	}

	// Fetch from store
	value, exists := r.kvStore.Get(key)
	if !exists {
		return nil, nil // Return nil for not found (nullable field)
	}

	r.logger.Debug("GraphQL: fetched KV pair",
		logger.String("key", key))

	return model.MapKVPairFromStore(key, value), nil
}

// KvList resolves a list of key-value pairs with optional prefix filter
func (r *queryResolver) KvList(ctx context.Context, prefix *string, limit *int, offset *int) (*model.KVListResponse, error) {
	// Check authentication
	if err := r.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Get all keys
	allKeys := r.kvStore.List()

	// Filter by prefix if provided
	var filteredKeys []string
	if prefix != nil && *prefix != "" {
		for _, key := range allKeys {
			if strings.HasPrefix(key, *prefix) {
				filteredKeys = append(filteredKeys, key)
			}
		}
	} else {
		filteredKeys = allKeys
	}

	total := len(filteredKeys)

	// Apply pagination
	start := 0
	if offset != nil {
		start = *offset
		if start > total {
			start = total
		}
	}

	end := total
	if limit != nil {
		end = start + *limit
		if end > total {
			end = total
		}
	}

	paginatedKeys := filteredKeys[start:end]

	// Build response
	items := make([]*model.KVPair, 0, len(paginatedKeys))
	for _, key := range paginatedKeys {
		if value, exists := r.kvStore.Get(key); exists {
			// Check ACL for each key if enabled
			if r.aclEvaluator != nil {
				if err := r.checkACL(ctx, "kv", key, "read"); err != nil {
					continue // Skip keys user doesn't have access to
				}
			}
			items = append(items, model.MapKVPairFromStore(key, value))
		}
	}

	r.logger.Debug("GraphQL: listed KV pairs",
		logger.String("prefix", stringOrEmpty(prefix)),
		logger.Int("total", total),
		logger.Int("returned", len(items)))

	return &model.KVListResponse{
		Items:   items,
		Total:   total,
		HasMore: end < total,
	}, nil
}

// Helper functions
func (r *queryResolver) checkAuth(ctx context.Context) error {
	// Check if authentication is required
	// Implementation depends on how auth context is set
	return nil
}

func (r *queryResolver) checkACL(ctx context.Context, resourceType, resource, action string) error {
	// Check ACL permissions
	// Implementation depends on ACL evaluator interface
	return nil
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
```

**Deliverables:**
- [ ] Resolver scaffold created
- [ ] KV resolvers implemented
- [ ] Authentication checks integrated
- [ ] ACL checks integrated (if ACL enabled)
- [ ] Pagination logic working

---

### Day 8-9: Service Discovery Resolvers

#### Tasks

**5.1 Implement Service resolvers**

`internal/graphql/resolver/service.resolvers.go`:

```go
package resolver

import (
	"context"

	"github.com/neogan74/konsul/internal/graphql/model"
	"github.com/neogan74/konsul/internal/logger"
)

// Service resolves a single service by name
func (r *queryResolver) Service(ctx context.Context, name string) (*model.Service, error) {
	// Check authentication
	if err := r.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Check ACL
	if r.aclEvaluator != nil {
		if err := r.checkACL(ctx, "service", name, "read"); err != nil {
			return nil, err
		}
	}

	// Fetch from store
	svc, exists := r.serviceStore.Get(name)
	if !exists {
		return nil, nil // Return nil for not found
	}

	// Get full entry for expiration info
	entries := r.serviceStore.ListAll()
	var entry *store.ServiceEntry
	for _, e := range entries {
		if e.Service.Name == name {
			entry = &e
			break
		}
	}

	if entry == nil {
		return nil, nil
	}

	r.logger.Debug("GraphQL: fetched service",
		logger.String("name", name))

	return model.MapServiceFromStore(svc, *entry), nil
}

// Services resolves all services with pagination
func (r *queryResolver) Services(ctx context.Context, limit *int, offset *int) ([]*model.Service, error) {
	// Check authentication
	if err := r.checkAuth(ctx); err != nil {
		return nil, err
	}

	// Get all services
	entries := r.serviceStore.ListAll()

	// Apply pagination
	start := 0
	if offset != nil {
		start = *offset
		if start > len(entries) {
			start = len(entries)
		}
	}

	end := len(entries)
	if limit != nil {
		end = start + *limit
		if end > len(entries) {
			end = len(entries)
		}
	}

	paginatedEntries := entries[start:end]

	// Map to GraphQL models
	services := make([]*model.Service, 0, len(paginatedEntries))
	for _, entry := range paginatedEntries {
		// Check ACL for each service
		if r.aclEvaluator != nil {
			if err := r.checkACL(ctx, "service", entry.Service.Name, "read"); err != nil {
				continue // Skip services user doesn't have access to
			}
		}
		services = append(services, model.MapServiceFromStore(entry.Service, entry))
	}

	r.logger.Debug("GraphQL: listed services",
		logger.Int("total", len(entries)),
		logger.Int("returned", len(services)))

	return services, nil
}

// ServicesCount returns the total count of services
func (r *queryResolver) ServicesCount(ctx context.Context) (int, error) {
	// Check authentication
	if err := r.checkAuth(ctx); err != nil {
		return 0, err
	}

	services := r.serviceStore.List()
	return len(services), nil
}
```

**5.2 Implement nested Service.Checks resolver**

```go
// Checks resolves health checks for a service
func (r *serviceResolver) Checks(ctx context.Context, obj *model.Service) ([]*model.HealthCheck, error) {
	// Fetch health checks for this service
	checks := r.serviceStore.GetHealthChecks(obj.Name)

	// Map to GraphQL models
	result := make([]*model.HealthCheck, 0, len(checks))
	for _, check := range checks {
		result = append(result, model.MapHealthCheckFromStore(check))
	}

	return result, nil
}
```

**Deliverables:**
- [ ] Service resolvers implemented
- [ ] Nested resolvers for health checks
- [ ] Pagination working
- [ ] ACL checks integrated

---

### Day 10: System Health Resolver

#### Tasks

**6.1 Implement Health resolver**

`internal/graphql/resolver/health.resolvers.go`:

```go
package resolver

import (
	"context"
	"time"

	"github.com/neogan74/konsul/internal/graphql/model"
)

// Health resolves system health information
func (r *queryResolver) Health(ctx context.Context) (*model.SystemHealth, error) {
	// No auth required for health endpoint (public)

	// Get service stats
	allEntries := r.serviceStore.ListAll()
	activeServices := r.serviceStore.List()
	expiredCount := len(allEntries) - len(activeServices)

	// Get KV stats
	allKeys := r.kvStore.List()

	// Calculate uptime
	uptime := time.Since(r.startTime).String()

	return &model.SystemHealth{
		Status:    "healthy",
		Version:   r.version,
		Uptime:    uptime,
		Timestamp: time.Now(),
		Services: &model.ServiceStats{
			Total:   len(allEntries),
			Active:  len(activeServices),
			Expired: expiredCount,
		},
		KvStore: &model.KVStats{
			TotalKeys: len(allKeys),
		},
	}, nil
}
```

**Deliverables:**
- [ ] Health resolver implemented
- [ ] System stats calculated correctly
- [ ] No authentication required (public endpoint)

---

## Week 3: Integration & Testing

### Day 11-12: GraphQL Server Setup

#### Tasks

**7.1 Create GraphQL server**

`internal/graphql/server.go`:

```go
package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/neogan74/konsul/internal/graphql/generated"
	"github.com/neogan74/konsul/internal/graphql/resolver"
)

// Server wraps the GraphQL handler
type Server struct {
	handler    http.Handler
	playground http.Handler
}

// NewServer creates a new GraphQL server
func NewServer(deps resolver.ResolverDependencies) *Server {
	// Create resolver
	r := resolver.NewResolver(deps)

	// Create GraphQL handler
	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{
				Resolvers: r,
			},
		),
	)

	// Add middleware (will expand in Phase 3)
	// srv.Use(extension.FixedComplexityLimit(1000))

	return &Server{
		handler:    srv,
		playground: playground.Handler("GraphQL Playground", "/graphql"),
	}
}

// Handler returns the GraphQL HTTP handler
func (s *Server) Handler() http.Handler {
	return s.handler
}

// PlaygroundHandler returns the GraphiQL playground handler
func (s *Server) PlaygroundHandler() http.Handler {
	return s.playground
}
```

**7.2 Integrate with Fiber (main.go)**

Add to `cmd/konsul/main.go`:

```go
// After line ~357 (after Admin UI setup)

// GraphQL setup (if enabled)
if cfg.GraphQL.Enabled {
	gqlDeps := graphqlresolver.ResolverDependencies{
		KVStore:      kv,
		ServiceStore: svcStore,
		ACLEvaluator: aclEvaluator,
		JWTService:   jwtService,
		Logger:       appLogger,
		Version:      version,
	}

	gqlServer := graphql.NewServer(gqlDeps)

	// GraphQL endpoint
	app.All("/graphql", adaptor.HTTPHandlerFunc(gqlServer.Handler().ServeHTTP))

	// GraphQL Playground (disable in production)
	if cfg.GraphQL.PlaygroundEnabled {
		app.Get("/graphql/playground", adaptor.HTTPHandlerFunc(gqlServer.PlaygroundHandler().ServeHTTP))
		appLogger.Info("GraphQL Playground available at /graphql/playground")
	}

	appLogger.Info("GraphQL API enabled at /graphql")
}
```

**7.3 Add configuration**

Update `internal/config/config.go`:

```go
// Add to Config struct
type Config struct {
	// ... existing fields ...

	GraphQL GraphQLConfig `json:"graphql"`
}

type GraphQLConfig struct {
	Enabled           bool `json:"enabled"`
	PlaygroundEnabled bool `json:"playground_enabled"`
}

// Add environment variable loading
func Load() (*Config, error) {
	// ... existing code ...

	cfg.GraphQL = GraphQLConfig{
		Enabled:           getEnvBool("KONSUL_GRAPHQL_ENABLED", false),
		PlaygroundEnabled: getEnvBool("KONSUL_GRAPHQL_PLAYGROUND_ENABLED", true),
	}

	return cfg, nil
}
```

**Deliverables:**
- [ ] GraphQL server created
- [ ] Integration with Fiber complete
- [ ] Configuration added
- [ ] Environment variables documented

---

### Day 13-14: Testing

#### Tasks

**8.1 Unit tests for resolvers**

`internal/graphql/resolver/kv_test.go`:

```go
package resolver_test

import (
	"context"
	"testing"

	"github.com/neogan74/konsul/internal/graphql/resolver"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKVResolver(t *testing.T) {
	// Setup
	kvStore := store.NewKVStore()
	kvStore.Set("test-key", "test-value")
	kvStore.Set("prefix/key1", "value1")
	kvStore.Set("prefix/key2", "value2")

	deps := resolver.ResolverDependencies{
		KVStore: kvStore,
		Logger:  logger.NewFromConfig("info", "text"),
		Version: "test",
	}

	r := resolver.NewResolver(deps)
	ctx := context.Background()

	t.Run("Get existing key", func(t *testing.T) {
		result, err := r.Query().Kv(ctx, "test-key")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test-key", result.Key)
		assert.Equal(t, "test-value", result.Value)
	})

	t.Run("Get non-existing key", func(t *testing.T) {
		result, err := r.Query().Kv(ctx, "non-existing")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("List with prefix", func(t *testing.T) {
		prefix := "prefix/"
		result, err := r.Query().KvList(ctx, &prefix, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Items))
		assert.Equal(t, 2, result.Total)
	})

	t.Run("List with pagination", func(t *testing.T) {
		limit := 1
		offset := 0
		result, err := r.Query().KvList(ctx, nil, &limit, &offset)
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Items))
		assert.True(t, result.HasMore)
	})
}
```

**8.2 Integration tests**

`internal/graphql/integration_test.go`:

```go
package graphql_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neogan74/konsul/internal/graphql"
	"github.com/neogan74/konsul/internal/graphql/resolver"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLQueries(t *testing.T) {
	// Setup stores
	kvStore := store.NewKVStore()
	kvStore.Set("config/app", "production")

	serviceStore := store.NewServiceStoreWithTTL(30 * time.Second)
	serviceStore.Register(store.Service{
		Name:    "web",
		Address: "10.0.0.1",
		Port:    8080,
	})

	// Create GraphQL server
	deps := resolver.ResolverDependencies{
		KVStore:      kvStore,
		ServiceStore: serviceStore,
		Logger:       logger.NewFromConfig("info", "text"),
		Version:      "test",
	}

	server := graphql.NewServer(deps)

	t.Run("KV Query", func(t *testing.T) {
		query := `
		{
			kv(key: "config/app") {
				key
				value
			}
		}
		`

		resp := executeQuery(t, server, query)

		var result struct {
			Data struct {
				KV struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"kv"`
			} `json:"data"`
		}

		err := json.Unmarshal(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "config/app", result.Data.KV.Key)
		assert.Equal(t, "production", result.Data.KV.Value)
	})

	t.Run("Service Query", func(t *testing.T) {
		query := `
		{
			service(name: "web") {
				name
				address
				port
				status
			}
		}
		`

		resp := executeQuery(t, server, query)

		var result struct {
			Data struct {
				Service struct {
					Name    string `json:"name"`
					Address string `json:"address"`
					Port    int    `json:"port"`
					Status  string `json:"status"`
				} `json:"service"`
			} `json:"data"`
		}

		err := json.Unmarshal(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "web", result.Data.Service.Name)
		assert.Equal(t, "10.0.0.1", result.Data.Service.Address)
		assert.Equal(t, 8080, result.Data.Service.Port)
		assert.Equal(t, "ACTIVE", result.Data.Service.Status)
	})

	t.Run("Health Query", func(t *testing.T) {
		query := `
		{
			health {
				status
				version
				services {
					total
					active
				}
				kvStore {
					totalKeys
				}
			}
		}
		`

		resp := executeQuery(t, server, query)

		var result struct {
			Data struct {
				Health struct {
					Status  string `json:"status"`
					Version string `json:"version"`
					Services struct {
						Total  int `json:"total"`
						Active int `json:"active"`
					} `json:"services"`
					KVStore struct {
						TotalKeys int `json:"totalKeys"`
					} `json:"kvStore"`
				} `json:"health"`
			} `json:"data"`
		}

		err := json.Unmarshal(resp, &result)
		require.NoError(t, err)
		assert.Equal(t, "healthy", result.Data.Health.Status)
		assert.Equal(t, 1, result.Data.Health.Services.Total)
		assert.Equal(t, 1, result.Data.Health.KVStore.TotalKeys)
	})
}

func executeQuery(t *testing.T, server *graphql.Server, query string) []byte {
	reqBody := map[string]string{"query": query}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/graphql", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	return w.Body.Bytes()
}
```

**Deliverables:**
- [ ] Unit tests for all resolvers
- [ ] Integration tests for GraphQL queries
- [ ] Test coverage > 80%
- [ ] All tests passing

---

### Day 15: Documentation & Polish

#### Tasks

**9.1 Create GraphQL documentation**

`docs/graphql-api.md`:

```markdown
# GraphQL API Documentation

## Overview

Konsul provides a GraphQL API alongside the REST API for flexible querying.

## Endpoint

- **GraphQL API**: `POST /graphql`
- **GraphQL Playground**: `GET /graphql/playground` (development only)

## Configuration

Enable GraphQL via environment variables:

```bash
KONSUL_GRAPHQL_ENABLED=true
KONSUL_GRAPHQL_PLAYGROUND_ENABLED=true  # Disable in production
```

## Authentication

GraphQL API uses the same authentication as REST:

- JWT token via `Authorization: Bearer <token>` header
- API key via `X-API-Key: <key>` header

## Example Queries

### Get KV Pair

```graphql
query {
  kv(key: "config/database") {
    key
    value
    createdAt
  }
}
```

### List Services

```graphql
query {
  services {
    name
    address
    port
    status
    checks {
      status
      output
    }
  }
}
```

### System Health

```graphql
query {
  health {
    status
    version
    uptime
    services {
      total
      active
    }
  }
}
```

## Schema

Full schema documentation available via introspection or GraphQL Playground.
```

**9.2 Update README.md**

Add GraphQL section to main README:

```markdown
## GraphQL API

Konsul supports GraphQL for flexible querying:

```bash
# Enable GraphQL
KONSUL_GRAPHQL_ENABLED=true ./konsul

# Access playground
open http://localhost:8888/graphql/playground
```

See [GraphQL API Documentation](docs/graphql-api.md) for details.
```

**9.3 Add metrics**

Update `internal/metrics/metrics.go`:

```go
var (
	// ... existing metrics ...

	GraphQLRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "konsul_graphql_requests_total",
			Help: "Total GraphQL requests",
		},
		[]string{"operation", "status"},
	)

	GraphQLRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "konsul_graphql_request_duration_seconds",
			Help:    "GraphQL request latency",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	// ... existing registrations ...
	prometheus.MustRegister(GraphQLRequestsTotal)
	prometheus.MustRegister(GraphQLRequestDuration)
}
```

**Deliverables:**
- [ ] GraphQL API documentation complete
- [ ] README updated
- [ ] Metrics added for GraphQL operations
- [ ] Example queries documented

---

## Acceptance Criteria

### Functional Requirements

- [x] GraphQL endpoint available at `/graphql`
- [x] GraphiQL playground available at `/graphql/playground`
- [x] All KV store queries working (kv, kvList)
- [x] All Service queries working (service, services, servicesCount)
- [x] Health query returns system stats
- [x] Authentication integration (JWT/API key)
- [x] ACL enforcement (if enabled)
- [x] Pagination working for list queries
- [x] Custom scalars (Time, Duration) working

### Non-Functional Requirements

- [x] Test coverage > 80%
- [x] All resolver tests passing
- [x] Integration tests passing
- [x] Documentation complete
- [x] No breaking changes to existing REST API
- [x] Performance acceptable (< 100ms for simple queries)

### Quality Gates

- [ ] Code review completed
- [ ] All tests passing in CI
- [ ] No security vulnerabilities
- [ ] Documentation reviewed
- [ ] Metrics validated in Prometheus

---

## Dependencies

### Go Packages

```bash
go get github.com/99designs/gqlgen@latest
go get github.com/99designs/gqlgen/graphql@latest
go get github.com/99designs/gqlgen/graphql/handler@latest
go get github.com/99designs/gqlgen/graphql/playground@latest
```

### Internal Dependencies

- `internal/store` - KV and Service stores
- `internal/auth` - JWT and API key authentication
- `internal/acl` - ACL evaluation
- `internal/logger` - Structured logging
- `internal/metrics` - Prometheus metrics

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Schema changes break clients | High | Version schema, use deprecation |
| Performance issues with large datasets | Medium | Implement pagination, add limits |
| Authentication bypass | High | Thorough testing, code review |
| ACL not enforced | High | Integration tests for ACL scenarios |
| Breaking REST API | Medium | Separate code paths, testing |

---

## Success Metrics

- GraphQL endpoint serving requests successfully
- Zero breaking changes to REST API
- Test coverage > 80%
- Response time < 100ms for simple queries
- Documentation complete and accurate

---

## Next Steps (Phase 2)

After Phase 1 completion:

1. **Mutations** - Implement write operations (kvSet, registerService, etc.)
2. **Subscriptions** - Real-time updates via WebSocket
3. **Advanced Features** - DataLoaders, complexity limits, batch operations

See [ADR-0016](adr/0016-graphql-api-interface.md) for full roadmap.

---

## Questions & Clarifications

- Should GraphQL be enabled by default? **No, opt-in via config**
- Should playground be available in production? **No, development only**
- Authentication required for all queries? **Yes, except health**
- ACL granularity? **Same as REST - resource-level**
