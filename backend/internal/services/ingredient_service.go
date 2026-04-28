package services

import (
	"context"
	"strings"

	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/taxonomy"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type IngredientAutocompleteClient interface {
	AutocompleteIngredients(ctx context.Context, query string, number int) ([]spoonacular.IngredientSuggestion, error)
}

type IngredientSuggestionStore interface {
	SuggestIngredients(ctx context.Context, query string, limit int) ([]string, error)
}

type IngredientService struct {
	Client IngredientAutocompleteClient
	Local  IngredientSuggestionStore
}

func (s *IngredientService) Suggest(ctx context.Context, query string, limit int) ([]string, error) {
	cleaned := validation.NormalizeString(query)
	if len(cleaned) < 2 {
		return []string{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	if s == nil {
		return localIngredientSuggestions(cleaned, limit), nil
	}

	localSuggestions := func() []string {
		if s.Local != nil {
			items, err := s.Local.SuggestIngredients(ctx, cleaned, limit)
			if err == nil && len(items) > 0 {
				return items
			}
		}
		return localIngredientSuggestions(cleaned, limit)
	}

	if s.Client == nil {
		return localSuggestions(), nil
	}

	suggestions, err := s.Client.AutocompleteIngredients(ctx, cleaned, limit)
	if err != nil {
		return localSuggestions(), nil
	}

	out := make([]string, 0, len(suggestions))
	seen := make(map[string]struct{}, len(suggestions))
	for _, suggestion := range suggestions {
		name := strings.ToLower(validation.NormalizeString(suggestion.Name))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return localSuggestions(), nil
	}
	return out, nil
}

func localIngredientSuggestions(query string, limit int) []string {
	catalog := []string{
		"almond", "apple", "avocado", "banana", "beef", "bell pepper", "broccoli", "brown rice",
		"carrot", "chicken", "chickpea", "cinnamon", "cucumber", "egg", "garlic", "ginger",
		"greek yogurt", "green bean", "lentil", "lettuce", "milk", "mushroom", "oat", "olive oil",
		"onion", "peanut", "potato", "quinoa", "rice", "salmon", "spinach", "sweet potato",
		"tofu", "tomato", "tuna", "turkey", "walnut", "whole wheat", "yogurt",
	}
	out := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)
	normalizedQuery := taxonomy.NormalizeLooseToken(query)
	for _, item := range catalog {
		normalizedItem := taxonomy.NormalizeLooseToken(item)
		if !strings.Contains(normalizedItem, normalizedQuery) {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
		if len(out) >= limit {
			return out
		}
	}
	return out
}
