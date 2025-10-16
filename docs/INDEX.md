# Konsul Documentation Index

Complete documentation hub for Konsul - a lightweight service mesh and key-value store.

---

## 🚀 Getting Started

Start here if you're new to Konsul:

- **[Main README](../README.md)** - Project overview and quick start
- **[Deployment Guide](deployment.md)** - Deploy Konsul to production

---

## 📚 Core Features

### Key-Value Store
- **Operations**: See [Main README](../README.md#kv-storage)
- **Persistence**: [Persistence API](persistence-api.md) | [BadgerDB Implementation](persistence-badger.md) | [Architecture](persistence-implementation.md)

### Service Discovery
- **[DNS Service Discovery](dns-service-discovery.md)** - DNS-based service discovery
- **[DNS API Reference](dns-api.md)** - Complete DNS API documentation
- **[DNS Implementation](dns-implementation.md)** - Technical deep dive
- **[DNS Troubleshooting](dns-troubleshooting.md)** - Common issues and solutions
- **[DNS Complete Guide](DNS_DOCS_COMPLETE.md)** - All DNS docs in one place
- **[DNS Index](DNS_DOCS_INDEX.md)** - DNS documentation navigator

### Template Engine
- **[User Guide](template-engine.md)** - Getting started with templates
- **[API Reference](template-engine-api.md)** - Complete API documentation
- **[Implementation Guide](template-engine-implementation.md)** - Architecture and design
- **[Performance Guide](template-engine-performance.md)** - Optimization tips
- **[Troubleshooting](template-engine-troubleshooting.md)** - Common issues
- **[Complete Guide](TEMPLATE_DOCS_COMPLETE.md)** - All template docs in one place
- **[Template Index](TEMPLATE_DOCS_INDEX.md)** - Template documentation navigator
- **[Implementation Summary](TEMPLATE_IMPLEMENTATION.md)** - What was implemented

---

## 🔐 Security & Access Control

### Authentication
- **[Authentication Guide](authentication.md)** - JWT and API key authentication
- **[Authentication API](authentication-api.md)** - Complete API reference
- **[ACL Guide](acl-guide.md)** - Access Control Lists

---

## 📊 Observability

### Metrics
- **[Metrics Guide](metrics.md)** - Prometheus metrics and monitoring

### Logging
- **[Structured Logging](logging.md)** - Zap-based logging system
  - Log levels and formats
  - Request correlation
  - Integration with Loki, ELK
  - Query examples

### Tracing
- **[OpenTelemetry Tracing](tracing.md)** - Distributed tracing
  - W3C Trace Context
  - Integration with Tempo, Jaeger
  - Span attributes
  - Troubleshooting traces

---

## 🛡️ Traffic Management

### Rate Limiting
- **[Rate Limiting Guide](rate-limiting.md)** - Token bucket rate limiting
  - Configuration
  - Per-IP and per-API-key strategies
  - Metrics and monitoring
  - Troubleshooting
- **[Rate Limiting Management API](rate-limiting-api.md)** - Admin API for managing rate limits
  - Real-time statistics
  - Reset operations
  - Dynamic configuration
  - Client monitoring

---

## 🛠️ Tools & Interfaces

### CLI Tool
- **[konsulctl Documentation](konsulctl.md)** - Command-line interface
  - KV operations
  - Service management
  - Backup/restore
  - TLS support
  - Scripting examples

### Admin UI
- **[React Admin UI](admin-ui.md)** - Web-based administration
  - Dashboard
  - Service visualization
  - Configuration management
  - Development guide

---

## 🏗️ Architecture

### Architecture Decision Records (ADRs)
- **[ADR Index](adr/README.md)** - All architectural decisions
- [ADR-0001: Use Fiber Web Framework](adr/0001-use-fiber-web-framework.md)
- [ADR-0002: BadgerDB for Persistence](adr/0002-badger-for-persistence.md)
- [ADR-0003: JWT Authentication](adr/0003-jwt-authentication.md)
- [ADR-0004: Prometheus Metrics](adr/0004-prometheus-metrics.md)
- [ADR-0005: Structured Logging](adr/0005-structured-logging.md)
- [ADR-0006: DNS Service Discovery](adr/0006-dns-service-discovery.md)
- [ADR-0007: OpenTelemetry Tracing](adr/0007-opentelemetry-tracing.md)
- [ADR-0008: Migrate Fiber to Chi](adr/0008-migrate-fiber-to-chi.md)
- [ADR-0009: React Admin UI](adr/0009-react-admin-ui.md)
- [ADR-0010: ACL System](adr/0010-acl-system.md)
- [ADR-0011: Raft Clustering HA](adr/0011-raft-clustering-ha.md)
- [ADR-0012: CLI Tool konsulctl](adr/0012-cli-tool-konsulctl.md)
- [ADR-0013: Token Bucket Rate Limiting](adr/0013-token-bucket-rate-limiting.md)
- [ADR-0014: Rate Limiting Management API](adr/0014-rate-limiting-management-api.md)
- [ADR-0015: Template Engine](adr/0015-template-engine.md)

---

## 📖 Documentation by Category

### User Guides
Quick starts and tutorials for end users:
- [Main README](../README.md)
- [Authentication Guide](authentication.md)
- [DNS Service Discovery](dns-service-discovery.md)
- [Template Engine User Guide](template-engine.md)
- [Metrics Guide](metrics.md)
- [Rate Limiting Guide](rate-limiting.md)
- [konsulctl CLI](konsulctl.md)

### API References
Complete API documentation:
- [Authentication API](authentication-api.md)
- [DNS API](dns-api.md)
- [Persistence API](persistence-api.md)
- [Template Engine API](template-engine-api.md)

### Implementation Guides
Technical deep dives for developers:
- [DNS Implementation](dns-implementation.md)
- [Persistence Implementation](persistence-implementation.md)
- [Template Engine Implementation](template-engine-implementation.md)
- [ACL Guide](acl-guide.md)

### Troubleshooting Guides
Problem-solving resources:
- [DNS Troubleshooting](dns-troubleshooting.md)
- [Template Engine Troubleshooting](template-engine-troubleshooting.md)
- [Rate Limiting Troubleshooting](rate-limiting.md#troubleshooting)
- [Tracing Troubleshooting](tracing.md#troubleshooting)
- [Logging Troubleshooting](logging.md#troubleshooting)

### Operations Guides
Deployment and maintenance:
- [Deployment Guide](deployment.md)
- [Metrics and Monitoring](metrics.md)
- [Logging Guide](logging.md)
- [Tracing Guide](tracing.md)

---

## 📋 Complete Documentation Sets

Some features have multiple related docs. These "complete" guides combine everything:

- **[DNS Complete Guide](DNS_DOCS_COMPLETE.md)** - All DNS documentation
- **[Template Engine Complete Guide](TEMPLATE_DOCS_COMPLETE.md)** - All template documentation

---

## 🔍 Quick Reference

### Configuration
All environment variables and configuration options:
- [Server Configuration](../README.md#configuration)
- [TLS Configuration](../README.md#tlsssl-configuration)
- [Authentication Configuration](../README.md#authentication-configuration)
- [Persistence Configuration](../README.md#persistence-configuration)
- [DNS Configuration](../README.md#dns-configuration)
- [Rate Limiting Configuration](rate-limiting.md#configuration)
- [Tracing Configuration](tracing.md#configuration)
- [Logging Configuration](logging.md#configuration)

### API Endpoints
Quick reference for all API endpoints:
- [KV Store Endpoints](../README.md#kv-storage)
- [Service Discovery Endpoints](../README.md#service-discovery)
- [Authentication Endpoints](../README.md#authentication-endpoints)
- [Health Endpoints](../README.md#monitoring--health-checks)
- [Metrics Endpoint](../README.md#metrics-prometheus)
- [Backup Endpoints](persistence-api.md)

### CLI Commands
Quick reference for konsulctl:
- [KV Commands](konsulctl.md#kv-commands)
- [Service Commands](konsulctl.md#service-commands)
- [Backup Commands](konsulctl.md#backup-commands)
- [DNS Commands](konsulctl.md#dns-commands)

---

## 🎯 Common Tasks

### I want to...

| Task | Documentation |
|------|---------------|
| **Deploy Konsul** | [Deployment Guide](deployment.md) |
| **Use the CLI** | [konsulctl Documentation](konsulctl.md) |
| **Set up authentication** | [Authentication Guide](authentication.md) |
| **Enable DNS** | [DNS Service Discovery](dns-service-discovery.md) |
| **Create templates** | [Template Engine User Guide](template-engine.md) |
| **Monitor with Prometheus** | [Metrics Guide](metrics.md) |
| **Enable tracing** | [OpenTelemetry Tracing](tracing.md) |
| **Configure logging** | [Structured Logging](logging.md) |
| **Set up rate limiting** | [Rate Limiting Guide](rate-limiting.md) |
| **Use Admin UI** | [React Admin UI](admin-ui.md) |
| **Backup data** | [Persistence API](persistence-api.md) |
| **Troubleshoot DNS** | [DNS Troubleshooting](dns-troubleshooting.md) |
| **Understand architecture** | [ADR Index](adr/README.md) |

---

## 🗺️ Documentation Roadmap

See [TODO.md](TODO.md) for planned documentation improvements.

---

## 📊 Documentation Statistics

- **Total Documentation Files**: 29+ markdown files
- **Total Pages**: ~500+ pages
- **Total Words**: ~200,000+ words
- **Categories**: 8 major feature areas
- **Architecture Decisions**: 15 ADRs
- **API References**: 4 complete API docs
- **Troubleshooting Guides**: 5 dedicated guides

---

## 🤝 Contributing to Documentation

When contributing documentation:

1. **Follow the template structure** - Use existing docs as reference
2. **Include examples** - Show, don't just tell
3. **Add troubleshooting** - Document common issues
4. **Update this index** - Keep INDEX.md current
5. **Link related docs** - Cross-reference related content
6. **Use consistent formatting** - Match existing style

---

## 📞 Getting Help

- **Issues**: [GitHub Issues](https://github.com/neogan74/konsul/issues)
- **Documentation Search**: Use GitHub's search or `grep -r "keyword" docs/`
- **Examples**: See `examples/` directory in the repository

---

**Last Updated**: 2025-10-15
**Version**: 0.1.0
