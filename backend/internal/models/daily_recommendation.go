package models

import "time"

type DailyRecommendationSet struct {
	ID                 string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID             string    `gorm:"type:uuid;index;not null"`
	ProfileID          string    `gorm:"type:uuid;index;not null"`
	NutritionProfileID string    `gorm:"type:uuid;index;not null"`
	RunID              string    `gorm:"type:uuid;index;not null"`
	QuerySignature     string    `gorm:"index;not null"`
	Status             string    `gorm:"not null;default:'completed'"`
	SelectionMode      string    `gorm:"not null;default:'deterministic_fallback'"`
	ValidFrom          time.Time `gorm:"not null"`
	ValidUntil         time.Time `gorm:"index;not null"`
	SourceSummary      JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	DecisionSummary    JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt          time.Time `gorm:"not null;default:now()"`
}

func (DailyRecommendationSet) TableName() string {
	return "recommendation.daily_sets"
}

type DailyRecommendationMeal struct {
	ID                  string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SetID               string      `gorm:"type:uuid;index;not null"`
	UserID              string      `gorm:"type:uuid;index;not null"`
	ProfileID           string      `gorm:"type:uuid;index;not null"`
	RecipeID            string      `gorm:"index;not null"`
	Title               string      `gorm:"not null"`
	FinalRank           int         `gorm:"not null;default:0"`
	FinalScore          float64     `gorm:"not null;default:0"`
	Calories            float64     `gorm:"not null;default:0"`
	Protein             float64     `gorm:"not null;default:0"`
	Carbs               float64     `gorm:"not null;default:0"`
	Fat                 float64     `gorm:"not null;default:0"`
	Sugar               float64     `gorm:"not null;default:0"`
	SodiumMg            float64     `gorm:"not null;default:0"`
	Ingredients         StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	AIExplanation       string      `gorm:"not null;default:''"`
	MatchReason         string      `gorm:"not null;default:''"`
	NutritionConfidence string      `gorm:"not null;default:'estimated'"`
	NutritionSource     string      `gorm:"not null;default:'local_catalog_nutrition_estimate'"`
	SafetyWarnings      StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	SourceProvenance    JSONMap     `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt           time.Time   `gorm:"not null;default:now()"`
}

func (DailyRecommendationMeal) TableName() string {
	return "recommendation.daily_set_meals"
}

type RecipeChoice struct {
	ID                    string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SetID                 string      `gorm:"type:uuid;index;not null"`
	UserID                string      `gorm:"type:uuid;index;not null"`
	ProfileID             string      `gorm:"type:uuid;index;not null"`
	RecipeID              string      `gorm:"index;not null"`
	Title                 string      `gorm:"not null"`
	Ingredients           StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	PreparationGuide      string      `gorm:"not null;default:''"`
	Substitutions         JSONMap     `gorm:"type:jsonb;not null;default:'{}'"`
	AIApplied             bool        `gorm:"not null;default:false"`
	AISkippedReason       string      `gorm:"not null;default:''"`
	AIOutputIgnoredReason string      `gorm:"not null;default:''"`
	ChosenAt              time.Time   `gorm:"index;not null"`
	ExpiresAt             time.Time   `gorm:"index;not null"`
	CreatedAt             time.Time   `gorm:"not null;default:now()"`
}

func (RecipeChoice) TableName() string {
	return "recommendation.recipe_choices"
}
