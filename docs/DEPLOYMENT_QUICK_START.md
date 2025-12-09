# Konsul Deployment Quick Start Guide

**Choose your deployment scenario**:

## 🚀 Quick Decision Matrix

| Your Situation | Recommended Setup | Time to Deploy |
|----------------|-------------------|----------------|
| **Dev/Testing** (1-3 services) | Docker Compose (single node) | 5 minutes |
| **Small Team** (5-20 services) | Docker Compose (single node) | 15 minutes |
| **Startup** (20-100 services) | Kubernetes (3-node cluster) | 30 minutes |
| **Enterprise** (100-500 services) | Kubernetes (5-node + agents) | 2 hours |
| **Global** (500+ services, multi-DC) | Multi-cluster + agents | 1 day |
| **Edge/IoT** (IoT devices) | Cloud + edge nodes | 2 hours |

---

## Scenario 1: Development (5 Minutes)

### One-Command Setup

```bash
# Start Konsul with Docker
docker run -d \
  --name konsul \
  -p 8500:8500 \
  -p 8600:8600 \
  konsul/konsul:latest

# Register a service
docker run -d \
  --name web \
  --link konsul \
  -e KONSUL_ADDRESS=http://konsul:8500 \
  myapp/web:latest

# Verify
curl http://localhost:8500/services
```

**Use Cases**: Development, testing, proof-of-concept

---

## Scenario 2: Small Team (15 Minutes)

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'
services:
  konsul:
    image: konsul:latest
    ports:
      - "8500:8500"
    environment:
      - KONSUL_PERSISTENCE_ENABLED=true
    volumes:
      - konsul-data:/data

  web:
    image: myapp/web:latest
    depends_on:
      - konsul
    command: |
      /bin/sh -c "
      konsulctl service register --name web --address \$HOSTNAME --port 3000
      exec node server.js
      "

volumes:
  konsul-data:
```

```bash
# Deploy
docker-compose up -d

# Check services
docker-compose exec konsul konsulctl service list
```

**Use Cases**: Small teams, staging environments, MVP

---

## Scenario 3: Growing Startup (30 Minutes)

### Kubernetes (3-Node Cluster)

```bash
# Install with Helm
helm repo add konsul https://charts.konsul.io
helm install konsul konsul/konsul \
  --set cluster.replicas=3 \
  --set persistence.enabled=true

# Deploy your app
kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: web
  annotations:
    konsul.io/inject: "true"
    konsul.io/service-name: "web"
spec:
  containers:
  - name: web
    image: myapp/web:latest
    ports:
    - containerPort: 3000
EOF

# Verify
kubectl exec -it konsul-0 -- konsulctl service list
```

**Use Cases**: Production-ready, HA required, growing teams

---

## Scenario 4: Enterprise (2 Hours)

### Kubernetes (5-Node + Agents)

```bash
# Install with enterprise features
helm install konsul konsul/konsul \
  --set cluster.replicas=5 \
  --set agent.enabled=true \
  --set agent.mode=daemonset \
  --set rbac.enabled=true \
  --set acl.enabled=true \
  --set audit.enabled=true \
  --set mesh.enabled=true

# Configure RBAC
konsulctl rbac role create --name developer --policies kv-read,service-read
konsulctl rbac assign --user alice --role developer

# Deploy with auto-injection
kubectl label namespace default konsul-injection=enabled
kubectl apply -f my-services.yaml
```

**Use Cases**: Enterprise, compliance requirements, service mesh

---

## Scenario 5: Multi-Datacenter (1 Day)

### Global Deployment

```bash
# DC1 (Primary)
helm install konsul-dc1 konsul/konsul \
  --set datacenter=us-east \
  --set cluster.replicas=5 \
  --set global.primaryDatacenter=us-east \
  --set meshGateway.enabled=true

# DC2 (Secondary)
helm install konsul-dc2 konsul/konsul \
  --set datacenter=eu-central \
  --set cluster.replicas=5 \
  --set global.federation.enabled=true

# Configure federation
konsulctl datacenter federate --primary us-east --secondary eu-central
```

**Use Cases**: Global services, geo-distribution, multi-region

---

## Common Patterns

### Pattern 1: Service Registration (Sidecar)

```yaml
# In your Kubernetes deployment
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

### Pattern 2: Configuration from KV

```bash
# Set config
konsulctl kv set config/app/db_url "postgresql://db:5432/myapp"

# Read in app
export DB_URL=$(konsulctl kv get config/app/db_url)
```

### Pattern 3: Service Discovery

```bash
# Using konsulctl
API_URL=$(konsulctl service get api --format json | jq -r '.[0].address')

# Using DNS
curl http://api.service.konsul:8080/health

# Using SDK (Go)
services, _ := konsul.Service.Get("api")
```

### Pattern 4: Health Checks

```yaml
metadata:
  annotations:
    konsul.io/service-checks: |
      [
        {
          "http": "http://localhost:3000/health",
          "interval": "10s"
        }
      ]
```

---

## Next Steps

After deployment:

1. **Configure Monitoring**: Enable Prometheus metrics
2. **Setup RBAC**: Create roles and policies
3. **Enable Audit Logging**: Track all operations
4. **Configure Backups**: Automated backup schedule
5. **Read Full Docs**: [ARCHITECTURE_USE_CASES.md](./ARCHITECTURE_USE_CASES.md)

---

## Troubleshooting

### Konsul won't start

```bash
# Check logs
docker logs konsul
kubectl logs konsul-0

# Verify config
konsulctl config validate
```

### Services not registering

```bash
# Check connectivity
curl http://konsul:8500/health

# Verify permissions (if ACL enabled)
konsulctl rbac check --user $USER --resource service --action write
```

### Can't discover services

```bash
# List all services
konsulctl service list

# Check service health
konsulctl service get myservice --health
```

---

## Support

- **Documentation**: [docs/](.)
- **GitHub Issues**: https://github.com/neogan74/konsul/issues
- **Community**: Join our Discord

---

**Last Updated**: 2025-12-06