package models

// Setting is a single key/value configuration row for panel-wide settings that
// admins can change at runtime (branding, default subscription template, etc.).
type Setting struct {
	Base
	Key   string `gorm:"uniqueIndex;size:128;not null" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

// Well-known setting keys.
const (
	SettingPanelTitle       = "panel.title"
	SettingSubUpdateInterval = "subscription.update_interval_hours"
	SettingDefaultDataLimit = "user.default_data_limit"
	SettingTheme            = "panel.default_theme"
	SettingLanguage         = "panel.default_language"
)
