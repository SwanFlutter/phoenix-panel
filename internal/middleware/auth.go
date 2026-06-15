// Package middleware provides Gin middleware for auth, rate limiting and security.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/phoenix-panel/phoenix/internal/models"
	"github.com/phoenix-panel/phoenix/internal/security"
)

// Context keys for values stashed by the auth middleware.
const (
	CtxAdminID  = "admin_id"
	CtxUsername = "admin_username"
	CtxRole     = "admin_role"
)

// Auth returns middleware that requires a valid Bearer JWT. The decoded claims
// are stored on the Gin context for downstream handlers.
func Auth(jwtMgr *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c)
		if raw == "" {
			abortUnauthorized(c, "missing or malformed Authorization header")
			return
		}
		claims, err := jwtMgr.Parse(raw)
		if err != nil {
			abortUnauthorized(c, "invalid or expired token")
			return
		}
		c.Set(CtxAdminID, claims.AdminID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

// RequireSudo must run after Auth; it rejects non-sudo admins.
func RequireSudo() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		if role != string(models.RoleSudo) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "forbidden: sudo privileges required",
			})
			return
		}
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.Header("WWW-Authenticate", "Bearer")
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
}

// CurrentAdminID returns the authenticated admin's ID, or 0 if unauthenticated.
func CurrentAdminID(c *gin.Context) uint {
	if v, ok := c.Get(CtxAdminID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// CurrentUsername returns the authenticated admin's username.
func CurrentUsername(c *gin.Context) string {
	if v, ok := c.Get(CtxUsername); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
