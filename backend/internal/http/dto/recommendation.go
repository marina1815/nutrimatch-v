package dto

type RecommendationResponse struct {
	RunID     string               `json:"runId"`
	ProfileID string               `json:"profileId"`
	Meals     []MealRecommendation `json:"meals"`
}

type MealRecommendation struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Calories    float64  `json:"calories"`
	Protein     float64  `json:"protein"`
	Carbs       float64  `json:"carbs"`
	Fat         float64  `json:"fat"`
	Sugar       float64  `json:"sugar"`
	SodiumMg    float64  `json:"sodiumMg"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Ingredients []string `json:"ingredients"`
	MatchReason string   `json:"matchReason"`
	Source      string   `json:"source"`
	Score       float64  `json:"score"`
}

type RecommendationExplanationResponse struct {
	RunID            string         `json:"runId"`
	ProfileID        string         `json:"profileId"`
	MealID           string         `json:"mealId"`
	Explanation      string         `json:"explanation"`
	AcceptedReasons  []string       `json:"acceptedReasons"`
	RejectedReasons  []string       `json:"rejectedReasons"`
	ScoreBreakdown   map[string]any `json:"scoreBreakdown"`
	FilterDecisions  map[string]any `json:"filterDecisions"`
	SourceProvenance map[string]any `json:"sourceProvenance"`
}

type RecommendationTraceResponse struct {
	RunID           string         `json:"runId"`
	ProfileID       string         `json:"profileId"`
	Status          string         `json:"status"`
	SourceSummary   map[string]any `json:"sourceSummary"`
	DecisionSummary map[string]any `json:"decisionSummary"`
	ExternalTrace   map[string]any `json:"externalTrace"`
	Candidates      []MealTrace    `json:"candidates"`
}

type MealTrace struct {
	MealID           string         `json:"mealId"`
	Title            string         `json:"title"`
	Accepted         bool           `json:"accepted"`
	FinalRank        int            `json:"finalRank"`
	FinalScore       float64        `json:"finalScore"`
	AcceptedReasons  []string       `json:"acceptedReasons"`
	RejectedReasons  []string       `json:"rejectedReasons"`
	ScoreBreakdown   map[string]any `json:"scoreBreakdown"`
	FilterDecisions  map[string]any `json:"filterDecisions"`
	SourceProvenance map[string]any `json:"sourceProvenance"`
}
