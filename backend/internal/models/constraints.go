package models

import "time"

type Constraints struct {
	ID                   string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID               string      `gorm:"type:uuid;uniqueIndex;not null"`
	Allergies            StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	Conditions           StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	ExcludedIngredients  StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	HasChronicDisease    bool        `gorm:"not null"`
	ChronicDiseases      StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	TakesMedication      bool        `gorm:"not null"`
	Medications          string      `gorm:"not null"`
	CreatedAt            time.Time   `gorm:"not null;default:now()"`
	UpdatedAt            time.Time   `gorm:"not null;default:now()"`
}

