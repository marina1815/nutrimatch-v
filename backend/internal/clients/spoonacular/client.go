package spoonacular

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout          = 10 * time.Second
	defaultResultCount      = 12
	maxResultCount          = 25
	defaultSearchPath       = "/recipes/complexSearch"
	defaultAutocompletePath = "/food/ingredients/autocomplete"
)

var ErrUpstreamFailure = errors.New("spoonacular upstream failure")

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

type SearchOptions struct {
	Query              string
	Cuisine            []string
	ExcludeCuisine     []string
	Diet               string
	Type               string
	IncludeIngredients []string
	ExcludeIngredients []string
	Intolerances       []string
	MaxReadyTime       int
	MinCalories        float64
	MaxCalories        float64
	MinProtein         float64
	MaxProtein         float64
	MaxCarbs           float64
	MaxFat             float64
	MaxSugar           float64
	MaxSodium          float64
	Number             int
}

type SearchResponse struct {
	Results  []Recipe `json:"results"`
	CacheHit bool     `json:"-"`
}

type Recipe struct {
	ID                  int          `json:"id"`
	Title               string       `json:"title"`
	Summary             string       `json:"summary"`
	Image               string       `json:"image"`
	ReadyInMinutes      int          `json:"readyInMinutes"`
	Servings            int          `json:"servings"`
	Nutrition           Nutrition    `json:"nutrition"`
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

type UpstreamError struct {
	StatusCode int
	Body       string
}

type IngredientSuggestion struct {
	Name string `json:"name"`
}

func (e *UpstreamError) Error() string {
	if e == nil {
		return ErrUpstreamFailure.Error()
	}
	if e.Body == "" {
		return fmt.Sprintf("%s: status %d", ErrUpstreamFailure.Error(), e.StatusCode)
	}
	return fmt.Sprintf("%s: status %d (%s)", ErrUpstreamFailure.Error(), e.StatusCode, e.Body)
}

func (e *UpstreamError) Unwrap() error {
	return ErrUpstreamFailure
}

func (c *Client) Search(ctx context.Context, opts SearchOptions) (*SearchResponse, error) {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	requestURL, err := c.buildSearchURL(opts)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamFailure, err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return nil, &UpstreamError{
			StatusCode: resp.StatusCode,
			Body:       readUpstreamMessage(payload),
		}
	}

	var out SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AutocompleteIngredients(ctx context.Context, query string, number int) ([]IngredientSuggestion, error) {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	requestURL, err := c.buildAutocompleteURL(query, number)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamFailure, err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return nil, &UpstreamError{
			StatusCode: resp.StatusCode,
			Body:       readUpstreamMessage(payload),
		}
	}

	var out []IngredientSuggestion
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) buildSearchURL(opts SearchOptions) (string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return "", errors.New("spoonacular base URL is empty")
	}
	if c.APIKey == "" {
		return "", errors.New("spoonacular API key is empty")
	}

	q := url.Values{}
	q.Set("apiKey", c.APIKey)
	q.Set("addRecipeInformation", "true")
	q.Set("addRecipeNutrition", "true")
	q.Set("fillIngredients", "true")
	q.Set("instructionsRequired", "true")
	q.Set("number", strconv.Itoa(clampCount(opts.Number)))

	if value := strings.TrimSpace(opts.Query); value != "" {
		q.Set("query", value)
	}
	if joined := joinCSV(opts.Cuisine); joined != "" {
		q.Set("cuisine", joined)
	}
	if joined := joinCSV(opts.ExcludeCuisine); joined != "" {
		q.Set("excludeCuisine", joined)
	}
	if value := strings.TrimSpace(opts.Diet); value != "" {
		q.Set("diet", value)
	}
	if value := strings.TrimSpace(opts.Type); value != "" {
		q.Set("type", value)
	}
	if joined := joinCSV(opts.IncludeIngredients); joined != "" {
		q.Set("includeIngredients", joined)
	}
	if joined := joinCSV(opts.ExcludeIngredients); joined != "" {
		q.Set("excludeIngredients", joined)
	}
	if joined := joinCSV(opts.Intolerances); joined != "" {
		q.Set("intolerances", joined)
	}
	if opts.MaxReadyTime > 0 {
		q.Set("maxReadyTime", strconv.Itoa(opts.MaxReadyTime))
	}

	setNumeric(q, "minCalories", opts.MinCalories)
	setNumeric(q, "maxCalories", opts.MaxCalories)
	setNumeric(q, "minProtein", opts.MinProtein)
	setNumeric(q, "maxProtein", opts.MaxProtein)
	setNumeric(q, "maxCarbs", opts.MaxCarbs)
	setNumeric(q, "maxFat", opts.MaxFat)
	setNumeric(q, "maxSugar", opts.MaxSugar)
	setNumeric(q, "maxSodium", opts.MaxSodium)

	return base + defaultSearchPath + "?" + q.Encode(), nil
}

func (c *Client) buildAutocompleteURL(query string, number int) (string, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return "", errors.New("spoonacular base URL is empty")
	}
	if c.APIKey == "" {
		return "", errors.New("spoonacular API key is empty")
	}

	q := strings.TrimSpace(query)
	if q == "" {
		return "", errors.New("ingredient query is empty")
	}

	values := url.Values{}
	values.Set("apiKey", c.APIKey)
	values.Set("query", q)
	values.Set("number", strconv.Itoa(clampAutocompleteCount(number)))

	return base + defaultAutocompletePath + "?" + values.Encode(), nil
}

func clampCount(value int) int {
	if value <= 0 {
		return defaultResultCount
	}
	if value > maxResultCount {
		return maxResultCount
	}
	return value
}

func clampAutocompleteCount(value int) int {
	if value <= 0 {
		return 5
	}
	if value > 10 {
		return 10
	}
	return value
}

func joinCSV(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
	}
	return strings.Join(out, ",")
}

func setNumeric(values url.Values, key string, amount float64) {
	if amount <= 0 {
		return
	}
	values.Set(key, strconv.FormatFloat(amount, 'f', -1, 64))
}

func readUpstreamMessage(payload map[string]any) string {
	for _, key := range []string{"message", "status", "error"} {
		if value, ok := payload[key]; ok {
			if text := strings.TrimSpace(fmt.Sprint(value)); text != "" {
				return text
			}
		}
	}
	return ""
}
