package gormrepo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VectorRepository struct {
	db *gorm.DB
}

func NewVectorRepository(db *gorm.DB) *VectorRepository {
	return &VectorRepository{db: db}
}

func (r *VectorRepository) UpsertProfileEmbedding(ctx context.Context, embedding *models.ProfileEmbedding) error {
	embedding.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "profile_id"}, {Name: "embedding_version"}},
		DoUpdates: clause.Assignments(map[string]any{
			"user_id":     embedding.UserID,
			"source_hash": embedding.SourceHash,
			"embedding":   gorm.Expr("?::vector", embedding.Embedding),
			"metadata":    embedding.Metadata,
			"updated_at":  embedding.UpdatedAt,
		}),
	}).Create(embedding).Error
}

func (r *VectorRepository) SearchSimilarProfileBundles(ctx context.Context, userID, profileID, embeddingVersion, vectorLiteral string, limit int) ([]repository.ProfileBundle, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	type vectorRow struct {
		UserID            string
		Age               int
		ActivityLevel     string
		Goal              string
		MaxReadyTime      int
		HasChronicDisease bool
		LikesJSON         string
		StylesJSON        string
		MealTypesJSON     string
		CuisinesJSON      string
		ConditionsJSON    string
		ChronicJSON       string
	}

	rows := []vectorRow{}
	err := r.db.WithContext(ctx).Raw(`
WITH nearest AS (
    SELECT pe.user_id, pe.profile_id, pe.embedding <=> ?::vector AS distance
    FROM recommendation.profile_embeddings pe
    WHERE pe.embedding_version = ?
      AND pe.profile_id <> ?
      AND pe.user_id <> ?
    ORDER BY distance ASC
    LIMIT ?
)
SELECT
    p.user_id,
    p.age,
    l.activity_level,
    l.goal,
    l.max_ready_time,
    c.has_chronic_disease,
    COALESCE((SELECT json_agg(x.ingredient_key) FROM health.profile_preference_ingredients x WHERE x.user_id = p.user_id AND x.kind = 'like'), '[]')::text AS likes_json,
    COALESCE((SELECT json_agg(x.meal_style_key) FROM health.profile_meal_styles x WHERE x.user_id = p.user_id), '[]')::text AS styles_json,
    COALESCE((SELECT json_agg(x.meal_type_key) FROM health.profile_meal_types x WHERE x.user_id = p.user_id), '[]')::text AS meal_types_json,
    COALESCE((SELECT json_agg(x.cuisine_key) FROM health.profile_cuisines x WHERE x.user_id = p.user_id AND x.kind = 'preferred'), '[]')::text AS cuisines_json,
    COALESCE((SELECT json_agg(x.condition_key) FROM health.profile_conditions x WHERE x.user_id = p.user_id), '[]')::text AS conditions_json,
    COALESCE((SELECT json_agg(x.condition_key) FROM health.profile_chronic_conditions x WHERE x.user_id = p.user_id), '[]')::text AS chronic_json
FROM nearest n
JOIN health.profiles p ON p.id = n.profile_id
JOIN health.lifestyles l ON l.user_id = p.user_id
JOIN health.constraints c ON c.user_id = p.user_id
ORDER BY n.distance ASC
`, vectorLiteral, embeddingVersion, profileID, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]repository.ProfileBundle, 0, len(rows))
	for _, row := range rows {
		out = append(out, repository.ProfileBundle{
			UserID:            row.UserID,
			Age:               row.Age,
			ActivityLevel:     row.ActivityLevel,
			Goal:              row.Goal,
			MaxReadyTime:      row.MaxReadyTime,
			MealStyles:        decodeStringList(row.StylesJSON),
			MealTypes:         decodeStringList(row.MealTypesJSON),
			PreferredCuisines: decodeStringList(row.CuisinesJSON),
			Likes:             decodeStringList(row.LikesJSON),
			Conditions:        decodeStringList(row.ConditionsJSON),
			ChronicDiseases:   decodeStringList(row.ChronicJSON),
			HasChronicDisease: row.HasChronicDisease,
		})
	}
	return out, nil
}

func decodeStringList(raw string) []string {
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}
