# Konsul Agent Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the Konsul Agent as a DaemonSet.

## Overview

The Konsul Agent runs on every Kubernetes node and provides:
- **Local caching** of service discovery and KV data (90% server load reduction)
- **Local health checks** for services running on the same node
- **Automatic synchronization** with the Konsul server
- **Sub-millisecond response times** for cached data

## Architecture

```
┌─────────────────────────────────────────┐
│           Konsul Server                  │
│         (konsul-system ns)               │
│         Port: 8888                       │
└──────────────┬──────────────────────────┘
               │
               │ Sync Protocol (/v1/agent/*)
               │
     ┌─────────┴──────────┬────────────────┐
     │                    │                │
┌────▼────┐         ┌────▼────┐     ┌────▼────┐
│ Agent   │         │ Agent   │     │ Agent   │
│ Node 1  │         │ Node 2  │     │ Node N  │
│:8502    │         │:8502    │     │:8502    │
└─────────┘         └─────────┘     └─────────┘
     │                    │                │
     └────────────────────┴────────────────┘
            Local Services Access
            (Apps use localhost:8502)
```

## Prerequisites

1. Kubernetes cluster (1.19+)
2. `kubectl` configured to access your cluster
3. Konsul server deployed and running

## Quick Start

### 1. Create Namespace

```bash
kubectl create namespace konsul-system
```

### 2. Deploy with Kustomize

```bash
# Deploy all manifests
kubectl apply -k k8s/agent/

# Or deploy individually
kubectl apply -f k8s/agent/rbac.yaml
kubectl apply -f k8s/agent/configmap.yaml
kubectl apply -f k8s/agent/daemonset.yaml
kubectl apply -f k8s/agent/service.yaml
```

### 3. Verify Deployment

```bash
# Check DaemonSet status
kubectl get daemonset konsul-agent -n konsul-system

# Check pods (one per node)
kubectl get pods -n konsul-system -l app=konsul-agent

# View logs
kubectl logs -n konsul-system -l app=konsul-agent --tail=50

# Check agent health
kubectl port-forward -n konsul-system daemonset/konsul-agent 8502:8502
curl http://localhost:8502/health
```

## Configuration

### ConfigMap

Edit `configmap.yaml` to customize:

```yaml
# Server connection
server_address: "http://konsul-server.konsul-system.svc.cluster.local:8888"

# Cache settings
cache:
  service_ttl: 60s
  kv_ttl: 300s
  max_entries: 10000

# Sync interval
sync:
  interval: 10s
  full_sync_interval: 300s
```

### Environment Variables

The DaemonSet automatically injects:
- `NODE_NAME` - Kubernetes node name
- `NODE_IP` - Node IP address
- `POD_NAME` - Pod name
- `POD_NAMESPACE` - Namespace
- `ZONE` - Availability zone (from node labels)
- `REGION` - Region (from node labels)

### Resource Limits

Adjust in `daemonset.yaml`:

```yaml
resources:
  requests:
    cpu: 50m
    memory: 64Mi
  limits:
    cpu: 200m
    memory: 256Mi
```

## Using the Agent

### From Application Pods

Applications should connect to the local agent instead of the server:

```bash
# Service discovery
curl http://localhost:8502/agent/catalog/service/my-service

# KV store
curl http://localhost:8502/agent/kv/config/app

# Health checks
curl http://localhost:8502/agent/checks
```

### Service Registration

Register services with the local agent:

```bash
curl -X POST http://localhost:8502/agent/service/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-service",
    "address": "10.0.0.1",
    "port": 8080,
    "tags": ["v1", "production"]
  }'
```

### Health Check Registration

```bash
curl -X POST http://localhost:8502/agent/check/register \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": "my-service",
    "name": "http-check",
    "http": "http://localhost:8080/health",
    "interval": "10s",
    "timeout": "5s"
  }'
```

## Monitoring

### Metrics

The agent exposes Prometheus metrics at `/agent/metrics`:

```bash
# Access metrics
kubectl port-forward -n konsul-system daemonset/konsul-agent 8502:8502
curl http://localhost:8502/agent/metrics
```

Key metrics:
- `konsul_agent_cache_hit_rate` - Cache hit rate
- `konsul_agent_cache_entries` - Total cached entries
- `konsul_agent_sync_count` - Total syncs performed
- `konsul_agent_sync_errors` - Sync error count

### Agent Statistics

```bash
# Get agent stats
curl http://localhost:8502/agent/stats
```

### Health Check

```bash
# Check agent health
curl http://localhost:8502/health

# Get agent info
curl http://localhost:8502/agent/self
```

## Troubleshooting

### Agent Not Starting

```bash
# Check pod events
kubectl describe pod -n konsul-system -l app=konsul-agent

# View logs
kubectl logs -n konsul-system -l app=konsul-agent

# Check RBAC permissions
kubectl auth can-i list pods --as=system:serviceaccount:konsul-system:konsul-agent
```

### Connection Issues

```bash
# Test server connectivity from agent pod
kubectl exec -n konsul-system -it <agent-pod> -- \
  curl -v http://konsul-server.konsul-system.svc.cluster.local:8888/health

# Check network policies
kubectl get networkpolicies -n konsul-system
```

### Cache Not Working

```bash
# Check cache stats
kubectl exec -n konsul-system -it <agent-pod> -- \
  curl http://localhost:8502/agent/metrics | grep cache

# Force full sync
kubectl exec -n konsul-system -it <agent-pod> -- \
  curl -X POST http://localhost:8502/agent/sync
```

## Advanced Configuration

### TLS Encryption

Enable TLS between agent and server:

1. Create TLS secrets:
```bash
kubectl create secret generic konsul-agent-tls \
  -n konsul-system \
  --from-file=ca.crt=path/to/ca.crt \
  --from-file=client.crt=path/to/client.crt \
  --from-file=client.key=path/to/client.key
```

2. Update ConfigMap:
```yaml
tls:
  enabled: true
  ca_cert: /etc/konsul/tls/ca.crt
  client_cert: /etc/konsul/tls/client.crt
  client_key: /etc/konsul/tls/client.key
```

3. Mount secrets in DaemonSet:
```yaml
volumeMounts:
  - name: tls
    mountPath: /etc/konsul/tls
    readOnly: true
volumes:
  - name: tls
    secret:
      secretName: konsul-agent-tls
```

### Node Selector

Run agents only on specific nodes:

```yaml
nodeSelector:
  konsul-agent: "true"
```

Label nodes:
```bash
kubectl label nodes <node-name> konsul-agent=true
```

### Host Network

The DaemonSet uses `hostNetwork: true` by default so applications can access the agent via `localhost:8502`.

To disable:
```yaml
hostNetwork: false
dnsPolicy: ClusterFirst
```

Then applications must use the Service DNS:
```
konsul-agent.konsul-system.svc.cluster.local:8502
```

## Cleanup

```bash
# Delete all agent resources
kubectl delete -k k8s/agent/

# Or delete individually
kubectl delete -f k8s/agent/daemonset.yaml
kubectl delete -f k8s/agent/service.yaml
kubectl delete -f k8s/agent/configmap.yaml
kubectl delete -f k8s/agent/rbac.yaml
```

## Performance

Expected performance with agent:
- **Cache Hit Rate**: >95%
- **Response Time**: <1ms for cached data
- **Server Load Reduction**: ~90%
- **Memory Usage**: ~64-128Mi per agent
- **CPU Usage**: ~50-100m per agent

## Next Steps

- Configure sidecar injection for automatic agent usage
- Set up Prometheus monitoring
- Implement custom health checks
- Configure watched KV prefixes for your applications
