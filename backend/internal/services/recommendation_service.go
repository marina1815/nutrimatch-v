package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
)

var (
	ErrProfileAccessDenied = errors.New("profile not found")
	ErrRecommendationQuota = errors.New("recommendation quota exceeded")
)

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

type RecipeSearcher interface {
	Search(ctx context.Context, opts spoonacular.SearchOptions) (*spoonacular.SearchResponse, error)
}

type AITextGenerator interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
}

type RecommendationService struct {
	Profiles     *ProfileService
	Recipes      RecipeSearcher
	LocalRecipes repository.LocalRecipeRepository
	AI           AITextGenerator
	MedicalRules repository.MedicalRuleRepository
	Traces       repository.RecommendationTraceRepository
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
	Name             string
	Query            string
	Include          []string
	Exclude          []string
	HardExclude      []string
	MealTypes        []string
	PreferredCuisine []string
	Fallback         bool
	RelaxTaste       bool
	RelaxNutrition   bool
	Number           int
	Relaxation       string
}

type aiAdvice struct {
	ID          string `json:"id"`
	Verdict     string `json:"verdict"`
	Explanation string `json:"explanation"`
}

type aiAdvicePromptPayload struct {
	Goal                 string                    `json:"goal"`
	ActivityLevel        string                    `json:"activityLevel"`
	PreferredMeals       []string                  `json:"preferredMeals"`
	PreferredIngredients []string                  `json:"preferredIngredients"`
	Candidates           []aiAdvicePromptCandidate `json:"candidates"`
}

type aiAdvicePromptCandidate struct {
	ID          string   `json:"id"`
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

type candidateFacts struct {
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
	recipe       spoonacular.Recipe
	sourcePlans  []string
	cacheSources []string
}

func (s *RecommendationService) GetRecommendations(ctx context.Context, userID, profileID, requestID string) (*dto.RecommendationResponse, error) {
	if s.Recipes == nil {
		return nil, errors.New("recipe client unavailable")
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

	cacheKey := security.SecureCacheKey(userID, profile.ID, nutritionProfile.CalculatedAt.Format(time.RFC3339))
	if s.Cache != nil {
		if cached, ok := s.Cache.Get(cacheKey); ok {
			return cached, nil
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
	primaryPlans, fallbackPlans := splitSearchPlans(plans)
	externalTrace := make(map[string]any)
	var enrichedRecipes []enrichedRecipe
	if s.LocalRecipes != nil {
		localRecipes, localTrace := s.searchLocalCatalog(ctx, preferences, constraints, nutritionProfile, signals)
		localTrace["reason"] = "local catalog baseline; external recipe API remains optional enrichment"
		externalTrace["local_catalog_primary"] = localTrace
		enrichedRecipes = append(enrichedRecipes, localRecipes...)
	} else {
		enrichedRecipes, externalTrace = s.searchAndEnrichRecipes(ctx, primaryPlans, lifestyle, preferences, constraints, nutritionProfile, matchedRules)
	}

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
	fallbackApplied := false
	if len(acceptedCandidates) == 0 && hasExternalSearchFailure(externalTrace) {
		localRecipes, localTrace := s.searchLocalCatalog(ctx, preferences, constraints, nutritionProfile, signals)
		externalTrace["local_catalog_fallback"] = localTrace
		if len(localRecipes) == 0 {
			localRecipes = localSafetyFallbackRecipes()
			externalTrace["local_safety_fallback"] = map[string]any{
				"provider":    "local_safety_fallback",
				"reason":      "local catalog returned no usable candidates; static emergency fallback applied",
				"resultCount": len(localRecipes),
				"fallback":    true,
			}
		}
		for _, recipe := range localRecipes {
			fallbackApplied = true
			enrichedRecipes = append(enrichedRecipes, recipe)
			candidate := s.evaluateCandidate(ctx, runID, userID, profile.ID, lifestyle, preferences, constraints, nutritionProfile, matchedRules, signals, recipe)
			candidates = append(candidates, candidate)
			if candidate.Accepted {
				acceptedCandidates = append(acceptedCandidates, candidate)
			}
		}
	}
	if len(acceptedCandidates) == 0 && len(fallbackPlans) > 0 && !hasExternalSearchFailure(externalTrace) {
		fallbackRecipes, fallbackTrace := s.searchAndEnrichRecipes(ctx, fallbackPlans, lifestyle, preferences, constraints, nutritionProfile, matchedRules)
		mergeExternalTrace(externalTrace, fallbackTrace)
		seenCandidates := candidateIDSet(candidates)
		for _, recipe := range fallbackRecipes {
			recipeID := fmt.Sprintf("%d", recipe.recipe.ID)
			if _, seen := seenCandidates[recipeID]; seen {
				continue
			}
			fallbackApplied = true
			enrichedRecipes = append(enrichedRecipes, recipe)
			candidate := s.evaluateCandidate(ctx, runID, userID, profile.ID, lifestyle, preferences, constraints, nutritionProfile, matchedRules, signals, recipe)
			candidates = append(candidates, candidate)
			seenCandidates[recipeID] = struct{}{}
			if candidate.Accepted {
				acceptedCandidates = append(acceptedCandidates, candidate)
			}
		}
	}
	if len(acceptedCandidates) == 0 && len(enrichedRecipes) == 0 && hasExternalSearchFailure(externalTrace) && externalTrace["local_catalog_fallback"] == nil {
		localRecipes, localTrace := s.searchLocalCatalog(ctx, preferences, constraints, nutritionProfile, signals)
		externalTrace["local_catalog_fallback"] = localTrace
		if len(localRecipes) == 0 {
			localRecipes = localSafetyFallbackRecipes()
			externalTrace["local_safety_fallback"] = map[string]any{
				"provider":    "local_safety_fallback",
				"reason":      "local catalog returned no usable candidates; static emergency fallback applied",
				"resultCount": len(localRecipes),
				"fallback":    true,
			}
		}
		for _, recipe := range localRecipes {
			fallbackApplied = true
			enrichedRecipes = append(enrichedRecipes, recipe)
			candidate := s.evaluateCandidate(ctx, runID, userID, profile.ID, lifestyle, preferences, constraints, nutritionProfile, matchedRules, signals, recipe)
			candidates = append(candidates, candidate)
			if candidate.Accepted {
				acceptedCandidates = append(acceptedCandidates, candidate)
			}
		}
	}

	aiApplied := false
	if s.AI != nil && len(acceptedCandidates) > 0 {
		aiApplied = s.applyAIAdvice(ctx, lifestyle, preferences, acceptedCandidates)
	}

	sort.SliceStable(acceptedCandidates, func(i, j int) bool {
		if acceptedCandidates[i].FinalScore == acceptedCandidates[j].FinalScore {
			return acceptedCandidates[i].Title < acceptedCandidates[j].Title
		}
		return acceptedCandidates[i].FinalScore > acceptedCandidates[j].FinalScore
	})

	meals := make([]dto.MealRecommendation, 0, len(acceptedCandidates))
	for index, candidate := range acceptedCandidates {
		candidate.FinalRank = index + 1
		meals = append(meals, dto.MealRecommendation{
			ID:          candidate.ExternalRecipeID,
			Title:       candidate.Title,
			Calories:    candidate.Calories,
			Protein:     candidate.Protein,
			Carbs:       candidate.Carbs,
			Fat:         candidate.Fat,
			Sugar:       candidate.Sugar,
			SodiumMg:    candidate.SodiumMg,
			Tags:        []string(candidate.Tags),
			Description: candidate.Description,
			Ingredients: []string(candidate.Ingredients),
			MatchReason: candidate.Explanation,
			Source:      candidate.Source,
			Score:       candidate.FinalScore,
		})
	}

	run := &models.RecommendationRun{
		ID:                 runID,
		UserID:             userID,
		ProfileID:          profile.ID,
		NutritionProfileID: nutritionProfile.ID,
		Status:             statusFromCandidates(len(meals)),
		QuerySignature:     cacheKey,
		SourceSummary: models.JSONMap{
			"plans":               len(plans),
			"enrichedRecipeCount": len(enrichedRecipes),
			"similarityLikes":     signals.Likes,
			"similarityStyles":    signals.MealStyles,
			"similarityMealTypes": signals.MealTypes,
			"similarityCuisines":  signals.Cuisines,
			"similaritySources":   signals.Sources,
			"semanticSimilarity":  signals.SemanticUsed,
			"fallbackApplied":     fallbackApplied,
		},
		DecisionSummary: models.JSONMap{
			"sourceHierarchy": []string{"external_recipe_api", "recipe_enrichment", "hard_filter", "deterministic_score", "vector_similarity", "ai_advice_explanation"},
			"totalCandidates": len(candidates),
			"accepted":        len(meals),
			"rejected":        len(candidates) - len(meals),
			"aiApplied":       true,
			"aiMode":          "explanation_only_non_authoritative",
		},
		ExternalTrace:       models.JSONMap(externalTrace),
		CorrelatedRequestID: requestID,
	}

	if err := s.persistRun(ctx, run, candidates); err != nil {
		return nil, err
	}

	response := &dto.RecommendationResponse{
		RunID:     runID,
		ProfileID: profile.ID,
		Meals:     meals,
	}
	if s.Cache != nil {
		s.Cache.Set(cacheKey, response)
	}
	return response, nil
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

	return &dto.RecommendationExplanationResponse{
		RunID:            run.ID,
		ProfileID:        profileID,
		MealID:           mealID,
		Explanation:      candidate.Explanation,
		AcceptedReasons:  []string(candidate.AcceptedReasons),
		RejectedReasons:  []string(candidate.RejectedReasons),
		ScoreBreakdown:   map[string]any(candidate.ScoreBreakdown),
		FilterDecisions:  map[string]any(candidate.FilterDecisions),
		SourceProvenance: map[string]any(candidate.SourceProvenance),
	}, nil
}

func (s *RecommendationService) searchAndEnrichRecipes(ctx context.Context, plans []searchPlan, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, matchedRules []models.MedicalRule) ([]enrichedRecipe, map[string]any) {
	recipesByID := make(map[int]*enrichedRecipe)
	externalTrace := make(map[string]any)

	for _, plan := range plans {
		if plan.Fallback && len(recipesByID) > 0 {
			continue
		}
		startedAt := time.Now()
		searchOpts := buildSearchOptions(plan, lifestyle, preferences, constraints, nutritionProfile, matchedRules)
		resp, searchErr := s.Recipes.Search(ctx, searchOpts)
		trace := buildExternalSearchTrace(searchOpts, resp, searchErr, time.Since(startedAt))
		trace["fallback"] = plan.Fallback
		trace["relaxation"] = plan.Relaxation
		externalTrace[plan.Name] = trace
		if searchErr != nil || resp == nil {
			continue
		}

		enriched := enrichRecipesFromSearchPlan(plan.Name, resp)
		for _, item := range enriched {
			existing, ok := recipesByID[item.recipe.ID]
			if !ok {
				copied := item
				recipesByID[item.recipe.ID] = &copied
				continue
			}
			existing.sourcePlans = mergeLists(existing.sourcePlans, item.sourcePlans)
			existing.cacheSources = mergeLists(existing.cacheSources, item.cacheSources)
		}
	}

	out := make([]enrichedRecipe, 0, len(recipesByID))
	for _, item := range recipesByID {
		out = append(out, *item)
	}
	return out, externalTrace
}

func splitSearchPlans(plans []searchPlan) ([]searchPlan, []searchPlan) {
	primary := make([]searchPlan, 0, len(plans))
	fallback := make([]searchPlan, 0)
	for _, plan := range plans {
		if plan.Fallback {
			fallback = append(fallback, plan)
			continue
		}
		primary = append(primary, plan)
	}
	return primary, fallback
}

func mergeExternalTrace(dst, src map[string]any) {
	for key, value := range src {
		dst[key] = value
	}
}

func candidateIDSet(candidates []*models.RecommendationCandidate) map[string]struct{} {
	out := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.ExternalRecipeID == "" {
			continue
		}
		out[candidate.ExternalRecipeID] = struct{}{}
	}
	return out
}

func hasExternalSearchFailure(trace map[string]any) bool {
	for _, value := range trace {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if errorClass, _ := item["errorClass"].(string); strings.HasPrefix(errorClass, "upstream_") {
			return true
		}
	}
	return false
}

func (s *RecommendationService) searchLocalCatalog(ctx context.Context, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, signals *SimilaritySignals) ([]enrichedRecipe, map[string]any) {
	trace := map[string]any{
		"provider":    "local_catalog",
		"reason":      "external recipe provider returned no usable candidates",
		"resultCount": 0,
		"fallback":    true,
		"errorClass":  "",
	}
	if s.LocalRecipes == nil {
		trace["errorClass"] = "local_catalog_unavailable"
		return nil, trace
	}

	query := repository.LocalRecipeQuery{
		QueryTerms: mergeLists(
			[]string(preferences.MealTypes),
			[]string(preferences.PreferredCuisines),
			[]string(nutritionProfile.RecommendedMealStyles),
			[]string{nutritionGoalKeyword(nutritionProfile), fallbackBalancedQuery(nutritionProfile)},
			signals.MealTypes,
			signals.Cuisines,
			signals.MealStyles,
		),
		Likes:               mergeLists([]string(preferences.Likes), signals.Likes),
		ExcludedIngredients: mergeLists([]string(constraints.ExcludedIngredients), []string(nutritionProfile.DerivedExcluded)),
		AllergyKeys:         []string(constraints.Allergies),
		Limit:               25,
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
			sourcePlans:  []string{"local_catalog_fallback"},
			cacheSources: []string{"embedded_xlsx_seed"},
		})
	}
	trace["resultCount"] = len(out)
	trace["querySignature"] = security.SecureCacheKey(mustJSON(query))
	return out, trace
}

func localSafetyFallbackRecipes() []enrichedRecipe {
	recipes := []spoonacular.Recipe{
		localRecipe(910001, "Bol poulet, riz complet et legumes", []string{"chicken", "brown rice", "broccoli", "carrot", "olive oil"}, 620, 52, 72, 14, 7, 520),
		localRecipe(910002, "Assiette dinde, quinoa et epinards", []string{"turkey", "quinoa", "spinach", "tomato", "olive oil"}, 590, 49, 58, 16, 6, 480),
		localRecipe(910003, "Bowl boeuf maigre et patate douce", []string{"beef", "sweet potato", "green bean", "onion"}, 640, 47, 68, 18, 9, 610),
		localRecipe(910004, "Salade thon, riz et concombre", []string{"tuna", "rice", "cucumber", "lettuce", "olive oil"}, 540, 46, 55, 13, 5, 560),
		localRecipe(910005, "Pois chiches, lentilles et legumes", []string{"chickpea", "lentil", "tomato", "spinach", "garlic"}, 610, 43, 88, 9, 10, 430),
	}

	out := make([]enrichedRecipe, 0, len(recipes))
	for _, recipe := range recipes {
		out = append(out, enrichedRecipe{
			recipe:       recipe,
			sourcePlans:  []string{"local_safety_fallback"},
			cacheSources: []string{},
		})
	}
	return out
}

func localRecipe(id int, title string, ingredients []string, calories, protein, carbs, fat, sugar, sodium float64) spoonacular.Recipe {
	items := make([]spoonacular.Ingredient, 0, len(ingredients))
	for _, ingredient := range ingredients {
		items = append(items, spoonacular.Ingredient{Name: ingredient})
	}
	return spoonacular.Recipe{
		ID:                  id,
		Title:               title,
		Summary:             buildLocalRecipeSummary(title, ingredients, calories, protein, carbs, fat),
		ReadyInMinutes:      30,
		Servings:            1,
		ExtendedIngredients: items,
		Nutrition: spoonacular.Nutrition{Nutrients: []spoonacular.Nutrient{
			{Name: "Calories", Amount: calories, Unit: "kcal"},
			{Name: "Protein", Amount: protein, Unit: "g"},
			{Name: "Carbohydrates", Amount: carbs, Unit: "g"},
			{Name: "Fat", Amount: fat, Unit: "g"},
			{Name: "Sugar", Amount: sugar, Unit: "g"},
			{Name: "Sodium", Amount: sodium, Unit: "mg"},
		}},
	}
}

func localCatalogRecipe(candidate repository.LocalRecipeCandidate) spoonacular.Recipe {
	ingredients := make([]spoonacular.Ingredient, 0, len(candidate.Ingredients))
	for _, ingredient := range candidate.Ingredients {
		ingredients = append(ingredients, spoonacular.Ingredient{Name: ingredient})
	}
	return spoonacular.Recipe{
		ID:                  stableLocalRecipeIntID(candidate.ID),
		Title:               candidate.Title,
		Summary:             buildLocalRecipeSummary(candidate.Title, candidate.Ingredients, candidate.Calories, candidate.Protein, candidate.Carbs, candidate.Fat),
		ReadyInMinutes:      30,
		Servings:            1,
		ExtendedIngredients: ingredients,
		Nutrition: spoonacular.Nutrition{Nutrients: []spoonacular.Nutrient{
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
	ingredientText := "ingredients locaux indexes"
	if len(visibleIngredients) > 0 {
		ingredientText = strings.Join(visibleIngredients, ", ")
	}
	return fmt.Sprintf("%s est propose avec %s. Profil nutritionnel estime: %.0f kcal, %.0fg proteines, %.0fg glucides et %.0fg lipides par portion.", title, ingredientText, calories, protein, carbs, fat)
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
	return "spoonacular"
}

func stableLocalRecipeIntID(id string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(id))
	return int(900000000 + hash.Sum32()%90000000)
}

func buildSearchPlans(preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, signals *SimilaritySignals) []searchPlan {
	queryTerms := mergeLists(
		[]string(preferences.MealStyles),
		[]string(preferences.Likes),
		[]string(preferences.MealTypes),
		[]string(preferences.PreferredCuisines),
		[]string(nutritionProfile.RecommendedMealStyles),
		signals.MealStyles,
		signals.Likes,
		signals.MealTypes,
		signals.Cuisines,
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
			Query:       buildQuery([]string{nutritionGoalKeyword(nutritionProfile)}, signals.MealStyles),
			Exclude:     hardExclude,
			HardExclude: hardExclude,
			Relaxation:  "goal query with hard safety exclusions only",
		},
	}

	if len(signals.MealStyles) > 0 || len(signals.Likes) > 0 || len(signals.MealTypes) > 0 || len(signals.Cuisines) > 0 {
		plans = append(plans, searchPlan{
			Name:        "similarity_expansion",
			Query:       buildQuery(mergeLists(signals.MealStyles, signals.MealTypes, signals.Cuisines), signals.Likes),
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

func enrichRecipesFromSearchPlan(planName string, resp *spoonacular.SearchResponse) []enrichedRecipe {
	if resp == nil {
		return nil
	}
	items := make([]enrichedRecipe, 0, len(resp.Results))
	cacheSources := []string{}
	if resp.CacheHit {
		cacheSources = append(cacheSources, "persistent_or_memory_cache")
	}
	for _, recipe := range resp.Results {
		items = append(items, enrichedRecipe{
			recipe:       recipe,
			sourcePlans:  []string{planName},
			cacheSources: cacheSources,
		})
	}
	return items
}

func buildSearchOptions(plan searchPlan, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, matchedRules []models.MedicalRule) spoonacular.SearchOptions {
	maxReadyTime := 45
	if lifestyle != nil && lifestyle.MaxReadyTime > 0 {
		maxReadyTime = lifestyle.MaxReadyTime
	}
	if plan.RelaxTaste && maxReadyTime < 60 {
		maxReadyTime = 60
	}

	excludeIngredients := plan.Exclude
	if plan.RelaxTaste {
		excludeIngredients = plan.HardExclude
	}
	number := plan.Number
	if number <= 0 {
		number = 12
	}

	opts := spoonacular.SearchOptions{
		Query:              plan.Query,
		ExcludeIngredients: excludeIngredients,
		Intolerances:       normalizeIntolerances(constraints.Allergies),
		MaxReadyTime:       maxReadyTime,
		Number:             number,
	}
	return opts
}

func (s *RecommendationService) evaluateCandidate(ctx context.Context, runID, userID, profileID string, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, matchedRules []models.MedicalRule, signals *SimilaritySignals, recipe enrichedRecipe) *models.RecommendationCandidate {
	facts := buildCandidateFacts(recipe.recipe, lifestyle)
	filterResult := evaluateHardFilters(preferences, constraints, nutritionProfile, matchedRules, facts)
	scoreResult := computeDeterministicScore(preferences, nutritionProfile, signals, facts, len(filterResult.rejectedReasons) == 0)
	vectorScore, recipeVectorHash := s.scoreRecipeVector(ctx, recipe.recipe.ID, preferences, constraints, nutritionProfile, facts)
	if len(filterResult.rejectedReasons) == 0 && vectorScore > 0 {
		scoreResult.score += vectorScore * 10
		scoreResult.acceptedReasons = append(scoreResult.acceptedReasons, "recipe vector matches nutrition intent")
	}
	scoreResult.scoreBreakdown["recipeVectorSimilarity"] = vectorScore

	provider := recipeProvider(recipe)
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
			"provider":      provider,
			"recipeId":      recipe.recipe.ID,
			"searchPlans":   recipe.sourcePlans,
			"cacheSources":  recipe.cacheSources,
			"recipeVector":  map[string]any{"version": RecipeEmbeddingVersion, "hash": recipeVectorHash},
			"pipeline":      []string{"recipe_enrichment", "hard_filter", "deterministic_score", "vector_similarity", "ai_advice_explanation"},
			"enrichedFacts": []string{"nutrition", "ingredients", "summary"},
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

func (s *RecommendationService) applyAIAdvice(ctx context.Context, lifestyle *models.Lifestyle, preferences *models.Preferences, candidates []*models.RecommendationCandidate) bool {
	if s == nil || s.AI == nil {
		return false
	}
	payload := aiAdvicePromptPayload{
		Candidates: make([]aiAdvicePromptCandidate, 0, len(candidates)),
	}
	if lifestyle != nil {
		payload.Goal = lifestyle.Goal
		payload.ActivityLevel = lifestyle.ActivityLevel
	}
	if preferences != nil {
		payload.PreferredMeals = sanitizePromptList([]string(preferences.MealStyles), 8)
		payload.PreferredIngredients = sanitizePromptList([]string(preferences.Likes), 8)
	}
	allowedIDs := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || !candidate.Accepted || len(candidate.RejectedReasons) > 0 {
			continue
		}
		allowedIDs[candidate.ExternalRecipeID] = struct{}{}
		payload.Candidates = append(payload.Candidates, aiAdvicePromptCandidate{
			ID:          candidate.ExternalRecipeID,
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
	if len(payload.Candidates) == 0 {
		return false
	}

	buf, _ := json.Marshal(payload)

	text, err := s.AI.GenerateText(ctx, "You are a non-authoritative nutrition explanation assistant. The deterministic firewall has already accepted these recipe IDs, and final safety remains controlled by code after your response. Return ONLY a JSON array with fields id, verdict, explanation. id must be one of the provided ids. verdict must be pass or review. Do not rank, do not score, do not invent meals, do not change ingredients, do not mention health constraints not present in the payload, and do not give medical advice. Explain briefly why the recipe seems aligned with the visible profile signals. Input: "+string(buf))
	if err != nil {
		return false
	}

	adviceItems, err := parseAIAdviceResponse(text)
	if err != nil {
		return false
	}

	byID := make(map[string]aiAdvice, len(adviceItems))
	for _, advice := range adviceItems {
		if _, ok := allowedIDs[advice.ID]; !ok {
			continue
		}
		byID[advice.ID] = advice
	}

	applied := false
	for _, candidate := range candidates {
		advice, ok := byID[candidate.ExternalRecipeID]
		if !ok || candidate == nil || !candidate.Accepted || len(candidate.RejectedReasons) > 0 {
			continue
		}
		sanitizedExplanation := sanitizeAIExplanation(advice.Explanation)
		if sanitizedExplanation == "" {
			continue
		}
		verdict := sanitizeAIVerdict(advice.Verdict)
		candidate.SourceProvenance["aiAdvice"] = map[string]any{
			"validated":                true,
			"verdict":                  verdict,
			"postAIValidation":         "kept_by_deterministic_firewall",
			"nonAuthoritative":         true,
			"scoreOrRankChanged":       false,
			"ingredientsOrTagsChanged": false,
			"explanation":              sanitizedExplanation,
		}
		candidate.Explanation = mergeExplanation(candidate.Explanation, sanitizedExplanation)
		applied = true
	}
	return applied
}

func parseAIAdviceResponse(text string) ([]aiAdvice, error) {
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
	var adviceItems []aiAdvice
	if err := json.Unmarshal([]byte(cleaned), &adviceItems); err != nil {
		return nil, err
	}
	return adviceItems, nil
}

func statusFromCandidates(accepted int) string {
	if accepted == 0 {
		return "no_matches"
	}
	return "completed"
}

func buildCandidateFacts(recipe spoonacular.Recipe, lifestyle *models.Lifestyle) candidateFacts {
	ingredients := extractIngredients(recipe.ExtendedIngredients)
	calories, protein, carbs, fat, sugar, sodium := extractNutrients(recipe.Nutrition.Nutrients)
	description := stripHTML(recipe.Summary)
	baseTags := inferTags(recipe.Title, description, ingredients, calories, protein, sugar, sodium)
	finalTags := append([]string{}, baseTags...)
	if lifestyle != nil {
		finalTags = append(finalTags, lifestyle.Goal, lifestyle.ActivityLevel)
	}

	return candidateFacts{
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

func evaluateHardFilters(preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, matchedRules []models.MedicalRule, facts candidateFacts) hardFilterResult {
	rejectedReasons := make([]string, 0)
	allergies := []string{}
	excludedIngredients := []string{}
	derivedExcluded := []string{}
	mealStyles := []string{}
	if constraints != nil {
		allergies = []string(constraints.Allergies)
		excludedIngredients = []string(constraints.ExcludedIngredients)
	}
	if nutritionProfile != nil {
		derivedExcluded = []string(nutritionProfile.DerivedExcluded)
	}
	if preferences != nil {
		mealStyles = []string(preferences.MealStyles)
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
	}

	blockedIngredients := mergeLists(allergies, excludedIngredients, derivedExcluded)
	if overlapCount(facts.ingredients, blockedIngredients) > 0 {
		rejectedReasons = append(rejectedReasons, "contains blocked ingredients")
		filterDecisions["blockedIngredients"] = blockedIngredients
	}

	for _, rule := range matchedRules {
		if overlapCount(facts.ingredients, []string(rule.BlockedIngredients)) > 0 {
			rejectedReasons = append(rejectedReasons, "violates medical rule "+rule.Code)
		}
		if overlapCount(facts.baseTags, []string(rule.BlockedTags)) > 0 {
			rejectedReasons = append(rejectedReasons, "matches blocked medical tag "+rule.Code)
		}
		if len(rule.RequiredTags) > 0 && overlapCount(facts.baseTags, []string(rule.RequiredTags)) == 0 {
			rejectedReasons = append(rejectedReasons, "missing required medical tag "+rule.Code)
		}
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

	if facts.calories > nutritionProfile.MaxMealCalories {
		rejectedReasons = append(rejectedReasons, "exceeds calorie ceiling")
	}
	if facts.protein < nutritionProfile.MinProteinPerMeal {
		rejectedReasons = append(rejectedReasons, "insufficient protein")
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

	filterDecisions["declaredMealStyles"] = mealStyles

	return hardFilterResult{
		rejectedReasons: rejectedReasons,
		filterDecisions: filterDecisions,
	}
}

func computeDeterministicScore(preferences *models.Preferences, nutritionProfile *models.NutritionProfile, signals *SimilaritySignals, facts candidateFacts, hardFiltersPassed bool) deterministicScoreResult {
	acceptedReasons := make([]string, 0)
	score := 0.0
	baseScore := 40.0
	likes := []string{}
	dislikes := []string{}
	similarityLikes := []string{}
	recommendedMealStyles := []string{}
	if preferences != nil {
		likes = []string(preferences.Likes)
		dislikes = []string(preferences.Dislikes)
	}
	if signals != nil {
		similarityLikes = signals.Likes
	}
	if nutritionProfile != nil {
		recommendedMealStyles = []string(nutritionProfile.RecommendedMealStyles)
	}
	nutrientBonus := nutrientAlignmentBonus(facts.calories, facts.protein, facts.carbs, facts.fat, nutritionProfile)
	preferenceOverlap := overlapCount(facts.ingredients, likes)
	dislikeOverlap := overlapCount(facts.ingredients, dislikes)
	similarityOverlap := overlapCount(facts.ingredients, similarityLikes)
	styleOverlap := overlapCount(facts.baseTags, recommendedMealStyles)

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
			acceptedReasons = append(acceptedReasons, "matches recommended meal styles")
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

func normalizeIntolerances(items []string) []string {
	return taxonomy.SpoonacularIntoleranceList(items)
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

func extractNutrients(nutrients []spoonacular.Nutrient) (float64, float64, float64, float64, float64, float64) {
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

func extractIngredients(items []spoonacular.Ingredient) []string {
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

func inferTags(title, description string, ingredients []string, calories, protein, sugar, sodium float64) []string {
	text := strings.ToLower(title + " " + description + " " + strings.Join(ingredients, " "))
	tags := []string{}
	addTag := func(tag string, patterns ...string) {
		for _, pattern := range patterns {
			if strings.Contains(text, pattern) {
				tags = append(tags, tag)
				return
			}
		}
	}
	addTag("fried", "fried")
	addTag("dessert", "dessert", "cake", "cookie")
	addTag("salty", "bacon", "sausage", "salted")
	addTag("healthy", "salad", "quinoa", "grilled")
	addTag("high-protein", "chicken", "beef", "tofu", "egg")
	if protein >= 20 {
		tags = append(tags, "high-protein")
	}
	if sugar > 0 && sugar <= 18 {
		tags = append(tags, "low-sugar")
	}
	if sodium > 0 && sodium <= 700 {
		tags = append(tags, "low-sodium")
	}
	if calories > 0 && calories <= 750 {
		tags = append(tags, "balanced")
	}
	return mergeLists(tags)
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
	if len(rejectedReasons) > 0 {
		return "Rejected because " + strings.Join(rejectedReasons, ", ")
	}
	if len(acceptedReasons) > 0 {
		return "Selected because " + strings.Join(acceptedReasons, ", ")
	}
	return "Selected after deterministic profile validation"
}

func buildExternalSearchTrace(opts spoonacular.SearchOptions, resp *spoonacular.SearchResponse, searchErr error, latency time.Duration) map[string]any {
	trace := map[string]any{
		"provider":            "spoonacular",
		"requestSignature":    security.SecureCacheKey(mustJSON(opts)),
		"queryPresent":        strings.TrimSpace(opts.Query) != "",
		"cuisineCount":        len(opts.Cuisine),
		"excludeCuisineCount": len(opts.ExcludeCuisine),
		"type":                opts.Type,
		"maxReadyTime":        opts.MaxReadyTime,
		"includeCount":        len(opts.IncludeIngredients),
		"excludeCount":        len(opts.ExcludeIngredients),
		"intoleranceCount":    len(opts.Intolerances),
		"latencyMs":           latency.Milliseconds(),
		"resultCount":         0,
		"cacheHit":            false,
		"errorClass":          "",
		"bounds": map[string]any{
			"maxCalories": opts.MaxCalories,
			"minProtein":  opts.MinProtein,
			"maxProtein":  opts.MaxProtein,
			"maxCarbs":    opts.MaxCarbs,
			"maxFat":      opts.MaxFat,
			"maxSugar":    opts.MaxSugar,
			"maxSodium":   opts.MaxSodium,
		},
	}
	if resp != nil {
		trace["resultCount"] = len(resp.Results)
		trace["cacheHit"] = resp.CacheHit
	}
	if searchErr != nil {
		trace["errorClass"] = classifySearchError(searchErr)
	}
	return trace
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

func sanitizeAIVerdict(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "pass":
		return "pass"
	case "review":
		return "review"
	default:
		return "review"
	}
}

func mergeExplanation(base, ai string) string {
	base = strings.TrimSpace(base)
	ai = strings.TrimSpace(ai)
	if ai == "" {
		return base
	}
	if base == "" {
		return "AI advice: " + ai
	}
	return base + " AI advice: " + ai
}

func classifySearchError(err error) string {
	if err == nil {
		return ""
	}
	var upstreamErr *spoonacular.UpstreamError
	if errors.As(err, &upstreamErr) {
		switch {
		case upstreamErr.StatusCode == 429:
			return "upstream_rate_limited"
		case upstreamErr.StatusCode >= 500:
			return "upstream_server_error"
		case upstreamErr.StatusCode >= 400:
			return "upstream_client_error"
		}
	}
	if errors.Is(err, spoonacular.ErrUpstreamFailure) {
		return "upstream_unavailable"
	}
	return "internal_error"
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
