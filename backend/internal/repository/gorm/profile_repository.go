package gormrepo

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) UpsertProfile(ctx context.Context, profile *models.Profile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"age", "sex", "weight", "height", "profession", "city", "updated_at"}),
	}).Create(profile).Error
}

func (r *ProfileRepository) UpsertLifestyle(ctx context.Context, lifestyle *models.Lifestyle) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"activity_level", "lifestyle_type", "goal", "updated_at"}),
	}).Create(lifestyle).Error
}

func (r *ProfileRepository) UpsertPreferences(ctx context.Context, preferences *models.Preferences) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"likes", "dislikes", "meal_styles", "meals_per_day", "updated_at"}),
	}).Create(preferences).Error
}

func (r *ProfileRepository) UpsertConstraints(ctx context.Context, constraints *models.Constraints) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"allergies", "conditions", "excluded_ingredients", "has_chronic_disease", "chronic_diseases", "takes_medication", "medications", "updated_at"}),
	}).Create(constraints).Error
}

func (r *ProfileRepository) GetProfile(ctx context.Context, userID string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, error) {
	var profile models.Profile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, nil, nil, nil, err
	}

	var lifestyle models.Lifestyle
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&lifestyle).Error; err != nil {
		return nil, nil, nil, nil, err
	}

	var preferences models.Preferences
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&preferences).Error; err != nil {
		return nil, nil, nil, nil, err
	}

	var constraints models.Constraints
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&constraints).Error; err != nil {
		return nil, nil, nil, nil, err
	}

	return &profile, &lifestyle, &preferences, &constraints, nil
}

