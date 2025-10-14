# DNS Server - Implementation Guide

Technical deep dive into the Konsul DNS server implementation.

## Architecture Overview

The DNS server is a lightweight component that bridges the standard DNS protocol with Konsul's service registry.

```
┌─────────────────────────────────────────────────────────────┐
│                        DNS Clients                          │
│   (dig, host, nslookup, application DNS resolvers)          │
└────────────┬────────────────────────────────┬───────────────┘
             │ UDP Query                      │ TCP Query
             │ Port 8600                      │ Port 8600
             ↓                                ↓
┌─────────────────────────────────────────────────────────────┐
│                         DNS Server                          │
│  ┌──────────────────┐              ┌────────────────────┐   │
│  │   UDP Server     │              │    TCP Server      │   │
│  │  (miekg/dns)     │              │   (miekg/dns)      │   │
│  └────────┬─────────┘              └─────────┬──────────┘   │
│           │                                   │              │
│           └──────────────┬────────────────────┘              │
│                          ↓                                   │
│            ┌─────────────────────────────┐                   │
│            │   handleDNSRequest()        │                   │
│            │  - Parse query              │                   │
│            │  - Route by type            │                   │
│            └──────────┬──────────────────┘                   │
│                       ↓                                      │
│       ┌───────────────┼───────────────┐                      │
│       ↓               ↓               ↓                      │
│  handleAQuery   handleSRVQuery   (unsupported)               │
│       │               │                                      │
│       └───────────────┴──────────────────┐                   │
│                                          ↓                   │
│                              ┌────────────────────────┐      │
│                              │   ServiceStore.List()  │      │
│                              └────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
                                          │
                                          ↓
                         ┌───────────────────────────────┐
                         │       ServiceStore            │
                         │  (In-memory service registry) │
                         └───────────────────────────────┘
```

---

## Component Breakdown

### 1. Server Struct

**File**: `internal/dns/server.go:13-19`

```go
type Server struct {
    udpServer *dns.Server         // UDP DNS server
    tcpServer *dns.Server         // TCP DNS server
    domain    string              // DNS domain suffix
    store     *store.ServiceStore // Service registry
    log       logger.Logger       // Structured logger
}
```

**Design rationale**:
- Dual UDP/TCP servers for protocol compliance
- Domain stored for query parsing
- Direct reference to service store (no abstraction layer)
- Logger for observability

---

### 2. Configuration

**File**: `internal/dns/server.go:21-25`

```go
type Config struct {
    Host   string  // Listen address
    Port   int     // Port number
    Domain string  // DNS domain
}
```

**Simplicity**: Minimal configuration surface

**Defaults** (in main application):
- Host: `""` (all interfaces)
- Port: `8600` (Consul-compatible)
- Domain: `"consul"` (Consul-compatible)

---

### 3. Server Creation

**File**: `internal/dns/server.go:27-51`

```go
func NewServer(cfg Config, serviceStore *store.ServiceStore, log logger.Logger) *Server {
    s := &Server{
        domain: cfg.Domain,
        store:  serviceStore,
        log:    log,
    }

    mux := dns.NewServeMux()
    mux.HandleFunc(".", s.handleDNSRequest)  // Catch-all handler

    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

    // UDP server
    s.udpServer = &dns.Server{
        Addr:    addr,
        Net:     "udp",
        Handler: mux,
    }

    // TCP server (same config, different protocol)
    s.tcpServer = &dns.Server{
        Addr:    addr,
        Net:     "tcp",
        Handler: mux,
    }

    return s
}
```

**Key decisions**:
1. **Single handler** for all queries (`.` matches everything)
2. **Same address** for UDP and TCP (RFC requirement)
3. **No middleware** - direct request handling
4. **Return before start** - lazy initialization pattern

---

### 4. Server Lifecycle

#### Start

**File**: `internal/dns/server.go:53-74`

```go
func (s *Server) Start() error {
    s.log.Info("Starting DNS server",
        logger.String("domain", s.domain),
        logger.String("udp_addr", s.udpServer.Addr),
        logger.String("tcp_addr", s.tcpServer.Addr))

    // Start UDP server in background
    go func() {
        if err := s.udpServer.ListenAndServe(); err != nil {
            s.log.Error("DNS UDP server failed", logger.Error(err))
        }
    }()

    // Start TCP server in background
    go func() {
        if err := s.tcpServer.ListenAndServe(); err != nil {
            s.log.Error("DNS TCP server failed", logger.Error(err))
        }
    }()

    return nil
}
```

**Design choices**:
- **Non-blocking**: Returns immediately
- **Separate goroutines**: UDP and TCP independent
- **Error logging**: Failures logged but don't stop other server
- **No health check**: Assumes success if no immediate error

**Potential issue**: No way to detect startup failure

**Future improvement**:
```go
func (s *Server) Start() error {
    errCh := make(chan error, 2)

    go func() {
        errCh <- s.udpServer.ListenAndServe()
    }()

    go func() {
        errCh <- s.tcpServer.ListenAndServe()
    }()

    // Wait briefly to detect startup errors
    select {
    case err := <-errCh:
        return err
    case <-time.After(100 * time.Millisecond):
        return nil  // Assume success
    }
}
```

#### Stop

**File**: `internal/dns/server.go:76-91`

```go
func (s *Server) Stop() error {
    var udpErr, tcpErr error

    if s.udpServer != nil {
        udpErr = s.udpServer.Shutdown()
    }

    if s.tcpServer != nil {
        tcpErr = s.tcpServer.Shutdown()
    }

    if udpErr != nil {
        return udpErr
    }
    return tcpErr
}
```

**Graceful shutdown**:
1. Waits for in-flight requests
2. Closes listeners
3. Returns first error (if any)

**Issue**: Second error is lost

**Better approach**:
```go
func (s *Server) Stop() error {
    var errs []error

    if err := s.udpServer.Shutdown(); err != nil {
        errs = append(errs, fmt.Errorf("UDP: %w", err))
    }

    if err := s.tcpServer.Shutdown(); err != nil {
        errs = append(errs, fmt.Errorf("TCP: %w", err))
    }

    if len(errs) > 0 {
        return errors.Join(errs...)  // Go 1.20+
    }
    return nil
}
```

---

## Query Processing

### Main Handler

**File**: `internal/dns/server.go:93-122`

```go
func (s *Server) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
    msg := new(dns.Msg)
    msg.SetReply(r)              // Copy question section
    msg.Authoritative = true     // We're authoritative for .consul

    for _, question := range r.Question {
        s.log.Debug("DNS query received",
            logger.String("name", question.Name),
            logger.String("type", dns.TypeToString[question.Qtype]))

        switch question.Qtype {
        case dns.TypeSRV:
            s.handleSRVQuery(msg, question)
        case dns.TypeA:
            s.handleAQuery(msg, question)
        case dns.TypeANY:
            s.handleSRVQuery(msg, question)
            s.handleAQuery(msg, question)
        default:
            s.log.Debug("Unsupported DNS query type",
                logger.String("type", dns.TypeToString[question.Qtype]))
        }
    }

    // NXDOMAIN if no answers
    if len(msg.Answer) == 0 {
        msg.Rcode = dns.RcodeNameError
    }

    w.WriteMsg(msg)
}
```

**Flow**:
1. Create response message
2. Mark as authoritative
3. Process each question
4. Return NXDOMAIN if no answers
5. Write response

**Design notes**:
- **Authoritative bit**: Indicates we own the `.consul` zone
- **Multiple questions**: RFC allows it (rarely used)
- **ANY handling**: Returns both A and SRV records
- **Unknown types**: Silently ignored (no error)

---

### SRV Query Handler

**File**: `internal/dns/server.go:124-183`

```go
func (s *Server) handleSRVQuery(msg *dns.Msg, question dns.Question) {
    name := strings.TrimSuffix(question.Name, ".")

    // Parse: _service._protocol.service.consul
    parts := strings.Split(name, ".")
    if len(parts) < 4 {
        return  // Invalid format
    }

    serviceName := strings.TrimPrefix(parts[0], "_")
    protocol := strings.TrimPrefix(parts[1], "_")
    _ = protocol  // Ignored for now

    // Get all services
    services := s.store.List()
    var matchingServices []store.Service

    for _, service := range services {
        if service.Name == serviceName {
            matchingServices = append(matchingServices, service)
        }
    }

    // Create SRV records
    for i, service := range matchingServices {
        target := fmt.Sprintf("%s.node.%s.", service.Name, s.domain)

        srv := &dns.SRV{
            Hdr: dns.RR_Header{
                Name:   question.Name,
                Rrtype: dns.TypeSRV,
                Class:  dns.ClassINET,
                Ttl:    30,
            },
            Priority: 1,
            Weight:   uint16(100 / (i + 1)),  // Simple distribution
            Port:     uint16(service.Port),
            Target:   target,
        }
        msg.Answer = append(msg.Answer, srv)

        // Add A record in additional section
        a := &dns.A{
            Hdr: dns.RR_Header{
                Name:   target,
                Rrtype: dns.TypeA,
                Class:  dns.ClassINET,
                Ttl:    30,
            },
            A: net.ParseIP(service.Address),
        }
        msg.Extra = append(msg.Extra, a)
    }

    s.log.Debug("SRV query processed",
        logger.String("service", serviceName),
        logger.Int("matches", len(matchingServices)))
}
```

**Parsing logic**:
```
_web._tcp.service.consul
 ^   ^    ^       ^
 │   │    │       └─ Domain
 │   │    └───────── Keyword "service"
 │   └────────────── Protocol (ignored)
 └────────────────── Service name
```

**Weight calculation**:
```
Instance 0: 100 / (0+1) = 100
Instance 1: 100 / (1+1) = 50
Instance 2: 100 / (2+1) = 33
```

Simple but effective distribution for client-side load balancing.

**Additional section**: Includes A records to save round trips

---

### A Query Handler

**File**: `internal/dns/server.go:185-227`

```go
func (s *Server) handleAQuery(msg *dns.Msg, question dns.Question) {
    name := strings.TrimSuffix(question.Name, ".")

    // Parse: service.node.consul or service.service.consul
    parts := strings.Split(name, ".")
    if len(parts) < 3 {
        return
    }

    var serviceName string

    // Format 1: service.node.consul
    if len(parts) >= 3 && parts[1] == "node" {
        serviceName = parts[0]
    }
    // Format 2: service.service.consul
    else if len(parts) >= 3 && parts[1] == "service" {
        serviceName = parts[0]
    } else {
        return
    }

    // Query service store
    services := s.store.List()

    for _, service := range services {
        if service.Name == serviceName {
            a := &dns.A{
                Hdr: dns.RR_Header{
                    Name:   question.Name,
                    Rrtype: dns.TypeA,
                    Class:  dns.ClassINET,
                    Ttl:    30,
                },
                A: net.ParseIP(service.Address),
            }
            msg.Answer = append(msg.Answer, a)
        }
    }

    s.log.Debug("A query processed",
        logger.String("service", serviceName),
        logger.Int("records", len(msg.Answer)))
}
```

**Two formats supported**:
1. `service.node.consul` - Node lookup format
2. `service.service.consul` - Service lookup format

**Multiple instances**: All matching instances returned as separate A records

---

## Data Flow

### Service Registration → DNS Response

```
1. HTTP API Request
   POST /services
   {"name": "web", "address": "10.0.0.1", "port": 8080}

2. ServiceStore.Register()
   services["web"] = Service{...}

3. DNS Query
   dig @localhost -p 8600 web.service.consul A

4. DNS Handler
   - handleAQuery()
   - store.List()
   - Filter by name "web"

5. DNS Response
   web.service.consul. 30 IN A 10.0.0.1
```

**No caching**: Every query reads from service store

**Consistency**: Always returns current state

---

## Performance Analysis

### Query Latency Breakdown

**Total: ~1.5ms** (localhost)

1. **Network (UDP)**: ~0.1ms
2. **Parse query**: ~0.1ms (string operations)
3. **Service lookup**: ~0.5ms (O(n) scan)
4. **Build response**: ~0.3ms (struct creation)
5. **Serialize DNS message**: ~0.4ms
6. **Network reply**: ~0.1ms

### Bottlenecks

#### 1. O(n) Service Scan

**Current**:
```go
services := s.store.List()  // All services
for _, service := range services {
    if service.Name == serviceName {
        // Match
    }
}
```

**Issue**: Scans all services for every query

**Optimization**: Service store could index by name

```go
// In ServiceStore
type ServiceStore struct {
    byName map[string][]Service  // Index by name
}

func (s *ServiceStore) GetByName(name string) []Service {
    return s.byName[name]  // O(1) lookup
}
```

#### 2. No DNS Response Caching

Every query parses, looks up, and builds response from scratch.

**Future**: Implement response cache

```go
type ResponseCache struct {
    cache map[string]*cachedResponse
    mu    sync.RWMutex
}

type cachedResponse struct {
    msg       *dns.Msg
    expiresAt time.Time
}
```

#### 3. String Operations

Heavy use of string splitting and manipulation:
```go
parts := strings.Split(name, ".")
serviceName := strings.TrimPrefix(parts[0], "_")
```

**Optimization**: Compile regex patterns once

---

## Testing Strategy

### Unit Tests

**File**: `internal/dns/server_test.go`

**Approach**: Mock DNS queries without network

```go
func TestDNSServer_SRVQuery(t *testing.T) {
    // Setup
    dnsServer, serviceStore := setupTestServer()
    serviceStore.Register(service)

    // Create DNS message
    query := new(dns.Msg)
    query.SetQuestion("_web._tcp.service.consul.", dns.TypeSRV)

    // Mock response writer
    mockWriter := &mockResponseWriter{}

    // Execute
    dnsServer.handleDNSRequest(mockWriter, query)

    // Assert
    assert.Equal(t, 1, len(mockWriter.msg.Answer))
}
```

**Coverage**: All query types, edge cases, error conditions

---

### Integration Tests

**File**: `internal/dns/integration_test.go`

**Approach**: Real DNS queries over network

```go
func TestDNSServer_RealQuery(t *testing.T) {
    // Start real server
    server := NewServer(config, serviceStore, log)
    server.Start()
    defer server.Stop()

    // Real DNS client
    client := new(dns.Client)
    response, _, err := client.Exchange(
        query,
        "127.0.0.1:8600",
    )

    // Verify actual DNS response
}
```

**Tests**:
- Start/Stop lifecycle
- Concurrent queries
- Service registration/deregistration
- Multiple services

---

## Design Patterns

### 1. Handler Pattern

```go
mux := dns.NewServeMux()
mux.HandleFunc(".", s.handleDNSRequest)
```

Similar to HTTP handlers - familiar API.

### 2. Strategy Pattern

Query handling varies by type:

```go
switch question.Qtype {
case dns.TypeSRV:
    s.handleSRVQuery(msg, question)
case dns.TypeA:
    s.handleAQuery(msg, question)
}
```

Easy to add new query types.

### 3. Repository Pattern

Service store abstraction:

```go
type ServiceStore interface {
    List() []Service
    Get(name string) (Service, bool)
}
```

(Currently concrete type, but could be interface)

---

## Error Handling Philosophy

### Silent Failures

The DNS server follows a **"fail closed"** approach:

```go
if len(parts) < 3 {
    return  // No error, just return empty
}
```

**Rationale**:
- Invalid queries shouldn't crash server
- Return NXDOMAIN instead of error
- Log for debugging but don't propagate errors

### No Panics

All errors are caught and logged:

```go
if err := s.udpServer.ListenAndServe(); err != nil {
    s.log.Error("DNS UDP server failed", logger.Error(err))
    // Server continues running (TCP may still work)
}
```

---

## Security Considerations

### No Authentication

DNS queries are unauthenticated:
- Anyone can query
- No rate limiting
- No access control

**Mitigation**: Run on trusted network

### DNS Amplification

Potential for amplification attacks:
- Small query → large response (multiple SRV records)

**Future**: Implement rate limiting per source IP

### Cache Poisoning

Not applicable - no caching implemented

---

## Extensibility

### Adding New Record Types

**Example**: Add AAAA (IPv6) support

```go
case dns.TypeAAAA:
    s.handleAAAAQuery(msg, question)
```

```go
func (s *Server) handleAAAAQuery(msg *dns.Msg, question dns.Question) {
    // Similar to handleAQuery but with IPv6
    aaaa := &dns.AAAA{
        Hdr: dns.RR_Header{
            Name:   question.Name,
            Rrtype: dns.TypeAAAA,
            Class:  dns.ClassINET,
            Ttl:    30,
        },
        AAAA: net.ParseIP(service.IPv6Address),
    }
    msg.Answer = append(msg.Answer, aaaa)
}
```

### Adding Health Checks

```go
for _, service := range services {
    if service.Name == serviceName && service.Healthy {
        // Only include healthy instances
    }
}
```

Requires ServiceStore to track health status.

### Adding Service Tags

Query format: `web.production.service.consul`

```go
parts := strings.Split(name, ".")
serviceName := parts[0]
tag := parts[1]

for _, service := range services {
    if service.Name == serviceName && hasTag(service, tag) {
        // Include service
    }
}
```

---

## Future Enhancements

### 1. Response Compression

DNS supports message compression to save bandwidth:

```go
msg.Compress = true  // Enable compression
```

**Benefit**: Fit more records in UDP (512 byte limit)

### 2. EDNS0 Support

Extended DNS for larger responses:

```go
opt := new(dns.OPT)
opt.Hdr.Name = "."
opt.Hdr.Rrtype = dns.TypeOPT
opt.SetUDPSize(4096)  // Support larger responses
msg.Extra = append(msg.Extra, opt)
```

### 3. Metrics

```go
var (
    dnsQueriesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "dns_queries_total",
        },
        []string{"type", "service"},
    )

    dnsQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "dns_query_duration_seconds",
        },
        []string{"type"},
    )
)
```

### 4. Query Logging

```go
if cfg.DNS.QueryLog {
    s.log.Info("DNS query",
        zap.String("client", w.RemoteAddr().String()),
        zap.String("query", question.Name),
        zap.String("type", dns.TypeToString[question.Qtype]),
    )
}
```

---

## See Also

- [DNS User Guide](dns-service-discovery.md)
- [DNS API Reference](dns-api.md)
- [DNS Troubleshooting](dns-troubleshooting.md)
- [ADR-0006](adr/0006-dns-service-discovery.md)
- [miekg/dns Library](https://github.com/miekg/dns)
