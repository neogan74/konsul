package agent

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/healthcheck"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

// API represents the agent HTTP API server
type API struct {
	agent *Agent
	app   *fiber.App
	log   logger.Logger
}

// NewAPI creates a new API server
func NewAPI(agent *Agent) *API {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler,
	})

	api := &API{
		agent: agent,
		app:   app,
		log:   agent.log,
	}

	api.setupRoutes()
	return api
}

// setupRoutes sets up the API routes
func (api *API) setupRoutes() {
	// Health endpoint
	api.app.Get("/health", api.handleHealth)

	// Agent self-inspection
	api.app.Get("/agent/self", api.handleSelf)
	api.app.Get("/agent/metrics", api.handleMetrics)
	api.app.Get("/agent/stats", api.handleStats)

	// Local service management
	api.app.Post("/agent/service/register", api.handleRegisterService)
	api.app.Delete("/agent/service/deregister/:name", api.handleDeregisterService)
	api.app.Get("/agent/services", api.handleListServices)

	// Service discovery (cached)
	api.app.Get("/agent/catalog/service/:name", api.handleGetService)
	api.app.Get("/agent/catalog/services", api.handleListAllServices)

	// KV store (cached)
	api.app.Get("/agent/kv/:key", api.handleGetKV)
	api.app.Put("/agent/kv/:key", api.handleSetKV)
	api.app.Delete("/agent/kv/:key", api.handleDeleteKV)

	// Health checks
	api.app.Post("/agent/check/register", api.handleRegisterCheck)
	api.app.Delete("/agent/check/deregister/:id", api.handleDeregisterCheck)
	api.app.Get("/agent/checks", api.handleListChecks)
	api.app.Put("/agent/check/update/:id", api.handleUpdateTTLCheck)
}

// Start starts the API server
func (api *API) Start() error {
	bindAddr := api.agent.config.BindAddress
	api.log.Info("Starting agent API server", logger.String("address", bindAddr))

	go func() {
		if err := api.app.Listen(bindAddr); err != nil {
			api.log.Error("Agent API server failed", logger.Error(err))
		}
	}()

	return nil
}

// Stop stops the API server
func (api *API) Stop() error {
	api.log.Info("Stopping agent API server")
	return api.app.Shutdown()
}

// Handlers

func (api *API) handleHealth(c *fiber.Ctx) error {
	healthy := api.agent.Health()
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	return c.JSON(fiber.Map{
		"status":   status,
		"agent_id": api.agent.info.ID,
		"uptime":   api.agent.Stats().Uptime,
	})
}

func (api *API) handleSelf(c *fiber.Ctx) error {
	return c.JSON(api.agent.Info())
}

func (api *API) handleMetrics(c *fiber.Ctx) error {
	stats := api.agent.Stats()

	metrics := fiber.Map{
		"cache": fiber.Map{
			"hit_rate":      stats.CacheHitRate,
			"entries":       stats.CacheEntries,
			"hits":          api.agent.cache.Hits(),
			"misses":        api.agent.cache.Misses(),
			"service_count": api.agent.cache.ServiceCount(),
			"kv_count":      api.agent.cache.KVCount(),
			"health_count":  api.agent.cache.HealthCount(),
		},
		"sync": fiber.Map{
			"last_sync":     stats.LastSyncTime,
			"errors_total":  stats.SyncErrorsTotal,
			"sync_count":    api.agent.syncEngine.GetSyncCount(),
			"last_index":    api.agent.syncEngine.GetLastIndex(),
			"pending_count": api.agent.syncEngine.GetPendingCount(),
		},
		"services": fiber.Map{
			"local_count": stats.LocalServices,
		},
	}

	return c.JSON(metrics)
}

func (api *API) handleStats(c *fiber.Ctx) error {
	return c.JSON(api.agent.Stats())
}

// Service handlers

func (api *API) handleRegisterService(c *fiber.Ctx) error {
	var svc store.Service
	if err := c.BodyParser(&svc); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid service data")
	}

	if err := api.agent.RegisterService(svc); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "registered",
		"service": svc.Name,
	})
}

func (api *API) handleDeregisterService(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Service name is required")
	}

	if err := api.agent.DeregisterService(name); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(fiber.Map{
		"status":  "deregistered",
		"service": name,
	})
}

func (api *API) handleListServices(c *fiber.Ctx) error {
	services := api.agent.ListLocalServices()
	return c.JSON(services)
}

// Catalog handlers

func (api *API) handleGetService(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Service name is required")
	}

	entries, err := api.agent.GetService(name)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if entries == nil {
		return fiber.NewError(fiber.StatusNotFound, "Service not found")
	}

	return c.JSON(entries)
}

func (api *API) handleListAllServices(c *fiber.Ctx) error {
	// For now, return only locally registered services
	// In the future, this could query the cache for all known services
	services := api.agent.ListLocalServices()
	return c.JSON(services)
}

// KV handlers

func (api *API) handleGetKV(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Key is required")
	}

	entry, err := api.agent.GetKV(key)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if entry == nil {
		return fiber.NewError(fiber.StatusNotFound, "Key not found")
	}

	return c.JSON(entry)
}

func (api *API) handleSetKV(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Key is required")
	}

	var entry store.KVEntry
	if err := c.BodyParser(&entry); err != nil {
		// If parsing as KVEntry fails, try to get raw value
		value := string(c.Body())
		if value == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Value is required")
		}
		entry = store.KVEntry{
			Value: value,
		}
	}

	if err := api.agent.SetKV(key, &entry); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "stored",
		"key":    key,
	})
}

func (api *API) handleDeleteKV(c *fiber.Ctx) error {
	key := c.Params("key")
	if key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Key is required")
	}

	if err := api.agent.DeleteKV(key); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
		"key":    key,
	})
}

// Health check handlers

func (api *API) handleRegisterCheck(c *fiber.Ctx) error {
	var def healthcheck.CheckDefinition
	if err := c.BodyParser(&def); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid check definition")
	}

	if def.ServiceID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Service ID is required")
	}

	if err := api.agent.healthChecker.RegisterCheck(def.ServiceID, &def); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"status":   "registered",
		"check_id": def.ID,
	})
}

func (api *API) handleDeregisterCheck(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Check ID is required")
	}

	if err := api.agent.healthChecker.DeregisterCheck(id); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(fiber.Map{
		"status":   "deregistered",
		"check_id": id,
	})
}

func (api *API) handleListChecks(c *fiber.Ctx) error {
	checks := api.agent.healthChecker.ListChecks()
	return c.JSON(checks)
}

func (api *API) handleUpdateTTLCheck(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Check ID is required")
	}

	if err := api.agent.healthChecker.UpdateTTLCheck(id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{
		"status":   "updated",
		"check_id": id,
	})
}

// Error handler

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   true,
		"message": message,
	})
}
