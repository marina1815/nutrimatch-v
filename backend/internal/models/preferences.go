package models

import "time"

type Preferences struct {
	ID        string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    string      `gorm:"type:uuid;uniqueIndex;not null"`
	Likes     StringSlice `gorm:"-"`
	Dislikes  StringSlice `gorm:"-"`
	CreatedAt time.Time   `gorm:"not null;default:now()"`
	UpdatedAt time.Time   `gorm:"not null;default:now()"`
}

func (Preferences) TableName() string {
	return "health.preferences"
}
