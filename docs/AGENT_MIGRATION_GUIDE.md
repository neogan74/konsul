# Konsul Agent Migration Guide

**Version**: 1.0
**Last Updated**: 2025-12-28
**Audience**: Platform Engineers, DevOps Teams, SREs

## Overview

This guide helps you migrate from server-only Konsul deployment to agent-based architecture for improved performance, reduced server load, and sub-millisecond local operations.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Architecture Comparison](#architecture-comparison)
- [Migration Benefits](#migration-benefits)
- [Migration Strategies](#migration-strategies)
- [Step-by-Step Migration](#step-by-step-migration)
- [Verification & Testing](#verification--testing)
- [Rollback Procedure](#rollback-procedure)
- [Performance Tuning](#performance-tuning)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

## Prerequisites

### Required

- Konsul server version ≥ 0.1.0 (with agent protocol support)
- Kubernetes cluster version ≥ 1.19
- `kubectl` access with cluster-admin permissions
- Prometheus operator (for monitoring)
- At least 64Mi RAM per node for agents

### Recommended

- Grafana for dashboard visualization
- Existing Konsul monitoring setup
- Backup of current Konsul data
- Load testing environment

## Architecture Comparison

### Before: Server-Only Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   App Pod    │     │   App Pod    │     │   App Pod    │
│              │     │              │     │              │
│  Direct API  │     │  Direct API  │     │  Direct API  │
│  Calls to    │     │  Calls to    │     │  Calls to    │
│  Server      │     │  Server      │     │  Server      │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
       └────────────────────┴────────────────────┘
                            │
                    ┌───────▼────────┐
                    │ Konsul Server  │
                    │ (High Load)    │
                    │ Port 8888      │
                    └────────────────┘
```

**Characteristics**:
- All requests hit the server directly
- High server load (100% baseline)
- Network latency for every request
- Server becomes bottleneck at scale
- Single point of contention

### After: Agent-Based Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   App Pod    │     │   App Pod    │     │   App Pod    │
│              │     │              │     │              │
│  Local API   │     │  Local API   │     │  Local API   │
│  localhost   │     │  localhost   │     │  localhost   │
│  :8502       │     │  :8502       │     │  :8502       │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
┌──────▼───────┐     ┌──────▼───────┐     ┌──────▼───────┐
│ Agent (Node1)│     │ Agent (Node2)│     │ Agent (Node3)│
│ LRU Cache    │     │ LRU Cache    │     │ LRU Cache    │
│ Health Checks│     │ Health Checks│     │ Health Checks│
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
       └────────────────────┴────────────────────┘
                            │
                     (Periodic Sync)
                            │
                    ┌───────▼────────┐
                    │ Konsul Server  │
                    │ (10% Load)     │
                    │ Port 8888      │
                    └────────────────┘
```

**Characteristics**:
- Apps call local agent (localhost:8502)
- 90% server load reduction
- <1ms response time (cache hits)
- Horizontal scalability (one agent per node)
- Distributed health checks

## Migration Benefits

### Performance Improvements

| Metric | Before (Server-Only) | After (Agent-Based) | Improvement |
|--------|---------------------|---------------------|-------------|
| **Service Discovery Latency** | 5-20ms | <1ms (cache hit) | **20x faster** |
| **KV Read Latency** | 3-15ms | <1ms (cache hit) | **15x faster** |
| **Server Load** | 100% baseline | ~10% | **90% reduction** |
| **Network Calls** | Every request | Sync only (every 10s) | **99% reduction** |
| **Cache Hit Rate** | N/A | >95% | **New capability** |

### Operational Benefits

✅ **Reduced server costs** - Smaller server instances needed
✅ **Better resilience** - Agent continues serving from cache during server maintenance
✅ **Improved reliability** - No single point of failure for reads
✅ **Local health checks** - Faster detection, lower network overhead
✅ **Horizontal scalability** - Performance scales with node count

### When NOT to Use Agents

❌ Single-node deployments (unnecessary complexity)
❌ Very dynamic environments (cache churn >50%)
❌ Strict consistency requirements for every read
❌ Memory-constrained nodes (<64Mi available)

## Migration Strategies

### Strategy 1: Phased Rollout (Recommended)

**Best for**: Production environments, risk-averse teams

**Timeline**: 2-4 weeks

**Phases**:
1. Deploy agents to dev/staging (Week 1)
2. Monitor and validate (Week 1-2)
3. Deploy to production non-critical namespaces (Week 2-3)
4. Full production rollout (Week 3-4)

**Pros**: Low risk, easy rollback, validation at each step
**Cons**: Longer timeline, mixed architecture temporarily

### Strategy 2: Blue-Green Deployment

**Best for**: Kubernetes environments with spare capacity

**Timeline**: 1 week

**Phases**:
1. Deploy agents alongside existing setup
2. Migrate traffic gradually using selectors
3. Validate and cut over
4. Remove old configuration

**Pros**: Fast rollback, minimal downtime
**Cons**: Requires extra resources temporarily

### Strategy 3: Big Bang

**Best for**: Non-production, testing environments

**Timeline**: 1-2 days

**Phases**:
1. Deploy agents
2. Update all applications
3. Validate

**Pros**: Fastest migration
**Cons**: Higher risk, requires downtime window

## Step-by-Step Migration

### Phase 1: Preparation (Day 1)

#### 1.1. Verify Prerequisites

```bash
# Check Kubernetes version
kubectl version --short

# Check available resources
kubectl top nodes

# Verify Konsul server version
kubectl exec -n konsul-system deployment/konsul -- /konsul version

# Check Prometheus is running
kubectl get pods -n monitoring | grep prometheus
```

#### 1.2. Backup Existing Data

```bash
# Backup KV store
kubectl exec -n konsul-system deployment/konsul -- \
  /konsul backup create /tmp/backup.tar.gz

# Copy backup locally
kubectl cp konsul-system/konsul-xxx:/tmp/backup.tar.gz ./konsul-backup-$(date +%Y%m%d).tar.gz

# Backup service registrations (export as JSON)
curl http://konsul-server:8888/services/ > services-backup-$(date +%Y%m%d).json
```

#### 1.3. Review Current Metrics

```bash
# Current server load
kubectl top pods -n konsul-system

# Current request rate
curl http://konsul-server:8888/metrics | grep http_requests_total
```

### Phase 2: Agent Deployment (Day 1-2)

#### 2.1. Create Namespace (if not exists)

```bash
kubectl create namespace konsul-system
```

#### 2.2. Deploy Agent Infrastructure

```bash
# Deploy all agent components
kubectl apply -k k8s/agent/

# Verify deployment
kubectl get daemonset -n konsul-system konsul-agent
kubectl get pods -n konsul-system -l app=konsul-agent
```

Expected output:
```
NAME           DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE
konsul-agent   3         3         3       3            3
```

#### 2.3. Verify Agent Health

```bash
# Check agent pods are running
kubectl get pods -n konsul-system -l app=konsul-agent

# Check logs
kubectl logs -n konsul-system -l app=konsul-agent --tail=50

# Test agent health endpoint
kubectl port-forward -n konsul-system daemonset/konsul-agent 8502:8502 &
curl http://localhost:8502/health
```

Expected response:
```json
{
  "status": "healthy",
  "agent_id": "agent-node1-xxx",
  "uptime": "5m30s"
}
```

### Phase 3: Application Migration (Day 2-7)

#### 3.1. Update Application Configuration

**Option A: Environment Variable**

```yaml
# Before
env:
  - name: KONSUL_ADDR
    value: "http://konsul-server.konsul-system.svc.cluster.local:8888"

# After
env:
  - name: KONSUL_ADDR
    value: "http://localhost:8502"
```

**Option B: ConfigMap**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  konsul_url: "http://localhost:8502"  # Changed from server:8888
```

#### 3.2. Migrate One Namespace at a Time

```bash
# Start with non-critical namespace
kubectl set env deployment/my-app -n dev \
  KONSUL_ADDR=http://localhost:8502

# Wait and verify
kubectl rollout status deployment/my-app -n dev

# Check application logs
kubectl logs -n dev deployment/my-app --tail=100 | grep -i konsul
```

#### 3.3. Verify Service Discovery Works

```bash
# From application pod
kubectl exec -n dev deployment/my-app -- \
  curl http://localhost:8502/agent/catalog/service/my-service

# Should return service entries from cache
```

### Phase 4: Validation (Day 7-14)

#### 4.1. Monitor Cache Performance

```bash
# Get cache hit rate
curl http://localhost:8502/agent/metrics | grep cache_hit_rate

# Expected: >0.95 (95%+)
```

#### 4.2. Check Server Load Reduction

```bash
# Before migration baseline
kubectl top pods -n konsul-system -l app=konsul-server

# After migration (should be ~10% of baseline)
kubectl top pods -n konsul-system -l app=konsul-server
```

#### 4.3. Verify Sync is Working

```bash
# Check sync metrics
kubectl exec -n konsul-system daemonset/konsul-agent -- \
  curl http://localhost:8502/agent/stats

# Look for:
# - last_sync_time: recent timestamp
# - sync_errors_total: 0 or very low
```

### Phase 5: Production Rollout (Day 14-28)

#### 5.1. Gradual Production Migration

```bash
# Migrate production namespaces one at a time
for ns in prod-app1 prod-app2 prod-app3; do
  echo "Migrating namespace: $ns"

  # Update deployments
  kubectl set env deployment --all -n $ns \
    KONSUL_ADDR=http://localhost:8502

  # Wait for rollout
  kubectl wait --for=condition=available --timeout=5m \
    deployment --all -n $ns

  # Verify
  kubectl get pods -n $ns

  # Wait before next namespace
  sleep 300  # 5 minutes
done
```

#### 5.2. Monitor Each Rollout

```bash
# Watch for errors
kubectl get events -n $ns --sort-by='.lastTimestamp'

# Check application health
kubectl get pods -n $ns

# Verify agent metrics
curl http://localhost:8502/agent/stats
```

## Verification & Testing

### Functional Tests

```bash
# Test service registration
curl -X POST http://localhost:8502/agent/service/register \
  -d '{
    "name": "test-service",
    "address": "10.0.0.1",
    "port": 8080,
    "tags": ["test"]
  }'

# Test service discovery
curl http://localhost:8502/agent/catalog/service/test-service

# Test KV operations
curl -X PUT http://localhost:8502/agent/kv/test-key \
  -d '{"value": "test-value"}'

curl http://localhost:8502/agent/kv/test-key

# Test health check registration
curl -X POST http://localhost:8502/agent/check/register \
  -d '{
    "service_id": "test-service",
    "name": "http-check",
    "http": "http://localhost:8080/health",
    "interval": "10s"
  }'
```

### Performance Tests

```bash
# Benchmark cache performance
ab -n 10000 -c 100 http://localhost:8502/agent/catalog/service/my-service

# Expected results:
# - Requests per second: >10,000
# - Mean latency: <1ms
# - 99th percentile: <2ms
```

### Chaos Tests

```bash
# Test agent resilience (stop server temporarily)
kubectl scale deployment/konsul-server -n konsul-system --replicas=0

# Agent should continue serving from cache
curl http://localhost:8502/agent/catalog/service/my-service
# Should still work!

# Restore server
kubectl scale deployment/konsul-server -n konsul-system --replicas=3

# Verify sync resumes
kubectl logs -n konsul-system -l app=konsul-agent | grep "sync completed"
```

## Rollback Procedure

### Quick Rollback (< 5 minutes)

```bash
# Revert applications to use server directly
kubectl set env deployment/my-app \
  KONSUL_ADDR=http://konsul-server.konsul-system.svc.cluster.local:8888

# Wait for rollout
kubectl rollout status deployment/my-app

# Optionally, remove agents
kubectl delete daemonset -n konsul-system konsul-agent
```

### Full Rollback

```bash
# 1. Update all applications
kubectl get deployments --all-namespaces -o json | \
  jq '.items[] | select(.spec.template.spec.containers[].env[]? | select(.name=="KONSUL_ADDR")) | .metadata.namespace + "/" + .metadata.name' | \
  xargs -I {} kubectl set env deployment {} \
    KONSUL_ADDR=http://konsul-server.konsul-system.svc.cluster.local:8888

# 2. Remove agent infrastructure
kubectl delete -k k8s/agent/

# 3. Verify server is handling traffic
kubectl logs -n konsul-system deployment/konsul-server | grep "http_requests"
```

## Performance Tuning

### Agent Configuration Optimization

```yaml
# k8s/agent/configmap.yaml adjustments

# For high-churn environments
cache:
  service_ttl: 30s  # Reduce from 60s
  kv_ttl: 120s      # Reduce from 300s

# For stable environments
cache:
  service_ttl: 120s  # Increase from 60s
  kv_ttl: 600s       # Increase from 300s

# For high-load scenarios
sync:
  interval: 5s       # Increase from 10s
  batch_size: 200    # Increase from 100
```

### Resource Tuning

```yaml
# k8s/agent/daemonset.yaml

resources:
  requests:
    cpu: 100m      # Increase if CPU-bound
    memory: 128Mi  # Increase if cache-bound
  limits:
    cpu: 500m      # Allow bursting
    memory: 512Mi  # Allow growth
```

## Troubleshooting

### Problem: Low Cache Hit Rate (<80%)

**Symptoms**:
```bash
curl http://localhost:8502/agent/metrics | grep cache_hit_rate
# Output: 0.65 (65%)
```

**Diagnosis**:
```bash
# Check cache churn
kubectl logs -n konsul-system daemonset/konsul-agent | grep "cache miss"

# Check TTL configuration
kubectl get configmap -n konsul-system konsul-agent-config -o yaml
```

**Solutions**:
1. Increase cache TTL
2. Increase max_entries if evicting due to size
3. Investigate if services are truly that dynamic

### Problem: Sync Errors

**Symptoms**:
```bash
kubectl logs -n konsul-system daemonset/konsul-agent | grep error
# Output: "sync failed: connection refused"
```

**Diagnosis**:
```bash
# Check server connectivity
kubectl exec -n konsul-system daemonset/konsul-agent -- \
  curl -v http://konsul-server.konsul-system.svc.cluster.local:8888/health

# Check network policies
kubectl get networkpolicies -n konsul-system
```

**Solutions**:
1. Verify server is running
2. Check network policies allow agent->server
3. Verify DNS resolution

### Problem: High Memory Usage

**Symptoms**:
```bash
kubectl top pods -n konsul-system -l app=konsul-agent
# Output: 512Mi/512Mi (OOMKilled)
```

**Diagnosis**:
```bash
# Check cache size
curl http://localhost:8502/agent/metrics | grep cache_entries

# Check for memory leaks
kubectl logs -n konsul-system daemonset/konsul-agent --previous
```

**Solutions**:
1. Reduce max_entries in cache config
2. Reduce cache TTL
3. Increase memory limits
4. Check for memory leaks (file issue)

## FAQ

### Q: Can I use agents with single-server Konsul?
**A**: Yes! Agents work with single-server deployments. You get caching benefits, though HA requires clustered servers.

### Q: What happens if an agent crashes?
**A**: The agent restarts automatically (Kubernetes restarts it). During restart, applications briefly use stale DNS or fail until the agent recovers. Consider setting appropriate liveness/readiness probes.

### Q: How do I monitor agent performance?
**A**: Use the Grafana dashboard at `k8s/agent/grafana-dashboard.json`. Import it into your Grafana instance to monitor cache hit rates, sync status, and performance.

### Q: Can agents run outside Kubernetes?
**A**: Yes! Build the agent binary and run it on any Linux/Windows/macOS system. Configure it via YAML file or environment variables.

### Q: Does agent mode support TLS?
**A**: Yes! Configure TLS in the agent ConfigMap under the `tls` section. See `k8s/agent/README.md` for examples.

### Q: What's the network overhead of agents?
**A**: Minimal. Agents sync every 10s by default (configurable). Bandwidth is <1KB/s per agent for typical workloads.

### Q: Can I use agents with service mesh (Istio/Linkerd)?
**A**: Yes! Agents work alongside service meshes. The mesh handles traffic routing, Konsul handles service discovery and configuration.

### Q: How do I upgrade agents?
**A**: Update the image tag in `k8s/agent/kustomization.yaml` and run `kubectl apply -k k8s/agent/`. DaemonSet performs rolling updates automatically.

### Q: What's the blast radius if agent config is wrong?
**A**: Limited to the node running the agent. Other nodes continue unaffected. Fix the config and Kubernetes will restart the agent.

### Q: Can I run multiple agents per node?
**A**: Not recommended. The DaemonSet ensures one agent per node. Multiple agents would compete for localhost:8502.

## Next Steps

After successful migration:

1. ✅ **Set up Monitoring**: Import Grafana dashboard
2. ✅ **Configure Alerting**: Set alerts for cache hit rate <90%, sync errors
3. ✅ **Performance Baseline**: Document new baseline metrics
4. ✅ **Update Runbooks**: Include agent troubleshooting steps
5. ✅ **Train Team**: Conduct knowledge transfer on agent architecture

## Support

- **Documentation**: `k8s/agent/README.md`
- **Architecture**: `docs/adr/0026-agent-mode-architecture.md`
- **Issues**: https://github.com/neogan74/konsul/issues
- **Discussions**: https://github.com/neogan74/konsul/discussions

---

**Migration Guide Version**: 1.0
**Last Updated**: 2025-12-28
**Feedback**: contributions@konsul.io