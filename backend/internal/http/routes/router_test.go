package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/http/handlers"
	"github.com/marina1815/nutrimatch/internal/http/middleware"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/marina1815/nutrimatch/internal/services"
	"golang.org/x/time/rate"
)

type memoryUserRepository struct {
	byID    map[string]*models.User
	byEmail map[string]*models.User
	nextID  int
}

func newMemoryUserRepository() *memoryUserRepository {
	return &memoryUserRepository{
		byID:    map[string]*models.User{},
		byEmail: map[string]*models.User{},
		nextID:  1,
	}
}

func (r *memoryUserRepository) Create(_ context.Context, user *models.User) error {
	if _, exists := r.byEmail[user.Email]; exists {
		return errors.New("duplicate email")
	}
	if user.ID == "" {
		user.ID = "user-" + strconv.Itoa(r.nextID)
		r.nextID++
	}
	copied := *user
	r.byID[user.ID] = &copied
	r.byEmail[user.Email] = &copied
	return nil
}

func (r *memoryUserRepository) GetByEmail(_ context.Context, email string) (*models.User, error) {
	user, ok := r.byEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	copied := *user
	return &copied, nil
}

func (r *memoryUserRepository) GetByID(_ context.Context, id string) (*models.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	copied := *user
	return &copied, nil
}

func (r *memoryUserRepository) UpdateFullName(_ context.Context, userID, fullName string) error {
	user, ok := r.byID[userID]
	if !ok {
		return errors.New("not found")
	}
	user.FullName = fullName
	r.byEmail[user.Email] = user
	return nil
}

type memorySessionRepository struct {
	byID          map[string]*models.Session
	byRefreshHash map[string]*models.Session
}

type memoryAuthFailureRepository struct {
	failures []models.AuthFailure
}

func newMemorySessionRepository() *memorySessionRepository {
	return &memorySessionRepository{
		byID:          map[string]*models.Session{},
		byRefreshHash: map[string]*models.Session{},
	}
}

func (r *memorySessionRepository) Create(_ context.Context, session *models.Session) error {
	copied := *session
	r.byID[session.ID] = &copied
	r.byRefreshHash[session.RefreshTokenHash] = &copied
	return nil
}

func (r *memorySessionRepository) GetByID(_ context.Context, sessionID string) (*models.Session, error) {
	session, ok := r.byID[sessionID]
	if !ok {
		return nil, errors.New("not found")
	}
	copied := *session
	return &copied, nil
}

func (r *memorySessionRepository) GetByRefreshHash(_ context.Context, refreshHash string) (*models.Session, error) {
	session, ok := r.byRefreshHash[refreshHash]
	if !ok {
		return nil, errors.New("not found")
	}
	copied := *session
	return &copied, nil
}

func (r *memorySessionRepository) Rotate(_ context.Context, sessionID, newRefreshHash string, expiresAt, idleExpiresAt time.Time) error {
	session, ok := r.byID[sessionID]
	if !ok {
		return errors.New("not found")
	}
	delete(r.byRefreshHash, session.RefreshTokenHash)
	session.RefreshTokenHash = newRefreshHash
	session.ExpiresAt = expiresAt
	session.IdleExpiresAt = idleExpiresAt
	session.LastSeenAt = time.Now()
	r.byRefreshHash[newRefreshHash] = session
	return nil
}

func (r *memorySessionRepository) Touch(_ context.Context, sessionID string, idleExpiresAt time.Time) error {
	session, ok := r.byID[sessionID]
	if !ok {
		return errors.New("not found")
	}
	session.IdleExpiresAt = idleExpiresAt
	session.LastSeenAt = time.Now()
	return nil
}

func (r *memorySessionRepository) Revoke(_ context.Context, sessionID string) error {
	session, ok := r.byID[sessionID]
	if !ok {
		return errors.New("not found")
	}
	now := time.Now()
	session.RevokedAt = &now
	return nil
}

func (r *memoryAuthFailureRepository) Create(_ context.Context, failure *models.AuthFailure) error {
	copied := *failure
	r.failures = append(r.failures, copied)
	return nil
}

func (r *memoryAuthFailureRepository) CountRecent(_ context.Context, emailHash, ipHash string, since time.Time) (int64, error) {
	var count int64
	for _, failure := range r.failures {
		if failure.OccurredAt.Before(since) {
			continue
		}
		if (emailHash != "" && failure.EmailHash == emailHash) || (ipHash != "" && failure.IPHash == ipHash) {
			count++
		}
	}
	return count, nil
}

type memoryProfileRepository struct {
	profiles      map[string]*models.Profile
	lifestyles    map[string]*models.Lifestyle
	preferences   map[string]*models.Preferences
	constraints   map[string]*models.Constraints
	nutritions    map[string]*models.NutritionProfile
	nextProfile   int
	nextNutrition int
}

func (r *memoryProfileRepository) UpsertProfile(_ context.Context, profile *models.Profile) error {
	if r.profiles == nil {
		r.profiles = map[string]*models.Profile{}
	}
	if profile.ID == "" {
		r.nextProfile++
		profile.ID = "profile-" + strconv.Itoa(r.nextProfile)
	}
	copied := *profile
	r.profiles[profile.UserID] = &copied
	return nil
}

func (r *memoryProfileRepository) UpsertLifestyle(_ context.Context, lifestyle *models.Lifestyle) error {
	if r.lifestyles == nil {
		r.lifestyles = map[string]*models.Lifestyle{}
	}
	copied := *lifestyle
	r.lifestyles[lifestyle.UserID] = &copied
	return nil
}

func (r *memoryProfileRepository) UpsertPreferences(_ context.Context, preferences *models.Preferences) error {
	if r.preferences == nil {
		r.preferences = map[string]*models.Preferences{}
	}
	copied := *preferences
	r.preferences[preferences.UserID] = &copied
	return nil
}

func (r *memoryProfileRepository) UpsertConstraints(_ context.Context, constraints *models.Constraints) error {
	if r.constraints == nil {
		r.constraints = map[string]*models.Constraints{}
	}
	copied := *constraints
	r.constraints[constraints.UserID] = &copied
	return nil
}

func (r *memoryProfileRepository) GetProfile(_ context.Context, userID string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, error) {
	profile := r.profiles[userID]
	lifestyle := r.lifestyles[userID]
	prefs := r.preferences[userID]
	constraints := r.constraints[userID]
	if profile == nil || lifestyle == nil || prefs == nil || constraints == nil {
		return nil, nil, nil, nil, errors.New("not found")
	}
	profileCopy := *profile
	lifestyleCopy := *lifestyle
	prefsCopy := *prefs
	constraintsCopy := *constraints
	return &profileCopy, &lifestyleCopy, &prefsCopy, &constraintsCopy, nil
}

func (r *memoryProfileRepository) ListProfileBundles(_ context.Context, _ string, _ int) ([]repository.ProfileBundle, error) {
	return []repository.ProfileBundle{}, nil
}

func (r *memoryProfileRepository) UpsertNutritionProfile(_ context.Context, profile *models.NutritionProfile) error {
	if r.nutritions == nil {
		r.nutritions = map[string]*models.NutritionProfile{}
	}
	if profile.ID == "" {
		r.nextNutrition++
		profile.ID = "nutrition-" + strconv.Itoa(r.nextNutrition)
	}
	copied := *profile
	r.nutritions[profile.UserID] = &copied
	return nil
}

func (r *memoryProfileRepository) GetNutritionProfile(_ context.Context, userID string) (*models.NutritionProfile, error) {
	profile := r.nutritions[userID]
	if profile == nil {
		return nil, errors.New("not found")
	}
	copied := *profile
	return &copied, nil
}

type memoryMedicalRuleRepository struct {
	rules []models.MedicalRule
}

func (r *memoryMedicalRuleRepository) ListActive(_ context.Context) ([]models.MedicalRule, error) {
	return append([]models.MedicalRule{}, r.rules...), nil
}

type memoryTraceRepository struct {
	runsByProfile     map[string]*models.RecommendationRun
	candidatesByRunID map[string][]*models.RecommendationCandidate
}

func (r *memoryTraceRepository) CreateRun(_ context.Context, run *models.RecommendationRun) error {
	if r.runsByProfile == nil {
		r.runsByProfile = map[string]*models.RecommendationRun{}
	}
	copied := *run
	r.runsByProfile[run.UserID+"|"+run.ProfileID] = &copied
	return nil
}

func (r *memoryTraceRepository) ReplaceCandidates(_ context.Context, runID string, candidates []*models.RecommendationCandidate) error {
	if r.candidatesByRunID == nil {
		r.candidatesByRunID = map[string][]*models.RecommendationCandidate{}
	}
	items := make([]*models.RecommendationCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		copied := *candidate
		items = append(items, &copied)
	}
	r.candidatesByRunID[runID] = items
	return nil
}

func (r *memoryTraceRepository) GetLatestRunByProfile(_ context.Context, userID, profileID string) (*models.RecommendationRun, []*models.RecommendationCandidate, error) {
	run := r.runsByProfile[userID+"|"+profileID]
	if run == nil {
		return nil, nil, errors.New("not found")
	}
	runCopy := *run
	items := r.candidatesByRunID[run.ID]
	out := make([]*models.RecommendationCandidate, 0, len(items))
	for _, candidate := range items {
		copied := *candidate
		out = append(out, &copied)
	}
	return &runCopy, out, nil
}

func (r *memoryTraceRepository) GetCandidateByRecipeID(_ context.Context, userID, profileID, recipeID string) (*models.RecommendationCandidate, error) {
	run := r.runsByProfile[userID+"|"+profileID]
	if run == nil {
		return nil, errors.New("not found")
	}
	for _, candidate := range r.candidatesByRunID[run.ID] {
		if candidate.UserID == userID && candidate.ProfileID == profileID && candidate.ExternalRecipeID == recipeID {
			copied := *candidate
			return &copied, nil
		}
	}
	return nil, errors.New("not found")
}

type noopAuditRepository struct{}

func (r *noopAuditRepository) Create(_ context.Context, _ *models.AuditEvent) error { return nil }

type noopExternalIdentityRepository struct{}

func (r *noopExternalIdentityRepository) GetByProviderSubject(_ context.Context, _, _, _ string) (*models.ExternalIdentity, error) {
	return nil, errors.New("not found")
}
func (r *noopExternalIdentityRepository) Create(_ context.Context, _ *models.ExternalIdentity) error {
	return nil
}
func (r *noopExternalIdentityRepository) UpdateLogin(_ context.Context, _, _, _ string, _ bool, _ time.Time) error {
	return nil
}

type fakeRecipeSearcher struct {
	resp *spoonacular.SearchResponse
	err  error
}

func (s *fakeRecipeSearcher) Search(_ context.Context, _ spoonacular.SearchOptions) (*spoonacular.SearchResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &spoonacular.SearchResponse{
		Results: []spoonacular.Recipe{
			{
				ID:      101,
				Title:   "Chicken Quinoa Bowl",
				Summary: "Balanced grilled chicken bowl.",
				Nutrition: spoonacular.Nutrition{
					Nutrients: []spoonacular.Nutrient{
						{Name: "Calories", Amount: 520},
						{Name: "Protein", Amount: 42},
						{Name: "Carbohydrates", Amount: 42},
						{Name: "Fat", Amount: 14},
						{Name: "Sugar", Amount: 8},
						{Name: "Sodium", Amount: 380},
					},
				},
				ExtendedIngredients: []spoonacular.Ingredient{
					{Name: "chicken"},
					{Name: "quinoa"},
				},
			},
		},
	}, nil
}

type fakeRouteAITextGenerator struct {
	text string
	err  error
}

func (g *fakeRouteAITextGenerator) GenerateText(_ context.Context, _ string) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	return g.text, nil
}

type fakeIngredientService struct{}

func (s *fakeIngredientService) Suggest(_ context.Context, query string, limit int) ([]string, error) {
	base := []string{"paprika", "papaya", "pasta"}
	out := make([]string, 0, limit)
	for _, item := range base {
		if !strings.HasPrefix(item, strings.ToLower(query)) {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func testConfig() *config.Config {
	return &config.Config{
		AppEnv:             "test",
		BodyLimitBytes:     1024 * 1024,
		JWTSecret:          "abcdefghijklmnopqrstuvwxyz123456",
		JWTIssuer:          "nutrimatch-test",
		JWTAudience:        "nutrimatch-users",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		SessionIdleTTL:     12 * time.Hour,
		AuthFailureWindow:  15 * time.Minute,
		AuthMaxFailures:    5,
		RefreshTokenPepper: "1234567890abcdefghijklmnopqrstuvwxyz",
		HealthDataKey:      "1234567890abcdefghijklmnopqrstuvwxyz",
		Argon2Time:         2,
		Argon2Memory:       65536,
		Argon2Threads:      1,
		Argon2KeyLength:    32,
		Argon2SaltLength:   16,
		CookieNameRefresh:  "nm_refresh",
		CookieNameCSRF:     "nm_csrf",
		CookieNameOIDC:     "nm_oidc",
		CookiePathRefresh:  "/api/v1/auth",
		CookiePathCSRF:     "/api/v1",
		CookieSameSite:     "Lax",
		CSRFHeaderName:     "X-CSRF-Token",
		CSRFTTL:            30 * time.Minute,
		TrustedOrigins:     []string{"http://frontend.test"},
		FrontendBaseURL:    "http://frontend.test",
		CookieSecure:       false,
	}
}

type testServerOptions struct {
	quota             *security.QuotaManager
	authMaxFailures   int
	authFailureWindow time.Duration
	searcher          services.RecipeSearcher
	ai                services.AITextGenerator
	medicalRules      []models.MedicalRule
}

func setupTestServer(t *testing.T) (*httptest.Server, *http.Client, *config.Config) {
	return setupTestServerWithOptions(t, testServerOptions{})
}

func setupTestServerWithOptions(t *testing.T, opts testServerOptions) (*httptest.Server, *http.Client, *config.Config) {
	t.Helper()
	middleware.ResetRateLimitStateForTest()

	cfg := testConfig()
	if opts.authMaxFailures > 0 {
		cfg.AuthMaxFailures = opts.authMaxFailures
	}
	if opts.authFailureWindow > 0 {
		cfg.AuthFailureWindow = opts.authFailureWindow
	}
	userRepo := newMemoryUserRepository()
	profileRepo := &memoryProfileRepository{}
	sessionRepo := newMemorySessionRepository()
	authFailureRepo := &memoryAuthFailureRepository{}
	medicalRuleRepo := &memoryMedicalRuleRepository{rules: append([]models.MedicalRule{}, opts.medicalRules...)}
	traceRepo := &memoryTraceRepository{}
	auditRepo := &noopAuditRepository{}
	externalRepo := &noopExternalIdentityRepository{}

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

	authService := &services.AuthService{
		Users:          userRepo,
		Sessions:       sessionRepo,
		Failures:       authFailureRepo,
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
	}
	profileService := &services.ProfileService{
		Profiles:     profileRepo,
		Users:        userRepo,
		Nutrition:    nutritionProfileService,
		MedicalRules: medicalRuleRepo,
	}
	recommendationService := &services.RecommendationService{
		Profiles:     profileService,
		Recipes:      &fakeRecipeSearcher{},
		MedicalRules: medicalRuleRepo,
		Traces:       traceRepo,
		Quota:        opts.quota,
	}
	if opts.searcher != nil {
		recommendationService.Recipes = opts.searcher
	}
	if opts.ai != nil {
		recommendationService.AI = opts.ai
	}

	authHandler := &handlers.AuthHandler{
		Cfg:      cfg,
		Auth:     authService,
		Users:    userRepo,
		Profiles: profileService,
		CSRF:     csrfManager,
		Audit:    auditService,
	}
	profileHandler := &handlers.ProfileHandler{
		Profiles:    profileService,
		Ingredients: &fakeIngredientService{},
		Audit:       auditService,
		Access:      accessPolicyService,
	}
	recommendationHandler := &handlers.RecommendationHandler{
		Service: recommendationService,
		Audit:   auditService,
		Access:  accessPolicyService,
	}
	router := SetupRouter(cfg, tokens, csrfManager, sessionRepo, nil, authHandler, profileHandler, recommendationHandler, &handlers.HealthHandler{})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := newClientWithJar(t)

	_ = externalRepo
	return server, client, cfg
}

func newClientWithJar(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed to create cookie jar: %v", err)
	}
	return &http.Client{Jar: jar}
}

func csrfTokenFromClient(t *testing.T, client *http.Client, baseURL string, cfg *config.Config) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/auth/csrf", nil)
	if err != nil {
		t.Fatalf("failed to create csrf request: %v", err)
	}
	req.Header.Set("Origin", cfg.TrustedOrigins[0])

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed csrf request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected csrf status 200, got %d", resp.StatusCode)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode csrf response: %v", err)
	}
	if data, ok := payload["data"].(map[string]any); ok {
		payload = data
	}

	token, _ := payload["csrf_token"].(string)
	return token
}

func doJSON(t *testing.T, client *http.Client, method, endpoint string, body any, headers map[string]string) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func decodeJSONBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return data
	}
	return payload
}

func registerUser(t *testing.T, client *http.Client, serverURL string, cfg *config.Config, name, email string) string {
	t.Helper()
	csrfToken := csrfTokenFromClient(t, client, serverURL, cfg)
	resp := doJSON(t, client, http.MethodPost, serverURL+"/api/v1/auth/register", map[string]string{
		"name":     name,
		"email":    email,
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: csrfToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected register status 200, got %d", resp.StatusCode)
	}
	payload := decodeJSONBody(t, resp)
	accessToken := strings.TrimSpace(payload["access_token"].(string))
	if accessToken == "" {
		t.Fatalf("expected access token after registration")
	}
	return accessToken
}

func defaultProfilePayload(fullName string) map[string]any {
	return map[string]any{
		"personal": map[string]any{
			"fullName":   fullName,
			"age":        25,
			"sex":        "male",
			"weight":     75,
			"height":     180,
			"profession": "Student",
			"city":       "Lagos",
		},
		"lifestyle": map[string]any{
			"activityLevel": "light",
			"lifestyleType": "student",
			"goal":          "weight_loss",
			"maxReadyTime":  35,
		},
		"preferences": map[string]any{
			"likes":             []string{"chicken", "quinoa"},
			"dislikes":          []string{"bacon"},
			"mealStyles":        []string{"healthy", "balanced"},
			"mealTypes":         []string{"main_course"},
			"preferredCuisines": []string{"mediterranean"},
			"excludedCuisines":  []string{"american"},
			"mealsPerDay":       3,
		},
		"constraints": map[string]any{
			"allergies":           []string{"dairy"},
			"conditions":          []string{},
			"excludedIngredients": []string{"grapefruit"},
			"hasChronicDisease":   false,
			"chronicDiseases":     []string{},
			"takesMedication":     false,
			"medications":         "",
		},
	}
}

func upsertProfileForToken(t *testing.T, client *http.Client, serverURL, accessToken string, payload map[string]any) string {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, serverURL+"/api/v1/profile", payload, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected profile status 200, got %d", resp.StatusCode)
	}
	result := decodeJSONBody(t, resp)
	profileID := strings.TrimSpace(result["profileId"].(string))
	if profileID == "" {
		t.Fatalf("expected profile id")
	}
	return profileID
}

func TestRouterRegisterProfileRecommendationFlow(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	csrfToken := csrfTokenFromClient(t, client, server.URL, cfg)

	registerResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/register", map[string]string{
		"name":     "Oussama Test",
		"email":    "oussa@example.com",
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: csrfToken,
	})
	if registerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected register status 200, got %d", registerResp.StatusCode)
	}
	registerPayload := decodeJSONBody(t, registerResp)
	accessToken := strings.TrimSpace(registerPayload["access_token"].(string))
	if accessToken == "" {
		t.Fatalf("expected access token")
	}

	profilePayload := map[string]any{
		"personal": map[string]any{
			"fullName":   "Oussama Test",
			"age":        25,
			"sex":        "male",
			"weight":     75,
			"height":     180,
			"profession": "Student",
			"city":       "Lagos",
		},
		"lifestyle": map[string]any{
			"activityLevel": "light",
			"lifestyleType": "student",
			"goal":          "weight_loss",
			"maxReadyTime":  35,
		},
		"preferences": map[string]any{
			"likes":             []string{"chicken", "quinoa"},
			"dislikes":          []string{"bacon"},
			"mealStyles":        []string{"healthy", "balanced"},
			"mealTypes":         []string{"main_course"},
			"preferredCuisines": []string{"mediterranean"},
			"excludedCuisines":  []string{"american"},
			"mealsPerDay":       3,
		},
		"constraints": map[string]any{
			"allergies":           []string{"dairy"},
			"conditions":          []string{},
			"excludedIngredients": []string{"grapefruit"},
			"hasChronicDisease":   false,
			"chronicDiseases":     []string{},
			"takesMedication":     false,
			"medications":         "",
		},
	}

	profileResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/profile", profilePayload, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if profileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected profile status 200, got %d", profileResp.StatusCode)
	}
	profileResult := decodeJSONBody(t, profileResp)
	profileID := profileResult["profileId"].(string)
	if profileID == "" {
		t.Fatalf("expected profile id")
	}

	getProfileResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if getProfileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get profile status 200, got %d", getProfileResp.StatusCode)
	}

	recommendationResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if recommendationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected recommendation status 200, got %d", recommendationResp.StatusCode)
	}
	recommendationPayload := decodeJSONBody(t, recommendationResp)
	meals, ok := recommendationPayload["meals"].([]any)
	if !ok || len(meals) == 0 {
		t.Fatalf("expected at least one recommendation")
	}
}

func TestRouterRedactsSensitiveMedicationsUnlessExplicitlyRequested(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	accessToken := registerUser(t, client, server.URL, cfg, "Sensitive User", "sensitive@example.com")
	payload := defaultProfilePayload("Sensitive User")
	constraints := payload["constraints"].(map[string]any)
	constraints["takesMedication"] = true
	constraints["medications"] = "daily statin"

	upsertProfileForToken(t, client, server.URL, accessToken, payload)

	summaryResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if summaryResp.StatusCode != http.StatusOK {
		t.Fatalf("expected summary profile status 200, got %d", summaryResp.StatusCode)
	}
	summaryPayload := decodeJSONBody(t, summaryResp)
	summaryConstraints := summaryPayload["constraints"].(map[string]any)
	if got := summaryConstraints["medications"].(string); got != "" {
		t.Fatalf("expected medications to be redacted by default, got %q", got)
	}
	if redacted, ok := summaryConstraints["medicationsRedacted"].(bool); !ok || !redacted {
		t.Fatalf("expected medicationsRedacted=true on summary response")
	}

	detailedResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile?includeSensitive=true", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if detailedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected detailed profile status 200, got %d", detailedResp.StatusCode)
	}
	detailedPayload := decodeJSONBody(t, detailedResp)
	detailedConstraints := detailedPayload["constraints"].(map[string]any)
	if got := detailedConstraints["medications"].(string); got != "daily statin" {
		t.Fatalf("expected medications to be returned when explicitly requested, got %q", got)
	}
	if redacted, ok := detailedConstraints["medicationsRedacted"].(bool); !ok || redacted {
		t.Fatalf("expected medicationsRedacted=false when sensitive data is requested")
	}
}

func TestRouterRejectsRegisterWithoutValidCSRF(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	resp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/register", map[string]string{
		"name":     "Oussama Test",
		"email":    "oussa2@example.com",
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: "invalid-token",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected forbidden without valid csrf, got %d", resp.StatusCode)
	}
}

func TestRouterLoginAndRefreshFlow(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	csrfToken := csrfTokenFromClient(t, client, server.URL, cfg)
	registerResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/register", map[string]string{
		"name":     "Refresh User",
		"email":    "refresh@example.com",
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: csrfToken,
	})
	if registerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected register status 200, got %d", registerResp.StatusCode)
	}
	registerResp.Body.Close()

	loginCSRF := csrfTokenFromClient(t, client, server.URL, cfg)
	loginResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
		"email":    "refresh@example.com",
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: loginCSRF,
	})
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginResp.StatusCode)
	}
	loginPayload := decodeJSONBody(t, loginResp)
	firstAccess := loginPayload["access_token"].(string)

	refreshCSRF := csrfTokenFromClient(t, client, server.URL, cfg)
	refreshResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/refresh", nil, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: refreshCSRF,
	})
	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d", refreshResp.StatusCode)
	}
	refreshPayload := decodeJSONBody(t, refreshResp)
	secondAccess := refreshPayload["access_token"].(string)

	if firstAccess == secondAccess {
		t.Fatalf("expected rotated access token after refresh")
	}
}

func TestRouterWhoAmIReturnsCurrentSessionMetadata(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	accessToken := registerUser(t, client, server.URL, cfg, "Session User", "session@example.com")
	profileID := upsertProfileForToken(t, client, server.URL, accessToken, defaultProfilePayload("Session User"))

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/auth/whoami", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected whoami status 200, got %d", resp.StatusCode)
	}

	payload := decodeJSONBody(t, resp)
	if payload["userId"] == "" {
		t.Fatalf("expected whoami userId")
	}
	if hasProfile, ok := payload["hasProfile"].(bool); !ok || !hasProfile {
		t.Fatalf("expected whoami hasProfile=true")
	}
	if got := strings.TrimSpace(payload["profileId"].(string)); got != profileID {
		t.Fatalf("expected whoami profileId %q, got %q", profileID, got)
	}
}

func TestRouterRejectsProfilePayloadWithUnknownFields(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	csrfToken := csrfTokenFromClient(t, client, server.URL, cfg)
	registerResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/register", map[string]string{
		"name":     "Schema User",
		"email":    "schema@example.com",
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: csrfToken,
	})
	if registerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected register status 200, got %d", registerResp.StatusCode)
	}
	registerPayload := decodeJSONBody(t, registerResp)
	accessToken := strings.TrimSpace(registerPayload["access_token"].(string))

	profilePayload := map[string]any{
		"personal": map[string]any{
			"fullName":   "Schema User",
			"age":        25,
			"sex":        "male",
			"weight":     75,
			"height":     180,
			"profession": "Student",
			"city":       "Lagos",
			"unexpected": "field",
		},
		"lifestyle": map[string]any{
			"activityLevel": "light",
			"lifestyleType": "student",
			"goal":          "weight_loss",
			"maxReadyTime":  35,
		},
		"preferences": map[string]any{
			"likes":             []string{"chicken"},
			"dislikes":          []string{},
			"mealStyles":        []string{"healthy"},
			"mealTypes":         []string{"main_course"},
			"preferredCuisines": []string{"mediterranean"},
			"excludedCuisines":  []string{},
			"mealsPerDay":       3,
		},
		"constraints": map[string]any{
			"allergies":           []string{},
			"conditions":          []string{},
			"excludedIngredients": []string{},
			"hasChronicDisease":   false,
			"chronicDiseases":     []string{},
			"takesMedication":     false,
			"medications":         "",
		},
	}

	profileResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/profile", profilePayload, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	defer profileResp.Body.Close()

	if profileResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected profile validation status 400 for unknown field, got %d", profileResp.StatusCode)
	}
}

func TestRouterRejectsAmbiguousProfilePayload(t *testing.T) {
	server, client, cfg := setupTestServer(t)
	accessToken := registerUser(t, client, server.URL, cfg, "Ambiguous User", "ambiguous@example.com")

	payload := defaultProfilePayload("Ambiguous User")
	preferences := payload["preferences"].(map[string]any)
	preferences["likes"] = []string{"chicken"}
	preferences["dislikes"] = []string{"Chicken"}
	constraints := payload["constraints"].(map[string]any)
	constraints["takesMedication"] = false
	constraints["medications"] = "daily statin"

	resp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/profile", payload, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected ambiguous profile payload to be rejected, got %d", resp.StatusCode)
	}
}

func TestRouterRejectsOverComplexProfilePayload(t *testing.T) {
	server, client, cfg := setupTestServer(t)
	accessToken := registerUser(t, client, server.URL, cfg, "Complex User", "complex@example.com")

	longToken := strings.Repeat("a", 50)
	likes := make([]string, 0, 25)
	dislikes := make([]string, 0, 25)
	for i := 0; i < 25; i++ {
		likes = append(likes, longToken+strconv.Itoa(i))
		dislikes = append(dislikes, "b"+longToken+strconv.Itoa(i))
	}

	payload := defaultProfilePayload("Complex User")
	preferences := payload["preferences"].(map[string]any)
	preferences["likes"] = likes
	preferences["dislikes"] = dislikes

	resp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/profile", payload, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected complex profile payload to be rejected, got %d", resp.StatusCode)
	}
}

func TestRouterIngredientSuggestions(t *testing.T) {
	server, client, cfg := setupTestServer(t)
	accessToken := registerUser(t, client, server.URL, cfg, "Suggest User", "suggest@example.com")

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile/ingredients/suggest?q=pa&limit=2", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected suggestion status 200, got %d", resp.StatusCode)
	}
	payload := decodeJSONBody(t, resp)
	items, ok := payload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ingredient suggestions")
	}
}

func TestRouterRejectsProtectedEndpointsWithoutBearerToken(t *testing.T) {
	server, client, _ := setupTestServer(t)

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile", nil, map[string]string{})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without bearer token, got %d", resp.StatusCode)
	}
}

func TestRouterRejectsAuthOriginOutsideTrustedList(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	csrfToken := csrfTokenFromClient(t, client, server.URL, cfg)
	resp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/register", map[string]string{
		"name":     "Origin User",
		"email":    "origin@example.com",
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           "http://evil.test",
		cfg.CSRFHeaderName: csrfToken,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected forbidden for non-trusted origin, got %d", resp.StatusCode)
	}
}

func TestRouterRejectsForeignRecommendationProfileAccess(t *testing.T) {
	server, clientA, cfg := setupTestServer(t)
	clientB := newClientWithJar(t)

	accessTokenA := registerUser(t, clientA, server.URL, cfg, "User A", "usera@example.com")
	profileIDA := upsertProfileForToken(t, clientA, server.URL, accessTokenA, defaultProfilePayload("User A"))

	accessTokenB := registerUser(t, clientB, server.URL, cfg, "User B", "userb@example.com")
	upsertProfileForToken(t, clientB, server.URL, accessTokenB, defaultProfilePayload("User B"))

	resp := doJSON(t, clientB, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileIDA, nil, map[string]string{
		"Authorization": "Bearer " + accessTokenB,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign profile access, got %d", resp.StatusCode)
	}
}

func TestRouterRecommendationQuotaExceeded(t *testing.T) {
	server, client, cfg := setupTestServerWithOptions(t, testServerOptions{
		quota: security.NewQuotaManager(rate.Every(time.Hour), 1),
	})

	accessToken := registerUser(t, client, server.URL, cfg, "Quota User", "quota@example.com")
	profileID := upsertProfileForToken(t, client, server.URL, accessToken, defaultProfilePayload("Quota User"))

	first := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if first.StatusCode != http.StatusOK {
		t.Fatalf("expected first recommendation status 200, got %d", first.StatusCode)
	}
	first.Body.Close()

	second := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	defer second.Body.Close()

	if second.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when recommendation quota is exceeded, got %d", second.StatusCode)
	}
}

func TestRouterRateLimitTriggersOnRepeatedLoginAttempts(t *testing.T) {
	server, client, cfg := setupTestServer(t)

	registerUser(t, client, server.URL, cfg, "Rate User", "rate@example.com")
	loginCSRF := csrfTokenFromClient(t, client, server.URL, cfg)

	statuses := make([]int, 0, 6)
	for i := 0; i < 6; i++ {
		resp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
			"email":    "rate@example.com",
			"password": "VeryStrongPass123!",
		}, map[string]string{
			"Origin":           cfg.TrustedOrigins[0],
			cfg.CSRFHeaderName: loginCSRF,
		})
		statuses = append(statuses, resp.StatusCode)
		resp.Body.Close()
	}

	if statuses[len(statuses)-1] != http.StatusTooManyRequests {
		t.Fatalf("expected final login attempt to be rate limited, got sequence %v", statuses)
	}
}

func TestRouterTemporarilyBlocksLoginAfterRepeatedFailures(t *testing.T) {
	server, client, cfg := setupTestServerWithOptions(t, testServerOptions{
		authMaxFailures:   3,
		authFailureWindow: 15 * time.Minute,
	})

	registerUser(t, client, server.URL, cfg, "Block User", "block@example.com")
	loginCSRF := csrfTokenFromClient(t, client, server.URL, cfg)

	for i := 0; i < 3; i++ {
		resp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
			"email":    "block@example.com",
			"password": "WrongPassword123!",
		}, map[string]string{
			"Origin":           cfg.TrustedOrigins[0],
			cfg.CSRFHeaderName: loginCSRF,
		})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected unauthorized on failed login attempt %d, got %d", i+1, resp.StatusCode)
		}
		resp.Body.Close()
	}

	blockedResp := doJSON(t, client, http.MethodPost, server.URL+"/api/v1/auth/login", map[string]string{
		"email":    "block@example.com",
		"password": "VeryStrongPass123!",
	}, map[string]string{
		"Origin":           cfg.TrustedOrigins[0],
		cfg.CSRFHeaderName: loginCSRF,
	})
	defer blockedResp.Body.Close()

	if blockedResp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected temporary auth block status 429, got %d", blockedResp.StatusCode)
	}
}

func TestRouterSensitiveProfileEndToEndFlow(t *testing.T) {
	server, client, cfg := setupTestServer(t)
	accessToken := registerUser(t, client, server.URL, cfg, "Sensitive Flow", "sensitive-flow@example.com")

	payload := defaultProfilePayload("Sensitive Flow")
	constraints := payload["constraints"].(map[string]any)
	constraints["hasChronicDisease"] = true
	constraints["chronicDiseases"] = []string{"hypertension"}
	constraints["takesMedication"] = true
	constraints["medications"] = "daily statin"

	profileID := upsertProfileForToken(t, client, server.URL, accessToken, payload)

	nutritionResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile/nutrition", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if nutritionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected nutrition profile status 200, got %d", nutritionResp.StatusCode)
	}
	nutritionPayload := decodeJSONBody(t, nutritionResp)
	if nutritionPayload["profileId"] != profileID {
		t.Fatalf("expected nutrition profileId %q, got %#v", profileID, nutritionPayload["profileId"])
	}

	recommendationResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if recommendationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected recommendation status 200, got %d", recommendationResp.StatusCode)
	}
	recommendationPayload := decodeJSONBody(t, recommendationResp)
	runID := strings.TrimSpace(recommendationPayload["runId"].(string))
	if runID == "" {
		t.Fatalf("expected recommendation run id")
	}
	meals, ok := recommendationPayload["meals"].([]any)
	if !ok || len(meals) == 0 {
		t.Fatalf("expected at least one safe recommendation")
	}
	firstMeal := meals[0].(map[string]any)
	mealID := strings.TrimSpace(firstMeal["id"].(string))
	if mealID == "" {
		t.Fatalf("expected first recommendation meal id")
	}

	traceResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/trace", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if traceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trace status 200, got %d", traceResp.StatusCode)
	}
	tracePayload := decodeJSONBody(t, traceResp)
	if tracePayload["runId"] != runID {
		t.Fatalf("expected trace run id %q, got %#v", runID, tracePayload["runId"])
	}

	explanationResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/explanation?mealId="+mealID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if explanationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected explanation status 200, got %d", explanationResp.StatusCode)
	}
	explanationPayload := decodeJSONBody(t, explanationResp)
	if explanationPayload["mealId"] != mealID {
		t.Fatalf("expected explanation meal id %q, got %#v", mealID, explanationPayload["mealId"])
	}
}

func TestRouterAllergyFailSafeRejectsUnsafeRecipeAndPreservesTrace(t *testing.T) {
	server, client, cfg := setupTestServerWithOptions(t, testServerOptions{
		searcher: &fakeRecipeSearcher{
			resp: &spoonacular.SearchResponse{
				Results: []spoonacular.Recipe{
					{
						ID:      301,
						Title:   "Creamy Dairy Pasta",
						Summary: "A creamy pasta dish with cheese sauce.",
						Nutrition: spoonacular.Nutrition{
							Nutrients: []spoonacular.Nutrient{
								{Name: "Calories", Amount: 610},
								{Name: "Protein", Amount: 24},
								{Name: "Carbohydrates", Amount: 58},
								{Name: "Fat", Amount: 18},
								{Name: "Sugar", Amount: 7},
								{Name: "Sodium", Amount: 540},
							},
						},
						ExtendedIngredients: []spoonacular.Ingredient{
							{Name: "dairy"},
							{Name: "pasta"},
						},
					},
					{
						ID:      302,
						Title:   "Lemon Chicken Quinoa Bowl",
						Summary: "Balanced grilled chicken bowl with quinoa and herbs.",
						Nutrition: spoonacular.Nutrition{
							Nutrients: []spoonacular.Nutrient{
								{Name: "Calories", Amount: 520},
								{Name: "Protein", Amount: 41},
								{Name: "Carbohydrates", Amount: 42},
								{Name: "Fat", Amount: 14},
								{Name: "Sugar", Amount: 6},
								{Name: "Sodium", Amount: 360},
							},
						},
						ExtendedIngredients: []spoonacular.Ingredient{
							{Name: "chicken"},
							{Name: "quinoa"},
							{Name: "lemon"},
						},
					},
				},
			},
		},
	})
	accessToken := registerUser(t, client, server.URL, cfg, "Allergy User", "allergy@example.com")
	profileID := upsertProfileForToken(t, client, server.URL, accessToken, defaultProfilePayload("Allergy User"))

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected recommendation status 200, got %d", resp.StatusCode)
	}
	payload := decodeJSONBody(t, resp)
	meals, ok := payload["meals"].([]any)
	if !ok || len(meals) != 1 {
		t.Fatalf("expected exactly one safe recommendation, got %#v", payload["meals"])
	}

	meal := meals[0].(map[string]any)
	if got := strings.TrimSpace(meal["id"].(string)); got != "302" {
		t.Fatalf("expected safe recipe 302 to remain, got %q", got)
	}

	traceResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/trace", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if traceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trace status 200, got %d", traceResp.StatusCode)
	}
	tracePayload := decodeJSONBody(t, traceResp)
	candidates, ok := tracePayload["candidates"].([]any)
	if !ok || len(candidates) != 2 {
		t.Fatalf("expected two trace candidates, got %#v", tracePayload["candidates"])
	}

	var rejectedCandidate map[string]any
	for _, item := range candidates {
		candidate := item.(map[string]any)
		if strings.TrimSpace(candidate["mealId"].(string)) == "301" {
			rejectedCandidate = candidate
			break
		}
	}
	if rejectedCandidate == nil {
		t.Fatalf("expected rejected allergen candidate in trace")
	}
	if accepted, ok := rejectedCandidate["accepted"].(bool); !ok || accepted {
		t.Fatalf("expected allergen candidate to be rejected, got %#v", rejectedCandidate["accepted"])
	}

	reasons, ok := rejectedCandidate["rejectedReasons"].([]any)
	if !ok || len(reasons) == 0 {
		t.Fatalf("expected rejection reasons for allergen candidate, got %#v", rejectedCandidate["rejectedReasons"])
	}
	foundBlockedIngredientReason := false
	for _, reason := range reasons {
		if strings.Contains(strings.ToLower(reason.(string)), "blocked ingredients") {
			foundBlockedIngredientReason = true
			break
		}
	}
	if !foundBlockedIngredientReason {
		t.Fatalf("expected blocked ingredient rejection reason, got %#v", rejectedCandidate["rejectedReasons"])
	}
}

func TestRouterChronicDiseaseRuleShapesNutritionAndRecommendations(t *testing.T) {
	server, client, cfg := setupTestServerWithOptions(t, testServerOptions{
		medicalRules: []models.MedicalRule{
			{
				Code:               "hypertension_rule",
				ConditionKey:       "hypertension",
				BlockedIngredients: models.StringSlice{"anchovy"},
				RequiredTags:       models.StringSlice{"low-sodium"},
				MaxSodiumMg:        500,
				Rationale:          "hypertension profile requires low sodium meals",
				Active:             true,
			},
		},
		searcher: &fakeRecipeSearcher{
			resp: &spoonacular.SearchResponse{
				Results: []spoonacular.Recipe{
					{
						ID:      401,
						Title:   "Anchovy Power Bowl",
						Summary: "Savory bowl with anchovy dressing.",
						Nutrition: spoonacular.Nutrition{
							Nutrients: []spoonacular.Nutrient{
								{Name: "Calories", Amount: 540},
								{Name: "Protein", Amount: 30},
								{Name: "Carbohydrates", Amount: 45},
								{Name: "Fat", Amount: 18},
								{Name: "Sugar", Amount: 6},
								{Name: "Sodium", Amount: 820},
							},
						},
						ExtendedIngredients: []spoonacular.Ingredient{
							{Name: "anchovy"},
							{Name: "rice"},
						},
					},
					{
						ID:      402,
						Title:   "Low Sodium Chicken Bowl",
						Summary: "Balanced grilled chicken bowl with herbs and quinoa.",
						Nutrition: spoonacular.Nutrition{
							Nutrients: []spoonacular.Nutrient{
								{Name: "Calories", Amount: 500},
								{Name: "Protein", Amount: 42},
								{Name: "Carbohydrates", Amount: 40},
								{Name: "Fat", Amount: 14},
								{Name: "Sugar", Amount: 5},
								{Name: "Sodium", Amount: 420},
							},
						},
						ExtendedIngredients: []spoonacular.Ingredient{
							{Name: "chicken"},
							{Name: "quinoa"},
							{Name: "parsley"},
						},
					},
				},
			},
		},
	})
	accessToken := registerUser(t, client, server.URL, cfg, "Hypertension User", "hypertension@example.com")

	payload := defaultProfilePayload("Hypertension User")
	lifestyle := payload["lifestyle"].(map[string]any)
	lifestyle["goal"] = "medical_diet"
	constraints := payload["constraints"].(map[string]any)
	constraints["hasChronicDisease"] = true
	constraints["chronicDiseases"] = []string{"hypertension"}
	profileID := upsertProfileForToken(t, client, server.URL, accessToken, payload)

	nutritionResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile/nutrition", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if nutritionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected nutrition profile status 200, got %d", nutritionResp.StatusCode)
	}
	nutritionPayload := decodeJSONBody(t, nutritionResp)
	if got := nutritionPayload["maxSodiumMgPerMeal"].(float64); got != 500 {
		t.Fatalf("expected chronic disease rule to cap sodium at 500mg, got %v", got)
	}
	derivedExcluded := nutritionPayload["derivedExcluded"].([]any)
	foundAnchovy := false
	for _, item := range derivedExcluded {
		if strings.EqualFold(item.(string), "anchovy") {
			foundAnchovy = true
			break
		}
	}
	if !foundAnchovy {
		t.Fatalf("expected derivedExcluded to include anchovy, got %#v", nutritionPayload["derivedExcluded"])
	}
	metadata := nutritionPayload["metadata"].(map[string]any)
	matchedRuleCodes := metadata["matchedRuleCodes"].([]any)
	if len(matchedRuleCodes) != 1 || matchedRuleCodes[0].(string) != "hypertension_rule" {
		t.Fatalf("expected matchedRuleCodes to include hypertension_rule, got %#v", matchedRuleCodes)
	}

	recommendationResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if recommendationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected recommendation status 200, got %d", recommendationResp.StatusCode)
	}
	recommendationPayload := decodeJSONBody(t, recommendationResp)
	meals := recommendationPayload["meals"].([]any)
	if len(meals) != 1 {
		t.Fatalf("expected one safe recommendation for hypertension profile, got %#v", recommendationPayload["meals"])
	}
	if got := strings.TrimSpace(meals[0].(map[string]any)["id"].(string)); got != "402" {
		t.Fatalf("expected recipe 402 to remain after chronic disease filtering, got %q", got)
	}

	traceResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/trace", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if traceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trace status 200, got %d", traceResp.StatusCode)
	}
	tracePayload := decodeJSONBody(t, traceResp)
	decisionSummary := tracePayload["decisionSummary"].(map[string]any)
	if decisionSummary["aiApplied"] != false {
		t.Fatalf("expected aiApplied=false for chronic disease profile, got %#v", decisionSummary["aiApplied"])
	}
}

func TestRouterMedicationRuleShapesNutritionAndRecommendations(t *testing.T) {
	server, client, cfg := setupTestServerWithOptions(t, testServerOptions{
		medicalRules: []models.MedicalRule{
			{
				Code:               "statin_rule",
				ConditionKey:       "",
				MedicationPattern:  "statin",
				BlockedIngredients: models.StringSlice{"grapefruit"},
				Rationale:          "statin interactions exclude grapefruit",
				Active:             true,
			},
		},
		searcher: &fakeRecipeSearcher{
			resp: &spoonacular.SearchResponse{
				Results: []spoonacular.Recipe{
					{
						ID:      501,
						Title:   "Grapefruit Chicken Salad",
						Summary: "Salad with grapefruit segments and grilled chicken.",
						Nutrition: spoonacular.Nutrition{
							Nutrients: []spoonacular.Nutrient{
								{Name: "Calories", Amount: 430},
								{Name: "Protein", Amount: 32},
								{Name: "Carbohydrates", Amount: 26},
								{Name: "Fat", Amount: 12},
								{Name: "Sugar", Amount: 11},
								{Name: "Sodium", Amount: 350},
							},
						},
						ExtendedIngredients: []spoonacular.Ingredient{
							{Name: "grapefruit"},
							{Name: "chicken"},
						},
					},
					{
						ID:      502,
						Title:   "Herbed Chicken Quinoa Plate",
						Summary: "Balanced chicken and quinoa plate with herbs.",
						Nutrition: spoonacular.Nutrition{
							Nutrients: []spoonacular.Nutrient{
								{Name: "Calories", Amount: 510},
								{Name: "Protein", Amount: 42},
								{Name: "Carbohydrates", Amount: 41},
								{Name: "Fat", Amount: 14},
								{Name: "Sugar", Amount: 5},
								{Name: "Sodium", Amount: 370},
							},
						},
						ExtendedIngredients: []spoonacular.Ingredient{
							{Name: "chicken"},
							{Name: "quinoa"},
							{Name: "lemon"},
						},
					},
				},
			},
		},
	})
	accessToken := registerUser(t, client, server.URL, cfg, "Medication User", "medication@example.com")

	payload := defaultProfilePayload("Medication User")
	constraints := payload["constraints"].(map[string]any)
	constraints["excludedIngredients"] = []string{}
	constraints["takesMedication"] = true
	constraints["medications"] = "daily statin"
	profileID := upsertProfileForToken(t, client, server.URL, accessToken, payload)

	nutritionResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/profile/nutrition", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if nutritionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected nutrition profile status 200, got %d", nutritionResp.StatusCode)
	}
	nutritionPayload := decodeJSONBody(t, nutritionResp)
	derivedExcluded := nutritionPayload["derivedExcluded"].([]any)
	foundGrapefruit := false
	for _, item := range derivedExcluded {
		if strings.EqualFold(item.(string), "grapefruit") {
			foundGrapefruit = true
			break
		}
	}
	if !foundGrapefruit {
		t.Fatalf("expected medication rule to derive grapefruit exclusion, got %#v", nutritionPayload["derivedExcluded"])
	}
	metadata := nutritionPayload["metadata"].(map[string]any)
	matchedRuleCodes := metadata["matchedRuleCodes"].([]any)
	if len(matchedRuleCodes) != 1 || matchedRuleCodes[0].(string) != "statin_rule" {
		t.Fatalf("expected matchedRuleCodes to include statin_rule, got %#v", matchedRuleCodes)
	}

	recommendationResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if recommendationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected recommendation status 200, got %d", recommendationResp.StatusCode)
	}
	recommendationPayload := decodeJSONBody(t, recommendationResp)
	meals := recommendationPayload["meals"].([]any)
	if len(meals) != 1 {
		t.Fatalf("expected one safe recommendation for medication profile, got %#v", recommendationPayload["meals"])
	}
	if got := strings.TrimSpace(meals[0].(map[string]any)["id"].(string)); got != "502" {
		t.Fatalf("expected recipe 502 to remain after medication filtering, got %q", got)
	}

	traceResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/trace", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if traceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trace status 200, got %d", traceResp.StatusCode)
	}
	tracePayload := decodeJSONBody(t, traceResp)
	decisionSummary := tracePayload["decisionSummary"].(map[string]any)
	if decisionSummary["aiApplied"] != false {
		t.Fatalf("expected aiApplied=false for medication-sensitive profile, got %#v", decisionSummary["aiApplied"])
	}
}

func TestRouterReturnsNoSafeMatchesWhenAllCandidatesAreRejected(t *testing.T) {
	server, client, cfg := setupTestServer(t)
	accessToken := registerUser(t, client, server.URL, cfg, "No Match User", "nomatch@example.com")

	payload := defaultProfilePayload("No Match User")
	preferences := payload["preferences"].(map[string]any)
	preferences["likes"] = []string{"quinoa"}
	constraints := payload["constraints"].(map[string]any)
	constraints["excludedIngredients"] = []string{"chicken"}

	profileID := upsertProfileForToken(t, client, server.URL, accessToken, payload)

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected no-safe-match response to stay 200, got %d", resp.StatusCode)
	}
	payloadResp := decodeJSONBody(t, resp)
	meals, ok := payloadResp["meals"].([]any)
	if !ok || len(meals) != 0 {
		t.Fatalf("expected zero safe meals, got %#v", payloadResp["meals"])
	}

	traceResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/trace", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if traceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trace status 200, got %d", traceResp.StatusCode)
	}
	tracePayload := decodeJSONBody(t, traceResp)
	if got := strings.TrimSpace(tracePayload["status"].(string)); got != "no_matches" {
		t.Fatalf("expected no_matches status, got %q", got)
	}
}

func TestRouterGracefullyDegradesWhenRecipeProviderIsUnavailable(t *testing.T) {
	server, client, cfg := setupTestServerWithOptions(t, testServerOptions{
		searcher: &fakeRecipeSearcher{err: spoonacular.ErrUpstreamFailure},
	})
	accessToken := registerUser(t, client, server.URL, cfg, "Upstream User", "upstream@example.com")
	profileID := upsertProfileForToken(t, client, server.URL, accessToken, defaultProfilePayload("Upstream User"))

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected graceful degradation status 200, got %d", resp.StatusCode)
	}
	payloadResp := decodeJSONBody(t, resp)
	meals, ok := payloadResp["meals"].([]any)
	if !ok || len(meals) != 0 {
		t.Fatalf("expected zero meals on upstream outage, got %#v", payloadResp["meals"])
	}

	traceResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/trace", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if traceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trace status 200, got %d", traceResp.StatusCode)
	}
	tracePayload := decodeJSONBody(t, traceResp)
	if got := strings.TrimSpace(tracePayload["status"].(string)); got != "no_matches" {
		t.Fatalf("expected no_matches status on upstream outage, got %q", got)
	}
	externalTrace := tracePayload["externalTrace"].(map[string]any)
	strictTrace := externalTrace["strict_profile"].(map[string]any)
	if strictTrace["errorClass"] != "upstream_unavailable" {
		t.Fatalf("expected upstream_unavailable trace class, got %#v", strictTrace["errorClass"])
	}
}

func TestRouterGracefullyDegradesWhenAIIsUnavailable(t *testing.T) {
	server, client, cfg := setupTestServerWithOptions(t, testServerOptions{
		ai: &fakeRouteAITextGenerator{err: errors.New("ai unavailable")},
	})
	accessToken := registerUser(t, client, server.URL, cfg, "AI User", "ai@example.com")
	profileID := upsertProfileForToken(t, client, server.URL, accessToken, defaultProfilePayload("AI User"))

	resp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected recommendations status 200 when AI is unavailable, got %d", resp.StatusCode)
	}
	payloadResp := decodeJSONBody(t, resp)
	meals, ok := payloadResp["meals"].([]any)
	if !ok || len(meals) == 0 {
		t.Fatalf("expected deterministic recommendations without AI, got %#v", payloadResp["meals"])
	}

	traceResp := doJSON(t, client, http.MethodGet, server.URL+"/api/v1/recommendations/"+profileID+"/trace", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if traceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trace status 200, got %d", traceResp.StatusCode)
	}
	tracePayload := decodeJSONBody(t, traceResp)
	decisionSummary := tracePayload["decisionSummary"].(map[string]any)
	if decisionSummary["aiApplied"] != false {
		t.Fatalf("expected aiApplied=false when AI rerank fails, got %#v", decisionSummary["aiApplied"])
	}
}
