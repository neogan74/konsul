# ADR-0029: Kubernetes Operator Design

**Date**: 2025-12-06

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: kubernetes, operator, automation, crds, controller

## Context

As documented in [ARCHITECTURE_USE_CASES.md](../ARCHITECTURE_USE_CASES.md), Kubernetes deployments at medium to large scale require significant manual configuration:

### Current Challenges

**Manual Operations**:
- Service registration via init containers or sidecars
- Configuration management across pods
- Agent deployment and lifecycle
- ACL policy application
- Service mesh configuration
- Upgrade coordination

**Pain Points**:
1. Boilerplate YAML in every deployment
2. Inconsistent service registration patterns
3. Manual agent injection
4. Configuration drift
5. Difficult upgrades
6. No GitOps-friendly workflow

### Requirements

**Automation**:
- Automatic service registration
- Auto-inject Konsul agent sidecars
- Declarative configuration (CRDs)
- GitOps workflows
- Rolling upgrades

**Enterprise**:
- Multi-tenancy (namespace isolation)
- RBAC integration
- Policy as Code
- Compliance (audit, encryption)

## Decision

Implement **Kubernetes Operator** using Kubebuilder framework with Custom Resource Definitions (CRDs) for declarative Konsul management.

### Architecture

```
┌────────────────────────────────────────────────────────────┐
│          Kubernetes Operator Architecture                  │
├────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌────────────── Control Plane ────────────────┐           │
│  │                                              │           │
│  │  ┌────────────────────────────────────┐     │           │
│  │  │  Konsul Operator                   │     │           │
│  │  │  ┌──────────────────────────────┐  │     │           │
│  │  │  │  Controllers:                │  │     │           │
│  │  │  │  - KonsulCluster             │  │     │           │
│  │  │  │  - ServiceEntry              │  │     │           │
│  │  │  │  - ServiceIntentions         │  │     │           │
│  │  │  │  - ACLPolicy                 │  │     │           │
│  │  │  │  - KVConfig                  │  │     │           │
│  │  │  └──────────────────────────────┘  │     │           │
│  │  │  ┌──────────────────────────────┐  │     │           │
│  │  │  │  Mutating Webhook:           │  │     │           │
│  │  │  │  - Agent injection           │  │     │           │
│  │  │  │  - Service annotation        │  │     │           │
│  │  │  └──────────────────────────────┘  │     │           │
│  │  └────────────────────────────────────┘     │           │
│  │                     │                        │           │
│  │                     │ Watch & Reconcile      │           │
│  │                     ▼                        │           │
│  │  ┌────────────────────────────────────┐     │           │
│  │  │  Kubernetes API Server             │     │           │
│  │  │  - Pods, Services, ConfigMaps      │     │           │
│  │  │  - CRDs (Konsul resources)         │     │           │
│  │  └────────────────────────────────────┘     │           │
│  └──────────────────────────────────────────────┘           │
│                                                             │
│  ┌────────────── Data Plane ────────────────┐              │
│  │                                           │              │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  │              │
│  │  │  Pod +  │  │  Pod +  │  │  Pod +  │  │              │
│  │  │  Agent  │  │  Agent  │  │  Agent  │  │              │
│  │  └─────────┘  └─────────┘  └─────────┘  │              │
│  │       │            │            │        │              │
│  │       └────────────┴────────────┘        │              │
│  │                    │                     │              │
│  │                    ▼                     │              │
│  │          ┌─────────────────┐             │              │
│  │          │ Konsul Cluster  │             │              │
│  │          │  (Servers)      │             │              │
│  │          └─────────────────┘             │              │
│  └───────────────────────────────────────────┘              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Custom Resource Definitions (CRDs)

**1. KonsulCluster CRD**:

```yaml
apiVersion: konsul.io/v1alpha1
kind: KonsulCluster
metadata:
  name: konsul
  namespace: konsul-system
spec:
  version: "1.0.0"
  datacenter: us-east

  # Server configuration
  servers:
    replicas: 5
    storage:
      size: 50Gi
      storageClass: fast-ssd
    resources:
      requests:
        cpu: "1"
        memory: 2Gi
      limits:
        cpu: "2"
        memory: 4Gi

  # Agent configuration
  agents:
    enabled: true
    mode: sidecar  # or daemonset
    autoInject: true
    resources:
      requests:
        cpu: 50m
        memory: 64Mi

  # Features
  features:
    acl:
      enabled: true
      defaultPolicy: deny
    mesh:
      enabled: true
      mTLS: true
    audit:
      enabled: true
      sink: file
    metrics:
      enabled: true
      serviceMonitor: true

  # Federation (multi-DC)
  federation:
    enabled: false
    primaryDatacenter: us-east

status:
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2025-12-06T10:00:00Z"
  phase: Running
  serverReplicas: 5
  agentReplicas: 100
```

**2. ServiceEntry CRD** (Declarative Service Registration):

```yaml
apiVersion: konsul.io/v1alpha1
kind: ServiceEntry
metadata:
  name: api-service
  namespace: default
spec:
  serviceName: api
  port: 8080
  tags:
    - http
    - backend
    - v1
  metadata:
    team: platform
    environment: production

  # Health checks
  healthChecks:
    - type: http
      http: http://localhost:8080/health
      interval: 10s
      timeout: 2s
    - type: tcp
      tcp: localhost:8080
      interval: 30s

  # Mesh configuration
  mesh:
    enabled: true
    upstreams:
      - name: database
        datacenter: us-east
      - name: cache
        datacenter: us-east
```

**3. ServiceIntentions CRD** (Service-to-Service Authorization):

```yaml
apiVersion: konsul.io/v1alpha1
kind: ServiceIntentions
metadata:
  name: api-intentions
  namespace: default
spec:
  destination:
    name: api
  sources:
    - name: web
      action: allow
      description: "Web frontend can call API"

    - name: mobile
      action: allow
      description: "Mobile app can call API"

    - name: "*"
      action: deny
      description: "Deny all other services"
```

**4. ACLPolicy CRD**:

```yaml
apiVersion: konsul.io/v1alpha1
kind: ACLPolicy
metadata:
  name: developer-policy
  namespace: konsul-system
spec:
  rules:
    kv:
      - path: "config/app/*"
        capabilities: [read, list]
      - path: "secrets/*"
        capabilities: [deny]

    service:
      - name: "web-*"
        capabilities: [read, write, register, deregister]
      - name: "database"
        capabilities: [read]

    health:
      capabilities: [read]
```

**5. KVConfig CRD** (Declarative KV Management):

```yaml
apiVersion: konsul.io/v1alpha1
kind: KVConfig
metadata:
  name: app-config
  namespace: default
spec:
  prefix: "config/app/"
  entries:
    - key: db_url
      value: "postgresql://db:5432/myapp"
    - key: cache_ttl
      value: "3600"
    - key: feature_flags/new_ui
      value: "true"

  # Source from ConfigMap
  fromConfigMap:
    name: app-config
    namespace: default
```

### Controllers

**KonsulCluster Controller**:

```go
package controllers

import (
    "context"
    konsulv1alpha1 "github.com/neogan74/konsul-operator/api/v1alpha1"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type KonsulClusterReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *KonsulClusterReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
    // Fetch KonsulCluster resource
    cluster := &konsulv1alpha1.KonsulCluster{}
    if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
        return reconcile.Result{}, client.IgnoreNotFound(err)
    }

    // Reconcile server StatefulSet
    if err := r.reconcileServers(ctx, cluster); err != nil {
        return reconcile.Result{}, err
    }

    // Reconcile agent DaemonSet or webhook
    if cluster.Spec.Agents.Enabled {
        if err := r.reconcileAgents(ctx, cluster); err != nil {
            return reconcile.Result{}, err
        }
    }

    // Reconcile mesh gateway
    if cluster.Spec.Features.Mesh.Enabled {
        if err := r.reconcileMeshGateway(ctx, cluster); err != nil {
            return reconcile.Result{}, err
        }
    }

    // Update status
    cluster.Status.Phase = "Running"
    cluster.Status.ServerReplicas = cluster.Spec.Servers.Replicas
    if err := r.Status().Update(ctx, cluster); err != nil {
        return reconcile.Result{}, err
    }

    return reconcile.Result{}, nil
}

func (r *KonsulClusterReconciler) reconcileServers(ctx context.Context, cluster *konsulv1alpha1.KonsulCluster) error {
    // Create or update StatefulSet
    sts := &appsv1.StatefulSet{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "konsul-server",
            Namespace: cluster.Namespace,
        },
        Spec: appsv1.StatefulSetSpec{
            Replicas: &cluster.Spec.Servers.Replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "app": "konsul-server",
                },
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "app": "konsul-server",
                    },
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:      "konsul",
                            Image:     fmt.Sprintf("konsul/konsul:%s", cluster.Spec.Version),
                            Resources: cluster.Spec.Servers.Resources,
                            // ... environment, ports, etc.
                        },
                    },
                },
            },
            VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
                {
                    ObjectMeta: metav1.ObjectMeta{
                        Name: "data",
                    },
                    Spec: corev1.PersistentVolumeClaimSpec{
                        AccessModes: []corev1.PersistentVolumeAccessMode{
                            corev1.ReadWriteOnce,
                        },
                        Resources: corev1.ResourceRequirements{
                            Requests: corev1.ResourceList{
                                corev1.ResourceStorage: cluster.Spec.Servers.Storage.Size,
                            },
                        },
                        StorageClassName: &cluster.Spec.Servers.Storage.StorageClass,
                    },
                },
            },
        },
    }

    // Set owner reference
    ctrl.SetControllerReference(cluster, sts, r.Scheme)

    // Create or update
    return r.createOrUpdate(ctx, sts)
}
```

**ServiceEntry Controller**:

```go
func (r *ServiceEntryReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
    serviceEntry := &konsulv1alpha1.ServiceEntry{}
    if err := r.Get(ctx, req.NamespacedName, serviceEntry); err != nil {
        return reconcile.Result{}, client.IgnoreNotFound(err)
    }

    // Register service with Konsul
    konsulClient := r.getKonsulClient()
    err := konsulClient.Service.Register(&Service{
        Name:     serviceEntry.Spec.ServiceName,
        Port:     serviceEntry.Spec.Port,
        Tags:     serviceEntry.Spec.Tags,
        Meta:     serviceEntry.Spec.Metadata,
        Checks:   convertHealthChecks(serviceEntry.Spec.HealthChecks),
    })

    if err != nil {
        return reconcile.Result{}, err
    }

    // Update status
    serviceEntry.Status.Registered = true
    serviceEntry.Status.LastSync = metav1.Now()
    return reconcile.Result{}, r.Status().Update(ctx, serviceEntry)
}
```

### Mutating Webhook (Agent Injection)

```go
package webhook

import (
    "context"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PodInjector struct {
    decoder *admission.Decoder
}

func (p *PodInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
    pod := &corev1.Pod{}
    if err := p.decoder.Decode(req, pod); err != nil {
        return admission.Errored(http.StatusBadRequest, err)
    }

    // Check if injection enabled
    if pod.Annotations["konsul.io/inject"] != "true" {
        return admission.Allowed("injection not requested")
    }

    // Inject Konsul agent sidecar
    pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
        Name:  "konsul-agent",
        Image: "konsul/agent:latest",
        Env: []corev1.EnvVar{
            {
                Name:  "KONSUL_SERVER_ADDRESS",
                Value: "http://konsul-server.konsul-system.svc.cluster.local:8500",
            },
            {
                Name: "POD_IP",
                ValueFrom: &corev1.EnvVarSource{
                    FieldRef: &corev1.ObjectFieldSelector{
                        FieldPath: "status.podIP",
                    },
                },
            },
        },
        Ports: []corev1.ContainerPort{
            {Name: "agent-api", ContainerPort: 8502},
        },
    })

    // Add init container for service registration
    pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
        Name:  "konsul-register",
        Image: "konsul/konsulctl:latest",
        Command: []string{
            "/bin/sh", "-c",
            "konsulctl service register --name ${SERVICE_NAME} --address ${POD_IP} --port ${SERVICE_PORT}",
        },
    })

    // Marshal modified pod
    marshaledPod, err := json.Marshal(pod)
    if err != nil {
        return admission.Errored(http.StatusInternalServerError, err)
    }

    return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
```

### GitOps Workflow

**Declarative Service Definition**:

```yaml
# gitops-repo/services/api/konsul.yaml
apiVersion: konsul.io/v1alpha1
kind: ServiceEntry
metadata:
  name: api
  namespace: production
spec:
  serviceName: api
  port: 8080
  tags: [http, backend, v1.2.0]
  healthChecks:
    - type: http
      http: http://localhost:8080/health
      interval: 10s
---
apiVersion: konsul.io/v1alpha1
kind: KVConfig
metadata:
  name: api-config
  namespace: production
spec:
  prefix: "config/api/"
  entries:
    - key: db_url
      value: "postgresql://prod-db:5432/api"
    - key: cache_ttl
      value: "3600"
```

**Argo CD / Flux Integration**:

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: api-service
  namespace: argocd
spec:
  source:
    repoURL: https://github.com/company/gitops-repo
    path: services/api
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Implementation Phases

**Phase 1: Core Operator (4 weeks)**
1. Kubebuilder scaffolding
2. KonsulCluster CRD and controller
3. Server StatefulSet reconciliation
4. Basic agent deployment

**Phase 2: Service Management (3 weeks)**
1. ServiceEntry CRD and controller
2. Service registration automation
3. Health check integration
4. KVConfig CRD

**Phase 3: Agent Injection (2 weeks)**
1. Mutating webhook setup
2. Automatic sidecar injection
3. Init container registration
4. Lifecycle management

**Phase 4: Advanced Features (3 weeks)**
1. ServiceIntentions CRD
2. ACLPolicy CRD
3. Mesh configuration
4. Federation support

**Phase 5: Production Readiness (2 weeks)**
1. E2E tests
2. Upgrade testing
3. Helm chart
4. Documentation

**Total**: 14 weeks (~3.5 months)

## Alternatives Considered

### Alternative 1: Helm Charts Only
- **Reason for rejection**: Not declarative, manual service registration

### Alternative 2: Service Mesh Operator (Istio/Linkerd)
- **Reason for rejection**: Different scope, Konsul has more than mesh

### Alternative 3: Manual kubectl apply
- **Reason for rejection**: No automation, error-prone

## Consequences

### Positive
- **Declarative management** via CRDs
- **GitOps-friendly** workflows
- **Auto-injection** reduces boilerplate
- **Simplified operations**
- **Kubernetes-native**

### Negative
- **K8s-specific** (doesn't help non-K8s)
- **Learning curve** (CRDs, operators)
- **Additional component** to maintain

## References

- [Kubebuilder](https://book.kubebuilder.io/)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [ARCHITECTURE_USE_CASES.md](../ARCHITECTURE_USE_CASES.md)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-06 | Konsul Team | Initial version |