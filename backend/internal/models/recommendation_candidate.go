package models

import "time"

type RecommendationCandidate struct {
	ID               string      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	RunID            string      `gorm:"type:uuid;index;not null"`
	UserID           string      `gorm:"type:uuid;index;not null"`
	ProfileID        string      `gorm:"type:uuid;index;not null"`
	ExternalRecipeID string      `gorm:"index;not null"`
	Title            string      `gorm:"not null"`
	Source           string      `gorm:"not null"`
	Stage            string      `gorm:"not null"`
	Accepted         bool        `gorm:"not null;default:false"`
	FinalRank        int         `gorm:"not null;default:0"`
	FinalScore       float64     `gorm:"not null;default:0"`
	Calories         float64     `gorm:"not null;default:0"`
	Protein          float64     `gorm:"not null;default:0"`
	Carbs            float64     `gorm:"not null;default:0"`
	Fat              float64     `gorm:"not null;default:0"`
	Sugar            float64     `gorm:"not null;default:0"`
	SodiumMg         float64     `gorm:"not null;default:0"`
	Ingredients      StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	Tags             StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	AcceptedReasons  StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	RejectedReasons  StringSlice `gorm:"type:jsonb;not null;default:'[]'"`
	ScoreBreakdown   JSONMap     `gorm:"type:jsonb;not null;default:'{}'"`
	FilterDecisions  JSONMap     `gorm:"type:jsonb;not null;default:'{}'"`
	SourceProvenance JSONMap     `gorm:"type:jsonb;not null;default:'{}'"`
	Explanation      string      `gorm:"not null;default:''"`
	Description      string      `gorm:"not null;default:''"`
	CreatedAt        time.Time   `gorm:"not null;default:now()"`
}

func (RecommendationCandidate) TableName() string {
	return "recommendation.candidates"
}
