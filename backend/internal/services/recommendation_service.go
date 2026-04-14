package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

var (
	ErrProfileAccessDenied = errors.New("profile not found")
	ErrRecommendationQuota = errors.New("recommendation quota exceeded")
)

type RecipeSearcher interface {
	Search(ctx context.Context, opts spoonacular.SearchOptions) (*spoonacular.SearchResponse, error)
}

type AITextGenerator interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
}

type RecommendationService struct {
	Profiles     *ProfileService
	Recipes      RecipeSearcher
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
	Name    string
	Query   string
	Include []string
	Exclude []string
}

type aiRerank struct {
	ID              string  `json:"id"`
	ConfidenceBonus float64 `json:"confidenceBonus"`
	Explanation     string  `json:"explanation"`
}

func (s *RecommendationService) GetRecommendations(ctx context.Context, userID, profileID, requestID string) (*dto.RecommendationResponse, error) {
	if s.Recipes == nil {
		return nil, errors.New("recipe client unavailable")
	}
	if s.Quota != nil && !s.Quota.Allow(userID) {
		return nil, ErrRecommendationQuota
	}

	profile, lifestyle, preferences, constraints, _, err := s.Profiles.Get(ctx, userID)
	if err != nil {
		return nil, err
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
		signals, err = s.Similarity.Expand(ctx, userID, profile.Age, lifestyle.ActivityLevel, lifestyle.Goal, []string(preferences.Likes), []string(preferences.MealStyles))
		if err != nil {
			return nil, err
		}
	}

	plans := buildSearchPlans(preferences, constraints, nutritionProfile, signals)
	recipesByID := make(map[int]spoonacular.Recipe)
	externalTrace := make(map[string]any)

	for _, plan := range plans {
		startedAt := time.Now()
		resp, searchErr := s.Recipes.Search(ctx, spoonacular.SearchOptions{
			Query:              plan.Query,
			IncludeIngredients: plan.Include,
			ExcludeIngredients: plan.Exclude,
			Intolerances:       normalizeIntolerances(constraints.Allergies),
			Number:             12,
		})

		externalTrace[plan.Name] = map[string]any{
			"query":     plan.Query,
			"include":   plan.Include,
			"exclude":   plan.Exclude,
			"latencyMs": time.Since(startedAt).Milliseconds(),
			"resultCount": func() int {
				if resp == nil {
					return 0
				}
				return len(resp.Results)
			}(),
			"error": func() string {
				if searchErr == nil {
					return ""
				}
				return searchErr.Error()
			}(),
		}

		if searchErr != nil || resp == nil {
			continue
		}
		for _, recipe := range resp.Results {
			if _, exists := recipesByID[recipe.ID]; exists {
				continue
			}
			recipesByID[recipe.ID] = recipe
		}
	}

	recipes := make([]spoonacular.Recipe, 0, len(recipesByID))
	for _, recipe := range recipesByID {
		recipes = append(recipes, recipe)
	}

	runID := uuid.NewString()
	candidates := make([]*models.RecommendationCandidate, 0, len(recipes))
	acceptedCandidates := make([]*models.RecommendationCandidate, 0, len(recipes))

	for _, recipe := range recipes {
		candidate := s.evaluateCandidate(runID, userID, profile.ID, lifestyle, preferences, constraints, nutritionProfile, matchedRules, signals, recipe)
		candidates = append(candidates, candidate)
		if candidate.Accepted {
			acceptedCandidates = append(acceptedCandidates, candidate)
		}
	}

	aiApplied := false
	if s.AI != nil && len(acceptedCandidates) > 0 && shouldApplyAIRerank(constraints, matchedRules) {
		s.applyAIRerank(ctx, lifestyle, preferences, acceptedCandidates)
		aiApplied = true
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
			"plans":            len(plans),
			"similarityLikes":  signals.Likes,
			"similarityStyles": signals.MealStyles,
		},
		DecisionSummary: models.JSONMap{
			"sourceHierarchy": []string{"deterministic_rules", "health_filters", "external_recipe_api", "ai_rerank"},
			"totalCandidates": len(candidates),
			"accepted":        len(meals),
			"rejected":        len(candidates) - len(meals),
			"aiApplied":       aiApplied,
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

func buildSearchPlans(preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, signals *SimilaritySignals) []searchPlan {
	queryTerms := mergeLists([]string(preferences.MealStyles), []string(preferences.Likes), []string(nutritionProfile.RecommendedMealStyles))
	include := mergeLists([]string(preferences.Likes), signals.Likes)
	exclude := mergeLists([]string(preferences.Dislikes), []string(constraints.Allergies), []string(constraints.ExcludedIngredients), []string(nutritionProfile.DerivedExcluded))

	plans := []searchPlan{
		{
			Name:    "strict_profile",
			Query:   buildQuery(queryTerms, nil),
			Include: include,
			Exclude: exclude,
		},
		{
			Name:    "goal_balanced",
			Query:   buildQuery([]string{nutritionGoalKeyword(nutritionProfile)}, signals.MealStyles),
			Include: nil,
			Exclude: exclude,
		},
	}

	if len(signals.MealStyles) > 0 || len(signals.Likes) > 0 {
		plans = append(plans, searchPlan{
			Name:    "similarity_expansion",
			Query:   buildQuery(signals.MealStyles, signals.Likes),
			Include: signals.Likes,
			Exclude: exclude,
		})
	}
	return plans
}

func (s *RecommendationService) evaluateCandidate(runID, userID, profileID string, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, nutritionProfile *models.NutritionProfile, matchedRules []models.MedicalRule, signals *SimilaritySignals, recipe spoonacular.Recipe) *models.RecommendationCandidate {
	ingredients := extractIngredients(recipe.ExtendedIngredients)
	calories, protein, carbs, fat, sugar, sodium := extractNutrients(recipe.Nutrition.Nutrients)
	description := stripHTML(recipe.Summary)
	heuristicTags := inferTags(recipe.Title, description, ingredients)

	acceptedReasons := []string{}
	rejectedReasons := []string{}
	scoreBreakdown := map[string]any{}
	filterDecisions := map[string]any{}

	score := 40.0
	if overlapCount(ingredients, []string(preferences.Likes)) > 0 {
		score += 12
		acceptedReasons = append(acceptedReasons, "ingredientes aligns with stated likes")
	}
	if overlapCount(ingredients, signals.Likes) > 0 {
		score += 6
		acceptedReasons = append(acceptedReasons, "boosted by similar user preferences")
	}
	if overlapCount(heuristicTags, []string(nutritionProfile.RecommendedMealStyles)) > 0 {
		score += 8
		acceptedReasons = append(acceptedReasons, "matches recommended meal styles")
	}

	blockedIngredients := mergeLists([]string(constraints.Allergies), []string(constraints.ExcludedIngredients), []string(nutritionProfile.DerivedExcluded))
	if overlapCount(ingredients, blockedIngredients) > 0 {
		rejectedReasons = append(rejectedReasons, "contains blocked ingredients")
		filterDecisions["blockedIngredients"] = blockedIngredients
	}

	for _, rule := range matchedRules {
		if overlapCount(ingredients, []string(rule.BlockedIngredients)) > 0 {
			rejectedReasons = append(rejectedReasons, "violates medical rule "+rule.Code)
		}
		if overlapCount(heuristicTags, []string(rule.BlockedTags)) > 0 {
			rejectedReasons = append(rejectedReasons, "matches blocked medical tag "+rule.Code)
		}
	}

	if calories > nutritionProfile.MaxMealCalories {
		rejectedReasons = append(rejectedReasons, "exceeds calorie ceiling")
	}
	if protein < nutritionProfile.MinProteinPerMeal {
		rejectedReasons = append(rejectedReasons, "insufficient protein")
	}
	if carbs > nutritionProfile.MaxCarbsPerMeal {
		rejectedReasons = append(rejectedReasons, "exceeds carbohydrate ceiling")
	}
	if fat > nutritionProfile.MaxFatPerMeal {
		rejectedReasons = append(rejectedReasons, "exceeds fat ceiling")
	}
	if sugar > nutritionProfile.MaxSugarPerMeal {
		rejectedReasons = append(rejectedReasons, "exceeds sugar ceiling")
	}
	if sodium > nutritionProfile.MaxSodiumMgPerMeal {
		rejectedReasons = append(rejectedReasons, "exceeds sodium ceiling")
	}

	if len(rejectedReasons) == 0 {
		acceptedReasons = append(acceptedReasons, "passes deterministic nutrition firewall")
		score += nutrientAlignmentBonus(calories, protein, carbs, fat, nutritionProfile)
	} else {
		score = 0
	}

	scoreBreakdown["base"] = 40
	scoreBreakdown["finalBeforeAI"] = score
	scoreBreakdown["nutrientAlignment"] = nutrientAlignmentBonus(calories, protein, carbs, fat, nutritionProfile)
	scoreBreakdown["preferenceOverlap"] = overlapCount(ingredients, []string(preferences.Likes))
	scoreBreakdown["similarityOverlap"] = overlapCount(ingredients, signals.Likes)
	filterDecisions["matchedRuleCodes"] = extractRuleCodes(matchedRules)
	filterDecisions["thresholds"] = map[string]any{
		"maxMealCalories":    nutritionProfile.MaxMealCalories,
		"minProteinPerMeal":  nutritionProfile.MinProteinPerMeal,
		"maxCarbsPerMeal":    nutritionProfile.MaxCarbsPerMeal,
		"maxFatPerMeal":      nutritionProfile.MaxFatPerMeal,
		"maxSugarPerMeal":    nutritionProfile.MaxSugarPerMeal,
		"maxSodiumMgPerMeal": nutritionProfile.MaxSodiumMgPerMeal,
	}

	return &models.RecommendationCandidate{
		RunID:            runID,
		UserID:           userID,
		ProfileID:        profileID,
		ExternalRecipeID: fmt.Sprintf("%d", recipe.ID),
		Title:            recipe.Title,
		Source:           "hybrid_orchestrator",
		Stage:            "deterministic_firewall",
		Accepted:         len(rejectedReasons) == 0,
		FinalScore:       score,
		Calories:         calories,
		Protein:          protein,
		Carbs:            carbs,
		Fat:              fat,
		Sugar:            sugar,
		SodiumMg:         sodium,
		Ingredients:      models.StringSlice(ingredients),
		Tags:             models.StringSlice(append(heuristicTags, lifestyle.Goal, lifestyle.ActivityLevel)),
		AcceptedReasons:  models.StringSlice(acceptedReasons),
		RejectedReasons:  models.StringSlice(rejectedReasons),
		ScoreBreakdown:   models.JSONMap(scoreBreakdown),
		FilterDecisions:  models.JSONMap(filterDecisions),
		SourceProvenance: models.JSONMap{"provider": "spoonacular", "recipeId": recipe.ID},
		Explanation:      buildExplanation(acceptedReasons, rejectedReasons),
		Description:      description,
	}
}

func (s *RecommendationService) applyAIRerank(ctx context.Context, lifestyle *models.Lifestyle, preferences *models.Preferences, candidates []*models.RecommendationCandidate) {
	payload := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		payload = append(payload, map[string]any{
			"id":          candidate.ExternalRecipeID,
			"title":       candidate.Title,
			"calories":    candidate.Calories,
			"protein":     candidate.Protein,
			"carbs":       candidate.Carbs,
			"fat":         candidate.Fat,
			"ingredients": candidate.Ingredients,
			"tags":        candidate.Tags,
		})
	}

	buf, _ := json.Marshal(map[string]any{
		"goal":        lifestyle.Goal,
		"activity":    lifestyle.ActivityLevel,
		"preferences": preferences,
		"candidates":  payload,
	})

	text, err := s.AI.GenerateText(ctx, "Re-rank ONLY these already-approved meals. Return ONLY JSON array with fields id, confidenceBonus (-5 to 5), explanation. Do not invent meals. Input: "+string(buf))
	if err != nil {
		return
	}

	var reranks []aiRerank
	if err := json.Unmarshal([]byte(text), &reranks); err != nil {
		return
	}

	byID := make(map[string]aiRerank, len(reranks))
	for _, rerank := range reranks {
		byID[rerank.ID] = rerank
	}

	for _, candidate := range candidates {
		rerank, ok := byID[candidate.ExternalRecipeID]
		if !ok {
			continue
		}
		bonus := rerank.ConfidenceBonus
		if bonus > 5 {
			bonus = 5
		}
		if bonus < -5 {
			bonus = -5
		}
		candidate.FinalScore += bonus
		candidate.SourceProvenance["aiConfidenceBonus"] = bonus
		if strings.TrimSpace(rerank.Explanation) != "" {
			candidate.Explanation = rerank.Explanation
		}
	}
}

func statusFromCandidates(accepted int) string {
	if accepted == 0 {
		return "no_matches"
	}
	return "completed"
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
	out := make([]string, 0, len(items))
	for _, item := range items {
		clean := normalizeKeyword(item)
		if clean == "shrimp" || clean == "shrimps" {
			clean = "shellfish"
		}
		if clean != "" {
			out = append(out, clean)
		}
	}
	return out
}

func normalizeKeyword(input string) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
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

func inferTags(title, description string, ingredients []string) []string {
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

func buildExplanation(acceptedReasons, rejectedReasons []string) string {
	if len(rejectedReasons) > 0 {
		return "Rejected because " + strings.Join(rejectedReasons, ", ")
	}
	if len(acceptedReasons) > 0 {
		return "Selected because " + strings.Join(acceptedReasons, ", ")
	}
	return "Selected after deterministic profile validation"
}

func shouldApplyAIRerank(constraints *models.Constraints, matchedRules []models.MedicalRule) bool {
	if constraints == nil {
		return len(matchedRules) == 0
	}
	if len(matchedRules) > 0 {
		return false
	}
	if constraints.TakesMedication || constraints.HasChronicDisease {
		return false
	}
	return len(constraints.Conditions) == 0 && len(constraints.ChronicDiseases) == 0
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

func stripHTML(input string) string {
	out := input
	out = strings.ReplaceAll(out, "<b>", "")
	out = strings.ReplaceAll(out, "</b>", "")
	out = strings.ReplaceAll(out, "<a>", "")
	out = strings.ReplaceAll(out, "</a>", "")
	out = strings.TrimFunc(out, func(r rune) bool { return unicode.IsSpace(r) })
	return out
}
