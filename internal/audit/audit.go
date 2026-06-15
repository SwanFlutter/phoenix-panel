// Package audit records security-relevant events to the audit_logs table.
package audit

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/phoenix-panel/phoenix/internal/models"
)

// Common action constants. Keep these stable — dashboards filter on them.
const (
	ActionAdminLogin   = "admin.login"
	ActionAdminLogout  = "admin.logout"
	ActionUserCreate   = "user.create"
	ActionUserUpdate   = "user.update"
	ActionUserDelete   = "user.delete"
	ActionUserReset    = "user.reset_traffic"
	ActionNodeCreate   = "node.create"
	ActionNodeUpdate   = "node.update"
	ActionNodeDelete   = "node.delete"
	ActionSettingWrite = "setting.write"
)

// Logger writes audit entries. It deliberately swallows write errors (logging
// them) so that auditing never breaks the primary request flow.
type Logger struct {
	db *gorm.DB
}

// New constructs an audit Logger.
func New(db *gorm.DB) *Logger { return &Logger{db: db} }

// Entry describes a single audit record to be written.
type Entry struct {
	ActorType string
	ActorID   uint
	ActorName string
	Action    string
	Target    string
	IP        string
	UserAgent string
	Success   bool
	Detail    models.JSONMap
}

// Write persists an audit entry.
func (l *Logger) Write(e Entry) {
	rec := models.AuditLog{
		ActorType: e.ActorType,
		ActorID:   e.ActorID,
		ActorName: e.ActorName,
		Action:    e.Action,
		Target:    e.Target,
		IP:        e.IP,
		UserAgent: e.UserAgent,
		Success:   e.Success,
		Detail:    e.Detail,
	}
	if err := l.db.Create(&rec).Error; err != nil {
		slog.Error("failed to write audit log", "action", e.Action, "err", err)
	}
}

// FromAdmin is a convenience that fills actor/IP/UA fields from a Gin context
// where the auth middleware has already run.
func (l *Logger) FromAdmin(c *gin.Context, adminID uint, adminName, action, target string, success bool, detail models.JSONMap) {
	l.Write(Entry{
		ActorType: "admin",
		ActorID:   adminID,
		ActorName: adminName,
		Action:    action,
		Target:    target,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Success:   success,
		Detail:    detail,
	})
}
