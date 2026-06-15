package models

import "time"

// CoreType identifies which proxy core backs an inbound/node.
type CoreType string

const (
	CoreXray    CoreType = "xray"
	CoreSingBox CoreType = "sing-box"
)

// NodeStatus reflects the last-known health of a node.
type NodeStatus string

const (
	NodeOnline      NodeStatus = "online"
	NodeOffline     NodeStatus = "offline"
	NodeConnecting  NodeStatus = "connecting"
	NodeDisabled    NodeStatus = "disabled"
	NodeUnknown     NodeStatus = "unknown"
)

// Node is a server running a proxy core that the panel orchestrates. A
// single-server deployment has exactly one "local" node.
type Node struct {
	Base
	Name        string     `gorm:"size:128;not null" json:"name"`
	Address     string     `gorm:"size:255;not null" json:"address"` // hostname or IP clients connect to
	APIHost     string     `gorm:"size:255" json:"api_host"`          // where the panel reaches the core API
	APIPort     int        `json:"api_port"`
	Core        CoreType   `gorm:"size:16;not null;default:xray" json:"core"`
	Status      NodeStatus `gorm:"size:16;not null;default:unknown" json:"status"`
	IsLocal     bool       `gorm:"not null;default:false" json:"is_local"`
	IsActive    bool       `gorm:"not null;default:true" json:"is_active"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
	XrayVersion string     `gorm:"size:32" json:"xray_version,omitempty"`

	Inbounds []Inbound `gorm:"foreignKey:NodeID" json:"inbounds,omitempty"`
}

// Inbound is a single listening endpoint on a node (protocol + port + transport).
type Inbound struct {
	Base
	NodeID uint `gorm:"index;not null" json:"node_id"`

	Tag       string   `gorm:"size:128;not null" json:"tag"` // unique tag within the core config
	Protocol  Protocol `gorm:"size:32;not null" json:"protocol"`
	Listen    string   `gorm:"size:64;default:0.0.0.0" json:"listen"`
	Port      int      `gorm:"not null" json:"port"`
	Network   string   `gorm:"size:32;default:tcp" json:"network"` // tcp, ws, grpc, http, quic
	Security  string   `gorm:"size:32;default:none" json:"security"` // none, tls, reality
	SNI       string   `gorm:"size:255" json:"sni,omitempty"`
	Host      string   `gorm:"size:255" json:"host,omitempty"`
	Path      string   `gorm:"size:255" json:"path,omitempty"`
	Flow      string   `gorm:"size:64" json:"flow,omitempty"`        // e.g. xtls-rprx-vision
	Fingerprint string `gorm:"size:32" json:"fingerprint,omitempty"` // uTLS fingerprint
	// REALITY-specific (stored as JSON blob to stay protocol-agnostic).
	RealitySettings JSONMap `gorm:"type:text" json:"reality_settings,omitempty"`
	// Free-form additional stream settings, serialized to JSON.
	Extra JSONMap `gorm:"type:text" json:"extra,omitempty"`

	IsActive bool `gorm:"not null;default:true" json:"is_active"`

	Node *Node `gorm:"foreignKey:NodeID" json:"-"`
}
