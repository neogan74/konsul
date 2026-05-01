package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/neogan74/konsul/internal/auth"
	konsulraft "github.com/neogan74/konsul/internal/raft"
)

// ClusterHandler handles cluster management endpoints.
type ClusterHandler struct {
	raftNode   *konsulraft.Node
	autopilot  *konsulraft.Autopilot
	jwtService *auth.JWTService // optional; enables join token generation and verification
}

// NewClusterHandler creates a new cluster handler.
// If raftNode is nil, cluster endpoints will return 503 Service Unavailable.
func NewClusterHandler(raftNode *konsulraft.Node) *ClusterHandler {
	return &ClusterHandler{
		raftNode: raftNode,
	}
}

// SetAutopilot attaches an autopilot instance so its health can be reported.
func (h *ClusterHandler) SetAutopilot(ap *konsulraft.Autopilot) {
	h.autopilot = ap
}

// SetJWTService attaches a JWT service for join token generation and verification.
func (h *ClusterHandler) SetJWTService(svc *auth.JWTService) {
	h.jwtService = svc
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
	cluster.Get("/autopilot", h.AutopilotHealth)
	cluster.Post("/transfer", h.TransferLeadership)
	cluster.Post("/token", h.GenerateJoinToken)
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
// When a JWT service is configured, the request may carry an X-Join-Token header
// with a token obtained from POST /cluster/token. Invalid tokens are rejected.
func (h *ClusterHandler) Join(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	// Verify join token when one is presented and the JWT service is available.
	if h.jwtService != nil {
		if tokenStr := c.Get("X-Join-Token"); tokenStr != "" {
			if err := h.jwtService.ValidateJoinToken(tokenStr); err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "invalid join token: " + err.Error(),
				})
			}
		}
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

// AutopilotHealth returns the current autopilot health report.
// GET /cluster/autopilot
func (h *ClusterHandler) AutopilotHealth(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	if h.autopilot == nil {
		return c.JSON(fiber.Map{
			"enabled": false,
			"message": "Autopilot is not configured on this node.",
		})
	}

	return c.JSON(fiber.Map{
		"enabled": true,
		"servers": h.autopilot.HealthReport(),
	})
}

// TransferLeadershipRequest is the body for POST /cluster/transfer.
type TransferLeadershipRequest struct {
	ToNodeID string `json:"to_node_id"` // optional; empty means best follower
	ToAddr   string `json:"to_addr"`    // optional; required when to_node_id is set
}

// TransferLeadership transfers Raft leadership to another node.
// POST /cluster/transfer
// Body: {"to_node_id": "node3", "to_addr": "10.0.0.3:7000"}  (both optional)
func (h *ClusterHandler) TransferLeadership(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	var req TransferLeadershipRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if err := h.raftNode.TransferLeadership(req.ToNodeID, req.ToAddr); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	msg := "leadership transfer initiated"
	if req.ToNodeID != "" {
		msg = "leadership transfer initiated to " + req.ToNodeID
	}
	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": msg,
	})
}

// GenerateJoinTokenRequest is the body for POST /cluster/token.
type GenerateJoinTokenRequest struct {
	TTL string `json:"ttl"` // e.g. "24h", "30m"; defaults to "1h"
}

// GenerateJoinToken issues a short-lived JWT that authorises a new node to join.
// POST /cluster/token
// Body: {"ttl": "24h"}
func (h *ClusterHandler) GenerateJoinToken(c *fiber.Ctx) error {
	if !h.checkRaftEnabled(c) {
		return nil
	}

	if h.jwtService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "auth not configured",
			"message": "Set KONSUL_JWT_SECRET to enable join token generation.",
		})
	}

	var req GenerateJoinTokenRequest
	// BodyParser failure is non-fatal; we fall back to the default TTL.
	_ = c.BodyParser(&req)

	ttl := time.Hour
	if req.TTL != "" {
		d, err := time.ParseDuration(req.TTL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid ttl: " + err.Error(),
			})
		}
		ttl = d
	}

	token, err := h.jwtService.GenerateJoinToken(ttl)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"token":      token,
		"expires_in": ttl.String(),
	})
}
