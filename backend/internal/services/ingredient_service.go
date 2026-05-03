package services

import (
	"context"
	"strings"

	"github.com/marina1815/nutrimatch/internal/catalog"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type IngredientSuggestionStore interface {
	SuggestIngredients(ctx context.Context, query string, limit int) ([]repository.CatalogOption, error)
	ListAllergies(ctx context.Context) ([]repository.CatalogOption, error)
}

type IngredientService struct {
	Local IngredientSuggestionStore
}

func (s *IngredientService) Suggest(ctx context.Context, query string, limit int) ([]repository.CatalogOption, error) {
	cleaned := ingredientSearchAlias(validation.NormalizeString(query))
	if len(cleaned) < 2 {
		return []repository.CatalogOption{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	if s == nil {
		return localIngredientSuggestions(cleaned, limit), nil
	}

	localSuggestions := func() []repository.CatalogOption {
		if s.Local != nil {
			items, err := s.Local.SuggestIngredients(ctx, cleaned, limit)
			if err == nil && len(items) > 0 {
				return items
			}
		}
		return localIngredientSuggestions(cleaned, limit)
	}

	return localSuggestions(), nil
}

func (s *IngredientService) ListAllergies(ctx context.Context) ([]repository.CatalogOption, error) {
	if s == nil || s.Local == nil {
		return []repository.CatalogOption{}, nil
	}
	return s.Local.ListAllergies(ctx)
}

func ingredientSearchAlias(query string) string {
	normalized := taxonomy.NormalizeLooseToken(query)
	aliases := map[string]string{
		"ail":        "garlic",
		"arachide":   "peanut",
		"arachides":  "peanut",
		"ble":        "wheat",
		"boeuf":      "beef",
		"champignon": "mushroom",
		"crevette":   "shrimp",
		"crevettes":  "shrimp",
		"dinde":      "turkey",
		"lait":       "milk",
		"oeuf":       "egg",
		"oeufs":      "egg",
		"oignon":     "onion",
		"poisson":    "fish",
		"porc":       "pork",
		"poulet":     "chicken",
		"riz":        "rice",
		"sucre":      "sugar",
		"thon":       "tuna",
	}
	if alias, ok := aliases[normalized]; ok {
		return alias
	}
	return query
}

func localIngredientSuggestions(query string, limit int) []repository.CatalogOption {
	fallbackCatalog := []string{
		"almond", "apple", "avocado", "banana", "beef", "bell pepper", "broccoli", "brown rice",
		"carrot", "chicken", "chickpea", "cinnamon", "cucumber", "egg", "garlic", "ginger",
		"greek yogurt", "green bean", "lentil", "lettuce", "milk", "mushroom", "oat", "olive oil",
		"onion", "peanut", "potato", "quinoa", "rice", "salmon", "spinach", "sweet potato",
		"tofu", "tomato", "tuna", "turkey", "walnut", "whole wheat", "yogurt",
	}
	out := make([]repository.CatalogOption, 0, limit)
	seen := make(map[string]struct{}, limit)
	normalizedQuery := taxonomy.NormalizeLooseToken(query)
	for _, item := range fallbackCatalog {
		normalizedItem := taxonomy.NormalizeLooseToken(item)
		if !strings.Contains(normalizedItem, normalizedQuery) {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, repository.CatalogOption{
			Value:  item,
			Label:  catalog.IngredientDisplayLabel(item),
			Source: "fallback",
		})
		if len(out) >= limit {
			return out
		}
	}
	return out
}
