package models

import "time"

type Constraints struct {
	ID                  string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID              string      `gorm:"type:uuid;uniqueIndex;not null"`
	Allergies           StringSlice `gorm:"-"`
	Conditions          StringSlice `gorm:"-"`
	ExcludedIngredients StringSlice `gorm:"-"`
	HasChronicDisease   bool        `gorm:"not null"`
	ChronicDiseases     StringSlice `gorm:"-"`
	TakesMedication     bool        `gorm:"not null"`
	Medications         string      `gorm:"not null"`
	CreatedAt           time.Time   `gorm:"not null;default:now()"`
	UpdatedAt           time.Time   `gorm:"not null;default:now()"`
}

func (Constraints) TableName() string {
	return "health.constraints"
}
