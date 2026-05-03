package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
)

func AuthRequired(tokens *security.TokenManager, sessions repository.SessionRepository, idleTTL time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			abortError(c, http.StatusUnauthorized, "MISSING_BEARER_TOKEN", "missing bearer token")
			return
		}
		token := strings.TrimPrefix(raw, "Bearer ")
		claims, err := tokens.ParseAccessToken(token)
		if err != nil {
			abortError(c, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
			return
		}

		session, err := sessions.GetByID(c.Request.Context(), claims.SessionID)
		now := time.Now()
		if err != nil || session.RevokedAt != nil || session.ExpiresAt.Before(now) || session.IdleExpiresAt.Before(now) {
			abortError(c, http.StatusUnauthorized, "SESSION_INVALID", "session invalid")
			return
		}
		if session.UserID != claims.Subject {
			abortError(c, http.StatusUnauthorized, "SESSION_SUBJECT_MISMATCH", "session subject mismatch")
			return
		}

		if idleTTL > 0 {
			idleExp := now.Add(idleTTL)
			if idleExp.After(session.ExpiresAt) {
				idleExp = session.ExpiresAt
			}
			_ = sessions.Touch(c.Request.Context(), session.ID, idleExp)
		}

		c.Set("user_id", claims.Subject)
		c.Set("session_id", claims.SessionID)
		c.Set("csrf_binding_id", session.CSRFBindingID)
		c.Set("auth_method", session.AuthMethod)
		c.Next()
	}
}
