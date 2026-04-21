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
	Key              string    `gorm:"primaryKey;column:key"`
	DisplayName      string    `gorm:"not null"`
	SpoonacularValue string    `gorm:"not null"`
	Source           string    `gorm:"not null;default:'system'"`
	CreatedAt        time.Time `gorm:"not null;default:now()"`
	UpdatedAt        time.Time `gorm:"not null;default:now()"`
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

type CatalogMealStyle struct {
	Key         string    `gorm:"primaryKey;column:key"`
	DisplayName string    `gorm:"not null"`
	Source      string    `gorm:"not null;default:'system'"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (CatalogMealStyle) TableName() string {
	return "catalog.meal_styles"
}

type CatalogMealType struct {
	Key         string    `gorm:"primaryKey;column:key"`
	DisplayName string    `gorm:"not null"`
	Source      string    `gorm:"not null;default:'system'"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (CatalogMealType) TableName() string {
	return "catalog.meal_types"
}

type CatalogCuisine struct {
	Key         string    `gorm:"primaryKey;column:key"`
	DisplayName string    `gorm:"not null"`
	Source      string    `gorm:"not null;default:'system'"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (CatalogCuisine) TableName() string {
	return "catalog.cuisines"
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

type ProfileMealStyle struct {
	UserID       string    `gorm:"type:uuid;primaryKey"`
	MealStyleKey string    `gorm:"primaryKey;column:meal_style_key"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
}

func (ProfileMealStyle) TableName() string {
	return "health.profile_meal_styles"
}

type ProfileMealType struct {
	UserID      string    `gorm:"type:uuid;primaryKey"`
	MealTypeKey string    `gorm:"primaryKey;column:meal_type_key"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
}

func (ProfileMealType) TableName() string {
	return "health.profile_meal_types"
}

type ProfileCuisine struct {
	UserID     string    `gorm:"type:uuid;primaryKey"`
	CuisineKey string    `gorm:"primaryKey;column:cuisine_key"`
	Kind       string    `gorm:"primaryKey"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
}

func (ProfileCuisine) TableName() string {
	return "health.profile_cuisines"
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
