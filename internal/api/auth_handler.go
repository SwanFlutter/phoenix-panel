package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/phoenix-panel/phoenix/internal/audit"
	"github.com/phoenix-panel/phoenix/internal/middleware"
	"github.com/phoenix-panel/phoenix/internal/models"
	"github.com/phoenix-panel/phoenix/internal/service"
)

// AuthHandler serves admin authentication endpoints.
type AuthHandler struct {
	auth  *service.AuthService
	audit *audit.Logger
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(auth *service.AuthService, auditLog *audit.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, audit: auditLog}
}

// Login handles POST /api/admin/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "username and password are required")
		return
	}

	res, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP())
	if err != nil {
		h.audit.Write(audit.Entry{
			ActorType: "public", ActorName: req.Username,
			Action: audit.ActionAdminLogin, IP: c.ClientIP(),
			UserAgent: c.Request.UserAgent(), Success: false,
		})
		failErr(c, err)
		return
	}

	h.audit.Write(audit.Entry{
		ActorType: "admin", ActorID: res.Admin.ID, ActorName: res.Admin.Username,
		Action: audit.ActionAdminLogin, IP: c.ClientIP(),
		UserAgent: c.Request.UserAgent(), Success: true,
	})

	c.JSON(http.StatusOK, loginResponse{
		Token:     res.Token,
		TokenType: "Bearer",
		ExpiresAt: res.ExpiresAt,
		Admin:     toAdminDTO(res.Admin),
	})
}

// Me handles GET /api/admin/me — returns the current admin identity.
func (h *AuthHandler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, adminDTO{
		ID:       middleware.CurrentAdminID(c),
		Username: middleware.CurrentUsername(c),
		Role:     models.AdminRole(c.GetString(middleware.CtxRole)),
	})
}

// ChangePassword handles POST /api/admin/change-password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "current_password and new_password (min 8) are required")
		return
	}
	id := middleware.CurrentAdminID(c)
	if err := h.auth.ChangePassword(c.Request.Context(), id, req.CurrentPassword, req.NewPassword); err != nil {
		failErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
