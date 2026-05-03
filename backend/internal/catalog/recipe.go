package catalog

import "errors"

var ErrUpstreamFailure = errors.New("catalog unavailable")

type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return "catalog upstream error"
}

type SearchOptions struct {
	Query              string
	Cuisine            []string
	ExcludeCuisine     []string
	Type               string
	IncludeIngredients []string
	ExcludeIngredients []string
	Intolerances       []string
	Number             int
	MaxCalories        float64
	MinProtein         float64
	MaxProtein         float64
	MaxCarbs           float64
	MaxFat             float64
	MaxSugar           float64
	MaxSodium          float64
}

type SearchResponse struct {
	Results  []Recipe
	CacheHit bool
}

type Recipe struct {
	ID                  int
	Title               string
	Summary             string
	Tags                []string
	ReadyInMinutes      int
	Servings            int
	ExtendedIngredients []Ingredient
	Nutrition           Nutrition
}

type Ingredient struct {
	Name string
}

type Nutrition struct {
	Nutrients []Nutrient
}

type Nutrient struct {
	Name   string
	Amount float64
	Unit   string
}
