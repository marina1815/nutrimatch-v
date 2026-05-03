package models

import "time"

type CatalogIngredient struct {
	Key         string    `gorm:"primaryKey;column:key"`
	DisplayName string    `gorm:"not null"`
	Source      string    `gorm:"not null;default:'user'"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (CatalogIngredient) TableName() string {
	return "catalog.ingredients"
}

type CatalogIntolerance struct {
	Key           string    `gorm:"primaryKey;column:key"`
	DisplayName   string    `gorm:"not null"`
	ProviderValue string    `gorm:"not null"`
	Source        string    `gorm:"not null;default:'system'"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

func (CatalogIntolerance) TableName() string {
	return "catalog.intolerances"
}

type CatalogCondition struct {
	Key         string    `gorm:"primaryKey;column:key"`
	DisplayName string    `gorm:"not null"`
	Source      string    `gorm:"not null;default:'system'"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (CatalogCondition) TableName() string {
	return "catalog.conditions"
}

type ProfilePreferenceIngredient struct {
	UserID        string    `gorm:"type:uuid;primaryKey"`
	IngredientKey string    `gorm:"primaryKey;column:ingredient_key"`
	Kind          string    `gorm:"primaryKey"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
}

func (ProfilePreferenceIngredient) TableName() string {
	return "health.profile_preference_ingredients"
}

type ProfileIntolerance struct {
	UserID         string    `gorm:"type:uuid;primaryKey"`
	IntoleranceKey string    `gorm:"primaryKey;column:intolerance_key"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
}

func (ProfileIntolerance) TableName() string {
	return "health.profile_intolerances"
}

type ProfileCondition struct {
	UserID       string    `gorm:"type:uuid;primaryKey"`
	ConditionKey string    `gorm:"primaryKey;column:condition_key"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
}

func (ProfileCondition) TableName() string {
	return "health.profile_conditions"
}

type ProfileChronicCondition struct {
	UserID       string    `gorm:"type:uuid;primaryKey"`
	ConditionKey string    `gorm:"primaryKey;column:condition_key"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
}

func (ProfileChronicCondition) TableName() string {
	return "health.profile_chronic_conditions"
}
