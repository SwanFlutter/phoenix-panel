package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Protocol enumerates the proxy protocols PHOENIX PANEL can issue to users.
// The set is the intersection/union supported across Xray-core and sing-box.
type Protocol string

const (
	ProtoVLESS       Protocol = "vless"
	ProtoVMess       Protocol = "vmess"
	ProtoTrojan      Protocol = "trojan"
	ProtoShadowsocks Protocol = "shadowsocks"
	ProtoHysteria2   Protocol = "hysteria2" // sing-box only
	ProtoTUIC        Protocol = "tuic"      // sing-box only
)

// Valid reports whether p is a recognized protocol.
func (p Protocol) Valid() bool {
	switch p {
	case ProtoVLESS, ProtoVMess, ProtoTrojan, ProtoShadowsocks, ProtoHysteria2, ProtoTUIC:
		return true
	}
	return false
}

// SupportedBy reports whether the given core can serve this protocol.
func (p Protocol) SupportedBy(core CoreType) bool {
	switch p {
	case ProtoHysteria2, ProtoTUIC:
		return core == CoreSingBox
	case ProtoVLESS, ProtoVMess, ProtoTrojan, ProtoShadowsocks:
		return true // both cores
	}
	return false
}

// JSONMap is a string-keyed map persisted as a JSON text column. It works on
// both SQLite and Postgres without driver-specific JSON types.
type JSONMap map[string]any

// Value implements driver.Valuer (marshals to JSON text).
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner (unmarshals JSON text/bytes).
func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("JSONMap: unsupported Scan source type")
	}
	if len(data) == 0 {
		*m = nil
		return nil
	}
	return json.Unmarshal(data, m)
}
