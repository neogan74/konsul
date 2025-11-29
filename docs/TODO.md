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
- [x] ACL system (Access Control Lists)
  - [x] Policy-based authorization with resource types (kv, service, health, backup, admin)
  - [x] Path/name pattern matching with wildcards (* and **)
  - [x] Policy evaluator with deny-by-default security model
  - [x] Policy CRUD API endpoints
  - [x] Policy file storage and loading from directory
  - [x] JWT token integration (policies in claims)
  - [x] ACL middleware for authorization enforcement
  - [x] Dynamic ACL middleware with automatic resource/capability inference
  - [x] konsulctl CLI commands for policy management
  - [x] ACL testing endpoint for debugging permissions
  - [x] Comprehensive test coverage (evaluator, handlers, middleware)
  - [x] Documentation (acl-guide.md, ADR-0010, policy examples)
- [x] Rate limiting per client
  - [x] Token bucket algorithm implementation
  - [x] Per-IP rate limiting
  - [x] Per-API-key rate limiting
  - [x] Configurable rates and burst sizes
  - [x] Rate limit middleware
  - [x] Prometheus metrics for rate limiting
  - [x] Automatic cleanup of unused limiters
  - [x] Comprehensive test coverage
  - [x] Admin API endpoints for rate limit management
  - [x] konsulctl commands to view/reset rate limits
  - [x] konsulctl commands to temporarily adjust rate limits
  - [x] Rate limit statistics and reporting via CLI

## 4. Monitoring & Metrics
- [x] Prometheus metrics endpoint (/metrics)
- [x] Health check endpoints (/health, /health/live, /health/ready)
- [x] Performance metrics (request latency, throughput, in-flight requests)
- [x] KV store metrics (operations, store size)
- [x] Service discovery metrics (operations, registered services, heartbeats, expired services)
- [x] System metrics (memory, goroutines, build info)
- [x] Dashboard integration (Grafana)
- [x] Web Admin UI (React + Vite + Tailwind CSS)
  - [x] Production build created (358KB JS, 20KB CSS)
  - [x] Integration with Fiber (serve static files)
  - [x] Dashboard view (services overview, metrics)
  - [x] Services management (list, register, deregister)
  - [x] KV store browser (CRUD operations)
  - [x] Health check visualization
  - [x] Authentication UI (login, API keys, logout)
  - [x] TypeScript (fully implemented)
  - [x] Protected routes with JWT authentication
  - [x] User menu with role display
  - [x] API Key management page (CRUD, revoke, copy)
  - [x] Automatic token refresh with axios interceptors
  - [ ] Real-time updates (WebSocket/SSE)
  - [ ] Light mode and theme toggle
  - [ ] Testing suite (Vitest + React Testing Library)

## 5. Advanced Service Discovery
- [x] Service tags and metadata
  - [x] Tag-based service queries (`/services/query/tags`)
  - [x] Metadata-based service queries (`/services/query/metadata`)
  - [x] Combined tag+metadata queries (`/services/query`)
  - [x] Service indexing by tags and metadata
  - [x] Validation for tags and metadata
  - [x] Comprehensive tests (index, query, validation)
  - [x] GraphQL integration for tags/metadata queries
  - [x] Documentation (service-tags-metadata-examples.md)
- [ ] Health check URLs (HTTP/TCP checks)
- [x] Load balancing strategies
  - [x] Round-robin load balancing
  - [x] Select by service name (`/lb/service/:name`)
  - [x] Select by tags (`/lb/tags`)
  - [x] Select by metadata (`/lb/metadata`)
  - [x] Select by combined query (`/lb/query`)
  - [x] Strategy configuration endpoint (`/lb/strategy`)
  - [x] Prometheus metrics integration
  - [x] Comprehensive tests
  - [x] Documentation (api-tags-metadata-loadbalancing.md)
- [ ] Service dependencies tracking

## 6. KV Store Enhancements
- [ ] Key prefixes/namespaces
- [x] Atomic operations (CAS - Compare-And-Swap)
- [x] Watch/subscribe to key changes
  - [x] WatchManager for managing watchers
  - [x] Pattern matching support (exact, *, **)
  - [x] WebSocket transport for real-time updates
  - [x] Server-Sent Events (SSE) transport
  - [x] ACL integration (event filtering by permissions)
  - [x] Per-client limits to prevent resource exhaustion
  - [x] Prometheus metrics for monitoring
  - [x] KVStore integration (notifies on Set/Delete/BatchSet/BatchDelete)
  - [x] Comprehensive test coverage
  - [x] CLI command (konsulctl kv watch)
  - [x] Full integration into main.go
  - [x] Client examples and documentation (kv-watch-guide.md)
- [ ] Bulk operations

## 7. API Improvements
- [x] GraphQL interface
- [x] WebSocket support for real-time updates (KV watch)
- [x] Batch operations API
  - [x] Batch KV Get (`POST /batch/kv/get`)
  - [x] Batch KV Set (`POST /batch/kv/set`)
  - [x] Batch KV Delete (`POST /batch/kv/delete`)
  - [x] Batch Service Get (`POST /batch/services/get`)
  - [x] Batch Service Register (`POST /batch/services/register`)
  - [x] Batch Service Deregister (`POST /batch/services/deregister`)
  - [x] Request validation and size limits
  - [x] Audit logging integration
  - [x] Prometheus metrics
  - [x] Comprehensive unit tests (14 tests)
  - [x] Full API documentation (batch-operations-api.md)
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
  - [x] Rate limit management commands
  - [x] Rate limit statistics viewing
  - [x] Admin operations (reset limits, adjust temporarily)
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
- [x] **Audit Logging** - Track all operations and changes
  - [x] Core audit package with async event manager
  - [x] File and stdout sinks with buffering
  - [x] HTTP middleware with action mappers
  - [x] Environment variable configuration
  - [x] Prometheus metrics (events, drops, flush duration)
  - [x] Applied to all critical routes (KV, service, ACL, backup, admin)
  - [x] Comprehensive documentation and examples
  - [x] Unit and integration tests (19 tests)
  - [x] Production-ready with graceful shutdown
  - [x] SIEM-ready JSON format
  - [x] Compliance support (SOC 2, HIPAA, PCI DSS, GDPR)
- [ ] **RBAC** - Role-based access control (enhanced beyond current ACL)
- [ ] **Multi-tenancy** - Namespace isolation with quotas
- [ ] **Disaster Recovery** - Cross-cluster replication
- [ ] **Network Segments** - Service isolation within clusters

## 11. Web Admin UI
- [x] Technology stack selection (React 19 + Vite + Tailwind v4)
- [x] Initial build setup and production bundle
- [x] Static file serving integration with Fiber
- [x] Core UI features:
  - [x] Dashboard with system overview (services, KV, health, uptime)
  - [x] Services page (list, filter, register, deregister, heartbeat)
  - [x] Service registration form (name, address, port, tags)
  - [x] KV store browser (list, search, pagination)
  - [x] KV editor (create, update, delete with JSON validation)
  - [x] Authentication flow (login, logout, protected routes)
  - [x] API Keys management page (create, revoke, delete, copy)
  - [x] Health page (service stats, system metrics, memory usage)
- [x] TypeScript (fully implemented with strict mode)
- [x] Mobile responsive design (works on all screen sizes)
- [ ] Settings page
- [ ] Advanced features:
  - [ ] Real-time service updates (WebSocket)
  - [ ] Service dependency graph
  - [ ] Metrics integration (charts/graphs)
  - [ ] Health check history timeline
  - [ ] Light/dark mode toggle
- [ ] Developer improvements:
  - [ ] Testing suite (Vitest + React Testing Library)
  - [ ] E2E tests (Playwright)
  - [ ] Accessibility improvements (WCAG compliance)

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
✅ Rate Limiting Admin API (Stats, Reset, Config Management)
✅ konsulctl Rate Limit Commands (View, Reset, Update)
✅ TLS/SSL Support with Auto-Generated Certificates
✅ CLI Tool with TLS Support
✅ Docker Multi-Stage Build (56MB image)
✅ Kubernetes Manifests (Complete YAML)
✅ Helm Chart with Full Configuration
✅ Production-Ready Deployment Options
✅ OpenTelemetry Distributed Tracing
✅ React Admin UI Build (Vite + Tailwind CSS v4)
✅ ACL System (Policy-based Authorization)
✅ ACL Middleware & Dynamic Resource Inference
✅ ACL CLI Commands & Management API
✅ KV Watch/Subscribe System (WebSocket & SSE)
✅ Watch Manager with Pattern Matching
✅ Watch Prometheus Metrics
✅ konsulctl kv watch CLI Command
✅ Watch Documentation & Examples (JavaScript, Go, curl)
✅ Admin UI Static File Serving Integration with Fiber
✅ Audit Logging System (Enterprise-Grade)
✅ Audit Event Capture with Async Buffering
✅ File & Stdout Audit Sinks
✅ Audit Middleware for All Critical Routes
✅ SIEM-Ready JSON Audit Logs
✅ Audit Metrics & Monitoring
✅ Service Tags & Metadata Querying
✅ Service Indexing by Tags/Metadata
✅ Load Balancing Strategies (Round-Robin)
✅ Load Balancer API Endpoints
✅ GraphQL Service Tags/Metadata Integration
✅ Batch Operations API (KV & Services)
✅ Batch KV Get/Set/Delete Operations
✅ Batch Service Register/Deregister/Get Operations
✅ Compare-And-Swap (CAS) Operations (Atomic KV & Service Updates)
✅ Admin UI Authentication System (Login, Logout, Protected Routes, User Menu)
✅ Admin UI API Key Management (CRUD, Revoke, Copy to Clipboard)
✅ Axios Interceptors for Automatic Token Refresh