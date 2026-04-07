package models

import "time"

type Lifestyle struct {
	ID            string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        string    `gorm:"type:uuid;uniqueIndex;not null"`
	ActivityLevel string    `gorm:"not null"`
	LifestyleType string    `gorm:"not null"`
	Goal          string    `gorm:"not null"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

