# Audit Logging Integration Example

This example demonstrates how to integrate audit logging into Konsul's handlers.

## Complete Integration Example

### 1. Initialize Audit Manager in main.go

```go
package main

import (
    "github.com/neogan74/konsul/internal/audit"
    "github.com/neogan74/konsul/internal/config"
    "github.com/neogan74/konsul/internal/middleware"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Initialize audit manager
    auditManager, err := audit.NewManager(audit.Config{
        Enabled:       cfg.Audit.Enabled,
        Sink:          cfg.Audit.Sink,
        FilePath:      cfg.Audit.FilePath,
        BufferSize:    cfg.Audit.BufferSize,
        FlushInterval: cfg.Audit.FlushInterval,
        DropPolicy:    audit.DropPolicy(cfg.Audit.DropPolicy),
    }, appLogger)
    if err != nil {
        log.Fatalf("Failed to initialize audit logging: %v", err)
    }

    // Ensure graceful shutdown
    defer func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := auditManager.Shutdown(ctx); err != nil {
            appLogger.Error("Failed to shutdown audit manager", logger.Error(err))
        }
    }()

    // Log startup info
    if auditManager.Enabled() {
        appLogger.Info("Audit logging enabled",
            logger.String("sink", cfg.Audit.Sink),
            logger.Int("buffer_size", cfg.Audit.BufferSize))
    }

    // Setup routes with audit middleware
    setupRoutes(app, auditManager)
}
```

### 2. Apply Middleware to Route Groups

```go
func setupRoutes(app *fiber.App, auditMgr *audit.Manager) {
    api := app.Group("/api/v1")

    // KV routes with audit logging
    kv := api.Group("/kv")
    kv.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditMgr,
        ResourceType: "kv",
        ActionMapper: middleware.KVActionMapper,
    }))
    kv.Put("/:key", handlers.HandleKVSet)
    kv.Get("/:key", handlers.HandleKVGet)
    kv.Delete("/:key", handlers.HandleKVDelete)

    // Service routes with audit logging
    service := api.Group("/service")
    service.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditMgr,
        ResourceType: "service",
        ActionMapper: middleware.ServiceActionMapper,
    }))
    service.Post("/", handlers.HandleServiceRegister)
    service.Delete("/:name", handlers.HandleServiceDeregister)
    service.Put("/:name/heartbeat", handlers.HandleServiceHeartbeat)

    // ACL routes with audit logging
    acl := api.Group("/acl")
    acl.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditMgr,
        ResourceType: "acl",
        ActionMapper: middleware.ACLActionMapper,
    }))
    acl.Post("/token", handlers.HandleCreateToken)
    acl.Delete("/token/:id", handlers.HandleRevokeToken)
    acl.Post("/policy", handlers.HandleCreatePolicy)
    acl.Put("/policy/:name", handlers.HandleUpdatePolicy)

    // Backup routes with audit logging
    backup := api.Group("/backup")
    backup.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditMgr,
        ResourceType: "backup",
        ActionMapper: middleware.BackupActionMapper,
    }))
    backup.Post("/", handlers.HandleBackupCreate)
    backup.Put("/:id", handlers.HandleBackupRestore)
    backup.Get("/:id", handlers.HandleBackupDownload)

    // Admin routes with audit logging
    admin := api.Group("/admin")
    admin.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      auditMgr,
        ResourceType: "admin",
        ActionMapper: middleware.AdminActionMapper,
    }))
    admin.Post("/ratelimit", handlers.HandleRateLimitCreate)
    admin.Put("/ratelimit/:id", handlers.HandleRateLimitUpdate)
}
```

### 3. Manual Event Recording (For Non-HTTP Operations)

```go
package handlers

import (
    "context"
    "github.com/neogan74/konsul/internal/audit"
)

// Example: Background cleanup job
func CleanupExpiredServices(ctx context.Context, mgr *audit.Manager, store *storage.Store) error {
    services, err := store.GetExpiredServices()
    if err != nil {
        return err
    }

    for _, svc := range services {
        if err := store.DeleteService(svc.ID); err != nil {
            continue
        }

        // Record audit event for background deletion
        event := &audit.Event{
            Action: "service.cleanup",
            Result: "success",
            Resource: audit.Resource{
                Type: "service",
                ID:   svc.ID,
            },
            Actor: audit.Actor{
                Type: "system",
                Name: "cleanup-job",
            },
            Metadata: map[string]string{
                "reason":      "ttl_expired",
                "expired_at":  svc.ExpiresAt.String(),
                "service_name": svc.Name,
            },
        }

        if _, err := mgr.Record(ctx, event); err != nil {
            log.Error("Failed to record audit event", logger.Error(err))
        }
    }

    return nil
}
```

### 4. Custom Action Mapper Example

```go
// Custom action mapper for GraphQL operations
func GraphQLActionMapper(c *fiber.Ctx) string {
    // Parse GraphQL query to determine operation
    var req struct {
        Query string `json:"query"`
    }

    if err := c.BodyParser(&req); err != nil {
        return "graphql.unknown"
    }

    // Simple query type detection
    if strings.Contains(req.Query, "mutation") {
        if strings.Contains(req.Query, "createService") {
            return "graphql.service.create"
        } else if strings.Contains(req.Query, "updateService") {
            return "graphql.service.update"
        }
        return "graphql.mutation"
    } else if strings.Contains(req.Query, "query") {
        return "graphql.query"
    }

    return "graphql.unknown"
}

// Apply to GraphQL endpoint
graphql := api.Group("/graphql")
graphql.Use(middleware.AuditMiddleware(middleware.AuditConfig{
    Manager:      auditMgr,
    ResourceType: "graphql",
    ActionMapper: GraphQLActionMapper,
}))
graphql.Post("/", handlers.HandleGraphQL)
```

## Sample Output

When audit logging is enabled, you'll see events like:

```json
{
  "event_id": "8f7a3c2e-9b4d-4f1a-8e2c-5d6f7a8b9c0d",
  "timestamp": "2025-02-14T15:30:45Z",
  "action": "kv.set",
  "result": "success",
  "resource": {
    "type": "kv",
    "id": "config/database/url"
  },
  "actor": {
    "id": "user-456",
    "type": "user",
    "name": "jane.smith",
    "roles": ["developer"]
  },
  "source_ip": "10.0.1.42",
  "auth_method": "jwt",
  "http_method": "PUT",
  "http_path": "/api/v1/kv/config/database/url",
  "http_status": 200,
  "request_hash": "a3c5e7d9f1b2e4a6c8d0f2e4a6c8d0f2"
}
```

## Testing Audit Integration

```go
func TestAuditLogging(t *testing.T) {
    // Create temp dir for audit logs
    dir := t.TempDir()
    auditPath := filepath.Join(dir, "audit.log")

    // Initialize audit manager
    mgr, err := audit.NewManager(audit.Config{
        Enabled:       true,
        Sink:          "file",
        FilePath:      auditPath,
        BufferSize:    10,
        FlushInterval: 10 * time.Millisecond,
    }, logger.GetDefault())
    require.NoError(t, err)
    defer mgr.Shutdown(context.Background())

    // Setup test app
    app := fiber.New()
    app.Use(middleware.AuditMiddleware(middleware.AuditConfig{
        Manager:      mgr,
        ResourceType: "kv",
        ActionMapper: middleware.KVActionMapper,
    }))
    app.Put("/kv/:key", func(c *fiber.Ctx) error {
        return c.SendStatus(200)
    })

    // Make request
    req := httptest.NewRequest("PUT", "/kv/test-key", strings.NewReader(`{"value":"test"}`))
    resp, _ := app.Test(req)
    require.Equal(t, 200, resp.StatusCode)

    // Wait for flush
    time.Sleep(50 * time.Millisecond)
    mgr.Shutdown(context.Background())

    // Verify audit log
    data, err := os.ReadFile(auditPath)
    require.NoError(t, err)
    require.Contains(t, string(data), "kv.set")
    require.Contains(t, string(data), "success")
}
```

## Configuration for Different Environments

### Development

```bash
export KONSUL_AUDIT_ENABLED=true
export KONSUL_AUDIT_SINK=stdout
export KONSUL_AUDIT_DROP_POLICY=drop
```

### Production

```bash
export KONSUL_AUDIT_ENABLED=true
export KONSUL_AUDIT_SINK=file
export KONSUL_AUDIT_FILE_PATH=/var/log/konsul/audit.log
export KONSUL_AUDIT_BUFFER_SIZE=4096
export KONSUL_AUDIT_FLUSH_INTERVAL=5s
export KONSUL_AUDIT_DROP_POLICY=drop
```

### High-Compliance Environment

```bash
export KONSUL_AUDIT_ENABLED=true
export KONSUL_AUDIT_SINK=file
export KONSUL_AUDIT_FILE_PATH=/var/log/konsul/audit.log
export KONSUL_AUDIT_BUFFER_SIZE=8192
export KONSUL_AUDIT_FLUSH_INTERVAL=1s
export KONSUL_AUDIT_DROP_POLICY=block  # Never drop events
```

## Next Steps

1. Review the full [Audit Logging Guide](../audit-logging.md)
2. Configure log rotation for production
3. Set up SIEM integration
4. Configure Prometheus alerts for dropped events
5. Review audit logs regularly for security events
