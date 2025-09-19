package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/neogan74/konsul/internal/config"
	"github.com/neogan74/konsul/internal/handlers"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/middleware"
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
		logger.String("log_format", cfg.Log.Format))

	// Set build info metrics
	metrics.BuildInfo.WithLabelValues(version, runtime.Version()).Set(1)

	app := fiber.New()

	// Add middleware
	app.Use(middleware.RequestLogging(appLogger))
	app.Use(middleware.MetricsMiddleware())

	// Initialize stores
	kv := store.NewKVStore()
	svcStore := store.NewServiceStoreWithTTL(cfg.Service.TTL)

	// Initialize handlers
	kvHandler := handlers.NewKVHandler(kv)
	serviceHandler := handlers.NewServiceHandler(svcStore)
	healthHandler := handlers.NewHealthHandler(kv, svcStore, version)

	// Initialize store metrics
	metrics.KVStoreSize.Set(float64(len(kv.List())))
	metrics.RegisteredServicesTotal.Set(float64(len(svcStore.List())))

	// KV endpoints
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

	// Metrics endpoint for Prometheus
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Start background cleanup process
	go func() {
		ticker := time.NewTicker(cfg.Service.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count := svcStore.CleanupExpired()
				if count > 0 {
					appLogger.Info("Cleaned up expired services", logger.Int("count", count))
					metrics.ExpiredServicesTotal.Add(float64(count))
					metrics.RegisteredServicesTotal.Set(float64(len(svcStore.List())))
				}
			}
		}
	}()

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
	if err := app.Shutdown(); err != nil {
		appLogger.Error("Server forced to shutdown", logger.Error(err))
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	appLogger.Info("Server exited gracefully")
}
