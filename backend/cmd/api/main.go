package main

import (
	"log"

	"github.com/marina1815/nutrimatch/internal/clients/googleai"
	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/database"
	"github.com/marina1815/nutrimatch/internal/http/handlers"
	"github.com/marina1815/nutrimatch/internal/http/routes"
	"github.com/marina1815/nutrimatch/internal/repository/gorm"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/marina1815/nutrimatch/internal/services"
)

func main() {
	cfg := config.Load()
	if cfg.DBURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := database.Connect(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	userRepo := gormrepo.NewUserRepository(db)
	profileRepo := gormrepo.NewProfileRepository(db)
	sessionRepo := gormrepo.NewSessionRepository(db)

	tokens := &security.TokenManager{
		Secret:      []byte(cfg.JWTSecret),
		Issuer:      cfg.JWTIssuer,
		Audience:    cfg.JWTAudience,
		AccessTTL:   cfg.AccessTokenTTL,
		RefreshTTL:  cfg.RefreshTokenTTL,
		TokenPepper: []byte(cfg.RefreshTokenPepper),
	}

	authService := &services.AuthService{
		Users:          userRepo,
		Sessions:       sessionRepo,
		Tokens:         tokens,
		PasswordParams: security.Argon2Params{
			Time:       cfg.Argon2Time,
			Memory:     cfg.Argon2Memory,
			Threads:    cfg.Argon2Threads,
			KeyLength:  cfg.Argon2KeyLength,
			SaltLength: cfg.Argon2SaltLength,
		},
	}

	profileService := &services.ProfileService{
		Profiles: profileRepo,
		Users:    userRepo,
	}

	recipeClient := &spoonacular.Client{
		BaseURL: cfg.SpoonacularBaseURL,
		APIKey:  cfg.SpoonacularAPIKey,
	}
	aiClient := &googleai.Client{
		BaseURL: cfg.GoogleAIBaseURL,
		APIKey:  cfg.GoogleAIAPIKey,
		Model:   cfg.GoogleAIModel,
	}

	recommendationService := &services.RecommendationService{
		Profiles: profileService,
		Recipes:  recipeClient,
		AI:       aiClient,
	}

	authHandler := &handlers.AuthHandler{
		Cfg:   cfg,
		Auth:  authService,
		Users: userRepo,
	}

	profileHandler := &handlers.ProfileHandler{Profiles: profileService}
	recHandler := &handlers.RecommendationHandler{Service: recommendationService}
	healthHandler := &handlers.HealthHandler{}

	router := routes.SetupRouter(cfg, tokens, sessionRepo, authHandler, profileHandler, recHandler, healthHandler)
	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}

