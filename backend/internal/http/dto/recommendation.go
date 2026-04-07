package dto

type RecommendationResponse struct {
	ProfileID string                 `json:"profileId"`
	Meals     []MealRecommendation   `json:"meals"`
}

type MealRecommendation struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Calories    float64  `json:"calories"`
	Protein     float64  `json:"protein"`
	Carbs       float64  `json:"carbs"`
	Fat         float64  `json:"fat"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Ingredients []string `json:"ingredients"`
	MatchReason string   `json:"matchReason"`
}

