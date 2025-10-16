package acl

import (
	"sync"
	"time"

	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/metrics"
)

// Evaluator evaluates ACL policies for authorization
type Evaluator struct {
	policies map[string]*Policy
	mu       sync.RWMutex
	log      logger.Logger
}

// NewEvaluator creates a new ACL evaluator
func NewEvaluator(log logger.Logger) *Evaluator {
	return &Evaluator{
		policies: make(map[string]*Policy),
		log:      log,
	}
}

// AddPolicy adds a policy to the evaluator
func (e *Evaluator) AddPolicy(policy *Policy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.policies[policy.Name]; exists {
		return ErrPolicyExists
	}

	e.policies[policy.Name] = policy
	e.log.Info("ACL policy added", logger.String("policy", policy.Name))
	return nil
}

// UpdatePolicy updates an existing policy
func (e *Evaluator) UpdatePolicy(policy *Policy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.policies[policy.Name]; !exists {
		return ErrPolicyNotFound
	}

	e.policies[policy.Name] = policy
	e.log.Info("ACL policy updated", logger.String("policy", policy.Name))
	return nil
}

// DeletePolicy removes a policy
func (e *Evaluator) DeletePolicy(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.policies[name]; !exists {
		return ErrPolicyNotFound
	}

	delete(e.policies, name)
	e.log.Info("ACL policy deleted", logger.String("policy", name))
	return nil
}

// GetPolicy retrieves a policy by name
func (e *Evaluator) GetPolicy(name string) (*Policy, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, exists := e.policies[name]
	if !exists {
		return nil, ErrPolicyNotFound
	}

	return policy, nil
}

// ListPolicies returns all policy names
func (e *Evaluator) ListPolicies() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.policies))
	for name := range e.policies {
		names = append(names, name)
	}
	return names
}

// Evaluate checks if the given policies allow the specified capability on a resource
// Returns true if access is allowed, false if denied
func (e *Evaluator) Evaluate(policyNames []string, resource Resource, capability Capability) bool {
	start := time.Now()
	e.mu.RLock()
	defer e.mu.RUnlock()

	var allowed bool
	defer func() {
		// Record metrics
		duration := time.Since(start).Seconds()
		result := "deny"
		if allowed {
			result = "allow"
		}
		metrics.ACLEvaluationsTotal.WithLabelValues(string(resource.Type), string(capability), result).Inc()
		metrics.ACLEvaluationDuration.WithLabelValues(string(resource.Type)).Observe(duration)
	}()

	if len(policyNames) == 0 {
		e.log.Debug("No policies provided - denying by default")
		return false
	}

	// Collect all applicable policies
	policies := make([]*Policy, 0, len(policyNames))
	for _, name := range policyNames {
		if policy, exists := e.policies[name]; exists {
			policies = append(policies, policy)
		} else {
			e.log.Warn("Policy not found", logger.String("policy", name))
		}
	}

	if len(policies) == 0 {
		e.log.Debug("No valid policies found - denying by default")
		return false
	}

	// Evaluate policies based on resource type
	switch resource.Type {
	case ResourceTypeKV:
		allowed = e.evaluateKV(policies, resource.Path, capability)
	case ResourceTypeService:
		allowed = e.evaluateService(policies, resource.Path, capability)
	case ResourceTypeHealth:
		allowed = e.evaluateHealth(policies, capability)
	case ResourceTypeBackup:
		allowed = e.evaluateBackup(policies, capability)
	case ResourceTypeAdmin:
		allowed = e.evaluateAdmin(policies, capability)
	default:
		e.log.Warn("Unknown resource type", logger.String("type", string(resource.Type)))
		allowed = false
	}

	return allowed
}

// evaluateKV evaluates KV rules
func (e *Evaluator) evaluateKV(policies []*Policy, kvPath string, capability Capability) bool {
	e.log.Debug("Evaluating KV access",
		logger.String("path", kvPath),
		logger.String("capability", string(capability)),
		logger.Int("policies", len(policies)))

	for _, policy := range policies {
		for _, rule := range policy.KV {
			if rule.Matches(kvPath) {
				// Check for explicit deny
				if rule.HasCapability(CapabilityDeny) {
					e.log.Debug("Explicit deny found",
						logger.String("policy", policy.Name),
						logger.String("path", kvPath),
						logger.String("rule_path", rule.Path))
					return false
				}

				// Check if capability is allowed
				if rule.HasCapability(capability) {
					e.log.Debug("Access granted",
						logger.String("policy", policy.Name),
						logger.String("path", kvPath),
						logger.String("capability", string(capability)))
					return true
				}
			}
		}
	}

	e.log.Debug("No matching rule found - denying by default",
		logger.String("path", kvPath),
		logger.String("capability", string(capability)))
	return false
}

// evaluateService evaluates service rules
func (e *Evaluator) evaluateService(policies []*Policy, serviceName string, capability Capability) bool {
	e.log.Debug("Evaluating service access",
		logger.String("service", serviceName),
		logger.String("capability", string(capability)),
		logger.Int("policies", len(policies)))

	for _, policy := range policies {
		for _, rule := range policy.Service {
			if rule.Matches(serviceName) {
				// Check for explicit deny
				if rule.HasCapability(CapabilityDeny) {
					e.log.Debug("Explicit deny found",
						logger.String("policy", policy.Name),
						logger.String("service", serviceName))
					return false
				}

				// Check if capability is allowed
				if rule.HasCapability(capability) {
					e.log.Debug("Access granted",
						logger.String("policy", policy.Name),
						logger.String("service", serviceName),
						logger.String("capability", string(capability)))
					return true
				}
			}
		}
	}

	e.log.Debug("No matching rule found - denying by default",
		logger.String("service", serviceName),
		logger.String("capability", string(capability)))
	return false
}

// evaluateHealth evaluates health rules
func (e *Evaluator) evaluateHealth(policies []*Policy, capability Capability) bool {
	e.log.Debug("Evaluating health access",
		logger.String("capability", string(capability)),
		logger.Int("policies", len(policies)))

	for _, policy := range policies {
		for _, rule := range policy.Health {
			// Check for explicit deny
			if rule.HasCapability(CapabilityDeny) {
				e.log.Debug("Explicit deny found", logger.String("policy", policy.Name))
				return false
			}

			// Check if capability is allowed
			if rule.HasCapability(capability) {
				e.log.Debug("Access granted",
					logger.String("policy", policy.Name),
					logger.String("capability", string(capability)))
				return true
			}
		}
	}

	e.log.Debug("No matching rule found - denying by default")
	return false
}

// evaluateBackup evaluates backup rules
func (e *Evaluator) evaluateBackup(policies []*Policy, capability Capability) bool {
	e.log.Debug("Evaluating backup access",
		logger.String("capability", string(capability)),
		logger.Int("policies", len(policies)))

	for _, policy := range policies {
		for _, rule := range policy.Backup {
			// Check for explicit deny
			if rule.HasCapability(CapabilityDeny) {
				e.log.Debug("Explicit deny found", logger.String("policy", policy.Name))
				return false
			}

			// Check if capability is allowed
			if rule.HasCapability(capability) {
				e.log.Debug("Access granted",
					logger.String("policy", policy.Name),
					logger.String("capability", string(capability)))
				return true
			}
		}
	}

	e.log.Debug("No matching rule found - denying by default")
	return false
}

// evaluateAdmin evaluates admin rules
func (e *Evaluator) evaluateAdmin(policies []*Policy, capability Capability) bool {
	e.log.Debug("Evaluating admin access",
		logger.String("capability", string(capability)),
		logger.Int("policies", len(policies)))

	for _, policy := range policies {
		for _, rule := range policy.Admin {
			// Check for explicit deny
			if rule.HasCapability(CapabilityDeny) {
				e.log.Debug("Explicit deny found", logger.String("policy", policy.Name))
				return false
			}

			// Check if capability is allowed
			if rule.HasCapability(capability) {
				e.log.Debug("Access granted",
					logger.String("policy", policy.Name),
					logger.String("capability", string(capability)))
				return true
			}
		}
	}

	e.log.Debug("No matching rule found - denying by default")
	return false
}

// LoadPolicies loads multiple policies at once
func (e *Evaluator) LoadPolicies(policies []*Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, policy := range policies {
		if err := policy.Validate(); err != nil {
			e.log.Error("Invalid policy",
				logger.String("policy", policy.Name),
				logger.Error(err))
			return err
		}
		e.policies[policy.Name] = policy
	}

	e.log.Info("ACL policies loaded", logger.Int("count", len(policies)))
	return nil
}

// Count returns the number of policies
func (e *Evaluator) Count() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.policies)
}
