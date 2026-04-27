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
			abortError(c, http.StatusForbidden, "MISSING_CSRF_COOKIE", "missing csrf cookie")
			return
		}
		headerToken := c.GetHeader(cfg.CSRFHeaderName)
		if headerToken == "" || headerToken != cookieToken {
			abortError(c, http.StatusForbidden, "INVALID_CSRF_TOKEN", "invalid csrf token")
			return
		}
		sessionID := c.GetString("session_id")
		csrfBindingID := c.GetString("csrf_binding_id")
		if sessionID != "" {
			err = manager.ValidateTokenForSession(cookieToken, sessionID, csrfBindingID)
		} else {
			err = manager.ValidateToken(cookieToken)
		}
		if err != nil {
			abortError(c, http.StatusForbidden, "INVALID_CSRF_TOKEN", "invalid csrf token")
			return
		}
		c.Next()
	}
}
