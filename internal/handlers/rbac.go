package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/middleware"
	"github.com/neogan74/konsul/internal/rbac"
)

// RBACHandler handles RBAC role and assignment management.
type RBACHandler struct {
	manager rbac.RoleManager
	log     logger.Logger
}

// NewRBACHandler creates a new RBACHandler.
func NewRBACHandler(manager rbac.RoleManager, log logger.Logger) *RBACHandler {
	return &RBACHandler{
		manager: manager,
		log:     log,
	}
}

// RegisterRoutes registers all RBAC routes on the fiber app.
func (h *RBACHandler) RegisterRoutes(app *fiber.App) {
	app.Post("/rbac/roles", h.CreateRole)
	app.Get("/rbac/roles", h.ListRoles)
	app.Get("/rbac/roles/:name", h.GetRole)
	app.Put("/rbac/roles/:name", h.UpdateRole)
	app.Delete("/rbac/roles/:name", h.DeleteRole)
	app.Post("/rbac/assignments", h.AssignRole)
	app.Delete("/rbac/assignments/:subject_id", h.UnassignRole)
	app.Get("/rbac/assignments", h.ListAssignments)
	app.Get("/rbac/assignments/:subject_id", h.GetAssignment)
	app.Get("/rbac/subjects/:subject_id/policies", h.GetEffectivePolicies)
}

// nsFromCtx extracts the namespace from Fiber locals, defaulting to "default".
// Phase 04 will install middleware that populates c.Locals("namespace").
func nsFromCtx(c *fiber.Ctx) string {
	ns, _ := c.Locals("namespace").(string)
	if ns == "" {
		return "default"
	}
	return ns
}

// CreateRole handles POST /rbac/roles.
func (h *RBACHandler) CreateRole(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	ns := nsFromCtx(c)

	var role rbac.Role
	if err := c.BodyParser(&role); err != nil {
		log.Debug("Failed to parse role request body", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if role.Name == "" {
		return middleware.BadRequest(c, "Role name is required")
	}

	if err := h.manager.CreateRole(ns, &role); err != nil {
		switch err {
		case rbac.ErrRoleExists:
			return middleware.Conflict(c, "Role already exists")
		case rbac.ErrCyclicDependency:
			return middleware.UnprocessableEntity(c, "Cyclic dependency detected in parent roles")
		case rbac.ErrMaxDepthExceeded:
			return middleware.UnprocessableEntity(c, "Max role inheritance depth exceeded")
		default:
			log.Error("Failed to create role", logger.String("role", role.Name), logger.Error(err))
			return middleware.InternalError(c, "Failed to create role")
		}
	}

	log.Info("RBAC role created", logger.String("role", role.Name))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "role created",
		"role":    role,
	})
}

// ListRoles handles GET /rbac/roles.
func (h *RBACHandler) ListRoles(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)
	ns := nsFromCtx(c)

	roles, err := h.manager.ListRoles(ns)
	if err != nil {
		log.Error("Failed to list roles", logger.Error(err))
		return middleware.InternalError(c, "Failed to list roles")
	}

	return c.JSON(fiber.Map{
		"roles": roles,
		"count": len(roles),
	})
}

// GetRole handles GET /rbac/roles/:name.
func (h *RBACHandler) GetRole(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)
	ns := nsFromCtx(c)

	role, err := h.manager.GetRole(ns, name)
	if err != nil {
		if err == rbac.ErrRoleNotFound {
			return middleware.NotFound(c, "Role not found")
		}
		log.Error("Failed to get role", logger.String("role", name), logger.Error(err))
		return middleware.InternalError(c, "Failed to get role")
	}

	return c.JSON(role)
}

// UpdateRole handles PUT /rbac/roles/:name.
func (h *RBACHandler) UpdateRole(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)
	ns := nsFromCtx(c)

	var role rbac.Role
	if err := c.BodyParser(&role); err != nil {
		log.Debug("Failed to parse role request body", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if role.Name != "" && role.Name != name {
		return middleware.BadRequest(c, "Role name in body must match URL parameter")
	}
	role.Name = name

	if err := h.manager.UpdateRole(ns, &role); err != nil {
		switch err {
		case rbac.ErrRoleNotFound:
			return middleware.NotFound(c, "Role not found")
		case rbac.ErrCyclicDependency:
			return middleware.UnprocessableEntity(c, "Cyclic dependency detected in parent roles")
		case rbac.ErrMaxDepthExceeded:
			return middleware.UnprocessableEntity(c, "Max role inheritance depth exceeded")
		default:
			log.Error("Failed to update role", logger.String("role", name), logger.Error(err))
			return middleware.InternalError(c, "Failed to update role")
		}
	}

	log.Info("RBAC role updated", logger.String("role", name))
	return c.JSON(fiber.Map{
		"message": "role updated",
		"role":    role,
	})
}

// DeleteRole handles DELETE /rbac/roles/:name.
func (h *RBACHandler) DeleteRole(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)
	ns := nsFromCtx(c)

	if err := h.manager.DeleteRole(ns, name); err != nil {
		if err == rbac.ErrRoleNotFound {
			return middleware.NotFound(c, "Role not found")
		}
		log.Error("Failed to delete role", logger.String("role", name), logger.Error(err))
		return middleware.InternalError(c, "Failed to delete role")
	}

	log.Info("RBAC role deleted", logger.String("role", name))
	return c.Status(fiber.StatusNoContent).SendString("")
}

// assignRoleRequest is the request body for AssignRole.
type assignRoleRequest struct {
	SubjectID string     `json:"subject_id"`
	RoleNames []string   `json:"role_names"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// AssignRole handles POST /rbac/assignments.
func (h *RBACHandler) AssignRole(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req assignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		log.Debug("Failed to parse assignment request body", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	if req.SubjectID == "" {
		return middleware.BadRequest(c, "subject_id is required")
	}
	if len(req.RoleNames) == 0 {
		return middleware.BadRequest(c, "role_names must not be empty")
	}

	ns := nsFromCtx(c)
	if err := h.manager.AssignRole(ns, req.SubjectID, req.RoleNames, req.ExpiresAt); err != nil {
		switch err {
		case rbac.ErrRoleNotFound:
			return middleware.NotFound(c, "Role not found")
		default:
			log.Error("Failed to assign roles", logger.String("subject_id", req.SubjectID), logger.Error(err))
			return middleware.InternalError(c, "Failed to assign roles")
		}
	}

	log.Info("RBAC roles assigned", logger.String("subject_id", req.SubjectID))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":    "roles assigned",
		"subject_id": req.SubjectID,
	})
}

// UnassignRole handles DELETE /rbac/assignments/:subject_id.
func (h *RBACHandler) UnassignRole(c *fiber.Ctx) error {
	subjectID := c.Params("subject_id")
	log := middleware.GetLogger(c)

	ns := nsFromCtx(c)
	if err := h.manager.UnassignRole(ns, subjectID); err != nil {
		if err == rbac.ErrAssignmentNotFound {
			return middleware.NotFound(c, "Assignment not found")
		}
		log.Error("Failed to unassign roles", logger.String("subject_id", subjectID), logger.Error(err))
		return middleware.InternalError(c, "Failed to unassign roles")
	}

	log.Info("RBAC roles unassigned", logger.String("subject_id", subjectID))
	return c.Status(fiber.StatusNoContent).SendString("")
}

// ListAssignments handles GET /rbac/assignments.
func (h *RBACHandler) ListAssignments(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	ns := nsFromCtx(c)
	assignments, err := h.manager.ListAssignments(ns)
	if err != nil {
		log.Error("Failed to list assignments", logger.Error(err))
		return middleware.InternalError(c, "Failed to list assignments")
	}

	return c.JSON(fiber.Map{
		"assignments": assignments,
		"count":       len(assignments),
	})
}

// GetAssignment handles GET /rbac/assignments/:subject_id.
func (h *RBACHandler) GetAssignment(c *fiber.Ctx) error {
	subjectID := c.Params("subject_id")
	log := middleware.GetLogger(c)

	ns := nsFromCtx(c)
	assignment, err := h.manager.GetAssignment(ns, subjectID)
	if err != nil {
		if err == rbac.ErrAssignmentNotFound {
			return middleware.NotFound(c, "Assignment not found")
		}
		log.Error("Failed to get assignment", logger.String("subject_id", subjectID), logger.Error(err))
		return middleware.InternalError(c, "Failed to get assignment")
	}

	return c.JSON(assignment)
}

// GetEffectivePolicies handles GET /rbac/subjects/:subject_id/policies.
func (h *RBACHandler) GetEffectivePolicies(c *fiber.Ctx) error {
	subjectID := c.Params("subject_id")
	log := middleware.GetLogger(c)

	ns := nsFromCtx(c)
	policies, err := h.manager.GetEffectivePolicies(ns, subjectID)
	if err != nil {
		if err == rbac.ErrAssignmentNotFound {
			return middleware.NotFound(c, "Assignment not found")
		}
		log.Error("Failed to get effective policies", logger.String("subject_id", subjectID), logger.Error(err))
		return middleware.InternalError(c, "Failed to get effective policies")
	}

	return c.JSON(fiber.Map{
		"subject_id": subjectID,
		"policies":   policies,
		"count":      len(policies),
	})
}
