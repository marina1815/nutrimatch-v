package dto

type NutritionProfileResponse struct {
	ProfileID             string         `json:"profileId"`
	BMI                   float64        `json:"bmi"`
	BMICategory           string         `json:"bmiCategory"`
	BMR                   float64        `json:"bmr"`
	EstimatedCalories     float64        `json:"estimatedCalories"`
	TargetCalories        float64        `json:"targetCalories"`
	TargetProteinGrams    float64        `json:"targetProteinGrams"`
	TargetCarbsGrams      float64        `json:"targetCarbsGrams"`
	TargetFatGrams        float64        `json:"targetFatGrams"`
	MaxMealCalories       float64        `json:"maxMealCalories"`
	MinProteinPerMeal     float64        `json:"minProteinPerMeal"`
	MaxCarbsPerMeal       float64        `json:"maxCarbsPerMeal"`
	MaxFatPerMeal         float64        `json:"maxFatPerMeal"`
	MaxSugarPerMeal       float64        `json:"maxSugarPerMeal"`
	MaxSodiumMgPerMeal    float64        `json:"maxSodiumMgPerMeal"`
	DerivedRestrictions   []string       `json:"derivedRestrictions"`
	DerivedExcluded       []string       `json:"derivedExcluded"`
	RecommendedMealStyles []string       `json:"recommendedMealStyles"`
	Metadata              map[string]any `json:"metadata"`
}
