package models

import "time"

type NutritionProfile struct {
	ID                    string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID                string      `gorm:"type:uuid;uniqueIndex;not null"`
	ProfileID             string      `gorm:"type:uuid;not null"`
	BMI                   float64     `gorm:"not null"`
	BMICategory           string      `gorm:"not null"`
	BMR                   float64     `gorm:"not null"`
	EstimatedCalories     float64     `gorm:"not null"`
	TargetCalories        float64     `gorm:"not null"`
	TargetProteinGrams    float64     `gorm:"not null"`
	TargetCarbsGrams      float64     `gorm:"not null"`
	TargetFatGrams        float64     `gorm:"not null"`
	MaxMealCalories       float64     `gorm:"not null"`
	MinProteinPerMeal     float64     `gorm:"not null"`
	MaxCarbsPerMeal       float64     `gorm:"not null"`
	MaxFatPerMeal         float64     `gorm:"not null"`
	MaxSugarPerMeal       float64     `gorm:"not null"`
	MaxSodiumMgPerMeal    float64     `gorm:"not null"`
	DerivedRestrictions   StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	DerivedExcluded       StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	RecommendedMealStyles StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	Metadata              JSONMap     `gorm:"type:jsonb;not null;default:'{}'"`
	CalculatedAt          time.Time   `gorm:"not null;default:now()"`
	CreatedAt             time.Time   `gorm:"not null;default:now()"`
	UpdatedAt             time.Time   `gorm:"not null;default:now()"`
}

func (NutritionProfile) TableName() string {
	return "health.nutrition_profiles"
}
