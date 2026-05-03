package models

import "time"

type ExternalIdentity struct {
	ID            string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        string    `gorm:"type:uuid;index;not null"`
	Provider      string    `gorm:"index;not null"`
	Issuer        string    `gorm:"index;not null"`
	Subject       string    `gorm:"index;not null"`
	Email         string    `gorm:"index;not null;default:''"`
	EmailVerified bool      `gorm:"not null;default:false"`
	LastLoginAt   time.Time `gorm:"not null;default:now()"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
	UpdatedAt     time.Time `gorm:"not null;default:now()"`
}

func (ExternalIdentity) TableName() string {
	return "identity.external_identities"
}
