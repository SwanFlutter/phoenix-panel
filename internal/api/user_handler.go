package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/phoenix-panel/phoenix/internal/audit"
	"github.com/phoenix-panel/phoenix/internal/config"
	"github.com/phoenix-panel/phoenix/internal/middleware"
	"github.com/phoenix-panel/phoenix/internal/models"
	"github.com/phoenix-panel/phoenix/internal/service"
)

// UserHandler serves admin-facing user management endpoints.
type UserHandler struct {
	users *service.UserService
	audit *audit.Logger
	cfg   *config.Config
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(users *service.UserService, auditLog *audit.Logger, cfg *config.Config) *UserHandler {
	return &UserHandler{users: users, audit: auditLog, cfg: cfg}
}

func (h *UserHandler) baseURL() string { return h.cfg.Server.BaseURL }

// Create handles POST /api/admin/users.
func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	user, err := h.users.Create(c.Request.Context(), service.CreateUserInput{
		Username:     req.Username,
		DataLimit:    req.DataLimit,
		ExpireAt:     req.ExpireAt,
		DataStrategy: req.DataStrategy,
		SSMethod:     req.SSMethod,
		Note:         req.Note,
		InboundIDs:   req.InboundIDs,
	})
	if err != nil {
		failErr(c, err)
		return
	}
	h.audit.FromAdmin(c, middleware.CurrentAdminID(c), middleware.CurrentUsername(c),
		audit.ActionUserCreate, "user:"+user.Username, true, nil)
	c.JSON(http.StatusCreated, toUserDTO(*user, h.baseURL()))
}

// List handles GET /api/admin/users.
func (h *UserHandler) List(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	params := service.ListParams{
		Offset: offset,
		Limit:  limit,
		Status: models.UserStatus(c.Query("status")),
		Search: c.Query("search"),
	}
	users, total, err := h.users.List(c.Request.Context(), params)
	if err != nil {
		failErr(c, err)
		return
	}
	items := make([]userDTO, 0, len(users))
	for i := range users {
		items = append(items, toUserDTO(users[i], h.baseURL()))
	}
	c.JSON(http.StatusOK, listUsersResponse{
		Items: items, Total: total, Offset: params.Offset, Limit: params.Limit,
	})
}

// Get handles GET /api/admin/users/:id.
func (h *UserHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, err := h.users.GetByID(c.Request.Context(), id)
	if err != nil {
		failErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(*user, h.baseURL()))
}

// Update handles PATCH /api/admin/users/:id.
func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	in := service.UpdateUserInput{
		Status:       req.Status,
		DataLimit:    req.DataLimit,
		DataStrategy: req.DataStrategy,
		Note:         req.Note,
		InboundIDs:   req.InboundIDs,
	}
	// Translate expiry semantics: clear_expiry wins over expire_at.
	if req.ClearExpiry {
		var nilTime *time.Time
		in.ExpireAt = &nilTime
	} else if req.ExpireAt != nil {
		in.ExpireAt = &req.ExpireAt
	}

	user, err := h.users.Update(c.Request.Context(), id, in)
	if err != nil {
		failErr(c, err)
		return
	}
	h.audit.FromAdmin(c, middleware.CurrentAdminID(c), middleware.CurrentUsername(c),
		audit.ActionUserUpdate, "user:"+user.Username, true, nil)
	c.JSON(http.StatusOK, toUserDTO(*user, h.baseURL()))
}

// ResetTraffic handles POST /api/admin/users/:id/reset.
func (h *UserHandler) ResetTraffic(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, err := h.users.ResetTraffic(c.Request.Context(), id)
	if err != nil {
		failErr(c, err)
		return
	}
	h.audit.FromAdmin(c, middleware.CurrentAdminID(c), middleware.CurrentUsername(c),
		audit.ActionUserReset, "user:"+user.Username, true, nil)
	c.JSON(http.StatusOK, toUserDTO(*user, h.baseURL()))
}

// RegenerateSub handles POST /api/admin/users/:id/regenerate-sub.
func (h *UserHandler) RegenerateSub(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, err := h.users.RegenerateSubToken(c.Request.Context(), id)
	if err != nil {
		failErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserDTO(*user, h.baseURL()))
}

// Delete handles DELETE /api/admin/users/:id.
func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.users.Delete(c.Request.Context(), id); err != nil {
		failErr(c, err)
		return
	}
	h.audit.FromAdmin(c, middleware.CurrentAdminID(c), middleware.CurrentUsername(c),
		audit.ActionUserDelete, "user:"+strconv.FormatUint(uint64(id), 10), true, nil)
	c.Status(http.StatusNoContent)
}

// parseID extracts and validates the :id path param.
func parseID(c *gin.Context) (uint, bool) {
	raw := c.Param("id")
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		fail(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return uint(n), true
}
