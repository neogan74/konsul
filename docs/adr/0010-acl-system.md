# ADR-0010: Access Control List (ACL) System

**Date**: 2025-10-09

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: security, authorization, acl, rbac, access-control

## Context

Konsul currently has authentication (JWT and API keys) but lacks fine-grained authorization. Current limitations:

### Current State (ADR-0003)
- **Authentication**: JWT tokens and API keys identify users
- **Coarse-grained authorization**: All-or-nothing access via `REQUIRE_AUTH`
- **Role claims**: JWT contains roles, but not enforced granularly
- **API key permissions**: Permissions array exists but not enforced

### Problems

1. **All users have full access** once authenticated
2. **Cannot restrict KV operations** (e.g., read-only users)
3. **Cannot limit service operations** (e.g., only register, not deregister)
4. **No path-based access control** (e.g., allow `/kv/app1/*`, deny `/kv/app2/*`)
5. **No audit trail** of who did what
6. **Cannot implement multi-tenancy** without authorization
7. **Compliance requirements** need fine-grained access control

### Requirements

1. **Fine-grained control**: Per-resource, per-operation authorization
2. **Path-based rules**: Wildcards and prefix matching
3. **Deny by default**: Secure by default, explicit allow
4. **Multiple resources**: KV store, services, health checks, backups, admin
5. **Role-based**: Group permissions into reusable roles
6. **Policy language**: Human-readable, version-controllable
7. **Performance**: Minimal latency overhead (<1ms)
8. **Backward compatible**: Existing setups continue working
9. **Audit ready**: Log authorization decisions

## Decision

We will implement a **Consul-inspired ACL system** with the following design:

### Architecture

**ACL Components**:
1. **Policies**: Define permissions (rules)
2. **Tokens**: JWT/API keys linked to policies
3. **Rules**: Resource + path + action specifications
4. **Evaluator**: Fast policy matching engine
5. **Middleware**: Authorization enforcement layer

### Policy Format

HCL-inspired syntax (parsed as JSON internally):

```hcl
# Example policy: developer.hcl
policy "developer" {
  description = "Developer with limited access"

  kv {
    path "app/config/*" {
      capabilities = ["read", "list"]
    }

    path "app/secrets/*" {
      capabilities = ["deny"]
    }
  }

  service {
    name "web-*" {
      capabilities = ["read", "write", "register", "deregister"]
    }

    name "database" {
      capabilities = ["read"]
    }
  }

  health {
    capabilities = ["read"]
  }
}

# Example policy: admin.hcl
policy "admin" {
  description = "Full administrative access"

  kv {
    path "*" {
      capabilities = ["read", "write", "list", "delete"]
    }
  }

  service {
    name "*" {
      capabilities = ["read", "write", "register", "deregister"]
    }
  }

  health {
    capabilities = ["read", "write"]
  }

  backup {
    capabilities = ["create", "restore", "list", "delete"]
  }

  admin {
    capabilities = ["read", "write"]
  }
}

# Example policy: readonly.hcl
policy "readonly" {
  description = "Read-only access to everything"

  kv {
    path "*" {
      capabilities = ["read", "list"]
    }
  }

  service {
    name "*" {
      capabilities = ["read"]
    }
  }

  health {
    capabilities = ["read"]
  }
}
```

### Resource Types

**1. KV Store** (`kv`)
- Capabilities: `read`, `write`, `list`, `delete`, `deny`
- Path matching: `/kv/:key` → check against path rules

**2. Services** (`service`)
- Capabilities: `read`, `write`, `register`, `deregister`, `list`, `deny`
- Name matching: Service name → check against name rules

**3. Health Checks** (`health`)
- Capabilities: `read`, `write`, `list`, `deny`
- Applies to health check endpoints

**4. Backups** (`backup`)
- Capabilities: `create`, `restore`, `list`, `delete`, `export`, `import`, `deny`
- Controls backup/restore operations

**5. Admin** (`admin`)
- Capabilities: `read`, `write`, `deny`
- Controls auth, metrics, and system endpoints

### Token-Policy Binding

**JWT Claims Extension**:
```json
{
  "user_id": "user123",
  "username": "alice",
  "roles": ["developer"],
  "policies": ["developer", "kv-readonly"],
  "exp": 1234567890
}
```

**API Key Metadata**:
```json
{
  "id": "key-123",
  "name": "ci-pipeline",
  "policies": ["ci-deploy"],
  "created_at": "2025-10-09T10:00:00Z"
}
```

### Evaluation Algorithm

```
1. Extract token from request (JWT or API key)
2. Get policies attached to token
3. For each policy:
   a. Match resource type (kv, service, etc.)
   b. Match path/name (with wildcard support)
   c. Check if capability includes requested action
   d. If explicit "deny" → DENY immediately
   e. If match with allow → mark as allowed
4. If any policy allows → ALLOW
5. Otherwise → DENY (default deny)
```

**Path Matching**:
- Exact: `app/config/db` matches `app/config/db`
- Prefix: `app/*` matches `app/config`, `app/config/db`, etc.
- Glob: `*/config` matches `app1/config`, `app2/config`

**Performance**:
- Pre-compiled regex patterns
- Trie-based path matching
- In-memory policy cache
- Target: <1ms authorization check

### Storage

**Policy Storage**:
```
/policies/
  ├── admin.json
  ├── developer.json
  └── readonly.json
```

- Stored in BadgerDB (if persistence enabled)
- Loaded at startup, cached in memory
- Hot-reload on policy changes
- API endpoints to manage policies

**Token-Policy Mapping**:
- JWT: policies in claims
- API Key: policies in metadata
- No additional storage needed

## Alternatives Considered

### Alternative 1: Simple Role-Based Access (RBAC)
- **Pros**:
  - Simpler to implement
  - Easy to understand
  - Sufficient for many use cases
- **Cons**:
  - Not fine-grained (can't restrict specific KV paths)
  - Inflexible for complex scenarios
  - Harder to implement multi-tenancy
- **Reason for rejection**: Insufficient granularity for enterprise needs

### Alternative 2: Attribute-Based Access Control (ABAC)
- **Pros**:
  - Maximum flexibility
  - Context-aware decisions (time, IP, etc.)
  - Very expressive
- **Cons**:
  - Much more complex to implement
  - Harder to reason about
  - Performance overhead
  - Overkill for most use cases
- **Reason for rejection**: Too complex; ACL provides better balance

### Alternative 3: OPA (Open Policy Agent) Integration
- **Pros**:
  - Industry-standard policy engine
  - Rego policy language
  - Rich ecosystem
  - Auditing and compliance features
- **Cons**:
  - External dependency
  - gRPC or HTTP overhead
  - Steeper learning curve (Rego)
  - More operational complexity
- **Reason for rejection**: Prefer embedded solution; can add OPA later

### Alternative 4: AWS IAM-Style Policies
- **Pros**:
  - Very expressive
  - Familiar to AWS users
  - JSON-based
  - Conditions support
- **Cons**:
  - Overly complex for our needs
  - Verbose JSON policies
  - Harder to write correctly
  - Complex evaluation engine
- **Reason for rejection**: Too heavyweight; Consul model better fit

### Alternative 5: No ACLs (Current State)
- **Pros**:
  - Simple
  - No implementation needed
  - No performance overhead
- **Cons**:
  - All-or-nothing security
  - Cannot implement multi-tenancy
  - Not suitable for production
  - Compliance issues
- **Reason for rejection**: Security and multi-tenancy requirements

## Consequences

### Positive
- **Fine-grained security**: Control access at path/resource level
- **Multi-tenancy ready**: Isolate namespaces with ACLs
- **Audit compliance**: Log who accessed what
- **Flexible authorization**: Policies can be combined
- **Consul-compatible**: Similar to HashiCorp Consul ACLs
- **Performance**: In-memory evaluation, minimal overhead
- **Versioned policies**: Store in Git, track changes
- **Backward compatible**: Disabled by default
- **Principle of least privilege**: Deny by default

### Negative
- **Complexity increase**: More moving parts to configure
- **Migration effort**: Existing deployments need policy setup
- **Learning curve**: Teams need to understand ACL syntax
- **Testing overhead**: Need to test authorization paths
- **Performance impact**: ~0.5-1ms per request
- **Policy management**: Need UI/CLI for policy CRUD
- **Debugging**: Authorization failures harder to diagnose
- **Storage overhead**: Policies stored in BadgerDB

### Neutral
- Need to choose HCL vs JSON for policy syntax
- Policy hot-reload adds complexity
- Metrics for authorization decisions needed
- Documentation effort significant

## Implementation Notes

### Phase 1: Core ACL Engine (2-3 weeks)

**Data Structures**:
```go
type Policy struct {
    Name        string
    Description string
    KV          []KVRule
    Service     []ServiceRule
    Health      []HealthRule
    Backup      []BackupRule
    Admin       []AdminRule
}

type KVRule struct {
    Path         string   // "app/config/*"
    Capabilities []string // ["read", "list"]
}

type ServiceRule struct {
    Name         string   // "web-*"
    Capabilities []string // ["read", "write", "register"]
}
```

**ACL Middleware**:
```go
func ACLMiddleware(aclService *ACLService) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Extract token (JWT or API key)
        token := extractToken(c)

        // Get resource and action from request
        resource, action := getResourceAction(c)

        // Evaluate policies
        allowed := aclService.Evaluate(token, resource, action)

        if !allowed {
            return c.Status(403).JSON(fiber.Map{
                "error": "Forbidden",
                "message": "Insufficient permissions",
            })
        }

        return c.Next()
    }
}
```

**Policy Evaluation**:
```go
func (s *ACLService) Evaluate(token Token, resource Resource, action string) bool {
    policies := s.getPolicies(token)

    for _, policy := range policies {
        rules := policy.GetRulesForResource(resource.Type)

        for _, rule := range rules {
            if rule.Matches(resource.Path) {
                if rule.HasCapability("deny") {
                    return false // Explicit deny
                }
                if rule.HasCapability(action) {
                    return true // Explicit allow
                }
            }
        }
    }

    return false // Default deny
}
```

### Phase 2: Policy Management API (1 week)

**Endpoints**:
```
POST   /acl/policies         - Create policy
GET    /acl/policies         - List policies
GET    /acl/policies/:name   - Get policy
PUT    /acl/policies/:name   - Update policy
DELETE /acl/policies/:name   - Delete policy
POST   /acl/policies/reload  - Hot reload policies
```

### Phase 3: CLI Integration (1 week)

```bash
# Policy management
konsulctl acl policy create -file developer.hcl
konsulctl acl policy list
konsulctl acl policy get developer
konsulctl acl policy delete developer

# Token-policy binding
konsulctl acl token attach-policy --token <id> --policy developer
konsulctl acl token detach-policy --token <id> --policy developer

# Test authorization
konsulctl acl test --token <id> --resource kv --path app/config --action read
```

### Phase 4: Web UI Integration (1 week)

- Policy editor with syntax highlighting
- Policy validation
- Token-policy management
- Authorization test tool

### Configuration

```bash
# Enable ACL system
KONSUL_ACL_ENABLED=true

# Default policy (deny or allow)
KONSUL_ACL_DEFAULT_POLICY=deny

# Master token for bootstrapping
KONSUL_ACL_MASTER_TOKEN=<secret>

# Policy directory
KONSUL_ACL_POLICY_DIR=./policies
```

### Metrics

```
konsul_acl_evaluations_total{result="allow|deny"}
konsul_acl_evaluation_duration_seconds
konsul_acl_policy_load_errors_total
konsul_acl_policies_loaded
```

### Migration Path

**Step 1**: Enable ACL system (disabled by default)
**Step 2**: Create default "admin" policy with full access
**Step 3**: Attach admin policy to existing tokens
**Step 4**: Create granular policies as needed
**Step 5**: Switch to deny-by-default mode

### Security Considerations

- Master token for bootstrapping (single-use)
- Policy changes require admin capability
- Audit log all authorization decisions
- Rate limit policy evaluation requests
- Encrypt policies at rest
- Sign policies to prevent tampering

### Future Enhancements

- Namespaces (tenant isolation)
- Time-based policies (allow during business hours)
- IP-based restrictions
- Policy templates
- OPA integration option
- LDAP/AD group mapping
- Policy testing framework
- Authorization replay for debugging

## References

- [HashiCorp Consul ACL System](https://www.consul.io/docs/security/acl)
- [OPA (Open Policy Agent)](https://www.openpolicyagent.org/)
- [AWS IAM Policies](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_policies.html)
- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [ADR-0003: JWT Authentication](./0003-jwt-authentication.md) (builds upon)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-09 | Konsul Team | Initial proposal |
