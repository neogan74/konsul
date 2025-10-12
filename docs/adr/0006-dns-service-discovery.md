# ADR-0006: DNS Interface for Service Discovery

**Date**: 2024-09-25

**Status**: Accepted

**Deciders**: Konsul Core Team

**Tags**: service-discovery, dns, networking, integration

## Context

While Konsul provides HTTP APIs for service discovery, many applications and infrastructure tools expect DNS-based service discovery. Requirements:

- Standard DNS protocol support (UDP/TCP)
- Query services by name (e.g., `redis.service.consul`)
- Return service addresses and ports via SRV records
- Support for A/AAAA records for simple IP lookups
- Low latency (< 5ms typical)
- Compatible with existing DNS infrastructure
- Optional feature (can be disabled)

DNS is ubiquitous and many tools (databases, proxies, load balancers) support DNS-based discovery out of the box.

## Decision

We will implement a **built-in DNS server** for Konsul that:

- Runs on port 8600 (default, configurable)
- Serves DNS queries for registered services
- Uses domain pattern: `<service-name>.service.<domain>`
- Returns SRV records with service address and port
- Returns A records with service IP addresses
- Supports both UDP and TCP DNS protocols
- Reads from the same ServiceStore as HTTP API
- Can be disabled via configuration

### Query Patterns

**Service lookup (SRV)**:
```
redis.service.consul → SRV record with address:port
```

**Address lookup (A)**:
```
redis.service.consul → A record with IP address
```

### Design
- Single UDP/TCP server on configurable port (default 8600)
- Uses same ServiceStore for consistency
- Returns all healthy instances for a service
- Random shuffle for client-side load balancing
- Configurable domain suffix (default: "consul" for compatibility)

## Alternatives Considered

### Alternative 1: External DNS Integration (CoreDNS plugin)
- **Pros**:
  - Leverage existing DNS infrastructure
  - More DNS features (caching, forwarding, etc.)
  - Separate concerns
  - Plugin ecosystem
- **Cons**:
  - Requires external dependency
  - More complex deployment
  - Additional operational burden
  - Synchronization between Konsul and DNS server
- **Reason for rejection**: Built-in DNS simpler for users; fewer moving parts

### Alternative 2: HTTP DNS (DNS-over-HTTPS)
- **Pros**:
  - Modern protocol
  - Uses existing HTTP infrastructure
  - Better security (TLS)
  - Firewall-friendly
- **Cons**:
  - Not widely supported by applications
  - More complex client integration
  - Higher latency than UDP DNS
  - Requires HTTPS setup
- **Reason for rejection**: Standard DNS more compatible with existing tools

### Alternative 3: No DNS Support (HTTP API Only)
- **Pros**:
  - Simpler implementation
  - Fewer ports to manage
  - One interface to maintain
- **Cons**:
  - Requires application changes
  - Less compatible with existing tools
  - Manual integration needed
  - Not drop-in replacement for Consul
- **Reason for rejection**: DNS integration critical for adoption and compatibility

### Alternative 4: mDNS (Multicast DNS)
- **Pros**:
  - Zero-configuration discovery
  - No server needed
  - Works across local network
- **Cons**:
  - Limited to local network (no cross-subnet)
  - No central coordination
  - Not suitable for datacenter deployments
  - Different use case than traditional DNS
- **Reason for rejection**: Use case doesn't match requirements; need centralized registry

## Consequences

### Positive
- Drop-in DNS compatibility with HashiCorp Consul
- Applications can discover services without code changes
- Works with existing tools (databases, proxies, etc.)
- Low latency for local queries
- No external dependencies
- Consistent with HTTP API (same data source)
- Simple client-side load balancing (multiple A records)

### Negative
- Additional port to manage (8600)
- DNS protocol implementation to maintain
- UDP packet size limits (512 bytes default, 4096 with EDNS0)
- Must handle DNS caching (TTL settings)
- Cannot return service metadata (only address/port)
- Need to handle DNS-specific errors

### Neutral
- DNS is UDP-first (need TCP fallback for large responses)
- Need to implement health checking integration
- TTL configuration affects freshness vs load trade-off
- DNS query patterns differ from HTTP API

## Implementation Notes

### Configuration
```go
DNS: DNSConfig{
    Enabled: true,
    Host:    "",     // Listen on all interfaces
    Port:    8600,   // Standard Consul DNS port
    Domain:  "consul",
}
```

### Query Format
```
<service-name>.service.<domain>
```

Examples:
- `redis.service.consul` → All Redis instances
- `web.service.consul` → All web service instances

### Record Types

**SRV Record**:
```
_<service>._tcp.<service>.service.consul
Priority: 1
Weight: 1
Port: <service-port>
Target: <service-address>
```

**A Record**:
```
<service>.service.consul → IP address(es)
```

### Features
- Return all healthy service instances
- Random shuffle for load distribution
- Configurable TTL (default: 0 for no caching)
- Support for both UDP and TCP
- EDNS0 support for larger responses

### Integration Points
- Uses `ServiceStore.Get()` to retrieve services
- No separate cache (always fresh)
- Health check integration (future)
- Metrics for DNS queries

### Performance Considerations
- Keep responses small (UDP 512 byte limit)
- Limit number of returned instances if needed
- Monitor query latency
- Consider DNS caching strategy

### Future Enhancements
- Health check integration (only return healthy instances)
- Configurable TTL per service
- DNS caching layer
- Support for service tags in queries
- Prepared queries support
- PTR records for reverse lookups

## References

- [RFC 1035 - Domain Names](https://tools.ietf.org/html/rfc1035)
- [RFC 2782 - DNS SRV](https://tools.ietf.org/html/rfc2782)
- [Consul DNS Interface](https://www.consul.io/docs/discovery/dns)
- [miekg/dns Go library](https://github.com/miekg/dns)
- [Konsul DNS package](../../internal/dns/)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2024-09-25 | Konsul Team | Initial version |
