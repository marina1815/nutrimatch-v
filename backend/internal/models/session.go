package models

import "time"

type Session struct {
	ID               string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           string     `gorm:"type:uuid;index;not null"`
	RefreshTokenHash string     `gorm:"uniqueIndex;not null"`
	ExpiresAt        time.Time  `gorm:"not null"`
	CreatedAt        time.Time  `gorm:"not null;default:now()"`
	RevokedAt        *time.Time `gorm:"default:null"`
	UserAgent        string     `gorm:"not null"`
	IP              string     `gorm:"not null"`
}

