package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireTrustedOrigin(trustedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(trustedOrigins))
	for _, origin := range trustedOrigins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			referer := strings.TrimSpace(c.GetHeader("Referer"))
			if referer == "" {
				if strings.TrimSpace(c.GetHeader("Sec-Fetch-Site")) == "" {
					c.Next()
					return
				}
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing trusted origin"})
				return
			}

			parsed, err := url.Parse(referer)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid origin"})
				return
			}
			origin = parsed.Scheme + "://" + parsed.Host
		}

		if _, ok := allowed[origin]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin not allowed"})
			return
		}

		c.Next()
	}
}
