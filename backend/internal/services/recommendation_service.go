package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"html"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/marina1815/nutrimatch/internal/catalog"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
)

var (
	ErrProfileAccessDenied        = errors.New("profile not found")
	ErrRecommendationQuota        = errors.New("recommendation quota exceeded")
	ErrRecommendationMealNotFound = errors.New("recommendation meal not found")
)

const (
	dailyRecommendationCount = 20
	dailyRecommendationTTL   = 24 * time.Hour
	choiceSuppressionTTL     = 7 * 24 * time.Hour
	dailyAIBatchTimeout      = 70 * time.Second
	localCatalogQueryLimit   = 10000
)

var (
	htmlTagPattern = regexp.MustCompile(`<[^>]*>`)
)

type AITextGenerator interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
}

type aiCodedError interface {
	AIErrorCode() string
}

type RecommendationService struct {
	Profiles     *ProfileService
	LocalRecipes repository.LocalRecipeRepository
	AI           AITextGenerator
	MedicalRules repository.MedicalRuleRepository
	Traces       repository.RecommendationTraceRepository
	Daily        repository.DailyRecommendationRepository
	Similarity   *SimilarityService
	Quota        *security.QuotaManager
	Cache        *security.TTLCache[*dto.RecommendationResponse]
	TxManager    repository.TxManager
}

type recommendationTraceBundle struct {
	Run        *models.RecommendationRun
	Candidates []*models.RecommendationCandidate
}

type searchPlan struct {
	Name           string
	Query          string
	Include        []string
	Exclude        []string
	HardExclude    []string
	Fallback       bool
	RelaxTaste     bool
	RelaxNutrition bool
	Number         int
	Relaxation     string
}

type aiExplanation struct {
	MealID          string   `json:"mealId"`
	Recommended     bool     `json:"recommended"`
	RejectionReason string   `json:"rejectionReason"`
	Explanation     string   `json:"explanation"`
	ExtraKeys       []string `json:"-"`
}

type aiExplanationResult struct {
	Applied           bool
	ValidationApplied bool
	SkippedReason     string
	IgnoredReason     string
	RejectedMealCount int
	ReplacementCount  int
}

type aiExplanationPromptPayload struct {
	Goal                 string                         `json:"goal"`
	ActivityLevel        string                         `json:"activityLevel"`
	PreferredMeals       []string                       `json:"preferredMeals"`
	PreferredIngredients []string                       `json:"preferredIngredients"`
	Candidates           []aiExplanationPromptCandidate `json:"candidates"`
}

type aiExplanationPromptCandidate struct {
	MealID      string   `json:"mealId"`
	Title       string   `json:"title"`
	Calories    float64  `json:"calories"`
	Protein     float64  `json:"protein"`
	Carbs       float64  `json:"carbs"`
	Fat         float64  `json:"fat"`
	Sugar       float64  `json:"sugar"`
	SodiumMg    float64  `json:"sodiumMg"`
	Ingredients []string `json:"ingredients"`
	Tags        []string `json:"tags"`
}

type aiPreparationOutput struct {
	PreparationGuide string                 `json:"preparationGuide"`
	Substitutions    []dto.MealSubstitution `json:"substitutions"`
}

type candidateFacts struct {
	title       string
	ingredients []string
	description string
	baseTags    []string
	finalTags   []string
	calories    float64
	protein     float64
	carbs       float64
	fat         float64
	sugar       float64
	sodium      float64
}

type hardFilterResult struct {
	rejectedReasons []string
	filterDecisions map[string]any
}

type deterministicScoreResult struct {
	score           float64
	acceptedReasons []string
	scoreBreakdown  map[string]any
}

type enrichedRecipe struct {
	recipe       catalog.Recipe
	sourcePlans  []string
	cacheSources []string
}

func (s *RecommendationService) GetRecommendations(ctx context.Context, userID, profileID, requestID string) (*dto.RecommendationResponse, error) {
	profile, lifestyle, preferences, constraints, _, err := s.Profiles.Get(ctx, userID)
	if err != nil {
		return nil, ErrProfileAccessDenied
	}
	if profileID != "" && profile.ID != profileID {
		return nil, ErrProfileAccessDenied
	}

	nutritionProfile, err := s.Profiles.GetNutritionProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if s.Daily != nil {
		set, setMeals, err := s.Daily.GetActiveSet(ctx, userID, profile.ID, now)
		if err != nil {
			return nil, err
		}
		if set != nil {
			activeChoice, err := s.Daily.GetChoiceForSet(ctx, set.ID, userID, profile.ID)
			if err != nil {
				return nil, err
			}
			return recommendationResponseFromDailySet(set, setMeals, profile.ID, now, activeChoice), nil
		}
	}

	if s.LocalRecipes == nil {
		return nil, errors.New("local recipe catalog unavailable")
	}
	if s.Quota != nil {
		allowed, err := s.Quota.Allow(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, ErrRecommendationQuota
		}
	}

	rules, err := s.MedicalRules.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	matchedRules := MatchMedicalRules(rules, constraints)

	signals := &SimilaritySignals{}
	if s.Similarity != nil {
		signals, err = s.Similarity.Expand(ctx, userID, profile, lifestyle, preferences, constraints)
		if err != nil {
			return nil, err
		}
	}

	plans := buildSearchPlans(preferences, constraints, nutritionProfile, signals)
	sourceHierarchy := []string{"local_catalog", "hard_filter", "deterministic_score", "vector_similarity", "ai_explanation"}
	externalTrace := make(map[string]any)
	localRecipes, localTrace := s.searchLocalCatalog(ctx, preferences, constraints, nutritionProfile, signals)
	localTrace["reason"] = "local catalog primary; external recipe API removed"
	localTrace["fallback"] = false
	externalTrace["local_catalog_primary"] = localTrace
	enrichedRecipes := append([]enrichedRecipe{}, localRecipes...)

	runID := uuid.NewString()
	candidates := make([]*models.RecommendationCandidate, 0, len(enrichedRecipes))
	acceptedCandidates := make([]*models.RecommendationCandidate, 0, len(enrichedRecipes))

	for _, recipe := range enrichedRecipes {
		candidate := s.evaluateCandidate(ctx, runID, userID, profile.ID, lifestyle, preferences, constraints, nutritionProfile, matchedRules, signals, recipe)
		candidates = append(candidates, candidate)
		if candidate.Accepted {
			acceptedCandidates = append(acceptedCandidates, candidate)
		}
	}

	sort.SliceStable(acceptedCandidates, func(i, j int) bool {
		if acceptedCandidates[i].FinalScore == acceptedCandidates[j].FinalScore {
			return acceptedCandidates[i].Title < acceptedCandidates[j].Title
		}
		return acceptedCandidates[i].FinalScore > acceptedCandidates[j].FinalScore
	})

	dailyTrace := map[string]any{}
	if s.Daily != nil {
		filtered, trace, err := s.dailyCandidatePool(ctx, userID, profile.ID, acceptedCandidates, now)
		if err != nil {
			return nil, err
		}
		acceptedCandidates = filtered
		dailyTrace = trace
	}

	selectedCandidates, aiResult, selectionMode := s.selectDailyCandidatesAndExplain(ctx, userID, profile.ID, acceptedCandidates, now)
	if len(selectedCandidates) == 0 {
		aiResult.SkippedReason = "no_accepted_meals"
	}

	meals := make([]dto.MealRecommendation, 0, len(selectedCandidates))
	for index, candidate := range selectedCandidates {
		candidate.FinalRank = index + 1
		meals = append(meals, mealRecommendationFromCandidate(candidate))
	}

	validUntil := now.Add(dailyRecommendationTTL)
	querySignature := security.SecureCacheKey(userID, profile.ID, nutritionProfile.CalculatedAt.Format(time.RFC3339), now.Format("2006-01-02"))
	run := &models.RecommendationRun{
		ID:                 runID,
		UserID:             userID,
		ProfileID:          profile.ID,
		NutritionProfileID: nutritionProfile.ID,
		Status:             statusFromCandidates(len(meals)),
		QuerySignature:     querySignature,
		SourceSummary: models.JSONMap{
			"plans":               len(plans),
			"enrichedRecipeCount": len(enrichedRecipes),
			"similarityLikes":     signals.Likes,
			"similaritySources":   signals.Sources,
			"semanticSimilarity":  signals.SemanticUsed,
			"fallbackApplied":     selectionMode == "backend_random_ai_unavailable",
			"dailyWindowHours":    int(dailyRecommendationTTL.Hours()),
		},
		DecisionSummary: models.JSONMap{
			"sourceHierarchy":           sourceHierarchy,
			"totalCandidates":           len(candidates),
			"safeCandidates":            len(acceptedCandidates),
			"accepted":                  len(selectedCandidates),
			"rejected":                  len(candidates) - len(meals),
			"aiExplanationApplied":      aiResult.Applied,
			"aiValidationApplied":       aiResult.ValidationApplied,
			"aiRejectedMealCount":       aiResult.RejectedMealCount,
			"aiReplacementCount":        aiResult.ReplacementCount,
			"aiSkippedReason":           aiResult.SkippedReason,
			"aiOutputIgnoredReason":     aiResult.IgnoredReason,
			"aiMode":                    "backend_random_then_ai_validation_and_explanation",
			"aiCanChooseUnsafeMeal":     false,
			"aiCanChangeScoreOrRanking": false,
			"selectionMode":             selectionMode,
			"dailyExclusion":            dailyTrace,
		},
		ExternalTrace:       models.JSONMap(externalTrace),
		CorrelatedRequestID: requestID,
	}

	if err := s.persistRun(ctx, run, candidates); err != nil {
		return nil, err
	}

	var set *models.DailyRecommendationSet
	if s.Daily != nil {
		set = &models.DailyRecommendationSet{
			UserID:             userID,
			ProfileID:          profile.ID,
			NutritionProfileID: nutritionProfile.ID,
			RunID:              runID,
			QuerySignature:     querySignature,
			Status:             run.Status,
			SelectionMode:      selectionMode,
			ValidFrom:          now,
			ValidUntil:         validUntil,
			SourceSummary:      run.SourceSummary,
			DecisionSummary:    run.DecisionSummary,
		}
		if err := s.Daily.CreateSet(ctx, set, dailyMealsFromCandidates(set, userID, profile.ID, selectedCandidates)); err != nil {
			return nil, err
		}
	}

	response := &dto.RecommendationResponse{
		RunID:                 runID,
		ProfileID:             profile.ID,
		Meals:                 meals,
		GeneratedAt:           now,
		ValidUntil:            validUntil,
		NextRefreshAt:         validUntil,
		SecondsUntilRefresh:   int(validUntil.Sub(now).Seconds()),
		SelectionMode:         selectionMode,
		AIExplanationApplied:  aiResult.Applied,
		AIValidationApplied:   aiResult.ValidationApplied,
		AIRejectedMealCount:   aiResult.RejectedMealCount,
		AIReplacementCount:    aiResult.ReplacementCount,
		AISkippedReason:       aiResult.SkippedReason,
		AIOutputIgnoredReason: aiResult.IgnoredReason,
	}
	return response, nil
}

func (s *RecommendationService) RefreshDailyExplanations(ctx context.Context, userID, profileID string) (*dto.RecommendationResponse, error) {
	if s == nil || s.Daily == nil {
		return nil, errors.New("daily recommendation repository unavailable")
	}
	profile, lifestyle, preferences, constraints, _, err := s.Profiles.Get(ctx, userID)
	if err != nil {
		return nil, ErrProfileAccessDenied
	}
	if profileID != "" && profile.ID != profileID {
		return nil, ErrProfileAccessDenied
	}
	nutritionProfile, err := s.Profiles.GetNutritionProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	set, meals, err := s.Daily.GetActiveSet(ctx, userID, profile.ID, now)
	if err != nil {
		return nil, err
	}
	if set == nil {
		return nil, ErrRecommendationMealNotFound
	}
	activeChoice, err := s.Daily.GetChoiceForSet(ctx, set.ID, userID, profile.ID)
	if err != nil {
		return nil, err
	}
	if activeChoice != nil {
		return recommendationResponseFromDailySet(set, meals, profile.ID, now, activeChoice), nil
	}
	if len(meals) == 0 {
		return recommendationResponseFromDailySet(set, meals, profile.ID, now, nil), nil
	}
	if s.AI == nil {
		set.DecisionSummary["aiExplanationApplied"] = false
		set.DecisionSummary["aiValidationApplied"] = false
		set.DecisionSummary["aiSkippedReason"] = "ai_key_missing"
		set.DecisionSummary["aiOutputIgnoredReason"] = ""
		set.DecisionSummary["aiRejectedMealCount"] = 0
		set.DecisionSummary["aiReplacementCount"] = 0
		if err := s.Daily.UpdateSetExplanations(ctx, set.ID, map[string]string{}, set.DecisionSummary, set.SelectionMode); err != nil {
			return nil, err
		}
		clearDailyMealAIExplanations(meals)
		return recommendationResponseFromDailySet(set, meals, profile.ID, now, nil), nil
	}

	if s.LocalRecipes == nil {
		return nil, errors.New("local recipe catalog unavailable")
	}
	rules, err := s.MedicalRules.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	matchedRules := MatchMedicalRules(rules, constraints)
	signals := &SimilaritySignals{}
	if s.Similarity != nil {
		signals, err = s.Similarity.Expand(ctx, userID, profile, lifestyle, preferences, constraints)
		if err != nil {
			return nil, err
		}
	}
	localRecipes, _ := s.searchLocalCatalog(ctx, preferences, constraints, nutritionProfile, signals)
	acceptedCandidates := make([]*models.RecommendationCandidate, 0, len(localRecipes))
	for _, recipe := range localRecipes {
		candidate := s.evaluateCandidate(ctx, set.RunID, userID, profile.ID, lifestyle, preferences, constraints, nutritionProfile, matchedRules, signals, recipe)
		if candidate.Accepted {
			acceptedCandidates = append(acceptedCandidates, candidate)
		}
	}
	sort.SliceStable(acceptedCandidates, func(i, j int) bool {
		if acceptedCandidates[i].FinalScore == acceptedCandidates[j].FinalScore {
			return acceptedCandidates[i].Title < acceptedCandidates[j].Title
		}
		return acceptedCandidates[i].FinalScore > acceptedCandidates[j].FinalScore
	})
	filtered, _, err := s.dailyCandidatePool(ctx, userID, profile.ID, acceptedCandidates, now)
	if err != nil {
		return nil, err
	}
	selected, aiResult, selectionMode := s.selectDailyCandidatesAndExplain(ctx, userID, profile.ID, filtered, now)
	for index, candidate := range selected {
		candidate.FinalRank = index + 1
	}

	status := statusFromCandidates(len(selected))
	set.DecisionSummary["aiExplanationApplied"] = aiResult.Applied
	set.DecisionSummary["aiValidationApplied"] = aiResult.ValidationApplied
	set.DecisionSummary["aiSkippedReason"] = aiResult.SkippedReason
	set.DecisionSummary["aiOutputIgnoredReason"] = aiResult.IgnoredReason
	set.DecisionSummary["aiRejectedMealCount"] = aiResult.RejectedMealCount
	set.DecisionSummary["aiReplacementCount"] = aiResult.ReplacementCount
	set.DecisionSummary["selectionMode"] = selectionMode
	if err := s.Daily.ReplaceSetMeals(ctx, set.ID, dailyMealsFromCandidates(set, userID, profile.ID, selected), set.DecisionSummary, selectionMode, status); err != nil {
		return nil, err
	}
	set.SelectionMode = selectionMode
	set.Status = status
	refreshedMeals := dailyMealsFromCandidates(set, userID, profile.ID, selected)
	return recommendationResponseFromDailySet(set, refreshedMeals, profile.ID, now, nil), nil
}

func clearDailyMealAIExplanations(meals []*models.DailyRecommendationMeal) {
	for _, meal := range meals {
		if meal != nil {
			meal.AIExplanation = ""
		}
	}
}

func recommendationResponseFromDailySet(set *models.DailyRecommendationSet, meals []*models.DailyRecommendationMeal, profileID string, now time.Time, activeChoice *models.RecipeChoice) *dto.RecommendationResponse {
	aiApplied, _ := set.DecisionSummary["aiExplanationApplied"].(bool)
	mealDTOs := make([]dto.MealRecommendation, 0, len(meals))
	if activeChoice == nil {
		for _, meal := range meals {
			mealDTOs = append(mealDTOs, mealRecommendationFromDailyMeal(meal, aiApplied))
		}
	}
	seconds := 0
	if set.ValidUntil.After(now) {
		seconds = int(set.ValidUntil.Sub(now).Seconds())
	}
	aiSkipped, _ := set.DecisionSummary["aiSkippedReason"].(string)
	aiIgnored, _ := set.DecisionSummary["aiOutputIgnoredReason"].(string)
	aiValidationApplied, _ := set.DecisionSummary["aiValidationApplied"].(bool)
	aiRejectedCount := intFromJSONMap(set.DecisionSummary, "aiRejectedMealCount")
	aiReplacementCount := intFromJSONMap(set.DecisionSummary, "aiReplacementCount")
	return &dto.RecommendationResponse{
		RunID:                 set.RunID,
		ProfileID:             profileID,
		Meals:                 mealDTOs,
		ActiveChoice:          mealChoiceResponseFromModel(activeChoice, findDailyMealByRecipeID(meals, recipeIDFromChoice(activeChoice)), profileID, aiApplied),
		GeneratedAt:           set.ValidFrom,
		ValidUntil:            set.ValidUntil,
		NextRefreshAt:         set.ValidUntil,
		SecondsUntilRefresh:   seconds,
		SelectionMode:         set.SelectionMode,
		AIExplanationApplied:  aiApplied,
		AIValidationApplied:   aiValidationApplied,
		AIRejectedMealCount:   aiRejectedCount,
		AIReplacementCount:    aiReplacementCount,
		AISkippedReason:       aiSkipped,
		AIOutputIgnoredReason: aiIgnored,
	}
}

func mealRecommendationFromDailyMeal(meal *models.DailyRecommendationMeal, aiApplied bool) dto.MealRecommendation {
	if meal == nil {
		return dto.MealRecommendation{}
	}
	aiExplanation := ""
	if aiApplied {
		aiExplanation = meal.AIExplanation
	}
	return dto.MealRecommendation{
		ID:                  meal.RecipeID,
		Title:               meal.Title,
		Calories:            meal.Calories,
		Protein:             meal.Protein,
		Carbs:               meal.Carbs,
		Fat:                 meal.Fat,
		Sugar:               meal.Sugar,
		SodiumMg:            meal.SodiumMg,
		Ingredients:         ingredientItems([]string(meal.Ingredients)),
		MatchReason:         meal.MatchReason,
		Source:              "local_catalog",
		Score:               meal.FinalScore,
		NutritionConfidence: meal.NutritionConfidence,
		NutritionSource:     meal.NutritionSource,
		SafetyWarnings:      []string(meal.SafetyWarnings),
		AIExplanation:       aiExplanation,
	}
}

func mealRecommendationFromCandidate(candidate *models.RecommendationCandidate) dto.MealRecommendation {
	if candidate == nil {
		return dto.MealRecommendation{}
	}
	return dto.MealRecommendation{
		ID:                  candidate.ExternalRecipeID,
		Title:               candidate.Title,
		Calories:            candidate.Calories,
		Protein:             candidate.Protein,
		Carbs:               candidate.Carbs,
		Fat:                 candidate.Fat,
		Sugar:               candidate.Sugar,
		SodiumMg:            candidate.SodiumMg,
		Tags:                []string(candidate.Tags),
		Description:         candidate.Description,
		Ingredients:         ingredientItems([]string(candidate.Ingredients)),
		MatchReason:         candidate.Explanation,
		Source:              candidate.Source,
		Score:               candidate.FinalScore,
		NutritionConfidence: stringFromMap(candidate.SourceProvenance, "nutritionConfidence"),
		NutritionSource:     stringFromMap(candidate.SourceProvenance, "nutritionSource"),
		SafetyWarnings:      stringSliceFromMap(candidate.SourceProvenance, "safetyWarnings"),
		AIExplanation:       aiExplanationFromProvenance(candidate.SourceProvenance),
	}
}

func ingredientItems(values []string) []dto.CatalogItem {
	items := make([]dto.CatalogItem, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		canonical := catalog.NormalizeIngredientValue(value)
		if canonical == "" {
			continue
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		items = append(items, dto.CatalogItem{
			Value: canonical,
			Label: catalog.IngredientDisplayLabel(canonical),
		})
	}
	return items
}

func mealChoiceResponseFromModel(choice *models.RecipeChoice, meal *models.DailyRecommendationMeal, profileID string, aiExplanationApplied bool) *dto.MealChoiceResponse {
	if choice == nil {
		return nil
	}
	mealDTO := dto.MealRecommendation{
		ID:            choice.RecipeID,
		Title:         choice.Title,
		Ingredients:   ingredientItems([]string(choice.Ingredients)),
		Source:        "local_catalog",
		AIExplanation: "",
	}
	if meal != nil {
		mealDTO = mealRecommendationFromDailyMeal(meal, aiExplanationApplied)
	}
	return &dto.MealChoiceResponse{
		ProfileID:             profileID,
		Meal:                  mealDTO,
		PreparationGuide:      choice.PreparationGuide,
		Substitutions:         substitutionsFromJSON(choice.Substitutions),
		AIApplied:             choice.AIApplied,
		AISkippedReason:       choice.AISkippedReason,
		AIOutputIgnoredReason: choice.AIOutputIgnoredReason,
		ChosenAt:              choice.ChosenAt,
		ExcludedUntil:         choice.ExpiresAt,
	}
}

func substitutionsFromJSON(values models.JSONMap) []dto.MealSubstitution {
	raw, ok := values["items"]
	if !ok {
		return []dto.MealSubstitution{}
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return []dto.MealSubstitution{}
	}
	var substitutions []dto.MealSubstitution
	if err := json.Unmarshal(encoded, &substitutions); err != nil {
		return []dto.MealSubstitution{}
	}
	return substitutions
}

func recipeIDFromChoice(choice *models.RecipeChoice) string {
	if choice == nil {
		return ""
	}
	return choice.RecipeID
}

func findDailyMealByRecipeID(meals []*models.DailyRecommendationMeal, recipeID string) *models.DailyRecommendationMeal {
	for _, meal := range meals {
		if meal != nil && meal.RecipeID == recipeID {
			return meal
		}
	}
	return nil
}

func dailyMealsFromCandidates(set *models.DailyRecommendationSet, userID, profileID string, candidates []*models.RecommendationCandidate) []*models.DailyRecommendationMeal {
	meals := make([]*models.DailyRecommendationMeal, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		meals = append(meals, &models.DailyRecommendationMeal{
			UserID:              userID,
			ProfileID:           profileID,
			RecipeID:            candidate.ExternalRecipeID,
			Title:               candidate.Title,
			FinalRank:           candidate.FinalRank,
			FinalScore:          candidate.FinalScore,
			Calories:            candidate.Calories,
			Protein:             candidate.Protein,
			Carbs:               candidate.Carbs,
			Fat:                 candidate.Fat,
			Sugar:               candidate.Sugar,
			SodiumMg:            candidate.SodiumMg,
			Ingredients:         candidate.Ingredients,
			AIExplanation:       aiExplanationFromProvenance(candidate.SourceProvenance),
			MatchReason:         candidate.Explanation,
			NutritionConfidence: stringFromMap(candidate.SourceProvenance, "nutritionConfidence"),
			NutritionSource:     stringFromMap(candidate.SourceProvenance, "nutritionSource"),
			SafetyWarnings:      models.StringSlice(stringSliceFromMap(candidate.SourceProvenance, "safetyWarnings")),
			SourceProvenance:    candidate.SourceProvenance,
		})
	}
	return meals
}

func candidatesFromDailyMeals(meals []*models.DailyRecommendationMeal) []*models.RecommendationCandidate {
	candidates := make([]*models.RecommendationCandidate, 0, len(meals))
	for _, meal := range meals {
		if meal == nil {
			continue
		}
		candidates = append(candidates, &models.RecommendationCandidate{
			UserID:           meal.UserID,
			ProfileID:        meal.ProfileID,
			ExternalRecipeID: meal.RecipeID,
			Title:            meal.Title,
			Accepted:         true,
			FinalRank:        meal.FinalRank,
			FinalScore:       meal.FinalScore,
			Calories:         meal.Calories,
			Protein:          meal.Protein,
			Carbs:            meal.Carbs,
			Fat:              meal.Fat,
			Sugar:            meal.Sugar,
			SodiumMg:         meal.SodiumMg,
			Ingredients:      meal.Ingredients,
			SourceProvenance: meal.SourceProvenance,
		})
	}
	return candidates
}

func (s *RecommendationService) dailyCandidatePool(ctx context.Context, userID, profileID string, candidates []*models.RecommendationCandidate, now time.Time) ([]*models.RecommendationCandidate, map[string]any, error) {
	trace := map[string]any{
		"inputSafeCandidates": len(candidates),
		"chosenExclusionDays": int(choiceSuppressionTTL.Hours() / 24),
		"previousSetExcluded": false,
	}
	suppressedIDs, err := s.Daily.GetSuppressedRecipeIDs(ctx, userID, profileID, now)
	if err != nil {
		return nil, nil, err
	}
	previousIDs, err := s.Daily.GetPreviousShownRecipeIDs(ctx, userID, profileID, now)
	if err != nil {
		return nil, nil, err
	}
	suppressed := stringSet(suppressedIDs)
	previous := stringSet(previousIDs)
	withoutSuppressed := filterCandidatesByID(candidates, suppressed)
	withoutPrevious := filterCandidatesByID(withoutSuppressed, previous)
	if len(withoutPrevious) >= dailyRecommendationCount {
		trace["previousSetExcluded"] = true
		trace["suppressedRecipeCount"] = len(suppressedIDs)
		trace["previousRecipeCount"] = len(previousIDs)
		trace["outputSafeCandidates"] = len(withoutPrevious)
		return withoutPrevious, trace, nil
	}
	trace["suppressedRecipeCount"] = len(suppressedIDs)
	trace["previousRecipeCount"] = len(previousIDs)
	trace["outputSafeCandidates"] = len(withoutSuppressed)
	trace["previousSetExcludedReason"] = "not_enough_safe_candidates_after_previous_set_exclusion"
	return withoutSuppressed, trace, nil
}

func (s *RecommendationService) selectDailyCandidatesAndExplain(ctx context.Context, userID, profileID string, candidates []*models.RecommendationCandidate, now time.Time) ([]*models.RecommendationCandidate, aiExplanationResult, string) {
	target := dailyRecommendationCount
	if len(candidates) < target {
		target = len(candidates)
	}
	if target <= 0 {
		return nil, aiExplanationResult{SkippedReason: "no_safe_candidates"}, "none"
	}

	seed := stableSeed(userID, profileID, now.Format("2006-01-02"))
	pool := deterministicWeightedPick(candidates, len(candidates), seed)
	selected := takeNextUntestedCandidates(pool, map[string]struct{}{}, target)
	if s == nil || s.AI == nil {
		return selected, aiExplanationResult{SkippedReason: "ai_key_missing"}, "backend_random_ai_unavailable"
	}

	result := aiExplanationResult{}
	seen := candidateIDSet(selected)
	rejected := map[string]struct{}{}
	explanations := make(map[string]string, target)
	pending := append([]*models.RecommendationCandidate{}, selected...)

	for len(pending) > 0 {
		items, err := s.validateAIRecommendationBatch(ctx, pending)
		if err != nil {
			if strings.HasPrefix(err.Error(), "ai_output_") {
				result.IgnoredReason = err.Error()
			} else {
				result.SkippedReason = classifyAIError(err)
			}
			clearAIExplanationsFromCandidates(selected)
			return selected, result, "backend_random_ai_unavailable"
		}

		nextPending := make([]*models.RecommendationCandidate, 0)
		for _, candidate := range pending {
			if candidate == nil {
				continue
			}
			item := items[candidate.ExternalRecipeID]
			if item.Recommended {
				explanations[candidate.ExternalRecipeID] = sanitizeAIExplanation(item.Explanation)
				continue
			}

			result.RejectedMealCount++
			rejected[candidate.ExternalRecipeID] = struct{}{}
			selected = removeCandidateByID(selected, candidate.ExternalRecipeID)
			replacement := takeFirstAvailableCandidate(pool, seen, rejected)
			if replacement == nil {
				continue
			}
			seen[replacement.ExternalRecipeID] = struct{}{}
			selected = append(selected, replacement)
			nextPending = append(nextPending, replacement)
			result.ReplacementCount++
		}
		pending = nextPending
	}

	if len(selected) == 0 {
		return selected, aiExplanationResult{SkippedReason: "no_ai_validated_meals", RejectedMealCount: result.RejectedMealCount, ReplacementCount: result.ReplacementCount}, "none"
	}
	applyAIExplanationsToCandidates(selected, explanations)
	result.Applied = true
	result.ValidationApplied = true
	if len(selected) < target {
		return selected, result, "backend_random_ai_partial"
	}
	return selected, result, "backend_random_ai_validated"
}

func (s *RecommendationService) validateAIRecommendationBatch(ctx context.Context, candidates []*models.RecommendationCandidate) (map[string]aiExplanation, error) {
	ctxAI, cancel := context.WithTimeout(ctx, dailyAIBatchTimeout)
	defer cancel()
	text, err := s.AI.GenerateText(ctxAI, buildDailyAIValidationPrompt(candidates))
	if err != nil {
		return nil, err
	}
	items, err := parseAIExplanationResponse(text)
	if err != nil {
		return nil, err
	}
	return validateAIValidatedMeals(candidates, items)
}

func deterministicWeightedPick(candidates []*models.RecommendationCandidate, target int, seed int64) []*models.RecommendationCandidate {
	if target <= 0 || len(candidates) == 0 {
		return nil
	}
	pool := append([]*models.RecommendationCandidate{}, candidates...)
	rng := rand.New(rand.NewSource(seed))
	selected := make([]*models.RecommendationCandidate, 0, minInt(target, len(pool)))
	for len(selected) < target && len(pool) > 0 {
		total := 0.0
		for _, candidate := range pool {
			weight := candidate.FinalScore
			if weight < 1 {
				weight = 1
			}
			total += weight
		}
		draw := rng.Float64() * total
		acc := 0.0
		chosenIndex := 0
		for i, candidate := range pool {
			weight := candidate.FinalScore
			if weight < 1 {
				weight = 1
			}
			acc += weight
			if draw <= acc {
				chosenIndex = i
				break
			}
		}
		selected = append(selected, pool[chosenIndex])
		pool = append(pool[:chosenIndex], pool[chosenIndex+1:]...)
	}
	return selected
}

func takeNextUntestedCandidates(pool []*models.RecommendationCandidate, seen map[string]struct{}, target int) []*models.RecommendationCandidate {
	out := make([]*models.RecommendationCandidate, 0, target)
	for _, candidate := range pool {
		if candidate == nil || candidate.ExternalRecipeID == "" {
			continue
		}
		if _, exists := seen[candidate.ExternalRecipeID]; exists {
			continue
		}
		seen[candidate.ExternalRecipeID] = struct{}{}
		out = append(out, candidate)
		if len(out) >= target {
			return out
		}
	}
	return out
}

func takeFirstAvailableCandidate(pool []*models.RecommendationCandidate, seen, rejected map[string]struct{}) *models.RecommendationCandidate {
	for _, candidate := range pool {
		if candidate == nil || candidate.ExternalRecipeID == "" {
			continue
		}
		if _, exists := seen[candidate.ExternalRecipeID]; exists {
			continue
		}
		if _, wasRejected := rejected[candidate.ExternalRecipeID]; wasRejected {
			continue
		}
		return candidate
	}
	return nil
}

func candidateIDSet(candidates []*models.RecommendationCandidate) map[string]struct{} {
	out := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.ExternalRecipeID != "" {
			out[candidate.ExternalRecipeID] = struct{}{}
		}
	}
	return out
}

func removeCandidateByID(candidates []*models.RecommendationCandidate, recipeID string) []*models.RecommendationCandidate {
	out := candidates[:0]
	for _, candidate := range candidates {
		if candidate == nil || candidate.ExternalRecipeID == recipeID {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func clearAIExplanationsFromCandidates(candidates []*models.RecommendationCandidate) {
	for _, candidate := range candidates {
		if candidate == nil || candidate.SourceProvenance == nil {
			continue
		}
		provenance := copyJSONMap(map[string]any(candidate.SourceProvenance))
		delete(provenance, "aiExplanation")
		candidate.SourceProvenance = models.JSONMap(provenance)
	}
}

func buildDailyAIValidationPrompt(candidates []*models.RecommendationCandidate) string {
	payload := struct {
		Task       string                         `json:"task"`
		Rules      []string                       `json:"rules"`
		Candidates []aiExplanationPromptCandidate `json:"candidates"`
	}{
		Task: "Valide chaque recette deja filtree par le backend et explique uniquement celles recommandees.",
		Rules: []string{
			"Retourne uniquement un tableau JSON.",
			"Retourne exactement un objet par mealId fourni, sans changer l'ordre.",
			"Chaque objet doit contenir uniquement mealId, recommended, rejectionReason et explanation.",
			"recommended est un booleen. Si recommended=false, rejectionReason doit expliquer le risque detecte.",
			"Si recommended=true, explanation doit tenir en une phrase francaise courte de 180 caracteres maximum.",
			"N'invente aucune recette, aucun score, aucun rang, aucune consigne externe et aucun nouvel ID.",
			"Tu peux seulement refuser une recette candidate; le backend choisira tout remplacement dans son pool deja sur.",
		},
		Candidates: make([]aiExplanationPromptCandidate, 0, len(candidates)),
	}
	for _, candidate := range candidates {
		payload.Candidates = append(payload.Candidates, aiExplanationPromptCandidate{
			MealID:      candidate.ExternalRecipeID,
			Title:       candidate.Title,
			Calories:    candidate.Calories,
			Protein:     candidate.Protein,
			Carbs:       candidate.Carbs,
			Fat:         candidate.Fat,
			Sugar:       candidate.Sugar,
			SodiumMg:    candidate.SodiumMg,
			Ingredients: sanitizePromptList([]string(candidate.Ingredients), 16),
			Tags:        sanitizePromptList([]string(candidate.Tags), 10),
		})
	}
	return "Tu es un assistant nutritionnel prudent. Le backend reste l'autorite absolue et a deja applique allergies, maladies, medicaments et exclusions. Reponds en francais.\nJSON:\n" + mustJSON(payload)
}

func validateAIValidatedMeals(pool []*models.RecommendationCandidate, items []aiExplanation) (map[string]aiExplanation, error) {
	if len(items) != len(pool) {
		return nil, fmt.Errorf("ai_output_expected_%d_items_got_%d", len(pool), len(items))
	}
	allowed := make(map[string]*models.RecommendationCandidate, len(pool))
	for _, candidate := range pool {
		allowed[candidate.ExternalRecipeID] = candidate
	}
	seen := map[string]struct{}{}
	out := make(map[string]aiExplanation, len(items))
	for _, item := range items {
		candidate, ok := allowed[item.MealID]
		if !ok || candidate == nil || !candidate.Accepted || len(candidate.RejectedReasons) > 0 {
			return nil, fmt.Errorf("ai_output_unknown_or_unsafe_meal_id")
		}
		if _, duplicate := seen[item.MealID]; duplicate {
			return nil, fmt.Errorf("ai_output_duplicate_meal_id")
		}
		item.Explanation = sanitizeAIExplanation(item.Explanation)
		item.RejectionReason = sanitizeAIExplanation(item.RejectionReason)
		if item.Recommended && item.Explanation == "" {
			return nil, fmt.Errorf("ai_output_empty_explanation")
		}
		if !item.Recommended && item.RejectionReason == "" {
			return nil, fmt.Errorf("ai_output_empty_rejection_reason")
		}
		seen[item.MealID] = struct{}{}
		out[item.MealID] = item
	}
	return out, nil
}

func applyAIExplanationsToCandidates(candidates []*models.RecommendationCandidate, explanations map[string]string) {
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		explanation := explanations[candidate.ExternalRecipeID]
		if explanation == "" {
			continue
		}
		provenance := copyJSONMap(map[string]any(candidate.SourceProvenance))
		provenance["aiExplanation"] = map[string]any{
			"validated":                true,
			"postAIValidation":         "validated_or_explained_only",
			"nonAuthoritative":         true,
			"selectionChanged":         false,
			"scoreOrRankChanged":       false,
			"ingredientsOrTagsChanged": false,
			"explanation":              explanation,
			"mode":                     "daily_backend_random_ai_validation",
		}
		candidate.SourceProvenance = models.JSONMap(provenance)
	}
}

func filterCandidatesByID(candidates []*models.RecommendationCandidate, blocked map[string]struct{}) []*models.RecommendationCandidate {
	if len(blocked) == 0 {
		return candidates
	}
	out := make([]*models.RecommendationCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if _, ok := blocked[candidate.ExternalRecipeID]; ok {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func stringSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out[item] = struct{}{}
		}
	}
	return out
}

func stableSeed(parts ...string) int64 {
	hash := fnv.New64a()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return int64(hash.Sum64() & 0x7fffffffffffffff)
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func classifyAIError(err error) string {
	if err == nil {
		return ""
	}
	var coded aiCodedError
	if errors.As(err, &coded) {
		if code := coded.AIErrorCode(); code != "" {
			return code
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "ai_timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "ai_network_unreachable"
	}
	return "ai_generation_failed"
}

func boolFromJSONMap(values models.JSONMap, key string) bool {
	if values == nil {
		return false
	}
	value, _ := values[key].(bool)
	return value
}

func intFromJSONMap(values models.JSONMap, key string) int {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		parsed, _ := value.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func (s *RecommendationService) persistRun(ctx context.Context, run *models.RecommendationRun, candidates []*models.RecommendationCandidate) error {
	if s.Traces == nil {
		return nil
	}
	if s.TxManager == nil {
		if err := s.Traces.CreateRun(ctx, run); err != nil {
			return err
		}
		return s.Traces.ReplaceCandidates(ctx, run.ID, candidates)
	}

	return s.TxManager.WithinTransaction(ctx, func(repos repository.Repositories) error {
		if err := repos.RecommendationRuns.CreateRun(ctx, run); err != nil {
			return err
		}
		return repos.RecommendationRuns.ReplaceCandidates(ctx, run.ID, candidates)
	})
}

func (s *RecommendationService) GetTrace(ctx context.Context, userID, profileID string) (*dto.RecommendationTraceResponse, error) {
	if s.Traces == nil {
		return nil, errors.New("trace repository unavailable")
	}
	run, candidates, err := s.Traces.GetLatestRunByProfile(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}

	traceCandidates := make([]dto.MealTrace, 0, len(candidates))
	for _, candidate := range candidates {
		traceCandidates = append(traceCandidates, dto.MealTrace{
			MealID:           candidate.ExternalRecipeID,
			Title:            candidate.Title,
			Accepted:         candidate.Accepted,
			FinalRank:        candidate.FinalRank,
			FinalScore:       candidate.FinalScore,
			AcceptedReasons:  []string(candidate.AcceptedReasons),
			RejectedReasons:  []string(candidate.RejectedReasons),
			ScoreBreakdown:   map[string]any(candidate.ScoreBreakdown),
			FilterDecisions:  map[string]any(candidate.FilterDecisions),
			SourceProvenance: map[string]any(candidate.SourceProvenance),
		})
	}

	return &dto.RecommendationTraceResponse{
		RunID:           run.ID,
		ProfileID:       profileID,
		Status:          run.Status,
		SourceSummary:   map[string]any(run.SourceSummary),
		DecisionSummary: map[string]any(run.DecisionSummary),
		ExternalTrace:   map[string]any(run.ExternalTrace),
		Candidates:      traceCandidates,
	}, nil
}

func (s *RecommendationService) GetExplanation(ctx context.Context, userID, profileID, mealID string) (*dto.RecommendationExplanationResponse, error) {
	if s.Traces == nil {
		return nil, errors.New("trace repository unavailable")
	}
	run, _, err := s.Traces.GetLatestRunByProfile(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}
	candidate, err := s.Traces.GetCandidateByRecipeID(ctx, userID, profileID, mealID)
	if err != nil {
		return nil, err
	}

	explanation := candidate.Explanation
	sourceProvenance := map[string]any(candidate.SourceProvenance)
	aiText := aiExplanationFromProvenance(sourceProvenance)
	aiApplied := aiText != ""

	return &dto.RecommendationExplanationResponse{
		RunID:                run.ID,
		ProfileID:            profileID,
		MealID:               mealID,
		Explanation:          explanation,
		AIExplanation:        aiText,
		AIExplanationApplied: aiApplied,
		AcceptedReasons:      []string(candidate.AcceptedReasons),
		RejectedReasons:      []string(candidate.RejectedReasons),
		ScoreBreakdown:       map[string]any(candidate.ScoreBreakdown),
		FilterDecisions:      map[string]any(candidate.FilterDecisions),
		SourceProvenance:     sourceProvenance,
	}, nil
}

func (s *RecommendationService) ChooseMeal(ctx context.Context, userID, profileID, mealID, requestID string) (*dto.MealChoiceResponse, error) {
	if s.Daily == nil {
		return nil, errors.New("daily recommendation repository unavailable")
	}
	profile, _, preferences, constraints, _, err := s.Profiles.Get(ctx, userID)
	if err != nil {
		return nil, ErrProfileAccessDenied
	}
	if profileID != "" && profile.ID != profileID {
		return nil, ErrProfileAccessDenied
	}
	nutritionProfile, err := s.Profiles.GetNutritionProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	set, meal, err := s.Daily.GetMealInActiveSet(ctx, userID, profile.ID, mealID, now)
	if err != nil {
		return nil, err
	}
	if meal == nil {
		return nil, ErrRecommendationMealNotFound
	}
	if set != nil {
		existingChoice, err := s.Daily.GetChoiceForSet(ctx, set.ID, userID, profile.ID)
		if err != nil {
			return nil, err
		}
		if existingChoice != nil {
			existingMeal := meal
			if existingChoice.RecipeID != meal.RecipeID {
				existingMeal = nil
			}
			return mealChoiceResponseFromModel(existingChoice, existingMeal, profile.ID, boolFromJSONMap(set.DecisionSummary, "aiExplanationApplied")), nil
		}
	}

	guide, substitutions, aiResult := s.generatePreparationGuide(ctx, meal, preferences, constraints, nutritionProfile)
	expiresAt := now.Add(choiceSuppressionTTL)
	choice := &models.RecipeChoice{
		SetID:                 meal.SetID,
		UserID:                userID,
		ProfileID:             profile.ID,
		RecipeID:              meal.RecipeID,
		Title:                 meal.Title,
		Ingredients:           meal.Ingredients,
		PreparationGuide:      guide,
		Substitutions:         models.JSONMap{"items": substitutions},
		AIApplied:             aiResult.Applied,
		AISkippedReason:       aiResult.SkippedReason,
		AIOutputIgnoredReason: aiResult.IgnoredReason,
		ChosenAt:              now,
		ExpiresAt:             expiresAt,
	}
	if err := s.Daily.CreateChoice(ctx, choice); err != nil {
		return nil, err
	}

	return mealChoiceResponseFromModel(choice, meal, profile.ID, set != nil && boolFromJSONMap(set.DecisionSummary, "aiExplanationApplied")), nil
}

func (s *RecommendationService) generatePreparationGuide(ctx context.Context, meal *models.DailyRecommendationMeal, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile) (string, []dto.MealSubstitution, aiExplanationResult) {
	if s == nil || s.AI == nil {
		return "", []dto.MealSubstitution{}, aiExplanationResult{SkippedReason: "ai_key_missing"}
	}
	ctxAI, cancel := context.WithTimeout(ctx, dailyAIBatchTimeout)
	defer cancel()
	prompt := buildPreparationGuidePrompt(meal, preferences, constraints, nutritionProfile)
	text, err := s.AI.GenerateText(ctxAI, prompt)
	if err != nil {
		return "", []dto.MealSubstitution{}, aiExplanationResult{SkippedReason: classifyAIError(err)}
	}
	output, err := parsePreparationOutput(text)
	if err != nil {
		return "", []dto.MealSubstitution{}, aiExplanationResult{IgnoredReason: err.Error()}
	}
	guide := sanitizePreparationGuide(output.PreparationGuide)
	if guide == "" {
		return "", []dto.MealSubstitution{}, aiExplanationResult{IgnoredReason: "ai_output_empty_preparation_guide"}
	}
	substitutions := validateMealSubstitutions(output.Substitutions, constraints, nutritionProfile)
	return guide, substitutions, aiExplanationResult{Applied: true}
}

func buildPreparationGuidePrompt(meal *models.DailyRecommendationMeal, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile) string {
	blocked := []string{}
	softDislikes := []string{}
	if constraints != nil {
		blocked = mergeLists(blocked, []string(constraints.Allergies), []string(constraints.ExcludedIngredients))
	}
	if nutritionProfile != nil {
		blocked = mergeLists(blocked, []string(nutritionProfile.DerivedExcluded))
	}
	if preferences != nil {
		softDislikes = sanitizePromptList([]string(preferences.Dislikes), 20)
	}
	payload := struct {
		Task        string   `json:"task"`
		Rules       []string `json:"rules"`
		MealID      string   `json:"mealId"`
		Title       string   `json:"title"`
		Ingredients []string `json:"ingredients"`
		Blocked     []string `json:"blockedIngredients"`
		Dislikes    []string `json:"softDislikes"`
	}{
		Task: "Genere un guide de preparation estime et des substitutions optionnelles compatibles.",
		Rules: []string{
			"Retourne uniquement un objet JSON.",
			"Champs autorises: preparationGuide, substitutions.",
			"substitutions est une liste d'objets {from,to,reason}.",
			"N'ajoute aucune substitution qui touche aux ingredients interdits.",
			"Ne donne pas de conseil medical.",
		},
		MealID:      meal.RecipeID,
		Title:       meal.Title,
		Ingredients: sanitizePromptList([]string(meal.Ingredients), 30),
		Blocked:     sanitizePromptList(blocked, 40),
		Dislikes:    softDislikes,
	}
	return "Tu aides a preparer une recette NutriMatch deja validee par le backend. Reponds en francais.\nJSON:\n" + mustJSON(payload)
}

func parsePreparationOutput(text string) (*aiPreparationOutput, error) {
	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	if !strings.HasPrefix(cleaned, "{") {
		start := strings.Index(cleaned, "{")
		end := strings.LastIndex(cleaned, "}")
		if start >= 0 && end > start {
			cleaned = cleaned[start : end+1]
		}
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return nil, err
	}
	for key := range raw {
		if key != "preparationGuide" && key != "substitutions" {
			return nil, fmt.Errorf("ai_output_forbidden_field_%s", key)
		}
	}
	var output aiPreparationOutput
	if raw["preparationGuide"] != nil {
		if err := json.Unmarshal(raw["preparationGuide"], &output.PreparationGuide); err != nil {
			return nil, fmt.Errorf("ai_output_invalid_preparation_guide")
		}
	}
	if raw["substitutions"] != nil {
		if err := json.Unmarshal(raw["substitutions"], &output.Substitutions); err != nil {
			return nil, fmt.Errorf("ai_output_invalid_substitutions")
		}
	}
	return &output, nil
}

func validateMealSubstitutions(items []dto.MealSubstitution, constraints *models.Constraints, nutritionProfile *models.NutritionProfile) []dto.MealSubstitution {
	blocked := []string{}
	if constraints != nil {
		blocked = mergeLists(blocked, []string(constraints.Allergies), []string(constraints.ExcludedIngredients))
	}
	if nutritionProfile != nil {
		blocked = mergeLists(blocked, []string(nutritionProfile.DerivedExcluded))
	}
	valid := make([]dto.MealSubstitution, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		item.From = normalizeDisplayTerm(item.From)
		item.To = normalizeDisplayTerm(item.To)
		item.Reason = sanitizePreparationGuide(item.Reason)
		if item.From == "" || item.To == "" || item.Reason == "" {
			continue
		}
		if hits := blockedIngredientHits([]string{item.To}, blocked); len(hits) > 0 {
			continue
		}
		key := item.From + "->" + item.To
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		valid = append(valid, item)
	}
	return valid
}

func deterministicPreparationGuide(meal *models.DailyRecommendationMeal) string {
	if meal == nil {
		return ""
	}
	ingredients := sanitizePromptList([]string(meal.Ingredients), 12)
	if len(ingredients) == 0 {
		return "Prepare cette recette avec une cuisson douce, en goutant et ajustant l'assaisonnement progressivement."
	}
	return "Prepare les ingredients principaux (" + strings.Join(ingredients, ", ") + "), cuis-les separement si besoin, puis assemble le plat en ajustant l'assaisonnement progressivement. Ce guide est estime depuis le catalogue local."
}

func sanitizePreparationGuide(input string) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
	if cleaned == "" {
		return ""
	}
	if len(cleaned) > 900 {
		cleaned = cleaned[:900]
		cleaned = strings.TrimSpace(cleaned)
	}
	return cleaned
}

func (s *RecommendationService) searchLocalCatalog(ctx context.Context, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, signals *SimilaritySignals) ([]enrichedRecipe, map[string]any) {
	trace := map[string]any{
		"provider":    "local_catalog",
		"reason":      "local catalogue complet",
		"resultCount": 0,
		"fallback":    false,
		"errorClass":  "",
	}
	if s.LocalRecipes == nil {
		trace["errorClass"] = "local_catalog_unavailable"
		return nil, trace
	}

	query := repository.LocalRecipeQuery{
		QueryTerms: mergeLists(
			[]string(nutritionProfile.DerivedRecommendationTags),
			[]string{nutritionGoalKeyword(nutritionProfile), fallbackBalancedQuery(nutritionProfile)},
		),
		Likes:               mergeLists([]string(preferences.Likes), signals.Likes),
		ExcludedIngredients: mergeLists([]string(constraints.ExcludedIngredients), []string(nutritionProfile.DerivedExcluded)),
		AllergyKeys:         []string(constraints.Allergies),
		ConditionKeys:       mergeLists([]string(constraints.Conditions), []string(constraints.ChronicDiseases)),
		Limit:               localCatalogQueryLimit,
	}
	candidates, err := s.LocalRecipes.Search(ctx, query)
	if err != nil {
		trace["errorClass"] = "local_catalog_error"
		trace["errorHash"] = security.SecureCacheKey(err.Error())
		return nil, trace
	}

	out := make([]enrichedRecipe, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, enrichedRecipe{
			recipe:       localCatalogRecipe(candidate),
			sourcePlans:  []string{"local_catalog"},
			cacheSources: []string{"embedded_xlsx_seed"},
		})
	}
	trace["resultCount"] = len(out)
	trace["querySignature"] = security.SecureCacheKey(mustJSON(query))
	return out, trace
}

func localCatalogRecipe(candidate repository.LocalRecipeCandidate) catalog.Recipe {
	ingredients := make([]catalog.Ingredient, 0, len(candidate.Ingredients))
	for _, ingredient := range candidate.Ingredients {
		ingredients = append(ingredients, catalog.Ingredient{Name: ingredient})
	}
	tags := mergeLists(candidate.Tags, candidate.MedicalRiskFlags)
	return catalog.Recipe{
		ID:                  localRecipeIntID(candidate.ID),
		Title:               candidate.Title,
		Summary:             buildLocalRecipeSummary(candidate.Title, candidate.Ingredients, candidate.Calories, candidate.Protein, candidate.Carbs, candidate.Fat),
		Tags:                tags,
		ReadyInMinutes:      30,
		Servings:            1,
		ExtendedIngredients: ingredients,
		Nutrition: catalog.Nutrition{Nutrients: []catalog.Nutrient{
			{Name: "Calories", Amount: candidate.Calories, Unit: "kcal"},
			{Name: "Protein", Amount: candidate.Protein, Unit: "g"},
			{Name: "Carbohydrates", Amount: candidate.Carbs, Unit: "g"},
			{Name: "Fat", Amount: candidate.Fat, Unit: "g"},
			{Name: "Sugar", Amount: candidate.Sugar, Unit: "g"},
			{Name: "Sodium", Amount: candidate.SodiumMg, Unit: "mg"},
		}},
	}
}

func buildLocalRecipeSummary(title string, ingredients []string, calories, protein, carbs, fat float64) string {
	visibleIngredients := sanitizePromptList(ingredients, 5)
	ingredientText := "des ingrédients locaux indexés"
	if len(visibleIngredients) > 0 {
		ingredientText = strings.Join(visibleIngredients, ", ")
	}
	return fmt.Sprintf("%s est proposé avec %s. Profil nutritionnel estimé: %.0f kcal, %.0fg protéines, %.0fg glucides et %.0fg lipides par portion.", title, ingredientText, calories, protein, carbs, fat)
}

func recipeProvider(recipe enrichedRecipe) string {
	for _, plan := range recipe.sourcePlans {
		if plan == "local_safety_fallback" {
			return "local_safety_fallback"
		}
		if strings.HasPrefix(plan, "local_") {
			return strings.TrimSuffix(plan, "_fallback")
		}
	}
	return "local_catalog"
}

func nutritionMetadataForProvider(provider string) (confidence, source string) {
	switch provider {
	case "local_catalog", "local_safety_fallback":
		return "estimated", provider + "_nutrition_estimate"
	default:
		return "reported", provider + "_nutrition"
	}
}

func nutritionSafetyWarnings(confidence string) []string {
	if confidence != "estimated" {
		return nil
	}
	return []string{"valeurs nutritionnelles estimees; les decisions de securite reposent sur les allergies, exclusions et regles medicales"}
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func stringSliceFromMap(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	switch items := values[key].(type) {
	case []string:
		return items
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if value, ok := item.(string); ok && value != "" {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}

func copyJSONMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func aiExplanationFromProvenance(values map[string]any) string {
	if values == nil {
		return ""
	}
	raw, ok := values["aiExplanation"].(map[string]any)
	if !ok {
		return ""
	}
	explanation, _ := raw["explanation"].(string)
	return explanation
}

func stableLocalRecipeIntID(id string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(id))
	return int(900000000 + hash.Sum32()%90000000)
}

func localRecipeIntID(id string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(id))
	if err == nil && parsed > 0 {
		return parsed
	}
	return stableLocalRecipeIntID(id)
}

func buildSearchPlans(preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, signals *SimilaritySignals) []searchPlan {
	queryTerms := mergeLists(
		[]string(preferences.Likes),
		[]string(nutritionProfile.DerivedRecommendationTags),
		signals.Likes,
	)
	hardExclude := mergeLists([]string(constraints.Allergies), []string(constraints.ExcludedIngredients), []string(nutritionProfile.DerivedExcluded))

	plans := []searchPlan{
		{
			Name:        "profile_query",
			Query:       buildQuery(queryTerms, nil),
			Exclude:     hardExclude,
			HardExclude: hardExclude,
			Relaxation:  "soft preference query with hard safety exclusions only",
		},
		{
			Name:        "goal_balanced",
			Query:       buildQuery([]string{nutritionGoalKeyword(nutritionProfile)}, nil),
			Exclude:     hardExclude,
			HardExclude: hardExclude,
			Relaxation:  "goal query with hard safety exclusions only",
		},
	}

	if len(signals.Likes) > 0 {
		plans = append(plans, searchPlan{
			Name:        "similarity_expansion",
			Query:       buildQuery(nil, signals.Likes),
			Exclude:     hardExclude,
			HardExclude: hardExclude,
			Relaxation:  "similarity query with hard safety exclusions only",
		})
	}
	plans = append(plans,
		searchPlan{
			Name:           "fallback_goal_candidates",
			Query:          nutritionGoalKeyword(nutritionProfile),
			Exclude:        hardExclude,
			HardExclude:    hardExclude,
			Fallback:       true,
			RelaxTaste:     true,
			RelaxNutrition: true,
			Number:         25,
			Relaxation:     "drop preference filters and provider nutrient bounds; keep safety exclusions",
		},
		searchPlan{
			Name:           "fallback_balanced_safety_net",
			Query:          fallbackBalancedQuery(nutritionProfile),
			Exclude:        hardExclude,
			HardExclude:    hardExclude,
			Fallback:       true,
			RelaxTaste:     true,
			RelaxNutrition: true,
			Number:         25,
			Relaxation:     "broad healthy candidate pool; deterministic firewall remains authoritative",
		},
	)
	return plans
}

func (s *RecommendationService) evaluateCandidate(ctx context.Context, runID, userID, profileID string, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, matchedRules []models.MedicalRule, signals *SimilaritySignals, recipe enrichedRecipe) *models.RecommendationCandidate {
	facts := buildCandidateFacts(recipe.recipe, lifestyle)
	provider := recipeProvider(recipe)
	nutritionConfidence, nutritionSource := nutritionMetadataForProvider(provider)
	filterResult := evaluateHardFilters(preferences, constraints, nutritionProfile, matchedRules, facts, true)
	scoreResult := computeDeterministicScore(preferences, nutritionProfile, signals, facts, len(filterResult.rejectedReasons) == 0)
	vectorScore, recipeVectorHash := s.scoreRecipeVector(ctx, recipe.recipe.ID, preferences, constraints, nutritionProfile, facts)
	if len(filterResult.rejectedReasons) == 0 && vectorScore > 0 {
		scoreResult.score += vectorScore * 10
		scoreResult.acceptedReasons = append(scoreResult.acceptedReasons, "recipe vector matches nutrition intent")
	}
	scoreResult.scoreBreakdown["recipeVectorSimilarity"] = vectorScore

	safetyWarnings := nutritionSafetyWarnings(nutritionConfidence)
	return &models.RecommendationCandidate{
		RunID:            runID,
		UserID:           userID,
		ProfileID:        profileID,
		ExternalRecipeID: fmt.Sprintf("%d", recipe.recipe.ID),
		Title:            recipe.recipe.Title,
		Source:           provider,
		Stage:            candidateStage(len(filterResult.rejectedReasons) == 0),
		Accepted:         len(filterResult.rejectedReasons) == 0,
		FinalScore:       scoreResult.score,
		Calories:         facts.calories,
		Protein:          facts.protein,
		Carbs:            facts.carbs,
		Fat:              facts.fat,
		Sugar:            facts.sugar,
		SodiumMg:         facts.sodium,
		Ingredients:      models.StringSlice(facts.ingredients),
		Tags:             models.StringSlice(facts.finalTags),
		AcceptedReasons:  models.StringSlice(scoreResult.acceptedReasons),
		RejectedReasons:  models.StringSlice(filterResult.rejectedReasons),
		ScoreBreakdown:   models.JSONMap(scoreResult.scoreBreakdown),
		FilterDecisions:  models.JSONMap(filterResult.filterDecisions),
		SourceProvenance: models.JSONMap{
			"provider":            provider,
			"recipeId":            recipe.recipe.ID,
			"searchPlans":         recipe.sourcePlans,
			"cacheSources":        recipe.cacheSources,
			"recipeVector":        map[string]any{"version": RecipeEmbeddingVersion, "hash": recipeVectorHash},
			"pipeline":            []string{"recipe_enrichment", "hard_filter", "deterministic_score", "vector_similarity", "ai_explanation"},
			"enrichedFacts":       []string{"nutrition", "ingredients", "summary"},
			"nutritionConfidence": nutritionConfidence,
			"nutritionSource":     nutritionSource,
			"safetyWarnings":      safetyWarnings,
		},
		Explanation: buildExplanation(scoreResult.acceptedReasons, filterResult.rejectedReasons),
		Description: facts.description,
	}
}

func (s *RecommendationService) scoreRecipeVector(ctx context.Context, recipeID int, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, facts candidateFacts) (float64, string) {
	if s == nil || s.Similarity == nil || s.Similarity.Embeddings == nil {
		return 0, ""
	}
	score, hash, err := s.Similarity.Embeddings.ScoreRecipe(ctx, fmt.Sprintf("%d", recipeID), preferences, constraints, nutritionProfile, facts)
	if err != nil {
		return 0, hash
	}
	return score, hash
}

func parseAIExplanationResponse(text string) ([]aiExplanation, error) {
	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	if !strings.HasPrefix(cleaned, "[") {
		start := strings.Index(cleaned, "[")
		end := strings.LastIndex(cleaned, "]")
		if start >= 0 && end > start {
			cleaned = cleaned[start : end+1]
		}
	}
	var rawItems []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(cleaned), &rawItems); err != nil {
		return nil, err
	}
	explanationItems := make([]aiExplanation, 0, len(rawItems))
	for _, raw := range rawItems {
		for key := range raw {
			if key != "mealId" && key != "recommended" && key != "rejectionReason" && key != "explanation" {
				return nil, fmt.Errorf("ai_output_forbidden_field_%s", key)
			}
		}
		var item aiExplanation
		if err := json.Unmarshal(raw["mealId"], &item.MealID); err != nil {
			return nil, fmt.Errorf("ai_output_invalid_meal_id")
		}
		recommendedRaw, ok := raw["recommended"]
		if !ok {
			return nil, fmt.Errorf("ai_output_missing_recommended")
		}
		if err := json.Unmarshal(recommendedRaw, &item.Recommended); err != nil {
			return nil, fmt.Errorf("ai_output_invalid_recommended")
		}
		if rawExplanation, ok := raw["explanation"]; ok {
			if err := json.Unmarshal(rawExplanation, &item.Explanation); err != nil {
				return nil, fmt.Errorf("ai_output_invalid_explanation")
			}
		}
		if rawReason, ok := raw["rejectionReason"]; ok {
			if err := json.Unmarshal(rawReason, &item.RejectionReason); err != nil {
				return nil, fmt.Errorf("ai_output_invalid_rejection_reason")
			}
		}
		if _, ok := raw["explanation"]; !ok {
			return nil, fmt.Errorf("ai_output_invalid_explanation")
		}
		item.MealID = strings.TrimSpace(item.MealID)
		if item.MealID == "" {
			return nil, fmt.Errorf("ai_output_missing_meal_id")
		}
		explanationItems = append(explanationItems, item)
	}
	return explanationItems, nil
}

func statusFromCandidates(accepted int) string {
	if accepted == 0 {
		return "no_matches"
	}
	return "completed"
}

func buildCandidateFacts(recipe catalog.Recipe, lifestyle *models.Lifestyle) candidateFacts {
	ingredients := extractIngredients(recipe.ExtendedIngredients)
	calories, protein, carbs, fat, sugar, sodium := extractNutrients(recipe.Nutrition.Nutrients)
	description := stripHTML(recipe.Summary)
	baseTags := mergeLists(recipe.Tags, inferTags(recipe.Title, description, ingredients, calories, protein, carbs, fat, sugar, sodium))
	finalTags := append([]string{}, baseTags...)
	if lifestyle != nil {
		finalTags = append(finalTags, lifestyle.Goal, lifestyle.ActivityLevel)
	}

	return candidateFacts{
		title:       recipe.Title,
		ingredients: ingredients,
		description: description,
		baseTags:    baseTags,
		finalTags:   mergeLists(finalTags),
		calories:    calories,
		protein:     protein,
		carbs:       carbs,
		fat:         fat,
		sugar:       sugar,
		sodium:      sodium,
	}
}

func evaluateHardFilters(preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, matchedRules []models.MedicalRule, facts candidateFacts, nutrientsVerified bool) hardFilterResult {
	rejectedReasons := make([]string, 0)
	allergies := []string{}
	excludedIngredients := []string{}
	derivedExcluded := []string{}
	if constraints != nil {
		allergies = []string(constraints.Allergies)
		excludedIngredients = []string(constraints.ExcludedIngredients)
	}
	if nutritionProfile != nil {
		derivedExcluded = []string(nutritionProfile.DerivedExcluded)
	}
	filterDecisions := map[string]any{
		"matchedRuleCodes": extractRuleCodes(matchedRules),
		"requiredRuleTags": extractRequiredTags(matchedRules),
		"thresholds": map[string]any{
			"maxMealCalories":    nutritionProfile.MaxMealCalories,
			"minProteinPerMeal":  nutritionProfile.MinProteinPerMeal,
			"maxCarbsPerMeal":    nutritionProfile.MaxCarbsPerMeal,
			"maxFatPerMeal":      nutritionProfile.MaxFatPerMeal,
			"maxSugarPerMeal":    nutritionProfile.MaxSugarPerMeal,
			"maxSodiumMgPerMeal": nutritionProfile.MaxSodiumMgPerMeal,
		},
		"nutrientHardLimitsApplied": nutrientsVerified,
	}

	hardFilterTerms := candidateHardFilterTerms(facts)
	blockedIngredients := mergeLists(allergies, excludedIngredients, derivedExcluded)
	if hits := blockedIngredientHits(hardFilterTerms, blockedIngredients); len(hits) > 0 {
		rejectedReasons = append(rejectedReasons, "contains blocked ingredients")
		filterDecisions["blockedIngredients"] = hits
	}

	for _, rule := range matchedRules {
		if hits := blockedIngredientHits(hardFilterTerms, []string(rule.BlockedIngredients)); len(hits) > 0 {
			rejectedReasons = append(rejectedReasons, "violates medical rule "+rule.Code)
			filterDecisions["ruleBlockedIngredients."+rule.Code] = hits
		}
		if overlapCount(facts.baseTags, []string(rule.BlockedTags)) > 0 {
			rejectedReasons = append(rejectedReasons, "matches blocked medical tag "+rule.Code)
		}
		if len(rule.RequiredTags) > 0 && overlapCount(facts.baseTags, []string(rule.RequiredTags)) == 0 {
			rejectedReasons = append(rejectedReasons, "missing required medical tag "+rule.Code)
		}
		if nutrientsVerified {
			if rule.MaxCalories > 0 && facts.calories > rule.MaxCalories {
				rejectedReasons = append(rejectedReasons, "exceeds medical calorie limit "+rule.Code)
			}
			if rule.MaxProteinGrams > 0 && facts.protein > rule.MaxProteinGrams {
				rejectedReasons = append(rejectedReasons, "exceeds medical protein limit "+rule.Code)
			}
			if rule.MaxCarbsGrams > 0 && facts.carbs > rule.MaxCarbsGrams {
				rejectedReasons = append(rejectedReasons, "exceeds medical carbohydrate limit "+rule.Code)
			}
			if rule.MaxFatGrams > 0 && facts.fat > rule.MaxFatGrams {
				rejectedReasons = append(rejectedReasons, "exceeds medical fat limit "+rule.Code)
			}
			if rule.MaxSugarGrams > 0 && facts.sugar > rule.MaxSugarGrams {
				rejectedReasons = append(rejectedReasons, "exceeds medical sugar limit "+rule.Code)
			}
			if rule.MaxSodiumMg > 0 && facts.sodium > rule.MaxSodiumMg {
				rejectedReasons = append(rejectedReasons, "exceeds medical sodium limit "+rule.Code)
			}
			if rule.MinProteinGrams > 0 && facts.protein < rule.MinProteinGrams {
				rejectedReasons = append(rejectedReasons, "below medical protein floor "+rule.Code)
			}
		}
	}

	if nutrientsVerified {
		if facts.calories > nutritionProfile.MaxMealCalories {
			rejectedReasons = append(rejectedReasons, "exceeds calorie ceiling")
		}
		if facts.protein < nutritionProfile.MinProteinPerMeal {
			filterDecisions["proteinFloorSoftTargetMissed"] = true
		}
		if facts.carbs > nutritionProfile.MaxCarbsPerMeal {
			rejectedReasons = append(rejectedReasons, "exceeds carbohydrate ceiling")
		}
		if facts.fat > nutritionProfile.MaxFatPerMeal {
			rejectedReasons = append(rejectedReasons, "exceeds fat ceiling")
		}
		if facts.sugar > nutritionProfile.MaxSugarPerMeal {
			rejectedReasons = append(rejectedReasons, "exceeds sugar ceiling")
		}
		if facts.sodium > nutritionProfile.MaxSodiumMgPerMeal {
			rejectedReasons = append(rejectedReasons, "exceeds sodium ceiling")
		}
	}

	return hardFilterResult{
		rejectedReasons: rejectedReasons,
		filterDecisions: filterDecisions,
	}
}

func candidateHardFilterTerms(facts candidateFacts) []string {
	terms := make([]string, 0, len(facts.ingredients)+1)
	if strings.TrimSpace(facts.title) != "" {
		terms = append(terms, facts.title)
	}
	terms = append(terms, facts.ingredients...)
	return mergeDisplayTerms(terms)
}

func computeDeterministicScore(preferences *models.Preferences, nutritionProfile *models.NutritionProfile, signals *SimilaritySignals, facts candidateFacts, hardFiltersPassed bool) deterministicScoreResult {
	acceptedReasons := make([]string, 0)
	score := 0.0
	baseScore := 40.0
	likes := []string{}
	dislikes := []string{}
	similarityLikes := []string{}
	derivedRecommendationTags := []string{}
	if preferences != nil {
		likes = []string(preferences.Likes)
		dislikes = []string(preferences.Dislikes)
	}
	if signals != nil {
		similarityLikes = signals.Likes
	}
	if nutritionProfile != nil {
		derivedRecommendationTags = []string(nutritionProfile.DerivedRecommendationTags)
	}
	nutrientBonus := nutrientAlignmentBonus(facts.calories, facts.protein, facts.carbs, facts.fat, nutritionProfile)
	preferenceOverlap := overlapCount(facts.ingredients, likes)
	dislikeOverlap := overlapCount(facts.ingredients, dislikes)
	similarityOverlap := overlapCount(facts.ingredients, similarityLikes)
	styleOverlap := overlapCount(facts.baseTags, derivedRecommendationTags)

	if hardFiltersPassed {
		score = baseScore
		if preferenceOverlap > 0 {
			score += 12
			acceptedReasons = append(acceptedReasons, "ingredients align with stated likes")
		}
		if similarityOverlap > 0 {
			score += 6
			acceptedReasons = append(acceptedReasons, "boosted by similar user preferences")
		}
		if styleOverlap > 0 {
			score += 8
			acceptedReasons = append(acceptedReasons, "matches derived recommendation tags")
		}
		if dislikeOverlap > 0 {
			score -= dislikeOverlap * 8
		}
		score += nutrientBonus
		acceptedReasons = append(acceptedReasons, "passes deterministic nutrition firewall")
	}

	return deterministicScoreResult{
		score:           score,
		acceptedReasons: acceptedReasons,
		scoreBreakdown: map[string]any{
			"base":                baseScore,
			"finalBeforeAI":       score,
			"nutrientAlignment":   nutrientBonus,
			"preferenceOverlap":   preferenceOverlap,
			"dislikePenalty":      dislikeOverlap * 8,
			"similarityOverlap":   similarityOverlap,
			"recommendedStyleHit": styleOverlap,
		},
	}
}

func candidateStage(accepted bool) string {
	if !accepted {
		return "hard_filter_rejected"
	}
	return "deterministic_scored"
}

func buildQuery(styles, likes []string) string {
	terms := append(normalizeList(styles), normalizeList(likes)...)
	if len(terms) == 0 {
		return "healthy"
	}
	return strings.Join(terms, " ")
}

func normalizeList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		clean := normalizeKeyword(item)
		if clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func normalizeKeyword(input string) string {
	trimmed := taxonomy.NormalizeLooseToken(input)
	if trimmed == "" {
		return ""
	}
	mapped := map[string]string{
		"traditionnel":    "traditional",
		"recettes saines": "healthy",
		"oriental":        "middle eastern",
		"moderne":         "modern",
		"repas froids":    "cold",
		"rapide":          "quick",
		"equilibre":       "balanced",
		"equilibree":      "balanced",
		"équilibré":       "balanced",
	}
	if v, ok := mapped[trimmed]; ok {
		return v
	}
	return trimmed
}

func mergeLists(lists ...[]string) []string {
	merged := []string{}
	seen := map[string]struct{}{}
	for _, list := range lists {
		for _, item := range list {
			key := normalizeKeyword(item)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, key)
		}
	}
	return merged
}

func extractNutrients(nutrients []catalog.Nutrient) (float64, float64, float64, float64, float64, float64) {
	var calories, protein, carbs, fat, sugar, sodium float64
	for _, nutrient := range nutrients {
		switch strings.ToLower(nutrient.Name) {
		case "calories":
			calories = nutrient.Amount
		case "protein":
			protein = nutrient.Amount
		case "carbohydrates":
			carbs = nutrient.Amount
		case "fat":
			fat = nutrient.Amount
		case "sugar":
			sugar = nutrient.Amount
		case "sodium":
			sodium = nutrient.Amount
		}
	}
	return calories, protein, carbs, fat, sugar, sodium
}

func extractIngredients(items []catalog.Ingredient) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		name := normalizeKeyword(item.Name)
		if name != "" {
			out = append(out, singularize(name))
		}
	}
	return out
}

func singularize(input string) string {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) > 2 && strings.HasSuffix(trimmed, "s") {
		return strings.TrimSuffix(trimmed, "s")
	}
	return trimmed
}

func inferTags(title, description string, ingredients []string, calories, protein, carbs, fat, sugar, sodium float64) []string {
	return catalog.InferSafetyTags(title, description, ingredients, calories, protein, carbs, fat, sugar, sodium)
}

func blockedIngredientHits(ingredients, blocked []string) []string {
	if len(ingredients) == 0 || len(blocked) == 0 {
		return nil
	}
	blockedTerms := map[string]string{}
	for _, item := range blocked {
		for _, term := range expandedIngredientTerms(item) {
			if term != "" {
				blockedTerms[term] = normalizeDisplayTerm(item)
			}
		}
	}
	if len(blockedTerms) == 0 {
		return nil
	}
	hits := make([]string, 0)
	seen := map[string]struct{}{}
	for _, ingredient := range ingredients {
		normalizedIngredient := normalizeDisplayTerm(ingredient)
		if normalizedIngredient == "" {
			continue
		}
		for blockedTerm, source := range blockedTerms {
			if ingredientMatchesBlockedTerm(normalizedIngredient, blockedTerm) {
				hit := normalizedIngredient
				if source != "" && source != normalizedIngredient {
					hit = normalizedIngredient + " -> " + source
				}
				if _, ok := seen[hit]; !ok {
					seen[hit] = struct{}{}
					hits = append(hits, hit)
				}
			}
		}
	}
	sort.Strings(hits)
	return hits
}

func expandedIngredientTerms(input string) []string {
	base := normalizeDisplayTerm(input)
	if base == "" {
		return nil
	}
	terms := []string{base, singularize(base)}
	switch base {
	case "egg", "eggs", "oeuf", "oeufs":
		terms = append(terms, "egg white", "egg yolk", "mayonnaise", "albumin", "lysozyme", "lecithin")
	case "dairy", "milk", "lait", "lactose":
		terms = append(terms, "milk", "cheese", "yogurt", "butter", "cream", "lactose", "whey", "casein")
	case "peanut", "peanuts", "arachide", "arachides":
		terms = append(terms, "peanut", "groundnut")
	case "tree nut", "tree_nut", "nuts", "nut", "fruits a coque":
		terms = append(terms, "almond", "walnut", "hazelnut", "cashew", "pistachio", "pecan", "macadamia", "brazil nut")
	case "fish", "poisson":
		terms = append(terms, "fish", "tuna", "salmon", "cod", "anchovy", "sardine")
	case "shellfish", "seafood", "fruit de mer", "fruits de mer":
		terms = append(terms, "shrimp", "prawn", "crab", "lobster", "clam", "mussel", "oyster", "scallop")
	case "gluten":
		terms = append(terms, "wheat", "barley", "rye", "flour", "semolina", "spelt")
	case "wheat", "ble", "blé":
		terms = append(terms, "wheat", "flour", "semolina", "bulgur", "couscous")
	case "soy", "soja":
		terms = append(terms, "soy", "soya", "tofu", "tempeh", "edamame")
	case "sesame", "sesame seed":
		terms = append(terms, "sesame", "tahini")
	case "sulfite", "sulfites":
		terms = append(terms, "sulfite", "sulphite")
	}
	return mergeDisplayTerms(terms)
}

func ingredientMatchesBlockedTerm(ingredient, blocked string) bool {
	if ingredient == "" || blocked == "" {
		return false
	}
	if ingredient == blocked {
		return true
	}
	ingredientTokens := tokenSet(ingredient)
	blockedTokens := tokenSet(blocked)
	if containsAllTokens(ingredientTokens, blockedTokens) {
		return true
	}
	if len(blocked) >= 4 && strings.Contains(" "+ingredient+" ", " "+blocked+" ") {
		return true
	}
	return false
}

func normalizeDisplayTerm(input string) string {
	value := normalizeKeyword(input)
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.Join(strings.Fields(value), " ")
	return singularize(value)
}

func mergeDisplayTerms(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		term := normalizeDisplayTerm(item)
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}
	return out
}

func tokenSet(input string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range strings.Fields(input) {
		token = singularize(normalizeKeyword(token))
		if token != "" {
			out[token] = struct{}{}
		}
	}
	return out
}

func containsAllTokens(haystack, needles map[string]struct{}) bool {
	if len(needles) == 0 {
		return false
	}
	for token := range needles {
		if _, ok := haystack[token]; !ok {
			return false
		}
	}
	return true
}

func overlapCount(left, right []string) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	set := make(map[string]struct{}, len(right))
	for _, item := range right {
		set[normalizeKeyword(item)] = struct{}{}
	}
	count := 0.0
	for _, item := range left {
		if _, ok := set[normalizeKeyword(item)]; ok {
			count++
		}
	}
	return count
}

func containsAnyNormalized(values []string, expected ...string) bool {
	if len(values) == 0 || len(expected) == 0 {
		return false
	}
	lookup := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := normalizeKeyword(value)
		if key != "" {
			lookup[key] = struct{}{}
		}
	}
	for _, value := range expected {
		key := normalizeKeyword(value)
		if _, ok := lookup[key]; ok {
			return true
		}
	}
	return false
}

func hasAnyTextTerm(text string, terms ...string) bool {
	text = normalizeDisplayTerm(text)
	if text == "" {
		return false
	}
	for _, term := range terms {
		term = normalizeDisplayTerm(term)
		if term != "" && strings.Contains(" "+text+" ", " "+term+" ") {
			return true
		}
	}
	return false
}

func nutrientAlignmentBonus(calories, protein, carbs, fat float64, nutritionProfile *models.NutritionProfile) float64 {
	bonus := 0.0
	if calories <= nutritionProfile.MaxMealCalories {
		bonus += 8
	}
	if protein >= nutritionProfile.MinProteinPerMeal {
		bonus += 8
	}
	if carbs <= nutritionProfile.MaxCarbsPerMeal {
		bonus += 4
	}
	if fat <= nutritionProfile.MaxFatPerMeal {
		bonus += 4
	}
	return bonus
}

func extractRuleCodes(rules []models.MedicalRule) []string {
	out := make([]string, 0, len(rules))
	for _, rule := range rules {
		out = append(out, rule.Code)
	}
	sort.Strings(out)
	return out
}

func extractRequiredTags(rules []models.MedicalRule) []string {
	out := make([]string, 0)
	for _, rule := range rules {
		out = append(out, []string(rule.RequiredTags)...)
	}
	return mergeLists(out)
}

func buildExplanation(acceptedReasons, rejectedReasons []string) string {
	return ""
}

func sanitizePromptList(items []string, limit int) []string {
	cleaned := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := normalizeKeyword(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		cleaned = append(cleaned, value)
		if limit > 0 && len(cleaned) >= limit {
			break
		}
	}
	return cleaned
}

func sanitizeAIExplanation(input string) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
	if cleaned == "" {
		return ""
	}
	if len(cleaned) > 220 {
		cleaned = cleaned[:220]
		cleaned = strings.TrimSpace(cleaned)
	}
	return cleaned
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func nutritionGoalKeyword(profile *models.NutritionProfile) string {
	if profile.MaxSodiumMgPerMeal <= 700 {
		return "low sodium"
	}
	if profile.MaxSugarPerMeal <= 18 {
		return "low sugar"
	}
	if profile.MinProteinPerMeal >= 20 {
		return "high protein"
	}
	return "balanced"
}

func fallbackBalancedQuery(profile *models.NutritionProfile) string {
	goal := nutritionGoalKeyword(profile)
	if goal == "balanced" {
		return "healthy balanced"
	}
	return goal + " healthy"
}

func stripHTML(input string) string {
	out := html.UnescapeString(input)
	out = htmlTagPattern.ReplaceAllString(out, "")
	out = strings.Join(strings.Fields(out), " ")
	return strings.TrimFunc(out, func(r rune) bool { return unicode.IsSpace(r) })
}
