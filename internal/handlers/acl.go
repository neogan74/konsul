package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/acl"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/middleware"
)

// ACLHandler handles ACL policy management
type ACLHandler struct {
	evaluator *acl.Evaluator
	policyDir string
	log       logger.Logger
}

// NewACLHandler creates a new ACL handler
func NewACLHandler(evaluator *acl.Evaluator, policyDir string, log logger.Logger) *ACLHandler {
	return &ACLHandler{
		evaluator: evaluator,
		policyDir: policyDir,
		log:       log,
	}
}

// CreatePolicy creates a new ACL policy
func (h *ACLHandler) CreatePolicy(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var policy acl.Policy
	if err := c.BodyParser(&policy); err != nil {
		log.Error("Failed to parse policy", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	// Validate policy
	if err := policy.Validate(); err != nil {
		log.Error("Invalid policy", logger.String("policy", policy.Name), logger.Error(err))
		return middleware.BadRequest(c, "Invalid policy: "+err.Error())
	}

	// Add policy to evaluator
	if err := h.evaluator.AddPolicy(&policy); err != nil {
		if err == acl.ErrPolicyExists {
			return middleware.Conflict(c, "Policy already exists")
		}
		log.Error("Failed to add policy", logger.String("policy", policy.Name), logger.Error(err))
		return middleware.InternalError(c, "Failed to add policy")
	}

	// Save policy to disk if policy directory is configured
	if h.policyDir != "" {
		if err := h.savePolicyToFile(&policy); err != nil {
			log.Error("Failed to save policy to file", logger.String("policy", policy.Name), logger.Error(err))
			// Continue - policy is in memory
		}
	}

	log.Info("ACL policy created", logger.String("policy", policy.Name))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "policy created",
		"policy":  policy,
	})
}

// GetPolicy retrieves a policy by name
func (h *ACLHandler) GetPolicy(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	policy, err := h.evaluator.GetPolicy(name)
	if err != nil {
		if err == acl.ErrPolicyNotFound {
			return middleware.NotFound(c, "Policy not found")
		}
		log.Error("Failed to get policy", logger.String("policy", name), logger.Error(err))
		return middleware.InternalError(c, "Failed to get policy")
	}

	return c.JSON(policy)
}

// ListPolicies lists all policies
func (h *ACLHandler) ListPolicies(c *fiber.Ctx) error {
	names := h.evaluator.ListPolicies()
	return c.JSON(fiber.Map{
		"policies": names,
		"count":    len(names),
	})
}

// UpdatePolicy updates an existing policy
func (h *ACLHandler) UpdatePolicy(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	var policy acl.Policy
	if err := c.BodyParser(&policy); err != nil {
		log.Error("Failed to parse policy", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	// Ensure name matches URL parameter
	if policy.Name != name {
		return middleware.BadRequest(c, "Policy name in body must match URL parameter")
	}

	// Validate policy
	if err := policy.Validate(); err != nil {
		log.Error("Invalid policy", logger.String("policy", policy.Name), logger.Error(err))
		return middleware.BadRequest(c, "Invalid policy: "+err.Error())
	}

	// Update policy in evaluator
	if err := h.evaluator.UpdatePolicy(&policy); err != nil {
		if err == acl.ErrPolicyNotFound {
			return middleware.NotFound(c, "Policy not found")
		}
		log.Error("Failed to update policy", logger.String("policy", policy.Name), logger.Error(err))
		return middleware.InternalError(c, "Failed to update policy")
	}

	// Save policy to disk if policy directory is configured
	if h.policyDir != "" {
		if err := h.savePolicyToFile(&policy); err != nil {
			log.Error("Failed to save policy to file", logger.String("policy", policy.Name), logger.Error(err))
		}
	}

	log.Info("ACL policy updated", logger.String("policy", policy.Name))
	return c.JSON(fiber.Map{
		"message": "policy updated",
		"policy":  policy,
	})
}

// DeletePolicy deletes a policy
func (h *ACLHandler) DeletePolicy(c *fiber.Ctx) error {
	name := c.Params("name")
	log := middleware.GetLogger(c)

	if err := h.evaluator.DeletePolicy(name); err != nil {
		if err == acl.ErrPolicyNotFound {
			return middleware.NotFound(c, "Policy not found")
		}
		log.Error("Failed to delete policy", logger.String("policy", name), logger.Error(err))
		return middleware.InternalError(c, "Failed to delete policy")
	}

	// Remove policy file if it exists
	if h.policyDir != "" {
		policyFile := filepath.Join(h.policyDir, name+".json")
		if err := os.Remove(policyFile); err != nil && !os.IsNotExist(err) {
			log.Error("Failed to delete policy file", logger.String("file", policyFile), logger.Error(err))
		}
	}

	log.Info("ACL policy deleted", logger.String("policy", name))
	return c.JSON(fiber.Map{
		"message": "policy deleted",
		"policy":  name,
	})
}

// TestPolicy tests if a policy would allow a specific operation
func (h *ACLHandler) TestPolicy(c *fiber.Ctx) error {
	log := middleware.GetLogger(c)

	var req struct {
		Policies   []string `json:"policies"`
		Resource   string   `json:"resource"`
		Path       string   `json:"path"`
		Capability string   `json:"capability"`
	}

	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse test request", logger.Error(err))
		return middleware.BadRequest(c, "Invalid JSON body")
	}

	// Build resource
	var resource acl.Resource
	switch req.Resource {
	case "kv":
		resource = acl.NewKVResource(req.Path)
	case "service":
		resource = acl.NewServiceResource(req.Path)
	case "health":
		resource = acl.NewHealthResource()
	case "backup":
		resource = acl.NewBackupResource()
	case "admin":
		resource = acl.NewAdminResource()
	default:
		return middleware.BadRequest(c, "Invalid resource type")
	}

	// Evaluate
	allowed := h.evaluator.Evaluate(req.Policies, resource, acl.Capability(req.Capability))

	return c.JSON(fiber.Map{
		"allowed":    allowed,
		"policies":   req.Policies,
		"resource":   req.Resource,
		"path":       req.Path,
		"capability": req.Capability,
	})
}

// LoadPolicies loads all policies from the policy directory
func (h *ACLHandler) LoadPolicies() error {
	if h.policyDir == "" {
		return nil
	}

	// Create policy directory if it doesn't exist
	if err := os.MkdirAll(h.policyDir, 0755); err != nil {
		return err
	}

	// Read all JSON files in the policy directory
	files, err := filepath.Glob(filepath.Join(h.policyDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			h.log.Error("Failed to read policy file", logger.String("file", file), logger.Error(err))
			continue
		}

		policy, err := acl.FromJSON(data)
		if err != nil {
			h.log.Error("Failed to parse policy file", logger.String("file", file), logger.Error(err))
			continue
		}

		if err := h.evaluator.AddPolicy(policy); err != nil && err != acl.ErrPolicyExists {
			h.log.Error("Failed to add policy", logger.String("policy", policy.Name), logger.Error(err))
			continue
		}

		h.log.Info("Loaded ACL policy from file", logger.String("policy", policy.Name), logger.String("file", file))
	}

	return nil
}

// savePolicyToFile saves a policy to a JSON file
func (h *ACLHandler) savePolicyToFile(policy *acl.Policy) error {
	// Create policy directory if it doesn't exist
	if err := os.MkdirAll(h.policyDir, 0755); err != nil {
		return err
	}

	policyFile := filepath.Join(h.policyDir, policy.Name+".json")
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(policyFile, data, 0644)
}
