# Konsul Helm Chart

Official Helm chart for deploying Konsul - a lightweight service discovery and KV store.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PV provisioner support in the underlying infrastructure (if persistence is enabled)

## Installing the Chart

```bash
# Install with default values
helm install konsul ./helm/konsul --namespace konsul --create-namespace

# Install with custom values
helm install konsul ./helm/konsul \
  --namespace konsul \
  --create-namespace \
  --values my-values.yaml
```

## Uninstalling the Chart

```bash
helm uninstall konsul --namespace konsul
```

## Configuration

See [values.yaml](values.yaml) for the full list of configuration options.

### Key Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `konsul` |
| `image.tag` | Image tag | `latest` |
| `service.type` | Service type | `ClusterIP` |
| `persistence.enabled` | Enable persistence | `true` |
| `persistence.data.size` | Data volume size | `1Gi` |
| `persistence.backups.size` | Backup volume size | `2Gi` |
| `config.logLevel` | Log level | `info` |
| `config.auth.enabled` | Enable authentication | `false` |
| `config.tls.enabled` | Enable TLS | `false` |
| `ingress.enabled` | Enable ingress | `false` |
| `serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | `false` |

### Example Configurations

#### Minimal Setup

```yaml
replicaCount: 1

persistence:
  enabled: false
```

#### Production Setup

```yaml
replicaCount: 1

image:
  repository: konsul
  tag: "0.1.0"

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
    size: 10Gi
  backups:
    size: 20Gi

config:
  logLevel: info
  logFormat: json

  auth:
    enabled: true
    jwtSecret: "use-k8s-secret-reference"
    requireAuth: true

  tls:
    enabled: true
    certFile: /certs/tls.crt
    keyFile: /certs/tls.key

  rateLimit:
    enabled: true
    requestsPerSec: 1000
    burst: 100

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
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

## Upgrading

```bash
helm upgrade konsul ./helm/konsul --namespace konsul --values my-values.yaml
```

## Values

For a complete list of values, see [values.yaml](values.yaml).
