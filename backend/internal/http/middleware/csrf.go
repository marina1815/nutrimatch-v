package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/security"
)

func RequireCSRF(cfg *config.Config, manager *security.CSRFManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieToken, err := c.Cookie(cfg.CookieNameCSRF)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing csrf cookie"})
			return
		}
		headerToken := c.GetHeader(cfg.CSRFHeaderName)
		if headerToken == "" || headerToken != cookieToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid csrf token"})
			return
		}
		if err := manager.ValidateToken(cookieToken); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid csrf token"})
			return
		}
		c.Next()
	}
}
