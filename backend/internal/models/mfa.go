package models

import "time"

type TOTPSecret struct {
	UserID           string     `gorm:"type:uuid;primaryKey"`
	SecretCiphertext string     `gorm:"not null"`
	Enabled          bool       `gorm:"not null;default:false"`
	ConfirmedAt      *time.Time `gorm:"default:null"`
	CreatedAt        time.Time  `gorm:"not null;default:now()"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()"`
}

func (TOTPSecret) TableName() string {
	return "identity.totp_secrets"
}

type WebAuthnCredential struct {
	ID             string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID         string     `gorm:"type:uuid;index;not null"`
	CredentialID   string     `gorm:"uniqueIndex;not null"`
	CredentialJSON JSONMap    `gorm:"type:jsonb;not null;default:'{}'"`
	DisplayName    string     `gorm:"not null;default:''"`
	CreatedAt      time.Time  `gorm:"not null;default:now()"`
	LastUsedAt     *time.Time `gorm:"default:null"`
}

func (WebAuthnCredential) TableName() string {
	return "identity.webauthn_credentials"
}

type WebAuthnChallenge struct {
	ID          string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      string     `gorm:"type:uuid;index;not null"`
	Kind        string     `gorm:"not null"`
	SessionData JSONMap    `gorm:"type:jsonb;not null;default:'{}'"`
	ExpiresAt   time.Time  `gorm:"not null"`
	ConsumedAt  *time.Time `gorm:"default:null"`
	CreatedAt   time.Time  `gorm:"not null;default:now()"`
}

func (WebAuthnChallenge) TableName() string {
	return "identity.webauthn_challenges"
}

type MFALoginChallenge struct {
	ID              string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          string      `gorm:"type:uuid;index;not null"`
	PreferredMethod string      `gorm:"not null;default:''"`
	AllowedMethods  StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	ExpiresAt       time.Time   `gorm:"not null"`
	ConsumedAt      *time.Time  `gorm:"default:null"`
	UserAgentHash   string      `gorm:"not null;default:''"`
	IPHash          string      `gorm:"not null;default:''"`
	CreatedAt       time.Time   `gorm:"not null;default:now()"`
}

func (MFALoginChallenge) TableName() string {
	return "identity.mfa_login_challenges"
}
