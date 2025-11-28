# ADR-0014: Rate Limiting Management API and Observability

**Date**: 2025-10-09

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: rate-limiting, api, cli, observability, operations

## Context

ADR-0013 established the core token bucket rate limiting implementation. While functional, operators lack tools to manage and observe rate limiting behavior in production.

### Current Limitations

1. **No visibility**: Cannot see which clients are being rate-limited
2. **No management**: Cannot reset or adjust limits without restart
3. **No client feedback**: Clients don't know when they can retry
4. **Limited debugging**: Hard to diagnose rate limit issues
5. **No exemptions**: Cannot whitelist specific clients
6. **No audit trail**: No history of rate limit violations

### Requirements

**Observability**:
- View active rate-limited clients
- See rate limit statistics
- Identify top offenders
- Historical violation data

**Management**:
- Reset individual client limits
- Temporarily adjust limits
- Whitelist/blacklist clients
- Configure custom per-client limits

**Client Experience**:
- Standard rate limit headers
- Retry-After information
- Clear error messages
- Current limit status

**Operations**:
- CLI commands for troubleshooting
- REST API for automation
- Prometheus metrics (already implemented)
- Audit logging

## Decision

We will implement a comprehensive **Rate Limiting Management System** with:

1. **Standard HTTP Headers** (RFC 6585, Draft RateLimit Headers)
2. **Admin REST API** for management
3. **CLI Commands** (konsulctl) for operations
4. **Enhanced Metrics and Dashboards**
5. **Audit Logging** for compliance

### Architecture

```
┌─────────────────────────────────────────────────────┐
│         Rate Limiting Management Layer              │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │   Headers    │  │  Admin API   │  │   CLI    │ │
│  │              │  │              │  │          │ │
│  │ X-RateLimit- │  │ GET /stats   │  │ konsulctl│ │
│  │   Limit      │  │ POST /reset  │  │ ratelimit│ │
│  │   Remaining  │  │ PUT /config  │  │          │ │
│  │   Reset      │  │              │  │          │ │
│  └──────────────┘  └──────────────┘  └──────────┘ │
│         │                  │               │        │
│         └──────────────────┴───────────────┘        │
│                         │                           │
│              ┌──────────▼──────────┐                │
│              │  Rate Limit Service │                │
│              │   (ADR-0013)        │                │
│              └─────────────────────┘                │
└─────────────────────────────────────────────────────┘
```

## Design

### 1. Standard HTTP Headers

Implement **RFC 6585** and **draft-ietf-httpapi-ratelimit-headers** standards:

**Response Headers**:
```http
X-RateLimit-Limit: 100           # Requests per window
X-RateLimit-Remaining: 45        # Requests remaining
X-RateLimit-Reset: 1696867200    # Unix timestamp when limit resets
Retry-After: 15                  # Seconds until retry (429 only)
```

**Example Responses**:

**Successful Request (200 OK)**:
```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1696867260
Content-Type: application/json

{"key": "mykey", "value": "myvalue"}
```

**Rate Limited (429 Too Many Requests)**:
```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1696867275
Retry-After: 15
Content-Type: application/json

{
  "error": "rate_limit_exceeded",
  "message": "Rate limit exceeded. Please retry after 15 seconds.",
  "limit": 100,
  "reset_at": "2025-10-09T12:34:35Z",
  "retry_after": 15
}
```

### 2. Admin REST API

**Endpoints**:

#### Statistics
```http
GET /admin/ratelimit/stats
```

**Response**:
```json
{
  "enabled": true,
  "config": {
    "requests_per_sec": 100.0,
    "burst": 20,
    "by_ip": true,
    "by_apikey": true
  },
  "statistics": {
    "total_requests": 1502345,
    "total_allowed": 1498234,
    "total_denied": 4111,
    "active_ip_limiters": 245,
    "active_apikey_limiters": 89
  },
  "timestamp": "2025-10-09T12:34:56Z"
}
```

#### List Active Clients
```http
GET /admin/ratelimit/clients?type=ip&limit=50&sort=violations
```

**Response**:
```json
{
  "clients": [
    {
      "identifier": "192.168.1.100",
      "type": "ip",
      "tokens_remaining": 5.3,
      "requests_allowed": 9823,
      "requests_denied": 156,
      "last_request": "2025-10-09T12:34:50Z",
      "status": "limited"
    },
    {
      "identifier": "10.0.1.50",
      "type": "ip",
      "tokens_remaining": 18.7,
      "requests_allowed": 4520,
      "requests_denied": 0,
      "last_request": "2025-10-09T12:34:55Z",
      "status": "ok"
    }
  ],
  "total": 245,
  "page": 1,
  "limit": 50
}
```

#### Get Client Status
```http
GET /admin/ratelimit/client/:type/:id
```

**Example**: `GET /admin/ratelimit/client/ip/192.168.1.100`

**Response**:
```json
{
  "identifier": "192.168.1.100",
  "type": "ip",
  "config": {
    "rate": 100.0,
    "burst": 20
  },
  "current": {
    "tokens_remaining": 5.3,
    "requests_allowed": 9823,
    "requests_denied": 156,
    "last_request": "2025-10-09T12:34:50Z",
    "first_seen": "2025-10-09T08:00:00Z"
  },
  "violations": [
    {
      "timestamp": "2025-10-09T12:34:45Z",
      "endpoint": "/kv/config",
      "remaining_tokens": 0
    }
  ]
}
```

#### Reset Client Limit
```http
POST /admin/ratelimit/reset
Content-Type: application/json

{
  "type": "ip",
  "identifier": "192.168.1.100"
}
```

**Response**:
```json
{
  "success": true,
  "message": "Rate limit reset for ip:192.168.1.100",
  "timestamp": "2025-10-09T12:35:00Z"
}
```

#### Adjust Client Limit
```http
PUT /admin/ratelimit/client/:type/:id
Content-Type: application/json

{
  "rate": 200.0,
  "burst": 50,
  "duration": "1h"  // Temporary, reverts after duration
}
```

**Response**:
```json
{
  "success": true,
  "message": "Custom rate limit applied",
  "identifier": "ip:192.168.1.100",
  "config": {
    "rate": 200.0,
    "burst": 50,
    "expires_at": "2025-10-09T13:35:00Z"
  }
}
```

#### Whitelist Client
```http
POST /admin/ratelimit/whitelist
Content-Type: application/json

{
  "type": "ip",
  "identifier": "10.0.1.10",
  "reason": "Internal monitoring system",
  "expires_at": "2025-12-31T23:59:59Z"  // Optional
}
```

#### Blacklist Client
```http
POST /admin/ratelimit/blacklist
Content-Type: application/json

{
  "type": "ip",
  "identifier": "203.0.113.50",
  "reason": "Malicious activity detected",
  "duration": "24h"
}
```

#### Update Global Configuration
```http
PUT /admin/ratelimit/config
Content-Type: application/json

{
  "requests_per_sec": 150.0,
  "burst": 30,
  "by_ip": true,
  "by_apikey": true
}
```

### 3. CLI Commands

**konsulctl ratelimit** subcommands:

#### View Statistics
```bash
# Overall statistics
konsulctl ratelimit stats

# Output
Rate Limiting Statistics:
  Status:           Enabled
  Requests/sec:     100.0
  Burst:            20
  By IP:            Yes
  By API Key:       Yes

Statistics (last 24h):
  Total Requests:   1,502,345
  Allowed:          1,498,234 (99.73%)
  Denied:           4,111 (0.27%)

Active Clients:
  IP Limiters:      245
  API Key Limiters: 89
```

#### List Top Offenders
```bash
# Top clients by violations
konsulctl ratelimit top --by violations --limit 10

# Output
Top 10 Clients by Violations:
┌─────────────────┬──────┬──────────┬────────────┬────────────────────┐
│ Identifier      │ Type │ Allowed  │ Denied     │ Last Request       │
├─────────────────┼──────┼──────────┼────────────┼────────────────────┤
│ 192.168.1.100   │ IP   │ 9,823    │ 156        │ 2s ago             │
│ 10.0.5.230      │ IP   │ 5,120    │ 89         │ 15s ago            │
│ key-abc-123     │ Key  │ 45,230   │ 67         │ 1m ago             │
└─────────────────┴──────┴──────────┴────────────┴────────────────────┘
```

#### View Client Details
```bash
# Specific client status
konsulctl ratelimit status --ip 192.168.1.100

# Output
Client: 192.168.1.100 (IP)
  Status:           Limited
  Rate:             100.0 req/sec
  Burst:            20
  Tokens Remaining: 5.3

Counters:
  Requests Allowed: 9,823
  Requests Denied:  156

Activity:
  First Seen:       4h ago
  Last Request:     2s ago

Recent Violations:
  - 2s ago  → /kv/config (0 tokens)
  - 5s ago  → /services/ (0 tokens)
  - 12s ago → /kv/app (0 tokens)
```

#### Reset Client Limit
```bash
# Reset specific client
konsulctl ratelimit reset --ip 192.168.1.100

# Output
✓ Rate limit reset for ip:192.168.1.100

# Reset all clients
konsulctl ratelimit reset --all
```

#### Adjust Client Limit
```bash
# Temporarily increase limit
konsulctl ratelimit adjust --ip 192.168.1.100 \
  --rate 200 \
  --burst 50 \
  --duration 1h

# Output
✓ Custom rate limit applied to ip:192.168.1.100
  Rate:     200 req/sec
  Burst:    50
  Expires:  in 1 hour
```

#### Whitelist/Blacklist
```bash
# Whitelist client (no rate limit)
konsulctl ratelimit whitelist add --ip 10.0.1.10 \
  --reason "Internal monitoring"

# Blacklist client (block all requests)
konsulctl ratelimit blacklist add --ip 203.0.113.50 \
  --reason "Malicious activity" \
  --duration 24h

# List whitelisted clients
konsulctl ratelimit whitelist list

# Remove from blacklist
konsulctl ratelimit blacklist remove --ip 203.0.113.50
```

#### Watch Mode
```bash
# Live monitoring of rate limit events
konsulctl ratelimit watch

# Output (live updates)
12:34:56 [ALLOWED] ip:192.168.1.100 → /kv/config (18 tokens)
12:34:57 [DENIED]  ip:10.0.5.230   → /services/ (0 tokens) ⚠️
12:34:58 [ALLOWED] key:key-abc-123 → /register (45 tokens)
12:34:59 [DENIED]  ip:10.0.5.230   → /services/ (0 tokens) ⚠️
```

### 4. Enhanced Data Structures

**Extended Limiter State**:
```go
type Limiter struct {
    // Existing fields
    rate       float64
    burst      int
    tokens     float64
    lastUpdate time.Time

    // New fields for observability
    requestsAllowed  uint64
    requestsDenied   uint64
    firstSeen        time.Time
    lastRequest      time.Time
    violations       []Violation
    customConfig     *CustomConfig  // Override default config

    mu sync.Mutex
}

type Violation struct {
    Timestamp time.Time
    Endpoint  string
    Remaining float64
}

type CustomConfig struct {
    Rate      float64
    Burst     int
    ExpiresAt time.Time
}
```

**Whitelist/Blacklist**:
```go
type AccessList struct {
    whitelisted map[string]*WhitelistEntry
    blacklisted map[string]*BlacklistEntry
    mu          sync.RWMutex
}

type WhitelistEntry struct {
    Identifier string
    Type       string // "ip" or "apikey"
    Reason     string
    AddedAt    time.Time
    ExpiresAt  *time.Time
}

type BlacklistEntry struct {
    Identifier string
    Type       string
    Reason     string
    AddedAt    time.Time
    ExpiresAt  time.Time
}
```

### 5. Middleware Enhancement

**Updated Middleware**:
```go
func RateLimitMiddleware(service *ratelimit.Service) fiber.Handler {
    return func(c *fiber.Ctx) error {
        ip := c.IP()
        apiKey := extractAPIKey(c)

        // Check blacklist
        if service.IsBlacklisted(ip, apiKey) {
            return c.Status(403).JSON(fiber.Map{
                "error": "forbidden",
                "message": "Access denied",
            })
        }

        // Check whitelist (bypass rate limiting)
        if service.IsWhitelisted(ip, apiKey) {
            return c.Next()
        }

        // Check rate limits
        allowed, remaining, resetAt := service.CheckLimit(ip, apiKey)

        // Add headers to all responses
        c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", service.Limit()))
        c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
        c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

        if !allowed {
            retryAfter := int(time.Until(resetAt).Seconds())
            c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))

            return c.Status(429).JSON(fiber.Map{
                "error": "rate_limit_exceeded",
                "message": fmt.Sprintf("Rate limit exceeded. Retry after %d seconds.", retryAfter),
                "limit": service.Limit(),
                "reset_at": resetAt.Format(time.RFC3339),
                "retry_after": retryAfter,
            })
        }

        return c.Next()
    }
}
```

## Alternatives Considered

### Alternative 1: No Management API (Manual Config Only)
- **Pros**: Simpler, no additional code
- **Cons**: Cannot respond to incidents, poor operational experience
- **Reason for rejection**: Operations need runtime management

### Alternative 2: Separate Management Service
- **Pros**: Microservice architecture, independent scaling
- **Cons**: Additional complexity, network latency, more infrastructure
- **Reason for rejection**: Overkill for rate limiting management

### Alternative 3: Configuration File Only
- **Pros**: Simple, version-controlled, declarative
- **Cons**: Requires restart, no runtime changes, poor incident response
- **Reason for rejection**: Need runtime adjustment capability

### Alternative 4: Redis-Based Management
- **Pros**: Distributed, persistent state, shared across instances
- **Cons**: External dependency, adds latency, operational complexity
- **Reason for rejection**: In-memory sufficient; clustering will handle distribution

## Consequences

### Positive
- **Observability**: Full visibility into rate limiting behavior
- **Operations**: Runtime management without restarts
- **Client UX**: Standard headers help clients implement retry logic
- **Debugging**: Easy to identify and troubleshoot rate limit issues
- **Flexibility**: Whitelist/blacklist for exceptions
- **Audit**: Historical data for compliance and analysis
- **Standards compliant**: RFC 6585 headers

### Negative
- **Complexity increase**: More code to maintain
- **Memory overhead**: Storing violation history
- **API surface**: More endpoints to secure
- **State management**: Need to track more per-client data
- **Performance**: Additional header generation overhead

### Neutral
- Admin API needs authentication (ACL system)
- CLI depends on API availability
- Headers add ~100 bytes per response

## Implementation Notes

### Phase 1: Headers ✅ **COMPLETED**
- ✅ Implement header calculation (`GetHeaders()` method in Limiter)
- ✅ Add to middleware (both `RateLimitMiddleware` and `RateLimitWithConfig`)
- ✅ Test with various scenarios (12 middleware tests)
- ✅ Update documentation

**Implementation**: `internal/ratelimit/limiter.go:536`, `internal/middleware/ratelimit.go`

### Phase 2: Admin API ✅ **COMPLETED**
- ✅ Statistics endpoint (`GET /admin/ratelimit/stats`)
- ✅ Client listing (`GET /admin/ratelimit/clients`)
- ✅ Reset/adjust operations (`POST /admin/ratelimit/reset/*`, `PUT /admin/ratelimit/client/:type/:id`)
- ✅ Whitelist/blacklist (`GET/POST/DELETE /admin/ratelimit/whitelist`, `/blacklist`)

**Implementation**: `internal/handlers/ratelimit.go` (35 tests)

### Phase 3: CLI Commands ✅ **COMPLETED**
- ✅ Basic commands (stats, config, clients, client, reset)
- ✅ Advanced commands (update, adjust, whitelist, blacklist)
- ⏳ Watch mode (planned for future)
- ✅ Output formatting (table/text output)

**Implementation**: `cmd/konsulctl/ratelimit_commands.go` (15 client methods, 9 CLI commands)

**Available Commands:**
```bash
konsulctl ratelimit stats                    # View statistics
konsulctl ratelimit config                   # View configuration
konsulctl ratelimit clients [--type TYPE]    # List active clients
konsulctl ratelimit client <identifier>      # Client status
konsulctl ratelimit reset <ip|apikey|all>    # Reset limits
konsulctl ratelimit update --rate N --burst N # Update config
konsulctl ratelimit adjust --type TYPE --id ID --rate N --burst N
konsulctl ratelimit whitelist <list|add|remove>
konsulctl ratelimit blacklist <list|add|remove>
```

### Phase 4: Enhanced Metrics (Future)
- [ ] Grafana dashboard
- [ ] Alert rules
- [ ] Violation tracking
- [ ] Client activity metrics

### Configuration

```bash
# Enable management features
KONSUL_RATE_LIMIT_MANAGEMENT_ENABLED=true

# Violation history
KONSUL_RATE_LIMIT_HISTORY_SIZE=100  # Per client

# Admin API
KONSUL_RATE_LIMIT_ADMIN_PATH=/admin/ratelimit

# Headers
KONSUL_RATE_LIMIT_HEADERS_ENABLED=true
```

### Security Considerations

**Admin API Protection**:
- Require authentication (JWT or API key)
- ACL-based authorization (admin role)
- Audit all management operations
- Rate limit the admin API itself

**Blacklist Safety**:
- Prevent self-blacklisting
- Require confirmation for permanent blocks
- Automatic expiry by default

### Testing Strategy

**Unit Tests**:
- Header calculation accuracy
- Whitelist/blacklist logic
- Custom rate limit application

**Integration Tests**:
- Full admin API workflow
- CLI command execution
- Header presence in responses

**Load Tests**:
- Header overhead measurement
- Management API under load
- Memory impact of violation history

## References

- [RFC 6585 - Additional HTTP Status Codes](https://tools.ietf.org/html/rfc6585)
- [Draft RateLimit Header Fields](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-ratelimit-headers)
- [GitHub REST API Rate Limiting](https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting)
- [Stripe API Rate Limits](https://stripe.com/docs/rate-limits)
- [ADR-0013: Token Bucket Rate Limiting](./0013-token-bucket-rate-limiting.md)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-09 | Konsul Team | Initial proposal |
| 2025-11-28 | Konsul Team | Phase 1-3 completed, status changed to Accepted |
