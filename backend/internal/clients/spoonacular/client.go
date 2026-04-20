package spoonacular

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

type SearchOptions struct {
	Query            string
	IncludeIngredients []string
	ExcludeIngredients []string
	Intolerances     []string
	Diets            []string
	Cuisines         []string
	Number           int
}

type SearchResponse struct {
	Results []Recipe `json:"results"`
}

type Recipe struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Summary     string  `json:"summary"`
	Image       string  `json:"image"`
	ReadyInMinutes int  `json:"readyInMinutes"`
	Servings    int     `json:"servings"`
	Nutrition   Nutrition `json:"nutrition"`
	ExtendedIngredients []Ingredient `json:"extendedIngredients"`
}

type Ingredient struct {
	Name string `json:"name"`
}

type Nutrition struct {
	Nutrients []Nutrient `json:"nutrients"`
}

type Nutrient struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

func (c *Client) Search(ctx context.Context, opts SearchOptions) (*SearchResponse, error) {
	if c.APIKey == "" || strings.HasPrefix(c.APIKey, "YOUR_") {
		return c.simulateSearch(opts), nil
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 10 * time.Second}
	}

	base := strings.TrimRight(c.BaseURL, "/")
	endpoint := base + "/recipes/complexSearch"
	q := url.Values{}
	q.Set("apiKey", c.APIKey)
	q.Set("addRecipeInformation", "true")
	q.Set("fillIngredients", "true")
	q.Set("number", fmt.Sprintf("%d", opts.Number))

	if opts.Query != "" {
		q.Set("query", opts.Query)
	}
	if len(opts.IncludeIngredients) > 0 {
		q.Set("includeIngredients", strings.Join(opts.IncludeIngredients, ","))
	}
	if len(opts.ExcludeIngredients) > 0 {
		q.Set("excludeIngredients", strings.Join(opts.ExcludeIngredients, ","))
	}
	if len(opts.Intolerances) > 0 {
		q.Set("intolerances", strings.Join(opts.Intolerances, ","))
	}
	if len(opts.Diets) > 0 {
		q.Set("diet", strings.Join(opts.Diets, ","))
	}
	if len(opts.Cuisines) > 0 {
		q.Set("cuisine", strings.Join(opts.Cuisines, ","))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("spoonacular status %d", resp.StatusCode)
	}

	var out SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) simulateSearch(opts SearchOptions) *SearchResponse {
	return &SearchResponse{
		Results: []Recipe{
			{
				ID:      101,
				Title:   "Saumon Grillé au Quinoa et Avocat",
				Summary: "Un délicieux saumon riche en oméga-3, servi avec du quinoa et de l'avocat frais.",
				Image:   "https://images.unsplash.com/photo-1467003909585-2f8a72700288?w=400",
				ReadyInMinutes: 25,
				Servings: 1,
				Nutrition: Nutrition{
					Nutrients: []Nutrient{
						{Name: "Calories", Amount: 540, Unit: "kcal"},
						{Name: "Protein", Amount: 55, Unit: "g"},
						{Name: "Carbohydrates", Amount: 38, Unit: "g"},
						{Name: "Fat", Amount: 18, Unit: "g"},
						{Name: "Sugar", Amount: 4, Unit: "g"},
						{Name: "Sodium", Amount: 320, Unit: "mg"},
					},
				},
				ExtendedIngredients: []Ingredient{{Name: "salmon"}, {Name: "quinoa"}, {Name: "avocado"}, {Name: "lemon"}},
			},
			{
				ID:      102,
				Title:   "Bowl Protéiné Pois Chiches & Feta",
				Summary: "Légère et riche en protéines végétales et animales, parfaite pour un déjeuner équilibré.",
				Image:   "https://images.unsplash.com/photo-1512621776951-a57141f2eefd?w=400",
				ReadyInMinutes: 15,
				Servings: 1,
				Nutrition: Nutrition{
					Nutrients: []Nutrient{
						{Name: "Calories", Amount: 490, Unit: "kcal"},
						{Name: "Protein", Amount: 50, Unit: "g"},
						{Name: "Carbohydrates", Amount: 42, Unit: "g"},
						{Name: "Fat", Amount: 16, Unit: "g"},
						{Name: "Sugar", Amount: 6, Unit: "g"},
						{Name: "Sodium", Amount: 280, Unit: "mg"},
					},
				},
				ExtendedIngredients: []Ingredient{{Name: "chickpeas"}, {Name: "chicken"}, {Name: "feta"}, {Name: "olive oil"}},
			},
			{
				ID:      103,
				Title:   "Poulet au Curry Doux et Riz Complet",
				Summary: "Un classique équilibré riche en protéines avec des épices douces.",
				Image:   "https://images.unsplash.com/photo-1603133872878-684f208fb84b?w=400",
				ReadyInMinutes: 30,
				Servings: 1,
				Nutrition: Nutrition{
					Nutrients: []Nutrient{
						{Name: "Calories", Amount: 580, Unit: "kcal"},
						{Name: "Protein", Amount: 58, Unit: "g"},
						{Name: "Carbohydrates", Amount: 48, Unit: "g"},
						{Name: "Fat", Amount: 14, Unit: "g"},
						{Name: "Sugar", Amount: 5, Unit: "g"},
						{Name: "Sodium", Amount: 350, Unit: "mg"},
					},
				},
				ExtendedIngredients: []Ingredient{{Name: "chicken"}, {Name: "rice"}, {Name: "curry"}, {Name: "coconut milk"}},
			},
			{
				ID:      104,
				Title:   "Bowl de Lentilles aux Légumes Rôtis & Tahini",
				Summary: "Un bowl végétarien riche en fibres et en protéines végétales.",
				Image:   "https://images.unsplash.com/photo-1540420773420-3366772f4999?w=400",
				ReadyInMinutes: 35,
				Servings: 1,
				Nutrition: Nutrition{
					Nutrients: []Nutrient{
						{Name: "Calories", Amount: 520, Unit: "kcal"},
						{Name: "Protein", Amount: 52, Unit: "g"},
						{Name: "Carbohydrates", Amount: 50, Unit: "g"},
						{Name: "Fat", Amount: 14, Unit: "g"},
						{Name: "Sugar", Amount: 8, Unit: "g"},
						{Name: "Sodium", Amount: 290, Unit: "mg"},
					},
				},
				ExtendedIngredients: []Ingredient{{Name: "lentils"}, {Name: "sweet potato"}, {Name: "broccoli"}, {Name: "tahini"}, {Name: "egg"}},
			},
			{
				ID:      105,
				Title:   "Steak de Thon Sésame & Edamame",
				Summary: "Un plat léger riche en protéines et en oméga-3 avec une touche asiatique.",
				Image:   "https://images.unsplash.com/photo-1519984388953-d2406bc725e1?w=400",
				ReadyInMinutes: 20,
				Servings: 1,
				Nutrition: Nutrition{
					Nutrients: []Nutrient{
						{Name: "Calories", Amount: 510, Unit: "kcal"},
						{Name: "Protein", Amount: 60, Unit: "g"},
						{Name: "Carbohydrates", Amount: 22, Unit: "g"},
						{Name: "Fat", Amount: 20, Unit: "g"},
						{Name: "Sugar", Amount: 3, Unit: "g"},
						{Name: "Sodium", Amount: 380, Unit: "mg"},
					},
				},
				ExtendedIngredients: []Ingredient{{Name: "tuna"}, {Name: "sesame"}, {Name: "edamame"}, {Name: "soy sauce"}},
			},
			{
				ID:      106,
				Title:   "Wrap de Dinde aux Crudités & Houmous",
				Summary: "Un wrap frais et rapide à préparer, idéal en pause déjeuner.",
				Image:   "https://images.unsplash.com/photo-1600335895229-6e75511892c8?w=400",
				ReadyInMinutes: 10,
				Servings: 1,
				Nutrition: Nutrition{
					Nutrients: []Nutrient{
						{Name: "Calories", Amount: 500, Unit: "kcal"},
						{Name: "Protein", Amount: 50, Unit: "g"},
						{Name: "Carbohydrates", Amount: 38, Unit: "g"},
						{Name: "Fat", Amount: 16, Unit: "g"},
						{Name: "Sugar", Amount: 4, Unit: "g"},
						{Name: "Sodium", Amount: 310, Unit: "mg"},
					},
				},
				ExtendedIngredients: []Ingredient{{Name: "turkey"}, {Name: "hummus"}, {Name: "lettuce"}, {Name: "tomato"}},
			},
		},
	}
}



