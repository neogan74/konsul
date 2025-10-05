# Deployment Guide

This guide covers deploying Konsul using Docker, Kubernetes, and Helm.

## Docker Deployment

### Building the Image

```bash
# Simple build
docker build -t konsul:latest .

# Build with version info
docker build -t konsul:0.1.0 \
  --build-arg VERSION=0.1.0 \
  --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
  --build-arg VCS_REF=$(git rev-parse --short HEAD) \
  .
```

### Running with Docker

**Basic deployment:**
```bash
docker run -d \
  --name konsul \
  -p 8888:8888 \
  -p 8600:8600/udp \
  -p 8600:8600/tcp \
  konsul:latest
```

**With persistence:**
```bash
docker run -d \
  --name konsul \
  -p 8888:8888 \
  -p 8600:8600/udp \
  -e KONSUL_PERSISTENCE_ENABLED=true \
  -e KONSUL_PERSISTENCE_TYPE=badger \
  -v konsul-data:/app/data \
  -v konsul-backups:/app/backups \
  konsul:latest
```

**With TLS:**
```bash
docker run -d \
  --name konsul \
  -p 8888:8888 \
  -e KONSUL_TLS_ENABLED=true \
  -e KONSUL_TLS_AUTO_CERT=true \
  -v konsul-certs:/app/certs \
  konsul:latest
```

**Full production setup:**
```bash
docker run -d \
  --name konsul \
  -p 8888:8888 \
  -p 8600:8600/udp \
  -e KONSUL_LOG_FORMAT=json \
  -e KONSUL_LOG_LEVEL=info \
  -e KONSUL_PERSISTENCE_ENABLED=true \
  -e KONSUL_TLS_ENABLED=true \
  -e KONSUL_TLS_AUTO_CERT=true \
  -e KONSUL_AUTH_ENABLED=true \
  -e KONSUL_JWT_SECRET=your-secret-key-min-32-chars \
  -e KONSUL_RATE_LIMIT_ENABLED=true \
  -v konsul-data:/app/data \
  -v konsul-backups:/app/backups \
  konsul:latest
```

### Docker Compose

Create `docker-compose.yaml`:
```yaml
version: '3.8'

services:
  konsul:
    image: konsul:latest
    ports:
      - "8888:8888"
      - "8600:8600/udp"
      - "8600:8600/tcp"
    environment:
      KONSUL_LOG_FORMAT: json
      KONSUL_LOG_LEVEL: info
      KONSUL_PERSISTENCE_ENABLED: "true"
      KONSUL_PERSISTENCE_TYPE: badger
      KONSUL_TLS_ENABLED: "true"
      KONSUL_TLS_AUTO_CERT: "true"
    volumes:
      - konsul-data:/app/data
      - konsul-backups:/app/backups
      - konsul-certs:/app/certs
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8888/health/live"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s

volumes:
  konsul-data:
  konsul-backups:
  konsul-certs:
```

Run with:
```bash
docker-compose up -d
```

## Kubernetes Deployment

### Using Raw Manifests

**1. Create namespace and resources:**
```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/serviceaccount.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

**2. Verify deployment:**
```bash
kubectl get pods -n konsul
kubectl logs -n konsul -l app.kubernetes.io/name=konsul
```

**3. Access the service:**
```bash
# Port forward for local access
kubectl port-forward -n konsul svc/konsul 8888:8888

# Or use kubectl proxy
kubectl proxy
# Access at: http://localhost:8001/api/v1/namespaces/konsul/services/konsul:http/proxy/
```

### Custom Configuration

Edit `k8s/configmap.yaml` to customize settings:
```yaml
data:
  KONSUL_LOG_LEVEL: "debug"
  KONSUL_PERSISTENCE_ENABLED: "true"
  KONSUL_AUTH_ENABLED: "true"
  # ... other settings
```

Then apply:
```bash
kubectl apply -f k8s/configmap.yaml
kubectl rollout restart deployment/konsul -n konsul
```

## Helm Deployment

### Installing the Chart

**1. Add repository (if published):**
```bash
helm repo add konsul https://charts.konsul.io
helm repo update
```

**2. Install from local chart:**
```bash
# Basic installation
helm install konsul ./helm/konsul

# Install in specific namespace
helm install konsul ./helm/konsul --namespace konsul --create-namespace

# Install with custom values
helm install konsul ./helm/konsul \
  --namespace konsul \
  --create-namespace \
  --values custom-values.yaml
```

### Configuration

Create `custom-values.yaml`:
```yaml
replicaCount: 1

image:
  repository: konsul
  tag: "0.1.0"
  pullPolicy: IfNotPresent

resources:
  limits:
    cpu: 1000m
    memory: 512Mi
  requests:
    cpu: 200m
    memory: 128Mi

persistence:
  enabled: true
  storageClass: "fast-ssd"
  data:
    size: 5Gi
  backups:
    size: 10Gi

config:
  logLevel: info
  logFormat: json

  auth:
    enabled: true
    jwtSecret: "your-super-secret-key-min-32-chars"
    requireAuth: true

  tls:
    enabled: true
    autoCert: true

  rateLimit:
    enabled: true
    requestsPerSec: 1000
    burst: 100

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: konsul.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: konsul-tls
      hosts:
        - konsul.example.com

serviceMonitor:
  enabled: true
  interval: 30s
```

Install with custom values:
```bash
helm install konsul ./helm/konsul \
  --namespace konsul \
  --create-namespace \
  --values custom-values.yaml
```

### Helm Operations

**Upgrade:**
```bash
helm upgrade konsul ./helm/konsul \
  --namespace konsul \
  --values custom-values.yaml
```

**Rollback:**
```bash
helm rollback konsul -n konsul
```

**Uninstall:**
```bash
helm uninstall konsul -n konsul
```

**Check status:**
```bash
helm status konsul -n konsul
helm get values konsul -n konsul
```

## Production Considerations

### High Availability

For HA deployment, consider:
- Multiple replicas (when clustering is implemented)
- Pod anti-affinity rules
- Resource limits and requests
- Persistent storage with backup/restore

### Security

**Enable TLS:**
```yaml
config:
  tls:
    enabled: true
    certFile: /certs/tls.crt
    keyFile: /certs/tls.key
```

**Enable Authentication:**
```yaml
config:
  auth:
    enabled: true
    jwtSecret: "use-a-k8s-secret-here"
    requireAuth: true
```

**Store secrets properly:**
```bash
kubectl create secret generic konsul-jwt-secret \
  --from-literal=secret=your-super-secret-key \
  -n konsul
```

Then reference in values:
```yaml
config:
  auth:
    enabled: true
    jwtSecretRef: konsul-jwt-secret
```

### Monitoring

**Enable Prometheus:**
```yaml
serviceMonitor:
  enabled: true
  interval: 30s
```

**Add Grafana dashboard:**
- Import dashboard from `docs/grafana-dashboard.json`
- Connect to Prometheus data source
- View metrics for KV operations, services, health checks

### Backup Strategy

**Automated backups with CronJob:**
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: konsul-backup
  namespace: konsul
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: konsul:latest
            command:
            - konsulctl
            - backup
            - create
            - --server
            - http://konsul:8888
          restartPolicy: OnFailure
```

### Resource Planning

Recommended resources based on load:

**Small (< 100 services, < 1000 keys):**
```yaml
resources:
  requests:
    cpu: 100m
    memory: 64Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

**Medium (< 1000 services, < 10000 keys):**
```yaml
resources:
  requests:
    cpu: 500m
    memory: 256Mi
  limits:
    cpu: 2000m
    memory: 1Gi
```

**Large (> 1000 services, > 10000 keys):**
```yaml
resources:
  requests:
    cpu: 1000m
    memory: 512Mi
  limits:
    cpu: 4000m
    memory: 2Gi
```

## Troubleshooting

**Check logs:**
```bash
# Docker
docker logs konsul

# Kubernetes
kubectl logs -n konsul -l app.kubernetes.io/name=konsul

# Follow logs
kubectl logs -n konsul -l app.kubernetes.io/name=konsul -f
```

**Check health:**
```bash
# Docker
curl http://localhost:8888/health

# Kubernetes
kubectl exec -n konsul deploy/konsul -- wget -qO- http://localhost:8888/health
```

**Debug pod issues:**
```bash
kubectl describe pod -n konsul -l app.kubernetes.io/name=konsul
kubectl get events -n konsul --sort-by='.lastTimestamp'
```

**Access konsulctl inside container:**
```bash
# Docker
docker exec -it konsul konsulctl kv list

# Kubernetes
kubectl exec -n konsul deploy/konsul -- konsulctl kv list
```
