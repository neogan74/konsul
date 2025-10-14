# TODO - Konsul Project Roadmap

## 1. Persistence Layer
- [x] Add optional persistence to disk (BadgerDB)
- [x] Implement backup/restore functionality
- [x] Add WAL (Write-Ahead Logging) for crash recovery
- [x] Pluggable persistence interface (memory/BadgerDB)
- [x] Atomic transactions and batch operations
- [x] Data export/import via JSON
- [x] Comprehensive test coverage

## 2. Clustering & Replication
- [ ] Multi-node support with Raft consensus
- [ ] Implement leader election
- [ ] Add data replication across nodes

## 3. Security Features
- [x] Authentication (API keys/JWT)
  - [x] JWT service with token generation and validation
  - [x] Refresh token support
  - [x] API key service with CRUD operations
  - [x] JWT middleware for HTTP handlers
  - [x] API key middleware for HTTP handlers
  - [x] Role-based access (roles in JWT claims)
  - [x] Permission-based access (permissions in API keys)
  - [x] Configurable public paths
  - [x] Auth endpoints (login, refresh, verify)
  - [x] API key management endpoints
  - [x] Comprehensive test coverage
- [x] TLS/SSL support
  - [x] TLS configuration (cert/key files)
  - [x] Auto-generated self-signed certificates for development
  - [x] Environment variable configuration
  - [x] ListenTLS integration with Fiber
- [ ] ACL for KV store access
- [x] Rate limiting per client
  - [x] Token bucket algorithm implementation
  - [x] Per-IP rate limiting
  - [x] Per-API-key rate limiting
  - [x] Configurable rates and burst sizes
  - [x] Rate limit middleware
  - [x] Prometheus metrics for rate limiting
  - [x] Automatic cleanup of unused limiters
  - [x] Comprehensive test coverage
  - [ ] Admin API endpoints for rate limit management
  - [ ] konsulctl commands to view/reset rate limits
  - [ ] konsulctl commands to temporarily adjust rate limits
  - [ ] Rate limit statistics and reporting via CLI

## 4. Monitoring & Metrics
- [x] Prometheus metrics endpoint (/metrics)
- [x] Health check endpoints (/health, /health/live, /health/ready)
- [x] Performance metrics (request latency, throughput, in-flight requests)
- [x] KV store metrics (operations, store size)
- [x] Service discovery metrics (operations, registered services, heartbeats, expired services)
- [x] System metrics (memory, goroutines, build info)
- [x] Dashboard integration (Grafana)
- [x] Web Admin UI (React + Vite + Tailwind CSS)
  - [x] Production build created (332KB JS, 20KB CSS)
  - [ ] Integration with Fiber (serve static files)
  - [ ] Dashboard view (services overview, metrics)
  - [ ] Services management (list, register, deregister)
  - [ ] KV store browser (CRUD operations)
  - [ ] Real-time updates (WebSocket/SSE)
  - [ ] Authentication UI (login, API keys)
  - [ ] Health check visualization
  - [ ] Dark mode support
  - [ ] TypeScript migration
  - [ ] Testing suite (Vitest + React Testing Library)

## 5. Advanced Service Discovery
- [ ] Service tags and metadata
- [ ] Health check URLs (HTTP/TCP checks)
- [ ] Load balancing strategies
- [ ] Service dependencies tracking

## 6. KV Store Enhancements
- [ ] Key prefixes/namespaces
- [ ] Atomic operations (CAS - Compare-And-Swap)
- [ ] Watch/subscribe to key changes
- [ ] Bulk operations

## 7. API Improvements
- [ ] GraphQL interface
- [ ] WebSocket support for real-time updates
- [ ] Batch operations API
- [ ] API versioning (v1, v2)

## 8. Developer Experience
- [x] Docker image with multi-stage build
  - [x] Multi-stage Dockerfile with optimized layers
  - [x] Non-root user security
  - [x] Health checks
  - [x] Build args for versioning
  - [x] Both konsul and konsulctl binaries included
- [x] Kubernetes manifests/Helm chart
  - [x] Complete K8s manifests (namespace, deployment, service, configmap, pvc, rbac)
  - [x] Helm chart with full templating
  - [x] Configurable values for all features
  - [x] ServiceMonitor for Prometheus
  - [x] Ingress support
  - [x] Security contexts and RBAC
- [x] CLI client tool (konsulctl)
  - [x] TLS support for all commands
  - [ ] Rate limit management commands
  - [ ] Rate limit statistics viewing
  - [ ] Admin operations (reset limits, adjust temporarily)
- [ ] SDK/client libraries (Go, Python, JS)

## 9. Consul-Inspired Features (High Value)
- [x] **DNS Interface** - Service discovery via DNS queries (SRV/A records)
- [x] **Advanced Health Checks** - HTTP/TCP/gRPC/script-based checks
- [x] **Template Engine** - Consul-template equivalent for config generation
- [ ] **Multi-Datacenter** - WAN federation and cross-DC service discovery
- [ ] **Service Mesh (Connect)** - mTLS and service-to-service communication
- [ ] **Envoy Proxy Integration** - Sidecar proxy support
- [ ] **Intentions** - Service communication policies
- [ ] **Namespaces** - Multi-tenancy and isolation
- [ ] **Prepared Queries** - Predefined service discovery queries
- [ ] **Events System** - Distributed event broadcasting

## 10. Enterprise-Grade Features
- [ ] **Audit Logging** - Track all operations and changes
- [ ] **RBAC** - Role-based access control
- [ ] **Multi-tenancy** - Namespace isolation with quotas
- [ ] **Disaster Recovery** - Cross-cluster replication
- [ ] **Network Segments** - Service isolation within clusters

## 11. Web Admin UI
- [x] Technology stack selection (React 19 + Vite + Tailwind v4)
- [x] Initial build setup and production bundle
- [ ] Static file serving integration with Fiber
- [ ] Core UI features:
  - [ ] Dashboard with system overview
  - [ ] Services page (list, filter, search)
  - [ ] Service details view
  - [ ] Service registration form
  - [ ] KV store browser
  - [ ] KV editor (create, update, delete)
  - [ ] Settings page
  - [ ] Authentication flow
- [ ] Advanced features:
  - [ ] Real-time service updates (WebSocket)
  - [ ] Service dependency graph
  - [ ] Metrics integration (charts)
  - [ ] Health check history timeline
  - [ ] Dark/light mode toggle
  - [ ] Mobile responsive design
- [ ] Developer improvements:
  - [ ] TypeScript migration
  - [ ] Component library/design system
  - [ ] Testing suite (Vitest)
  - [ ] E2E tests (Playwright)
  - [ ] Accessibility (WCAG compliance)

## Completed Features
✅ Health Check System with TTL
✅ Comprehensive KV Store Testing
✅ Code Organization (handlers separation)
✅ Configuration Management (env variables)
✅ Error Handling & Structured Logging
✅ Zap Logger Integration
✅ JWT Authentication & Authorization
✅ API Key Management System
✅ Authentication Middleware (JWT & API Key)
✅ Role & Permission Based Access Control
✅ Token Bucket Rate Limiting
✅ Per-IP and Per-API-Key Rate Limiting
✅ Rate Limit Metrics & Monitoring
✅ TLS/SSL Support with Auto-Generated Certificates
✅ CLI Tool with TLS Support
✅ Docker Multi-Stage Build (56MB image)
✅ Kubernetes Manifests (Complete YAML)
✅ Helm Chart with Full Configuration
✅ Production-Ready Deployment Options
✅ OpenTelemetry Distributed Tracing
✅ React Admin UI Build (Vite + Tailwind CSS v4)