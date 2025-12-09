# ADR-0023: Dependency Injection with Uber FX

**Date**: 2025-12-06

**Status**: Proposed

**Deciders**: Konsul Core Team

**Tags**: architecture, dependency-injection, refactoring, maintainability

## Context

The `cmd/konsul/main.go` file has grown to over 700 lines and contains significant complexity related to:

- **Manual dependency construction**: Creating and wiring 15+ components (stores, handlers, services, managers)
- **Complex initialization order**: Dependencies must be initialized in specific sequence
- **Resource lifecycle management**: Multiple `defer` statements for cleanup scattered throughout
- **Configuration distribution**: Config values passed manually to each component
- **Testing difficulty**: Hard to test initialization logic and mock dependencies
- **Maintainability issues**: Adding new components requires careful manual wiring
- **Error-prone refactoring**: Easy to introduce bugs when changing dependency graph

Current main.go includes:
- Config loading
- Logger initialization
- Tracing setup (OpenTelemetry)
- Persistence engine
- Multiple stores (KV, Service)
- Load balancer
- 10+ handlers (KV, Service, Auth, ACL, Health, Backup, Batch, RateLimit, Watch)
- Middleware setup
- Route registration
- Background processes (cleanup, metrics)
- DNS server
- GraphQL server
- Graceful shutdown coordination

This complexity makes the codebase harder to:
- Understand for new contributors
- Test (especially integration tests)
- Refactor (risk of breaking initialization order)
- Extend (adding new features requires touching main.go)
- Debug (initialization failures are hard to trace)

Example of current manual wiring complexity:

```go
// Manual dependency construction in main.go
kv, err := store.NewKVStoreWithPersistence(engine, appLogger)
svcStore, err := store.NewServiceStoreWithPersistence(cfg.Service.TTL, engine, appLogger)
balancer := loadbalancer.New(svcStore, loadbalancer.StrategyRoundRobin)
kvHandler := handlers.NewKVHandler(kv)
serviceHandler := handlers.NewServiceHandler(svcStore)
healthHandler := handlers.NewHealthHandler(kv, svcStore, version)
// ... 15 more components
```

This pattern doesn't scale well as the application grows.

## Decision

We will adopt **Uber FX** as a dependency injection framework to:

- **Decouple initialization logic**: Move component construction to provider functions
- **Explicit dependency graphs**: FX automatically resolves dependencies
- **Lifecycle management**: Standardized OnStart/OnStop hooks for resources
- **Testing support**: Easy to inject mocks and test components in isolation
- **Modular architecture**: Group related components into FX modules
- **Graceful shutdown**: FX coordinates shutdown order automatically
- **Observability**: FX provides dependency graph visualization

### Architecture with FX Modules

```go
// cmd/konsul/main.go (simplified)
func main() {
    fx.New(
        // Configuration module
        config.Module,

        // Logging module
        logger.Module,

        // Telemetry module (tracing, metrics)
        telemetry.Module,

        // Storage module (persistence, stores)
        storage.Module,

        // Core services module (load balancer, watch manager)
        services.Module,

        // HTTP handlers module
        handlers.Module,

        // GraphQL module (if enabled)
        graphql.Module,

        // DNS module (if enabled)
        dns.Module,

        // Server module (fiber app, routes)
        server.Module,

        // Application lifecycle
        fx.Invoke(runApplication),
    ).Run()
}
```

### Module Structure Example

**Storage Module** (`internal/storage/module.go`):

```go
package storage

import (
    "go.uber.org/fx"
    "github.com/neogan74/konsul/internal/store"
    "github.com/neogan74/konsul/internal/persistence"
)

var Module = fx.Module("storage",
    fx.Provide(
        providePersistenceEngine,
        provideKVStore,
        provideServiceStore,
    ),
)

func providePersistenceEngine(
    cfg *config.Config,
    logger *logger.Logger,
    lc fx.Lifecycle,
) (persistence.Engine, error) {
    if !cfg.Persistence.Enabled {
        return nil, nil
    }

    engine, err := persistence.NewEngine(persistence.Config{
        Enabled:    cfg.Persistence.Enabled,
        Type:       cfg.Persistence.Type,
        DataDir:    cfg.Persistence.DataDir,
        BackupDir:  cfg.Persistence.BackupDir,
        SyncWrites: cfg.Persistence.SyncWrites,
        WALEnabled: cfg.Persistence.WALEnabled,
    }, logger)
    if err != nil {
        return nil, err
    }

    // FX lifecycle management
    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            return engine.Close()
        },
    })

    return engine, nil
}

func provideKVStore(
    cfg *config.Config,
    engine persistence.Engine,
    logger *logger.Logger,
    lc fx.Lifecycle,
) (*store.KVStore, error) {
    var kv *store.KVStore
    var err error

    if cfg.Persistence.Enabled {
        kv, err = store.NewKVStoreWithPersistence(engine, logger)
    } else {
        kv = store.NewKVStore()
    }

    if err != nil {
        return nil, err
    }

    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            return kv.Close()
        },
    })

    return kv, nil
}

func provideServiceStore(
    cfg *config.Config,
    engine persistence.Engine,
    logger *logger.Logger,
    lc fx.Lifecycle,
) (*store.ServiceStore, error) {
    var svcStore *store.ServiceStore
    var err error

    if cfg.Persistence.Enabled {
        svcStore, err = store.NewServiceStoreWithPersistence(cfg.Service.TTL, engine, logger)
    } else {
        svcStore = store.NewServiceStoreWithTTL(cfg.Service.TTL)
    }

    if err != nil {
        return nil, err
    }

    lc.Append(fx.Hook{
        OnStop: func(ctx context.Context) error {
            return svcStore.Close()
        },
    })

    return svcStore, nil
}
```

**Handlers Module** (`internal/handlers/module.go`):

```go
package handlers

import (
    "go.uber.org/fx"
)

var Module = fx.Module("handlers",
    fx.Provide(
        NewKVHandler,
        NewServiceHandler,
        NewLoadBalancerHandler,
        NewHealthHandler,
        NewHealthCheckHandler,
        NewBackupHandler,
        NewBatchHandler,
        NewAuthHandler,
        NewACLHandler,
        NewRateLimitHandler,
        NewKVWatchHandler,
    ),
)

// Each handler constructor becomes a provider
func NewKVHandler(kv *store.KVStore) *KVHandler {
    return &KVHandler{store: kv}
}

func NewServiceHandler(svcStore *store.ServiceStore) *ServiceHandler {
    return &ServiceHandler{store: svcStore}
}

func NewHealthHandler(
    kv *store.KVStore,
    svcStore *store.ServiceStore,
    version string, // Can inject simple values too
) *HealthHandler {
    return &HealthHandler{
        kvStore:      kv,
        serviceStore: svcStore,
        version:      version,
    }
}

// ... other handler constructors
```

**Server Module** (`internal/server/module.go`):

```go
package server

import (
    "go.uber.org/fx"
    "github.com/gofiber/fiber/v2"
)

var Module = fx.Module("server",
    fx.Provide(
        provideFiberApp,
        provideRouter,
    ),
    fx.Invoke(registerRoutes),
)

func provideFiberApp(cfg *config.Config) *fiber.App {
    return fiber.New(fiber.Config{
        // Config from cfg
    })
}

type RouteParams struct {
    fx.In

    App                  *fiber.App
    Cfg                  *config.Config
    Logger               *logger.Logger
    KVHandler            *handlers.KVHandler
    ServiceHandler       *handlers.ServiceHandler
    LoadBalancerHandler  *handlers.LoadBalancerHandler
    HealthHandler        *handlers.HealthHandler
    HealthCheckHandler   *handlers.HealthCheckHandler
    BackupHandler        *handlers.BackupHandler
    BatchHandler         *handlers.BatchHandler
    AuthHandler          *handlers.AuthHandler          `optional:"true"`
    ACLHandler           *handlers.ACLHandler           `optional:"true"`
    RateLimitHandler     *handlers.RateLimitHandler     `optional:"true"`
    KVWatchHandler       *handlers.KVWatchHandler       `optional:"true"`
    GraphQLServer        *graphql.Server                `optional:"true"`
    AuditManager         *audit.Manager
    JWTService           *auth.JWTService               `optional:"true"`
    ACLEvaluator         *acl.Evaluator                 `optional:"true"`
}

func registerRoutes(params RouteParams) error {
    app := params.App

    // Apply global middleware
    app.Use(middleware.RequestLogging(params.Logger))
    app.Use(middleware.MetricsMiddleware())

    // Register KV routes
    kvRoutes := app.Group("/kv")
    if params.ACLEvaluator != nil {
        kvRoutes.Use(middleware.DynamicACLMiddleware(params.ACLEvaluator))
    }
    kvRoutes.Get("/", params.KVHandler.List)
    kvRoutes.Get("/:key", params.KVHandler.Get)
    kvRoutes.Put("/:key", params.KVHandler.Set)
    kvRoutes.Delete("/:key", params.KVHandler.Delete)

    // ... register all other routes

    return nil
}
```

**Application Invoke** (`cmd/konsul/main.go`):

```go
func runApplication(
    lc fx.Lifecycle,
    app *fiber.App,
    cfg *config.Config,
    logger *logger.Logger,
    dnsServer *dns.Server,
    svcStore *store.ServiceStore,
) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            // Start background cleanup
            go runServiceCleanup(svcStore, cfg, logger)

            // Start HTTP server
            go func() {
                addr := cfg.Address()
                logger.Info("Starting server", logger.String("address", addr))
                if cfg.Server.TLS.Enabled {
                    if err := app.ListenTLS(addr, cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
                        logger.Error("Server error", logger.Error(err))
                    }
                } else {
                    if err := app.Listen(addr); err != nil {
                        logger.Error("Server error", logger.Error(err))
                    }
                }
            }()

            return nil
        },
        OnStop: func(ctx context.Context) error {
            logger.Info("Shutting down server...")
            return app.ShutdownWithContext(ctx)
        },
    })
}
```

### Conditional Dependencies

For optional features (GraphQL, DNS, Auth, ACL):

```go
// GraphQL module (conditional)
package graphql

var Module = fx.Module("graphql",
    fx.Provide(
        fx.Annotate(
            provideGraphQLServer,
            fx.ResultTags(`optional:"true"`),
        ),
    ),
)

func provideGraphQLServer(
    cfg *config.Config,
    deps resolver.ResolverDependencies,
) (*Server, error) {
    if !cfg.GraphQL.Enabled {
        return nil, nil // Return nil if disabled
    }

    return NewServer(deps), nil
}
```

### Dependency Graph Visualization

FX can generate dependency graphs:

```bash
# Generate dependency visualization
go run cmd/konsul/main.go --fx-visualize=dependency-graph.dot
dot -Tpng dependency-graph.dot -o dependency-graph.png
```

### Testing with FX

```go
// Integration test with FX
func TestKVHandlerIntegration(t *testing.T) {
    app := fx.New(
        fx.Supply(testConfig()),
        logger.Module,
        storage.Module,
        handlers.Module,
        fx.NopLogger, // Disable FX logs in tests
    )

    require.NoError(t, app.Start(context.Background()))
    defer app.Stop(context.Background())

    // Test handler logic
}

// Unit test with mocks
func TestKVHandlerUnit(t *testing.T) {
    mockStore := &MockKVStore{}
    handler := handlers.NewKVHandler(mockStore)

    // Test handler logic
}
```

## Alternatives Considered

### Alternative 1: Google Wire
- **Pros**:
  - Compile-time dependency injection (code generation)
  - Zero runtime overhead
  - Type-safe
  - No reflection
  - Simpler mental model
- **Cons**:
  - No built-in lifecycle management
  - Requires code generation step
  - Less flexible than runtime DI
  - No dependency graph visualization
  - Manual shutdown coordination still needed
  - Less ergonomic for complex apps
- **Reason for rejection**: Lack of lifecycle management is a major limitation for our use case with many resources requiring cleanup

### Alternative 2: Manual refactoring without DI framework
- **Pros**:
  - No external dependencies
  - Full control
  - Simpler for small apps
  - No learning curve
  - No framework overhead
- **Cons**:
  - Doesn't solve the core problem
  - Still requires manual wiring
  - Lifecycle management still ad-hoc
  - Testing still difficult
  - Doesn't scale with growth
  - High maintenance burden
- **Reason for rejection**: Doesn't address fundamental issues of manual dependency wiring

### Alternative 3: Custom DI container
- **Pros**:
  - Tailored to our exact needs
  - No external dependencies
  - Full control over behavior
  - Learning opportunity
- **Cons**:
  - Reinventing the wheel
  - Significant development effort
  - Testing and maintenance burden
  - Likely less robust than battle-tested solutions
  - Community support lacking
  - Documentation burden
- **Reason for rejection**: Not worth the effort when mature solutions exist

### Alternative 4: Service locator pattern
- **Pros**:
  - Simple to implement
  - Centralized dependency registry
  - Runtime flexibility
- **Cons**:
  - Anti-pattern (hidden dependencies)
  - Runtime errors instead of compile-time
  - Hard to test
  - Tight coupling to service locator
  - Difficult to reason about dependencies
  - No lifecycle management
- **Reason for rejection**: Known anti-pattern with significant drawbacks

### Alternative 5: Keep current approach
- **Pros**:
  - No changes needed
  - No new dependencies
  - Familiar to team
  - Works currently
- **Cons**:
  - Problem will get worse as app grows
  - Already hard to maintain
  - Testing difficulty
  - Error-prone
  - Poor developer experience
- **Reason for rejection**: Technical debt will compound over time

## Consequences

### Positive
- Dramatically reduced main.go complexity (700+ lines → ~100 lines)
- Explicit dependency graph visible in module structure
- Automatic dependency resolution and lifecycle management
- Easier to test components in isolation
- Standardized resource cleanup (no scattered defer statements)
- Better modularity and code organization
- Easier onboarding for new contributors
- Dependency graph visualization for documentation
- Reduced likelihood of initialization bugs
- Graceful shutdown coordination handled by framework
- Easier to add new features (just add providers)
- Better error messages for missing dependencies
- Support for optional dependencies (GraphQL, DNS, etc.)

### Negative
- New dependency on uber-go/fx
- Learning curve for team unfamiliar with FX
- Some runtime overhead (minimal, primarily at startup)
- Reflection-based (though minimal performance impact)
- Slightly more verbose constructor signatures
- FX-specific patterns to learn
- Dependency errors move from compile-time to startup-time
- More files/packages (but better organized)

### Neutral
- Different mental model for dependency management
- Testing approach changes (can inject mocks more easily)
- Module boundaries need to be thoughtfully designed
- Some boilerplate for module definitions
- Need to document module structure and patterns
- Migration requires touching many files (one-time cost)

## Implementation Notes

### Migration Strategy

**Phase 1: Core Infrastructure (Week 1)**
- Add FX dependency to go.mod
- Create basic module structure
- Migrate config and logger modules
- Migrate storage module (persistence, stores)
- Test that basic app starts with FX

**Phase 2: Services and Handlers (Week 2)**
- Migrate all handler constructors to providers
- Create handlers module
- Migrate service layer (load balancer, auth, ACL)
- Create services module
- Update route registration

**Phase 3: Optional Features (Week 3)**
- Migrate GraphQL module (conditional)
- Migrate DNS module (conditional)
- Migrate audit, telemetry modules
- Migrate middleware setup

**Phase 4: Cleanup and Testing (Week 4)**
- Remove old initialization code from main.go
- Add integration tests using FX
- Generate dependency graph documentation
- Update developer documentation
- Code review and refinement

### Module Organization

```
internal/
├── config/
│   └── module.go          # Config provider
├── logger/
│   └── module.go          # Logger provider
├── storage/
│   └── module.go          # Persistence, KV, ServiceStore
├── services/
│   └── module.go          # LoadBalancer, WatchManager, Auth, ACL
├── handlers/
│   └── module.go          # All HTTP handlers
├── server/
│   └── module.go          # Fiber app, routes, middleware
├── graphql/
│   └── module.go          # GraphQL server (optional)
├── dns/
│   └── module.go          # DNS server (optional)
└── telemetry/
    └── module.go          # Tracing, metrics
```

### FX Best Practices

1. **Use fx.In and fx.Out for parameter objects**:
   - Avoid long parameter lists
   - Support optional dependencies with `optional:"true"`
   - Group related dependencies

2. **Module design**:
   - One module per logical subsystem
   - Clear boundaries between modules
   - Minimal cross-module dependencies

3. **Lifecycle hooks**:
   - Use OnStart for initialization
   - Use OnStop for cleanup
   - Respect context cancellation

4. **Testing**:
   - Use fx.New in tests with test-specific modules
   - Use fx.NopLogger to silence FX logs
   - Inject mocks via fx.Supply or fx.Provide

5. **Error handling**:
   - Providers can return errors
   - FX will prevent startup if provider fails
   - Log errors clearly

### Rollback Plan

If FX introduces unforeseen issues:
1. FX changes are in separate commits
2. Can revert to manual initialization
3. No changes to business logic (only wiring)
4. Core functionality unchanged

### Performance Considerations

- FX overhead is primarily at startup (negligible)
- Runtime overhead is minimal (just function calls)
- No performance impact on request handling
- Dependency resolution happens once at startup

### Documentation Updates Needed

- Architecture documentation (module structure)
- Developer guide (how to add new components)
- Testing guide (how to test with FX)
- Dependency graph diagram
- Migration guide for contributors

## References

- [Uber FX Documentation](https://uber-go.github.io/fx/)
- [FX GitHub Repository](https://github.com/uber-go/fx)
- [FX Best Practices](https://uber-go.github.io/fx/best-practices.html)
- [Dependency Injection in Go with FX](https://blog.drewolson.org/dependency-injection-in-go)
- [Google Wire (Alternative)](https://github.com/google/wire)
- [Dependency Injection Patterns](https://martinfowler.com/articles/injection.html)
- [Go Project Layout](https://github.com/golang-standards/project-layout)

---

## Revision History

| Date | Author | Changes |
|------|--------|---------|
| 2025-12-06 | Konsul Team | Initial version |