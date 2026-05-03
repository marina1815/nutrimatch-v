package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/marina1815/nutrimatch/internal/clients/googleai"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/database"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/http/handlers"
	"github.com/marina1815/nutrimatch/internal/http/routes"
	"github.com/marina1815/nutrimatch/internal/localdata"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/repository/gorm"
	redisrepo "github.com/marina1815/nutrimatch/internal/repository/redis"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/marina1815/nutrimatch/internal/services"
	"golang.org/x/time/rate"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration validation failed: %v", err)
	}

	db, err := database.Connect(cfg.DBURL, cfg.AppEnv)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	userRepo := gormrepo.NewUserRepository(db)
	profileRepo := gormrepo.NewProfileRepository(db)
	sessionRepo := gormrepo.NewSessionRepository(db)
	medicalRuleRepo := gormrepo.NewMedicalRuleRepository(db)
	recommendationTraceRepo := gormrepo.NewRecommendationTraceRepository(db)
	dailyRecommendationRepo := gormrepo.NewDailyRecommendationRepository(db)
	vectorRepo := gormrepo.NewVectorRepository(db)
	localRecipeRepo := gormrepo.NewLocalRecipeRepository(db)
	auditRepo := gormrepo.NewAuditRepository(db)
	authFailureRepo := gormrepo.NewAuthFailureRepository(db)
	rateLimitBucketRepo := gormrepo.NewRateLimitBucketRepository(db)
	var rateLimitStore repository.RateLimitBucketRepository = rateLimitBucketRepo
	if cfg.RateLimitStore == "redis" {
		redisClient, err := redisrepo.NewClient(cfg.RedisURL)
		if err != nil {
			log.Fatalf("redis configuration failed: %v", err)
		}
		redisRateRepo := redisrepo.NewRateLimitBucketRepository(redisClient)
		if err := redisRateRepo.Ping(context.Background()); err != nil {
			log.Fatalf("redis connection failed: %v", err)
		}
		defer redisRateRepo.Close()
		rateLimitStore = redisRateRepo
	}
	retentionRepo := gormrepo.NewRetentionRepository(db)
	externalIdentityRepo := gormrepo.NewExternalIdentityRepository(db)
	mfaRepo := gormrepo.NewMFARepository(db)
	txManager := gormrepo.NewTxManager(db)

	tokens := &security.TokenManager{
		Secret:      []byte(cfg.JWTSecret),
		Issuer:      cfg.JWTIssuer,
		Audience:    cfg.JWTAudience,
		AccessTTL:   cfg.AccessTokenTTL,
		RefreshTTL:  cfg.RefreshTokenTTL,
		TokenPepper: []byte(cfg.RefreshTokenPepper),
	}
	csrfManager := &security.CSRFManager{
		Secret: []byte(cfg.JWTSecret),
		TTL:    cfg.CSRFTTL,
	}
	stateManager := &security.StateManager{
		Secret: []byte(cfg.JWTSecret),
		TTL:    cfg.CSRFTTL,
	}
	healthCipher, err := security.NewCipher(cfg.HealthDataKey)
	if err != nil {
		log.Fatalf("health data cipher initialization failed: %v", err)
	}
	mfaCipher, err := security.NewCipher(cfg.MFASecretKey)
	if err != nil {
		log.Fatalf("mfa cipher initialization failed: %v", err)
	}
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.WebAuthnRPID,
		RPDisplayName: cfg.WebAuthnRPDisplayName,
		RPOrigins:     cfg.WebAuthnOrigins,
	})
	if err != nil {
		log.Fatalf("webauthn initialization failed: %v", err)
	}
	recommendationCache := security.NewTTLCache[*dto.RecommendationResponse](3 * time.Minute)
	recommendationQuota := security.NewPersistentQuotaManager(rateLimitStore, rate.Every(2*time.Second), 3)
	if seed, err := localdata.LoadCatalogSeed(); err != nil {
		log.Printf("local recipe catalog seed load failed: %v", err)
	} else if err := localRecipeRepo.Seed(context.Background(), seed); err != nil {
		log.Printf("local recipe catalog seed failed: %v", err)
	}

	authService := &services.AuthService{
		Users:          userRepo,
		Sessions:       sessionRepo,
		Failures:       authFailureRepo,
		TxManager:      txManager,
		Tokens:         tokens,
		SessionIdleTTL: cfg.SessionIdleTTL,
		FailureWindow:  cfg.AuthFailureWindow,
		MaxFailures:    cfg.AuthMaxFailures,
		PasswordParams: security.Argon2Params{
			Time:       cfg.Argon2Time,
			Memory:     cfg.Argon2Memory,
			Threads:    cfg.Argon2Threads,
			KeyLength:  cfg.Argon2KeyLength,
			SaltLength: cfg.Argon2SaltLength,
		},
	}
	auditService := &services.AuditService{Repo: auditRepo}
	retentionService := &services.RetentionService{
		Repo: retentionRepo,
		Policy: repository.RetentionPolicy{
			AuthFailureRetentionDays:         cfg.AuthFailureRetentionDays,
			RateLimitBucketRetentionHours:    cfg.RateLimitBucketRetentionHours,
			SessionRetentionDays:             cfg.SessionRetentionDays,
			RecommendationTraceRetentionDays: cfg.RecommendationTraceRetentionDays,
		},
	}
	if err := retentionService.Apply(context.Background()); err != nil {
		log.Printf("retention cleanup failed: %v", err)
	}
	retentionService.Start(context.Background(), 24*time.Hour, log.Default())
	accessPolicyService := &services.AccessPolicyService{}
	nutritionProfileService := &services.NutritionProfileService{
		Profiles:     profileRepo,
		MedicalRules: medicalRuleRepo,
		TxManager:    txManager,
	}

	profileService := &services.ProfileService{
		Profiles:     profileRepo,
		Users:        userRepo,
		TxManager:    txManager,
		Cipher:       healthCipher,
		Indexer:      security.NewBlindIndexer(cfg.BlindIndexKey),
		Nutrition:    nutritionProfileService,
		MedicalRules: medicalRuleRepo,
	}
	similarityService := &services.SimilarityService{
		Profiles:   profileRepo,
		Vectors:    vectorRepo,
		Embeddings: &services.EmbeddingService{Vectors: vectorRepo},
		Semantic:   &services.LocalSemanticExpander{},
	}
	ingredientService := &services.IngredientService{Local: localRecipeRepo}

	var aiClient services.AITextGenerator
	if cfg.GoogleAIAPIKey != "" {
		aiClient = &googleai.Client{
			BaseURL: cfg.GoogleAIBaseURL,
			APIKey:  cfg.GoogleAIAPIKey,
			Model:   cfg.GoogleAIModel,
		}
	}

	recommendationService := &services.RecommendationService{
		Profiles:     profileService,
		LocalRecipes: localRecipeRepo,
		AI:           aiClient,
		MedicalRules: medicalRuleRepo,
		Traces:       recommendationTraceRepo,
		Daily:        dailyRecommendationRepo,
		Similarity:   similarityService,
		Quota:        recommendationQuota,
		Cache:        recommendationCache,
		TxManager:    txManager,
	}
	oidcService := &services.OIDCService{
		Config:       cfg,
		StateManager: stateManager,
		Users:        userRepo,
		External:     externalIdentityRepo,
		TxManager:    txManager,
		Auth:         authService,
	}
	mfaService := &services.MFAService{
		Repo:     mfaRepo,
		Users:    userRepo,
		Cipher:   mfaCipher,
		WebAuthn: webAuthn,
		Issuer:   cfg.WebAuthnRPDisplayName,
	}

	authHandler := &handlers.AuthHandler{
		Cfg:      cfg,
		Auth:     authService,
		Users:    userRepo,
		Profiles: profileService,
		CSRF:     csrfManager,
		OIDC:     oidcService,
		MFA:      mfaService,
		Audit:    auditService,
	}

	profileHandler := &handlers.ProfileHandler{
		Profiles:    profileService,
		Ingredients: ingredientService,
		Audit:       auditService,
		Access:      accessPolicyService,
	}
	recHandler := &handlers.RecommendationHandler{
		Service: recommendationService,
		Audit:   auditService,
		Access:  accessPolicyService,
	}
	healthHandler := &handlers.HealthHandler{}

	router := routes.SetupRouter(cfg, tokens, csrfManager, sessionRepo, rateLimitStore, authHandler, profileHandler, recHandler, healthHandler)
	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server failed: %v", err)
	}
}
