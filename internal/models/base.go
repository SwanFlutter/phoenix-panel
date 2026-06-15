// Package models defines the GORM data models and shared domain types for PHOENIX PANEL.
package models

import (
	"time"

	"gorm.io/gorm"
)

// Base is embedded in every persisted model and carries common bookkeeping
// columns. We use an explicit auto-increment ID plus soft-delete support.
type Base struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// AllModels returns every model that participates in auto-migration, in
// dependency order. Keep this in sync when adding new tables.
func AllModels() []any {
	return []any{
		&Admin{},
		&Setting{},
		&Node{},
		&Inbound{},
		&User{},
		&UserInbound{},
		&TrafficLog{},
		&AuditLog{},
	}
}
