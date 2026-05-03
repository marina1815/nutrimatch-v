package dto

import "time"

type RecommendationResponse struct {
	RunID                 string               `json:"runId"`
	ProfileID             string               `json:"profileId"`
	Meals                 []MealRecommendation `json:"meals"`
	ActiveChoice          *MealChoiceResponse  `json:"activeChoice,omitempty"`
	GeneratedAt           time.Time            `json:"generatedAt"`
	ValidUntil            time.Time            `json:"validUntil"`
	NextRefreshAt         time.Time            `json:"nextRefreshAt"`
	SecondsUntilRefresh   int                  `json:"secondsUntilRefresh"`
	SelectionMode         string               `json:"selectionMode"`
	AIExplanationApplied  bool                 `json:"aiExplanationApplied"`
	AIValidationApplied   bool                 `json:"aiValidationApplied"`
	AIRejectedMealCount   int                  `json:"aiRejectedMealCount"`
	AIReplacementCount    int                  `json:"aiReplacementCount"`
	AISkippedReason       string               `json:"aiSkippedReason,omitempty"`
	AIOutputIgnoredReason string               `json:"aiOutputIgnoredReason,omitempty"`
}

type CatalogItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type MealRecommendation struct {
	ID                  string        `json:"id"`
	Title               string        `json:"title"`
	Calories            float64       `json:"calories"`
	Protein             float64       `json:"protein"`
	Carbs               float64       `json:"carbs"`
	Fat                 float64       `json:"fat"`
	Sugar               float64       `json:"sugar"`
	SodiumMg            float64       `json:"sodiumMg"`
	Tags                []string      `json:"tags"`
	Description         string        `json:"description"`
	Ingredients         []CatalogItem `json:"ingredients"`
	MatchReason         string        `json:"matchReason"`
	Source              string        `json:"source"`
	Score               float64       `json:"score"`
	NutritionConfidence string        `json:"nutritionConfidence"`
	NutritionSource     string        `json:"nutritionSource"`
	SafetyWarnings      []string      `json:"safetyWarnings"`
	AIExplanation       string        `json:"aiExplanation,omitempty"`
}

type MealChoiceResponse struct {
	ProfileID             string             `json:"profileId"`
	Meal                  MealRecommendation `json:"meal"`
	PreparationGuide      string             `json:"preparationGuide"`
	Substitutions         []MealSubstitution `json:"substitutions"`
	AIApplied             bool               `json:"aiApplied"`
	AISkippedReason       string             `json:"aiSkippedReason,omitempty"`
	AIOutputIgnoredReason string             `json:"aiOutputIgnoredReason,omitempty"`
	ChosenAt              time.Time          `json:"chosenAt"`
	ExcludedUntil         time.Time          `json:"excludedUntil"`
}

type MealSubstitution struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Reason string `json:"reason"`
}

type RecommendationExplanationResponse struct {
	RunID                 string         `json:"runId"`
	ProfileID             string         `json:"profileId"`
	MealID                string         `json:"mealId"`
	Explanation           string         `json:"explanation"`
	AIExplanation         string         `json:"aiExplanation,omitempty"`
	AIExplanationApplied  bool           `json:"aiExplanationApplied"`
	AISkippedReason       string         `json:"aiSkippedReason,omitempty"`
	AIOutputIgnoredReason string         `json:"aiOutputIgnoredReason,omitempty"`
	AcceptedReasons       []string       `json:"acceptedReasons"`
	RejectedReasons       []string       `json:"rejectedReasons"`
	ScoreBreakdown        map[string]any `json:"scoreBreakdown"`
	FilterDecisions       map[string]any `json:"filterDecisions"`
	SourceProvenance      map[string]any `json:"sourceProvenance"`
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
