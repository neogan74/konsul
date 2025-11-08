# Access Control List (ACL) System

The ACL system in Konsul provides fine-grained authorization for controlling access to resources. It implements a Consul-inspired ACL model with path-based rules, policy composition, and deny-by-default security.

## Table of Contents

- [Overview](#overview)
- [Concepts](#concepts)
- [Getting Started](#getting-started)
- [Policy Format](#policy-format)
- [Managing Policies](#managing-policies)
- [Authentication Integration](#authentication-integration)
- [Testing Policies](#testing-policies)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The ACL system provides:

- **Fine-grained access control** at the resource level (KV, Services, Health, Backups, Admin)
- **Path-based rules** with wildcard support
- **Policy composition** - attach multiple policies to a token
- **Deny-by-default** security model
- **Explicit deny** to block specific resources
- **REST API** for policy management
- **CLI tool** (konsulctl) for operations
- **File-based persistence** for policies

## Concepts

### Resources

Konsul defines five resource types:

1. **KV Store** (`kv`) - Key-value storage
2. **Services** (`service`) - Service registry
3. **Health Checks** (`health`) - Health monitoring
4. **Backups** (`backup`) - Backup and restore operations
5. **Admin** (`admin`) - System administration

### Capabilities

Each resource type supports specific capabilities:

| Resource | Capabilities |
|----------|-------------|
| KV | `read`, `write`, `list`, `delete`, `deny` |
| Service | `read`, `write`, `list`, `register`, `deregister`, `deny` |
| Health | `read`, `write`, `deny` |
| Backup | `create`, `restore`, `export`, `import`, `deny` |
| Admin | `read`, `write`, `admin`, `deny` |

### Policies

A **policy** is a named set of rules that define what actions are allowed on which resources. Policies are:

- Stored as JSON files
- Validated on creation/update
- Can be combined (multiple policies per token)
- Support wildcard path matching

### Evaluation

ACL evaluation follows these rules:

1. **Default Deny**: No access unless explicitly granted
2. **Explicit Deny Wins**: If any policy denies, access is denied
3. **First Match**: First matching rule in a policy applies
4. **Any Allow**: If any attached policy allows, access is granted (unless denied)

## Getting Started

### 1. Enable ACL System

Enable ACLs in your configuration:

**config.yaml**:
```yaml
acl:
  enabled: true
  policy_dir: ./policies
  default_policy: deny
```

**Environment variables**:
```bash
export KONSUL_ACL_ENABLED=true
export KONSUL_ACL_POLICY_DIR=./policies
export KONSUL_ACL_DEFAULT_POLICY=deny
```

### 2. Create Your First Policy

Create a policy file `policies/admin.json`:

```json
{
  "name": "admin",
  "description": "Full administrative access",
  "kv": [
    {
      "path": "*",
      "capabilities": ["read", "write", "list", "delete"]
    }
  ],
  "service": [
    {
      "name": "*",
      "capabilities": ["read", "write", "list", "register", "deregister"]
    }
  ],
  "health": [
    {
      "capabilities": ["read", "write"]
    }
  ],
  "backup": [
    {
      "capabilities": ["create", "restore", "export", "import"]
    }
  ],
  "admin": [
    {
      "capabilities": ["read", "write", "admin"]
    }
  ]
}
```

### 3. Load the Policy

**Via CLI**:
```bash
konsulctl acl policy create policies/admin.json
```

**Via API**:
```bash
curl -X POST http://localhost:8888/acl/policies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <admin-token>" \
  -d @policies/admin.json
```

**Via Server Startup** (place in policy_dir):
Konsul automatically loads all `*.json` files from `policy_dir` on startup.

### 4. Attach Policy to Token

When generating JWT tokens:

```go
token, err := jwtService.GenerateTokenWithPolicies(
    userID,
    username,
    []string{"admin"},      // roles
    []string{"admin"},      // policies
)
```

The resulting JWT will contain:
```json
{
  "user_id": "user123",
  "username": "alice",
  "roles": ["admin"],
  "policies": ["admin"]
}
```

## Policy Format

### Structure

```json
{
  "name": "string",          // Required: Unique policy name
  "description": "string",   // Optional: Human-readable description
  "kv": [...],              // Optional: KV store rules
  "service": [...],         // Optional: Service rules
  "health": [...],          // Optional: Health check rules
  "backup": [...],          // Optional: Backup rules
  "admin": [...]            // Optional: Admin rules
}
```

### KV Rules

```json
"kv": [
  {
    "path": "app/config/*",           // Path pattern
    "capabilities": ["read", "list"]   // Allowed actions
  },
  {
    "path": "app/secrets/**",          // Double-star for recursive
    "capabilities": ["deny"]           // Explicit deny
  }
]
```

**Path Patterns**:
- `app/config` - Exact match
- `app/*` - Match any single segment (`app/db`, `app/cache`)
- `app/**` - Match recursive (`app/config/prod/db`)
- `*` - Match all paths

### Service Rules

```json
"service": [
  {
    "name": "web-*",                  // Service name pattern
    "capabilities": ["read", "write", "register"]
  },
  {
    "name": "database",               // Exact name
    "capabilities": ["read"]
  }
]
```

**Name Patterns**:
- `web-*` - Prefix match (`web-app`, `web-api`)
- `*-prod` - Suffix match (`api-prod`, `db-prod`)
- `*` - Match all services

### Health, Backup, Admin Rules

```json
"health": [
  {
    "capabilities": ["read", "write"]
  }
],
"backup": [
  {
    "capabilities": ["create", "export"]
  }
],
"admin": [
  {
    "capabilities": ["read"]
  }
]
```

These resources have no path/name patterns - capabilities apply globally.

## Managing Policies

### List Policies

**CLI**:
```bash
konsulctl acl policy list
```

**API**:
```bash
curl http://localhost:8888/acl/policies \
  -H "Authorization: Bearer <token>"
```

**Response**:
```json
{
  "policies": ["admin", "developer", "readonly"],
  "count": 3
}
```

### Get Policy

**CLI**:
```bash
konsulctl acl policy get admin
```

**API**:
```bash
curl http://localhost:8888/acl/policies/admin \
  -H "Authorization: Bearer <token>"
```

**Response**:
```json
{
  "name": "admin",
  "description": "Full administrative access",
  "kv": [...],
  "service": [...],
  ...
}
```

### Create Policy

**CLI**:
```bash
konsulctl acl policy create policies/developer.json
```

**API**:
```bash
curl -X POST http://localhost:8888/acl/policies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d @policies/developer.json
```

### Update Policy

**CLI**:
```bash
konsulctl acl policy update policies/developer.json
```

**API**:
```bash
curl -X PUT http://localhost:8888/acl/policies/developer \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d @policies/developer.json
```

### Delete Policy

**CLI**:
```bash
konsulctl acl policy delete developer
```

**API**:
```bash
curl -X DELETE http://localhost:8888/acl/policies/developer \
  -H "Authorization: Bearer <token>"
```

## Authentication Integration

### JWT Tokens

JWT tokens carry policies in the `policies` claim:

```go
// Generate token with policies
token, err := jwtService.GenerateTokenWithPolicies(
    userID,
    username,
    []string{"developer"},              // Roles
    []string{"developer", "readonly"},  // Policies
)
```

**Token Claims**:
```json
{
  "user_id": "user123",
  "username": "alice",
  "roles": ["developer"],
  "policies": ["developer", "readonly"],
  "exp": 1234567890,
  "iat": 1234567800
}
```

### API Keys

API keys can include policies in metadata:

```go
apiKey := &APIKey{
    ID:       "key-123",
    Name:     "ci-pipeline",
    Metadata: map[string]string{
        "policies": "ci-deploy,readonly",
    },
}
```

### Middleware

ACL evaluation happens in middleware:

```go
// Static ACL check for specific resource type
app.Group("/kv").Use(
    middleware.JWTAuth(jwtService, publicPaths),
    middleware.ACLMiddleware(aclEvaluator, acl.ResourceTypeKV, acl.CapabilityWrite),
)

// Dynamic ACL check (infers resource from request)
app.Use(
    middleware.JWTAuth(jwtService, publicPaths),
    middleware.DynamicACLMiddleware(aclEvaluator),
)
```

## Testing Policies

### Test Command

Use `konsulctl acl test` to verify policy permissions:

```bash
konsulctl acl test <policies> <resource> <path> <capability>
```

**Examples**:

```bash
# Test if developer policy allows reading app/config
konsulctl acl test developer kv app/config/db read
# Output: ALLOWED ✓

# Test if developer policy allows writing to secrets
konsulctl acl test developer kv app/secrets/api-key write
# Output: DENIED ✗

# Test multiple policies
konsulctl acl test developer,readonly service web-app register
# Output: ALLOWED ✓ (developer policy allows)

# Test admin access
konsulctl acl test admin kv sensitive/data delete
# Output: ALLOWED ✓
```

### Test API Endpoint

```bash
curl -X POST http://localhost:8888/acl/test \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "policies": ["developer"],
    "resource": "kv",
    "path": "app/config/db",
    "capability": "read"
  }'
```

**Response**:
```json
{
  "allowed": true,
  "policies": ["developer"],
  "resource": "kv",
  "path": "app/config/db",
  "capability": "read"
}
```

## Best Practices

### 1. Principle of Least Privilege

Grant only the minimum permissions required:

```json
{
  "name": "ci-deploy",
  "kv": [
    {
      "path": "deploy/*",
      "capabilities": ["read", "write"]  // Only what's needed
    }
  ],
  "service": [
    {
      "name": "app-*",
      "capabilities": ["register"]  // Not deregister
    }
  ]
}
```

### 2. Use Explicit Deny

Block access to sensitive resources explicitly:

```json
{
  "name": "developer",
  "kv": [
    {
      "path": "secrets/**",
      "capabilities": ["deny"]  // Explicit block
    },
    {
      "path": "app/**",
      "capabilities": ["read", "write"]
    }
  ]
}
```

### 3. Policy Composition

Combine focused policies instead of monolithic ones:

```bash
# Good: Compose specific policies
konsulctl acl test developer,kv-reader,service-viewer ...

# Bad: One large policy with everything
konsulctl acl test god-mode ...
```

### 4. Path Specificity

Use specific paths over wildcards when possible:

```json
{
  "kv": [
    {
      "path": "app/config/database",  // Specific
      "capabilities": ["read"]
    }
  ]
}
```

Instead of:

```json
{
  "kv": [
    {
      "path": "*",  // Too broad
      "capabilities": ["read"]
    }
  ]
}
```

### 5. Version Control

Store policies in Git:

```bash
policies/
├── admin.json
├── developer.json
├── ci-deploy.json
└── readonly.json
```

Track changes, review pull requests, and roll back if needed.

### 6. Regular Audits

Periodically review policies:

```bash
# List all policies
konsulctl acl policy list

# Review each policy
for policy in $(konsulctl acl policy list); do
  konsulctl acl policy get $policy
done
```

### 7. Testing Before Deployment

Always test policies before applying:

```bash
# Test critical paths
konsulctl acl test new-policy kv production/secrets read
konsulctl acl test new-policy service critical-app deregister
```

## Troubleshooting

### Problem: Permission Denied

**Symptom**: API returns 403 Forbidden

**Debug Steps**:

1. Check token policies:
```bash
# Decode JWT to see policies
jwt decode <token>
```

2. Test policy:
```bash
konsulctl acl test <policy-name> <resource> <path> <capability>
```

3. Check logs:
```bash
grep "ACL" /var/log/konsul.log
```

### Problem: Policy Not Loading

**Symptom**: Policy doesn't exist after creating file

**Solutions**:

1. Check policy directory path:
```yaml
acl:
  policy_dir: ./policies  # Verify this path
```

2. Validate JSON syntax:
```bash
jq empty policies/policy.json
```

3. Check file permissions:
```bash
ls -la policies/
```

4. Restart Konsul:
```bash
systemctl restart konsul
```

5. Load manually:
```bash
konsulctl acl policy create policies/policy.json
```

### Problem: Explicit Deny Not Working

**Symptom**: Access allowed despite deny rule

**Explanation**: Deny rules must match the path/resource exactly. Check rule order:

```json
{
  "kv": [
    {
      "path": "app/**",
      "capabilities": ["read"]  // This matches first
    },
    {
      "path": "app/secrets/*",
      "capabilities": ["deny"]  // This never gets evaluated
    }
  ]
}
```

**Fix**: Order deny rules before allow rules or use more specific paths.

### Problem: Wildcard Not Matching

**Symptom**: Wildcard pattern doesn't match expected paths

**Solution**: Understand wildcard semantics:

- `app/*` matches `app/config` but NOT `app/config/db`
- `app/**` matches `app/config` AND `app/config/db`
- Pattern is anchored (must match from start)

Test with:
```bash
konsulctl acl test policy kv "app/config/db/prod" read
```

### Problem: Multiple Policies Conflict

**Symptom**: Unexpected behavior with multiple policies

**Evaluation Order**:
1. Check for explicit deny in any policy → DENY
2. Check for allow in any policy → ALLOW
3. No match → DENY (default)

**Debug**:
```bash
# Test with individual policies first
konsulctl acl test policy1 kv path read
konsulctl acl test policy2 kv path read

# Then test combined
konsulctl acl test policy1,policy2 kv path read
```

## Metrics

ACL system exports Prometheus metrics:

```
# Total ACL evaluations
konsul_acl_evaluations_total{resource="kv", capability="read", result="allow|deny"}

# Evaluation duration
konsul_acl_evaluation_duration_seconds{resource="kv"}

# Loaded policies
konsul_acl_policies_loaded
```

Query in Prometheus:
```promql
# Success rate
sum(rate(konsul_acl_evaluations_total{result="allow"}[5m])) /
sum(rate(konsul_acl_evaluations_total[5m]))

# P99 evaluation latency
histogram_quantile(0.99, konsul_acl_evaluation_duration_seconds_bucket)
```

## API Reference

### List Policies
```
GET /acl/policies
Authorization: Bearer <token>
```

### Get Policy
```
GET /acl/policies/:name
Authorization: Bearer <token>
```

### Create Policy
```
POST /acl/policies
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "policy-name",
  "description": "...",
  ...
}
```

### Update Policy
```
PUT /acl/policies/:name
Content-Type: application/json
Authorization: Bearer <token>

{
  "name": "policy-name",
  ...
}
```

### Delete Policy
```
DELETE /acl/policies/:name
Authorization: Bearer <token>
```

### Test Policy
```
POST /acl/test
Content-Type: application/json
Authorization: Bearer <token>

{
  "policies": ["policy1", "policy2"],
  "resource": "kv",
  "path": "app/config",
  "capability": "read"
}
```

## Further Reading

- [ADR-0010: ACL System](./adr/0010-acl-system.md) - Architectural decision record
- [Policy Examples](../policies/README.md) - Example policies for common use cases
- [API Reference](./api-reference.md) - Complete API documentation
- [Authentication Guide](./authentication.md) - JWT and API key authentication

## Support

For issues or questions:
- GitHub Issues: https://github.com/neogan74/konsul/issues
- Documentation: https://github.com/neogan74/konsul/tree/main/docs
