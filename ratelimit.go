package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	r        rate.Limit
	burst    int
	ttl      time.Duration
}

func NewIPRateLimiter(perMinute, burst int, ttl time.Duration) *IPRateLimiter {
	l := &IPRateLimiter{
		visitors: make(map[string]*visitor),
		r:        rate.Limit(float64(perMinute) / 60.0),
		burst:    burst,
		ttl:      ttl,
	}
	go l.cleanupLoop()
	return l
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.r, l.burst)}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()

	return v.limiter.Allow()
}

func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()

		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > l.ttl {
				delete(l.visitors, ip)
			}
		}

		l.mu.Unlock()
	}
}

// clientIP returns the real client IP, preferring the CF-Connecting-IP header
// set by Cloudflare over Gin's ClientIP (which reads RemoteAddr when TrustedProxies is nil).
func clientIP(c *gin.Context) string {
	if ip := c.GetHeader("CF-Connecting-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

func RateLimitMiddleware(l *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(clientIP(c)) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"errors": []string{"too many requests"},
			})
			return
		}
		c.Next()
	}
}
