# Konsul Documentation Index

**Last Updated**: 2025-12-06

Welcome to the Konsul documentation! This index helps you find the right documentation for your needs.

## 🚀 Quick Start

**New to Konsul?** Start here:
1. [Quick Start Guide](DEPLOYMENT_QUICK_START.md) - Get running in 5 minutes
2. [Architecture Overview](ARCHITECTURE_USE_CASES.md#deployment-scenarios-overview) - Understand deployment options
3. [TODO Roadmap](TODO.md) - See what's implemented and what's planned

## 📚 Documentation Categories

### 1. Architecture & Design

#### High-Level Architecture
- **[Architecture Use Cases](ARCHITECTURE_USE_CASES.md)** - Comprehensive deployment scenarios
  - Small Team (3-10 servers)
  - Growing Startup (10-50 servers)
  - Medium Enterprise (100-500 servers)
  - Large Enterprise (1000+ servers)
  - Edge/IoT Deployment
  - Hybrid Cloud

#### Architecture Decision Records (ADRs)
- **[ADR Index](adr/README.md)** - All 29 architectural decisions
- **[Core Architecture ADRs](adr/README.md#-core-architecture-7-adrs)**
  - [ADR-0026: Agent Mode](adr/0026-agent-mode-architecture.md) ⭐ Critical for scale
  - [ADR-0027: Multi-DC Federation](adr/0027-multi-datacenter-federation.md) ⭐ Global deployment
  - [ADR-0029: Kubernetes Operator](adr/0029-kubernetes-operator-design.md) ⭐ Cloud-native
  - [ADR-0023: Dependency Injection](adr/0023-dependency-injection-with-uber-fx.md)
  - [ADR-0011: Raft Clustering](adr/0011-raft-clustering-ha.md)

### 2. Security & Access Control

- **[ADR-0025: Enhanced RBAC](adr/0025-enhanced-rbac-system.md)** - Enterprise role-based access
- **[ADR-0010: ACL System](adr/0010-acl-system.md)** - Policy-based authorization
- **[ADR-0003: JWT Authentication](adr/0003-jwt-authentication.md)** - Authentication system
- **[ADR-0019: Audit Logging](adr/0019-audit-logging.md)** - Compliance and audit trails
- **[ADR-0013: Rate Limiting](adr/0013-token-bucket-rate-limiting.md)** - API protection

### 3. Service Discovery

- **[ADR-0006: DNS Interface](adr/0006-dns-service-discovery.md)** - DNS-based discovery
- **[ADR-0017: Service Tags & Metadata](adr/0017-service-tags-metadata.md)** - Advanced querying
- **[ADR-0018: Load Balancing](adr/0018-load-balancing-strategies.md)** - Traffic distribution

### 4. Data Storage

- **[ADR-0002: BadgerDB Persistence](adr/0002-badger-for-persistence.md)** - Storage backend
- **[ADR-0020: Compare-And-Swap](adr/0020-compare-and-swap-operations.md)** - Atomic operations
- **[ADR-0021: KV Watch/Subscribe](adr/0021-kv-watch-subscribe.md)** - Real-time updates

### 5. APIs & Interfaces

- **[ADR-0016: GraphQL API](adr/0016-graphql-api-interface.md)** - Modern API interface
- **[ADR-0001: Fiber Framework](adr/0001-use-fiber-web-framework.md)** - HTTP framework
- **[ADR-0014: Rate Limiting API](adr/0014-rate-limiting-management-api.md)** - Management API

### 6. Observability

- **[ADR-0004: Prometheus Metrics](adr/0004-prometheus-metrics.md)** - Metrics collection
- **[ADR-0007: OpenTelemetry Tracing](adr/0007-opentelemetry-tracing.md)** - Distributed tracing
- **[ADR-0005: Structured Logging](adr/0005-structured-logging.md)** - Logging system

### 7. Edge & IoT

- **[ADR-0028: Edge Computing Strategy](adr/0028-edge-computing-strategy.md)** ⭐ IoT deployment
  - Lightweight nodes (<10MB)
  - MQTT integration
  - Offline-first sync
  - Device management

### 8. Developer Tools

- **[ADR-0009: React Admin UI](adr/0009-react-admin-ui.md)** - Web interface
- **[ADR-0012: konsulctl CLI](adr/0012-cli-tool-konsulctl.md)** - Command-line tool
- **[ADR-0015: Template Engine](adr/0015-template-engine.md)** - Configuration templates

---

## 📖 Documentation by Use Case

### I want to deploy Konsul

**Choose your scenario**:
- **Development/Testing** → [Quick Start (5 min)](DEPLOYMENT_QUICK_START.md#scenario-1-development-5-minutes)
- **Small Production (3-10 servers)** → [Small Team Guide](ARCHITECTURE_USE_CASES.md#scenario-1-small-team-3-10-servers)
- **Growing Startup (10-50 servers)** → [Startup Guide](ARCHITECTURE_USE_CASES.md#scenario-2-growing-startup-10-50-servers)
- **Enterprise (100+ servers)** → [Enterprise Guide](ARCHITECTURE_USE_CASES.md#scenario-3-medium-enterprise-100-500-servers)
- **Global/Multi-DC** → [Multi-DC Guide](ARCHITECTURE_USE_CASES.md#scenario-4-large-enterprise-1000-servers-multi-cluster)
- **Edge/IoT Devices** → [Edge Guide](ARCHITECTURE_USE_CASES.md#scenario-5-edgeiot-deployment)

### I want to understand architectural decisions

**By topic**:
- **Why Agent Mode?** → [ADR-0026](adr/0026-agent-mode-architecture.md#context)
- **Why Multi-DC Federation?** → [ADR-0027](adr/0027-multi-datacenter-federation.md#context)
- **Why Raft vs Other Consensus?** → [ADR-0011](adr/0011-raft-clustering-ha.md#alternatives-considered)
- **Why Uber FX vs Google Wire?** → [ADR-0024](adr/0024-dependency-injection-framework-comparison.md)
- **Why GraphQL?** → [ADR-0016](adr/0016-graphql-api-interface.md#context)

### I want to implement a feature

**Implementation guides**:
- **Clustering & HA** → [ADR-0011](adr/0011-raft-clustering-ha.md#implementation-notes)
- **Agent Mode** → [ADR-0026](adr/0026-agent-mode-architecture.md#implementation-components)
- **RBAC System** → [ADR-0025](adr/0025-enhanced-rbac-system.md#implementation-notes)
- **Multi-DC** → [ADR-0027](adr/0027-multi-datacenter-federation.md#implementation-phases)
- **K8s Operator** → [ADR-0029](adr/0029-kubernetes-operator-design.md#implementation-phases)
- **Edge Nodes** → [ADR-0028](adr/0028-edge-computing-strategy.md#implementation-phases)

### I want to scale Konsul

**Scalability guides**:
1. **0-50 services**: Single node → 3-node cluster
2. **50-100 services**: Add agent mode → [ADR-0026](adr/0026-agent-mode-architecture.md)
3. **100-500 services**: 5-node cluster + agents
4. **500+ services**: Multi-cluster + federation → [ADR-0027](adr/0027-multi-datacenter-federation.md)
5. **Global scale**: Multi-DC + mesh gateway

### I want to secure Konsul

**Security guides**:
- **Authentication** → [ADR-0003](adr/0003-jwt-authentication.md)
- **Authorization** → [ADR-0010](adr/0010-acl-system.md)
- **Enhanced RBAC** → [ADR-0025](adr/0025-enhanced-rbac-system.md)
- **Rate Limiting** → [ADR-0013](adr/0013-token-bucket-rate-limiting.md)
- **Audit Logging** → [ADR-0019](adr/0019-audit-logging.md)

---

## 🗺️ Roadmap & Planning

### Current Status
- **[TODO Roadmap](TODO.md)** - Feature checklist with status
- **[Product Backlog](BACKLOG.md)** - Prioritized backlog with effort estimates
- **[ADR Status](adr/README.md#-status-overview)** - 16 Accepted, 13 Proposed

### Strategic Initiatives

#### Phase 1: Production Readiness (Q1 2025) - 47 SP
**Status**: Proposed
- Raft Clustering (21 SP) - [ADR-0011](adr/0011-raft-clustering-ha.md)
- Leader Election (8 SP)
- Data Replication (13 SP)
- Cluster Management API (5 SP)

#### Phase 2: Enterprise Security (Q2 2025) - 47 SP
**Status**: Proposed
- Enhanced RBAC (21 SP) - [ADR-0025](adr/0025-enhanced-rbac-system.md)
- LDAP/AD Integration (13 SP)
- Temporal Assignments (5 SP)
- Autopilot (8 SP)

#### Phase 3: Scalability (Q2-Q3 2025) - 55 SP
**Status**: Proposed
- Agent Mode (21 SP) - [ADR-0026](adr/0026-agent-mode-architecture.md)
- K8s Operator (14 SP) - [ADR-0029](adr/0029-kubernetes-operator-design.md)
- Multi-DC Federation (20 SP) - [ADR-0027](adr/0027-multi-datacenter-federation.md)

#### Phase 4: Innovation (Q3-Q4 2025)
**Status**: Proposed
- Edge Computing (14 SP) - [ADR-0028](adr/0028-edge-computing-strategy.md)
- AI/ML Integration (68 SP) - [TODO](TODO.md#12-aiml-integration-)
- Chaos Engineering (34 SP) - [TODO](TODO.md#14-chaos-engineering-)

---

## 📊 Feature Matrix

### By Deployment Scale

| Feature | Small<br>(3-10) | Startup<br>(10-50) | Enterprise<br>(100-500) | Global<br>(1000+) |
|---------|---------|----------|------------|---------|
| **Service Discovery** | ✅ Direct | ✅ DNS | ✅ Agent Cache | ✅ Multi-DC |
| **High Availability** | ❌ | ✅ 3-node | ✅ 5-node | ✅ Multi-cluster |
| **Agent Mode** | ❌ | ❌ | ✅ DaemonSet | ✅ Sidecar |
| **ACL/RBAC** | Optional | ✅ ACL | ✅ RBAC | ✅ LDAP/AD |
| **Audit Logging** | Optional | ✅ File | ✅ SIEM | ✅ Distributed |
| **Multi-DC** | ❌ | ❌ | Optional | ✅ Required |
| **Service Mesh** | ❌ | ❌ | Optional | ✅ mTLS |

### By Status

| Status | Count | Examples |
|--------|-------|----------|
| **✅ Production** | 16 | Fiber, BadgerDB, JWT Auth, Prometheus, GraphQL |
| **🚧 Proposed** | 13 | Raft, Agent Mode, Multi-DC, RBAC, K8s Operator |
| **💡 Planned** | 50+ | AI/ML, Edge, Chaos Engineering, FinOps |

---

## 🔍 Quick Reference

### Common Tasks

**Register a Service**:
```bash
# Direct registration
konsulctl service register --name web --address 192.168.1.10 --port 3000

# Kubernetes (automatic)
kubectl label pod myapp konsul.io/inject=true
```

**Configure via KV Store**:
```bash
# Set configuration
konsulctl kv set config/app/db_url "postgresql://db:5432/myapp"

# Read in application
export DB_URL=$(konsulctl kv get config/app/db_url)
```

**Discover Services**:
```bash
# Via konsulctl
konsulctl service get api

# Via DNS
dig @konsul-dns api.service.konsul SRV

# Via GraphQL
curl -X POST http://konsul:8500/graphql \
  -d '{"query": "{ service(name: \"api\") { address port } }"}'
```

### Architecture Patterns

**Sidecar Pattern** (Service Registration):
```yaml
# Kubernetes with agent sidecar
apiVersion: v1
kind: Pod
metadata:
  annotations:
    konsul.io/inject: "true"
spec:
  containers:
  - name: app
    image: myapp:latest
```

**Agent Pattern** (Scalability):
```yaml
# DaemonSet for node-level agents
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: konsul-agent
spec:
  template:
    spec:
      hostNetwork: true
      containers:
      - name: agent
        image: konsul/agent:latest
```

**Federation Pattern** (Multi-DC):
```yaml
# Primary datacenter
datacenter: us-east
primary_datacenter: us-east
wan:
  enabled: true
  members:
    - konsul-us-west.example.com
    - konsul-eu.example.com
```

---

## 🆘 Troubleshooting

### Common Issues

**Problem**: Services not registering
- **Solution**: Check agent connectivity → [Agent Mode Guide](adr/0026-agent-mode-architecture.md#troubleshooting)

**Problem**: Can't discover services
- **Solution**: Verify DNS configuration → [DNS ADR](adr/0006-dns-service-discovery.md)

**Problem**: Authorization denied
- **Solution**: Check ACL policies → [ACL Guide](adr/0010-acl-system.md)

**Problem**: High server load
- **Solution**: Deploy agents → [Agent Mode ADR](adr/0026-agent-mode-architecture.md#context)

**Problem**: Cross-DC queries failing
- **Solution**: Check mesh gateway → [Multi-DC ADR](adr/0027-multi-datacenter-federation.md#mesh-gateway-configuration)

---

## 📝 Contributing

### Documentation Structure

```
docs/
├── DOCUMENTATION_INDEX.md          # This file (navigation hub)
├── TODO.md                         # Feature roadmap
├── BACKLOG.md                      # Prioritized backlog
├── ARCHITECTURE_USE_CASES.md      # Deployment scenarios
├── DEPLOYMENT_QUICK_START.md      # Quick start guide
└── adr/
    ├── README.md                   # ADR index
    ├── template.md                 # ADR template
    ├── 0001-*.md                   # Individual ADRs
    └── ...
```

### Adding Documentation

1. **New Feature**: Create ADR using [template](adr/template.md)
2. **Deployment Guide**: Update [ARCHITECTURE_USE_CASES.md](ARCHITECTURE_USE_CASES.md)
3. **Quick Reference**: Update [DEPLOYMENT_QUICK_START.md](DEPLOYMENT_QUICK_START.md)
4. **Roadmap**: Update [TODO.md](TODO.md) and [BACKLOG.md](BACKLOG.md)
5. **Update Index**: Add links to this file

### Review Process

- Architecture decisions require ADR review
- Deployment guides reviewed by SRE team
- API changes documented in ADRs
- Breaking changes require migration guide

---

## 🔗 External Resources

### Official Links
- **GitHub Repository**: https://github.com/neogan74/konsul
- **Issue Tracker**: https://github.com/neogan74/konsul/issues
- **Discussions**: https://github.com/neogan74/konsul/discussions

### Related Projects
- **HashiCorp Consul**: https://www.consul.io/
- **etcd**: https://etcd.io/
- **Netflix Eureka**: https://github.com/Netflix/eureka

### Learning Resources
- **Service Discovery Patterns**: Martin Fowler
- **Raft Consensus**: https://raft.github.io/
- **Kubernetes Operators**: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/

---

## 📅 Document Maintenance

**Review Schedule**:
- **DOCUMENTATION_INDEX.md**: Monthly
- **TODO.md**: Weekly
- **BACKLOG.md**: Bi-weekly
- **ADRs**: Immutable (create new to supersede)

**Last Review**: 2025-12-06
**Next Review**: 2025-12-20
**Maintained By**: Konsul Core Team

---

## 📈 Documentation Stats

- **Total Documents**: 35+
- **ADRs**: 29
- **Deployment Guides**: 6 scenarios
- **Implementation Guides**: 16 phases
- **Total Word Count**: ~50,000 words
- **Estimated Read Time**: ~4 hours (complete documentation)

---

**Need help?** Start with the [Quick Start Guide](DEPLOYMENT_QUICK_START.md) or browse [ADRs by category](adr/README.md#adr-summary-by-category).