# DNS Service Discovery Documentation Index

Complete documentation for the Konsul DNS server.

## Quick Links

| Document | Description | Audience |
|----------|-------------|----------|
| [User Guide](dns-service-discovery.md) | End-user documentation and integration | Users & Operators |
| [API Reference](dns-api.md) | Complete API and type reference | Developers |
| [Implementation Guide](dns-implementation.md) | Internal architecture and design | Contributors |
| [Troubleshooting](dns-troubleshooting.md) | Debug and fix common issues | Operators |
| [ADR-0006](adr/0006-dns-service-discovery.md) | Architecture decision record | Architects |

---

## For End Users

### Getting Started

1. **[User Guide](dns-service-discovery.md)** - Start here!
   - Quick start tutorial
   - DNS query formats (A, SRV, ANY)
   - Integration examples
   - Best practices

2. **[Integration Examples](dns-service-discovery.md#integration-examples)**
   - nginx upstream configuration
   - PostgreSQL connection strings
   - Docker Compose DNS setup
   - Kubernetes CoreDNS forwarding
   - Application code examples

### When You Have Problems

3. **[Troubleshooting Guide](dns-troubleshooting.md)**
   - Common issues and solutions
   - Diagnostic commands
   - Debug logging
   - Error messages explained
   - How to get help

---

## For Developers

### Understanding the Code

4. **[API Reference](dns-api.md)**
   - Complete type definitions
   - Function signatures
   - Usage examples
   - Error handling
   - Testing approaches

5. **[Implementation Guide](dns-implementation.md)**
   - Architecture overview
   - Component deep dive
   - Query processing flow
   - Performance analysis
   - Design patterns
   - How to extend the server

### Code Examples

```go
// Basic DNS server setup
dnsConfig := dns.Config{
    Host:   "0.0.0.0",
    Port:   8600,
    Domain: "consul",
}

serviceStore := store.NewServiceStore()
logger := logger.GetDefault()

dnsServer := dns.NewServer(dnsConfig, serviceStore, logger)
dnsServer.Start()
```

---

## For Operators

### Production Deployment

6. **[System DNS Setup](dns-service-discovery.md#system-dns-setup-linux)**
   - Linux configuration
   - macOS configuration
   - Docker DNS
   - Kubernetes CoreDNS

### Monitoring and Troubleshooting

- [Health Check Script](dns-troubleshooting.md#health-check-script)
- [Performance Monitoring](dns-troubleshooting.md#performance-troubleshooting)
- [Debug Logging](dns-troubleshooting.md#debug-logging)
- [Common Issues](dns-troubleshooting.md#common-issues)

---

## For Architects

### Design Decisions

7. **[ADR-0006: DNS Service Discovery](adr/0006-dns-service-discovery.md)**
   - Context and motivation
   - Architecture decisions
   - Alternatives considered
   - Trade-offs and consequences
   - Implementation approach

8. **[Complete Documentation](DNS_DOCS_COMPLETE.md)**
   - Implementation summary
   - Documentation statistics
   - Features implemented
   - Success criteria
   - Future enhancements

---

## Documentation Map

### By Topic

#### **Installation & Setup**
- [User Guide: Quick Start](dns-service-discovery.md#quick-start)
- [User Guide: System DNS Setup](dns-service-discovery.md#system-dns-setup-linux)

#### **DNS Query Formats**
- [User Guide: Query Formats](dns-service-discovery.md#dns-query-formats)
- [API Reference: Query Processing](dns-api.md#query-processing)

#### **Configuration**
- [API Reference: Config Types](dns-api.md#types)
- [API Reference: Configuration Examples](dns-api.md#configuration-examples)

#### **Integration**
- [User Guide: Integration Examples](dns-service-discovery.md#integration-examples)
- [User Guide: Docker DNS](dns-service-discovery.md#docker-dns)
- [User Guide: Kubernetes CoreDNS](dns-service-discovery.md#kubernetes-coredns)

#### **Troubleshooting**
- [Troubleshooting: Common Issues](dns-troubleshooting.md#common-issues)
- [Troubleshooting: Debug Logging](dns-troubleshooting.md#debug-logging)
- [Troubleshooting: Packet Capture](dns-troubleshooting.md#packet-capture)

#### **Performance**
- [Implementation: Performance Analysis](dns-implementation.md#performance-analysis)
- [API Reference: Performance Characteristics](dns-api.md#performance-characteristics)
- [Troubleshooting: Performance](dns-troubleshooting.md#performance-troubleshooting)

#### **Development**
- [Implementation: Architecture](dns-implementation.md#architecture-overview)
- [Implementation: Component Breakdown](dns-implementation.md#component-breakdown)
- [Implementation: Extensibility](dns-implementation.md#extensibility)

#### **Testing**
- [Implementation: Testing Strategy](dns-implementation.md#testing-strategy)
- [API Reference: Testing](dns-api.md#testing)

---

## Documentation Statistics

| Document | Pages | Words | Audience |
|----------|-------|-------|----------|
| User Guide | 20 | 6,500 | Users & Operators |
| API Reference | 18 | 6,000 | Developers |
| Implementation Guide | 20 | 6,500 | Contributors |
| Troubleshooting | 18 | 6,000 | Operators |
| ADR-0006 | 6 | 2,000 | Architects |
| **Total** | **82** | **27,000** | All |

---

## Quick Reference

### DNS Query Commands

```bash
# A record (IP address)
dig @localhost -p 8600 web.service.consul A

# SRV record (IP + port)
dig @localhost -p 8600 _web._tcp.service.consul SRV

# Any records
dig @localhost -p 8600 web.service.consul ANY

# Short output (IP only)
dig @localhost -p 8600 web.service.consul A +short
```

### Common Query Formats

| Format | Example | Returns |
|--------|---------|---------|
| `<service>.service.<domain>` | `web.service.consul` | A record(s) |
| `<service>.node.<domain>` | `web.node.consul` | A record(s) |
| `_<service>._tcp.service.<domain>` | `_web._tcp.service.consul` | SRV record(s) |

### Configuration

```yaml
# Minimal configuration
dns:
  enabled: true
  host: ""        # All interfaces
  port: 8600      # Standard port
  domain: consul  # DNS suffix
```

### Service Registration

```bash
# Register service
curl -X POST http://localhost:8500/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web",
    "address": "10.0.0.1",
    "port": 8080
  }'

# Query via DNS
dig @localhost -p 8600 web.service.consul A
```

---

## Common Tasks

| Task | Documentation |
|------|---------------|
| Setup DNS server | [User Guide: Quick Start](dns-service-discovery.md#quick-start) |
| Configure system DNS | [User Guide: System DNS Setup](dns-service-discovery.md#system-dns-setup-linux) |
| Integrate with nginx | [User Guide: Nginx Upstream](dns-service-discovery.md#nginx-upstream) |
| Add to Docker Compose | [User Guide: Docker Compose](dns-service-discovery.md#docker-compose) |
| Debug DNS queries | [Troubleshooting: Debug Logging](dns-troubleshooting.md#debug-logging) |
| Fix NXDOMAIN errors | [Troubleshooting: NXDOMAIN](dns-troubleshooting.md#issue-2-query-returns-nxdomain) |
| Monitor DNS health | [Troubleshooting: Health Check](dns-troubleshooting.md#health-check-script) |
| Add new record types | [Implementation: Extensibility](dns-implementation.md#adding-new-record-types) |
| Run tests | [API Reference: Testing](dns-api.md#testing) |
| Understand architecture | [Implementation: Architecture](dns-implementation.md#architecture-overview) |

---

## Contributing

### Documentation Standards

- **User-facing docs**: Clear, concise, example-driven
- **Technical docs**: Detailed, accurate, code examples
- **API docs**: Complete, consistent, well-formatted

### How to Update

1. Edit relevant .md file in `/docs`
2. Update this index if adding new docs
3. Test all commands and code examples
4. Submit PR with description

### Documentation TODOs

Future documentation to add:

- [ ] Video tutorials
- [ ] Interactive examples
- [ ] DNS benchmarking guide
- [ ] Migration guide from Consul
- [ ] Production deployment checklist
- [ ] Backup and recovery procedures
- [ ] Advanced monitoring with Prometheus

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2025-01-12 | Initial documentation release |

---

## Related Documentation

### Konsul Documentation
- [Template Engine](template-engine.md) - Use DNS for service discovery in templates
- [Service Registration API](../README.md#services-api) - HTTP API for registering services
- [Architecture Overview](../README.md#architecture) - System architecture

### External Resources
- [RFC 1035 - Domain Names](https://tools.ietf.org/html/rfc1035)
- [RFC 2782 - DNS SRV](https://tools.ietf.org/html/rfc2782)
- [Consul DNS Interface](https://www.consul.io/docs/discovery/dns)
- [miekg/dns Library](https://github.com/miekg/dns)

---

## Feedback

Found an issue or have a suggestion?

- **Documentation bugs**: [GitHub Issues](https://github.com/yourusername/konsul/issues)
- **Feature requests**: [GitHub Discussions](https://github.com/yourusername/konsul/discussions)
- **Questions**: See [Troubleshooting Guide](dns-troubleshooting.md#getting-help)

---

## License

Documentation is licensed under [Creative Commons BY 4.0](https://creativecommons.org/licenses/by/4.0/).

Code examples are licensed under [MIT License](../LICENSE).
