# RBAC Phase 1: Wiring & Architecture Research

## 1. JWT Validation + ACL Enforcement Middleware

**File:** `/internal/middleware/jwt.go`

```go
// JWTAuth creates a middleware for JWT authentication
func JWTAuth(jwtService *auth.JWTService, publicPaths []string) fiber.Handler {
    // Public path exact matching or prefix matching (only if path ends with /*)
    return func(c *fiber.Ctx) error {
        // Extract "Bearer <token>" from Authorization header
        token := parts[1]
        claims, err := jwtService.ValidateToken(token)
        
        // Store in Fiber context as locals
        c.Locals("user_id", claims.UserID)
        c.Locals("username", claims.Username)
        c.Locals("roles", claims.Roles)
        c.Locals("claims", claims)  // Full Claims object
        return c.Next()
    }
}

// GetClaims retrieves JWT claims from context
func GetClaims(c *fiber.Ctx) *auth.Claims {
    if claims, ok := c.Locals("claims").(*auth.Claims); ok {
        return claims
    }
    return nil
}
```

**Claims Struct** (`/internal/auth/jwt.go`):
```go
type Claims struct {
    UserID   string   `json:"user_id"`
    Username string   `json:"username"`
    Roles    []string `json:"roles"`
    Policies []string `json:"policies,omitempty"` // ACL policies (KEY)
    jwt.RegisteredClaims
}
```

**ACL Middleware** (`/internal/middleware/acl.go`):
```go
func ACLMiddleware(evaluator *acl.Evaluator, resourceType acl.ResourceType, 
                   capability acl.Capability) fiber.Handler {
    return func(c *fiber.Ctx) error {
        claims := GetClaims(c)
        if claims == nil || len(claims.Policies) == 0 {
            return c.Status(fiber.StatusForbidden).JSON(...)
        }
        // Build resource (e.g., KV key, service name) from path params
        // Evaluate policies against (resource, capability, claims.Policies)
        return c.Next()
    }
}
```

---

## 2. ACL Handler

**File:** `/internal/handlers/acl.go`

```go
type ACLHandler struct {
    evaluator *acl.Evaluator
    policyDir string
    log       logger.Logger
}

func NewACLHandler(evaluator *acl.Evaluator, policyDir string, 
                  log logger.Logger) *ACLHandler {
    return &ACLHandler{
        evaluator: evaluator,
        policyDir: policyDir,
        log:       log,
    }
}

// CreatePolicy creates a new ACL policy
func (h *ACLHandler) CreatePolicy(c *fiber.Ctx) error {
    log := middleware.GetLogger(c)
    
    var policy acl.Policy
    if err := c.BodyParser(&policy); err != nil {
        log.Debug("Failed to parse policy request body", logger.Error(err))
        return middleware.BadRequest(c, "Invalid JSON body")
    }
    
    if err := policy.Validate(); err != nil {
        return middleware.BadRequest(c, "Invalid policy: "+err.Error())
    }
    
    if err := h.evaluator.AddPolicy(&policy); err != nil {
        if err == acl.ErrPolicyExists {
            return middleware.Conflict(c, "Policy already exists")
        }
    }
    return nil
}
```

---

## 3. Persistence Layer: BadgerDB

**File:** `/internal/persistence/interface.go`

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
    BeginTx() (Transaction, error)
}
```

**Key Serialization Pattern** (`/internal/persistence/badger.go`):
```go
const (
    kvPrefix      = "kv:"       // All KV store keys prefixed "kv:<key>"
    servicePrefix = "svc:"      // All service keys prefixed "svc:<name>"
)

// Get example:
func (b *BadgerEngine) Get(key string) ([]byte, error) {
    return b.db.View(func(txn *badger.Txn) error {
        item, err := txn.Get([]byte(kvPrefix + key))  // Prefix applied
        value, err := item.ValueCopy(nil)
        return err
    })
}
```

---

## 4. Config Struct: Auth & ACL Sections

**File:** `/internal/config/config.go`

```go
type Config struct {
    Server      ServerConfig
    Service     ServiceConfig
    Log         LogConfig
    Persistence PersistenceConfig
    Raft        RaftConfig
    Auth        AuthConfig      // <-- JWT, API key, public paths
    ACL         ACLConfig       // <-- Policies, default policy, policy dir
    // ... others
}

type AuthConfig struct {
    Enabled       bool
    JWTSecret     string
    JWTExpiry     time.Duration
    RefreshExpiry time.Duration
    Issuer        string
    APIKeyPrefix  string
    RequireAuth   bool
    PublicPaths   []string  // Routes that skip JWT auth
}

type ACLConfig struct {
    Enabled       bool
    DefaultPolicy string // "allow" or "deny"
    PolicyDir     string // Directory containing policy JSON files
}

type TLSConfig struct {
    Enabled  bool
    CertFile string
    KeyFile  string
    AutoCert bool
}
```

---

## 5. Dependency Injection Wiring (main.go)

**Pattern: Bottom-up construction with deferred cleanup**

```go
// Lines 47-70: Load config + initialize logger
cfg, err := config.Load()
appLogger := logger.NewFromConfig(cfg.Log.Level, cfg.Log.Format)

// Lines 187-208: Initialize persistence engine
var engine persistence.Engine
if cfg.Persistence.Enabled {
    engine, err = persistence.NewEngine(persistence.Config{...}, appLogger)
    defer func() { engine.Close() }()
}

// Lines 210-237: Initialize stores (KV, Service) with optional persistence
kv, _ = store.NewKVStoreWithPersistence(engine, appLogger)
svcStore, _ = store.NewServiceStoreWithPersistence(cfg.Service.TTL, engine, appLogger)
defer func() { kv.Close(); svcStore.Close() }()

// Lines 240-335: Initialize Raft clustering
raftNode, _ := konsulraft.NewNode(raftCfg, kv, svcStore)
defer func() { raftNode.Shutdown() }()

// Lines 337-354: Initialize handlers (KV, Service, Health, etc.)
// Handlers receive stores and raftNode for consistency
kvHandler := handlers.NewKVHandler(kv, raftNode)
serviceHandler := handlers.NewServiceHandler(svcStore, raftNode)

// Lines 360-372: Initialize AUTH services
jwtService := auth.NewJWTService(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, 
                                  cfg.Auth.RefreshExpiry, cfg.Auth.Issuer)
apiKeyService := auth.NewAPIKeyService(cfg.Auth.APIKeyPrefix)
authHandler := handlers.NewAuthHandler(jwtService, apiKeyService)

// Lines 374-391: Initialize ACL evaluator + load policies from disk
aclEvaluator := acl.NewEvaluator(appLogger)
aclHandler := handlers.NewACLHandler(aclEvaluator, cfg.ACL.PolicyDir, appLogger)
aclHandler.LoadPolicies()

// Lines 416-449: Register routes with middleware stack
// Auth routes (public)
app.Post("/auth/login", authHandler.Login)

// Protected routes get middleware chain
aclRoutes := app.Group("/acl")
aclRoutes.Use(middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths))
aclRoutes.Use(middleware.ACLMiddleware(aclEvaluator, acl.ResourceTypeAdmin, 
                                       acl.CapabilityWrite))
```

**Key Wiring Points:**
- JWTService created from config secret + expiry + issuer
- ACL Evaluator initialized with policies loaded from `cfg.ACL.PolicyDir`
- Handlers receive services + evaluators + raftNode (for clustering)
- Middleware stacked: JWT → ACL → Audit (if enabled)
- Claims.Policies field populated during token generation (not validated client-side)

---

## Files Summary

| File | Key Purpose |
|------|-------------|
| `/internal/middleware/jwt.go` | JWTAuth handler, claims extraction via c.Locals() |
| `/internal/middleware/acl.go` | ACLMiddleware policy enforcement |
| `/internal/handlers/acl.go` | REST endpoints for policy CRUD |
| `/internal/auth/jwt.go` | Claims struct with Policies field |
| `/internal/persistence/interface.go` | Engine interface (CRUD + tx) |
| `/internal/persistence/badger.go` | BadgerDB impl, prefix-based key serialization |
| `/internal/config/config.go` | AuthConfig + ACLConfig structs |
| `/cmd/konsul/main.go` | DI: config → services → handlers → routes |

---

## Key Observations

1. **Claims flow:** Token → JWTAuth middleware → c.Locals("claims") → GetClaims() → ACLMiddleware
2. **Policies field:** Attached during token generation, not parsed from claims later
3. **Resource type inference:** ACLMiddleware needs explicit ResourceType + Capability params OR uses DynamicACLMiddleware for method-based inference
4. **Badger serialization:** String prefixes ("kv:", "svc:") applied to all keys
5. **Config-driven:** All major services (JWT, ACL, Raft, persistence) are optional feature flags

