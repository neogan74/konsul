# ACL Policies

This directory contains example ACL policies for Konsul. These policies demonstrate various permission models for different user roles and use cases.

## Available Policies

### 1. Admin Policy (`admin.json`)
Full administrative access to all resources.

**Use Case**: System administrators, DevOps leads

**Permissions**:
- KV Store: Full access (read, write, list, delete) to all keys
- Services: Full access to all services
- Health Checks: Read and write
- Backups: Create, restore, export, import
- Admin Operations: Full access

### 2. Developer Policy (`developer.json`)
Developer access with controlled permissions.

**Use Case**: Application developers

**Permissions**:
- KV Store: Read access to `app/config/*`, denied access to `app/secrets/*`
- Services: Full access to `web-*` services, read-only access to `database` service
- Health Checks: Read-only
- Backups: Denied
- Admin Operations: Denied

### 3. Read-Only Policy (`readonly.json`)
Read-only access to all resources, no modifications allowed.

**Use Case**: Auditors, viewers, monitoring tools

**Permissions**:
- KV Store: Read and list all keys
- Services: Read and list all services
- Health Checks: Read-only
- Backups: Denied
- Admin Operations: Denied

### 4. CI/CD Deploy Policy (`ci-deploy.json`)
Permissions for continuous integration and deployment pipelines.

**Use Case**: CI/CD systems (GitHub Actions, GitLab CI, Jenkins)

**Permissions**:
- KV Store: Full access to `deploy/*`, read access to `config/production/*`
- Services: Full access to `app-*` and `api-*` services
- Health Checks: Read and write
- Backups: Create and export
- Admin Operations: Denied

### 5. Monitoring Policy (`monitoring.json`)
Read-only access for monitoring and observability tools.

**Use Case**: Prometheus, Grafana, Datadog, monitoring systems

**Permissions**:
- KV Store: Read and list all keys
- Services: Read and list all services
- Health Checks: Read-only
- Backups: Denied
- Admin Operations: Read-only (for metrics)

## Using Policies

### Via API

Create a policy:
```bash
curl -X POST http://localhost:8888/acl/policies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d @policies/admin.json
```

List policies:
```bash
curl http://localhost:8888/acl/policies \
  -H "Authorization: Bearer <token>"
```

Get policy details:
```bash
curl http://localhost:8888/acl/policies/admin \
  -H "Authorization: Bearer <token>"
```

Delete a policy:
```bash
curl -X DELETE http://localhost:8888/acl/policies/admin \
  -H "Authorization: Bearer <token>"
```

### Via CLI (konsulctl)

Create a policy:
```bash
konsulctl acl policy create policies/admin.json
```

List policies:
```bash
konsulctl acl policy list
```

Get policy details:
```bash
konsulctl acl policy get admin
```

Update a policy:
```bash
konsulctl acl policy update policies/admin.json
```

Delete a policy:
```bash
konsulctl acl policy delete admin
```

Test ACL permissions:
```bash
# Test if developer policy allows reading app/config/db
konsulctl acl test developer kv app/config/db read

# Test if developer policy allows writing to app/secrets
konsulctl acl test developer kv app/secrets/api-key write

# Test multiple policies
konsulctl acl test developer,readonly service web-app register
```

## Attaching Policies to Tokens

### JWT Tokens

When generating JWT tokens, include policies in the claims:

```go
token, err := jwtService.GenerateTokenWithPolicies(
    userID,
    username,
    roles,
    []string{"developer", "readonly"}, // Policies
)
```

The JWT will contain:
```json
{
  "user_id": "user123",
  "username": "alice",
  "roles": ["developer"],
  "policies": ["developer", "readonly"]
}
```

### API Keys

API keys can be associated with policies through metadata:

```go
apiKey := &APIKey{
    ID:       "key-123",
    Name:     "ci-pipeline",
    Policies: []string{"ci-deploy"},
}
```

## Policy Format

Policies are defined in JSON with the following structure:

```json
{
  "name": "policy-name",
  "description": "Policy description",
  "kv": [
    {
      "path": "path/pattern/*",
      "capabilities": ["read", "write", "list", "delete", "deny"]
    }
  ],
  "service": [
    {
      "name": "service-pattern*",
      "capabilities": ["read", "write", "list", "register", "deregister", "deny"]
    }
  ],
  "health": [
    {
      "capabilities": ["read", "write", "deny"]
    }
  ],
  "backup": [
    {
      "capabilities": ["create", "restore", "export", "import", "deny"]
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
