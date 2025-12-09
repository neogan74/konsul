# Konsul Architecture & Deployment Use Cases

**Last Updated**: 2025-12-06

This document provides comprehensive architecture guidance for deploying Konsul at different scales, from small teams to large enterprises.

## Table of Contents

1. [Deployment Scenarios Overview](#deployment-scenarios-overview)
2. [Small Team (3-10 Servers)](#scenario-1-small-team-3-10-servers)
3. [Growing Startup (10-50 Servers)](#scenario-2-growing-startup-10-50-servers)
4. [Medium Enterprise (100-500 Servers)](#scenario-3-medium-enterprise-100-500-servers)
5. [Large Enterprise (1000+ Servers, Multi-Cluster)](#scenario-4-large-enterprise-1000-servers-multi-cluster)
6. [Edge/IoT Deployment](#scenario-5-edgeiot-deployment)
7. [Hybrid Cloud](#scenario-6-hybrid-cloud-deployment)
8. [Agent Mode Architecture](#agent-mode-architecture)
9. [Best Practices by Scale](#best-practices-by-scale)

---

## Deployment Scenarios Overview

| Scenario | Scale | Konsul Nodes | Deployment Model | Use Cases |
|----------|-------|--------------|------------------|-----------|
| Small Team | 3-10 servers | 1 (standalone) | Docker Compose | Microservices, simple discovery |
| Growing Startup | 10-50 servers | 3 (HA cluster) | Docker/K8s hybrid | Multi-environment, CI/CD |
| Medium Enterprise | 100-500 servers | 5-node cluster | Kubernetes | Multi-tenant, compliance |
| Large Enterprise | 1000+ servers | Multi-cluster + agents | K8s + agents | Global, multi-DC, service mesh |
| Edge/IoT | 100s-1000s devices | Edge nodes + cloud | Lightweight edge | IoT, edge computing |
| Hybrid Cloud | Variable | Multi-cloud cluster | K8s federation | Cloud portability |

---

## Scenario 1: Small Team (3-10 Servers)

### Overview
**Team Size**: 1-5 developers
**Infrastructure**: 3-10 Docker hosts or VMs
**Services**: 5-20 microservices
**Traffic**: <1000 req/sec

### Architecture

```
┌─────────────────────────────────────────────────────┐
│              Single Konsul Node (Standalone)         │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │  Web     │  │  API     │  │  Worker  │          │
│  │  Service │  │  Service │  │  Service │          │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘          │
│        │             │             │               │
│        └─────────────┴─────────────┘               │
│                      │                             │
│              ┌───────▼────────┐                    │
│              │  Konsul Server │                    │
│              │  (Standalone)  │                    │
│              │  - Service Reg │                    │
│              │  - KV Store    │                    │
│              │  - Health      │                    │
│              └────────────────┘                    │
│                                                     │
└─────────────────────────────────────────────────────┘
```

### Deployment

**1. Deploy Konsul Server (Docker Compose)**

```yaml
# docker-compose.yml
version: '3.8'

services:
  konsul:
    image: konsul:latest
    container_name: konsul-server
    ports:
      - "8500:8500"    # HTTP API
      - "8600:8600"    # DNS
      - "8080:8080"    # Metrics
    environment:
      - KONSUL_ADDRESS=0.0.0.0:8500
      - KONSUL_LOG_LEVEL=info
      - KONSUL_PERSISTENCE_ENABLED=true
      - KONSUL_PERSISTENCE_TYPE=badger
      - KONSUL_PERSISTENCE_DATA_DIR=/data
    volumes:
      - konsul-data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8500/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Your services
  web:
    image: myapp/web:latest
    depends_on:
      - konsul
    environment:
      - KONSUL_ADDRESS=http://konsul:8500
      - SERVICE_NAME=web
      - SERVICE_PORT=3000
    entrypoint: ["/bin/sh", "-c"]
    command:
      - |
        # Register with Konsul on startup
        konsulctl service register --name web --address $HOSTNAME --port 3000 --tags http,frontend

        # Start application
        exec node server.js

  api:
    image: myapp/api:latest
    depends_on:
      - konsul
    environment:
      - KONSUL_ADDRESS=http://konsul:8500
    entrypoint: ["/bin/sh", "-c"]
    command:
      - |
        # Register with Konsul
        konsulctl service register --name api --address $HOSTNAME --port 8080 --tags http,backend

        # Load config from KV store
        export DB_URL=$(konsulctl kv get config/api/db_url)

        # Start application
        exec ./api-server

volumes:
  konsul-data:
```

**2. Service Registration Pattern**

Use **sidecar pattern** with `konsulctl` in entrypoint:

```bash
#!/bin/bash
# entrypoint.sh

# Register service
konsulctl service register \
  --name $SERVICE_NAME \
  --address $HOSTNAME \
  --port $SERVICE_PORT \
  --tags $SERVICE_TAGS

# Trap exit to deregister
trap 'konsulctl service deregister --name $SERVICE_NAME' EXIT

# Start application
exec "$@"
```

**3. Configuration Management**

Store configs in KV store:

```bash
# Setup configs
konsulctl kv set config/api/db_url "postgresql://db:5432/myapp"
konsulctl kv set config/api/redis_url "redis://redis:6379"
konsulctl kv set config/feature_flags/new_ui "true"

# Read in application
DB_URL=$(konsulctl kv get config/api/db_url)
```

**4. Service Discovery**

```bash
# Find service instances
API_URL=$(konsulctl service get api --format json | jq -r '.[0].address')

# Use DNS
curl http://api.service.konsul:8080/health
```

### Pros & Cons

✅ **Pros**:
- Simple setup (single server)
- Low operational overhead
- Docker Compose deployment
- Perfect for development/staging

❌ **Cons**:
- No high availability
- Single point of failure
- Limited to ~1000 req/sec

### When to Upgrade
- Team grows >10 people
- Services >20
- Production traffic >1000 req/sec
- Need HA and zero downtime

---

## Scenario 2: Growing Startup (10-50 Servers)

### Overview
**Team Size**: 5-20 developers
**Infrastructure**: 10-50 servers (Docker + Kubernetes mix)
**Services**: 20-100 microservices
**Traffic**: 1000-10,000 req/sec

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│              3-Node Konsul Cluster (HA)                       │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │   Konsul-1   │  │   Konsul-2   │  │   Konsul-3   │       │
│  │   (Leader)   │◄─┤  (Follower)  │◄─┤  (Follower)  │       │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘       │
│         │                 │                 │               │
│         └─────────────────┴─────────────────┘               │
│                           │                                 │
│  ┌────────────────────────┴─────────────────────────┐       │
│  │                                                   │       │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │       │
│  │  │   Web    │  │   API    │  │  Worker  │       │       │
│  │  │  (K8s)   │  │  (K8s)   │  │  (Docker)│       │       │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘       │       │
│  │       │             │             │             │       │
│  │       └─────────────┴─────────────┘             │       │
│  │                Service Registration              │       │
│  └────────────────────────────────────────────────┘       │
│                                                             │
└──────────────────────────────────────────────────────────────┘
```

### Deployment

**1. Deploy 3-Node Konsul Cluster (Kubernetes)**

```yaml
# konsul-cluster.yaml
apiVersion: v1
kind: Service
metadata:
  name: konsul
  namespace: konsul-system
spec:
  selector:
    app: konsul
  ports:
    - name: http
      port: 8500
      targetPort: 8500
    - name: dns
      port: 8600
      targetPort: 8600
    - name: raft
      port: 7000
      targetPort: 7000
  clusterIP: None  # Headless for StatefulSet
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: konsul
  namespace: konsul-system
spec:
  serviceName: konsul
  replicas: 3
  selector:
    matchLabels:
      app: konsul
  template:
    metadata:
      labels:
        app: konsul
    spec:
      containers:
      - name: konsul
        image: konsul:latest
        env:
        - name: KONSUL_CLUSTER_ENABLED
          value: "true"
        - name: KONSUL_NODE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: KONSUL_RAFT_BIND_ADDR
          value: "0.0.0.0:7000"
        - name: KONSUL_RAFT_JOIN
          value: "konsul-0.konsul.konsul-system.svc.cluster.local:7000,konsul-1.konsul.konsul-system.svc.cluster.local:7000,konsul-2.konsul.konsul-system.svc.cluster.local:7000"
        - name: KONSUL_RAFT_BOOTSTRAP
          value: "$([[ $HOSTNAME == konsul-0 ]] && echo true || echo false)"
        ports:
        - containerPort: 8500
          name: http
        - containerPort: 8600
          name: dns
        - containerPort: 7000
          name: raft
        volumeMounts:
        - name: data
          mountPath: /data
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8500
          initialDelaySeconds: 10
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8500
          initialDelaySeconds: 30
          periodSeconds: 10
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
```

**2. Service Registration in Kubernetes (Init Container Pattern)**

```yaml
# web-service.yaml
apiVersion: v1
kind: Pod
metadata:
  name: web
  annotations:
    konsul.io/service-name: "web"
    konsul.io/service-port: "3000"
    konsul.io/service-tags: "http,frontend"
spec:
  initContainers:
  - name: konsul-register
    image: konsul/konsulctl:latest
    env:
    - name: KONSUL_ADDRESS
      value: "http://konsul.konsul-system.svc.cluster.local:8500"
    - name: SERVICE_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['konsul.io/service-name']
    - name: POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    - name: SERVICE_PORT
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['konsul.io/service-port']
    command:
    - /bin/sh
    - -c
    - |
      konsulctl service register \
        --name $SERVICE_NAME \
        --address $POD_IP \
        --port $SERVICE_PORT \
        --tags $(echo $KONSUL_SERVICE_TAGS | tr ',' ' ')

  containers:
  - name: web
    image: myapp/web:latest
    ports:
    - containerPort: 3000
    env:
    - name: KONSUL_ADDRESS
      value: "http://konsul.konsul-system.svc.cluster.local:8500"

  # Deregister on pod termination
  lifecycle:
    preStop:
      exec:
        command:
        - /bin/sh
        - -c
        - |
          konsulctl service deregister --name web
```

**3. Configuration with ConfigMaps + KV Store**

```bash
# Store sensitive configs in Konsul KV
konsulctl kv set config/prod/db_password "secret123"
konsulctl kv set config/prod/api_key "key456"

# Store non-sensitive in ConfigMap
kubectl create configmap app-config \
  --from-literal=log_level=info \
  --from-literal=environment=production
```

**4. Service Discovery in Application**

```go
// Go example
package main

import (
    "github.com/neogan74/konsul-go-sdk/client"
)

func main() {
    // Create Konsul client
    konsul := client.New(&client.Config{
        Address: "http://konsul.konsul-system.svc.cluster.local:8500",
    })

    // Discover API service
    services, _ := konsul.Service.Get("api")
    apiURL := services[0].Address

    // Load config from KV
    dbPassword, _ := konsul.KV.Get("config/prod/db_password")

    // Use in application
    connectDB(dbPassword)
    callAPI(apiURL)
}
```

### Pros & Cons

✅ **Pros**:
- High availability (3-node cluster)
- Survives single node failure
- Kubernetes-native deployment
- Supports 10K+ req/sec

❌ **Cons**:
- More complex setup
- Requires Kubernetes knowledge
- Higher resource usage

### When to Upgrade
- Services >100
- Traffic >10K req/sec
- Multiple data centers
- Need advanced features (RBAC, audit)

---

## Scenario 3: Medium Enterprise (100-500 Servers)

### Overview
**Team Size**: 20-100 developers
**Infrastructure**: 100-500 servers across multiple K8s clusters
**Services**: 100-500 microservices
**Traffic**: 10K-100K req/sec

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│              5-Node Konsul Cluster + Agents                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────────── Control Plane ──────────────────┐          │
│  │  ┌───────┐  ┌───────┐  ┌───────┐  ┌───────┐  ┌───────┐    │
│  │  │ KS-1  │  │ KS-2  │  │ KS-3  │  │ KS-4  │  │ KS-5  │    │
│  │  │Leader │◄─┤Follower│◄─┤Follower│◄─┤Follower│◄─┤Follower│    │
│  │  └───┬───┘  └───┬───┘  └───┬───┘  └───┬───┘  └───┬───┘    │
│  │      │          │          │          │          │        │
│  │      └──────────┴──────────┴──────────┴──────────┘        │
│  │                          │                                 │
│  └──────────────────────────┼──────────────────────────────────┘
│                             │                                  │
│  ┌──────────────── Data Plane (Agents) ──────────────────┐    │
│  │                          │                            │    │
│  │  ┌─────────┬─────────────┴─────────────┬─────────┐   │    │
│  │  │         │                           │         │   │    │
│  │  ▼         ▼                           ▼         ▼   │    │
│  │ Agent-1  Agent-2  ...  Agent-N      Agent-100       │    │
│  │  │         │             │             │            │    │
│  │  │         │             │             │            │    │
│  │  ▼         ▼             ▼             ▼            │    │
│  │ ┌───────┐ ┌───────┐   ┌───────┐   ┌───────┐       │    │
│  │ │Service│ │Service│   │Service│   │Service│       │    │
│  │ │  Pod  │ │  Pod  │   │  Pod  │   │  Pod  │       │    │
│  │ └───────┘ └───────┘   └───────┘   └───────┘       │    │
│  │                                                     │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Deployment

**1. Deploy 5-Node Control Plane (Helm)**

```bash
# Install Konsul with Helm
helm repo add konsul https://charts.konsul.io
helm repo update

helm install konsul konsul/konsul \
  --namespace konsul-system \
  --create-namespace \
  --set cluster.enabled=true \
  --set cluster.replicas=5 \
  --set persistence.enabled=true \
  --set persistence.size=50Gi \
  --set rbac.enabled=true \
  --set acl.enabled=true \
  --set audit.enabled=true \
  --set metrics.enabled=true \
  --set metrics.serviceMonitor.enabled=true
```

**2. Deploy Konsul Agents (DaemonSet)**

```yaml
# konsul-agent-daemonset.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: konsul-agent
  namespace: konsul-system
spec:
  selector:
    matchLabels:
      app: konsul-agent
  template:
    metadata:
      labels:
        app: konsul-agent
    spec:
      hostNetwork: true
      containers:
      - name: konsul-agent
        image: konsul/agent:latest
        env:
        - name: KONSUL_SERVER_ADDRESS
          value: "http://konsul.konsul-system.svc.cluster.local:8500"
        - name: KONSUL_AGENT_MODE
          value: "true"
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        - name: NODE_IP
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
        ports:
        - containerPort: 8502
          name: agent-api
        volumeMounts:
        - name: varrun
          mountPath: /var/run/konsul
        securityContext:
          privileged: true  # For service mesh features
      volumes:
      - name: varrun
        hostPath:
          path: /var/run/konsul
          type: DirectoryOrCreate
```

**3. Service Registration via Agent (Sidecar)**

```yaml
# service-with-agent.yaml
apiVersion: v1
kind: Pod
metadata:
  name: api-service
  annotations:
    konsul.io/inject-agent: "true"
    konsul.io/service-name: "api"
    konsul.io/service-port: "8080"
spec:
  containers:
  - name: api
    image: myapp/api:latest
    ports:
    - containerPort: 8080
    env:
    - name: KONSUL_AGENT_ADDRESS
      value: "http://localhost:8502"  # Agent sidecar

  # Konsul agent sidecar (auto-injected via mutation webhook)
  - name: konsul-sidecar
    image: konsul/agent:latest
    env:
    - name: KONSUL_SERVER_ADDRESS
      value: "http://konsul.konsul-system.svc.cluster.local:8500"
    - name: POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    - name: SERVICE_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['konsul.io/service-name']
    ports:
    - containerPort: 8502
      name: agent-api
    lifecycle:
      postStart:
        exec:
          command:
          - /bin/sh
          - -c
          - |
            # Register with Konsul server via agent
            curl -X POST http://localhost:8502/register \
              -d "{\"name\":\"$SERVICE_NAME\",\"address\":\"$POD_IP\",\"port\":8080}"
      preStop:
        exec:
          command:
          - /bin/sh
          - -c
          - |
            curl -X DELETE http://localhost:8502/deregister/$SERVICE_NAME
```

**4. Configuration Management with Namespaces**

```bash
# Production namespace
konsulctl kv set --namespace prod config/api/db_url "postgresql://prod-db:5432/app"
konsulctl kv set --namespace prod config/api/cache_ttl "3600"

# Staging namespace
konsulctl kv set --namespace staging config/api/db_url "postgresql://staging-db:5432/app"
konsulctl kv set --namespace staging config/api/cache_ttl "60"

# Read in application (namespace-aware)
export NAMESPACE=${ENVIRONMENT:-prod}
DB_URL=$(konsulctl kv get --namespace $NAMESPACE config/api/db_url)
```

**5. RBAC and Access Control**

```bash
# Create roles
konsulctl rbac role create \
  --name developer \
  --policies kv-read,service-read,health-read

konsulctl rbac role create \
  --name devops \
  --policies kv-write,service-write,admin-read

# Assign roles to users
konsulctl rbac assign --user alice --role developer
konsulctl rbac assign --user bob --role devops

# LDAP integration
konsulctl rbac map-group \
  --group "CN=Engineering,OU=Groups,DC=company,DC=com" \
  --role developer
```

### Agent Mode Benefits

✅ **Advantages**:
1. **Reduced Server Load**: Agents handle local service registration
2. **Better Performance**: Local caching reduces latency
3. **Network Efficiency**: Batch updates to servers
4. **Service Mesh**: Agents can act as sidecar proxies
5. **Health Checks**: Agents perform local health checks

**Agent Architecture**:

```
┌────────────────────────────────────────┐
│           Konsul Agent                  │
├────────────────────────────────────────┤
│                                         │
│  ┌──────────────────────────────────┐  │
│  │   Local Cache                    │  │
│  │   - Service registry             │  │
│  │   - KV store snapshot            │  │
│  │   - Health check results         │  │
│  └──────────────────────────────────┘  │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │   Health Check Engine            │  │
│  │   - HTTP/TCP checks              │  │
│  │   - Local execution              │  │
│  └──────────────────────────────────┘  │
│                                         │
│  ┌──────────────────────────────────┐  │
│  │   Sync Engine                    │  │
│  │   - Batch updates to server      │  │
│  │   - Delta synchronization        │  │
│  └──────────────────────────────────┘  │
│                                         │
└────────────────────────────────────────┘
```

### Pros & Cons

✅ **Pros**:
- Enterprise-grade HA (5-node cluster)
- Agent mode reduces server load
- RBAC and multi-tenancy
- Supports 100K+ req/sec
- Audit logging and compliance

❌ **Cons**:
- Complex deployment
- Higher operational overhead
- Requires dedicated ops team

### When to Upgrade
- Services >500
- Multiple data centers
- Need service mesh
- Global deployment

---

## Scenario 4: Large Enterprise (1000+ Servers, Multi-Cluster)

### Overview
**Team Size**: 100-1000+ developers
**Infrastructure**: 1000+ servers across multiple regions and data centers
**Services**: 500-5000 microservices
**Traffic**: 100K-1M+ req/sec

### Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                    Multi-Datacenter Deployment                        │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌────────────────────┐       ┌────────────────────┐                │
│  │   DC1: US-East     │       │   DC2: US-West     │                │
│  ├────────────────────┤       ├────────────────────┤                │
│  │ 5-Node Cluster     │◄─────►│ 5-Node Cluster     │                │
│  │ + 100 Agents       │  WAN  │ + 100 Agents       │                │
│  └────────────────────┘       └────────────────────┘                │
│           │                            │                             │
│           │                            │                             │
│  ┌────────▼────────┐          ┌───────▼─────────┐                   │
│  │   DC3: EU       │          │   DC4: Asia     │                   │
│  ├─────────────────┤          ├─────────────────┤                   │
│  │ 5-Node Cluster  │◄────────►│ 5-Node Cluster  │                   │
│  │ + 100 Agents    │   WAN    │ + 100 Agents    │                   │
│  └─────────────────┘          └─────────────────┘                   │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │         Federated Service Discovery                      │        │
│  │  - Cross-DC service queries                             │        │
│  │  - Geo-aware routing                                    │        │
│  │  - Global KV replication (selective)                    │        │
│  └─────────────────────────────────────────────────────────┘        │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### Deployment Strategy

**1. Multi-Datacenter Setup**

```bash
# DC1 (Primary - US-East)
helm install konsul-dc1 konsul/konsul \
  --namespace konsul-system \
  --set datacenter=us-east \
  --set cluster.replicas=5 \
  --set global.primaryDatacenter=us-east \
  --set meshGateway.enabled=true \
  --set meshGateway.replicas=3

# DC2 (US-West)
helm install konsul-dc2 konsul/konsul \
  --namespace konsul-system \
  --set datacenter=us-west \
  --set cluster.replicas=5 \
  --set global.primaryDatacenter=us-east \
  --set global.federation.enabled=true \
  --set global.federation.createFederationSecret=true \
  --set meshGateway.enabled=true

# DC3 (EU)
helm install konsul-dc3 konsul/konsul \
  --namespace konsul-system \
  --set datacenter=eu-central \
  --set cluster.replicas=5 \
  --set global.federation.enabled=true

# DC4 (Asia)
helm install konsul-dc4 konsul/konsul \
  --namespace konsul-system \
  --set datacenter=asia-pacific \
  --set cluster.replicas=5 \
  --set global.federation.enabled=true
```

**2. Agent Deployment at Scale (Kubernetes Operator)**

```yaml
# konsul-operator.yaml
apiVersion: konsul.io/v1alpha1
kind: KonsulCluster
metadata:
  name: konsul-cluster
  namespace: konsul-system
spec:
  datacenter: us-east
  replicas: 5

  agent:
    enabled: true
    mode: daemonset
    autoInject: true
    sidecar:
      resources:
        requests:
          cpu: 50m
          memory: 64Mi
        limits:
          cpu: 100m
          memory: 128Mi

  mesh:
    enabled: true
    mTLS: true
    intentions: true

  federation:
    enabled: true
    primaryDatacenter: us-east
    membershipGossip: true

  observability:
    metrics: true
    tracing: true
    logging:
      level: info
```

**3. Automatic Service Registration (Admission Webhook)**

```yaml
# No manual registration needed - webhook injects agent sidecar
apiVersion: v1
kind: Pod
metadata:
  name: api-service
  labels:
    app: api
    version: v1
  # Webhook automatically injects based on labels
spec:
  containers:
  - name: api
    image: myapp/api:v1.0.0
    ports:
    - containerPort: 8080
    env:
    - name: KONSUL_ENABLED
      value: "true"  # Webhook detects this

# After webhook injection:
# - Konsul agent sidecar added
# - Service auto-registered
# - Health checks configured
# - mTLS certificates injected
```

**4. Global Configuration with Replication**

```bash
# Global configs (replicated across DCs)
konsulctl kv set --global config/shared/api_version "v2"
konsulctl kv set --global config/shared/feature_flags/new_auth "true"

# DC-specific configs
konsulctl kv set --datacenter us-east config/local/db_url "postgresql://us-east-db:5432/app"
konsulctl kv set --datacenter eu-central config/local/db_url "postgresql://eu-db:5432/app"

# Read with datacenter awareness
DB_URL=$(konsulctl kv get --datacenter local config/local/db_url)
```

**5. Service Mesh with mTLS**

```yaml
# Service with mesh enabled
apiVersion: v1
kind: Service
metadata:
  name: api
  annotations:
    konsul.io/mesh-enabled: "true"
    konsul.io/mtls-mode: "strict"
spec:
  ports:
  - port: 8080
  selector:
    app: api
---
# Intention (service-to-service authorization)
apiVersion: konsul.io/v1alpha1
kind: ServiceIntentions
metadata:
  name: api-intentions
spec:
  destination:
    name: api
  sources:
  - name: web
    action: allow
  - name: mobile
    action: allow
  - name: "*"
    action: deny
```

**6. Cross-DC Service Discovery**

```go
// Application code with geo-aware discovery
package main

import (
    "github.com/neogan74/konsul-go-sdk/client"
)

func main() {
    konsul := client.New(&client.Config{
        Address: "http://localhost:8502",  // Local agent
        Datacenter: "us-east",
    })

    // Discover service in local DC (low latency)
    localServices, _ := konsul.Service.Get("database", &client.QueryOptions{
        Datacenter: "us-east",
    })

    // Discover service globally with fallback
    globalServices, _ := konsul.Service.Get("payment-api", &client.QueryOptions{
        AllDatacenters: true,
        PreferLocal: true,
    })

    // Use nearest service
    svc := selectNearestService(globalServices)
    callService(svc.Address)
}
```

### Agent Mode at Scale

**Agent Responsibilities**:
1. **Service Registration**: Local services register with agent
2. **Health Checks**: Agent performs local health checks
3. **Caching**: Agent caches service registry and KV store
4. **Service Mesh**: Agent acts as sidecar proxy (Envoy integration)
5. **Batch Updates**: Agent batches updates to reduce server load

**Performance Characteristics**:
- **Registration Latency**: <1ms (local agent)
- **Discovery Latency**: <1ms (cached) or <10ms (server query)
- **Server Load**: Reduced by 90% (agents handle most queries)
- **Network Traffic**: Reduced by 80% (delta sync)

### Monitoring & Observability

```yaml
# Grafana dashboards
- Cluster health per DC
- Agent metrics (registration rate, cache hit ratio)
- Cross-DC replication lag
- Service mesh traffic (request rate, error rate, latency)
- KV operation latency per DC

# Prometheus queries
sum(rate(konsul_agent_registrations_total[5m])) by (datacenter)
konsul_cluster_replication_lag_seconds{dc="us-east"}
sum(konsul_service_instances) by (datacenter, service)
```

### Disaster Recovery

```bash
# Automated backups per DC
konsulctl backup create --datacenter us-east --output /backups/us-east-$(date +%Y%m%d).tar.gz

# Cross-DC replication for DR
konsulctl replication configure \
  --source us-east \
  --target us-west \
  --mode async \
  --lag-limit 5s

# Failover procedure
konsulctl datacenter failover --from us-east --to us-west
```

### Pros & Cons

✅ **Pros**:
- Global scale (1M+ req/sec)
- Multi-DC with geo-aware routing
- Service mesh with mTLS
- 99.99% availability
- Enterprise features (RBAC, audit, compliance)

❌ **Cons**:
- Very complex deployment
- Requires dedicated platform team
- High operational cost
- Network complexity

---

## Scenario 5: Edge/IoT Deployment

### Overview
**Devices**: 100s-1000s of edge devices
**Infrastructure**: Cloud + Edge nodes
**Use Cases**: IoT, CDN, retail, manufacturing

### Architecture

```
┌────────────────────────────────────────────────────────────┐
│                 Cloud Cluster (Central)                     │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────┐              │
│  │  Konsul Server Cluster (5 nodes)        │              │
│  │  - Central registry                     │              │
│  │  - Aggregated metrics                   │              │
│  │  - Policy management                    │              │
│  └──────────────────┬───────────────────────┘              │
│                     │                                      │
│                     │ Sync                                 │
│                     ▼                                      │
│  ┌─────────────────────────────────────────────────────┐  │
│  │        Edge Nodes (Lightweight)                     │  │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐   │  │
│  │  │Edge-1  │  │Edge-2  │  │Edge-N  │  │ IoT    │   │  │
│  │  │<10MB   │  │<10MB   │  │<10MB   │  │Gateway │   │  │
│  │  └────┬───┘  └────┬───┘  └────┬───┘  └────┬───┘   │  │
│  │       │           │           │           │       │  │
│  │       ▼           ▼           ▼           ▼       │  │
│  │  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐   │  │
│  │  │Local   │  │Sensors │  │Cameras │  │Devices │   │  │
│  │  │Services│  │        │  │        │  │        │   │  │
│  │  └────────┘  └────────┘  └────────┘  └────────┘   │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Deployment

```yaml
# Lightweight edge node config
apiVersion: v1
kind: ConfigMap
metadata:
  name: konsul-edge-config
data:
  config.yaml: |
    mode: edge
    server_address: "https://konsul-cloud.company.com:8500"

    # Lightweight mode
    memory_limit: 50Mi
    cache_size: 1000

    # Offline support
    offline_mode: true
    sync_interval: 60s

    # MQTT for IoT
    mqtt:
      enabled: true
      broker: tcp://localhost:1883
      qos: 1
```

---

## Agent Mode Architecture

### Design

```
┌──────────────────────────────────────────────────────────┐
│                  Konsul Agent                             │
├──────────────────────────────────────────────────────────┤
│                                                           │
│  ┌───────────────────────────────────────┐               │
│  │  Agent API (Port 8502)               │               │
│  │  - /register                         │               │
│  │  - /deregister                       │               │
│  │  - /services                         │               │
│  │  - /kv/*                             │               │
│  │  - /health                           │               │
│  └───────────────┬───────────────────────┘               │
│                  │                                       │
│  ┌───────────────▼───────────────────┐                   │
│  │  Local Cache                     │                   │
│  │  - Service Registry (1min TTL)   │                   │
│  │  - KV Store Snapshot            │                   │
│  │  - Health Check Results         │                   │
│  └───────────────┬───────────────────┘                   │
│                  │                                       │
│  ┌───────────────▼───────────────────┐                   │
│  │  Health Check Engine             │                   │
│  │  - HTTP/TCP checks (local)       │                   │
│  │  - Check scheduling              │                   │
│  │  - Result aggregation            │                   │
│  └───────────────┬───────────────────┘                   │
│                  │                                       │
│  ┌───────────────▼───────────────────┐                   │
│  │  Sync Engine                     │                   │
│  │  - Delta sync (only changes)     │                   │
│  │  - Batch updates (every 10s)     │                   │
│  │  - Compression                   │                   │
│  │  - Retry logic                   │                   │
│  └───────────────┬───────────────────┘                   │
│                  │                                       │
│                  ▼                                       │
│         Konsul Server Cluster                           │
│                                                           │
└──────────────────────────────────────────────────────────┘
```

### ADR Needed

We should create **ADR-0026: Agent Mode Architecture** covering:
- Agent responsibilities
- Communication protocol
- Caching strategy
- Health check delegation
- Service mesh integration
- Performance characteristics

---

## Best Practices by Scale

| Scale | Deployment | Registration | Discovery | Config | HA |
|-------|-----------|--------------|-----------|--------|-----|
| **Small** | Single node | konsulctl in entrypoint | DNS/API direct | KV store | Optional |
| **Growing** | 3-node cluster | Init container | DNS/SDK | KV + ConfigMaps | Required |
| **Medium** | 5-node + agents | Agent sidecar | Agent cache | Namespaced KV + RBAC | Required |
| **Large** | Multi-DC + agents | Admission webhook | Agent + geo-aware | Global + local KV | Multi-DC |
| **Edge** | Cloud + edge | MQTT/HTTP | Local cache | Offline sync | Cloud HA |

---

## Summary

### Key Takeaways

1. **Start Simple**: Single node for small teams
2. **Add HA Early**: 3-node cluster as you grow
3. **Use Agents at Scale**: >100 services benefit from agent mode
4. **Multi-DC for Global**: Federation for geo-distributed services
5. **Edge for IoT**: Lightweight nodes for edge computing

### Next Steps

1. **Create ADR-0026**: Agent Mode Architecture
2. **Create ADR-0027**: Multi-Datacenter Federation
3. **Create ADR-0028**: Edge Computing Strategy
4. **Implement Agent Mode**: Priority P1
5. **Build Kubernetes Operator**: Priority P1

---

**Last Updated**: 2025-12-06
**Authors**: Konsul Team
**Review**: Quarterly