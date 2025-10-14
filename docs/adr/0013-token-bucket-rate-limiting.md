# ADR-0013: Token Bucket Rate Limiting

**Date**: 2024-09-28

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: security, performance, rate-limiting, abuse-prevention

## Context

Konsul provides public APIs for KV storage, service discovery, and other operations. Without rate limiting, the system is vulnerable to:

### Problems

1. **Denial of Service (DoS)**: Malicious actors can overwhelm the server
2. **Resource exhaustion**: Unbounded requests deplete CPU, memory, and network
3. **Fair usage**: Noisy neighbors affect other clients
4. **Cost control**: Cloud deployments with metered traffic
5. **Accidental abuse**: Buggy clients creating request loops
6. **API quality**: No protection against misbehaving integrations

### Requirements

**Effectiveness**:
- Protect against burst attacks
- Allow legitimate burst traffic
- Maintain average rate limits
- Per-client isolation

**Fairness**:
- Independent limits per client
- Support both IP and API key limiting
- Configurable rates and bursts

**Performance**:
- Minimal overhead (<1ms per request)
- Memory efficient
- No external dependencies
- Lock-free where possible

**Operational**:
- Easy to configure
- Prometheus metrics
- Admin APIs for management
- Automatic cleanup of idle limiters

**User Experience**:
- Clear error messages
- Rate limit headers
- Reset time information

## Decision

We will implement **Token Bucket rate limiting** with per-client isolation using an in-memory store.

### Why Token Bucket?

**Token Bucket Algorithm**:
- Tokens added at constant rate (e.g., 100/sec)
- Each request consumes 1 token
- Maximum tokens = burst size
- Request allowed if ≥1 token available
- Handles bursts naturally while maintaining average rate

**Advantages**:
1. **Simple to understand and implement**
2. **Burst-friendly**: Allows legitimate bursts
3. **Memory efficient**: Just a counter and timestamp per client
4. **Fast**: O(1) operations with minimal locking
5. **Industry standard**: Used by AWS, Google Cloud, etc.

### Architecture

```
┌─────────────────────────────────────────────┐
│           Rate Limit Service                │
├─────────────────────────────────────────────┤
│                                             │
│  ┌──────────────┐      ┌──────────────┐   │
│  │  IP Store    │      │ API Key Store│   │
│  │              │      │              │   │
│  │ 192.168.1.1  │      │ key-abc-123  │   │
│  │ → Limiter    │      │ → Limiter    │   │
│  │              │      │              │   │
│  │ 10.0.1.50    │      │ key-xyz-789  │   │
│  │ → Limiter    │      │ → Limiter    │   │
│  └──────────────┘      └──────────────┘   │
│                                             │
└─────────────────────────────────────────────┘
         ↑                    ↑
         │                    │
    Per-IP Limiting    Per-API-Key Limiting
```

### Implementation

**Limiter (Single Client)**:
```go
type Limiter struct {
    rate       float64   // tokens per second
    burst      int       // maximum burst size
    tokens     float64   // current tokens
    lastUpdate time.Time // last token update time
    mu         sync.Mutex
}

func (l *Limiter) Allow() bool {
    l.mu.Lock()
    defer l.mu.Unlock()

    now := time.Now()
    elapsed := now.Sub(l.lastUpdate).Seconds()

    // Add tokens based on elapsed time
    l.tokens += elapsed * l.rate
    if l.tokens > float64(l.burst) {
        l.tokens = float64(l.burst)
    }

    l.lastUpdate = now

    // Check if we have at least one token
    if l.tokens >= 1.0 {
        l.tokens -= 1.0
        return true
    }

    return false
}
```

**Store (Multiple Clients)**:
```go
type Store struct {
    limiters map[string]*Limiter // client ID → limiter
    rate     float64
    burst    int
    mu       sync.RWMutex
}

func (s *Store) Allow(clientID string) bool {
    limiter := s.GetLimiter(clientID) // Get or create
    return limiter.Allow()
}
```

**Service (Dual Strategy)**:
```go
type Service struct {
    config   Config
    ipStore  *Store    // Per-IP limiters
    keyStore *Store    // Per-API-key limiters
}

func (s *Service) AllowIP(ip string) bool {
    if !s.config.ByIP {
        return true
    }
    return s.ipStore.Allow(ip)
}

func (s *Service) AllowAPIKey(apiKey string) bool {
    if !s.config.ByAPIKey {
        return true
    }
    return s.keyStore.Allow(apiKey)
}
```

### Middleware Integration

```go
func RateLimitMiddleware(service *ratelimit.Service) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Extract identifier
        ip := c.IP()
        apiKey := extractAPIKey(c) // From header or JWT

        // Check IP limit
        if !service.AllowIP(ip) {
            metrics.RateLimitExceeded.WithLabelValues("ip").Inc()
            return c.Status(429).JSON(fiber.Map{
                "error":      "rate limit exceeded",
                "message":    "Too many requests. Please try again later.",
                "identifier": "ip:" + ip,
            })
        }

        // Check API key limit
        if apiKey != "" && !service.AllowAPIKey(apiKey) {
            metrics.RateLimitExceeded.WithLabelValues("apikey").Inc()
            return c.Status(429).JSON(fiber.Map{
                "error":      "rate limit exceeded",
                "message":    "Too many requests. Please try again later.",
                "identifier": "apikey:" + apiKey,
            })
        }

        metrics.RateLimitRequests.WithLabelValues("ip", "allowed").Inc()
        return c.Next()
    }
}
```

### Configuration

```bash
# Enable rate limiting
KONSUL_RATE_LIMIT_ENABLED=true

# Rate configuration
KONSUL_RATE_LIMIT_REQUESTS_PER_SEC=100.0  # Average rate
KONSUL_RATE_LIMIT_BURST=20                # Max burst size

# Limiting strategies
KONSUL_RATE_LIMIT_BY_IP=true      # Limit per IP address
KONSUL_RATE_LIMIT_BY_APIKEY=false # Limit per API key

# Cleanup
KONSUL_RATE_LIMIT_CLEANUP=5m      # Remove idle limiters
```

### Example Scenarios

**Scenario 1: Normal Traffic**
```
Rate: 100 req/sec, Burst: 20
Client makes 10 requests instantly → Allowed (burst capacity)
Then makes 100 req/sec sustained → Allowed
Then makes 150 req/sec → 50 rejected
```

**Scenario 2: Burst Traffic**
```
Rate: 100 req/sec, Burst: 20
Client idle for 10 seconds
Then makes 20 requests instantly → All allowed (burst used)
Then makes 21st request immediately → Rejected
Wait 1 second → 100 more tokens → Next 100 requests allowed
```

**Scenario 3: Multiple Clients**
```
IP 1: 100 req/sec → Allowed
IP 2: 100 req/sec → Allowed (independent limit)
IP 3: 150 req/sec → 50 rejected
```

### Cleanup Strategy

**Problem**: Over time, map grows with inactive clients

**Solution**: Periodic cleanup
```go
func (s *Store) cleanupExpired() {
    threshold := 5 * time.Minute // Remove after 5min idle

    for key, limiter := range s.limiters {
        if time.Since(limiter.lastUpdate) > threshold {
            delete(s.limiters, key)
        }
    }
}
```

**Benefits**:
- Memory bounded
- Automatic garbage collection
- No manual intervention

## Alternatives Considered

### Alternative 1: Leaky Bucket
- **Pros**:
  - Enforces strict constant rate
  - Smoother traffic output
  - Prevents bursts
- **Cons**:
  - Not burst-friendly
  - Penalizes legitimate burst traffic
  - More complex implementation
- **Reason for rejection**: Token bucket better for API use case

### Alternative 2: Fixed Window
- **Pros**:
  - Very simple (counter per time window)
  - Low memory usage
  - Easy to implement
- **Cons**:
  - Vulnerable to boundary attacks (2x rate at window edges)
  - Doesn't handle bursts well
  - Reset causes traffic spikes
- **Reason for rejection**: Boundary attack vulnerability

### Alternative 3: Sliding Window Log
- **Pros**:
  - No boundary attack vulnerability
  - Accurate rate calculation
  - Smooth enforcement
- **Cons**:
  - High memory usage (store each request timestamp)
  - Slower (must scan request log)
  - Complex to implement
- **Reason for rejection**: Memory and performance overhead

### Alternative 4: Sliding Window Counter
- **Pros**:
  - Hybrid approach (fixed + sliding)
  - Better than fixed window
  - Reasonable accuracy
- **Cons**:
  - More complex than token bucket
  - Still has edge cases
  - More memory than token bucket
- **Reason for rejection**: Token bucket simpler and sufficient

### Alternative 5: External Service (Redis)
- **Pros**:
  - Distributed rate limiting
  - Survives restarts
  - Shared across instances
- **Cons**:
  - External dependency
  - Network latency
  - Additional ops complexity
  - Cost
- **Reason for rejection**: In-memory sufficient; clustering will handle distribution

### Alternative 6: No Rate Limiting
- **Pros**:
  - No complexity
  - No performance overhead
  - Simpler codebase
- **Cons**:
  - Vulnerable to abuse
  - No fair usage
  - Not production-ready
- **Reason for rejection**: Unacceptable for production systems

## Consequences

### Positive
- **DoS protection**: Prevents resource exhaustion
- **Fair usage**: Each client gets independent limit
- **Burst-friendly**: Legitimate bursts allowed
- **Low overhead**: <1ms per request, minimal memory
- **No dependencies**: Pure in-memory, no Redis/etc.
- **Configurable**: Flexible rate and burst configuration
- **Observable**: Prometheus metrics for monitoring
- **Automatic cleanup**: Memory bounded over time
- **Dual strategy**: Support both IP and API key limiting

### Negative
- **Memory usage**: Grows with number of unique clients
- **Not distributed**: Each node has independent limits (until clustering)
- **State loss**: Limits reset on restart
- **No persistence**: No rate limit state saved
- **Cleanup threshold**: Hard-coded 5-minute idle threshold

### Neutral
- Need monitoring for rate limit violations
- May need to adjust rates based on usage patterns
- Cleanup interval affects memory usage
- IP-based limiting affected by proxies/NAT

## Implementation Notes

### Metrics

**Prometheus Metrics**:
```
# Total rate limit checks
konsul_rate_limit_requests_total{limiter_type="ip|apikey", status="allowed|denied"}

# Total rate limit violations
konsul_rate_limit_exceeded_total{limiter_type="ip|apikey"}

# Active clients being tracked
konsul_rate_limit_active_clients{limiter_type="ip|apikey"}
```

**Usage**:
```bash
# Check rate limit violations
sum(rate(konsul_rate_limit_exceeded_total[5m]))

# Active clients
konsul_rate_limit_active_clients{limiter_type="ip"}
```

### Response Format

**Rate Limited Response**:
```json
{
  "error": "rate limit exceeded",
  "message": "Too many requests. Please try again later.",
  "identifier": "ip:192.168.1.1"
}
```

**HTTP Status**: 429 Too Many Requests

**Headers** (future):
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1633024800
Retry-After: 1
```

### Admin Operations (Future)

**CLI Commands**:
```bash
# View rate limit stats
konsulctl ratelimit stats

# Reset specific client
konsulctl ratelimit reset --ip 192.168.1.1
konsulctl ratelimit reset --apikey key-abc-123

# Temporarily adjust limits
konsulctl ratelimit adjust --ip 192.168.1.1 --rate 200 --burst 50

# List top clients
konsulctl ratelimit top --by-requests
```

**API Endpoints**:
```
GET    /ratelimit/stats          - Get rate limiting statistics
POST   /ratelimit/reset/:id      - Reset rate limit for client
GET    /ratelimit/clients        - List active rate-limited clients
PUT    /ratelimit/config         - Update rate limit configuration
```

### Testing Strategy

**Unit Tests**:
```go
func TestTokenBucket(t *testing.T) {
    limiter := NewLimiter(10.0, 5) // 10/sec, burst 5

    // Burst should allow 5 requests
    for i := 0; i < 5; i++ {
        assert.True(t, limiter.Allow())
    }

    // 6th request should fail
    assert.False(t, limiter.Allow())

    // Wait 0.1 second (1 token added)
    time.Sleep(100 * time.Millisecond)
    assert.True(t, limiter.Allow())
}
```

**Load Tests**:
- Simulate concurrent clients
- Verify rate enforcement accuracy
- Check memory usage under load
- Test cleanup effectiveness

### Performance Characteristics

**Time Complexity**:
- `Allow()`: O(1) - constant time check
- `GetLimiter()`: O(1) - map lookup
- `Cleanup()`: O(n) - but infrequent

**Space Complexity**:
- Per-client overhead: ~96 bytes
- 10,000 clients: ~960KB
- 100,000 clients: ~9.6MB

**Latency**:
- Typical: <100µs (microseconds)
- 99th percentile: <1ms
- Lock contention minimal with independent limiters

### Tuning Guidelines

**Low Traffic (<10 req/sec)**:
```bash
KONSUL_RATE_LIMIT_REQUESTS_PER_SEC=50
KONSUL_RATE_LIMIT_BURST=10
```

**Medium Traffic (100 req/sec)**:
```bash
KONSUL_RATE_LIMIT_REQUESTS_PER_SEC=100
KONSUL_RATE_LIMIT_BURST=20  # Default
```

**High Traffic (1000 req/sec)**:
```bash
KONSUL_RATE_LIMIT_REQUESTS_PER_SEC=1000
KONSUL_RATE_LIMIT_BURST=200
```

**Per-User Limiting**:
```bash
KONSUL_RATE_LIMIT_BY_IP=false
KONSUL_RATE_LIMIT_BY_APIKEY=true
KONSUL_RATE_LIMIT_REQUESTS_PER_SEC=50  # Lower per user
```

### Security Considerations

**IP Spoofing**:
- Trust X-Forwarded-For only from trusted proxies
- Use rightmost external IP if behind proxies
- Combine with API key limiting for authenticated users

**Bypass Attacks**:
- Rotating IPs (botnets): Consider API key limits
- Distributed attacks: Deploy at edge (Cloudflare, etc.)

**Fair Usage**:
- Burst size prevents legitimate users from being blocked
- Per-client isolation ensures fairness

### Future Enhancements

**Phase 1** (Completed):
- ✅ Token bucket implementation
- ✅ Per-IP and per-API-key limiting
- ✅ Prometheus metrics
- ✅ Automatic cleanup

**Phase 2** (Planned):
- [ ] Admin API endpoints
- [ ] CLI commands for management
- [ ] Rate limit headers in responses
- [ ] Configurable cleanup threshold
- [ ] Whitelist/blacklist support

**Phase 3** (Future):
- [ ] Distributed rate limiting (with Raft clustering)
- [ ] Redis backend option
- [ ] Custom rate limits per API key
- [ ] Time-based limits (daytime vs nighttime rates)
- [ ] Quota management (daily/monthly limits)
- [ ] Anomaly detection and auto-blocking

## References

- [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
- [AWS API Gateway Throttling](https://docs.aws.amazon.com/apigateway/latest/developerguide/api-gateway-request-throttling.html)
- [Google Cloud Rate Limiting](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)
- [RFC 6585 - 429 Too Many Requests](https://tools.ietf.org/html/rfc6585)
- [Cloudflare Rate Limiting](https://developers.cloudflare.com/waf/rate-limiting-rules/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-28 | Konsul Team | Initial version |
