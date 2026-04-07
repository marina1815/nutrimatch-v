package models

import "time"

type Profile struct {
	ID         string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     string    `gorm:"type:uuid;uniqueIndex;not null"`
	Age        int       `gorm:"not null"`
	Sex        string    `gorm:"not null"`
	Weight     float64   `gorm:"not null"`
	Height     float64   `gorm:"not null"`
	Profession string    `gorm:"not null"`
	City       string    `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null;default:now()"`
	UpdatedAt  time.Time `gorm:"not null;default:now()"`
}

