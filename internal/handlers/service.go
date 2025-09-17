package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/store"
)

type ServiceHandler struct {
	store *store.ServiceStore
}

func NewServiceHandler(serviceStore *store.ServiceStore) *ServiceHandler {
	return &ServiceHandler{store: serviceStore}
}

func (h *ServiceHandler) Register(c *fiber.Ctx) error {
	var svc store.Service
	if err := c.BodyParser(&svc); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if svc.Name == "" || svc.Address == "" || svc.Port == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing fields"})
	}
	h.store.Register(svc)
	return c.JSON(fiber.Map{"message": "service registered", "service": svc})
}

func (h *ServiceHandler) List(c *fiber.Ctx) error {
	return c.JSON(h.store.List())
}

func (h *ServiceHandler) Get(c *fiber.Ctx) error {
	name := c.Params("name")
	svc, ok := h.store.Get(name)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "service not found"})
	}
	return c.JSON(svc)
}

func (h *ServiceHandler) Deregister(c *fiber.Ctx) error {
	name := c.Params("name")
	h.store.Deregister(name)
	return c.JSON(fiber.Map{"message": "service deregistered", "name": name})
}

func (h *ServiceHandler) Heartbeat(c *fiber.Ctx) error {
	name := c.Params("name")
	if h.store.Heartbeat(name) {
		return c.JSON(fiber.Map{"message": "heartbeat updated", "service": name})
	}
	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "service not found"})
}