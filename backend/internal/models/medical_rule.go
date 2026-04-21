package models

import "time"

type MedicalRule struct {
	ID                 string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code               string      `gorm:"uniqueIndex;not null"`
	ConditionKey       string      `gorm:"index;not null"`
	MedicationPattern  string      `gorm:"index;not null;default:''"`
	BlockedIngredients StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	BlockedTags        StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	RequiredTags       StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	MaxCalories        float64     `gorm:"not null;default:0"`
	MaxProteinGrams    float64     `gorm:"not null;default:0"`
	MaxCarbsGrams      float64     `gorm:"not null;default:0"`
	MaxFatGrams        float64     `gorm:"not null;default:0"`
	MaxSugarGrams      float64     `gorm:"not null;default:0"`
	MaxSodiumMg        float64     `gorm:"not null;default:0"`
	MinProteinGrams    float64     `gorm:"not null;default:0"`
	Severity           string      `gorm:"not null;default:'high'"`
	Rationale          string      `gorm:"not null"`
	Active             bool        `gorm:"not null;default:true"`
	CreatedAt          time.Time   `gorm:"not null;default:now()"`
	UpdatedAt          time.Time   `gorm:"not null;default:now()"`
}

func (MedicalRule) TableName() string {
	return "catalog.medical_rules"
}
