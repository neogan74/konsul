package handlers

import (
	"github.com/gofiber/fiber/v2"
	konsulraft "github.com/neogan74/konsul/internal/raft"
)

// ClusterHandler handles cluster management endpoints.
type ClusterHandler struct {
	raftNode *konsulraft.Node
}

// NewClusterHandler creates a new cluster handler.
// If raftNode is nil, cluster endpoints will return 503 Service Unavailable.
func NewClusterHandler(raftNode *konsulraft.Node) *ClusterHandler {
	return &ClusterHandler{
		raftNode: raftNode,
	}
}

// RegisterRoutes registers cluster routes.
func (h *ClusterHandler) RegisterRoutes(app *fiber.App) {
	cluster := app.Group("/cluster")

	cluster.Get("/status", h.Status)
	cluster.Get("/leader", h.Leader)
	cluster.Get("/peers", h.Peers)
	cluster.Post("/join", h.Join)
	cluster.Delete("/leave/:id", h.Leave)
	cluster.Post("/snapshot", h.Snapshot)
}

// checkRaftEnabled returns error response if Raft is not enabled.
func (h *ClusterHandler) checkRaftEnabled(c *fiber.Ctx) bool {
	if h.raftNode == nil {
		_ = c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "clustering not enabled",
			"message": "This node is running in standalone mode. Start with --raft-enabled to enable clustering.",
		})
		return false
	}
	return true
}

// Status returns the current cluster status.
// GET /cluster/status
func (h *ClusterHandler) Status(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	info, err := h.raftNode.GetClusterInfo()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(info)
}

// Leader returns information about the current leader.
// GET /cluster/leader
func (h *ClusterHandler) Leader(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	leaderAddr := h.raftNode.LeaderAddr()
	leaderID := h.raftNode.LeaderID()

	if leaderAddr == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "no leader",
			"message": "No leader is currently elected. The cluster may be initializing or partitioned.",
		})
	}

	return c.JSON(fiber.Map{
		"leader_id":   leaderID,
		"leader_addr": leaderAddr,
		"is_self":     h.raftNode.IsLeader(),
	})
}

// Peers returns the list of cluster peers.
// GET /cluster/peers
func (h *ClusterHandler) Peers(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	info, err := h.raftNode.GetClusterInfo()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"peers": info.Peers,
		"count": len(info.Peers),
	})
}

// JoinRequest represents a request to join a node to the cluster.
type JoinRequest struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

// Join adds a new node to the cluster.
// POST /cluster/join
// Body: {"node_id": "node2", "address": "10.0.0.2:7000"}
func (h *ClusterHandler) Join(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	// Only leader can add nodes
	if !h.raftNode.IsLeader() {
		leaderAddr := h.raftNode.LeaderAddr()
		return c.Status(fiber.StatusTemporaryRedirect).JSON(fiber.Map{
			"error":       "not leader",
			"message":     "This node is not the leader. Redirect to leader.",
			"leader_addr": leaderAddr,
		})
	}

	var req JoinRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.NodeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "node_id is required",
		})
	}
	if req.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "address is required",
		})
	}

	if err := h.raftNode.Join(req.NodeID, req.Address); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Node joined successfully",
		"node_id": req.NodeID,
		"address": req.Address,
	})
}

// Leave removes a node from the cluster.
// DELETE /cluster/leave/:id
func (h *ClusterHandler) Leave(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	// Only leader can remove nodes
	if !h.raftNode.IsLeader() {
		leaderAddr := h.raftNode.LeaderAddr()
		return c.Status(fiber.StatusTemporaryRedirect).JSON(fiber.Map{
			"error":       "not leader",
			"message":     "This node is not the leader. Redirect to leader.",
			"leader_addr": leaderAddr,
		})
	}

	nodeID := c.Params("id")
	if nodeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "node id is required",
		})
	}

	if err := h.raftNode.Leave(nodeID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Node removed successfully",
		"node_id": nodeID,
	})
}

// Snapshot triggers a manual snapshot.
// POST /cluster/snapshot
func (h *ClusterHandler) Snapshot(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	if err := h.raftNode.Snapshot(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Snapshot created successfully",
	})
}
