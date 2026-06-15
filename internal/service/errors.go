package service

import "strings"

// isUniqueViolation reports whether err looks like a unique-constraint failure
// on either SQLite or PostgreSQL, without importing driver-specific packages.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || // sqlite: "UNIQUE constraint failed"
		strings.Contains(msg, "duplicate key") || // postgres
		strings.Contains(msg, "23505") // postgres SQLSTATE
}
