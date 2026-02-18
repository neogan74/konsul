package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	hashiraft "github.com/hashicorp/raft"
	"github.com/neogan74/konsul/internal/healthcheck"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/middleware"
	konsulraft "github.com/neogan74/konsul/internal/raft"
	"github.com/neogan74/konsul/internal/store"
)

type HealthCheckHandler struct {
	serviceStore *store.ServiceStore
	raftNode     *konsulraft.Node
}

func NewHealthCheckHandler(serviceStore *store.ServiceStore, raftNode *konsulraft.Node) *HealthCheckHandler {
	return &HealthCheckHandler{
		serviceStore: serviceStore,
		raftNode:     raftNode,
	}
}

func (h *HealthCheckHandler) ListChecks(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	log.Debug("Listing all health checks")

	checks := h.serviceStore.GetAllHealthChecks()

	log.Info("Health checks listed successfully", logger.Int("count", len(checks)))
	return c.JSON(checks)
}

func (h *HealthCheckHandler) GetServiceChecks(c *fiber.Ctx) error {
	serviceName := c.Params("name")
	log := middleware.GetLogger(c)

	log.Debug("Getting health checks for service", logger.String("service", serviceName))

	checks := h.serviceStore.GetHealthChecks(serviceName)

	log.Info("Service health checks retrieved",
		logger.String("service", serviceName),
		logger.Int("count", len(checks)))

	return c.JSON(checks)
}

func (h *HealthCheckHandler) UpdateTTLCheck(c *fiber.Ctx) error {
	checkID := c.Params("id")
	log := middleware.GetLogger(c)

	log.Debug("Updating TTL check", logger.String("check_id", checkID))

	var err error
	if h.raftNode != nil {
		cmd, marshalErr := konsulraft.NewCommand(konsulraft.CmdHealthTTLUpdate, konsulraft.HealthTTLUpdatePayload{
			CheckID: checkID,
		})
		if marshalErr != nil {
			log.Error("Failed to build raft log entry", logger.Error(marshalErr))
			return middleware.InternalError(c, "Failed to update TTL check")
		}
		if _, applyErr := h.raftNode.ApplyEntry(cmd, 5*time.Second); applyErr != nil {
			if errors.Is(applyErr, hashiraft.ErrNotLeader) {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":  "not leader",
					"leader": h.raftNode.Leader(),
				})
			}
			err = applyErr
		}
	} else {
		err = h.serviceStore.UpdateTTLCheck(checkID)
	}
	if err != nil {
		log.Error("Failed to update TTL check",
			logger.String("check_id", checkID),
			logger.Error(err))
		return middleware.BadRequest(c, err.Error())
	}

	log.Info("TTL check updated successfully", logger.String("check_id", checkID))
	return c.JSON(fiber.Map{"message": "TTL check updated", "check_id": checkID})
}

func (h *HealthCheckHandler) AddCheck(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var checkDef healthcheck.CheckDefinition
	if err := c.BodyParser(&checkDef); err != nil {
		log.Error("Failed to parse health check definition", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	log.Debug("Adding health check",
		logger.String("name", checkDef.Name),
		logger.String("service", checkDef.ServiceID))

	// We need access to the health manager through the service store
	// For now, we'll return an error since we don't have direct access
	return middleware.BadRequest(c, "Direct health check addition not yet implemented")
}
