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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(raw, "Bearer ")
		claims, err := tokens.ParseAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		session, err := sessions.GetByID(c.Request.Context(), claims.SessionID)
		now := time.Now()
		if err != nil || session.RevokedAt != nil || session.ExpiresAt.Before(now) || session.IdleExpiresAt.Before(now) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session revoked"})
			return
		}
		if session.UserID != claims.Subject {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session subject mismatch"})
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
		c.Set("auth_method", session.AuthMethod)
		c.Next()
	}
}
