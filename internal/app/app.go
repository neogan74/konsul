package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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
)

const shutdownTimeout = 5 * time.Second

// Builder wires Konsul application dependencies.
type Builder struct {
	cfg            *config.Config
	version        string
	logger         logger.Logger
	fiberApp       *fiber.App
	engine         persistence.Engine
	kvStore        *store.KVStore
	serviceStore   *store.ServiceStore
	rateLimitSvc   *ratelimit.Service
	tracerProvider *telemetry.TracerProvider
	dnsServer      *dns.Server
	closers        []func()
}

// NewBuilder creates a new application builder.
func NewBuilder(cfg *config.Config, version string) *Builder {
	return &Builder{cfg: cfg, version: version}
}

// Build assembles the Konsul application components.
func (b *Builder) Build(ctx context.Context) (*App, error) {
	b.initLogger()
	b.recordStartupMetrics()
	b.initFiber()
	b.initTracing(ctx)
	b.initMiddleware()

	if err := b.initPersistence(); err != nil {
		b.cleanupOnError()
		return nil, err
	}

	if err := b.initStores(); err != nil {
		b.cleanupOnError()
		return nil, err
	}

	b.initHandlers()
	b.initDNS()

	return &App{
		cfg:            b.cfg,
		version:        b.version,
		logger:         b.logger,
		fiberApp:       b.fiberApp,
		serviceStore:   b.serviceStore,
		rateLimitSvc:   b.rateLimitSvc,
		tracerProvider: b.tracerProvider,
		dnsServer:      b.dnsServer,
		closers:        b.closers,
	}, nil
}

func (b *Builder) initLogger() {
	b.logger = logger.NewFromConfig(b.cfg.Log.Level, b.cfg.Log.Format)
	logger.SetDefault(b.logger)
}

func (b *Builder) recordStartupMetrics() {
	metrics.BuildInfo.WithLabelValues(b.version, runtime.Version()).Set(1)

	b.logger.Info("Starting Konsul",
		logger.String("version", b.version),
		logger.String("address", b.cfg.Address()),
		logger.String("log_level", b.cfg.Log.Level),
		logger.String("log_format", b.cfg.Log.Format),
		logger.String("persistence_enabled", fmt.Sprintf("%t", b.cfg.Persistence.Enabled)),
		logger.String("persistence_type", b.cfg.Persistence.Type),
	)
}

func (b *Builder) initFiber() {
	b.fiberApp = fiber.New()
}

func (b *Builder) initTracing(ctx context.Context) {
	tracingCfg := telemetry.TracingConfig{
		Enabled:        b.cfg.Tracing.Enabled,
		Endpoint:       b.cfg.Tracing.Endpoint,
		ServiceName:    b.cfg.Tracing.ServiceName,
		ServiceVersion: b.cfg.Tracing.ServiceVersion,
		Environment:    b.cfg.Tracing.Environment,
		SamplingRatio:  b.cfg.Tracing.SamplingRatio,
		InsecureConn:   b.cfg.Tracing.InsecureConn,
	}

	provider, err := telemetry.InitTracing(ctx, tracingCfg)
	if err != nil {
		b.logger.Error("Failed to initialize tracing", logger.Error(err))
		return
	}

	if b.cfg.Tracing.Enabled {
		b.logger.Info("OpenTelemetry tracing initialized",
			logger.String("endpoint", b.cfg.Tracing.Endpoint),
			logger.String("service_name", b.cfg.Tracing.ServiceName),
		)

		b.addCloser(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			if err := provider.Shutdown(shutdownCtx); err != nil {
				b.logger.Error("Failed to shutdown tracer provider", logger.Error(err))
			}
		})
	}

	b.tracerProvider = provider
}

func (b *Builder) initMiddleware() {
	b.fiberApp.Use(middleware.RequestLogging(b.logger))
	b.fiberApp.Use(middleware.MetricsMiddleware())

	if b.cfg.Tracing.Enabled {
		b.fiberApp.Use(middleware.TracingMiddleware(b.cfg.Tracing.ServiceName))
	}

	if b.cfg.RateLimit.Enabled {
		b.rateLimitSvc = ratelimit.NewService(ratelimit.Config{
			Enabled:         b.cfg.RateLimit.Enabled,
			RequestsPerSec:  b.cfg.RateLimit.RequestsPerSec,
			Burst:           b.cfg.RateLimit.Burst,
			ByIP:            b.cfg.RateLimit.ByIP,
			ByAPIKey:        b.cfg.RateLimit.ByAPIKey,
			CleanupInterval: b.cfg.RateLimit.CleanupInterval,
		})

		b.fiberApp.Use(middleware.RateLimitMiddleware(b.rateLimitSvc))

		b.logger.Info("Rate limiting enabled",
			logger.String("requests_per_sec", fmt.Sprintf("%.1f", b.cfg.RateLimit.RequestsPerSec)),
			logger.Int("burst", b.cfg.RateLimit.Burst),
			logger.String("by_ip", fmt.Sprintf("%t", b.cfg.RateLimit.ByIP)),
			logger.String("by_apikey", fmt.Sprintf("%t", b.cfg.RateLimit.ByAPIKey)),
		)
	}
}

func (b *Builder) initPersistence() error {
	if !b.cfg.Persistence.Enabled {
		return nil
	}

	engine, err := persistence.NewEngine(persistence.Config{
		Enabled:    b.cfg.Persistence.Enabled,
		Type:       b.cfg.Persistence.Type,
		DataDir:    b.cfg.Persistence.DataDir,
		BackupDir:  b.cfg.Persistence.BackupDir,
		SyncWrites: b.cfg.Persistence.SyncWrites,
		WALEnabled: b.cfg.Persistence.WALEnabled,
	}, b.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize persistence engine: %w", err)
	}

	b.engine = engine

	b.addCloser(func() {
		if err := engine.Close(); err != nil {
			b.logger.Error("Failed to close persistence engine", logger.Error(err))
		}
	})

	return nil
}

func (b *Builder) initStores() error {
	var (
		kv  *store.KVStore
		err error
	)

	if b.cfg.Persistence.Enabled {
		kv, err = store.NewKVStoreWithPersistence(b.engine, b.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize KV store: %w", err)
		}

		serviceStore, svcErr := store.NewServiceStoreWithPersistence(b.cfg.Service.TTL, b.engine, b.logger)
		if svcErr != nil {
			return fmt.Errorf("failed to initialize service store: %w", svcErr)
		}
		b.serviceStore = serviceStore
	} else {
		kv = store.NewKVStore()
		b.serviceStore = store.NewServiceStoreWithTTL(b.cfg.Service.TTL)
	}

	b.kvStore = kv

	b.addCloser(func() {
		if err := kv.Close(); err != nil {
			b.logger.Error("Failed to close KV store", logger.Error(err))
		}
	})

	b.addCloser(func() {
		if err := b.serviceStore.Close(); err != nil {
			b.logger.Error("Failed to close service store", logger.Error(err))
		}
	})

	metrics.KVStoreSize.Set(float64(len(b.kvStore.List())))
	metrics.RegisteredServicesTotal.Set(float64(len(b.serviceStore.List())))

	return nil
}

func (b *Builder) initHandlers() {
	kvHandler := handlers.NewKVHandler(b.kvStore)
	serviceHandler := handlers.NewServiceHandler(b.serviceStore)
	healthHandler := handlers.NewHealthHandler(b.kvStore, b.serviceStore, b.version)
	healthCheckHandler := handlers.NewHealthCheckHandler(b.serviceStore)
	backupHandler := handlers.NewBackupHandler(b.engine, b.logger)

	var (
		jwtService  *auth.JWTService
		authHandler *handlers.AuthHandler
	)

	if b.cfg.Auth.Enabled {
		jwtService = auth.NewJWTService(
			b.cfg.Auth.JWTSecret,
			b.cfg.Auth.JWTExpiry,
			b.cfg.Auth.RefreshExpiry,
			b.cfg.Auth.Issuer,
		)
		apiKeyService := auth.NewAPIKeyService(b.cfg.Auth.APIKeyPrefix)
		authHandler = handlers.NewAuthHandler(jwtService, apiKeyService)
	}

	if b.cfg.Auth.Enabled {
		b.fiberApp.Post("/auth/login", authHandler.Login)
		b.fiberApp.Post("/auth/refresh", authHandler.Refresh)
		b.fiberApp.Get("/auth/verify", middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths), authHandler.Verify)

		b.fiberApp.Post("/auth/apikeys", middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths), authHandler.CreateAPIKey)
		b.fiberApp.Get("/auth/apikeys", middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths), authHandler.ListAPIKeys)
		b.fiberApp.Get("/auth/apikeys/:id", middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths), authHandler.GetAPIKey)
		b.fiberApp.Put("/auth/apikeys/:id", middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths), authHandler.UpdateAPIKey)
		b.fiberApp.Delete("/auth/apikeys/:id", middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths), authHandler.DeleteAPIKey)
		b.fiberApp.Post("/auth/apikeys/:id/revoke", middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths), authHandler.RevokeAPIKey)
	}

	if b.cfg.Auth.RequireAuth && b.cfg.Auth.Enabled {
		b.fiberApp.Use(middleware.JWTAuth(jwtService, b.cfg.Auth.PublicPaths))
	}

	b.fiberApp.Get("/kv/", kvHandler.List)
	b.fiberApp.Get("/kv/:key", kvHandler.Get)
	b.fiberApp.Put("/kv/:key", kvHandler.Set)
	b.fiberApp.Delete("/kv/:key", kvHandler.Delete)

	b.fiberApp.Put("/register", serviceHandler.Register)
	b.fiberApp.Get("/services/", serviceHandler.List)
	b.fiberApp.Get("/services/:name", serviceHandler.Get)
	b.fiberApp.Delete("/deregister/:name", serviceHandler.Deregister)
	b.fiberApp.Put("/heartbeat/:name", serviceHandler.Heartbeat)

	b.fiberApp.Get("/health", healthHandler.Check)
	b.fiberApp.Get("/health/live", healthHandler.Liveness)
	b.fiberApp.Get("/health/ready", healthHandler.Readiness)

	b.fiberApp.Get("/health/checks", healthCheckHandler.ListChecks)
	b.fiberApp.Get("/health/service/:name", healthCheckHandler.GetServiceChecks)
	b.fiberApp.Put("/health/check/:id", healthCheckHandler.UpdateTTLCheck)

	b.fiberApp.Post("/backup", backupHandler.CreateBackup)
	b.fiberApp.Post("/restore", backupHandler.RestoreBackup)
	b.fiberApp.Get("/export", backupHandler.ExportData)
	b.fiberApp.Post("/import", backupHandler.ImportData)
	b.fiberApp.Get("/backups", backupHandler.ListBackups)

	b.fiberApp.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

func (b *Builder) initDNS() {
	if !b.cfg.DNS.Enabled {
		return
	}

	cfg := dns.Config{
		Host:   b.cfg.DNS.Host,
		Port:   b.cfg.DNS.Port,
		Domain: b.cfg.DNS.Domain,
	}

	b.dnsServer = dns.NewServer(cfg, b.serviceStore, b.logger)
}

func (b *Builder) addCloser(closer func()) {
	b.closers = append(b.closers, closer)
}

func (b *Builder) cleanupOnError() {
	for i := len(b.closers) - 1; i >= 0; i-- {
		b.closers[i]()
	}
}

// App represents a configured Konsul application ready to run.
type App struct {
	cfg            *config.Config
	version        string
	logger         logger.Logger
	fiberApp       *fiber.App
	serviceStore   *store.ServiceStore
	rateLimitSvc   *ratelimit.Service
	tracerProvider *telemetry.TracerProvider
	dnsServer      *dns.Server
	closers        []func()
	backgroundStop []func()
}

// Run starts the Konsul application and handles graceful shutdown.
func (a *App) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.prepareTLS(); err != nil {
		return err
	}

	a.startBackgroundTasks()
	a.startDNS()

	serverErr := make(chan error, 1)

	go func() {
		if a.cfg.Server.TLS.Enabled {
			serverErr <- a.fiberApp.ListenTLS(a.cfg.Address(), a.cfg.Server.TLS.CertFile, a.cfg.Server.TLS.KeyFile)
		} else {
			serverErr <- a.fiberApp.Listen(a.cfg.Address())
		}
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			a.logger.Error("Failed to start server", logger.Error(err))
			a.stopBackgroundTasks()
			a.stopDNS()
			a.runClosers()
			return err
		}
		return nil
	case <-ctx.Done():
	}

	a.logger.Info("Shutting down server...")

	a.stopBackgroundTasks()
	a.stopDNS()

	if err := a.fiberApp.Shutdown(); err != nil {
		a.logger.Error("Server forced to shutdown", logger.Error(err))
	}

	a.runClosers()

	if err := <-serverErr; err != nil {
		return err
	}

	a.logger.Info("Server exited gracefully")
	return nil
}

func (a *App) startBackgroundTasks() {
	if a.serviceStore != nil {
		stop := a.startServiceCleanup()
		a.backgroundStop = append(a.backgroundStop, stop)
	}

	if a.rateLimitSvc != nil {
		stop := a.startRateLimitMetrics()
		a.backgroundStop = append(a.backgroundStop, stop)
	}
}

func (a *App) stopBackgroundTasks() {
	for i := len(a.backgroundStop) - 1; i >= 0; i-- {
		a.backgroundStop[i]()
	}
	a.backgroundStop = nil
}

func (a *App) startServiceCleanup() func() {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(a.cfg.Service.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count := a.serviceStore.CleanupExpired()
				if count > 0 {
					a.logger.Info("Cleaned up expired services", logger.Int("count", count))
					metrics.ExpiredServicesTotal.Add(float64(count))
					metrics.RegisteredServicesTotal.Set(float64(len(a.serviceStore.List())))
				}
			case <-stop:
				return
			}
		}
	}()

	return func() { close(stop) }
}

func (a *App) startRateLimitMetrics() func() {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				stats := a.rateLimitSvc.Stats()
				if ipCount, ok := stats["ip_limiters"].(int); ok {
					metrics.RateLimitActiveClients.WithLabelValues("ip").Set(float64(ipCount))
				}
				if keyCount, ok := stats["apikey_limiters"].(int); ok {
					metrics.RateLimitActiveClients.WithLabelValues("apikey").Set(float64(keyCount))
				}
			case <-stop:
				return
			}
		}
	}()

	return func() { close(stop) }
}

func (a *App) startDNS() {
	if a.dnsServer == nil {
		return
	}

	if err := a.dnsServer.Start(); err != nil {
		a.logger.Error("Failed to start DNS server", logger.Error(err))
		return
	}

	a.logger.Info("DNS server started",
		logger.String("domain", a.cfg.DNS.Domain),
		logger.Int("port", a.cfg.DNS.Port),
	)
}

func (a *App) stopDNS() {
	if a.dnsServer == nil {
		return
	}

	if err := a.dnsServer.Stop(); err != nil {
		a.logger.Error("Failed to stop DNS server", logger.Error(err))
		return
	}

	a.logger.Info("DNS server stopped")
}

func (a *App) runClosers() {
	for i := len(a.closers) - 1; i >= 0; i-- {
		a.closers[i]()
	}
}

func (a *App) prepareTLS() error {
	if !a.cfg.Server.TLS.Enabled {
		a.logger.Info("Server starting", logger.String("address", a.cfg.Address()))
		return nil
	}

	if a.cfg.Server.TLS.AutoCert {
		certFile := "./certs/server.crt"
		keyFile := "./certs/server.key"

		if err := os.MkdirAll("./certs", 0o755); err != nil {
			a.logger.Error("Failed to create certs directory", logger.Error(err))
			return fmt.Errorf("failed to create certs directory: %w", err)
		}

		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			a.logger.Info("Generating self-signed TLS certificate for development")
			if err := konsultls.GenerateSelfSignedCert(certFile, keyFile); err != nil {
				a.logger.Error("Failed to generate self-signed certificate", logger.Error(err))
				return fmt.Errorf("failed to generate certificate: %w", err)
			}
			a.logger.Info("Self-signed certificate generated",
				logger.String("cert", certFile),
				logger.String("key", keyFile),
			)
		}

		a.cfg.Server.TLS.CertFile = certFile
		a.cfg.Server.TLS.KeyFile = keyFile
	}

	a.logger.Info("Server starting with TLS",
		logger.String("address", a.cfg.Address()),
		logger.String("cert", a.cfg.Server.TLS.CertFile),
		logger.String("key", a.cfg.Server.TLS.KeyFile),
	)

	return nil
}
