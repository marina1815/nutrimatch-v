package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/marina1815/nutrimatch/internal/repository"
)

type fakeIngredientCatalog struct {
	allergies []repository.CatalogOption
}

func (f fakeIngredientCatalog) Suggest(context.Context, string, int) ([]repository.CatalogOption, error) {
	return []repository.CatalogOption{}, nil
}

func (f fakeIngredientCatalog) ListAllergies(context.Context) ([]repository.CatalogOption, error) {
	return f.allergies, nil
}

func TestProfileAllergyOptionsHideTechnicalLabelsAndDeduplicate(t *testing.T) {
	handler := &ProfileHandler{
		Ingredients: fakeIngredientCatalog{allergies: []repository.CatalogOption{
			{Value: "egg allergy", Label: "Oeufs allergy", Source: "liste des allergies.xlsx"},
			{Value: "cross between foods", Label: "Cross between foods", Source: "croise.xlsx"},
			{Value: "pollen food allergy syndrome pfas", Label: "Pollen food pfas", Source: "liste des allergies.xlsx"},
			{Value: "oral allergy syndrome oas", Label: "Oral oas", Source: "liste des allergies.xlsx"},
			{Value: "non allergy", Label: "non allergy", Source: "liste des allergies.xlsx"},
			{Value: "mustard", Label: "Mustard allergy", Source: "liste des allergies.xlsx"},
		}},
	}

	options, err := handler.listAllergyOptions(context.Background())
	if err != nil {
		t.Fatalf("unexpected allergy taxonomy error: %v", err)
	}

	eggCount := 0
	for _, option := range options {
		combined := strings.ToLower(option.Value + " " + option.Label)
		for _, forbidden := range []string{"cross", "oas", "pfas", "non allergy", "allergy:"} {
			if strings.Contains(combined, forbidden) {
				t.Fatalf("technical allergy option leaked to UI: %+v", option)
			}
		}
		if option.Value == "egg" || strings.EqualFold(option.Label, "Œufs") || strings.EqualFold(option.Label, "Oeufs") {
			eggCount++
		}
	}
	if eggCount != 1 {
		t.Fatalf("expected one deduplicated egg option, got %d in %+v", eggCount, options)
	}
}
