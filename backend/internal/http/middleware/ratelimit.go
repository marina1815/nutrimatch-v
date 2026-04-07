package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var (
	limiters = map[string]*rate.Limiter{}
	mu       sync.Mutex
)

func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := getLimiter(ip)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit"})
			return
		}
		c.Next()
	}
}

func getLimiter(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	limiter, exists := limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(time.Second), 5)
		limiters[ip] = limiter
		// Rate limiters are short-lived; entries can be cleaned periodically if needed.
	}
	return limiter
}
