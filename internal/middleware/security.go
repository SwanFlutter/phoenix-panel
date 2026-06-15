package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets a baseline of hardening headers on every response
// (OWASP A05: Security Misconfiguration).
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-XSS-Protection", "0") // modern browsers: rely on CSP, disable legacy auditor
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'")
		// HSTS is only meaningful over HTTPS; harmless to send and ignored on HTTP.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
