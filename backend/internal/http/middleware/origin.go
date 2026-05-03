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
				abortError(c, http.StatusForbidden, "MISSING_TRUSTED_ORIGIN", "missing trusted origin")
				return
			}

			parsed, err := url.Parse(referer)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				abortError(c, http.StatusForbidden, "INVALID_ORIGIN", "invalid origin")
				return
			}
			origin = parsed.Scheme + "://" + parsed.Host
		}

		if _, ok := allowed[origin]; !ok {
			abortError(c, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "origin not allowed")
			return
		}

		c.Next()
	}
}
