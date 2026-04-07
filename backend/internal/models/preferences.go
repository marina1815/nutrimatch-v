package models

import "time"

type Preferences struct {
	ID          string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      string      `gorm:"type:uuid;uniqueIndex;not null"`
	Likes       StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	Dislikes    StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	MealStyles  StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	MealsPerDay int         `gorm:"not null"`
	CreatedAt   time.Time   `gorm:"not null;default:now()"`
	UpdatedAt   time.Time   `gorm:"not null;default:now()"`
}

