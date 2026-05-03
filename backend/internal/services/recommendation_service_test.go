package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/marina1815/nutrimatch/internal/catalog"
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

func (r *fakeUserRepository) UpdatePasswordHash(_ context.Context, _ string, passwordHash string) error {
	if r.user != nil {
		r.user.PasswordHash = passwordHash
	}
	return nil
}

func (r *fakeUserRepository) UpdatePreferredMFAMethod(_ context.Context, _ string, method string) error {
	if r.user != nil {
		r.user.PreferredMFAMethod = method
	}
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
func (r *fakeSessionRepository) Rotate(_ context.Context, _, _, _, _ string, _, _ time.Time) error {
	return nil
}
func (r *fakeSessionRepository) Touch(_ context.Context, _ string, _ time.Time) error { return nil }
func (r *fakeSessionRepository) Revoke(_ context.Context, _ string) error             { return nil }
func (r *fakeSessionRepository) RevokeForUser(_ context.Context, _, _ string) error   { return nil }
func (r *fakeSessionRepository) RevokeOthers(_ context.Context, _, _ string) error    { return nil }
func (r *fakeSessionRepository) ListByUser(_ context.Context, _ string, _ int) ([]models.Session, error) {
	return nil, nil
}

type fakeTxManager struct {
	called bool
	repos  repository.Repositories
}

func (m *fakeTxManager) WithinTransaction(_ context.Context, fn func(repository.Repositories) error) error {
	m.called = true
	return fn(m.repos)
}

type fakeLocalRecipeRepositoryForService struct {
	called     bool
	candidates []repository.LocalRecipeCandidate
	err        error
}

type fakeAITextGenerator struct {
	text    string
	texts   []string
	err     error
	errs    []error
	prompt  string
	prompts []string
}

func (r *fakeLocalRecipeRepositoryForService) Search(_ context.Context, _ repository.LocalRecipeQuery) ([]repository.LocalRecipeCandidate, error) {
	r.called = true
	if r.err != nil {
		return nil, r.err
	}
	if len(r.candidates) > 0 {
		return r.candidates, nil
	}
	return []repository.LocalRecipeCandidate{{
		ID:          "101",
		Title:       "Chicken Quinoa Bowl",
		Ingredients: []string{"chicken", "quinoa"},
		Calories:    520,
		Protein:     42,
		Carbs:       42,
		Fat:         14,
		Sugar:       8,
		SodiumMg:    380,
		Score:       50,
	}}, nil
}

func (r *fakeLocalRecipeRepositoryForService) SuggestIngredients(_ context.Context, _ string, _ int) ([]repository.CatalogOption, error) {
	return []repository.CatalogOption{}, nil
}

func (r *fakeLocalRecipeRepositoryForService) ListAllergies(_ context.Context) ([]repository.CatalogOption, error) {
	return []repository.CatalogOption{}, nil
}

func (g *fakeAITextGenerator) GenerateText(_ context.Context, prompt string) (string, error) {
	g.prompt = prompt
	g.prompts = append(g.prompts, prompt)
	if len(g.errs) > 0 {
		err := g.errs[0]
		g.errs = g.errs[1:]
		if err != nil {
			return "", err
		}
	}
	if g.err != nil {
		return "", g.err
	}
	if len(g.texts) > 0 {
		text := g.texts[0]
		g.texts = g.texts[1:]
		return text, nil
	}
	return g.text, nil
}

func TestBuildQuery(t *testing.T) {
	query := buildQuery([]string{"oriental"}, []string{"chicken"})
	if query == "" {
		t.Fatalf("expected query")
	}
}

func TestFallbackSearchPlanKeepsSafetyExclusionsOnly(t *testing.T) {
	plans := buildSearchPlans(
		&models.Preferences{
			Likes:    models.StringSlice{"chicken"},
			Dislikes: models.StringSlice{"mushroom"},
		},
		&models.Constraints{
			Allergies: models.StringSlice{"peanut"},
		},
		&models.NutritionProfile{
			DerivedExcluded: models.StringSlice{"grapefruit"},
		},
		&SimilaritySignals{},
	)

	var fallback searchPlan
	for _, plan := range plans {
		if plan.Name == "fallback_goal_candidates" {
			fallback = plan
			break
		}
	}
	if !fallback.Fallback {
		t.Fatalf("expected fallback plan to be present")
	}
	if containsString(fallback.Exclude, "mushroom") {
		t.Fatalf("expected fallback to drop soft dislikes from provider search")
	}
	if !containsString(fallback.Exclude, "peanut") || !containsString(fallback.Exclude, "grapefruit") {
		t.Fatalf("expected fallback to keep safety exclusions, got %v", fallback.Exclude)
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
		preferences: &models.Preferences{UserID: "user-1"},
		constraints: &models.Constraints{UserID: "user-1"},
	}
	localRecipes := &fakeLocalRecipeRepositoryForService{}
	service := &RecommendationService{
		Profiles:     &ProfileService{Users: userRepo, Profiles: profileRepo},
		LocalRecipes: localRecipes,
	}

	_, err := service.GetRecommendations(context.Background(), "user-1", "other-profile", "req-1")
	if !errors.Is(err, ErrProfileAccessDenied) {
		t.Fatalf("expected ErrProfileAccessDenied, got %v", err)
	}
	if localRecipes.called {
		t.Fatalf("expected local recipe search not to run for foreign profile IDs")
	}
}

func TestEvaluateCandidateRejectsMissingRequiredMedicalTag(t *testing.T) {
	service := &RecommendationService{}
	candidate := service.evaluateCandidate(
		context.Background(),
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
			recipe: catalog.Recipe{
				ID:      10,
				Title:   "Vegetable salad",
				Summary: "Fresh vegetables",
				Nutrition: catalog.Nutrition{
					Nutrients: []catalog.Nutrient{
						{Name: "Calories", Amount: 320},
						{Name: "Protein", Amount: 8},
						{Name: "Carbohydrates", Amount: 22},
						{Name: "Fat", Amount: 10},
						{Name: "Sugar", Amount: 6},
						{Name: "Sodium", Amount: 250},
					},
				},
				ExtendedIngredients: []catalog.Ingredient{{Name: "lettuce"}},
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
		context.Background(),
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
			recipe: catalog.Recipe{
				ID:      11,
				Title:   "Chicken bowl",
				Summary: "High protein meal",
				Nutrition: catalog.Nutrition{
					Nutrients: []catalog.Nutrient{
						{Name: "Calories", Amount: 480},
						{Name: "Protein", Amount: 38},
						{Name: "Carbohydrates", Amount: 25},
						{Name: "Fat", Amount: 14},
						{Name: "Sugar", Amount: 4},
						{Name: "Sodium", Amount: 300},
					},
				},
				ExtendedIngredients: []catalog.Ingredient{{Name: "chicken"}},
			},
		},
	)

	if candidate.Accepted {
		t.Fatalf("expected candidate to be rejected when medical protein ceiling is exceeded")
	}
}

func TestEvaluateCandidateRejectsHypercholesterolemiaAndDigestiveTags(t *testing.T) {
	service := &RecommendationService{}
	profile := &models.NutritionProfile{
		MaxMealCalories:    1200,
		MinProteinPerMeal:  0,
		MaxCarbsPerMeal:    140,
		MaxFatPerMeal:      80,
		MaxSugarPerMeal:    60,
		MaxSodiumMgPerMeal: 1500,
	}

	cholesterolCandidate := service.evaluateCandidate(
		context.Background(),
		"run-1",
		"user-1",
		"profile-1",
		&models.Lifestyle{Goal: "medical_diet", ActivityLevel: "light"},
		&models.Preferences{},
		&models.Constraints{},
		profile,
		[]models.MedicalRule{
			{
				Code:        "hypercholesterolemia_lipid_control",
				BlockedTags: models.StringSlice{"cholesterol-risk", "saturated-fat"},
			},
		},
		&SimilaritySignals{},
		enrichedRecipe{
			recipe: catalog.Recipe{
				ID:      12,
				Title:   "Creamy bacon pasta",
				Summary: "Butter, cream and bacon.",
				Nutrition: catalog.Nutrition{Nutrients: []catalog.Nutrient{
					{Name: "Calories", Amount: 780},
					{Name: "Protein", Amount: 22},
					{Name: "Carbohydrates", Amount: 70},
					{Name: "Fat", Amount: 32},
					{Name: "Sugar", Amount: 7},
					{Name: "Sodium", Amount: 820},
				}},
				ExtendedIngredients: []catalog.Ingredient{{Name: "bacon"}, {Name: "cream"}, {Name: "butter"}},
			},
		},
	)
	if cholesterolCandidate.Accepted {
		t.Fatalf("expected cholesterol-risk candidate to be rejected")
	}

	digestiveCandidate := service.evaluateCandidate(
		context.Background(),
		"run-2",
		"user-1",
		"profile-1",
		&models.Lifestyle{Goal: "medical_diet", ActivityLevel: "light"},
		&models.Preferences{},
		&models.Constraints{},
		profile,
		[]models.MedicalRule{
			{
				Code:        "digestive_sensitivity_gentle_control",
				BlockedTags: models.StringSlice{"digestive-risk", "gas-forming", "very-spicy"},
			},
		},
		&SimilaritySignals{},
		enrichedRecipe{
			recipe: catalog.Recipe{
				ID:      13,
				Title:   "Spicy bean cabbage bowl",
				Summary: "Green chili, beans and cabbage.",
				Nutrition: catalog.Nutrition{Nutrients: []catalog.Nutrient{
					{Name: "Calories", Amount: 440},
					{Name: "Protein", Amount: 20},
					{Name: "Carbohydrates", Amount: 48},
					{Name: "Fat", Amount: 12},
					{Name: "Sugar", Amount: 5},
					{Name: "Sodium", Amount: 430},
				}},
				ExtendedIngredients: []catalog.Ingredient{{Name: "beans"}, {Name: "cabbage"}, {Name: "green chili"}},
			},
		},
	)
	if digestiveCandidate.Accepted {
		t.Fatalf("expected digestive-risk candidate to be rejected")
	}
}

func TestSelectDailyCandidatesUsesOnlySafePoolAndPreservesScores(t *testing.T) {
	now := time.Date(2026, 5, 3, 8, 0, 0, 0, time.UTC)
	candidates := []*models.RecommendationCandidate{
		dailyCandidate("meal-1", 50),
		dailyCandidate("meal-2", 40),
		dailyCandidate("meal-3", 30),
		dailyCandidate("meal-4", 20),
		dailyCandidate("meal-5", 10),
	}
	seed := stableSeed("user-1", "profile-1", now.Format("2006-01-02"))
	expectedPoolOrder := deterministicWeightedPick(candidates, len(candidates), seed)
	items := make([]string, 0, len(expectedPoolOrder))
	for _, candidate := range expectedPoolOrder {
		items = append(items, `{"mealId":"`+candidate.ExternalRecipeID+`","recommended":true,"rejectionReason":"","explanation":"Explication valide en francais."}`)
	}
	service := &RecommendationService{AI: &fakeAITextGenerator{text: `[` + strings.Join(items, ",") + `]`}}

	selected, result, mode := service.selectDailyCandidatesAndExplain(context.Background(), "user-1", "profile-1", candidates, now)

	if mode != "backend_random_ai_validated" || !result.Applied || !result.ValidationApplied || result.IgnoredReason != "" || result.SkippedReason != "" {
		t.Fatalf("expected backend-random AI validation, got mode=%s result=%+v", mode, result)
	}
	if len(selected) != len(candidates) {
		t.Fatalf("expected %d selected meals, got %d", len(candidates), len(selected))
	}
	for i, candidate := range selected {
		if candidate.ExternalRecipeID != expectedPoolOrder[i].ExternalRecipeID {
			t.Fatalf("expected backend pool order at %d to stay %q, got %q", i, expectedPoolOrder[i].ExternalRecipeID, candidate.ExternalRecipeID)
		}
		if candidate.FinalScore != expectedPoolOrder[i].FinalScore {
			t.Fatalf("expected score to stay unchanged for %s", candidate.ExternalRecipeID)
		}
		if aiExplanationFromProvenance(candidate.SourceProvenance) == "" {
			t.Fatalf("expected stored AI explanation for %s", candidate.ExternalRecipeID)
		}
	}
}

func TestSelectDailyCandidatesReplacesAIRejectedMealsFromBackendSafePool(t *testing.T) {
	now := time.Date(2026, 5, 3, 8, 0, 0, 0, time.UTC)
	candidates := make([]*models.RecommendationCandidate, 0, 21)
	for i := 1; i <= 21; i++ {
		candidates = append(candidates, dailyCandidate(fmt.Sprintf("meal-%02d", i), float64(60-i)))
	}
	seed := stableSeed("user-1", "profile-1", now.Format("2006-01-02"))
	pool := deterministicWeightedPick(candidates, len(candidates), seed)
	firstBatch := pool[:dailyRecommendationCount]
	rejectedID := firstBatch[0].ExternalRecipeID
	replacementID := pool[dailyRecommendationCount].ExternalRecipeID

	firstItems := make([]string, 0, len(firstBatch))
	for _, candidate := range firstBatch {
		if candidate.ExternalRecipeID == rejectedID {
			firstItems = append(firstItems, `{"mealId":"`+candidate.ExternalRecipeID+`","recommended":false,"rejectionReason":"Risque detecte par double validation.","explanation":""}`)
			continue
		}
		firstItems = append(firstItems, `{"mealId":"`+candidate.ExternalRecipeID+`","recommended":true,"rejectionReason":"","explanation":"Explication valide en francais."}`)
	}
	secondItems := `{"mealId":"` + replacementID + `","recommended":true,"rejectionReason":"","explanation":"Remplacement valide en francais."}`
	service := &RecommendationService{AI: &fakeAITextGenerator{texts: []string{
		`[` + strings.Join(firstItems, ",") + `]`,
		`[` + secondItems + `]`,
	}}}

	selected, result, mode := service.selectDailyCandidatesAndExplain(context.Background(), "user-1", "profile-1", candidates, now)

	if mode != "backend_random_ai_validated" || !result.Applied || !result.ValidationApplied {
		t.Fatalf("expected full double validation, got mode=%s result=%+v", mode, result)
	}
	if result.RejectedMealCount != 1 || result.ReplacementCount != 1 {
		t.Fatalf("expected one rejection and one replacement, got %+v", result)
	}
	if len(selected) != dailyRecommendationCount {
		t.Fatalf("expected %d selected meals, got %d", dailyRecommendationCount, len(selected))
	}
	if containsCandidateID(selected, rejectedID) {
		t.Fatalf("AI-rejected meal %s must not remain selected", rejectedID)
	}
	if !containsCandidateID(selected, replacementID) {
		t.Fatalf("expected backend-safe replacement %s in final set", replacementID)
	}
}

func TestSelectDailyCandidatesRejectsInvalidAIIDsAndFallsBack(t *testing.T) {
	now := time.Date(2026, 5, 3, 8, 0, 0, 0, time.UTC)
	candidates := []*models.RecommendationCandidate{
		dailyCandidate("meal-1", 50),
		dailyCandidate("meal-2", 40),
		dailyCandidate("meal-3", 30),
	}
	service := &RecommendationService{AI: &fakeAITextGenerator{
		text: `[{"mealId":"meal-1","recommended":true,"rejectionReason":"","explanation":"OK."},{"mealId":"meal-2","recommended":true,"rejectionReason":"","explanation":"OK."},{"mealId":"unsafe-new-meal","recommended":true,"rejectionReason":"","explanation":"Try to inject."}]`,
	}}

	selected, result, mode := service.selectDailyCandidatesAndExplain(context.Background(), "user-1", "profile-1", candidates, now)

	if mode != "backend_random_ai_unavailable" || result.IgnoredReason == "" || result.Applied {
		t.Fatalf("expected backend fallback on invalid AI id, got mode=%s result=%+v", mode, result)
	}
	if len(selected) != len(candidates) {
		t.Fatalf("expected fallback to keep safe target count, got %d", len(selected))
	}
	for _, candidate := range selected {
		if candidate.ExternalRecipeID == "unsafe-new-meal" {
			t.Fatalf("invalid AI meal id must never be selected")
		}
		if candidate.FinalScore == 0 || aiExplanationFromProvenance(candidate.SourceProvenance) != "" {
			t.Fatalf("expected deterministic fallback without local explanation or score mutation for %+v", candidate)
		}
	}
}

func TestGetRecommendationsUsesLocalCatalogWithoutExternalProvider(t *testing.T) {
	userRepo := &fakeUserRepository{user: &models.User{ID: "user-1", FullName: "User"}}
	profileRepo := &fakeProfileRepository{
		profile:     &models.Profile{ID: "profile-1", UserID: "user-1", Age: 25},
		lifestyle:   &models.Lifestyle{UserID: "user-1", Goal: "weight_loss", ActivityLevel: "light"},
		preferences: &models.Preferences{UserID: "user-1"},
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
	service := &RecommendationService{
		Profiles:     &ProfileService{Users: userRepo, Profiles: profileRepo},
		LocalRecipes: &fakeLocalRecipeRepositoryForService{},
		MedicalRules: &fakeMedicalRuleRepository{},
		Traces:       traceRepo,
	}

	response, err := service.GetRecommendations(context.Background(), "user-1", "profile-1", "req-1")
	if err != nil {
		t.Fatalf("expected recommendations from local catalog, got %v", err)
	}
	if len(response.Meals) == 0 {
		t.Fatalf("expected local catalog meals")
	}
	if response.Meals[0].Source != "local_catalog" {
		t.Fatalf("expected local catalog source, got %q", response.Meals[0].Source)
	}
	if traceRepo.run == nil || traceRepo.run.Status != "completed" {
		t.Fatalf("expected persisted completed run, got %+v", traceRepo.run)
	}
	if traceRepo.run.ExternalTrace == nil || traceRepo.run.ExternalTrace["local_catalog_primary"] == nil {
		t.Fatalf("expected trace to capture local catalog source")
	}
}

func TestGetRecommendationsFiltersUnsafeLocalCandidates(t *testing.T) {
	userRepo := &fakeUserRepository{user: &models.User{ID: "user-1", FullName: "User"}}
	profileRepo := &fakeProfileRepository{
		profile:     &models.Profile{ID: "profile-1", UserID: "user-1", Age: 25},
		lifestyle:   &models.Lifestyle{UserID: "user-1", Goal: "weight_loss", ActivityLevel: "light"},
		preferences: &models.Preferences{UserID: "user-1", Likes: models.StringSlice{"chicken"}},
		constraints: &models.Constraints{UserID: "user-1", ExcludedIngredients: models.StringSlice{"mushroom"}},
		nutritionProfile: &models.NutritionProfile{
			ID:                 "nutrition-1",
			UserID:             "user-1",
			ProfileID:          "profile-1",
			CalculatedAt:       time.Now(),
			MaxMealCalories:    650,
			MinProteinPerMeal:  20,
			MaxCarbsPerMeal:    70,
			MaxFatPerMeal:      25,
			MaxSugarPerMeal:    20,
			MaxSodiumMgPerMeal: 900,
		},
	}
	traceRepo := &memoryTraceRepositoryForService{}
	localRecipes := &fakeLocalRecipeRepositoryForService{candidates: []repository.LocalRecipeCandidate{
		{ID: "1", Title: "Mushroom Chicken Bowl", Ingredients: []string{"chicken", "mushroom"}, Calories: 420, Protein: 34, Carbs: 28, Fat: 12, Sugar: 7, SodiumMg: 360, Score: 60},
		{ID: "2", Title: "Lean Chicken Salad", Ingredients: []string{"chicken", "lettuce"}, Calories: 420, Protein: 34, Carbs: 28, Fat: 12, Sugar: 7, SodiumMg: 360, Score: 50},
	}}
	service := &RecommendationService{
		Profiles:     &ProfileService{Users: userRepo, Profiles: profileRepo},
		LocalRecipes: localRecipes,
		MedicalRules: &fakeMedicalRuleRepository{},
		Traces:       traceRepo,
	}

	response, err := service.GetRecommendations(context.Background(), "user-1", "profile-1", "req-1")
	if err != nil {
		t.Fatalf("expected recommendations with fallback, got %v", err)
	}
	if len(response.Meals) != 1 || response.Meals[0].ID != "2" {
		t.Fatalf("expected accepted safe local meal, got %+v", response.Meals)
	}
	if traceRepo.run == nil || traceRepo.run.DecisionSummary["totalCandidates"] != 2 {
		t.Fatalf("expected trace to record all local candidates, got %+v", traceRepo.run)
	}
}

func TestSelectDailyCandidatesIgnoresForbiddenAIControlFields(t *testing.T) {
	now := time.Date(2026, 5, 3, 8, 0, 0, 0, time.UTC)
	candidates := []*models.RecommendationCandidate{dailyCandidate("meal-2", 40)}
	service := &RecommendationService{AI: &fakeAITextGenerator{
		text: `[{"mealId":"meal-2","recommended":true,"rejectionReason":"","score":99,"explanation":"Try to change score."}]`,
	}}

	selected, result, mode := service.selectDailyCandidatesAndExplain(context.Background(), "user-1", "profile-1", candidates, now)

	if mode != "backend_random_ai_unavailable" || !strings.Contains(result.IgnoredReason, "ai_output_forbidden_field_score") {
		t.Fatalf("expected forbidden AI control field to trigger fallback, got mode=%s result=%+v", mode, result)
	}
	if len(selected) != 1 || selected[0].FinalScore != 40 {
		t.Fatalf("expected forbidden AI output not to change score, got %+v", selected)
	}
}

func TestParseAIExplanationResponseAcceptsMarkdownFencedJSON(t *testing.T) {
	items, err := parseAIExplanationResponse("```json\n[{\"mealId\":\"meal-1\",\"recommended\":true,\"rejectionReason\":\"\",\"explanation\":\"Good fit.\"}]\n```")
	if err != nil {
		t.Fatalf("expected fenced JSON to parse, got %v", err)
	}
	if len(items) != 1 || items[0].MealID != "meal-1" {
		t.Fatalf("expected parsed advice item, got %+v", items)
	}
}

func TestEvaluateCandidateCarriesEnrichmentProvenance(t *testing.T) {
	service := &RecommendationService{}
	candidate := service.evaluateCandidate(
		context.Background(),
		"run-1",
		"user-1",
		"profile-1",
		&models.Lifestyle{Goal: "weight_loss", ActivityLevel: "light"},
		&models.Preferences{Likes: models.StringSlice{"chicken"}},
		&models.Constraints{},
		&models.NutritionProfile{
			MaxMealCalories:           900,
			MinProteinPerMeal:         10,
			MaxCarbsPerMeal:           100,
			MaxFatPerMeal:             50,
			MaxSugarPerMeal:           30,
			MaxSodiumMgPerMeal:        1200,
			DerivedRecommendationTags: models.StringSlice{"healthy"},
		},
		nil,
		&SimilaritySignals{},
		enrichedRecipe{
			recipe: catalog.Recipe{
				ID:      44,
				Title:   "Chicken Bowl",
				Summary: "Grilled chicken and quinoa.",
				Nutrition: catalog.Nutrition{
					Nutrients: []catalog.Nutrient{
						{Name: "Calories", Amount: 520},
						{Name: "Protein", Amount: 36},
						{Name: "Carbohydrates", Amount: 32},
						{Name: "Fat", Amount: 12},
						{Name: "Sugar", Amount: 6},
						{Name: "Sodium", Amount: 300},
					},
				},
				ExtendedIngredients: []catalog.Ingredient{{Name: "chicken"}, {Name: "quinoa"}},
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
		&models.Preferences{},
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
		true,
	)

	if len(result.rejectedReasons) < 2 {
		t.Fatalf("expected multiple hard filter rejections, got %v", result.rejectedReasons)
	}
	if result.filterDecisions["blockedIngredients"] == nil {
		t.Fatalf("expected blocked ingredients to be recorded in filter decisions")
	}
}

func TestEvaluateHardFiltersRejectsBlockedAllergenInTitle(t *testing.T) {
	result := evaluateHardFilters(
		&models.Preferences{},
		&models.Constraints{Allergies: models.StringSlice{"egg"}},
		&models.NutritionProfile{
			MaxMealCalories:    900,
			MinProteinPerMeal:  0,
			MaxCarbsPerMeal:    120,
			MaxFatPerMeal:      80,
			MaxSugarPerMeal:    30,
			MaxSodiumMgPerMeal: 1000,
		},
		nil,
		candidateFacts{
			title:       "Egg Lecithin",
			ingredients: []string{"natural emulsifier"},
			calories:    120,
			protein:     20,
			carbs:       5,
			fat:         2,
			sugar:       1,
			sodium:      30,
		},
		true,
	)

	if !containsString(result.rejectedReasons, "contains blocked ingredients") {
		t.Fatalf("expected egg in title to be treated as a hard allergen hit, got %v", result.rejectedReasons)
	}
	if result.filterDecisions["blockedIngredients"] == nil {
		t.Fatalf("expected title-based allergen hit to be recorded in filter decisions")
	}
}

func TestEvaluateHardFiltersRejectsEggAllergyHiddenInLysozyme(t *testing.T) {
	result := evaluateHardFilters(
		&models.Preferences{},
		&models.Constraints{Allergies: models.StringSlice{"egg"}},
		&models.NutritionProfile{
			MaxMealCalories:    900,
			MinProteinPerMeal:  0,
			MaxCarbsPerMeal:    120,
			MaxFatPerMeal:      80,
			MaxSugarPerMeal:    30,
			MaxSodiumMgPerMeal: 1000,
		},
		nil,
		candidateFacts{
			title:       "Cheeses With Lysozymes",
			ingredients: []string{"milk", "lysozymes"},
			calories:    260,
			protein:     20,
			carbs:       5,
			fat:         15,
			sugar:       1,
			sodium:      250,
		},
		true,
	)

	if !containsString(result.rejectedReasons, "contains blocked ingredients") {
		t.Fatalf("expected lysozyme to be treated as an egg-allergy hard hit, got %v", result.rejectedReasons)
	}
}

func TestEvaluateHardFiltersTreatsProteinFloorAsSoftTarget(t *testing.T) {
	result := evaluateHardFilters(
		&models.Preferences{},
		&models.Constraints{},
		&models.NutritionProfile{
			MaxMealCalories:    900,
			MinProteinPerMeal:  45,
			MaxCarbsPerMeal:    120,
			MaxFatPerMeal:      80,
			MaxSugarPerMeal:    30,
			MaxSodiumMgPerMeal: 1000,
		},
		nil,
		candidateFacts{
			title:       "Vegetable Couscous",
			ingredients: []string{"couscous", "carrot"},
			calories:    420,
			protein:     12,
			carbs:       70,
			fat:         8,
			sugar:       5,
			sodium:      200,
		},
		true,
	)

	if len(result.rejectedReasons) != 0 {
		t.Fatalf("expected protein floor to remain a soft scoring target, got hard rejections %v", result.rejectedReasons)
	}
	if result.filterDecisions["proteinFloorSoftTargetMissed"] != true {
		t.Fatalf("expected protein floor miss to be recorded as a soft target")
	}
}

func TestEvaluateHardFiltersIgnoresRemovedMealTypeDecoration(t *testing.T) {
	result := evaluateHardFilters(
		&models.Preferences{},
		&models.Constraints{},
		&models.NutritionProfile{
			MaxMealCalories:    900,
			MinProteinPerMeal:  0,
			MaxCarbsPerMeal:    120,
			MaxFatPerMeal:      80,
			MaxSugarPerMeal:    80,
			MaxSodiumMgPerMeal: 1000,
		},
		nil,
		candidateFacts{
			title:       "Hazelnut Spread",
			ingredients: []string{"sugar", "hazelnuts", "cocoa"},
			baseTags:    []string{"sugary", "sweetened"},
			calories:    620,
			protein:     12,
			carbs:       60,
			fat:         20,
			sugar:       42,
			sodium:      50,
		},
		true,
	)

	if containsString(result.rejectedReasons, "outside requested meal types") {
		t.Fatalf("meal types are no longer product constraints and must not reject recipes, got %v", result.rejectedReasons)
	}
}

func TestComputeDeterministicScoreRunsOnlyAfterHardFilterPass(t *testing.T) {
	profile := &models.NutritionProfile{
		MaxMealCalories:           700,
		MinProteinPerMeal:         20,
		MaxCarbsPerMeal:           60,
		MaxFatPerMeal:             20,
		DerivedRecommendationTags: models.StringSlice{"healthy", "balanced"},
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

func TestRecommendationResponseHidesStaleAIExplanationsWhenAIWasNotApplied(t *testing.T) {
	now := time.Now().UTC()
	set := &models.DailyRecommendationSet{
		ID:         "set-1",
		RunID:      "run-1",
		ValidFrom:  now.Add(-time.Hour),
		ValidUntil: now.Add(time.Hour),
		DecisionSummary: models.JSONMap{
			"aiExplanationApplied": false,
			"aiSkippedReason":      "ai_key_missing",
		},
	}
	meals := []*models.DailyRecommendationMeal{
		{
			RecipeID:      "meal-1",
			Title:         "Safe meal",
			AIExplanation: "Selected because this stale text came from an old fallback.",
		},
	}

	response := recommendationResponseFromDailySet(set, meals, "profile-1", now, nil)
	if response.AIExplanationApplied {
		t.Fatalf("expected AI to be marked unavailable")
	}
	if got := response.Meals[0].AIExplanation; got != "" {
		t.Fatalf("expected stale AI explanation to be hidden, got %q", got)
	}

	set.DecisionSummary["aiExplanationApplied"] = true
	response = recommendationResponseFromDailySet(set, meals, "profile-1", now, nil)
	if got := response.Meals[0].AIExplanation; got == "" {
		t.Fatalf("expected validated AI explanation to be exposed when applied")
	}
}

func TestStripHTMLRemovesRecipeProviderLinks(t *testing.T) {
	input := `<a href="https://catalog.com/recipes/slow-cooker-lamb-curry-1583131">Slow cooker lamb curry</a>, <b>rich</b> &amp; balanced.`
	got := stripHTML(input)
	want := "Slow cooker lamb curry, rich & balanced."
	if got != want {
		t.Fatalf("expected cleaned summary %q, got %q", want, got)
	}
}

func dailyCandidate(id string, score float64) *models.RecommendationCandidate {
	return &models.RecommendationCandidate{
		ExternalRecipeID: id,
		Title:            "Recette " + id,
		Accepted:         true,
		FinalScore:       score,
		Explanation:      "",
		Ingredients:      models.StringSlice{"chicken", "quinoa"},
		Tags:             models.StringSlice{"balanced"},
		SourceProvenance: models.JSONMap{},
	}
}

func containsCandidateID(candidates []*models.RecommendationCandidate, recipeID string) bool {
	for _, candidate := range candidates {
		if candidate != nil && candidate.ExternalRecipeID == recipeID {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
