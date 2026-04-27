package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/marina1815/nutrimatch/internal/clients/googleai"
	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/database"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/http/handlers"
	"github.com/marina1815/nutrimatch/internal/http/routes"
	"github.com/marina1815/nutrimatch/internal/repository/gorm"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/marina1815/nutrimatch/internal/services"
	"golang.org/x/time/rate"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(cfg.DBURL, cfg.AppEnv)
	if err != nil {
		log.Fatal(err)
	}

	userRepo := gormrepo.NewUserRepository(db)
	profileRepo := gormrepo.NewProfileRepository(db)
	sessionRepo := gormrepo.NewSessionRepository(db)
	medicalRuleRepo := gormrepo.NewMedicalRuleRepository(db)
	recommendationTraceRepo := gormrepo.NewRecommendationTraceRepository(db)
	vectorRepo := gormrepo.NewVectorRepository(db)
	searchResponseCacheRepo := gormrepo.NewSearchResponseCacheRepository(db)
	auditRepo := gormrepo.NewAuditRepository(db)
	authFailureRepo := gormrepo.NewAuthFailureRepository(db)
	rateLimitBucketRepo := gormrepo.NewRateLimitBucketRepository(db)
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
		log.Fatal(err)
	}
	mfaCipher, err := security.NewCipher(cfg.MFASecretKey)
	if err != nil {
		log.Fatal(err)
	}
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPID:          cfg.WebAuthnRPID,
		RPDisplayName: cfg.WebAuthnRPDisplayName,
		RPOrigins:     cfg.WebAuthnOrigins,
	})
	if err != nil {
		log.Fatal(err)
	}
	recommendationCache := security.NewTTLCache[*dto.RecommendationResponse](3 * time.Minute)
	searchCache := security.NewTTLCache[*spoonacular.SearchResponse](2 * time.Minute)
	recommendationQuota := security.NewPersistentQuotaManager(rateLimitBucketRepo, rate.Every(2*time.Second), 3)

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
		Nutrition:    nutritionProfileService,
		MedicalRules: medicalRuleRepo,
	}
	similarityService := &services.SimilarityService{
		Profiles:   profileRepo,
		Vectors:    vectorRepo,
		Embeddings: &services.EmbeddingService{Vectors: vectorRepo},
		Semantic:   &services.LocalSemanticExpander{},
	}
	ingredientService := &services.IngredientService{}

	recipeClient := &spoonacular.Client{
		BaseURL: cfg.SpoonacularBaseURL,
		APIKey:  cfg.SpoonacularAPIKey,
	}
	ingredientService.Client = recipeClient
	recipeSearcher := &spoonacular.ResilientSearcher{
		Base:                    recipeClient,
		Cache:                   searchCache,
		Persistent:              searchResponseCacheRepo,
		PersistentTTL:           cfg.SpoonacularSearchCacheTTL,
		MaxRetries:              1,
		RetryDelay:              150 * time.Millisecond,
		CircuitBreakerThreshold: cfg.SpoonacularCircuitFailures,
		CircuitBreakerCooldown:  cfg.SpoonacularCircuitCooldown,
	}
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
		Recipes:      recipeSearcher,
		AI:           aiClient,
		MedicalRules: medicalRuleRepo,
		Traces:       recommendationTraceRepo,
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

	router := routes.SetupRouter(cfg, tokens, csrfManager, sessionRepo, rateLimitBucketRepo, authHandler, profileHandler, recHandler, healthHandler)
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
		log.Fatal(err)
	}
}
