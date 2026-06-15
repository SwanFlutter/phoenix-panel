package models

// AuditLog records security-relevant actions for traceability (OWASP A09:
// Security Logging and Monitoring Failures).
type AuditLog struct {
	Base
	ActorType string  `gorm:"size:16;index" json:"actor_type"` // admin | system | public
	ActorID   uint    `gorm:"index" json:"actor_id"`
	ActorName string  `gorm:"size:64" json:"actor_name"`
	Action    string  `gorm:"size:64;index;not null" json:"action"` // e.g. user.create, admin.login
	Target    string  `gorm:"size:128" json:"target,omitempty"`     // affected entity reference
	IP        string  `gorm:"size:64" json:"ip,omitempty"`
	UserAgent string  `gorm:"size:255" json:"user_agent,omitempty"`
	Success   bool    `gorm:"not null;default:true" json:"success"`
	Detail    JSONMap `gorm:"type:text" json:"detail,omitempty"`
}

// TrafficLog stores periodic snapshots of per-user usage deltas pulled from a
// node, enabling historical charts and billing reconciliation.
type TrafficLog struct {
	Base
	UserID   uint  `gorm:"index;not null" json:"user_id"`
	NodeID   uint  `gorm:"index" json:"node_id"`
	Up       int64 `gorm:"not null" json:"up"`   // delta bytes since last snapshot
	Down     int64 `gorm:"not null" json:"down"` // delta bytes since last snapshot
}
