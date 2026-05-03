package localdata

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog.seed.json
var catalogSeedJSON []byte

type CatalogSeed struct {
	Version             int                          `json:"version"`
	Ingredients         []SeedIngredient             `json:"ingredients"`
	Allergies           []SeedAllergy                `json:"allergies"`
	Recipes             []SeedRecipe                 `json:"recipes"`
	IngredientAllergies []SeedIngredientAllergy      `json:"ingredientAllergies"`
	CrossAllergies      []SeedIngredientCrossAllergy `json:"crossAllergies"`
}

type SeedIngredient struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Source      string `json:"source"`
}

type SeedAllergy struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	Source      string `json:"source"`
}

type SeedRecipe struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Ingredients []string      `json:"ingredients"`
	Nutrition   SeedNutrition `json:"nutrition"`
	Source      string        `json:"source"`
}

type SeedNutrition struct {
	Calories float64 `json:"calories"`
	Protein  float64 `json:"protein"`
	Carbs    float64 `json:"carbs"`
	Fat      float64 `json:"fat"`
	Sugar    float64 `json:"sugar"`
	SodiumMg float64 `json:"sodiumMg"`
}

type SeedIngredientAllergy struct {
	IngredientKey string `json:"ingredientKey"`
	AllergyKey    string `json:"allergyKey"`
	Source        string `json:"source"`
}

type SeedIngredientCrossAllergy struct {
	IngredientKey      string `json:"ingredientKey"`
	CrossIngredientKey string `json:"crossIngredientKey"`
	AllergyKey         string `json:"allergyKey"`
}

func LoadCatalogSeed() (*CatalogSeed, error) {
	var seed CatalogSeed
	if err := json.Unmarshal(catalogSeedJSON, &seed); err != nil {
		return nil, err
	}
	return &seed, nil
}
