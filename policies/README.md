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
      "capabilities": ["read", "write", "admin", "deny"]
    }
  ]
}
```

### Path Matching

Paths support wildcard patterns:
- `*` - Matches any characters within a single path segment
- `**` - Matches any characters including path separators
- `app/*` - Matches `app/config`, `app/db`, but not `app/config/prod`
- `app/**` - Matches `app/config`, `app/config/prod`, `app/config/prod/db`

### Capabilities

**KV Store**:
- `read` - Read key values
- `write` - Write/update key values
- `list` - List keys
- `delete` - Delete keys
- `deny` - Explicitly deny access

**Services**:
- `read` - Read service information
- `write` - Update service information
- `list` - List services
- `register` - Register new services
- `deregister` - Deregister services
- `deny` - Explicitly deny access

**Health Checks**:
- `read` - Read health check status
- `write` - Update health check status
- `deny` - Explicitly deny access

**Backups**:
- `create` - Create backups
- `restore` - Restore from backups
- `export` - Export data
- `import` - Import data
- `deny` - Explicitly deny access

**Admin**:
- `read` - Read admin information (metrics, etc.)
- `write` - Admin write operations (policy management)
- `admin` - Full admin access
- `deny` - Explicitly deny access

## Evaluation Logic

ACL policies are evaluated with the following rules:

1. **Default Deny**: If no policy allows an action, it is denied by default
2. **Explicit Deny Wins**: If any policy explicitly denies an action, it is denied
3. **First Match**: The first matching rule within a policy determines the outcome
4. **Multiple Policies**: If multiple policies are attached to a token, any one allowing the action grants access (unless explicitly denied)

## Best Practices

1. **Principle of Least Privilege**: Grant only the minimum permissions needed
2. **Use Explicit Deny**: Use `deny` capability to explicitly block access to sensitive resources
3. **Path Patterns**: Use specific path patterns instead of wildcards when possible
4. **Policy Composition**: Combine multiple focused policies rather than creating one large policy
5. **Regular Reviews**: Periodically review and update policies
6. **Testing**: Use `konsulctl acl test` to verify policies before deployment
7. **Version Control**: Store policies in Git for change tracking
8. **Documentation**: Document the purpose and use case for each policy

## Configuration

Enable ACL system in Konsul configuration:

```yaml
acl:
  enabled: true
  policy_dir: ./policies  # Directory to load policies from
  default_policy: deny    # Default action when no policy matches
```

Or via environment variables:

```bash
KONSUL_ACL_ENABLED=true
KONSUL_ACL_POLICY_DIR=./policies
KONSUL_ACL_DEFAULT_POLICY=deny
```

## Troubleshooting

### Policy Not Loading

Check the logs for policy loading errors:
```bash
grep "ACL policy" /var/log/konsul.log
```

### Permission Denied

Test the policy to understand why access was denied:
```bash
konsulctl acl test <policy-name> <resource> <path> <capability>
```

### Policy Validation Errors

Validate policy JSON:
```bash
jq empty policies/admin.json  # Check JSON syntax
```

## Additional Resources

- [Konsul ACL Documentation](../docs/acl.md)
- [ADR-0010: ACL System](../docs/adr/0010-acl-system.md)
- [API Reference](../docs/api-reference.md#acl-endpoints)
