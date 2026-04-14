package gormrepo

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
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

func (r *ProfileRepository) UpsertNutritionProfile(ctx context.Context, profile *models.NutritionProfile) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"profile_id",
			"bmi",
			"bmi_category",
			"bmr",
			"estimated_calories",
			"target_calories",
			"target_protein_grams",
			"target_carbs_grams",
			"target_fat_grams",
			"max_meal_calories",
			"min_protein_per_meal",
			"max_carbs_per_meal",
			"max_fat_per_meal",
			"max_sugar_per_meal",
			"max_sodium_mg_per_meal",
			"derived_restrictions",
			"derived_excluded",
			"recommended_meal_styles",
			"metadata",
			"calculated_at",
			"updated_at",
		}),
	}).Create(profile).Error
}

func (r *ProfileRepository) GetNutritionProfile(ctx context.Context, userID string) (*models.NutritionProfile, error) {
	var profile models.NutritionProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *ProfileRepository) ListProfileBundles(ctx context.Context, excludeUserID string, limit int) ([]repository.ProfileBundle, error) {
	type profileJoin struct {
		UserID            string
		Age               int
		ActivityLevel     string
		Goal              string
		MealStyles        models.StringSlice
		Likes             models.StringSlice
		Conditions        models.StringSlice
		ChronicDiseases   models.StringSlice
		HasChronicDisease bool
	}

	query := r.db.WithContext(ctx).
		Table(models.Profile{}.TableName() + " AS p").
		Select("p.user_id, p.age, l.activity_level, l.goal, pr.meal_styles, pr.likes, c.conditions, c.chronic_diseases, c.has_chronic_disease").
		Joins("JOIN " + models.Lifestyle{}.TableName() + " AS l ON l.user_id = p.user_id").
		Joins("JOIN " + models.Preferences{}.TableName() + " AS pr ON pr.user_id = p.user_id").
		Joins("JOIN " + models.Constraints{}.TableName() + " AS c ON c.user_id = p.user_id")

	if excludeUserID != "" {
		query = query.Where("p.user_id <> ?", excludeUserID)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var rows []profileJoin
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]repository.ProfileBundle, 0, len(rows))
	for _, row := range rows {
		out = append(out, repository.ProfileBundle{
			UserID:            row.UserID,
			Age:               row.Age,
			ActivityLevel:     row.ActivityLevel,
			Goal:              row.Goal,
			MealStyles:        []string(row.MealStyles),
			Likes:             []string(row.Likes),
			Conditions:        []string(row.Conditions),
			ChronicDiseases:   []string(row.ChronicDiseases),
			HasChronicDisease: row.HasChronicDisease,
		})
	}

	return out, nil
}
