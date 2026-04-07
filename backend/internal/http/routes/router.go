package routes

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/http/handlers"
	"github.com/marina1815/nutrimatch/internal/http/middleware"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
)

func SetupRouter(cfg *config.Config, tokens *security.TokenManager, sessions repository.SessionRepository, auth *handlers.AuthHandler, profiles *handlers.ProfileHandler, recs *handlers.RecommendationHandler, health *handlers.HealthHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.BodyLimit(cfg.BodyLimitBytes))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimit())

	origins := strings.Split(cfg.CORSOrigins, ",")
	cleaned := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	corsCfg := cors.Config{
		AllowOrigins:     cleaned,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}
	r.Use(cors.New(corsCfg))

	r.GET("/api/v1/health", health.Ping)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/register", auth.Register)
		v1.POST("/auth/login", auth.Login)
		v1.POST("/auth/refresh", auth.Refresh)
		v1.POST("/auth/logout", auth.Logout)

		protected := v1.Group("")
		protected.Use(middleware.AuthRequired(tokens, sessions))
		protected.POST("/profile", profiles.Upsert)
		protected.GET("/profile", profiles.Get)
		protected.GET("/recommendations/:profileId", recs.Get)
	}

	return r
}
