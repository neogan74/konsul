# HTTP Handler Registration, Metrics, and Wiring Patterns

## 1. Handler Registration Pattern in cmd/konsul/main.go

**Pattern**: Direct Fiber App methods + handler groups via `.Group()`.

Handlers are instantiated directly and methods registered via `app.Post()`, `app.Get()`, `app.Put()`, `app.Delete()`. For grouped endpoints, `app.Group("/path")` creates a route group where middleware is applied to the group before registering routes.

**ACL Handler Registration Example** (lines 443-463):
```go
aclRoutes := app.Group("/acl")
if cfg.Auth.Enabled {
    aclRoutes.Use(middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths))
    aclRoutes.Use(middleware.ACLMiddleware(aclEvaluator, acl.ResourceTypeAdmin, acl.CapabilityWrite))
}
if auditManager.Enabled() {
    aclRoutes.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditManager,
        ResourceType: "acl",
        ActionMapper: middleware.ACLActionMapper,
    }))
}
aclRoutes.Post("/policies", aclHandler.CreatePolicy)
aclRoutes.Get("/policies", aclHandler.ListPolicies)
aclRoutes.Get("/policies/:name", aclHandler.GetPolicy)
aclRoutes.Put("/policies/:name", aclHandler.UpdatePolicy)
aclRoutes.Delete("/policies/:name", aclHandler.DeletePolicy)
aclRoutes.Post("/test", aclHandler.TestPolicy)
```

**Instantiation** (lines 374-390):
```go
var aclEvaluator *acl.Evaluator
var aclHandler *handlers.ACLHandler
if cfg.ACL.Enabled {
    aclEvaluator = acl.NewEvaluator(appLogger)
    aclHandler = handlers.NewACLHandler(aclEvaluator, cfg.ACL.PolicyDir, appLogger)
}
```

---

## 2. ACL Handler Structure (internal/handlers/acl.go)

**Constructor**:
```go
type ACLHandler struct {
    evaluator *acl.Evaluator
    policyDir string
    log       logger.Logger
}

func NewACLHandler(evaluator *acl.Evaluator, policyDir string, log logger.Logger) *ACLHandler {
    return &ACLHandler{
        evaluator: evaluator,
        policyDir: policyDir,
        log:       log,
    }
}
```

**Request/Response Shapes**:
- **CreatePolicy**: `POST /acl/policies` with JSON body containing `acl.Policy` struct. Response: `{message, policy}` (status 201).
- **GetPolicy**: `GET /acl/policies/:name`. Response: `acl.Policy` struct (status 200).
- **ListPolicies**: `GET /acl/policies`. Response: `{policies: [names], count}` (status 200).
- **UpdatePolicy**: `PUT /acl/policies/:name` with updated `acl.Policy`. Response: `{message, policy}` (status 200).
- **DeletePolicy**: `DELETE /acl/policies/:name`. Response: `{message, policy: name}` (status 200).
- **TestPolicy**: `POST /acl/test` with body `{policies: [], resource, path, capability}`. Response: `{allowed, policies, resource, path, capability}` (status 200).

**Error Handling**:
- Uses middleware helper functions: `middleware.BadRequest(c, msg)`, `middleware.NotFound(c, msg)`, `middleware.Conflict(c, msg)`, `middleware.InternalError(c, msg)`.
- Checks for domain errors: `acl.ErrPolicyExists`, `acl.ErrPolicyNotFound`.
- Logs via `middleware.GetLogger(c)` (extracted from fiber context locals).
- Policy validation via `policy.Validate()`.

---

## 3. Prometheus Metrics Pattern (internal/metrics/metrics.go)

**Pattern**: `promauto` auto-registration (no manual registry calls needed).

Uses `promauto.NewCounterVec()`, `promauto.NewHistogramVec()`, `promauto.NewGauge()`, etc. at package level. All metrics are automatically registered on init.

**Structure**: CounterVec, HistogramVec, Gauge with labels (dimension strings).

**Examples**:

*Counter* (total count):
```go
HTTPRequestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "konsul_http_requests_total",
        Help: "Total number of HTTP requests",
    },
    []string{"method", "path", "status"},
)
```

*Histogram* (latency/distribution):
```go
HTTPRequestDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "konsul_http_request_duration_seconds",
        Help:    "HTTP request latencies in seconds",
        Buckets: prometheus.DefBuckets,
    },
    []string{"method", "path", "status"},
)
```

*Gauge* (single value):
```go
KVStoreSize = promauto.NewGauge(
    prometheus.GaugeOpts{
        Name: "konsul_kv_store_size",
        Help: "Number of keys in the KV store",
    },
)
```

**ACL Metrics** (lines 115-145):
- `ACLEvaluationsTotal`: CounterVec with labels `[resource_type, capability, result]`.
- `ACLEvaluationDuration`: HistogramVec with custom buckets `[0.0001, 0.0005, ..., 0.1]`, labels `[resource_type]`.
- `ACLPoliciesLoaded`: Gauge (no labels).
- `ACLPolicyLoadErrors`: Counter (no labels).

Usage in main.go (line 386):
```go
metrics.ACLPoliciesLoaded.Set(float64(policyCount))
```

---

## 4. Test Pattern (internal/handlers/acl_test.go)

**Setup Function**:
```go
func setupACLHandler(policyDir string) (*ACLHandler, *fiber.App) {
    log := logger.GetDefault()
    evaluator := acl.NewEvaluator(log)
    handler := NewACLHandler(evaluator, policyDir, log)
    app := fiber.New()

    // Inject logger into fiber context
    app.Use(func(c *fiber.Ctx) error {
        c.Locals("logger", log)
        return c.Next()
    })

    // Register routes
    app.Post("/acl/policies", handler.CreatePolicy)
    // ...
    return handler, app
}
```

**Table-Driven Tests**: Not consistently used; each test is individual (e.g., `TestACLHandler_CreatePolicy`, `TestACLHandler_CreatePolicy_InvalidJSON`, `TestACLHandler_CreatePolicy_Duplicate`).

**Auth Mocking**: Bypassed via middleware injection in setupACLHandler (no JWT middleware used in tests). Logger is injected via `c.Locals("logger", log)` to satisfy middleware.GetLogger(c) calls.

**Test Structure**:
```go
func TestACLHandler_CreatePolicy(t *testing.T) {
    tmpDir := t.TempDir()
    _, app := setupACLHandler(tmpDir)

    policy := acl.Policy{...}
    body, _ := json.Marshal(policy)
    req := httptest.NewRequest(http.MethodPost, "/acl/policies", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    resp, err := app.Test(req)

    if resp.StatusCode != http.StatusCreated {
        t.Errorf("expected 201, got %d", resp.StatusCode)
    }
    // Assert response body
}
```

---

## 5. Module Path

**go.mod line 1**:
```
module github.com/neogan74/konsul
```

Module requires Go 1.24.0.

---

## Key Observations

1. **Handler Injection**: Handlers receive evaluator, logger, and optional store references in constructor. No global state.
2. **Middleware Layering**: Auth → ACL → Audit applied at group level for protection.
3. **Error Standardization**: All handlers use middleware error helpers for consistent responses.
4. **Metrics Simplicity**: promauto eliminates boilerplate; counters/histograms follow `konsul_` prefix naming.
5. **Test Isolation**: setupACLHandler creates fresh fiber app + evaluator per test; no shared state.
