# TODO - Konsul Project Roadmap

## 1. Persistence Layer
- [ ] Add optional persistence to disk (BoltDB/BadgerDB)
- [ ] Implement backup/restore functionality
- [ ] Add WAL (Write-Ahead Logging) for crash recovery

## 2. Clustering & Replication
- [ ] Multi-node support with Raft consensus
- [ ] Implement leader election
- [ ] Add data replication across nodes

## 3. Security Features
- [ ] Authentication (API keys/JWT)
- [ ] TLS/SSL support
- [ ] ACL for KV store access
- [ ] Rate limiting per client

## 4. Monitoring & Metrics
- [x] Prometheus metrics endpoint (/metrics)
- [x] Health check endpoints (/health, /health/live, /health/ready)
- [x] Performance metrics (request latency, throughput, in-flight requests)
- [x] KV store metrics (operations, store size)
- [x] Service discovery metrics (operations, registered services, heartbeats, expired services)
- [x] System metrics (memory, goroutines, build info)
- [ ] Dashboard integration (Grafana)

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
- [ ] Docker image with multi-stage build
- [ ] Kubernetes manifests/Helm chart
- [ ] CLI client tool
- [ ] SDK/client libraries (Go, Python, JS)

## Completed Features
✅ Health Check System with TTL
✅ Comprehensive KV Store Testing
✅ Code Organization (handlers separation)
✅ Configuration Management (env variables)
✅ Error Handling & Structured Logging
✅ Zap Logger Integration