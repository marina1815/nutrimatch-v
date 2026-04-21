package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
	"golang.org/x/time/rate"
)

var defaultHTTPRateLimitStore = security.NewInMemoryTokenBucketStore()

type ratePolicyConfig struct {
	BucketType string
	Limit      rate.Limit
	Burst      int
}

func RateLimit(store repository.RateLimitBucketRepository) gin.HandlerFunc {
	bucketStore := chooseHTTPRateLimitStore(store)

	return func(c *gin.Context) {
		key := rateKey(c)
		policy := ratePolicy(c)
		allowed, err := bucketStore.TakeToken(c.Request.Context(), key, policy.BucketType, float64(policy.Limit), policy.Burst, time.Now())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "rate limiter unavailable"})
			return
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit"})
			return
		}
		c.Next()
	}
}

func chooseHTTPRateLimitStore(store repository.RateLimitBucketRepository) security.TokenBucketStore {
	if store != nil {
		return store
	}
	return defaultHTTPRateLimitStore
}

func rateKey(c *gin.Context) string {
	return security.SecureCacheKey("http_rate_limit", c.ClientIP(), c.FullPath())
}

func ratePolicy(c *gin.Context) ratePolicyConfig {
	path := c.FullPath()
	if path == "/api/v1/auth/login" || path == "/api/v1/auth/register" || path == "/api/v1/auth/refresh" {
		return ratePolicyConfig{
			BucketType: "auth_http_rate_limit",
			Limit:      rate.Every(1500 * time.Millisecond),
			Burst:      5,
		}
	}
	return ratePolicyConfig{
		BucketType: "default_http_rate_limit",
		Limit:      rate.Every(200 * time.Millisecond),
		Burst:      20,
	}
}

func ResetRateLimitStateForTest() {
	defaultHTTPRateLimitStore.Reset()
}
