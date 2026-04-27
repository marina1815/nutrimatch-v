package models

import "time"

type ProfileEmbedding struct {
	ID               string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID           string    `gorm:"type:uuid;not null"`
	ProfileID        string    `gorm:"type:uuid;not null"`
	EmbeddingVersion string    `gorm:"not null"`
	SourceHash       string    `gorm:"not null"`
	Embedding        string    `gorm:"type:vector(768)"`
	Metadata         JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt        time.Time `gorm:"not null;default:now()"`
	UpdatedAt        time.Time `gorm:"not null;default:now()"`
}

func (ProfileEmbedding) TableName() string {
	return "recommendation.profile_embeddings"
}

type RecipeEmbedding struct {
	ID               string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ExternalRecipeID string    `gorm:"not null"`
	Source           string    `gorm:"not null;default:'spoonacular'"`
	EmbeddingVersion string    `gorm:"not null"`
	SourceHash       string    `gorm:"not null"`
	Embedding        string    `gorm:"type:vector(768)"`
	Metadata         JSONMap   `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt        time.Time `gorm:"not null;default:now()"`
	UpdatedAt        time.Time `gorm:"not null;default:now()"`
}

func (RecipeEmbedding) TableName() string {
	return "recommendation.recipe_embeddings"
}
