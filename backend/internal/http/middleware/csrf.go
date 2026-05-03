package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/security"
)

func RequireCSRF(cfg *config.Config, manager *security.CSRFManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		headerToken := strings.TrimSpace(c.GetHeader(cfg.CSRFHeaderName))
		if headerToken == "" {
			abortError(c, http.StatusForbidden, "MISSING_CSRF_COOKIE", "missing csrf cookie")
			return
		}
		if !requestHasCookieValue(c, cfg.CookieNameCSRF, headerToken) {
			abortError(c, http.StatusForbidden, "INVALID_CSRF_TOKEN", "invalid csrf token")
			return
		}
		sessionID := c.GetString("session_id")
		csrfBindingID := c.GetString("csrf_binding_id")
		if sessionID != "" {
			err := manager.ValidateTokenForSession(headerToken, sessionID, csrfBindingID)
			if err != nil {
				abortError(c, http.StatusForbidden, "INVALID_CSRF_TOKEN", "invalid csrf token")
				return
			}
		} else {
			err := manager.ValidateToken(headerToken)
			if err != nil {
				abortError(c, http.StatusForbidden, "INVALID_CSRF_TOKEN", "invalid csrf token")
				return
			}
		}
		c.Next()
	}
}

func requestHasCookieValue(c *gin.Context, name, expected string) bool {
	if c == nil || c.Request == nil {
		return false
	}
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name == name && cookie.Value == expected {
			return true
		}
	}
	return false
}
