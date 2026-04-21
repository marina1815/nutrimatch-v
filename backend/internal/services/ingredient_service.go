package services

import (
	"context"
	"strings"

	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type IngredientAutocompleteClient interface {
	AutocompleteIngredients(ctx context.Context, query string, number int) ([]spoonacular.IngredientSuggestion, error)
}

type IngredientService struct {
	Client IngredientAutocompleteClient
}

func (s *IngredientService) Suggest(ctx context.Context, query string, limit int) ([]string, error) {
	if s == nil || s.Client == nil {
		return []string{}, nil
	}

	cleaned := validation.NormalizeString(query)
	if len(cleaned) < 2 {
		return []string{}, nil
	}

	suggestions, err := s.Client.AutocompleteIngredients(ctx, cleaned, limit)
	if err != nil {
		return nil, err
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
	return out, nil
}
