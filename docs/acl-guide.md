# ACL (Access Control List) System Guide

Konsul's ACL system provides fine-grained authorization control for KV store access and other resources. This guide explains how to configure and use ACLs.

## Overview

The ACL system in Konsul is inspired by HashiCorp Consul's ACL design and provides:

- **Fine-grained access control** at the resource and path level
- **Policy-based authorization** with reusable policies
- **Wildcard path matching** for flexible rules
- **Default deny** security model
- **Multiple policy support** per token
- **Metrics and logging** for authorization decisions

## Configuration

Enable ACLs through environment variables:

```bash
# Enable ACL system
KONSUL_ACL_ENABLED=true

# Default policy (deny or allow) - recommend "deny"
KONSUL_ACL_DEFAULT_POLICY=deny

# Directory containing policy JSON files
KONSUL_ACL_POLICY_DIR=./policies

# Authentication must be enabled for ACLs
KONSUL_AUTH_ENABLED=true
KONSUL_JWT_SECRET=your-secret-key
```

## Policy Structure

Policies are defined in JSON format:

```json
{
  "name": "policy-name",
  "description": "Policy description",
  "kv": [
    {
      "path": "app/config/*",
      "capabilities": ["read", "list"]
    }
  ],
  "service": [
    {
      "name": "web-*",
      "capabilities": ["read", "register"]
    }
  ],
  "health": [
    {
      "capabilities": ["read"]
    }
  ],
  "backup": [
    {
      "capabilities": ["create", "restore"]
    }
  ],
  "admin": [
    {
      "capabilities": ["read", "write"]
    }
  ]
}
```

## Resource Types

### KV Store (`kv`)

Controls access to the key-value store.

**Capabilities:**
- `read` - Read keys
- `write` - Create/update keys
- `list` - List keys
- `delete` - Delete keys
- `deny` - Explicitly deny access

**Path Matching:**
- `app/config` - Exact match
- `app/*` - Single-level wildcard (matches `app/config` but not `app/config/nested`)
- `app/**` - Multi-level wildcard (matches `app/config/nested/deep`)

**Example:**
```json
{
  "kv": [
    {
      "path": "app/config/*",
      "capabilities": ["read", "list"]
    },
    {
      "path": "app/secrets/*",
      "capabilities": ["deny"]
    }
  ]
}
```

### Service (`service`)

Controls access to service registration and discovery.

**Capabilities:**
- `read` - Read service information
- `write` - Update service metadata
- `list` - List services
- `register` - Register new services
- `deregister` - Remove services
- `deny` - Explicitly deny access

**Name Matching:**
- `web` - Exact match
- `web-*` - Wildcard match (matches `web-frontend`, `web-api`, etc.)
- `*` - Match all services

**Example:**
```json
{
  "service": [
    {
      "name": "web-*",
      "capabilities": ["read", "register", "deregister"]
    },
    {
      "name": "database",
      "capabilities": ["read"]
    }
  ]
}
```

### Health (`health`)

Controls access to health check endpoints.

**Capabilities:**
- `read` - Read health status
- `write` - Update health checks
- `deny` - Explicitly deny access

### Backup (`backup`)

Controls access to backup operations.

**Capabilities:**
- `create` - Create backups
- `restore` - Restore from backups
- `export` - Export data
- `import` - Import data
- `list` - List backups
- `delete` - Delete backups
- `deny` - Explicitly deny access

### Admin (`admin`)

Controls access to ACL management and metrics.

**Capabilities:**
- `read` - View ACL policies and metrics
- `write` - Manage ACL policies
- `deny` - Explicitly deny access

## Built-in Policies

Konsul includes three example policies:

### 1. Admin Policy (`admin.json`)

Full access to all resources:

```json
{
  "name": "admin",
  "description": "Full administrative access",
  "kv": [{"path": "*", "capabilities": ["read", "write", "list", "delete"]}],
  "service": [{"name": "*", "capabilities": ["read", "write", "register", "deregister", "list"]}],
  "health": [{"capabilities": ["read", "write"]}],
  "backup": [{"capabilities": ["create", "restore", "export", "import", "list", "delete"]}],
  "admin": [{"capabilities": ["read", "write"]}]
}
```

### 2. Developer Policy (`developer.json`)

Limited access for developers:

- Read config, read/write data
- Deny secrets access
- Manage web/api services
- Read-only for database service
- Read health status

### 3. Read-Only Policy (`readonly.json`)

Read-only access to all resources.

## Attaching Policies to Tokens

Policies are attached to JWT tokens via the `policies` claim:

```go
token, err := jwtService.GenerateTokenWithPolicies(
    userID,
    username,
    []string{"developer"}, // roles
    []string{"developer", "readonly"}, // policies
)
```

Example JWT payload:

```json
{
  "user_id": "user123",
  "username": "alice",
  "roles": ["developer"],
  "policies": ["developer", "readonly"],
  "exp": 1234567890
}
```

## API Endpoints

### Manage Policies

**Create Policy:**
```http
POST /acl/policies
Authorization: Bearer <token-with-admin-write>
Content-Type: application/json

{
  "name": "custom-policy",
  "description": "Custom policy",
  "kv": [...]
}
```

**List Policies:**
```http
GET /acl/policies
Authorization: Bearer <token-with-admin-read>
```

**Get Policy:**
```http
GET /acl/policies/developer
Authorization: Bearer <token-with-admin-read>
```

**Update Policy:**
```http
PUT /acl/policies/developer
Authorization: Bearer <token-with-admin-write>
Content-Type: application/json

{
  "name": "developer",
  ...
}
```

**Delete Policy:**
```http
DELETE /acl/policies/developer
Authorization: Bearer <token-with-admin-write>
```

**Test Policy:**
```http
POST /acl/test
Authorization: Bearer <token>
Content-Type: application/json

{
  "policies": ["developer"],
  "resource": "kv",
  "path": "app/config/database",
  "capability": "read"
}

Response:
{
  "allowed": true,
  "policies": ["developer"],
  "resource": "kv",
  "path": "app/config/database",
  "capability": "read"
}
```

## Evaluation Algorithm

1. **Check token**: Extract policies from JWT claims
2. **No policies**: Deny access (default deny)
3. **For each policy**:
   - Match resource type (kv, service, etc.)
   - Match path/name pattern
   - Check for explicit `deny` capability → **DENY** immediately
   - Check if requested capability is allowed → Mark as allowed
4. **If any policy allows**: **ALLOW**
5. **Otherwise**: **DENY**

## Path Matching Examples

### Single-level Wildcard (`*`)

```
Pattern: app/*
Matches: app/config, app/data
Does NOT match: app/config/nested
```

### Multi-level Wildcard (`**`)

```
Pattern: app/**
Matches: app/config, app/config/nested, app/config/nested/deep
```

### Exact Match

```
Pattern: app/config/database
Matches: app/config/database
Does NOT match: app/config/database-prod
```

## Security Best Practices

1. **Use deny-by-default**: Set `KONSUL_ACL_DEFAULT_POLICY=deny`
2. **Principle of least privilege**: Grant minimum required permissions
3. **Explicit deny rules**: Use `deny` capability for sensitive paths
4. **Separate policies by role**: Don't give everyone admin access
5. **Review policies regularly**: Audit and update policies
6. **Monitor metrics**: Track ACL evaluation metrics in Prometheus
7. **Store policies in Git**: Version control your policies

## Metrics

Konsul exposes ACL metrics for monitoring:

```
# Total evaluations
konsul_acl_evaluations_total{resource_type="kv", capability="read", result="allow"}

# Evaluation latency
konsul_acl_evaluation_duration_seconds{resource_type="kv"}

# Loaded policies
konsul_acl_policies_loaded

# Load errors
konsul_acl_policy_load_errors_total
```

## Troubleshooting

### Access Denied

**Problem**: Getting 403 Forbidden errors

**Check**:
1. Is ACL enabled? (`KONSUL_ACL_ENABLED=true`)
2. Does the token have policies attached?
3. Do the policies allow the requested capability?
4. Use `/acl/test` endpoint to debug

### Policy Not Loading

**Problem**: Policy file not loading at startup

**Check**:
1. Verify `KONSUL_ACL_POLICY_DIR` is correct
2. Ensure JSON syntax is valid
3. Check file permissions (readable by Konsul)
4. Review startup logs for errors

### Performance Issues

**Problem**: Slow ACL evaluations

**Check**:
1. Monitor `konsul_acl_evaluation_duration_seconds` metric
2. Simplify policy rules if needed
3. Ensure policies are compiled (automatic)

## Example Workflow

1. **Enable ACLs**:
```bash
export KONSUL_ACL_ENABLED=true
export KONSUL_AUTH_ENABLED=true
export KONSUL_JWT_SECRET=my-secret
```

2. **Create policy** (`policies/developer.json`):
```json
{
  "name": "developer",
  "kv": [{"path": "app/*", "capabilities": ["read", "write"]}]
}
```

3. **Start Konsul**:
```bash
./konsul
# Logs: ACL system initialized, policies=1
```

4. **Generate token with policy**:
```go
token, _ := jwtService.GenerateTokenWithPolicies(
    "user1", "alice", []string{"dev"}, []string{"developer"},
)
```

5. **Use token**:
```bash
curl -H "Authorization: Bearer $TOKEN" \
     -X PUT \
     -d '{"value":"test"}' \
     http://localhost:8888/kv/app/config
# Success: ACL allows write to app/*
```

## Migration from No ACLs

1. **Enable authentication first** (without ACLs):
```bash
KONSUL_AUTH_ENABLED=true
KONSUL_REQUIRE_AUTH=false  # Don't require yet
```

2. **Create admin policy** with full access

3. **Enable ACLs** but don't require auth yet:
```bash
KONSUL_ACL_ENABLED=true
KONSUL_REQUIRE_AUTH=false
```

4. **Update tokens** to include policies

5. **Enable required auth**:
```bash
KONSUL_REQUIRE_AUTH=true
```

## See Also

- [ADR-0010: ACL System](adr/0010-acl-system.md)
- [Authentication API](authentication-api.md)
- [JWT Authentication](authentication.md)
