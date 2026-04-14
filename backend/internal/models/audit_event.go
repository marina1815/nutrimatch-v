package models

import "time"

type AuditEvent struct {
	ID            string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        string    `gorm:"type:uuid;index;not null;default:''"`
	SessionID     string    `gorm:"type:uuid;index;not null;default:''"`
	EventType     string    `gorm:"index;not null"`
	ResourceType  string    `gorm:"index;not null"`
	ResourceID    string    `gorm:"index;not null;default:''"`
	Outcome       string    `gorm:"index;not null"`
	IP            string    `gorm:"not null;default:''"`
	UserAgent     string    `gorm:"not null;default:''"`
	RequestID     string    `gorm:"index;not null;default:''"`
	Details       JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	ExternalTrace JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	OccurredAt    time.Time `gorm:"not null;default:now()"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
}

func (AuditEvent) TableName() string {
	return "security.audit_events"
}
