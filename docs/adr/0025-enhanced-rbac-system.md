# ADR-0025: Enhanced Role-Based Access Control (RBAC)

**Date**: 2025-12-06

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: security, rbac, authorization, roles, identity, enterprise

## Context

ADR-0010 introduced a **policy-based ACL system** that provides fine-grained authorization (path-based, resource-based permissions). While this is powerful, it lacks traditional **Role-Based Access Control (RBAC)** features that enterprises expect:

### Current Limitations (ADR-0010)

**Direct Token-to-Policy Binding**:
```
JWT/API Key → [Policies] → Permissions
```

**Problems**:
1. **No role abstraction**: Users directly assigned policies (verbose)
2. **No user groups**: Cannot assign permissions to teams/departments
3. **No role hierarchy**: Cannot inherit permissions (e.g., Admin inherits Developer)
4. **Complex management**: Adding/removing users requires policy changes
5. **No LDAP/AD integration**: Cannot map external groups to permissions
6. **Difficult auditing**: "Who has what role?" requires policy inspection
7. **No temporal roles**: Cannot grant temporary elevated access
8. **Scalability issues**: Managing policies for 1000+ users is unwieldy

### Real-World Enterprise Requirements

**Scenario 1: Developer Onboarding**
```
Current (ADR-0010): Manually attach 5-10 policies to each new developer token
Desired (RBAC):     Assign "Developer" role → automatically inherits all policies
```

**Scenario 2: Team-Based Access**
```
Current: Each team member has individual policy assignments
Desired: Create "Frontend Team" group → assign "Developer" role → all members inherit
```

**Scenario 3: Temporary Elevated Access**
```
Current: Manually add admin policies, remember to remove later
Desired: Grant "On-Call Admin" role for 24 hours → auto-expires
```

**Scenario 4: LDAP/Active Directory Integration**
```
Current: Manual synchronization of users and policies
Desired: LDAP group "engineering" → auto-mapped to "Developer" role
```

**Scenario 5: Role Hierarchy**
```
Current: Admin users need all developer policies + admin policies (duplicated)
Desired: Admin role inherits Developer role → automatic permission inheritance
```

### Requirements

1. **Roles as first-class entities**: Distinct from policies
2. **User-Role assignment**: Users/tokens assigned to roles
3. **Group-Role assignment**: LDAP/AD groups mapped to roles
4. **Role-Policy mapping**: Roles contain multiple policies
5. **Role hierarchy**: Roles can inherit from other roles
6. **Temporal roles**: Time-bound role assignments (TTL)
7. **Dynamic assignment**: Runtime role changes without token refresh
8. **LDAP/AD/OIDC integration**: External group mapping
9. **Audit trail**: Role assignment history
10. **Performance**: <2ms authorization check including role resolution

## Decision

We will implement **Enhanced RBAC** that builds upon ADR-0010's policy system, adding a **Role abstraction layer**:

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Enhanced RBAC System                       │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  User/Token                                                   │
│       │                                                       │
│       ├──► User-Role Assignment (direct)                     │
│       │                                                       │
│       └──► Group Membership → Group-Role Assignment          │
│                                                               │
│  Role (first-class entity)                                   │
│       │                                                       │
│       ├──► Contains: [Policies]                              │
│       │                                                       │
│       └──► Inherits: Parent Roles (hierarchy)                │
│                                                               │
│  Policy (from ADR-0010)                                      │
│       │                                                       │
│       └──► Defines: Resource + Path + Capabilities           │
│                                                               │
│  External Identity Provider (LDAP/AD/OIDC)                   │
│       │                                                       │
│       └──► Groups → Auto-mapped to Roles                     │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### RBAC Data Model

**Role Structure**:
```go
type Role struct {
    ID          string    `json:"id"`          // "role-developer"
    Name        string    `json:"name"`        // "Developer"
    Description string    `json:"description"` // "Standard developer access"

    // Policies attached to this role
    Policies    []string  `json:"policies"`    // ["kv-app-read", "service-register"]

    // Role hierarchy (inheritance)
    InheritsFrom []string `json:"inherits_from"` // ["base-user"]

    // Metadata
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    CreatedBy   string    `json:"created_by"`

    // Optional constraints
    MaxTTL      *Duration `json:"max_ttl,omitempty"`      // Max assignment duration
    AllowedIPs  []string  `json:"allowed_ips,omitempty"`  // IP restrictions
}
```

**User-Role Assignment**:
```go
type RoleAssignment struct {
    ID        string     `json:"id"`         // "assignment-123"
    UserID    string     `json:"user_id"`    // "user-alice" or token ID
    RoleID    string     `json:"role_id"`    // "role-developer"

    // Temporal assignment
    GrantedAt time.Time  `json:"granted_at"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"` // Optional TTL

    // Audit
    GrantedBy string     `json:"granted_by"`
    Reason    string     `json:"reason,omitempty"`
}
```

**Group-Role Mapping**:
```go
type GroupRoleMapping struct {
    ID          string    `json:"id"`           // "mapping-123"
    GroupName   string    `json:"group_name"`   // "CN=Engineers,OU=Groups,DC=company,DC=com"
    GroupSource string    `json:"group_source"` // "ldap", "oidc", "saml"
    RoleID      string    `json:"role_id"`      // "role-developer"

    // Auto-sync settings
    AutoSync    bool      `json:"auto_sync"`
    LastSync    time.Time `json:"last_sync"`

    CreatedAt   time.Time `json:"created_at"`
}
```

### Role Definition Examples

**Base User Role**:
```json
{
  "id": "role-base-user",
  "name": "Base User",
  "description": "Minimal access for all authenticated users",
  "policies": [
    "health-read",
    "metrics-read"
  ],
  "inherits_from": []
}
```

**Developer Role** (inherits from Base User):
```json
{
  "id": "role-developer",
  "name": "Developer",
  "description": "Standard developer access",
  "policies": [
    "kv-app-read-write",
    "service-register",
    "service-read"
  ],
  "inherits_from": ["role-base-user"]
}
```

**Senior Developer Role** (inherits from Developer):
```json
{
  "id": "role-senior-developer",
  "name": "Senior Developer",
  "description": "Enhanced developer access with deployment capabilities",
  "policies": [
    "kv-prod-read",
    "service-deregister",
    "backup-create"
  ],
  "inherits_from": ["role-developer"]
}
```

**Admin Role** (inherits from Senior Developer):
```json
{
  "id": "role-admin",
  "name": "Administrator",
  "description": "Full system administration",
  "policies": [
    "admin-full",
    "backup-restore",
    "acl-manage"
  ],
  "inherits_from": ["role-senior-developer"]
}
```

**On-Call Admin Role** (temporary):
```json
{
  "id": "role-oncall-admin",
  "name": "On-Call Administrator",
  "description": "Temporary elevated access for on-call rotation",
  "policies": [
    "admin-emergency",
    "backup-restore",
    "service-force-deregister"
  ],
  "inherits_from": ["role-developer"],
  "max_ttl": "24h"
}
```

### Role Hierarchy and Inheritance

**Inheritance Chain**:
```
Admin
  └─ inherits Senior Developer
      └─ inherits Developer
          └─ inherits Base User

Effective Permissions (Admin):
  = Admin policies
  + Senior Developer policies
  + Developer policies
  + Base User policies
```

**Resolution Algorithm**:
```
1. Collect all roles for user (direct + group-based)
2. For each role, traverse inheritance tree (depth-first)
3. Collect all policies from all roles
4. Deduplicate policies
5. Evaluate policies using ADR-0010 engine
```

### Authorization Flow

**Enhanced Flow (with RBAC)**:
```
1. Extract token from request (JWT or API key)
2. Identify user ID from token
3. Lookup user's direct role assignments
4. Lookup user's group memberships (from LDAP/OIDC claims)
5. Resolve group-based role assignments
6. For each role:
   a. Resolve role hierarchy (collect inherited roles)
   b. Collect all policies from role chain
7. Deduplicate policies
8. Evaluate using ADR-0010 policy engine
9. Cache result for subsequent requests (with TTL)
```

**Performance Optimizations**:
- Cache role → policy mappings (invalidate on role update)
- Cache user → roles mappings (TTL: 5 minutes)
- Pre-compute role inheritance chains
- Batch LDAP group lookups

### LDAP/Active Directory Integration

**LDAP Configuration**:
```yaml
ldap:
  enabled: true
  url: "ldaps://ldap.company.com:636"
  bind_dn: "CN=konsul-service,OU=ServiceAccounts,DC=company,DC=com"
  bind_password: "${LDAP_BIND_PASSWORD}"

  # User search
  user_base_dn: "OU=Users,DC=company,DC=com"
  user_filter: "(sAMAccountName={{username}})"
  user_attr: "sAMAccountName"

  # Group search
  group_base_dn: "OU=Groups,DC=company,DC=com"
  group_filter: "(member={{user_dn}})"
  group_attr: "cn"

  # Sync settings
  sync_interval: "15m"
  cache_ttl: "5m"
```

**Group-to-Role Mapping**:
```json
{
  "mappings": [
    {
      "group": "CN=Engineering,OU=Groups,DC=company,DC=com",
      "role": "role-developer",
      "auto_sync": true
    },
    {
      "group": "CN=SRE-Team,OU=Groups,DC=company,DC=com",
      "role": "role-senior-developer",
      "auto_sync": true
    },
    {
      "group": "CN=Admins,OU=Groups,DC=company,DC=com",
      "role": "role-admin",
      "auto_sync": true
    }
  ]
}
```

**OIDC Claims-Based Mapping**:
```json
// OIDC token claims
{
  "sub": "alice@company.com",
  "email": "alice@company.com",
  "groups": ["engineering", "frontend-team"],
  "roles": ["developer"]  // Optional: direct role claims
}

// Konsul mapping configuration
{
  "oidc_group_mappings": {
    "engineering": "role-developer",
    "frontend-team": "role-frontend-developer",
    "sre": "role-senior-developer"
  }
}
```

### Temporal Role Assignments

**Grant temporary elevated access**:
```bash
# CLI: Grant on-call admin role for 24 hours
konsulctl rbac assign-role \
  --user alice \
  --role oncall-admin \
  --ttl 24h \
  --reason "On-call rotation week of 2025-12-06"

# API equivalent
POST /rbac/assignments
{
  "user_id": "alice",
  "role_id": "role-oncall-admin",
  "expires_at": "2025-12-07T10:00:00Z",
  "reason": "On-call rotation"
}
```

**Auto-expiration**:
- Background job checks expired assignments every 5 minutes
- Expired assignments automatically revoked
- User notified before expiration (optional)
- Audit log records expiration

### API Endpoints

**Role Management**:
```
POST   /rbac/roles              - Create role
GET    /rbac/roles              - List roles
GET    /rbac/roles/:id          - Get role details
PUT    /rbac/roles/:id          - Update role
DELETE /rbac/roles/:id          - Delete role
GET    /rbac/roles/:id/policies - List policies in role
POST   /rbac/roles/:id/policies - Add policy to role
DELETE /rbac/roles/:id/policies/:policy_id - Remove policy from role
```

**Role Assignment**:
```
POST   /rbac/assignments                    - Assign role to user
GET    /rbac/assignments                    - List all assignments
GET    /rbac/assignments/user/:user_id      - List user's roles
DELETE /rbac/assignments/:id                - Revoke role assignment
POST   /rbac/assignments/:id/extend         - Extend TTL
GET    /rbac/assignments/expiring           - List soon-to-expire assignments
```

**Group-Role Mapping**:
```
POST   /rbac/group-mappings       - Create group-role mapping
GET    /rbac/group-mappings       - List mappings
DELETE /rbac/group-mappings/:id   - Delete mapping
POST   /rbac/group-mappings/sync  - Trigger LDAP sync
GET    /rbac/group-mappings/status - Sync status
```

**Role Queries**:
```
GET    /rbac/users/:user_id/effective-roles      - Resolved roles (with inheritance)
GET    /rbac/users/:user_id/effective-policies   - All policies from roles
GET    /rbac/users/:user_id/permissions          - Effective permissions
GET    /rbac/roles/:role_id/members              - Users with this role
GET    /rbac/roles/:role_id/inheritance-chain    - Role hierarchy tree
```

## Alternatives Considered

### Alternative 1: Keep Policy-Only System (ADR-0010)
- **Pros**:
  - Simpler (no role abstraction)
  - Direct policy assignment
  - Less storage overhead
- **Cons**:
  - Doesn't scale for large organizations
  - No group-based management
  - No role hierarchy
  - Poor user experience
- **Reason for rejection**: Insufficient for enterprise requirements

### Alternative 2: Casbin RBAC Library
- **Pros**:
  - Mature RBAC library
  - Well-tested
  - Supports multiple models (RBAC, ABAC)
  - Active community
- **Cons**:
  - External dependency
  - Learning curve (Casbin DSL)
  - May not integrate well with ADR-0010 policies
  - Less control over implementation
- **Reason for rejection**: Prefer embedded solution tailored to Konsul

### Alternative 3: Keycloak Integration
- **Pros**:
  - Full-featured identity management
  - RBAC built-in
  - LDAP/AD/OIDC support
  - Web UI for management
- **Cons**:
  - External service dependency
  - Operational complexity (another service to run)
  - Heavyweight for Konsul's needs
  - Network latency for authorization
- **Reason for rejection**: Too heavyweight; prefer embedded RBAC

### Alternative 4: Attribute-Based Access Control (ABAC)
- **Pros**:
  - Maximum flexibility
  - Context-aware (time, IP, resource attributes)
  - Very expressive
- **Cons**:
  - Much more complex than RBAC
  - Harder to understand and debug
  - Performance overhead
  - Overkill for most use cases
- **Reason for rejection**: RBAC + policies provide sufficient expressiveness

### Alternative 5: AWS IAM-Style (Users, Groups, Policies)
- **Pros**:
  - Well-known model
  - Groups + policies
  - Familiar to AWS users
- **Cons**:
  - No role hierarchy
  - Complex policy language
  - Groups are not roles (different abstraction)
  - Missing temporal assignments
- **Reason for rejection**: Roles provide better abstraction for our use case

## Consequences

### Positive
- **Simplified management**: Assign roles instead of individual policies
- **Role hierarchy**: Inherit permissions naturally
- **Group-based access**: LDAP/AD/OIDC integration
- **Temporal roles**: Time-bound elevated access
- **Better auditing**: "Who has what role?" is clear
- **Scalability**: Manage 1000+ users efficiently
- **Flexibility**: Combine roles and policies as needed
- **Industry standard**: RBAC is well-understood pattern
- **Backward compatible**: Builds on ADR-0010 policies
- **Multi-tenancy ready**: Roles can be scoped to namespaces

### Negative
- **Complexity increase**: Additional abstraction layer
- **Storage overhead**: Role definitions, assignments, mappings
- **Performance impact**: +1-2ms for role resolution
- **Migration effort**: Convert existing policy assignments to roles
- **LDAP dependency**: Requires LDAP/AD for group-based access
- **Cache invalidation**: Role changes require cache flushes
- **Learning curve**: Teams need to understand roles vs policies
- **Testing overhead**: More authorization scenarios to test

### Neutral
- Role-policy mapping needs UI/CLI tooling
- Metrics for role usage and assignments
- Documentation for role design best practices
- Background job for expired assignment cleanup
- LDAP sync monitoring and alerting

## Implementation Notes

### Phase 1: Core RBAC Engine (3-4 weeks)

**Data Structures**:
```go
package rbac

type RoleManager struct {
    roleStore       RoleStore
    assignmentStore AssignmentStore
    mappingStore    MappingStore
    policyEngine    *acl.Evaluator // From ADR-0010
    cache           *RoleCache
    logger          *logger.Logger
}

func (rm *RoleManager) GetEffectiveRoles(userID string) ([]Role, error) {
    // 1. Get direct role assignments
    directRoles := rm.assignmentStore.GetByUser(userID)

    // 2. Get group memberships (from LDAP/OIDC)
    groups := rm.getUserGroups(userID)

    // 3. Resolve group-role mappings
    groupRoles := rm.mappingStore.GetRolesByGroups(groups)

    // 4. Combine and deduplicate
    allRoles := append(directRoles, groupRoles...)

    // 5. Resolve inheritance hierarchy
    return rm.resolveInheritance(allRoles), nil
}

func (rm *RoleManager) GetEffectivePolicies(userID string) ([]Policy, error) {
    roles := rm.GetEffectiveRoles(userID)

    policies := []Policy{}
    for _, role := range roles {
        rolePolicies := rm.roleStore.GetPolicies(role.ID)
        policies = append(policies, rolePolicies...)
    }

    return deduplicatePolicies(policies), nil
}

func (rm *RoleManager) Authorize(userID string, resource Resource, action string) bool {
    // Get effective policies
    policies := rm.GetEffectivePolicies(userID)

    // Use ADR-0010 policy evaluator
    return rm.policyEngine.EvaluatePolicies(policies, resource, action)
}
```

**Role Inheritance Resolution**:
```go
func (rm *RoleManager) resolveInheritance(roles []Role) []Role {
    visited := make(map[string]bool)
    result := []Role{}

    var traverse func(role Role)
    traverse = func(role Role) {
        if visited[role.ID] {
            return // Prevent cycles
        }
        visited[role.ID] = true
        result = append(result, role)

        // Traverse parent roles
        for _, parentID := range role.InheritsFrom {
            parent := rm.roleStore.Get(parentID)
            traverse(parent)
        }
    }

    for _, role := range roles {
        traverse(role)
    }

    return result
}
```

**Middleware Integration**:
```go
func RBACMiddleware(roleManager *rbac.RoleManager) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Extract user ID from JWT/API key
        userID := extractUserID(c)

        // Get resource and action
        resource, action := getResourceAction(c)

        // Authorize using RBAC
        allowed := roleManager.Authorize(userID, resource, action)

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

### Phase 2: LDAP/AD Integration (2-3 weeks)

**LDAP Client**:
```go
package ldap

type LDAPClient struct {
    config LDAPConfig
    conn   *ldap.Conn
}

func (lc *LDAPClient) GetUserGroups(username string) ([]string, error) {
    // 1. Search for user DN
    userDN := lc.searchUserDN(username)

    // 2. Search for groups where user is member
    searchRequest := ldap.NewSearchRequest(
        lc.config.GroupBaseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        0, 0, false,
        fmt.Sprintf("(member=%s)", userDN),
        []string{lc.config.GroupAttr},
        nil,
    )

    result, err := lc.conn.Search(searchRequest)
    if err != nil {
        return nil, err
    }

    groups := []string{}
    for _, entry := range result.Entries {
        group := entry.GetAttributeValue(lc.config.GroupAttr)
        groups = append(groups, group)
    }

    return groups, nil
}
```

**Auto-Sync Background Job**:
```go
func (rm *RoleManager) StartGroupSyncJob(interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        rm.logger.Info("Starting LDAP group sync")

        mappings := rm.mappingStore.GetAutoSyncMappings()
        for _, mapping := range mappings {
            if err := rm.syncGroupMapping(mapping); err != nil {
                rm.logger.Error("Failed to sync group",
                    logger.String("group", mapping.GroupName),
                    logger.Error(err))
                metrics.LDAPSyncErrors.Inc()
            }
        }

        metrics.LDAPSyncSuccess.Inc()
        rm.logger.Info("LDAP group sync completed")
    }
}
```

### Phase 3: Temporal Assignments (1 week)

**Expiration Background Job**:
```go
func (rm *RoleManager) StartExpirationJob(checkInterval time.Duration) {
    ticker := time.NewTicker(checkInterval)
    defer ticker.Stop()

    for range ticker.C {
        now := time.Now()
        expired := rm.assignmentStore.GetExpired(now)

        for _, assignment := range expired {
            rm.logger.Info("Revoking expired role assignment",
                logger.String("user", assignment.UserID),
                logger.String("role", assignment.RoleID))

            rm.assignmentStore.Delete(assignment.ID)

            // Audit log
            rm.auditLog("role.assignment.expired", assignment)

            // Optional: Notify user
            rm.notifyUser(assignment.UserID, "Your role assignment has expired")

            metrics.RoleAssignmentsExpired.Inc()
        }

        // Invalidate caches
        rm.cache.Invalidate()
    }
}
```

### Phase 4: CLI Integration (1 week)

```bash
# Role management
konsulctl rbac role create \
  --name "Developer" \
  --description "Standard developer access" \
  --policies kv-app-rw,service-register \
  --inherits-from base-user

konsulctl rbac role list
konsulctl rbac role get developer
konsulctl rbac role delete developer

# Role hierarchy
konsulctl rbac role add-parent --role senior-dev --parent developer
konsulctl rbac role remove-parent --role senior-dev --parent developer
konsulctl rbac role inheritance-tree admin

# Role assignment
konsulctl rbac assign --user alice --role developer
konsulctl rbac assign --user bob --role oncall-admin --ttl 24h --reason "On-call rotation"
konsulctl rbac revoke --user alice --role developer
konsulctl rbac list-assignments --user alice

# Group mapping
konsulctl rbac map-group \
  --group "CN=Engineering,OU=Groups,DC=company,DC=com" \
  --role developer \
  --auto-sync

konsulctl rbac list-mappings
konsulctl rbac sync-ldap
konsulctl rbac sync-status

# Queries
konsulctl rbac effective-roles --user alice
konsulctl rbac effective-policies --user alice
konsulctl rbac effective-permissions --user alice
konsulctl rbac role-members --role developer
```

### Phase 5: Web UI (2 weeks)

**UI Components**:
- Role editor (create/edit/delete roles)
- Role hierarchy visualizer (tree view)
- User-role assignment interface
- Group-role mapping configuration
- Temporal assignment scheduler
- Effective permissions viewer
- Audit log viewer

### Configuration

```yaml
# config.yaml
rbac:
  enabled: true

  # Cache settings
  cache_ttl: "5m"
  cache_size: 10000

  # Temporal assignments
  expiration_check_interval: "5m"
  expiration_notification_hours: 24  # Notify 24h before expiry

  # LDAP integration
  ldap:
    enabled: true
    url: "ldaps://ldap.company.com:636"
    bind_dn: "${LDAP_BIND_DN}"
    bind_password: "${LDAP_BIND_PASSWORD}"
    user_base_dn: "OU=Users,DC=company,DC=com"
    group_base_dn: "OU=Groups,DC=company,DC=com"
    sync_interval: "15m"
    cache_ttl: "5m"

  # OIDC integration
  oidc:
    enabled: false
    groups_claim: "groups"  # JWT claim containing groups
```

### Storage Schema

**BadgerDB Keys**:
```
# Roles
/rbac/roles/{role_id}                    → Role JSON

# Assignments
/rbac/assignments/{assignment_id}        → Assignment JSON
/rbac/assignments/user/{user_id}         → List of assignment IDs
/rbac/assignments/role/{role_id}         → List of assignment IDs

# Group mappings
/rbac/mappings/{mapping_id}              → Mapping JSON
/rbac/mappings/group/{group_name}        → Mapping ID
/rbac/mappings/role/{role_id}            → List of mapping IDs

# Cache (optional, in-memory)
/rbac/cache/user-roles/{user_id}         → Cached roles (TTL: 5m)
/rbac/cache/user-groups/{user_id}        → Cached LDAP groups (TTL: 5m)
```

### Metrics

```
# Role metrics
konsul_rbac_roles_total
konsul_rbac_role_assignments_total
konsul_rbac_group_mappings_total

# Authorization metrics
konsul_rbac_authorization_duration_seconds
konsul_rbac_role_resolution_duration_seconds
konsul_rbac_cache_hit_ratio

# Temporal assignments
konsul_rbac_assignments_expiring_soon
konsul_rbac_assignments_expired_total

# LDAP metrics
konsul_rbac_ldap_sync_duration_seconds
konsul_rbac_ldap_sync_errors_total
konsul_rbac_ldap_groups_synced_total
```

### Migration Path

**Step 1: Enable RBAC** (alongside existing ACL)
```bash
KONSUL_RBAC_ENABLED=true
```

**Step 2: Create default roles**
```bash
konsulctl rbac role create --name admin --policies admin-full
konsulctl rbac role create --name developer --policies kv-rw,service-rw
konsulctl rbac role create --name readonly --policies kv-ro,service-ro
```

**Step 3: Migrate existing users**
```bash
# Assign roles to existing users based on their current policies
konsulctl rbac migrate-from-acl
```

**Step 4: Configure LDAP/OIDC**
```bash
# Map LDAP groups to roles
konsulctl rbac map-group --group "CN=Engineering" --role developer
```

**Step 5: Verify and cutover**
```bash
# Test effective permissions
konsulctl rbac effective-permissions --user alice

# Enable RBAC enforcement
KONSUL_RBAC_ENFORCE=true
```

### Security Considerations

- **Role assignment auditing**: Log all role grants/revokes
- **Least privilege**: Default to minimal roles
- **Role explosion prevention**: Limit role hierarchy depth (max 5 levels)
- **Circular dependency detection**: Prevent role inheritance cycles
- **Temporal role constraints**: Enforce max TTL per role
- **Group sync security**: Use read-only LDAP service account
- **Cache poisoning**: Validate cached data integrity
- **Role deletion safety**: Prevent deletion of assigned roles

### Testing Strategy

**Unit Tests**:
- Role inheritance resolution
- Effective policy calculation
- Cache invalidation
- Expiration logic

**Integration Tests**:
- LDAP group sync
- OIDC claims mapping
- End-to-end authorization
- Role hierarchy scenarios

**Performance Tests**:
- Authorization latency (target: <2ms)
- Cache hit rates
- Concurrent role resolution
- Large role hierarchy (1000+ roles)

### Performance Targets

- **Authorization latency**: <2ms (p99) with role resolution
- **Role resolution**: <1ms (cached), <5ms (uncached)
- **LDAP group lookup**: <100ms
- **Cache hit rate**: >95% for active users
- **Max roles per user**: 50
- **Max inheritance depth**: 5 levels

## References

- [ADR-0010: ACL System](./0010-acl-system.md) (foundation)
- [ADR-0003: JWT Authentication](./0003-jwt-authentication.md)
- [NIST RBAC Model](https://csrc.nist.gov/projects/role-based-access-control)
- [Kubernetes RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [HashiCorp Vault RBAC](https://www.vaultproject.io/docs/concepts/policies)
- [Casbin RBAC](https://casbin.org/docs/rbac)
- [LDAP Group Membership](https://ldap.com/dit-and-the-ldap-root-dse/)
- [OIDC Claims](https://openid.net/specs/openid-connect-core-1_0.html)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-06 | Konsul Team | Initial proposal |