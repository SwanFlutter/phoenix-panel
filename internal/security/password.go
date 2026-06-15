// Package security holds password hashing, token generation and JWT helpers.
package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// We hash admin passwords with Argon2id (memory-hard, OWASP-recommended).
// The encoded format is the standard PHC string:
//   $argon2id$v=19$m=65536,t=3,p=2$<salt-b64>$<hash-b64>
//
// Parameters follow OWASP minimums for argon2id.
const (
	argonMemory  = 64 * 1024 // 64 MiB
	argonTime    = 3
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

var (
	// ErrInvalidHash is returned when a stored hash cannot be parsed.
	ErrInvalidHash = errors.New("security: invalid password hash format")
	// ErrIncompatibleVersion is returned when the argon2 version mismatches.
	ErrIncompatibleVersion = errors.New("security: incompatible argon2 version")
)

// HashPassword returns an Argon2id PHC-encoded hash of the password.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("security: password must not be empty")
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security: read salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64 := base64.RawStdEncoding.EncodeToString
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		b64(salt), b64(hash),
	), nil
}

// VerifyPassword reports whether password matches the encoded Argon2id hash.
// Comparison is constant-time.
func VerifyPassword(password, encoded string) (bool, error) {
	mem, t, threads, salt, want, err := decodeArgon2(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, t, mem, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func decodeArgon2(encoded string) (mem, t uint32, threads uint8, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, hash]
	if len(parts) != 6 || parts[1] != "argon2id" {
		err = ErrInvalidHash
		return
	}
	var version int
	if _, e := fmt.Sscanf(parts[2], "v=%d", &version); e != nil {
		err = ErrInvalidHash
		return
	}
	if version != argon2.Version {
		err = ErrIncompatibleVersion
		return
	}
	if _, e := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &t, &threads); e != nil {
		err = ErrInvalidHash
		return
	}
	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		err = ErrInvalidHash
		return
	}
	if hash, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		err = ErrInvalidHash
		return
	}
	return
}
