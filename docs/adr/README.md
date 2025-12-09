# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records (ADRs) for Konsul. ADRs document significant architectural decisions made during the development of this project.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences.

## ADR Format

Each ADR follows a standard format (see [template.md](template.md)) with the following sections:

- **Status**: Proposed, Accepted, Deprecated, Superseded
- **Context**: The issue motivating this decision
- **Decision**: The change being proposed or decided
- **Consequences**: The resulting context after applying the decision

## ADR Summary by Category

### 🏗️ Core Architecture (7 ADRs)
- **ADR-0001**: Fiber Web Framework
- **ADR-0002**: BadgerDB Persistence
- **ADR-0011**: Raft Clustering & HA
- **ADR-0023**: Dependency Injection (Uber FX)
- **ADR-0024**: DI Framework Comparison
- **ADR-0026**: Agent Mode Architecture
- **ADR-0029**: Kubernetes Operator

### 🔐 Security & Access Control (6 ADRs)
- **ADR-0003**: JWT Authentication
- **ADR-0010**: ACL System
- **ADR-0013**: Rate Limiting
- **ADR-0019**: Audit Logging
- **ADR-0025**: Enhanced RBAC

### 📊 Observability (4 ADRs)
- **ADR-0004**: Prometheus Metrics
- **ADR-0005**: Structured Logging
- **ADR-0007**: OpenTelemetry Tracing

### 🌐 Service Discovery & Networking (5 ADRs)
- **ADR-0006**: DNS Interface
- **ADR-0017**: Service Tags & Metadata
- **ADR-0018**: Load Balancing Strategies
- **ADR-0027**: Multi-Datacenter Federation

### 💾 Data Storage & Operations (3 ADRs)
- **ADR-0020**: Compare-And-Swap (CAS)
- **ADR-0021**: KV Watch/Subscribe

### 🔌 APIs & Interfaces (2 ADRs)
- **ADR-0016**: GraphQL API
- **ADR-0014**: Rate Limiting Management API

### 🛠️ Developer Tools (3 ADRs)
- **ADR-0009**: React Admin UI
- **ADR-0012**: konsulctl CLI
- **ADR-0015**: Template Engine
- **ADR-0022**: Testing Strategy

### 🌍 Edge & IoT (1 ADR)
- **ADR-0028**: Edge Computing & IoT Strategy

### 📈 Status Overview
- **Accepted**: 16 ADRs (production-ready features)
- **Proposed**: 13 ADRs (under consideration/development)
- **Total**: 29 ADRs

---

## ADR Index

| ADR | Title | Status | Tags |
|-----|-------|--------|------|
| [ADR-0001](0001-use-fiber-web-framework.md) | Use Fiber Web Framework | Accepted | backend, http |
| [ADR-0002](0002-badger-for-persistence.md) | BadgerDB for Persistence Layer | Accepted | persistence, storage |
| [ADR-0003](0003-jwt-authentication.md) | JWT-Based Authentication | Accepted | security, auth |
| [ADR-0004](0004-prometheus-metrics.md) | Prometheus for Metrics | Accepted | observability, metrics |
| [ADR-0005](0005-structured-logging.md) | Structured Logging with Custom Logger | Accepted | logging, observability |
| [ADR-0006](0006-dns-service-discovery.md) | DNS Interface for Service Discovery | Accepted | service-discovery, dns |
| [ADR-0007](0007-opentelemetry-tracing.md) | OpenTelemetry for Distributed Tracing | Accepted | observability, tracing |
| [ADR-0008](0008-migrate-fiber-to-chi.md) | Migrate from Fiber to Chi Router | Proposed | backend, http |
| [ADR-0009](0009-react-admin-ui.md) | React-Based Admin UI with Vite and Tailwind CSS | Accepted | frontend, ui |
| [ADR-0010](0010-acl-system.md) | Access Control List (ACL) System | Proposed | security, authorization |
| [ADR-0011](0011-raft-clustering-ha.md) | Raft Consensus for Clustering and High Availability | Proposed | clustering, ha, raft |
| [ADR-0012](0012-cli-tool-konsulctl.md) | Command-Line Interface Tool (konsulctl) | Accepted | cli, tools |
| [ADR-0013](0013-token-bucket-rate-limiting.md) | Token Bucket Rate Limiting | Accepted | security, rate-limiting |
| [ADR-0014](0014-rate-limiting-management-api.md) | Rate Limiting Management API and Observability | Accepted | api, rate-limiting |
| [ADR-0015](0015-template-engine.md) | Template Engine for Configuration Management | Accepted | configuration, templates |
| [ADR-0016](0016-graphql-api-interface.md) | GraphQL API Interface | Accepted | api, graphql |
| [ADR-0017](0017-service-tags-metadata.md) | Service Tags and Metadata | Accepted | service-discovery |
| [ADR-0018](0018-load-balancing-strategies.md) | Load Balancing Strategies | Accepted | load-balancing |
| [ADR-0019](0019-audit-logging.md) | Audit Logging for Operational Changes | Accepted | security, compliance |
| [ADR-0020](0020-compare-and-swap-operations.md) | Compare-And-Swap (CAS) Operations | Accepted | concurrency, kv-store |
| [ADR-0021](0021-kv-watch-subscribe.md) | KV Store Watch/Subscribe System | Accepted | kv-store, real-time |
| [ADR-0022](0022-rate-limiting-comprehensive-testing.md) | Rate Limiting Comprehensive Testing Strategy | Accepted | testing, quality |
| [ADR-0023](0023-dependency-injection-with-uber-fx.md) | Dependency Injection with Uber FX | Proposed | architecture, refactoring |
| [ADR-0024](0024-dependency-injection-framework-comparison.md) | Dependency Injection Framework Comparison (FX vs Wire) | Proposed | architecture, comparison |
| [ADR-0025](0025-enhanced-rbac-system.md) | Enhanced Role-Based Access Control (RBAC) | Proposed | security, rbac, enterprise |
| [ADR-0026](0026-agent-mode-architecture.md) | Agent Mode Architecture | Proposed | architecture, scalability, agent |
| [ADR-0027](0027-multi-datacenter-federation.md) | Multi-Datacenter Federation | Proposed | multi-dc, federation, global |
| [ADR-0028](0028-edge-computing-strategy.md) | Edge Computing & IoT Strategy | Proposed | edge, iot, lightweight |
| [ADR-0029](0029-kubernetes-operator-design.md) | Kubernetes Operator Design | Proposed | kubernetes, operator, automation |

## Creating a New ADR

1. Copy the [template.md](template.md) file
2. Name it with the next sequential number: `XXXX-short-title.md`
3. Fill in the sections with relevant information
4. Update this README with a link to the new ADR
5. Submit as part of your pull request

## Lifecycle

ADRs are immutable once accepted. If a decision needs to be changed:

1. Create a new ADR that supersedes the old one
2. Update the old ADR's status to "Superseded by ADR-XXXX"
3. Reference the old ADR in the new one's context

## Resources

- [Michael Nygard's ADR article](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)
- [GitHub ADR organization](https://adr.github.io/)
