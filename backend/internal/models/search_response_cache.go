package models

import "time"

type SearchResponseCache struct {
	CacheKey  string    `gorm:"primaryKey;column:cache_key"`
	Source    string    `gorm:"not null;default:'spoonacular'"`
	Payload   JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	FetchedAt time.Time `gorm:"not null;default:now()"`
	ExpiresAt time.Time `gorm:"not null"`
}

func (SearchResponseCache) TableName() string {
	return "recommendation.search_response_cache"
}
