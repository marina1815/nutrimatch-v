package models

import "time"

type Session struct {
	ID               string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           string     `gorm:"type:uuid;index;not null"`
	AuthMethod       string     `gorm:"not null;default:'local'"`
	RefreshTokenHash string     `gorm:"uniqueIndex;not null"`
	ExpiresAt        time.Time  `gorm:"not null"`
	IdleExpiresAt    time.Time  `gorm:"not null"`
	CreatedAt        time.Time  `gorm:"not null;default:now()"`
	LastSeenAt       time.Time  `gorm:"not null;default:now()"`
	RevokedAt        *time.Time `gorm:"default:null"`
	UserAgentHash    string     `gorm:"not null;default:''"`
	IPHash           string     `gorm:"not null;default:''"`
}

func (Session) TableName() string {
	return "identity.sessions"
}
