package handlers

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/agent"
	"github.com/neogan74/konsul/internal/logger"
	"github.com/neogan74/konsul/internal/store"
)

// AgentRegistry manages connected agents
type AgentRegistry struct {
	agents map[string]*RegisteredAgent
	mu     sync.RWMutex
	log    logger.Logger
}

// RegisteredAgent represents a registered agent
type RegisteredAgent struct {
	Info          agent.AgentInfo
	LastSeen      time.Time
	LastSyncIndex int64
	RegisteredAt  time.Time
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry(log logger.Logger) *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*RegisteredAgent),
		log:    log,
	}
}

// RegisterAgent registers a new agent
func (r *AgentRegistry) RegisterAgent(info agent.AgentInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[info.ID] = &RegisteredAgent{
		Info:         info,
		LastSeen:     time.Now(),
		RegisteredAt: time.Now(),
	}

	r.log.Info("Agent registered",
		logger.String("agent_id", info.ID),
		logger.String("node", info.NodeName),
		logger.String("datacenter", info.Datacenter))

	return nil
}

// UpdateLastSeen updates the last seen time for an agent
func (r *AgentRegistry) UpdateLastSeen(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent, ok := r.agents[agentID]; ok {
		agent.LastSeen = time.Now()
	}
}

// UpdateSyncIndex updates the sync index for an agent
func (r *AgentRegistry) UpdateSyncIndex(agentID string, index int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent, ok := r.agents[agentID]; ok {
		agent.LastSyncIndex = index
	}
}

// GetAgent retrieves an agent by ID
func (r *AgentRegistry) GetAgent(agentID string) (*RegisteredAgent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[agentID]
	return agent, ok
}

// ListAgents returns all registered agents
func (r *AgentRegistry) ListAgents() []*RegisteredAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]*RegisteredAgent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}

	return agents
}

// RemoveAgent removes an agent from the registry
func (r *AgentRegistry) RemoveAgent(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.agents, agentID)
	r.log.Info("Agent removed", logger.String("agent_id", agentID))
}

// CleanupStaleAgents removes agents that haven't been seen recently
func (r *AgentRegistry) CleanupStaleAgents(timeout time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	staleAgents := make([]string, 0)
	now := time.Now()

	for id, agent := range r.agents {
		if now.Sub(agent.LastSeen) > timeout {
			staleAgents = append(staleAgents, id)
		}
	}

	for _, id := range staleAgents {
		delete(r.agents, id)
		r.log.Warn("Removed stale agent", logger.String("agent_id", id))
	}

	return len(staleAgents)
}

// AgentHandlers contains all agent-related HTTP handlers
type AgentHandlers struct {
	registry     *AgentRegistry
	serviceStore *store.ServiceStore
	kvStore      *store.KVStore
	log          logger.Logger
	globalIndex  int64
	indexMu      sync.Mutex
}

// NewAgentHandlers creates new agent handlers
func NewAgentHandlers(serviceStore *store.ServiceStore, kvStore *store.KVStore, log logger.Logger) *AgentHandlers {
	return &AgentHandlers{
		registry:     NewAgentRegistry(log),
		serviceStore: serviceStore,
		kvStore:      kvStore,
		log:          log,
		globalIndex:  0,
	}
}

// HandleAgentRegister handles agent registration
func (h *AgentHandlers) HandleAgentRegister(c *fiber.Ctx) error {
	var info agent.AgentInfo
	if err := c.BodyParser(&info); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid agent info",
		})
	}

	if info.ID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "agent ID is required",
		})
	}

	if err := h.registry.RegisterAgent(info); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":   "registered",
		"agent_id": info.ID,
	})
}

// HandleAgentSync handles agent sync requests
func (h *AgentHandlers) HandleAgentSync(c *fiber.Ctx) error {
	var req agent.SyncRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid sync request",
		})
	}

	// Update last seen
	h.registry.UpdateLastSeen(req.AgentID)

	// Build sync response with delta updates
	resp := agent.SyncResponse{
		CurrentIndex:   h.getCurrentIndex(),
		ServiceUpdates: []agent.ServiceUpdate{},
		KVUpdates:      []agent.KVUpdate{},
		HealthUpdates:  []agent.HealthUpdate{},
	}

	// For full sync or if last index is 0, return all data
	if req.FullSync || req.LastSyncIndex == 0 {
		resp.ServiceUpdates = h.getAllServiceUpdates()
		resp.KVUpdates = h.getAllKVUpdates(req.WatchedPrefixes)
	} else {
		// For delta sync, return only changes since last index
		// In a real implementation, you'd track changes and return only deltas
		// For now, we'll return all data
		resp.ServiceUpdates = h.getAllServiceUpdates()
		resp.KVUpdates = h.getAllKVUpdates(req.WatchedPrefixes)
	}

	// Update agent's sync index
	h.registry.UpdateSyncIndex(req.AgentID, resp.CurrentIndex)

	return c.JSON(resp)
}

// HandleBatchUpdate handles batch service updates from agents
func (h *AgentHandlers) HandleBatchUpdate(c *fiber.Ctx) error {
	var req struct {
		AgentID string                `json:"agent_id"`
		Updates []agent.ServiceUpdate `json:"updates"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid batch update request",
		})
	}

	// Update last seen
	h.registry.UpdateLastSeen(req.AgentID)

	// Process each update
	for _, update := range req.Updates {
		switch update.Type {
		case agent.UpdateTypeAdd, agent.UpdateTypeUpdate:
			if update.Service != nil {
				// Register service in the service store
				if err := h.serviceStore.Register(*update.Service); err != nil {
					h.log.Warn("Failed to register service from agent",
						logger.String("agent_id", req.AgentID),
						logger.String("service", update.ServiceName),
						logger.Error(err))
				}
			}

		case agent.UpdateTypeDelete:
			// Deregister service
			h.serviceStore.Deregister(update.ServiceName)
		}
	}

	h.log.Info("Processed batch update from agent",
		logger.String("agent_id", req.AgentID),
		logger.Int("updates", len(req.Updates)))

	return c.JSON(fiber.Map{
		"status": "processed",
		"count":  len(req.Updates),
	})
}

// HandleHealthUpdate handles health check status updates from agents
func (h *AgentHandlers) HandleHealthUpdate(c *fiber.Ctx) error {
	var update agent.HealthUpdate
	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid health update",
		})
	}

	// Extract agent ID from header
	agentID := c.Get("X-Agent-ID")
	if agentID != "" {
		h.registry.UpdateLastSeen(agentID)
	}

	h.log.Info("Health check status received",
		logger.String("agent_id", agentID),
		logger.String("service_id", update.ServiceID),
		logger.String("check_id", update.CheckID),
		logger.String("status", string(update.Status)))

	// In a real implementation, you'd update health check state in the service store
	// For now, we just log it

	return c.JSON(fiber.Map{
		"status": "received",
	})
}

// HandleListAgents lists all registered agents
func (h *AgentHandlers) HandleListAgents(c *fiber.Ctx) error {
	agents := h.registry.ListAgents()
	return c.JSON(agents)
}

// HandleGetAgent retrieves a specific agent
func (h *AgentHandlers) HandleGetAgent(c *fiber.Ctx) error {
	agentID := c.Params("id")
	if agentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "agent ID is required",
		})
	}

	agent, ok := h.registry.GetAgent(agentID)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "agent not found",
		})
	}

	return c.JSON(agent)
}

// Helper methods

func (h *AgentHandlers) getCurrentIndex() int64 {
	h.indexMu.Lock()
	defer h.indexMu.Unlock()

	h.globalIndex++
	return h.globalIndex
}

func (h *AgentHandlers) getAllServiceUpdates() []agent.ServiceUpdate {
	h.serviceStore.Mutex.RLock()
	defer h.serviceStore.Mutex.RUnlock()

	updates := make([]agent.ServiceUpdate, 0)
	for name, entry := range h.serviceStore.Data {
		updates = append(updates, agent.ServiceUpdate{
			Type:        agent.UpdateTypeAdd,
			ServiceName: name,
			Service:     &entry.Service,
			Entry:       &entry,
		})
	}

	return updates
}

func (h *AgentHandlers) getAllKVUpdates(prefixes []string) []agent.KVUpdate {
	h.kvStore.Mutex.RLock()
	defer h.kvStore.Mutex.RUnlock()

	updates := make([]agent.KVUpdate, 0)

	// If no prefixes specified, return all
	if len(prefixes) == 0 {
		for key, entry := range h.kvStore.Data {
			updates = append(updates, agent.KVUpdate{
				Type:  agent.UpdateTypeAdd,
				Key:   key,
				Entry: &entry,
			})
		}
		return updates
	}

	// Return only keys matching watched prefixes
	for key, entry := range h.kvStore.Data {
		for _, prefix := range prefixes {
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				updates = append(updates, agent.KVUpdate{
					Type:  agent.UpdateTypeAdd,
					Key:   key,
					Entry: &entry,
				})
				break
			}
		}
	}

	return updates
}

// CleanupStaleAgents periodically removes stale agents
func (h *AgentHandlers) CleanupStaleAgents(timeout time.Duration) {
	count := h.registry.CleanupStaleAgents(timeout)
	if count > 0 {
		h.log.Info("Cleaned up stale agents", logger.Int("count", count))
	}
}

// GetRegistry returns the agent registry
func (h *AgentHandlers) GetRegistry() *AgentRegistry {
	return h.registry
}

// MarshalJSON customizes JSON marshaling for RegisteredAgent
func (ra *RegisteredAgent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Info          agent.AgentInfo `json:"info"`
		LastSeen      time.Time       `json:"last_seen"`
		LastSyncIndex int64           `json:"last_sync_index"`
		RegisteredAt  time.Time       `json:"registered_at"`
		Uptime        string          `json:"uptime"`
	}{
		Info:          ra.Info,
		LastSeen:      ra.LastSeen,
		LastSyncIndex: ra.LastSyncIndex,
		RegisteredAt:  ra.RegisteredAt,
		Uptime:        fmt.Sprintf("%s", time.Since(ra.RegisteredAt)),
	})
}
