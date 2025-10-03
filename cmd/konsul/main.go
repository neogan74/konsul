package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/neogan74/konsul/internal/config"
	"github.com/neogan74/konsul/internal/dns"
	"github.com/neogan74/konsul/internal/handlers"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/persistence"
	"github.com/neogan74/konsul/internal/ratelimit"
	"github.com/neogan74/konsul/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize structured logger
	appLogger := logger.NewFromConfig(cfg.Log.Level, cfg.Log.Format)
	logger.SetDefault(appLogger)

	version := "0.1.0"
	appLogger.Info("Starting Konsul",
		logger.String("version", version),
		logger.String("address", cfg.Address()),
		logger.String("log_level", cfg.Log.Level),
		logger.String("log_format", cfg.Log.Format),
		logger.String("persistence_enabled", fmt.Sprintf("%t", cfg.Persistence.Enabled)),
		logger.String("persistence_type", cfg.Persistence.Type))

	// Set build info metrics
	metrics.BuildInfo.WithLabelValues(version, runtime.Version()).Set(1)

	app := fiber.New()

	// Add middleware
	app.Use(middleware.RequestLogging(appLogger))
	app.Use(middleware.MetricsMiddleware())

	// Initialize rate limiting
	var rateLimitService *ratelimit.Service
	if cfg.RateLimit.Enabled {
		rateLimitService = ratelimit.NewService(ratelimit.Config{
			Enabled:         cfg.RateLimit.Enabled,
			RequestsPerSec:  cfg.RateLimit.RequestsPerSec,
			Burst:           cfg.RateLimit.Burst,
			ByIP:            cfg.RateLimit.ByIP,
			ByAPIKey:        cfg.RateLimit.ByAPIKey,
			CleanupInterval: cfg.RateLimit.CleanupInterval,
		})

		app.Use(middleware.RateLimitMiddleware(rateLimitService))

		appLogger.Info("Rate limiting enabled",
			logger.Float64("requests_per_sec", cfg.RateLimit.RequestsPerSec),
			logger.Int("burst", cfg.RateLimit.Burst),
			logger.Bool("by_ip", cfg.RateLimit.ByIP),
			logger.Bool("by_apikey", cfg.RateLimit.ByAPIKey),
		)
	}

	// Initialize persistence engine
	var engine persistence.Engine
	if cfg.Persistence.Enabled {
		engine, err = persistence.NewEngine(persistence.Config{
			Enabled:    cfg.Persistence.Enabled,
			Type:       cfg.Persistence.Type,
			DataDir:    cfg.Persistence.DataDir,
			BackupDir:  cfg.Persistence.BackupDir,
			SyncWrites: cfg.Persistence.SyncWrites,
			WALEnabled: cfg.Persistence.WALEnabled,
		}, appLogger)
		if err != nil {
			log.Fatalf("Failed to initialize persistence engine: %v", err)
		}

		// Ensure graceful shutdown
		defer func() {
			if err := engine.Close(); err != nil {
				appLogger.Error("Failed to close persistence engine", logger.Error(err))
			}
		}()
	}

	// Initialize stores
	var kv *store.KVStore
	var svcStore *store.ServiceStore

	if cfg.Persistence.Enabled {
		kv, err = store.NewKVStoreWithPersistence(engine, appLogger)
		if err != nil {
			log.Fatalf("Failed to initialize KV store: %v", err)
		}

		svcStore, err = store.NewServiceStoreWithPersistence(cfg.Service.TTL, engine, appLogger)
		if err != nil {
			log.Fatalf("Failed to initialize service store: %v", err)
		}
	} else {
		kv = store.NewKVStore()
		svcStore = store.NewServiceStoreWithTTL(cfg.Service.TTL)
	}

	// Ensure stores are closed on shutdown
	defer func() {
		if err := kv.Close(); err != nil {
			appLogger.Error("Failed to close KV store", logger.Error(err))
		}
		if err := svcStore.Close(); err != nil {
			appLogger.Error("Failed to close service store", logger.Error(err))
		}
	}()

	// Initialize handlers
	kvHandler := handlers.NewKVHandler(kv)
	serviceHandler := handlers.NewServiceHandler(svcStore)
	healthHandler := handlers.NewHealthHandler(kv, svcStore, version)
	healthCheckHandler := handlers.NewHealthCheckHandler(svcStore)
	backupHandler := handlers.NewBackupHandler(engine, appLogger)

	// Initialize store metrics
	metrics.KVStoreSize.Set(float64(len(kv.List())))
	metrics.RegisteredServicesTotal.Set(float64(len(svcStore.List())))

	// KV endpoints
	app.Get("/kv/", kvHandler.List)
	app.Get("/kv/:key", kvHandler.Get)
	app.Put("/kv/:key", kvHandler.Set)
	app.Delete("/kv/:key", kvHandler.Delete)

	// Service discovery endpoints
	app.Put("/register", serviceHandler.Register)
	app.Get("/services/", serviceHandler.List)
	app.Get("/services/:name", serviceHandler.Get)
	app.Delete("/deregister/:name", serviceHandler.Deregister)
	app.Put("/heartbeat/:name", serviceHandler.Heartbeat)

	// Health check endpoints
	app.Get("/health", healthHandler.Check)
	app.Get("/health/live", healthHandler.Liveness)
	app.Get("/health/ready", healthHandler.Readiness)

	// Service health check endpoints
	app.Get("/health/checks", healthCheckHandler.ListChecks)
	app.Get("/health/service/:name", healthCheckHandler.GetServiceChecks)
	app.Put("/health/check/:id", healthCheckHandler.UpdateTTLCheck)

	// Backup/restore endpoints
	app.Post("/backup", backupHandler.CreateBackup)
	app.Post("/restore", backupHandler.RestoreBackup)
	app.Get("/export", backupHandler.ExportData)
	app.Post("/import", backupHandler.ImportData)
	app.Get("/backups", backupHandler.ListBackups)

	// Metrics endpoint for Prometheus
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Start background cleanup process
	go func() {
		ticker := time.NewTicker(cfg.Service.CleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			count := svcStore.CleanupExpired()
			if count > 0 {
				appLogger.Info("Cleaned up expired services", logger.Int("count", count))
				metrics.ExpiredServicesTotal.Add(float64(count))
				metrics.RegisteredServicesTotal.Set(float64(len(svcStore.List())))
			}
		}
	}()

	// Start DNS server if enabled
	var dnsServer *dns.Server
	if cfg.DNS.Enabled {
		dnsConfig := dns.Config{
			Host:   cfg.DNS.Host,
			Port:   cfg.DNS.Port,
			Domain: cfg.DNS.Domain,
		}
		dnsServer = dns.NewServer(dnsConfig, svcStore, appLogger)
		if err := dnsServer.Start(); err != nil {
			appLogger.Error("Failed to start DNS server", logger.Error(err))
		} else {
			appLogger.Info("DNS server started",
				logger.String("domain", cfg.DNS.Domain),
				logger.Int("port", cfg.DNS.Port))
		}
	}

	appLogger.Info("Server starting", logger.String("address", cfg.Address()))

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		if err := app.Listen(cfg.Address()); err != nil {
			appLogger.Error("Failed to start server", logger.Error(err))
			log.Fatalf("Listen error: %v", err)
		}
	}()
	<-quit
	appLogger.Info("Shutting down server...")

	// Shutdown DNS server if running
	if dnsServer != nil {
		if err := dnsServer.Stop(); err != nil {
			appLogger.Error("Failed to stop DNS server", logger.Error(err))
		} else {
			appLogger.Info("DNS server stopped")
		}
	}

	if err := app.Shutdown(); err != nil {
		appLogger.Error("Server forced to shutdown", logger.Error(err))
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	appLogger.Info("Server exited gracefully")
}
