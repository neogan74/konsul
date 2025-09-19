package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/store"
)

type ServiceHandler struct {
	store *store.ServiceStore
}

func NewServiceHandler(serviceStore *store.ServiceStore) *ServiceHandler {
	return &ServiceHandler{store: serviceStore}
}

func (h *ServiceHandler) Register(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	var svc store.Service

	if err := c.BodyParser(&svc); err != nil {
		log.Error("Failed to parse service registration body", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if svc.Name == "" || svc.Address == "" || svc.Port == 0 {
		log.Warn("Service registration missing required fields",
			logger.String("name", svc.Name),
			logger.String("address", svc.Address),
			logger.Int("port", svc.Port))
		return middleware.BadRequest(c, "Missing required fields: name, address, and port")
	}

	log.Info("Registering service",
		logger.String("service_name", svc.Name),
		logger.String("address", svc.Address),
		logger.Int("port", svc.Port))

	h.store.Register(svc)

	log.Info("Service registered successfully",
		logger.String("service_name", svc.Name))

	metrics.ServiceOperationsTotal.WithLabelValues("register", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))

	return c.JSON(fiber.Map{"message": "service registered", "service": svc})
}

func (h *ServiceHandler) List(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	services := h.store.List()

	log.Debug("Listing services", logger.Int("count", len(services)))
	return c.JSON(services)
}

func (h *ServiceHandler) Get(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	log.Debug("Getting service", logger.String("service_name", name))

	svc, ok := h.store.Get(name)
	if !ok {
		log.Warn("Service not found", logger.String("service_name", name))
		metrics.ServiceOperationsTotal.WithLabelValues("get", "not_found").Inc()
		return middleware.NotFound(c, "Service not found")
	}

	log.Info("Service retrieved successfully", logger.String("service_name", name))
	metrics.ServiceOperationsTotal.WithLabelValues("get", "success").Inc()
	return c.JSON(svc)
}

func (h *ServiceHandler) Deregister(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	log.Info("Deregistering service", logger.String("service_name", name))

	h.store.Deregister(name)

	log.Info("Service deregistered successfully", logger.String("service_name", name))
	metrics.ServiceOperationsTotal.WithLabelValues("deregister", "success").Inc()
	metrics.RegisteredServicesTotal.Set(float64(len(h.store.List())))
	return c.JSON(fiber.Map{"message": "service deregistered", "name": name})
}

func (h *ServiceHandler) Heartbeat(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	log.Debug("Processing heartbeat", logger.String("service_name", name))

	if h.store.Heartbeat(name) {
		log.Info("Heartbeat updated successfully", logger.String("service_name", name))
		metrics.ServiceHeartbeatsTotal.WithLabelValues(name, "success").Inc()
		return c.JSON(fiber.Map{"message": "heartbeat updated", "service": name})
	}

	log.Warn("Heartbeat failed - service not found", logger.String("service_name", name))
	metrics.ServiceHeartbeatsTotal.WithLabelValues(name, "not_found").Inc()
	return middleware.NotFound(c, "Service not found")
}