# Research: Namespace/Multi-Tenancy Patterns in Service Discovery Platforms

## 1. Consul Enterprise Namespaces

**HTTP API**: Both URL path and header supported:
- Header: `X-Consul-Namespace: team-a` (preferred for existing clients)
- Query param: `?ns=team-a`
- Path prefix: `/v1/kv/team-a/mykey` (Consul uses header, NOT path for namespace)

**Key behaviors**:
- Default namespace: `"default"` — all OSS data implicitly in "default"
- Namespace is a first-class resource: CRUD at `/v1/namespace`
- KV keys are scoped: same key `foo` in ns `a` and ns `b` are independent
- Services: same service name can exist in multiple namespaces
- ACL tokens are namespace-scoped; policies/roles can be global or namespace-local
- Cross-namespace service discovery allowed via `<svc>.svc.<ns>.dc.<dc>`

**Migration path**: Existing data stays in `"default"` namespace; zero breaking changes.

## 2. Kubernetes Namespace Patterns

**Isolation model**:
- All resources (pods, services, configmaps) are namespace-scoped
- Cluster-scoped resources (nodes, PVs) exist outside namespaces
- DNS: `<service>.<namespace>.svc.cluster.local`
- RBAC: RoleBinding is namespace-scoped; ClusterRoleBinding is global

**Key design decisions**:
- Namespace `"default"` exists always; `kube-system` for infra
- ResourceQuota and LimitRange scoped to namespace
- NetworkPolicy scopes firewall rules to namespace

## 3. Backwards-Compatible Namespace Introduction

**Strategy** (used by both Consul and K8s):
1. Introduce `"default"` as implicit namespace for all existing data
2. All existing API paths work unchanged (no namespace = "default")
3. New namespace-aware paths are additive
4. Header-based namespace selection: `X-Konsul-Namespace: default`
5. Migration: prefix-scan all BadgerDB keys, rewrite under `ns:default:` prefix

**Risk**: Key migration requires a one-time data migration or dual-read logic during transition.

## 4. HTTP API Design for Namespaces

**Option A — URL Path prefix** (e.g. `/ns/team-a/kv/mykey`):
- Pros: explicit, REST-pure, easy to route in Fiber
- Cons: breaking change to existing clients, longer paths
- Used by: AWS (account ID in path), some internal platforms

**Option B — HTTP Header** (e.g. `X-Konsul-Namespace: team-a`):
- Pros: zero breaking change, clients opt-in
- Cons: less visible, harder to discover, curl-unfriendly
- Used by: Consul Enterprise

**Option C — Query param** (e.g. `?namespace=team-a`):
- Pros: easy to add to any client, visible in logs/URLs
- Cons: must be added to every request, leaks in URLs

**Recommendation**: **Hybrid** — support header (backwards compat) + query param (discoverability) + optional path prefix for new `/ns/` routes. Default = `"default"` when absent.

## 5. Go Middleware Patterns for Namespace Extraction

```go
// Typed context key (avoids collisions)
type contextKey string
const namespaceKey contextKey = "namespace"

// Fiber middleware
func NamespaceMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        ns := c.Get("X-Konsul-Namespace")
        if ns == "" {
            ns = c.Query("namespace", "default")
        }
        if !isValidNamespace(ns) {
            return fiber.ErrBadRequest
        }
        c.Locals("namespace", ns)
        return c.Next()
    }
}

// Extract in handler
ns := c.Locals("namespace").(string)
```

**Validation rules**: `[a-z0-9][a-z0-9-]{0,62}` (DNS-compatible), reserved: `system`, `default`.

## Key Design Decisions for Konsul

| Decision | Choice | Rationale |
|---|---|---|
| Namespace location in HTTP | Header + query param | Zero breaking change |
| Default namespace name | `"default"` | Convention (Consul/K8s) |
| KV key format in BadgerDB | `ns:<ns>:<key>` | Prefix-scannable |
| Service namespace field | Added to Service struct | Explicit scoping |
| RBAC scope | Namespace-aware assignments | Matches Consul pattern |
| Namespace resource | CRUD API + persistence | First-class entity |

## Unresolved Questions
- Cross-namespace service discovery: allowed or blocked by default?
- Namespace quota limits (max services, KV keys per namespace)?
- Raft: namespace operations need their own commands or reuse existing with ns field?
