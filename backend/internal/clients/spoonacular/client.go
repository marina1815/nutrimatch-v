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

