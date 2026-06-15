package security

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"

	"github.com/google/uuid"
)

// NewUUID returns a random RFC-4122 v4 UUID string (VLESS/VMess/TUIC id).
func NewUUID() string {
	return uuid.NewString()
}

// NewToken returns a URL-safe, unguessable token of n random bytes
// (base64url, no padding). Used for subscription tokens.
func NewToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// NewPassword returns a random hex secret of n bytes, suitable for
// Trojan/Shadowsocks/Hysteria2 passwords.
func NewPassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
