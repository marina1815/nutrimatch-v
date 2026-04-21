package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
)

type fakeUserRepository struct {
	user           *models.User
	updateFullName string
}

func (r *fakeUserRepository) Create(_ context.Context, user *models.User) error {
	r.user = user
	if r.user.ID == "" {
		r.user.ID = "user-1"
	}
	return nil
}

func (r *fakeUserRepository) GetByEmail(_ context.Context, _ string) (*models.User, error) {
	if r.user == nil {
		return nil, errors.New("not found")
	}
	return r.user, nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, _ string) (*models.User, error) {
	if r.user == nil {
		return nil, errors.New("not found")
	}
	return r.user, nil
}

func (r *fakeUserRepository) UpdateFullName(_ context.Context, _ string, fullName string) error {
	r.updateFullName = fullName
	return nil
}

type fakeProfileRepository struct {
	profile           *models.Profile
	lifestyle         *models.Lifestyle
	preferences       *models.Preferences
	constraints       *models.Constraints
	nutritionProfile  *models.NutritionProfile
	upsertProfile     bool
	upsertLifestyle   bool
	upsertPreferences bool
	upsertConstraints bool
	upsertNutrition   bool
}

type fakeMedicalRuleRepository struct {
	rules []models.MedicalRule
}

func (r *fakeMedicalRuleRepository) ListActive(_ context.Context) ([]models.MedicalRule, error) {
	return append([]models.MedicalRule{}, r.rules...), nil
}

type memoryTraceRepositoryForService struct {
	run        *models.RecommendationRun
	candidates []*models.RecommendationCandidate
}

func (r *memoryTraceRepositoryForService) CreateRun(_ context.Context, run *models.RecommendationRun) error {
	copied := *run
	r.run = &copied
	return nil
}

func (r *memoryTraceRepositoryForService) ReplaceCandidates(_ context.Context, _ string, candidates []*models.RecommendationCandidate) error {
	r.candidates = make([]*models.RecommendationCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		copied := *candidate
		r.candidates = append(r.candidates, &copied)
	}
	return nil
}

func (r *memoryTraceRepositoryForService) GetLatestRunByProfile(_ context.Context, _, _ string) (*models.RecommendationRun, []*models.RecommendationCandidate, error) {
	if r.run == nil {
		return nil, nil, errors.New("not found")
	}
	items := make([]*models.RecommendationCandidate, 0, len(r.candidates))
	for _, candidate := range r.candidates {
		copied := *candidate
		items = append(items, &copied)
	}
	copiedRun := *r.run
	return &copiedRun, items, nil
}

func (r *memoryTraceRepositoryForService) GetCandidateByRecipeID(_ context.Context, _, _, recipeID string) (*models.RecommendationCandidate, error) {
	for _, candidate := range r.candidates {
		if candidate.ExternalRecipeID == recipeID {
			copied := *candidate
			return &copied, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *fakeProfileRepository) UpsertProfile(_ context.Context, profile *models.Profile) error {
	r.upsertProfile = true
	r.profile = profile
	return nil
}

func (r *fakeProfileRepository) UpsertLifestyle(_ context.Context, lifestyle *models.Lifestyle) error {
	r.upsertLifestyle = true
	r.lifestyle = lifestyle
	return nil
}

func (r *fakeProfileRepository) UpsertPreferences(_ context.Context, preferences *models.Preferences) error {
	r.upsertPreferences = true
	r.preferences = preferences
	return nil
}

func (r *fakeProfileRepository) UpsertConstraints(_ context.Context, constraints *models.Constraints) error {
	r.upsertConstraints = true
	r.constraints = constraints
	return nil
}

func (r *fakeProfileRepository) GetProfile(_ context.Context, _ string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, error) {
	if r.profile == nil || r.lifestyle == nil || r.preferences == nil || r.constraints == nil {
		return nil, nil, nil, nil, errors.New("not found")
	}
	return r.profile, r.lifestyle, r.preferences, r.constraints, nil
}

func (r *fakeProfileRepository) ListProfileBundles(_ context.Context, _ string, _ int) ([]repository.ProfileBundle, error) {
	return nil, nil
}

func (r *fakeProfileRepository) UpsertNutritionProfile(_ context.Context, profile *models.NutritionProfile) error {
	r.upsertNutrition = true
	r.nutritionProfile = profile
	return nil
}

func (r *fakeProfileRepository) GetNutritionProfile(_ context.Context, _ string) (*models.NutritionProfile, error) {
	if r.nutritionProfile == nil {
		return nil, errors.New("not found")
	}
	return r.nutritionProfile, nil
}

type fakeSessionRepository struct{}

func (r *fakeSessionRepository) Create(_ context.Context, _ *models.Session) error { return nil }
func (r *fakeSessionRepository) GetByID(_ context.Context, _ string) (*models.Session, error) {
	return nil, errors.New("not found")
}
func (r *fakeSessionRepository) GetByRefreshHash(_ context.Context, _ string) (*models.Session, error) {
	return nil, errors.New("not found")
}
func (r *fakeSessionRepository) Rotate(_ context.Context, _, _ string, _, _ time.Time) error {
	return nil
}
func (r *fakeSessionRepository) Touch(_ context.Context, _ string, _ time.Time) error { return nil }
func (r *fakeSessionRepository) Revoke(_ context.Context, _ string) error             { return nil }

type fakeTxManager struct {
	called bool
	repos  repository.Repositories
}

func (m *fakeTxManager) WithinTransaction(_ context.Context, fn func(repository.Repositories) error) error {
	m.called = true
	return fn(m.repos)
}

type fakeRecipeSearcher struct {
	called bool
	resp   *spoonacular.SearchResponse
	err    error
}

type fakeAITextGenerator struct {
	text   string
	err    error
	prompt string
}

func (s *fakeRecipeSearcher) Search(_ context.Context, _ spoonacular.SearchOptions) (*spoonacular.SearchResponse, error) {
	s.called = true
	if s.err != nil {
		return nil, s.err
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &spoonacular.SearchResponse{}, nil
}

func (g *fakeAITextGenerator) GenerateText(_ context.Context, prompt string) (string, error) {
	g.prompt = prompt
	if g.err != nil {
		return "", g.err
	}
	return g.text, nil
}

func TestBuildQuery(t *testing.T) {
	query := buildQuery([]string{"oriental"}, []string{"chicken"})
	if query == "" {
		t.Fatalf("expected query")
	}
}

func TestMatchMedicalRulesFindsMedicationAndConditionRules(t *testing.T) {
	rules := []models.MedicalRule{
		{Code: "diabetes_rule", ConditionKey: "diabetes"},
		{Code: "statin_rule", MedicationPattern: "statin"},
	}

	matched := MatchMedicalRules(rules, &models.Constraints{
		Conditions:  models.StringSlice{"diabetes"},
		Medications: "daily statin",
	})

	if len(matched) != 2 {
		t.Fatalf("expected two matched medical rules, got %d", len(matched))
	}
}

func TestProfileServiceUpsertUsesTransactionManager(t *testing.T) {
	userRepo := &fakeUserRepository{user: &models.User{ID: "user-1", FullName: "Existing"}}
	profileRepo := &fakeProfileRepository{}
	txManager := &fakeTxManager{
		repos: repository.Repositories{
			Users:    userRepo,
			Profiles: profileRepo,
			Sessions: &fakeSessionRepository{},
		},
	}

	service := &ProfileService{
		Users:     userRepo,
		Profiles:  profileRepo,
		TxManager: txManager,
	}

	err := service.Upsert(
		context.Background(),
		"user-1",
		&models.Profile{},
		&models.Lifestyle{},
		&models.Preferences{},
		&models.Constraints{},
		"Updated User",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !txManager.called {
		t.Fatalf("expected transaction manager to be used")
	}
	if userRepo.updateFullName != "Updated User" {
		t.Fatalf("expected full name update inside transaction")
	}
	if !profileRepo.upsertProfile || !profileRepo.upsertLifestyle || !profileRepo.upsertPreferences || !profileRepo.upsertConstraints {
		t.Fatalf("expected all profile repositories to be updated inside transaction")
	}
}

func TestRecommendationServiceRejectsForeignProfileID(t *testing.T) {
	userRepo := &fakeUserRepository{user: &models.User{ID: "user-1", FullName: "User"}}
	profileRepo := &fakeProfileRepository{
		profile:     &models.Profile{ID: "owned-profile", UserID: "user-1", Age: 25},
		lifestyle:   &models.Lifestyle{UserID: "user-1", Goal: "weight_loss", ActivityLevel: "light"},
		preferences: &models.Preferences{UserID: "user-1", MealsPerDay: 3},
		constraints: &models.Constraints{UserID: "user-1"},
	}
	searcher := &fakeRecipeSearcher{}
	service := &RecommendationService{
		Profiles: &ProfileService{Users: userRepo, Profiles: profileRepo},
		Recipes:  searcher,
	}

	_, err := service.GetRecommendations(context.Background(), "user-1", "other-profile", "req-1")
	if !errors.Is(err, ErrProfileAccessDenied) {
		t.Fatalf("expected ErrProfileAccessDenied, got %v", err)
	}
	if searcher.called {
		t.Fatalf("expected recipe search not to run for foreign profile IDs")
	}
}

func TestEvaluateCandidateRejectsMissingRequiredMedicalTag(t *testing.T) {
	service := &RecommendationService{}
	candidate := service.evaluateCandidate(
		"run-1",
		"user-1",
		"profile-1",
		&models.Lifestyle{Goal: "medical_diet", ActivityLevel: "light"},
		&models.Preferences{},
		&models.Constraints{},
		&models.NutritionProfile{
			MaxMealCalories:    900,
			MinProteinPerMeal:  10,
			MaxCarbsPerMeal:    100,
			MaxFatPerMeal:      50,
			MaxSugarPerMeal:    30,
			MaxSodiumMgPerMeal: 1200,
		},
		[]models.MedicalRule{
			{
				Code:         "diabetes_rule",
				RequiredTags: models.StringSlice{"high-protein"},
			},
		},
		&SimilaritySignals{},
		enrichedRecipe{
			recipe: spoonacular.Recipe{
				ID:      10,
				Title:   "Vegetable salad",
				Summary: "Fresh vegetables",
				Nutrition: spoonacular.Nutrition{
					Nutrients: []spoonacular.Nutrient{
						{Name: "Calories", Amount: 320},
						{Name: "Protein", Amount: 8},
						{Name: "Carbohydrates", Amount: 22},
						{Name: "Fat", Amount: 10},
						{Name: "Sugar", Amount: 6},
						{Name: "Sodium", Amount: 250},
					},
				},
				ExtendedIngredients: []spoonacular.Ingredient{{Name: "lettuce"}},
			},
		},
	)

	if candidate.Accepted {
		t.Fatalf("expected candidate to be rejected when required tag is missing")
	}
	if len(candidate.RejectedReasons) == 0 {
		t.Fatalf("expected rejection reasons")
	}
}

func TestEvaluateCandidateRejectsMedicalProteinCeiling(t *testing.T) {
	service := &RecommendationService{}
	candidate := service.evaluateCandidate(
		"run-1",
		"user-1",
		"profile-1",
		&models.Lifestyle{Goal: "medical_diet", ActivityLevel: "light"},
		&models.Preferences{},
		&models.Constraints{},
		&models.NutritionProfile{
			MaxMealCalories:    900,
			MinProteinPerMeal:  10,
			MaxCarbsPerMeal:    100,
			MaxFatPerMeal:      50,
			MaxSugarPerMeal:    30,
			MaxSodiumMgPerMeal: 1200,
		},
		[]models.MedicalRule{
			{
				Code:            "renal_rule",
				MaxProteinGrams: 28,
			},
		},
		&SimilaritySignals{},
		enrichedRecipe{
			recipe: spoonacular.Recipe{
				ID:      11,
				Title:   "Chicken bowl",
				Summary: "High protein meal",
				Nutrition: spoonacular.Nutrition{
					Nutrients: []spoonacular.Nutrient{
						{Name: "Calories", Amount: 480},
						{Name: "Protein", Amount: 38},
						{Name: "Carbohydrates", Amount: 25},
						{Name: "Fat", Amount: 14},
						{Name: "Sugar", Amount: 4},
						{Name: "Sodium", Amount: 300},
					},
				},
				ExtendedIngredients: []spoonacular.Ingredient{{Name: "chicken"}},
			},
		},
	)

	if candidate.Accepted {
		t.Fatalf("expected candidate to be rejected when medical protein ceiling is exceeded")
	}
}

func TestApplyAIRerankValidatesIDsAndPreservesDeterministicExplanation(t *testing.T) {
	ai := &fakeAITextGenerator{
		text: `[{"id":"meal-1","confidenceBonus":3.2,"explanation":"Closer to stated healthy preferences."},{"id":"ghost","confidenceBonus":5,"explanation":"Should be ignored"}]`,
	}
	service := &RecommendationService{AI: ai}
	candidates := []*models.RecommendationCandidate{
		{
			ExternalRecipeID: "meal-1",
			Title:            "Chicken Bowl",
			FinalScore:       50,
			Explanation:      "Selected because passes deterministic profile validation",
			Tags:             models.StringSlice{"healthy", "balanced"},
			SourceProvenance: models.JSONMap{},
		},
	}

	service.applyAIRerank(context.Background(), &models.Lifestyle{
		Goal:          "weight_loss",
		ActivityLevel: "light",
	}, &models.Preferences{
		Likes:      models.StringSlice{"chicken", "quinoa"},
		MealStyles: models.StringSlice{"healthy"},
	}, candidates)

	if candidates[0].FinalScore != 53.2 {
		t.Fatalf("expected validated AI bonus to be applied, got %v", candidates[0].FinalScore)
	}
	if !strings.Contains(candidates[0].Explanation, "Selected because") || !strings.Contains(candidates[0].Explanation, "AI rerank note:") {
		t.Fatalf("expected deterministic explanation to be preserved and augmented, got %q", candidates[0].Explanation)
	}
	if _, ok := candidates[0].SourceProvenance["aiRerank"]; !ok {
		t.Fatalf("expected validated AI provenance metadata")
	}
	if strings.Contains(strings.ToLower(ai.prompt), "medication") || strings.Contains(strings.ToLower(ai.prompt), "condition") {
		t.Fatalf("expected minimized prompt without sensitive health details, got %q", ai.prompt)
	}
}

func TestApplyAIRerankReturnsFalseWhenAIUnavailable(t *testing.T) {
	ai := &fakeAITextGenerator{err: errors.New("upstream unavailable")}
	service := &RecommendationService{AI: ai}
	candidates := []*models.RecommendationCandidate{
		{
			ExternalRecipeID: "meal-1",
			Title:            "Chicken Bowl",
			FinalScore:       50,
			Explanation:      "Selected because passes deterministic profile validation",
			SourceProvenance: models.JSONMap{},
		},
	}

	applied := service.applyAIRerank(context.Background(), &models.Lifestyle{
		Goal:          "weight_loss",
		ActivityLevel: "light",
	}, &models.Preferences{
		Likes:      models.StringSlice{"chicken"},
		MealStyles: models.StringSlice{"healthy"},
	}, candidates)

	if applied {
		t.Fatalf("expected ai rerank to report false when AI is unavailable")
	}
	if candidates[0].FinalScore != 50 {
		t.Fatalf("expected deterministic score to remain unchanged, got %v", candidates[0].FinalScore)
	}
}

func TestGetRecommendationsGracefullyHandlesRecipeUpstreamFailure(t *testing.T) {
	userRepo := &fakeUserRepository{user: &models.User{ID: "user-1", FullName: "User"}}
	profileRepo := &fakeProfileRepository{
		profile:     &models.Profile{ID: "profile-1", UserID: "user-1", Age: 25},
		lifestyle:   &models.Lifestyle{UserID: "user-1", Goal: "weight_loss", ActivityLevel: "light", MaxReadyTime: 30},
		preferences: &models.Preferences{UserID: "user-1", MealsPerDay: 3},
		constraints: &models.Constraints{UserID: "user-1"},
		nutritionProfile: &models.NutritionProfile{
			ID:                 "nutrition-1",
			UserID:             "user-1",
			ProfileID:          "profile-1",
			CalculatedAt:       time.Now(),
			MaxMealCalories:    800,
			MinProteinPerMeal:  10,
			MaxCarbsPerMeal:    100,
			MaxFatPerMeal:      50,
			MaxSugarPerMeal:    30,
			MaxSodiumMgPerMeal: 1200,
		},
	}
	traceRepo := &memoryTraceRepositoryForService{}
	searcher := &fakeRecipeSearcher{err: spoonacular.ErrUpstreamFailure}
	service := &RecommendationService{
		Profiles:     &ProfileService{Users: userRepo, Profiles: profileRepo},
		Recipes:      searcher,
		MedicalRules: &fakeMedicalRuleRepository{},
		Traces:       traceRepo,
	}

	response, err := service.GetRecommendations(context.Background(), "user-1", "profile-1", "req-1")
	if err != nil {
		t.Fatalf("expected graceful no-match response on upstream failure, got %v", err)
	}
	if len(response.Meals) != 0 {
		t.Fatalf("expected no meals when upstream is unavailable, got %d", len(response.Meals))
	}
	if traceRepo.run == nil || traceRepo.run.Status != "no_matches" {
		t.Fatalf("expected persisted no_matches run, got %+v", traceRepo.run)
	}
	if traceRepo.run.ExternalTrace == nil || traceRepo.run.ExternalTrace["strict_profile"] == nil {
		t.Fatalf("expected external trace to capture upstream failure")
	}
}

func TestApplyAIRerankClampsBonusAndSanitizesExplanation(t *testing.T) {
	ai := &fakeAITextGenerator{
		text: `[{"id":"meal-2","confidenceBonus":9,"explanation":"  Strong macro fit.\n\nKeeps protein high while staying balanced and quick for the user across the whole week without introducing any unsafe reasoning or extra meals that were not approved.  "}]`,
	}
	service := &RecommendationService{AI: ai}
	candidates := []*models.RecommendationCandidate{
		{
			ExternalRecipeID: "meal-2",
			Title:            "Quinoa Salad",
			FinalScore:       40,
			Explanation:      "Selected because passes deterministic profile validation",
			SourceProvenance: models.JSONMap{},
		},
	}

	service.applyAIRerank(context.Background(), &models.Lifestyle{
		Goal:          "energy_maintenance",
		ActivityLevel: "moderate",
	}, &models.Preferences{}, candidates)

	if candidates[0].FinalScore != 45 {
		t.Fatalf("expected AI bonus to be clamped to 5, got %v", candidates[0].FinalScore)
	}
	if strings.Contains(candidates[0].Explanation, "\n") {
		t.Fatalf("expected AI explanation to be sanitized onto one line")
	}
}

func TestBuildExternalSearchTraceSanitizesSearchDetails(t *testing.T) {
	trace := buildExternalSearchTrace(spoonacular.SearchOptions{
		Query:              "healthy chicken quinoa",
		IncludeIngredients: []string{"chicken", "quinoa"},
		ExcludeIngredients: []string{"bacon"},
		Intolerances:       []string{"dairy"},
		MaxCalories:        650,
	}, &spoonacular.SearchResponse{
		Results:  []spoonacular.Recipe{{ID: 1}},
		CacheHit: true,
	}, nil, 120*time.Millisecond)

	if trace["provider"] != "spoonacular" {
		t.Fatalf("expected spoonacular provider trace")
	}
	if trace["queryPresent"] != true {
		t.Fatalf("expected query presence flag")
	}
	if trace["includeCount"] != 2 || trace["excludeCount"] != 1 || trace["intoleranceCount"] != 1 {
		t.Fatalf("expected only counts to be stored in trace, got %+v", trace)
	}
	if _, exists := trace["query"]; exists {
		t.Fatalf("expected raw query not to be stored in trace")
	}
	if _, exists := trace["include"]; exists {
		t.Fatalf("expected raw include list not to be stored in trace")
	}
	if trace["cacheHit"] != true {
		t.Fatalf("expected cache hit to be recorded")
	}
}

func TestEnrichRecipesFromSearchPlanCarriesPlanAndCacheMetadata(t *testing.T) {
	items := enrichRecipesFromSearchPlan("strict_profile", &spoonacular.SearchResponse{
		CacheHit: true,
		Results: []spoonacular.Recipe{
			{ID: 1, Title: "Chicken Bowl"},
		},
	})

	if len(items) != 1 {
		t.Fatalf("expected one enriched recipe, got %d", len(items))
	}
	if len(items[0].sourcePlans) != 1 || items[0].sourcePlans[0] != "strict_profile" {
		t.Fatalf("expected source plan provenance, got %+v", items[0].sourcePlans)
	}
	if len(items[0].cacheSources) != 1 {
		t.Fatalf("expected cache provenance to be carried when response is cached")
	}
}

func TestEvaluateCandidateCarriesEnrichmentProvenance(t *testing.T) {
	service := &RecommendationService{}
	candidate := service.evaluateCandidate(
		"run-1",
		"user-1",
		"profile-1",
		&models.Lifestyle{Goal: "weight_loss", ActivityLevel: "light"},
		&models.Preferences{Likes: models.StringSlice{"chicken"}},
		&models.Constraints{},
		&models.NutritionProfile{
			MaxMealCalories:       900,
			MinProteinPerMeal:     10,
			MaxCarbsPerMeal:       100,
			MaxFatPerMeal:         50,
			MaxSugarPerMeal:       30,
			MaxSodiumMgPerMeal:    1200,
			RecommendedMealStyles: models.StringSlice{"healthy"},
		},
		nil,
		&SimilaritySignals{},
		enrichedRecipe{
			recipe: spoonacular.Recipe{
				ID:      44,
				Title:   "Chicken Bowl",
				Summary: "Grilled chicken and quinoa.",
				Nutrition: spoonacular.Nutrition{
					Nutrients: []spoonacular.Nutrient{
						{Name: "Calories", Amount: 520},
						{Name: "Protein", Amount: 36},
						{Name: "Carbohydrates", Amount: 32},
						{Name: "Fat", Amount: 12},
						{Name: "Sugar", Amount: 6},
						{Name: "Sodium", Amount: 300},
					},
				},
				ExtendedIngredients: []spoonacular.Ingredient{{Name: "chicken"}, {Name: "quinoa"}},
			},
			sourcePlans:  []string{"strict_profile", "goal_balanced"},
			cacheSources: []string{"persistent_or_memory_cache"},
		},
	)

	searchPlans, ok := candidate.SourceProvenance["searchPlans"].([]string)
	if !ok || len(searchPlans) != 2 {
		t.Fatalf("expected search plan provenance on candidate, got %+v", candidate.SourceProvenance["searchPlans"])
	}
	if candidate.SourceProvenance["enrichedFacts"] == nil {
		t.Fatalf("expected enriched facts provenance on candidate")
	}
}

func TestEvaluateHardFiltersSeparatesHardRejects(t *testing.T) {
	result := evaluateHardFilters(
		&models.Preferences{MealStyles: models.StringSlice{"healthy"}},
		&models.Constraints{
			Allergies:           models.StringSlice{"dairy"},
			ExcludedIngredients: models.StringSlice{"bacon"},
		},
		&models.NutritionProfile{
			MaxMealCalories:    700,
			MinProteinPerMeal:  20,
			MaxCarbsPerMeal:    60,
			MaxFatPerMeal:      20,
			MaxSugarPerMeal:    18,
			MaxSodiumMgPerMeal: 700,
			DerivedExcluded:    models.StringSlice{"sausage"},
		},
		[]models.MedicalRule{
			{Code: "hypertension_rule", BlockedTags: models.StringSlice{"salty"}},
		},
		candidateFacts{
			ingredients: []string{"bacon"},
			baseTags:    []string{"salty"},
			calories:    500,
			protein:     24,
			carbs:       30,
			fat:         18,
			sugar:       6,
			sodium:      400,
		},
	)

	if len(result.rejectedReasons) < 2 {
		t.Fatalf("expected multiple hard filter rejections, got %v", result.rejectedReasons)
	}
	if result.filterDecisions["blockedIngredients"] == nil {
		t.Fatalf("expected blocked ingredients to be recorded in filter decisions")
	}
}

func TestComputeDeterministicScoreRunsOnlyAfterHardFilterPass(t *testing.T) {
	profile := &models.NutritionProfile{
		MaxMealCalories:       700,
		MinProteinPerMeal:     20,
		MaxCarbsPerMeal:       60,
		MaxFatPerMeal:         20,
		RecommendedMealStyles: models.StringSlice{"healthy", "balanced"},
	}
	preferences := &models.Preferences{Likes: models.StringSlice{"chicken"}}
	signals := &SimilaritySignals{Likes: []string{"quinoa"}}
	facts := candidateFacts{
		ingredients: []string{"chicken", "quinoa"},
		baseTags:    []string{"healthy", "balanced"},
		calories:    520,
		protein:     32,
		carbs:       42,
		fat:         14,
	}

	blocked := computeDeterministicScore(preferences, profile, signals, facts, false)
	if blocked.score != 0 || len(blocked.acceptedReasons) != 0 {
		t.Fatalf("expected no deterministic score when hard filters fail, got score=%v reasons=%v", blocked.score, blocked.acceptedReasons)
	}

	passed := computeDeterministicScore(preferences, profile, signals, facts, true)
	if passed.score <= 40 {
		t.Fatalf("expected positive deterministic score uplift after hard filters pass, got %v", passed.score)
	}
	if len(passed.acceptedReasons) == 0 {
		t.Fatalf("expected accepted reasons after deterministic scoring")
	}
}
