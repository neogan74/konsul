# DNS Service Discovery - User Guide

DNS-based service discovery for Konsul.

## Overview

Konsul provides a built-in DNS server that allows applications to discover services using standard DNS queries. This enables zero-code integration with existing tools and applications that expect DNS-based service discovery.

### Features

- **Standard DNS Protocol** - UDP and TCP support
- **Multiple Record Types** - A, SRV, and ANY queries
- **Consul Compatible** - Drop-in replacement for Consul DNS
- **Low Latency** - Direct queries against in-memory store
- **Automatic Updates** - Service changes reflected immediately
- **No Configuration** - Works out of the box with sensible defaults

---

## Quick Start

### 1. Enable DNS Server

DNS is enabled by default in Konsul. The server listens on port 8600:

```yaml
# config.yaml
dns:
  enabled: true
  host: ""        # Listen on all interfaces
  port: 8600      # Standard Consul DNS port
  domain: consul  # DNS domain suffix
```

### 2. Register a Service

Register services via the HTTP API:

```bash
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080
  }'
```

### 3. Query via DNS

Query the service using standard DNS tools:

```bash
# SRV query (includes port)
dig @localhost -p 8600 _web._tcp.service.consul SRV

# A query (IP address only)
dig @localhost -p 8600 web.service.consul A

# Node format
dig @localhost -p 8600 web.node.consul A
```

---

## DNS Query Formats

### SRV Records (Service + Port)

**Format**: `_<service>._tcp.service.<domain>`

```bash
dig @localhost -p 8600 _web._tcp.service.consul SRV
```

**Response**:
```
_web._tcp.service.consul. 30 IN SRV 1 100 8080 web.node.consul.
web.node.consul.         30 IN A   10.0.0.1
```

**Use case**: When you need both address and port

---

### A Records (IP Address Only)

**Format 1**: `<service>.service.<domain>`

```bash
dig @localhost -p 8600 web.service.consul A
```

**Format 2**: `<service>.node.<domain>`

```bash
dig @localhost -p 8600 web.node.consul A
```

**Response**:
```
web.service.consul. 30 IN A 10.0.0.1
```

**Use case**: When port is known or hardcoded

---

### ANY Queries (All Record Types)

```bash
dig @localhost -p 8600 _web._tcp.service.consul ANY
```

**Response**: Returns both SRV and A records

---

## Integration Examples

### PostgreSQL Connection String

```bash
# Using DNS for host resolution
psql "postgresql://db.service.consul:5432/mydb?sslmode=disable"
```

### Nginx Upstream

```nginx
upstream backend {
    server web.service.consul:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://backend;
    }
}
```

### Docker Compose

```yaml
services:
  app:
    image: myapp:latest
    environment:
      - DB_HOST=postgres.service.consul
      - REDIS_HOST=redis.service.consul
    dns:
      - 10.0.0.1  # Konsul DNS server
    dns_search:
      - service.consul
```

### Application Code (No DNS Library Required)

**Python**:
```python
import psycopg2

# DNS resolution happens automatically
conn = psycopg2.connect(
    host="postgres.service.consul",
    port=5432,
    database="mydb"
)
```

**Go**:
```go
import "net"

// Standard library uses OS DNS resolver
addrs, err := net.LookupHost("web.service.consul")
```

**Node.js**:
```javascript
const http = require('http');

// DNS lookup happens automatically
http.get('http://api.service.consul:8080', (res) => {
    // ...
});
```

---

## DNS Configuration

### System DNS Setup (Linux)

Add Konsul as a DNS resolver:

```bash
# /etc/resolv.conf
nameserver 10.0.0.1  # Konsul DNS IP
nameserver 8.8.8.8   # Fallback DNS
search service.consul
```

**Note**: Use `resolvconf` or `systemd-resolved` for persistent configuration

---

### System DNS Setup (macOS)

```bash
# Add DNS resolver for .consul domain
sudo mkdir -p /etc/resolver
echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/consul
echo "port 8600" | sudo tee -a /etc/resolver/consul
```

Test:
```bash
dscacheutil -q host -a name web.service.consul
```

---

### Docker DNS

```bash
docker run \
  --dns=10.0.0.1 \
  --dns-search=service.consul \
  myapp:latest
```

Or in docker-compose.yml:
```yaml
version: '3'
services:
  app:
    image: myapp:latest
    dns: 10.0.0.1
    dns_search: service.consul
```

---

### Kubernetes CoreDNS

Forward `.consul` queries to Konsul:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods insecure
           fallthrough in-addr.arpa ip6.arpa
        }
        forward consul konsul.default.svc.cluster.local:8600
        forward . /etc/resolv.conf
        cache 30
    }
```

---

## Performance Considerations

### TTL (Time To Live)

Default TTL is **30 seconds**:

```
web.service.consul. 30 IN A 10.0.0.1
```

**Trade-offs**:
- **Lower TTL** (0-5s): More accurate, higher DNS load
- **Higher TTL** (60s+): Less load, stale data possible

**Current implementation**: Fixed 30s TTL

---

### Caching

**Client-side caching**: Managed by OS DNS resolver

**Disable client caching** (for testing):
```bash
# Linux
sudo systemd-resolve --flush-caches

# macOS
sudo dscacheutil -flushcache
```

---

### Query Performance

Typical latency: **< 2ms**

```bash
# Benchmark DNS queries
for i in {1..100}; do
  time dig @localhost -p 8600 web.service.consul A +short
done
```

**Bottlenecks**:
- Network latency (negligible on localhost)
- Service store lookup (in-memory, very fast)
- DNS message serialization

---

## Multiple Service Instances

When multiple instances are registered with the same name:

```bash
# Register three web instances
curl -X POST http://localhost:8500/services -d '{"name":"web","address":"10.0.0.1","port":8080}'
curl -X POST http://localhost:8500/services -d '{"name":"web","address":"10.0.0.2","port":8080}'
curl -X POST http://localhost:8500/services -d '{"name":"web","address":"10.0.0.3","port":8080}'
```

**DNS returns all instances**:
```bash
dig @localhost -p 8600 web.service.consul A +short
# 10.0.0.1
# 10.0.0.2
# 10.0.0.3
```

**Client-side load balancing**: Most DNS clients randomize the order

---

## Troubleshooting

### Query Returns NXDOMAIN

**Problem**: DNS query fails with "server can't find" error

```bash
dig @localhost -p 8600 myservice.service.consul A
# ;; Got answer: NXDOMAIN
```

**Causes**:
1. Service not registered
2. Service expired (TTL)
3. Wrong service name

**Solution**:

```bash
# 1. Check service is registered
curl http://localhost:8500/services | jq .

# 2. Verify service name
curl http://localhost:8500/services/myservice | jq .

# 3. Re-register service
curl -X POST http://localhost:8500/services -d '{
  "name": "myservice",
  "address": "10.0.0.1",
  "port": 8080
}'
```

---

### DNS Server Not Responding

**Problem**: Connection timeout or refused

```bash
dig @localhost -p 8600 web.service.consul A
# ;; connection timed out; no servers could be reached
```

**Diagnosis**:

```bash
# Check DNS server is running
netstat -tuln | grep 8600
# Should show: udp 0.0.0.0:8600

# Check Konsul logs
journalctl -u konsul -f | grep DNS

# Test network connectivity
nc -vz localhost 8600
```

**Solution**: Verify DNS enabled in config:

```yaml
dns:
  enabled: true
  port: 8600
```

---

### Wrong IP Address Returned

**Problem**: DNS returns incorrect or old IP address

**Causes**:
1. DNS caching (client-side)
2. Service not updated
3. Multiple instances with different IPs

**Solution**:

```bash
# Flush DNS cache
sudo systemd-resolve --flush-caches  # Linux
sudo dscacheutil -flushcache         # macOS

# Query with no caching
dig @localhost -p 8600 web.service.consul A +nocache

# Verify current registration
curl http://localhost:8500/services/web | jq .
```

---

### Port Not Included in Response

**Problem**: Application can't connect (missing port)

**Cause**: Using A record instead of SRV record

**Solution**: Use SRV query format:

```bash
# Wrong - A record has no port info
dig @localhost -p 8600 web.service.consul A

# Correct - SRV record includes port
dig @localhost -p 8600 _web._tcp.service.consul SRV
```

---

## Advanced Usage

### DNS Forwarding

Forward non-Konsul queries to upstream DNS:

Use **dnsmasq** or **CoreDNS** as a forwarding proxy:

```bash
# dnsmasq configuration
server=/consul/127.0.0.1#8600
server=8.8.8.8
```

---

### Health Checks (Future)

Currently, all registered services are returned. Future versions will support:

- Health check integration
- Only return healthy instances
- Configurable health check behavior

---

### Service Tags (Future)

Filter services by tags in DNS queries:

```bash
# Planned feature
dig @localhost -p 8600 web.production.service.consul SRV
```

---

## Comparison with Consul

Konsul DNS is designed to be **compatible** with HashiCorp Consul DNS:

| Feature | Consul | Konsul |
|---------|--------|--------|
| Port | 8600 | 8600 ✅ |
| SRV records | ✅ | ✅ |
| A records | ✅ | ✅ |
| Domain | `.consul` | `.consul` ✅ |
| Health checks | ✅ | 🚧 Planned |
| Service tags | ✅ | 🚧 Planned |
| Prepared queries | ✅ | ❌ |
| PTR records | ✅ | ❌ |

**Migration from Consul**: Drop-in replacement for basic DNS queries

---

## Best Practices

### 1. Use Consistent Domain

Stick with `consul` domain for compatibility:

```yaml
dns:
  domain: consul  # Recommended
```

### 2. Configure DNS Search Path

Simplify queries by adding search domain:

```bash
# /etc/resolv.conf
search service.consul
```

Then use short names:
```bash
ping web  # Resolves to web.service.consul
```

### 3. Monitor DNS Query Metrics

Track DNS performance and errors:

```bash
# View DNS metrics (when implemented)
curl http://localhost:8500/metrics | grep dns_
```

### 4. Use SRV Records for Dynamic Ports

Don't hardcode ports in application:

```bash
# Good - port from DNS
_service._tcp.service.consul SRV

# Bad - hardcoded port
service.service.consul A + port 8080
```

### 5. Handle NXDOMAIN Gracefully

Application should handle DNS failures:

```python
try:
    addr = socket.gethostbyname('service.service.consul')
except socket.gaierror:
    # Fallback or retry
    addr = 'localhost'
```

---

## Security Considerations

### DNS Spoofing

DNS queries are unauthenticated by default:

**Mitigations**:
- Run DNS on private network
- Use firewall rules to restrict access
- Consider DNSSEC (future)

### Rate Limiting

Protect against DNS amplification attacks:

**Current**: No rate limiting (trusted network assumption)

**Future**: Implement per-IP rate limiting

---

## See Also

- [ADR-0006: DNS Service Discovery](adr/0006-dns-service-discovery.md)
- [DNS API Reference](dns-api.md)
- [DNS Implementation Guide](dns-implementation.md)
- [DNS Troubleshooting](dns-troubleshooting.md)
- [RFC 1035 - Domain Names](https://tools.ietf.org/html/rfc1035)
- [RFC 2782 - DNS SRV](https://tools.ietf.org/html/rfc2782)
