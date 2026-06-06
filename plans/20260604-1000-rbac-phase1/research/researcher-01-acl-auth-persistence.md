# Research: ACL, Auth, Persistence Patterns in Konsul

## 1. ACL Types Definition (`internal/acl/types.go`)

### Policy Struct Fields
```go
type Policy struct {
  Name        string        // Unique policy identifier
  Description string
  KV          []KVRule      // KV store rules
  Service     []ServiceRule // Service rules
  Health      []HealthRule  // Health check rules
  Backup      []BackupRule  // Backup operation rules
  Admin       []AdminRule   // Admin operation rules
}
```

### Capability Constants
- **KV**: `read`, `write`, `list`, `delete`, `deny`
- **Service**: `register`, `deregister`
- **Backup**: `create`, `restore`, `export`, `import`
- **Admin**: `admin`

All capabilities are string type (`Capability string`).

### Resource Struct
```go
type Resource struct {
  Type ResourceType // kv, service, health, backup, admin
  Path string       // For KV: key path; For Service: service name
}
```

Resource types: `ResourceTypeKV`, `ResourceTypeService`, `ResourceTypeHealth`, `ResourceTypeBackup`, `ResourceTypeAdmin`.

### Key Implementation Details
- **Pattern Matching**: KVRule and ServiceRule compile wildcards to regex:
  - `*` = match single path segment (for KV, excludes `/`)
  - `**` = match any characters including `/`
- **Policy Validation**: `Validate()` compiles all patterns at load time
- **Rule Evaluation**: Each rule has `Matches()` and `HasCapability()` methods

---

## 2. ACL Evaluator Signature (`internal/acl/evaluator.go`)

### Evaluate Method
```go
func (e *Evaluator) Evaluate(
  policyNames []string,    // Policy names to evaluate
  resource Resource,       // Resource being accessed (type + path)
  capability Capability,   // Capability being requested
) bool                     // Returns true if allowed, false if denied
```

### Evaluator Design
- **Thread-Safe**: Uses `sync.RWMutex` for concurrent access
- **Default-Deny**: No matching rule or empty policies = deny
- **Explicit Deny First**: Checks for `CapabilityDeny` before allowing
- **Metrics Integration**: Records evaluations and duration in Prometheus
- **Type-Specific Logic**: Switch statement routes to `evaluateKV()`, `evaluateService()`, `evaluateHealth()`, `evaluateBackup()`, `evaluateAdmin()`

### Policy Management Methods
- `AddPolicy(policy *Policy) error`
- `UpdatePolicy(policy *Policy) error`
- `DeletePolicy(name string) error`
- `GetPolicy(name string) (*Policy, error)`
- `ListPolicies() []string`
- `LoadPolicies(policies []*Policy) error`

---

## 3. JWT Claims Structure (`internal/auth/jwt.go`)

### Claims Struct
```go
type Claims struct {
  UserID   string   // User identifier
  Username string
  Roles    []string
  Policies []string `json:"policies,omitempty"` // ACL policies attached to token
  jwt.RegisteredClaims // Standard JWT fields: ExpiresAt, IssuedAt, NotBefore, Issuer, Subject
}
```

### RefreshClaims Struct
```go
type RefreshClaims struct {
  UserID   string
  Username string
  Roles    []string
  Policies []string `json:"policies,omitempty"`
  jwt.RegisteredClaims
}
```

### Policy Carrying Mechanism
- **Token Generation**: `GenerateTokenWithPolicies(userID, username, roles, policies []string)` encodes policies into JWT payload
- **Token Validation**: `ValidateToken()` returns `*Claims` with Policies field intact
- **Refresh Flow**: `RefreshToken()` preserves policies from refresh token when issuing new access token
- **JSON Tags**: `"policies,omitempty"` allows omission when slice is empty

---

## 4. Auth Middleware Chain (`internal/middleware/`)

### JWT Middleware Flow
**File**: `internal/middleware/jwt.go`

1. **Public Path Check** (early exit):
   - Exact match check against `publicPathMap`
   - Prefix check against `publicPathPrefixes` (paths ending with `/*`)
   - No Authorization header required for public paths

2. **Token Extraction**:
   - Read `Authorization` header
   - Parse `Bearer <token>` format
   - Call `jwtService.ValidateToken(token)` → returns `*Claims` or error

3. **Context Population**:
   ```go
   c.Locals("user_id", claims.UserID)
   c.Locals("username", claims.Username)
   c.Locals("roles", claims.Roles)
   c.Locals("claims", claims)  // Full Claims struct
   ```

4. **Error Handling**: Specific responses for expired, missing, invalid tokens (HTTP 401)

### ACL Middleware Chain
**File**: `internal/middleware/acl.go`

1. **Claim Extraction**:
   ```go
   claims := GetClaims(c)  // From context
   if claims == nil { return Unauthorized }
   if len(claims.Policies) == 0 { return Forbidden }
   ```

2. **Resource & Capability Inference**:
   - **Static ACLMiddleware**: Caller specifies resource type & capability
   - **DynamicACLMiddleware**: `inferResourceAndCapability(c)` determines from request path and HTTP method

3. **Evaluation**:
   ```go
   allowed := evaluator.Evaluate(claims.Policies, resource, capability)
   ```

4. **Response**:
   - HTTP 401 if no claims
   - HTTP 403 with error details if unauthorized
   - Store resource & capability in context for audit logging

### Middleware Pattern
- **Composition**: Chain multiple handlers (JWT → ACL → handler)
- **Early Exit**: Public paths skip JWT validation entirely
- **Context Propagation**: Claims passed through Fiber context via `c.Locals()`
- **Helper Functions**: `GetClaims()`, `GetRoles()`, `GetUserID()`, `HasRole()` extract context data

---

## 5. BadgerDB Persistence (`internal/persistence/badger.go`)

### Engine Interface
```go
type Engine interface {
  // KV operations
  Get(key string) ([]byte, error)
  Set(key string, value []byte) error
  Delete(key string) error
  List(prefix string) ([]string, error)
  
  // Service operations
  GetService(name string) ([]byte, error)
  SetService(name string, data []byte, ttl time.Duration) error
  DeleteService(name string) error
  ListServices() ([]string, error)
  
  // Batch operations
  BatchSet(items map[string][]byte) error
  BatchDelete(keys []string) error
  
  // Management
  Close() error
  Backup(path string) error
  Restore(path string) error
  
  // Transactions
  BeginTx() (Transaction, error)
}
```

### Key Prefixes
- `"kv:"` - for KV store entries
- `"svc:"` - for service entries

Both prefixes are prepended during operations, stripped on retrieval.

### Serialization Format
- **KV Values**: Raw bytes (no assumption of format, caller encodes/decodes)
- **Service Values**: JSON (marshaled at call site, unmarshaled on retrieval)
- **TTL Handling**: Services use BadgerDB TTL via `badger.NewEntry().WithTTL()`

### BadgerDB Configuration
- **Compression**: Snappy enabled
- **Value Log**: 64MB files, 5 memtables in memory
- **WAL**: Configurable sync writes (`SyncWrites` bool)
- **GC**: Background garbage collection every 5 minutes at 50% threshold
- **Durability**: Full transaction support via `BeginTx()`

### Transaction Pattern
```go
type Transaction interface {
  Set(key string, value []byte) error
  Delete(key string) error
  Commit() error
  Rollback() error
}
```

ACID guarantees at transaction level; uses BadgerDB's native transaction API.

### Additional Methods
- `ExportData()` - returns `map[string]interface{}` with `kv` and `services` keys
- `ImportData(data map[string]interface{})` - bulk load from JSON structure

---

## 6. Configuration Pattern (`internal/config/config.go`)

### Config Struct Organization
```go
type Config struct {
  Server      ServerConfig
  Service     ServiceConfig
  Log         LogConfig
  Persistence PersistenceConfig
  Raft        RaftConfig
  Auth        AuthConfig
  ACL         ACLConfig
  // ... 8 more subsystem configs
}
```

### Environment Variable Mapping
**Pattern**: `KONSUL_<SUBSYSTEM>_<FIELD>`

Examples:
```
KONSUL_PORT=8888                    → Server.Port
KONSUL_JWT_SECRET=abc123            → Auth.JWTSecret
KONSUL_ACL_ENABLED=true             → ACL.Enabled
KONSUL_RAFT_NODE_ID=node1           → Raft.NodeID
KONSUL_PERSISTENCE_DATA_DIR=./data  → Persistence.DataDir
```

### Struct Tags
- **JSON**: Not used (config is not serialized to JSON directly)
- **Env Parsing**: Manual via `getEnvString()`, `getEnvInt()`, `getEnvBool()`, `getEnvDuration()`, `getEnvFloat()`, `getEnvUint64()`, `getEnvStringSlice()`

### Helper Functions
- `getEnvString(key, defaultValue)` - returns env var or default
- `getEnvDuration(key, defaultValue)` - parses time.ParseDuration format (e.g., `"15m"`, `"1s"`)
- `getEnvBool(key, defaultValue)` - parses strconv.ParseBool
- `splitAndTrim(s, delimiter)` - splits strings with whitespace trimming
- `parseRaftPeers(value)` - parses `"id1@host:port,id2@host:port"` format

### Validation
- **Validate()** method enforces:
  - Port ranges (1-65535)
  - Required fields for enabled subsystems
  - Valid enum values (log level, persistence type, policy direction, etc.)
  - Raft bootstrap consistency (node must be in peer list if bootstrap enabled)
  - Cross-subsystem dependencies (ACL requires Auth enabled)

### Example Auth Config Loading
```go
Auth: AuthConfig{
  Enabled:       getEnvBool("KONSUL_AUTH_ENABLED", false),
  JWTSecret:     getEnvString("KONSUL_JWT_SECRET", ""),
  JWTExpiry:     getEnvDuration("KONSUL_JWT_EXPIRY", 15*time.Minute),
  RefreshExpiry: getEnvDuration("KONSUL_REFRESH_EXPIRY", 7*24*time.Hour),
  Issuer:        getEnvString("KONSUL_JWT_ISSUER", "konsul"),
  PublicPaths:   getEnvStringSlice("KONSUL_PUBLIC_PATHS", []string{"/health", "/admin/*"}),
}
```

---

## Summary Table

| Component | Key Struct | Key Methods | Thread Safety |
|-----------|-----------|-------------|---------------|
| **ACL** | `Policy`, `Evaluator` | `Evaluate(policies, resource, capability)` | RWMutex |
| **Auth** | `Claims`, `JWTService` | `GenerateTokenWithPolicies()`, `ValidateToken()` | Stateless |
| **Middleware** | `JWTAuth`, `ACLMiddleware` | Chain handlers, extract context | Fiber context |
| **Persistence** | `BadgerEngine` | `Get/Set/List`, `BeginTx()` | BadgerDB transactions |
| **Config** | `Config` substructs | `Load()`, `Validate()` | Immutable after load |

---

## Key Integration Points

1. **Auth → Claims → ACL**: JWT middleware validates token, stores `Claims` (with `Policies`) in context; ACL middleware reads `Claims.Policies` and calls `Evaluator.Evaluate()`
2. **Config → Service Init**: Config loaded once at startup, passed to all subsystem constructors (JWTService, Evaluator, BadgerEngine)
3. **Persistence → Snapshot Recovery**: BadgerDB stores serialized snapshots; Raft FSM deserializes and applies to in-memory stores
4. **Metrics**: ACL evaluator records Prometheus metrics for monitoring authorization decisions

---

## Unresolved Questions

1. **GraphQL Authorization**: How are ACL policies enforced in GraphQL resolvers vs. REST handlers?
2. **Policy Persistence**: Are ACL policies persisted via BadgerDB or loaded from files only?
3. **Admin Token Issuance**: What is the flow for initially issuing admin tokens that bootstrap ACL-protected operations?
