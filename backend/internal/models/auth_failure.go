package models

import "time"

type AuthFailure struct {
	ID         string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	EmailHash  string    `gorm:"index;not null;default:''"`
	IPHash     string    `gorm:"index;not null;default:''"`
	Reason     string    `gorm:"not null"`
	OccurredAt time.Time `gorm:"not null;default:now()"`
}

func (AuthFailure) TableName() string {
	return "security.auth_failures"
}
