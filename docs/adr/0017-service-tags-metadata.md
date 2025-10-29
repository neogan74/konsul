# ADR-0017: Service Tags and Metadata

**Date**: 2025-10-28

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: service-discovery, filtering, metadata, search

## Context

Currently, Konsul's service registration only supports basic fields: `name`, `address`, `port`, and `checks`. This creates significant limitations for real-world service discovery scenarios:

### Current Limitations

1. **No service categorization**: Cannot group services by environment (prod/staging), version, or datacenter
2. **Limited filtering**: Cannot query services by attributes (e.g., "all HTTP services" or "all services in us-west-2")
3. **No metadata**: Cannot store custom information about services (team ownership, SLA tier, cost center)
4. **Poor multi-environment support**: No way to distinguish between services in different environments
5. **No version tracking**: Cannot identify which version of a service is deployed
6. **Limited routing decisions**: Load balancers/proxies cannot route based on service attributes
7. **No compatibility with Consul**: Missing features that exist in HashiCorp Consul

### Real-World Use Cases

**Use Case 1: Multi-Environment Deployment**
- Tag services with `env:production`, `env:staging`, `env:development`
- Query only production services for critical operations
- Separate monitoring/alerting by environment

**Use Case 2: Canary Deployments**
- Tag services with `version:v1.2.3`, `canary:true`
- Route 5% of traffic to canary instances
- Gradually shift traffic based on metrics

**Use Case 3: Geographic Routing**
- Tag services with `region:us-east-1`, `az:us-east-1a`
- Route requests to nearest datacenter
- Implement disaster recovery failover

**Use Case 4: Protocol Filtering**
- Tag services with `protocol:http`, `protocol:grpc`, `protocol:tcp`
- Service meshes can filter by protocol type
- API gateways can apply protocol-specific policies

**Use Case 5: Team Ownership**
- Metadata: `team:platform`, `owner:alice@example.com`
- Track which team owns which service
- Route alerts to correct team

**Use Case 6: Cost Management**
- Metadata: `cost-center:engineering`, `billing:project-x`
- Track infrastructure costs by project
- Implement chargeback models

### Requirements

**Functional Requirements**:
- Add **tags** to services (array of strings)
- Add **metadata** to services (key-value pairs)
- Filter services by tags (AND/OR logic)
- Filter services by metadata
- Query services with multiple filters
- Backward compatibility with existing services
- ACL support for tag/metadata filtering

**Non-Functional Requirements**:
- Minimal performance impact on service lookup
- Efficient filtering (avoid full table scans)
- Index tags for fast queries
- Support 50+ tags per service
- Support 100+ metadata keys per service
- Query performance <10ms for tag filtering

## Decision

We will add **tags** (string array) and **metadata** (key-value map) to service definitions, with comprehensive filtering and query capabilities.

### Service Structure

```go
type Service struct {
    Name     string                         `json:"name"`
    Address  string                         `json:"address"`
    Port     int                            `json:"port"`
    Tags     []string                       `json:"tags,omitempty"`           // NEW
    Meta     map[string]string              `json:"meta,omitempty"`           // NEW
    Checks   []*healthcheck.CheckDefinition `json:"checks,omitempty"`
}
```

### Tags Design

**Tags** are simple string labels:
- Format: `key:value` or `simple-label`
- Examples: `env:production`, `version:v1.2.3`, `http`, `grpc`, `canary`
- Use cases: Environment, version, protocol, feature flags
- Indexed for fast filtering
- Case-sensitive

**Tag Conventions**:
```
# Environment
env:production, env:staging, env:development

# Version
version:v1.2.3, version:v2.0.0

# Protocol
http, grpc, tcp, udp

# Features
canary, blue, green

# Geographic
region:us-east-1, az:us-east-1a, dc:aws
```

### Metadata Design

**Metadata** is structured key-value data:
- Format: `{"key": "value"}`
- Examples: `{"team": "platform", "owner": "alice@example.com"}`
- Use cases: Ownership, cost tracking, custom attributes
- Not indexed (slower filtering)
- Case-sensitive keys

**Metadata Conventions**:
```json
{
  "team": "platform",
  "owner": "alice@example.com",
  "cost-center": "engineering",
  "sla-tier": "critical",
  "oncall-slack": "#platform-oncall",
  "documentation": "https://wiki.example.com/api-service",
  "git-repo": "https://github.com/example/api-service"
}
```

### API Design

#### Registration with Tags and Metadata

**Request**:
```http
POST /v1/agent/service/register
Content-Type: application/json

{
  "name": "api-service",
  "address": "10.0.1.50",
  "port": 8080,
  "tags": ["env:production", "version:v1.2.3", "http", "region:us-east-1"],
  "meta": {
    "team": "platform",
    "owner": "alice@example.com",
    "cost-center": "engineering"
  },
  "checks": [...]
}
```

**Response**:
```http
HTTP/1.1 200 OK

{
  "message": "service registered",
  "service": {
    "name": "api-service",
    "address": "10.0.1.50",
    "port": 8080,
    "tags": ["env:production", "version:v1.2.3", "http", "region:us-east-1"],
    "meta": {
      "team": "platform",
      "owner": "alice@example.com",
      "cost-center": "engineering"
    }
  }
}
```

#### Query Services by Tags

**Endpoint**: `GET /v1/catalog/services?tag=<tag>`

**Examples**:
```bash
# Single tag filter
GET /v1/catalog/services?tag=env:production

# Multiple tags (AND logic)
GET /v1/catalog/services?tag=env:production&tag=http

# Tag prefix matching
GET /v1/catalog/services?tag=version:v1.*
```

**Response**:
```json
[
  {
    "name": "api-service",
    "address": "10.0.1.50",
    "port": 8080,
    "tags": ["env:production", "version:v1.2.3", "http"],
    "meta": {"team": "platform"}
  }
]
```

#### Query Services by Metadata

**Endpoint**: `GET /v1/catalog/services?meta=<key>:<value>`

**Examples**:
```bash
# Single metadata filter
GET /v1/catalog/services?meta=team:platform

# Multiple metadata filters (AND logic)
GET /v1/catalog/services?meta=team:platform&meta=sla-tier:critical
```

#### Combined Filters

**Endpoint**: `GET /v1/catalog/services?tag=<tag>&meta=<key>:<value>`

**Example**:
```bash
# Production services owned by platform team
GET /v1/catalog/services?tag=env:production&meta=team:platform
```

#### DNS Query with Tags

**Format**: `<tag>.<service>.service.konsul`

**Examples**:
```bash
# Query production instances of api-service
dig @localhost -p 8600 production.api-service.service.konsul

# Query HTTP protocol services
dig @localhost -p 8600 http.web-service.service.konsul
```

### Storage & Indexing

**In-Memory Index**:
```go
type ServiceStore struct {
    Data          map[string]ServiceEntry       // Name → Service
    TagIndex      map[string]map[string]bool    // Tag → {ServiceName: true}
    MetaIndex     map[string]map[string][]string // MetaKey → {MetaValue: [ServiceNames]}
    Mutex         sync.RWMutex
    // ... existing fields
}
```

**Index Updates**:
- On service registration: Add to tag/meta indexes
- On service deregistration: Remove from tag/meta indexes
- On heartbeat: No index update needed
- On expiration: Cleanup indexes

**Query Performance**:
- Tag queries: O(n) where n = services with matching tag (indexed)
- Metadata queries: O(n) where n = services with matching metadata (indexed)
- Combined queries: Intersection of results
- No tag/metadata: O(1) lookup by name

### Validation

**Tag Validation**:
- Maximum 64 tags per service
- Each tag max 255 characters
- Allowed characters: alphanumeric, `-`, `_`, `:`, `.`, `/`
- No duplicate tags

**Metadata Validation**:
- Maximum 64 metadata keys per service
- Key max 128 characters
- Value max 512 characters
- Allowed key characters: alphanumeric, `-`, `_`
- No reserved keys (internal use): `_`, `konsul_*`

### ACL Integration

**Tag-Based ACL**:
```json
{
  "service": [
    {
      "name": "api-service",
      "tags": ["env:production"],
      "capabilities": ["read", "write"]
    }
  ]
}
```

**ACL Evaluation**:
- Check service name permission first
- Then check if user has permission for service tags
- Deny if user lacks required tag permissions

## Alternatives Considered

### Alternative 1: Tags Only (No Metadata)

- **Pros**:
  - Simpler implementation
  - Easier to query
  - Better performance
  - Smaller payload size
- **Cons**:
  - Tags don't work well for complex data
  - No structured data support
  - Forces encoding in tags (e.g., `owner:alice@example.com`)
  - Poor user experience for non-filtering data
- **Reason for rejection**: Metadata is needed for ownership, documentation links, and other non-filtering use cases

### Alternative 2: Metadata Only (No Tags)

- **Pros**:
  - More flexible
  - Structured data support
  - Single concept to learn
- **Cons**:
  - Slower queries (no optimized indexing)
  - More verbose for simple labels
  - JSON parsing overhead
  - Harder to implement efficient filtering
- **Reason for rejection**: Tags are more efficient for filtering, which is the primary use case

### Alternative 3: Nested Metadata Structure

**Example**:
```json
{
  "meta": {
    "deployment": {
      "environment": "production",
      "version": "v1.2.3"
    },
    "ownership": {
      "team": "platform",
      "owner": "alice"
    }
  }
}
```

- **Pros**:
  - Better organization
  - Namespace separation
  - More structured
- **Cons**:
  - Complex querying (nested path syntax)
  - Harder to index
  - More parsing overhead
  - Incompatible with Consul API
- **Reason for rejection**: Flat structure is simpler and sufficient

### Alternative 4: GraphQL Schema for Services

- **Pros**:
  - Powerful querying
  - Flexible filtering
  - Type safety
  - Good tooling
- **Cons**:
  - Overkill for service discovery
  - Adds complexity (GraphQL server)
  - Higher latency
  - Not standard for service discovery
- **Reason for rejection**: REST API with query parameters is simpler and more performant

### Alternative 5: Labels (Kubernetes-Style)

**Example**: `{"env": "production", "version": "v1.2.3"}`

- **Pros**:
  - Familiar to Kubernetes users
  - Structured key-value
  - Easy to query
- **Cons**:
  - Redundant with both tags and metadata
  - Not compatible with Consul API
  - Forces all labels to be indexed
- **Reason for rejection**: Tags + metadata provides better separation of concerns

### Alternative 6: Do Nothing (Status Quo)

- **Pros**:
  - No implementation work
  - No complexity added
  - No performance impact
- **Cons**:
  - Cannot support multi-environment deployments
  - Poor user experience for filtering
  - Not competitive with Consul
  - Missing critical feature for production use
- **Reason for rejection**: Feature is essential for production service discovery

## Consequences

### Positive

- **Better service organization**: Group services by environment, version, protocol
- **Powerful filtering**: Query services by multiple attributes
- **Multi-environment support**: Separate prod/staging/dev services
- **Canary deployment support**: Tag and route to specific versions
- **Geographic routing**: Filter by region/AZ for latency optimization
- **Team ownership tracking**: Metadata for oncall, documentation, cost tracking
- **Consul API compatibility**: Aligns with HashiCorp Consul API
- **DNS filtering**: Query services by tags via DNS
- **Service mesh integration**: Proxies can route based on tags/metadata
- **Better monitoring**: Group metrics/alerts by environment, team, version
- **Cost tracking**: Chargeback models using metadata
- **Improved discoverability**: Search services by attributes

### Negative

- **Increased payload size**: Services now carry more data
- **Index maintenance overhead**: Tag/metadata indexes need updates
- **Query complexity**: More filter combinations to test
- **Memory usage**: Indexes consume more memory
- **API surface expansion**: More query parameters to support
- **Validation complexity**: Must validate tags and metadata
- **Migration needed**: Existing services need updates (optional)
- **Documentation burden**: Must document conventions and best practices
- **ACL complexity**: Tag-based ACL adds evaluation logic

### Neutral

- Need to document tag conventions
- CLI tools need updates for tags/metadata
- Admin UI needs tag/metadata input fields
- Prometheus metrics should include tag labels
- DNS query format changes

## Implementation Notes

### Phase 1: Core Data Model (Week 1)

**Tasks**:
1. Update `Service` struct with `Tags` and `Meta` fields
2. Update service store to support tags/metadata
3. Add tag/metadata indexing
4. Update persistence layer
5. Write unit tests

**Files**:
- `internal/store/service.go`
- `internal/store/service_test.go`

### Phase 2: API Endpoints (Week 1-2)

**Tasks**:
1. Update `POST /v1/agent/service/register` to accept tags/meta
2. Add `GET /v1/catalog/services?tag=<tag>` endpoint
3. Add `GET /v1/catalog/services?meta=<key>:<value>` endpoint
4. Update validation logic
5. Add ACL checks for tag-based filtering
6. Write integration tests

**Files**:
- `internal/handlers/service.go`
- `internal/handlers/catalog.go` (new)

### Phase 3: DNS Integration (Week 2)

**Tasks**:
1. Update DNS handler to support tag queries
2. Implement `<tag>.<service>.service.konsul` format
3. Add DNS tests

**Files**:
- `internal/dns/handler.go`

### Phase 4: CLI Support (Week 2-3)

**Tasks**:
1. Update `konsulctl service register` to accept tags/meta
2. Add `konsulctl service list --tag <tag>` command
3. Add `konsulctl service list --meta <key>:<value>` command
4. Add table formatting for tags/metadata
5. Update CLI documentation

**Files**:
- `cmd/konsulctl/service.go`

### Phase 5: Admin UI (Week 3)

**Tasks**:
1. Add tag input field (multi-select or comma-separated)
2. Add metadata input field (key-value pairs)
3. Add tag filter dropdown
4. Add metadata filter
5. Display tags/metadata in service list

**Files**:
- `ui/src/components/ServiceForm.tsx`
- `ui/src/components/ServiceList.tsx`
- `ui/src/components/ServiceFilters.tsx`

### Phase 6: Documentation (Week 3-4)

**Tasks**:
1. Document tag conventions
2. Document metadata conventions
3. Add API examples
4. Create migration guide
5. Update architecture docs

**Files**:
- `docs/service-tags-metadata.md`
- `docs/api-reference.md`
- `docs/migration-guides/tags-metadata.md`

### Migration Path

**Backward Compatibility**:
- Services without tags/metadata continue to work
- Tags and metadata are optional fields
- Old API format remains valid
- No breaking changes

**Migrating Existing Services**:
1. Identify services that need tags
2. Define tagging conventions for your organization
3. Update service registration code to include tags/meta
4. Re-register services with new fields
5. Update queries to use tag filters

**Example Migration**:
```bash
# Before
konsulctl service register api-service --address 10.0.1.50 --port 8080

# After
konsulctl service register api-service \
  --address 10.0.1.50 \
  --port 8080 \
  --tag env:production \
  --tag version:v1.2.3 \
  --tag http \
  --meta team:platform \
  --meta owner:alice@example.com
```

### Performance Considerations

**Indexing Strategy**:
- Tag index: `map[string]map[string]bool` - O(1) lookup by tag
- Metadata index: `map[string]map[string][]string` - O(1) lookup by key-value
- Memory overhead: ~1-2KB per service with 10 tags and 10 metadata keys

**Query Optimization**:
- Tag queries use indexed lookups (fast)
- Metadata queries use indexed lookups (fast)
- Multiple filters use set intersection (efficient)
- No filter falls back to list all (unchanged)

**Benchmarks** (Target):
- Service registration with tags/meta: <5ms
- Query by single tag: <10ms
- Query by multiple tags: <20ms
- Query by metadata: <20ms
- Combined tag + metadata query: <30ms

### Testing Strategy

**Unit Tests**:
- Service struct serialization with tags/meta
- Tag index updates
- Metadata index updates
- Filter logic (AND/OR)
- Validation (tag format, limits)

**Integration Tests**:
- Register service with tags/meta
- Query services by single tag
- Query services by multiple tags
- Query services by metadata
- Combined tag + metadata queries
- DNS queries with tags

**Performance Tests**:
- Benchmark query latency
- Benchmark memory usage
- Stress test with 10,000 services

### Configuration

**New Config Options**:
```yaml
# Service tags/metadata limits
service:
  max_tags: 64
  max_tag_length: 255
  max_metadata_keys: 64
  max_metadata_key_length: 128
  max_metadata_value_length: 512
```

**Environment Variables**:
```bash
KONSUL_SERVICE_MAX_TAGS=64
KONSUL_SERVICE_MAX_TAG_LENGTH=255
KONSUL_SERVICE_MAX_METADATA_KEYS=64
KONSUL_SERVICE_MAX_METADATA_KEY_LENGTH=128
KONSUL_SERVICE_MAX_METADATA_VALUE_LENGTH=512
```

### Security Considerations

**ACL Support**:
- Tag-based service filtering with ACL policies
- Metadata keys can be marked as sensitive (filtered in responses)
- Reserved metadata keys (e.g., `konsul_*`) for internal use only

**Validation**:
- Sanitize tag and metadata input
- Prevent injection attacks
- Limit payload size (DoS protection)

### Monitoring & Metrics

**New Metrics**:
```
konsul_service_tags_total{service="api-service"}
konsul_service_metadata_keys_total{service="api-service"}
konsul_catalog_queries_total{filter_type="tag|meta|combined"}
konsul_catalog_query_duration_seconds{filter_type="tag|meta|combined"}
```

### Example Use Cases

**Example 1: Canary Deployment**
```bash
# Register v1 (stable)
konsulctl service register api --tag version:v1.0.0 --tag stable

# Register v2 (canary)
konsulctl service register api --tag version:v2.0.0 --tag canary

# Query only stable instances
konsulctl service list --tag stable

# Query only canary instances
konsulctl service list --tag canary
```

**Example 2: Multi-Region Routing**
```bash
# Register services in different regions
konsulctl service register api --tag region:us-east-1 --address 10.1.0.50
konsulctl service register api --tag region:us-west-2 --address 10.2.0.50

# Query services in us-east-1
konsulctl service list --tag region:us-east-1
```

**Example 3: Protocol-Based Discovery**
```bash
# Register HTTP and gRPC services
konsulctl service register web --tag http --port 8080
konsulctl service register rpc --tag grpc --port 9090

# Service mesh queries only gRPC services
curl "http://localhost:8500/v1/catalog/services?tag=grpc"
```

**Example 4: Team Ownership**
```bash
# Register with ownership metadata
konsulctl service register api \
  --meta team:platform \
  --meta owner:alice@example.com \
  --meta oncall:#platform-oncall

# Query all platform team services
konsulctl service list --meta team:platform
```

## References

- [HashiCorp Consul - Service Tags](https://www.consul.io/docs/discovery/services#service-definition)
- [HashiCorp Consul - Service Metadata](https://www.consul.io/api-docs/catalog#meta)
- [Kubernetes Labels and Selectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
- [AWS Tags Best Practices](https://docs.aws.amazon.com/general/latest/gr/aws_tagging.html)
- [Service Mesh Interface (SMI)](https://smi-spec.io/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-28 | Konsul Team | Initial proposal |
