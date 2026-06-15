package models

import (
	"time"
)

// UserStatus is the lifecycle state of a subscription user.
type UserStatus string

const (
	UserActive   UserStatus = "active"
	UserDisabled UserStatus = "disabled"
	UserLimited  UserStatus = "limited"  // hit data cap
	UserExpired  UserStatus = "expired"  // passed expiry date
	UserOnHold   UserStatus = "on_hold"  // not started until first connect
)

// DataLimitStrategy controls what happens when a user reaches their data cap.
type DataLimitStrategy string

const (
	StrategyNoReset   DataLimitStrategy = "no_reset"
	StrategyDaily     DataLimitStrategy = "daily"
	StrategyWeekly    DataLimitStrategy = "weekly"
	StrategyMonthly   DataLimitStrategy = "monthly"
)

// User is a subscription holder. Identity material (uuid/password) is reused
// across every protocol link the panel issues for this user.
type User struct {
	Base

	Username string     `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Status   UserStatus `gorm:"size:16;not null;default:active" json:"status"`

	// Shared credential material across protocols.
	UUID           string `gorm:"uniqueIndex;size:64;not null" json:"uuid"`            // VLESS/VMess/TUIC id
	TrojanPassword string `gorm:"size:64;not null" json:"-"`                          // Trojan/Hysteria2 password
	SSPassword     string `gorm:"size:64;not null" json:"-"`                          // Shadowsocks password
	SSMethod       string `gorm:"size:48;not null;default:chacha20-ietf-poly1305" json:"ss_method"`

	// Subscription token used in the public /sub/{token} URL (unguessable).
	SubToken string `gorm:"uniqueIndex;size:64;not null" json:"sub_token"`

	// Quota & accounting (bytes). DataLimit == 0 means unlimited.
	DataLimit    int64             `gorm:"not null;default:0" json:"data_limit"`
	UsedUp       int64             `gorm:"not null;default:0" json:"used_up"`   // upload bytes
	UsedDown     int64             `gorm:"not null;default:0" json:"used_down"` // download bytes
	DataStrategy DataLimitStrategy `gorm:"size:16;not null;default:no_reset" json:"data_strategy"`
	LastResetAt  *time.Time        `json:"last_reset_at,omitempty"`

	// Expiry. ExpireAt == nil means never expires. For on_hold users we store
	// a duration instead and compute ExpireAt at first connection.
	ExpireAt     *time.Time `json:"expire_at,omitempty"`
	OnHoldExpire *int64     `json:"on_hold_expire_seconds,omitempty"` // seconds, used while status=on_hold

	Note      string     `gorm:"size:500" json:"note,omitempty"`
	SubLastAt *time.Time `json:"sub_last_at,omitempty"` // last time the sub URL was fetched
	OnlineAt  *time.Time `json:"online_at,omitempty"`   // last traffic seen

	// Which inbounds this user is provisioned on (many-to-many via UserInbound).
	Inbounds []Inbound `gorm:"many2many:user_inbounds;" json:"inbounds,omitempty"`
}

// UserInbound is the join row linking users to inbounds, allowing per-link
// enable/disable without deleting the association.
type UserInbound struct {
	UserID    uint `gorm:"primaryKey" json:"user_id"`
	InboundID uint `gorm:"primaryKey" json:"inbound_id"`
	Enabled   bool `gorm:"not null;default:true" json:"enabled"`
}

// UsedTotal returns combined up+down usage in bytes.
func (u User) UsedTotal() int64 { return u.UsedUp + u.UsedDown }

// IsExpired reports whether the user has passed their expiry timestamp.
func (u User) IsExpired(now time.Time) bool {
	return u.ExpireAt != nil && now.After(*u.ExpireAt)
}

// IsLimited reports whether the user reached their data cap.
func (u User) IsLimited() bool {
	return u.DataLimit > 0 && u.UsedTotal() >= u.DataLimit
}

// RemainingData returns bytes left, or -1 for unlimited.
func (u User) RemainingData() int64 {
	if u.DataLimit == 0 {
		return -1
	}
	r := u.DataLimit - u.UsedTotal()
	if r < 0 {
		return 0
	}
	return r
}

// EffectiveStatus computes the status the user should currently have, taking
// expiry and data limits into account. Disabled is sticky (operator override).
func (u User) EffectiveStatus(now time.Time) UserStatus {
	if u.Status == UserDisabled || u.Status == UserOnHold {
		return u.Status
	}
	if u.IsExpired(now) {
		return UserExpired
	}
	if u.IsLimited() {
		return UserLimited
	}
	return UserActive
}

// CanConnect reports whether the user is currently allowed to use the service.
func (u User) CanConnect(now time.Time) bool {
	return u.EffectiveStatus(now) == UserActive
}
