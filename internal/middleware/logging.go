package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID assigns a unique ID to each request (echoed in the X-Request-ID
// header) for correlation across logs.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Logger emits a structured slog line per request.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()

		reqID, _ := c.Get("request_id")
		slog.Info("http_request",
			"id", reqID,
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
			"latency_ms", time.Since(start).Milliseconds(),
			"size", c.Writer.Size(),
		)
	}
}

// Recovery converts panics into a 500 without leaking internals to the client.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				reqID, _ := c.Get("request_id")
				slog.Error("panic recovered", "id", reqID, "err", err, "path", c.Request.URL.Path)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
				}
			}
		}()
		c.Next()
	}
}
