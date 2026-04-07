package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/marina1815/nutrimatch/internal/clients/googleai"
	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/http/dto"
	"github.com/marina1815/nutrimatch/internal/models"
)

type RecommendationService struct {
	Profiles *ProfileService
	Recipes  *spoonacular.Client
	AI       *googleai.Client
}

type aiRanking struct {
	ID          string `json:"id"`
	MatchReason string `json:"matchReason"`
	Score       int    `json:"score"`
}

func (s *RecommendationService) GetRecommendations(ctx context.Context, userID string) ([]dto.MealRecommendation, error) {
	profile, lifestyle, preferences, constraints, _, err := s.Profiles.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	query := buildQuery(preferences.MealStyles, preferences.Likes)
	intolerances := normalizeIntolerances(constraints.Allergies)
	exclude := mergeLists(preferences.Dislikes, constraints.Allergies, constraints.ExcludedIngredients)
	include := normalizeList(preferences.Likes)

	resp, err := s.Recipes.Search(spoonacular.SearchOptions{
		Query:             query,
		IncludeIngredients: include,
		ExcludeIngredients: exclude,
		Intolerances:      intolerances,
		Number:            12,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Results) == 0 {
		resp, err = s.Recipes.Search(spoonacular.SearchOptions{
			Query:  query,
			Number: 12,
		})
		if err != nil {
			return nil, err
		}
	}

	meals := make([]dto.MealRecommendation, 0, len(resp.Results))
	for _, r := range resp.Results {
		ingredients := extractIngredients(r.ExtendedIngredients)
		if !passesFailSafe(ingredients, constraints.Allergies, constraints.ExcludedIngredients) {
			continue
		}
		cal, protein, carbs, fat := extractMacros(r.Nutrition.Nutrients)
		meals = append(meals, dto.MealRecommendation{
			ID:          fmt.Sprintf("%d", r.ID),
			Title:       r.Title,
			Calories:    cal,
			Protein:     protein,
			Carbs:       carbs,
			Fat:         fat,
			Tags:        []string{lifestyle.Goal, lifestyle.ActivityLevel},
			Description: stripHTML(r.Summary),
			Ingredients: ingredients,
			MatchReason: "",
		})
	}

	if s.AI != nil && s.AI.APIKey != "" && len(meals) > 0 {
		if ranked := s.rankWithAI(profile, lifestyle, preferences, constraints, meals); len(ranked) > 0 {
			return ranked, nil
		}
	}

	for i := range meals {
		meals[i].MatchReason = "Correspond a vos preferences et contraintes"
	}
	return meals, nil
}

func (s *RecommendationService) rankWithAI(profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, meals []dto.MealRecommendation) []dto.MealRecommendation {
	payload := buildAIPrompt(profile, lifestyle, preferences, constraints, meals)
	text, err := s.AI.GenerateText(payload)
	if err != nil {
		return nil
	}

	var rankings []aiRanking
	if err := json.Unmarshal([]byte(text), &rankings); err != nil {
		return nil
	}

	byID := map[string]aiRanking{}
	for _, r := range rankings {
		byID[r.ID] = r
	}

	out := make([]dto.MealRecommendation, 0, len(meals))
	for _, meal := range meals {
		rank, ok := byID[meal.ID]
		if !ok {
			meal.MatchReason = "Correspond a vos preferences et contraintes"
			out = append(out, meal)
			continue
		}
		meal.MatchReason = rank.MatchReason
		out = append(out, meal)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return byID[out[i].ID].Score > byID[out[j].ID].Score
	})
	return out
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
		"traditionnel": "traditional",
		"recettes saines": "healthy",
		"oriental": "middle eastern",
		"moderne": "modern",
		"repas froids": "cold",
		"rapide": "quick",
		"equilibre": "balanced",
		"équilibré": "balanced",
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

func extractMacros(nutrients []spoonacular.Nutrient) (float64, float64, float64, float64) {
	var cal, protein, carbs, fat float64
	for _, n := range nutrients {
		switch strings.ToLower(n.Name) {
		case "calories":
			cal = n.Amount
		case "protein":
			protein = n.Amount
		case "carbohydrates":
			carbs = n.Amount
		case "fat":
			fat = n.Amount
		}
	}
	return cal, protein, carbs, fat
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

func passesFailSafe(ingredients []string, allergies []string, excluded []string) bool {
	blocked := mergeLists(allergies, excluded)
	blockSet := map[string]struct{}{}
	for _, item := range blocked {
		blockSet[singularize(item)] = struct{}{}
	}
	for _, ing := range ingredients {
		if _, found := blockSet[singularize(ing)]; found {
			return false
		}
	}
	return true
}

func singularize(input string) string {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) > 2 && strings.HasSuffix(trimmed, "s") {
		return strings.TrimSuffix(trimmed, "s")
	}
	return trimmed
}

func buildAIPrompt(profile *models.Profile, lifestyle *models.Lifestyle, preferences *models.Preferences, constraints *models.Constraints, meals []dto.MealRecommendation) string {
	payload := map[string]any{
		"profile": map[string]any{
			"age":    profile.Age,
			"sex":    profile.Sex,
			"weight": profile.Weight,
			"height": profile.Height,
			"goal":   lifestyle.Goal,
			"activity": lifestyle.ActivityLevel,
		},
		"preferences": map[string]any{
			"likes": preferences.Likes,
			"dislikes": preferences.Dislikes,
			"mealStyles": preferences.MealStyles,
		},
		"constraints": map[string]any{
			"allergies": constraints.Allergies,
			"excluded": constraints.ExcludedIngredients,
		},
		"meals": meals,
	}
	buf, _ := json.Marshal(payload)
	return "You are ranking meal recommendations. Return ONLY a JSON array of objects with fields id, score (0-100), matchReason. Do not include any other text. Input: " + string(buf)
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
