# ACL Policy Examples

This directory contains example ACL policy files that can be loaded by Konsul for fine-grained authorization control.

## Overview

ACL policies in Konsul define what actions users and services can perform on different resources. Policies are written in JSON format and support:

- **Resource types**: `kv`, `service`, `health`, `backup`, `admin`
- **Path/name matching**: Exact matches, single-level wildcards (`*`), and multi-level wildcards (`**`)
- **Capabilities**: Specific permissions like `read`, `write`, `delete`, `register`, `deregister`, etc.
- **Explicit deny**: Use `deny` capability to explicitly block access

## Policy Files

### 1. `admin.json` - Full Administrative Access

**Use case**: System administrators, CI/CD pipelines with full control

```json
{
  "name": "admin",
  "description": "Full administrative access to all resources",
  "kv": [{"path": "*", "capabilities": ["read", "write", "list", "delete"]}],
  "service": [{"name": "*", "capabilities": ["read", "write", "register", "deregister", "list"]}],
  "health": [{"capabilities": ["read", "write"]}],
  "backup": [{"capabilities": ["create", "restore", "export", "import", "list", "delete"]}],
  "admin": [{"capabilities": ["read", "write"]}]
}
```

**Grants**:
- Full access to all KV store keys
- Full access to all services
- Full access to health checks
- Full backup/restore capabilities
- Full admin API access (policies, metrics)

---

### 2. `developer.json` - Developer Access

**Use case**: Application developers needing limited access to configs and services

```json
{
  "name": "developer",
  "description": "Developer access with limited permissions",
  "kv": [
    {"path": "app/config/*", "capabilities": ["read", "list"]},
    {"path": "app/data/*", "capabilities": ["read", "write", "list"]},
    {"path": "app/secrets/*", "capabilities": ["deny"]}
  ],
  "service": [
    {"name": "web-*", "capabilities": ["read", "write", "register", "deregister"]},
    {"name": "api-*", "capabilities": ["read", "write", "register", "deregister"]},
    {"name": "database", "capabilities": ["read"]}
  ],
  "health": [{"capabilities": ["read"]}]
}
```

**Grants**:
- Read-only access to `app/config/*` keys
- Read/write access to `app/data/*` keys
- **Explicit deny** for `app/secrets/*` keys
- Full control over `web-*` and `api-*` services
- Read-only access to `database` service
- Read-only access to health checks

**Example usage**:
```bash
# Allowed
curl -H "Authorization: Bearer $TOKEN" http://localhost:8888/kv/app/config/database
curl -H "Authorization: Bearer $TOKEN" -X PUT -d '{"value":"test"}' http://localhost:8888/kv/app/data/cache

# Denied
curl -H "Authorization: Bearer $TOKEN" http://localhost:8888/kv/app/secrets/password  # 403
curl -H "Authorization: Bearer $TOKEN" -X PUT http://localhost:8888/kv/app/config/db  # 403 (no write)
```

---

### 3. `readonly.json` - Read-Only Access

**Use case**: Monitoring tools, auditors, read-only dashboards

```json
{
  "name": "readonly",
  "description": "Read-only access to all resources",
  "kv": [{"path": "*", "capabilities": ["read", "list"]}],
  "service": [{"name": "*", "capabilities": ["read", "list"]}],
  "health": [{"capabilities": ["read"]}]
}
```

**Grants**:
- Read-only access to all KV store keys
- Read-only access to all services
- Read-only access to health checks
- No write, delete, or admin capabilities

---

## Loading Policies

### Method 1: Load from Directory (Startup)

Set the policy directory via environment variable:

```bash
export KONSUL_ACL_ENABLED=true
export KONSUL_ACL_POLICY_DIR=./policies
./konsul
```

Konsul will automatically load all `.json` files from the specified directory at startup.

### Method 2: Create via API

```bash
curl -X POST http://localhost:8888/acl/policies \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d @policies/developer.json
```

### Method 3: Create via CLI

```bash
konsulctl acl policy create policies/developer.json
```

## Attaching Policies to Tokens

Policies are attached to JWT tokens via the `policies` claim:

```bash
# Login with policies
curl -X POST http://localhost:8888/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "username": "alice",
    "roles": ["developer"],
    "policies": ["developer", "readonly"]
  }'
```

The resulting JWT will contain:

```json
{
  "user_id": "user123",
  "username": "alice",
  "roles": ["developer"],
  "policies": ["developer", "readonly"],
  "exp": 1234567890
}
```

## Testing Policies

Use the ACL test endpoint to debug permissions:

```bash
curl -X POST http://localhost:8888/acl/test \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "policies": ["developer"],
    "resource": "kv",
    "path": "app/config/database",
    "capability": "read"
  }'

# Response
{
  "allowed": true,
  "policies": ["developer"],
  "resource": "kv",
  "path": "app/config/database",
  "capability": "read"
}
```

Or use the CLI:

```bash
konsulctl acl test developer kv app/config/database read
# Output: ✓ Allowed
```

## Creating Custom Policies

### Example: CI/CD Pipeline Policy

Create `policies/ci-deploy.json`:

```json
{
  "name": "ci-deploy",
  "description": "CI/CD pipeline deployment access",
  "kv": [
    {
      "path": "app/config/prod/*",
      "capabilities": ["read"]
    },
    {
      "path": "deployments/**",
      "capabilities": ["read", "write", "list"]
    }
  ],
  "service": [
    {
      "name": "frontend-*",
      "capabilities": ["register", "deregister", "read"]
    },
    {
      "name": "backend-*",
      "capabilities": ["register", "deregister", "read"]
    }
  ],
  "health": [
    {
      "capabilities": ["read", "write"]
    }
  ],
  "backup": [
    {
      "capabilities": ["create"]
    }
  ]
}
```

**This policy allows**:
- Read production configs
- Read/write deployment metadata
- Register/deregister frontend and backend services
- Update health checks
- Create backups (but not restore)

### Example: Monitoring Policy

Create `policies/monitoring.json`:

```json
{
  "name": "monitoring",
  "description": "Monitoring system access",
  "kv": [
    {
      "path": "*",
      "capabilities": ["read", "list"]
    }
  ],
  "service": [
    {
      "name": "*",
      "capabilities": ["read", "list"]
    }
  ],
  "health": [
    {
      "capabilities": ["read"]
    }
  ],
  "admin": [
    {
      "capabilities": ["read"]
    }
  ]
}
```

**This policy allows**:
- Read-only access to all KV keys
- Read-only access to all services
- Read-only access to health checks
- Read-only access to metrics and admin endpoints

## Path Matching Examples

### Single-level Wildcard (`*`)

```
Pattern: app/*
✓ Matches: app/config, app/data, app/cache
✗ Does NOT match: app/config/nested, other/path
```

### Multi-level Wildcard (`**`)

```
Pattern: app/**
✓ Matches: app/config, app/config/nested, app/config/nested/deep
✗ Does NOT match: other/path
```

### Exact Match

```
Pattern: app/config/database
✓ Matches: app/config/database
✗ Does NOT match: app/config/database-prod, app/config
```

### Mixed Wildcards

```
Pattern: app/*/config
✓ Matches: app/frontend/config, app/backend/config
✗ Does NOT match: app/config, app/frontend/config/nested
```

## Capability Reference

### KV Store Capabilities
- `read` - Read key values
- `write` - Create/update keys
- `list` - List keys
- `delete` - Delete keys
- `deny` - Explicitly deny all access

### Service Capabilities
- `read` - Read service information
- `write` - Update service metadata
- `list` - List services
- `register` - Register new services
- `deregister` - Remove services
- `deny` - Explicitly deny all access

### Health Capabilities
- `read` - Read health status
- `write` - Update health checks
- `deny` - Explicitly deny all access

### Backup Capabilities
- `create` - Create backups
- `restore` - Restore from backups
- `export` - Export data
- `import` - Import data
- `list` - List backups
- `delete` - Delete backups
- `deny` - Explicitly deny all access

### Admin Capabilities
- `read` - View policies, metrics, admin info
- `write` - Manage policies, system configuration
- `deny` - Explicitly deny all access

## Policy Evaluation Rules

1. **No policies attached** → **DENY** (default deny)
2. **Explicit `deny` capability** → **DENY** immediately (overrides all)
3. **Any policy allows** → **ALLOW**
4. **No matching rule** → **DENY**

### Example: Multiple Policies

If a token has both `readonly` and `developer` policies:

```
Token has: ["readonly", "developer"]

Request: Write to app/data/cache
- readonly: No write capability → No match
- developer: Has write for app/data/* → ALLOW
Result: ALLOWED (at least one policy allows)

Request: Read app/secrets/password
- readonly: Has read for * → Would allow
- developer: Has explicit deny for app/secrets/* → DENY
Result: DENIED (explicit deny overrides all)
```

## Best Practices

1. **Start with least privilege** - Grant minimum required permissions
2. **Use explicit deny** - Block sensitive paths explicitly
3. **Organize by role** - Create policies matching job functions
4. **Test before deploying** - Use `/acl/test` endpoint to verify
5. **Version control** - Store policies in Git
6. **Document policies** - Add clear descriptions
7. **Review regularly** - Audit and update policies
8. **Monitor metrics** - Track ACL denials in Prometheus

## Troubleshooting

### Access Denied Errors

**Problem**: Getting 403 Forbidden

**Solutions**:
1. Check token has policies attached: `konsulctl acl test <policy> <resource> <path> <capability>`
2. Verify policy is loaded: `konsulctl acl policy list`
3. Check for explicit deny rules
4. Review path matching (exact vs wildcard)

### Policy Not Loading

**Problem**: Policy file not loading at startup

**Solutions**:
1. Verify JSON syntax is valid
2. Check `KONSUL_ACL_POLICY_DIR` is correct
3. Ensure file has `.json` extension
4. Check file permissions (readable by Konsul)
5. Review startup logs for errors

## See Also

- [ACL Guide](../docs/acl-guide.md) - Complete ACL documentation
- [ADR-0010](../docs/adr/0010-acl-system.md) - ACL architecture decision
- [Authentication Guide](../docs/authentication.md) - JWT and API key setup
