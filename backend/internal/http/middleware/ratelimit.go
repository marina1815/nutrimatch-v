package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var (
	visitors = map[string]*visitor{}
	mu       sync.Mutex
	once     sync.Once
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit() gin.HandlerFunc {
	startLimiterCleanup()
	return func(c *gin.Context) {
		key := rateKey(c)
		limit, burst := ratePolicy(c)
		limiter := getLimiter(key, limit, burst)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit"})
			return
		}
		c.Next()
	}
}

func getLimiter(key string, limit rate.Limit, burst int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	entry, exists := visitors[key]
	if !exists {
		entry = &visitor{
			limiter:  rate.NewLimiter(limit, burst),
			lastSeen: time.Now(),
		}
		visitors[key] = entry
		return entry.limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

func startLimiterCleanup() {
	once.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				cutoff := time.Now().Add(-15 * time.Minute)
				mu.Lock()
				for key, entry := range visitors {
					if entry.lastSeen.Before(cutoff) {
						delete(visitors, key)
					}
				}
				mu.Unlock()
			}
		}()
	})
}

func rateKey(c *gin.Context) string {
	return c.ClientIP() + "|" + c.FullPath()
}

func ratePolicy(c *gin.Context) (rate.Limit, int) {
	path := c.FullPath()
	if path == "/api/v1/auth/login" || path == "/api/v1/auth/register" || path == "/api/v1/auth/refresh" {
		return rate.Every(1500 * time.Millisecond), 5
	}
	return rate.Every(200 * time.Millisecond), 20
}
