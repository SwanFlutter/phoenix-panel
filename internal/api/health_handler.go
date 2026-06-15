package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler serves liveness/readiness checks.
type HealthHandler struct {
	db      *gorm.DB
	version string
}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler(db *gorm.DB, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version}
}

// Live handles GET /healthz — process is up.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "version": h.version})
}

// Ready handles GET /readyz — dependencies (DB) are reachable.
func (h *HealthHandler) Ready(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db_unreachable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
