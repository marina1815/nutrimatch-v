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

func SetupRouter(cfg *config.Config, tokens *security.TokenManager, csrf *security.CSRFManager, sessions repository.SessionRepository, auth *handlers.AuthHandler, profiles *handlers.ProfileHandler, recs *handlers.RecommendationHandler, health *handlers.HealthHandler) *gin.Engine {
	switch strings.ToLower(cfg.AppEnv) {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	trustedProxies := cfg.TrustedProxies
	if len(trustedProxies) == 0 {
		trustedProxies = nil
	}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		panic(err)
	}

	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.BodyLimit(cfg.BodyLimitBytes))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimit())

	cleaned := make([]string, 0, len(cfg.TrustedOrigins))
	for _, origin := range cfg.TrustedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}

	corsCfg := cors.Config{
		AllowOrigins:     cleaned,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", cfg.CSRFHeaderName, "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	}
	r.Use(cors.New(corsCfg))

	r.GET("/api/v1/health", health.Ping)

	v1 := r.Group("/api/v1")
	{
		authOriginGuard := middleware.RequireTrustedOrigin(cfg.TrustedOrigins)
		csrfGuard := middleware.RequireCSRF(cfg, csrf)
		v1.GET("/auth/csrf", authOriginGuard, auth.CSRFToken)
		v1.GET("/auth/oidc/login", auth.OIDCLogin)
		v1.GET("/auth/oidc/callback", auth.OIDCCallback)
		v1.POST("/auth/register", authOriginGuard, csrfGuard, auth.Register)
		v1.POST("/auth/login", authOriginGuard, csrfGuard, auth.Login)
		v1.POST("/auth/refresh", authOriginGuard, csrfGuard, auth.Refresh)
		v1.POST("/auth/logout", authOriginGuard, csrfGuard, auth.Logout)

		protected := v1.Group("")
		protected.Use(middleware.AuthRequired(tokens, sessions))
		protected.POST("/profile", profiles.Upsert)
		protected.GET("/profile", profiles.Get)
		protected.GET("/profile/nutrition", profiles.GetNutrition)
		protected.GET("/recommendations/:profileId", recs.Get)
		protected.GET("/recommendations/:profileId/trace", recs.Trace)
		protected.GET("/recommendations/:profileId/explanation", recs.Explain)
	}

	return r
}
