package models

import "time"

// AdminRole controls what an admin can do within the panel.
type AdminRole string

const (
	// RoleSudo can do everything, including managing other admins and settings.
	RoleSudo AdminRole = "sudo"
	// RoleAdmin can manage users and nodes but not other admins or global settings.
	RoleAdmin AdminRole = "admin"
)

// Admin is an operator account that logs into the management dashboard.
type Admin struct {
	Base
	Username     string     `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string     `gorm:"not null" json:"-"`
	Role         AdminRole  `gorm:"size:16;not null;default:admin" json:"role"`
	IsActive     bool       `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP  string     `gorm:"size:64" json:"last_login_ip,omitempty"`
}

// IsSudo reports whether this admin has full privileges.
func (a Admin) IsSudo() bool { return a.Role == RoleSudo }
