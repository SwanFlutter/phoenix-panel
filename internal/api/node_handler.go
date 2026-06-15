package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/phoenix-panel/phoenix/internal/audit"
	"github.com/phoenix-panel/phoenix/internal/middleware"
	"github.com/phoenix-panel/phoenix/internal/models"
	"github.com/phoenix-panel/phoenix/internal/service"
)

// NodeHandler serves admin-facing node and inbound management.
type NodeHandler struct {
	nodes *service.NodeService
	audit *audit.Logger
}

// NewNodeHandler constructs a NodeHandler.
func NewNodeHandler(nodes *service.NodeService, auditLog *audit.Logger) *NodeHandler {
	return &NodeHandler{nodes: nodes, audit: auditLog}
}

type createNodeRequest struct {
	Name    string          `json:"name" binding:"required"`
	Address string          `json:"address" binding:"required"`
	APIHost string          `json:"api_host"`
	APIPort int             `json:"api_port"`
	Core    models.CoreType `json:"core" binding:"required"`
}

type createInboundRequest struct {
	NodeID      uint             `json:"node_id" binding:"required"`
	Tag         string           `json:"tag" binding:"required"`
	Protocol    models.Protocol  `json:"protocol" binding:"required"`
	Listen      string           `json:"listen"`
	Port        int              `json:"port" binding:"required"`
	Network     string           `json:"network"`
	Security    string           `json:"security"`
	SNI         string           `json:"sni"`
	Host        string           `json:"host"`
	Path        string           `json:"path"`
	Flow        string           `json:"flow"`
	Fingerprint string           `json:"fingerprint"`
	Reality     models.JSONMap   `json:"reality_settings"`
	Extra       models.JSONMap   `json:"extra"`
}

// ListNodes handles GET /api/admin/nodes.
func (h *NodeHandler) ListNodes(c *gin.Context) {
	nodes, err := h.nodes.ListNodes(c.Request.Context())
	if err != nil {
		failErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": nodes})
}

// GetNode handles GET /api/admin/nodes/:id.
func (h *NodeHandler) GetNode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.nodes.GetNode(c.Request.Context(), id)
	if err != nil {
		failErr(c, err)
		return
	}
	c.JSON(http.StatusOK, node)
}

// CreateNode handles POST /api/admin/nodes.
func (h *NodeHandler) CreateNode(c *gin.Context) {
	var req createNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	node := &models.Node{
		Name:     req.Name,
		Address:  req.Address,
		APIHost:  req.APIHost,
		APIPort:  req.APIPort,
		Core:     req.Core,
		Status:   models.NodeUnknown,
		IsActive: true,
	}
	if err := h.nodes.CreateNode(c.Request.Context(), node); err != nil {
		failErr(c, err)
		return
	}
	h.audit.FromAdmin(c, middleware.CurrentAdminID(c), middleware.CurrentUsername(c),
		audit.ActionNodeCreate, "node:"+node.Name, true, nil)
	c.JSON(http.StatusCreated, node)
}

// DeleteNode handles DELETE /api/admin/nodes/:id.
func (h *NodeHandler) DeleteNode(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.nodes.DeleteNode(c.Request.Context(), id); err != nil {
		failErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateInbound handles POST /api/admin/inbounds.
func (h *NodeHandler) CreateInbound(c *gin.Context) {
	var req createInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	in := &models.Inbound{
		NodeID:          req.NodeID,
		Tag:             req.Tag,
		Protocol:        req.Protocol,
		Listen:          req.Listen,
		Port:            req.Port,
		Network:         req.Network,
		Security:        req.Security,
		SNI:             req.SNI,
		Host:            req.Host,
		Path:            req.Path,
		Flow:            req.Flow,
		Fingerprint:     req.Fingerprint,
		RealitySettings: req.Reality,
		Extra:           req.Extra,
		IsActive:        true,
	}
	if err := h.nodes.CreateInbound(c.Request.Context(), in); err != nil {
		failErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, in)
}

// DeleteInbound handles DELETE /api/admin/inbounds/:id.
func (h *NodeHandler) DeleteInbound(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.nodes.DeleteInbound(c.Request.Context(), id); err != nil {
		failErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
