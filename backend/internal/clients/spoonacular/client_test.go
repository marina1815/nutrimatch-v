package spoonacular

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchBuildsAdvancedQuery(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(SearchResponse{})
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		APIKey:  "secret-key",
	}

	_, err := client.Search(context.Background(), SearchOptions{
		Query:              "healthy chicken",
		Cuisine:            []string{"mediterranean"},
		ExcludeCuisine:     []string{"american"},
		Type:               "main course",
		IncludeIngredients: []string{"chicken", "quinoa"},
		ExcludeIngredients: []string{"bacon"},
		Intolerances:       []string{"dairy", "tree nut"},
		MaxReadyTime:       30,
		MaxCalories:        650,
		MinProtein:         20,
		MaxCarbs:           55,
		MaxFat:             22,
		MaxSugar:           18,
		MaxSodium:          700,
		Number:             99,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, fragment := range []string{
		"query=healthy+chicken",
		"cuisine=mediterranean",
		"excludeCuisine=american",
		"type=main+course",
		"includeIngredients=chicken%2Cquinoa",
		"excludeIngredients=bacon",
		"intolerances=dairy%2Ctree+nut",
		"maxReadyTime=30",
		"maxCalories=650",
		"minProtein=20",
		"maxCarbs=55",
		"maxFat=22",
		"maxSugar=18",
		"maxSodium=700",
		"number=25",
		"addRecipeNutrition=true",
	} {
		if !strings.Contains(receivedQuery, fragment) {
			t.Fatalf("expected query to contain %q, got %q", fragment, receivedQuery)
		}
	}

	if !strings.Contains(receivedQuery, "apiKey=secret-key") {
		t.Fatalf("expected apiKey query parameter to be present")
	}
}

func TestSearchReturnsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"quota exceeded"}`, http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		APIKey:  "secret-key",
	}

	_, err := client.Search(context.Background(), SearchOptions{})
	if err == nil {
		t.Fatalf("expected error")
	}

	upstreamErr, ok := err.(*UpstreamError)
	if !ok {
		t.Fatalf("expected UpstreamError, got %T", err)
	}
	if upstreamErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected status code: %d", upstreamErr.StatusCode)
	}
}

func TestAutocompleteIngredientsBuildsQuery(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode([]IngredientSuggestion{{Name: "paprika"}})
	}))
	defer server.Close()

	client := &Client{
		BaseURL: server.URL,
		APIKey:  "secret-key",
	}

	out, err := client.AutocompleteIngredients(context.Background(), "pap", 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 || out[0].Name != "paprika" {
		t.Fatalf("unexpected suggestions: %#v", out)
	}

	for _, fragment := range []string{
		"query=pap",
		"number=10",
		"apiKey=secret-key",
	} {
		if !strings.Contains(receivedQuery, fragment) {
			t.Fatalf("expected autocomplete query to contain %q, got %q", fragment, receivedQuery)
		}
	}
}
