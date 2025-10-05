package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/neogan74/konsul/internal/auth"
	"github.com/neogan74/konsul/internal/config"
	"github.com/neogan74/konsul/internal/dns"
	"github.com/neogan74/konsul/internal/handlers"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/persistence"
	"github.com/neogan74/konsul/internal/ratelimit"
	"github.com/neogan74/konsul/internal/store"
	"github.com/neogan74/konsul/internal/telemetry"
	konsultls "github.com/neogan74/konsul/internal/tls"
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
			logger.String("requests_per_sec", fmt.Sprintf("%.1f", cfg.RateLimit.RequestsPerSec)),
			logger.Int("burst", cfg.RateLimit.Burst),
			logger.String("by_ip", fmt.Sprintf("%t", cfg.RateLimit.ByIP)),
			logger.String("by_apikey", fmt.Sprintf("%t", cfg.RateLimit.ByAPIKey)),
		)

		// Update rate limit metrics periodically
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				stats := rateLimitService.Stats()
				if ipCount, ok := stats["ip_limiters"].(int); ok {
					metrics.RateLimitActiveClients.WithLabelValues("ip").Set(float64(ipCount))
				}
				if keyCount, ok := stats["apikey_limiters"].(int); ok {
					metrics.RateLimitActiveClients.WithLabelValues("apikey").Set(float64(keyCount))
				}
			}
		}()
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

	// Initialize auth services if enabled
	var jwtService *auth.JWTService
	var authHandler *handlers.AuthHandler
	if cfg.Auth.Enabled {
		jwtService = auth.NewJWTService(
			cfg.Auth.JWTSecret,
			cfg.Auth.JWTExpiry,
			cfg.Auth.RefreshExpiry,
			cfg.Auth.Issuer,
		)
		apiKeyService := auth.NewAPIKeyService(cfg.Auth.APIKeyPrefix)
		authHandler = handlers.NewAuthHandler(jwtService, apiKeyService)
	}

	// Auth endpoints (public)
	if cfg.Auth.Enabled {
		app.Post("/auth/login", authHandler.Login)
		app.Post("/auth/refresh", authHandler.Refresh)
		app.Get("/auth/verify", middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths), authHandler.Verify)

		// API key management endpoints (protected)
		app.Post("/auth/apikeys", middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths), authHandler.CreateAPIKey)
		app.Get("/auth/apikeys", middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths), authHandler.ListAPIKeys)
		app.Get("/auth/apikeys/:id", middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths), authHandler.GetAPIKey)
		app.Put("/auth/apikeys/:id", middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths), authHandler.UpdateAPIKey)
		app.Delete("/auth/apikeys/:id", middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths), authHandler.DeleteAPIKey)
		app.Post("/auth/apikeys/:id/revoke", middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths), authHandler.RevokeAPIKey)
	}

	// Apply auth middleware to protected routes if required
	if cfg.Auth.RequireAuth && cfg.Auth.Enabled {
		app.Use(middleware.JWTAuth(jwtService, cfg.Auth.PublicPaths))
	}

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

	// Handle TLS configuration
	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.AutoCert {
			certFile := "./certs/server.crt"
			keyFile := "./certs/server.key"

			// Create certs directory if it doesn't exist
			if err := os.MkdirAll("./certs", 0755); err != nil {
				appLogger.Error("Failed to create certs directory", logger.Error(err))
				log.Fatalf("Failed to create certs directory: %v", err)
			}

			// Generate self-signed certificate if files don't exist
			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				appLogger.Info("Generating self-signed TLS certificate for development")
				if err := konsultls.GenerateSelfSignedCert(certFile, keyFile); err != nil {
					appLogger.Error("Failed to generate self-signed certificate", logger.Error(err))
					log.Fatalf("Failed to generate certificate: %v", err)
				}
				appLogger.Info("Self-signed certificate generated",
					logger.String("cert", certFile),
					logger.String("key", keyFile))
			}

			cfg.Server.TLS.CertFile = certFile
			cfg.Server.TLS.KeyFile = keyFile
		}

		appLogger.Info("Server starting with TLS",
			logger.String("address", cfg.Address()),
			logger.String("cert", cfg.Server.TLS.CertFile),
			logger.String("key", cfg.Server.TLS.KeyFile))
	} else {
		appLogger.Info("Server starting", logger.String("address", cfg.Address()))
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		if cfg.Server.TLS.Enabled {
			if err := app.ListenTLS(cfg.Address(), cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil {
				appLogger.Error("Failed to start TLS server", logger.Error(err))
				log.Fatalf("Listen TLS error: %v", err)
			}
		} else {
			if err := app.Listen(cfg.Address()); err != nil {
				appLogger.Error("Failed to start server", logger.Error(err))
				log.Fatalf("Listen error: %v", err)
			}
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
