package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// ipLimiter is a per-client-IP token-bucket rate limiter with idle eviction.
// This is a lightweight in-process limiter suitable for single-node panels;
// for multi-node, front it with a shared store (e.g. Redis) instead.
type ipLimiter struct {
	mu       sync.Mutex
	clients  map[string]*client
	rps      rate.Limit
	burst    int
	ttl      time.Duration
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPLimiter(rps float64, burst int) *ipLimiter {
	l := &ipLimiter{
		clients: make(map[string]*client),
		rps:     rate.Limit(rps),
		burst:   burst,
		ttl:     10 * time.Minute,
	}
	go l.reaper()
	return l
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	c, ok := l.clients[ip]
	if !ok {
		lim := rate.NewLimiter(l.rps, l.burst)
		l.clients[ip] = &client{limiter: lim, lastSeen: time.Now()}
		return lim
	}
	c.lastSeen = time.Now()
	return c.limiter
}

// reaper periodically evicts idle clients to bound memory.
func (l *ipLimiter) reaper() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, c := range l.clients {
			if time.Since(c.lastSeen) > l.ttl {
				delete(l.clients, ip)
			}
		}
		l.mu.Unlock()
	}
}

// RateLimit returns middleware enforcing per-IP request rate limits.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	limiter := newIPLimiter(rps, burst)
	return func(c *gin.Context) {
		if !limiter.get(c.ClientIP()).Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, slow down",
			})
			return
		}
		c.Next()
	}
}
