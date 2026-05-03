package models

import "time"

type RateLimitBucket struct {
	Key        string    `gorm:"primaryKey"`
	BucketType string    `gorm:"not null"`
	Tokens     float64   `gorm:"not null;default:0"`
	UpdatedAt  time.Time `gorm:"not null;default:now()"`
}

func (RateLimitBucket) TableName() string {
	return "security.rate_limit_buckets"
}
