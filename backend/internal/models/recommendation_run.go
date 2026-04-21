package models

import "time"

type RecommendationRun struct {
	ID                  string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID              string    `gorm:"type:uuid;index;not null"`
	ProfileID           string    `gorm:"type:uuid;index;not null"`
	NutritionProfileID  string    `gorm:"type:uuid;index;not null"`
	Status              string    `gorm:"not null;default:'completed'"`
	QuerySignature      string    `gorm:"index;not null"`
	SourceSummary       JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	DecisionSummary     JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	ExternalTrace       JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	CorrelatedRequestID string    `gorm:"index;not null;default:''"`
	CreatedAt           time.Time `gorm:"not null;default:now()"`
}

func (RecommendationRun) TableName() string {
	return "recommendation.runs"
}
