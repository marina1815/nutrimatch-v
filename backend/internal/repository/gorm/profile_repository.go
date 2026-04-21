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
		DoUpdates: clause.AssignmentColumns([]string{"activity_level", "lifestyle_type", "goal", "max_ready_time", "updated_at"}),
	}).Create(lifestyle).Error
}

func (r *ProfileRepository) UpsertPreferences(ctx context.Context, preferences *models.Preferences) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"meals_per_day", "updated_at"}),
	}).Create(preferences).Error; err != nil {
		return err
	}

	if err := r.ensureIngredients(ctx, append(append([]string{}, preferences.Likes...), preferences.Dislikes...)); err != nil {
		return err
	}
	if err := r.ensureMealStyles(ctx, []string(preferences.MealStyles)); err != nil {
		return err
	}
	if err := r.ensureMealTypes(ctx, []string(preferences.MealTypes)); err != nil {
		return err
	}
	if err := r.ensureCuisines(ctx, append(append([]string{}, preferences.PreferredCuisines...), preferences.ExcludedCuisines...)); err != nil {
		return err
	}

	if err := r.replacePreferenceIngredients(ctx, preferences.UserID, "like", []string(preferences.Likes)); err != nil {
		return err
	}
	if err := r.replacePreferenceIngredients(ctx, preferences.UserID, "dislike", []string(preferences.Dislikes)); err != nil {
		return err
	}
	if err := r.replaceMealStyles(ctx, preferences.UserID, []string(preferences.MealStyles)); err != nil {
		return err
	}
	if err := r.replaceMealTypes(ctx, preferences.UserID, []string(preferences.MealTypes)); err != nil {
		return err
	}
	if err := r.replaceCuisines(ctx, preferences.UserID, "preferred", []string(preferences.PreferredCuisines)); err != nil {
		return err
	}
	return r.replaceCuisines(ctx, preferences.UserID, "excluded", []string(preferences.ExcludedCuisines))
}

func (r *ProfileRepository) UpsertConstraints(ctx context.Context, constraints *models.Constraints) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"has_chronic_disease", "takes_medication", "medications", "updated_at"}),
	}).Create(constraints).Error; err != nil {
		return err
	}

	if err := r.ensureIngredients(ctx, []string(constraints.ExcludedIngredients)); err != nil {
		return err
	}
	if err := r.ensureIntolerances(ctx, []string(constraints.Allergies)); err != nil {
		return err
	}
	allConditions := append(append([]string{}, constraints.Conditions...), constraints.ChronicDiseases...)
	if err := r.ensureConditions(ctx, allConditions); err != nil {
		return err
	}

	if err := r.replacePreferenceIngredients(ctx, constraints.UserID, "exclude", []string(constraints.ExcludedIngredients)); err != nil {
		return err
	}
	if err := r.replaceIntolerances(ctx, constraints.UserID, []string(constraints.Allergies)); err != nil {
		return err
	}
	if err := r.replaceConditions(ctx, constraints.UserID, models.ProfileCondition{}.TableName(), []string(constraints.Conditions)); err != nil {
		return err
	}
	return r.replaceConditions(ctx, constraints.UserID, models.ProfileChronicCondition{}.TableName(), []string(constraints.ChronicDiseases))
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

	likes, err := r.listPreferenceIngredients(ctx, userID, "like")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	dislikes, err := r.listPreferenceIngredients(ctx, userID, "dislike")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	excluded, err := r.listPreferenceIngredients(ctx, userID, "exclude")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	mealStyles, err := r.listValues(ctx, models.ProfileMealStyle{}.TableName(), "meal_style_key", userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	mealTypes, err := r.listValues(ctx, models.ProfileMealType{}.TableName(), "meal_type_key", userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	preferredCuisines, err := r.listValuesByKind(ctx, models.ProfileCuisine{}.TableName(), "cuisine_key", userID, "preferred")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	excludedCuisines, err := r.listValuesByKind(ctx, models.ProfileCuisine{}.TableName(), "cuisine_key", userID, "excluded")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	intolerances, err := r.listValues(ctx, models.ProfileIntolerance{}.TableName(), "intolerance_key", userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	conditions, err := r.listValues(ctx, models.ProfileCondition{}.TableName(), "condition_key", userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	chronicConditions, err := r.listValues(ctx, models.ProfileChronicCondition{}.TableName(), "condition_key", userID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	preferences.Likes = models.StringSlice(likes)
	preferences.Dislikes = models.StringSlice(dislikes)
	preferences.MealStyles = models.StringSlice(mealStyles)
	preferences.MealTypes = models.StringSlice(mealTypes)
	preferences.PreferredCuisines = models.StringSlice(preferredCuisines)
	preferences.ExcludedCuisines = models.StringSlice(excludedCuisines)

	constraints.Allergies = models.StringSlice(intolerances)
	constraints.Conditions = models.StringSlice(conditions)
	constraints.ExcludedIngredients = models.StringSlice(excluded)
	constraints.ChronicDiseases = models.StringSlice(chronicConditions)

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
	type profileSeed struct {
		UserID            string
		Age               int
		ActivityLevel     string
		Goal              string
		MaxReadyTime      int
		HasChronicDisease bool
	}

	query := r.db.WithContext(ctx).
		Table(models.Profile{}.TableName() + " AS p").
		Select("p.user_id, p.age, l.activity_level, l.goal, l.max_ready_time, c.has_chronic_disease").
		Joins("JOIN " + models.Lifestyle{}.TableName() + " AS l ON l.user_id = p.user_id").
		Joins("JOIN " + models.Constraints{}.TableName() + " AS c ON c.user_id = p.user_id")

	if excludeUserID != "" {
		query = query.Where("p.user_id <> ?", excludeUserID)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var rows []profileSeed
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []repository.ProfileBundle{}, nil
	}

	userIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}

	likesByUser, err := r.loadGroupedValues(ctx, models.ProfilePreferenceIngredient{}.TableName(), "ingredient_key", userIDs, map[string]any{"kind": "like"})
	if err != nil {
		return nil, err
	}
	stylesByUser, err := r.loadGroupedValues(ctx, models.ProfileMealStyle{}.TableName(), "meal_style_key", userIDs, nil)
	if err != nil {
		return nil, err
	}
	mealTypesByUser, err := r.loadGroupedValues(ctx, models.ProfileMealType{}.TableName(), "meal_type_key", userIDs, nil)
	if err != nil {
		return nil, err
	}
	preferredCuisinesByUser, err := r.loadGroupedValues(ctx, models.ProfileCuisine{}.TableName(), "cuisine_key", userIDs, map[string]any{"kind": "preferred"})
	if err != nil {
		return nil, err
	}
	conditionsByUser, err := r.loadGroupedValues(ctx, models.ProfileCondition{}.TableName(), "condition_key", userIDs, nil)
	if err != nil {
		return nil, err
	}
	chronicByUser, err := r.loadGroupedValues(ctx, models.ProfileChronicCondition{}.TableName(), "condition_key", userIDs, nil)
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
			MealStyles:        stylesByUser[row.UserID],
			MealTypes:         mealTypesByUser[row.UserID],
			PreferredCuisines: preferredCuisinesByUser[row.UserID],
			Likes:             likesByUser[row.UserID],
			Conditions:        conditionsByUser[row.UserID],
			ChronicDiseases:   chronicByUser[row.UserID],
			HasChronicDisease: row.HasChronicDisease,
		})
	}

	return out, nil
}

func (r *ProfileRepository) ensureIngredients(ctx context.Context, values []string) error {
	if len(values) == 0 {
		return nil
	}

	records := make([]models.CatalogIngredient, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		records = append(records, models.CatalogIngredient{
			Key:         value,
			DisplayName: value,
			Source:      "user",
		})
	}
	if len(records) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&records).Error
}

func (r *ProfileRepository) ensureMealStyles(ctx context.Context, values []string) error {
	if len(values) == 0 {
		return nil
	}

	records := make([]models.CatalogMealStyle, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		records = append(records, models.CatalogMealStyle{
			Key:         value,
			DisplayName: value,
			Source:      "system",
		})
	}
	if len(records) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&records).Error
}

func (r *ProfileRepository) ensureMealTypes(ctx context.Context, values []string) error {
	if len(values) == 0 {
		return nil
	}

	records := make([]models.CatalogMealType, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		records = append(records, models.CatalogMealType{
			Key:         value,
			DisplayName: value,
			Source:      "system",
		})
	}
	if len(records) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&records).Error
}

func (r *ProfileRepository) ensureCuisines(ctx context.Context, values []string) error {
	if len(values) == 0 {
		return nil
	}

	records := make([]models.CatalogCuisine, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		records = append(records, models.CatalogCuisine{
			Key:         value,
			DisplayName: value,
			Source:      "system",
		})
	}
	if len(records) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&records).Error
}

func (r *ProfileRepository) ensureIntolerances(ctx context.Context, values []string) error {
	if len(values) == 0 {
		return nil
	}

	records := make([]models.CatalogIntolerance, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		records = append(records, models.CatalogIntolerance{
			Key:              value,
			DisplayName:      value,
			SpoonacularValue: value,
			Source:           "system",
		})
	}
	if len(records) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&records).Error
}

func (r *ProfileRepository) ensureConditions(ctx context.Context, values []string) error {
	if len(values) == 0 {
		return nil
	}

	records := make([]models.CatalogCondition, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		records = append(records, models.CatalogCondition{
			Key:         value,
			DisplayName: value,
			Source:      "system",
		})
	}
	if len(records) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&records).Error
}

func (r *ProfileRepository) replacePreferenceIngredients(ctx context.Context, userID, kind string, values []string) error {
	tx := r.db.WithContext(ctx)
	if err := tx.Where("user_id = ? AND kind = ?", userID, kind).Delete(&models.ProfilePreferenceIngredient{}).Error; err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}

	rows := make([]models.ProfilePreferenceIngredient, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		rows = append(rows, models.ProfilePreferenceIngredient{
			UserID:        userID,
			IngredientKey: value,
			Kind:          kind,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func (r *ProfileRepository) replaceMealStyles(ctx context.Context, userID string, values []string) error {
	tx := r.db.WithContext(ctx)
	if err := tx.Where("user_id = ?", userID).Delete(&models.ProfileMealStyle{}).Error; err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}

	rows := make([]models.ProfileMealStyle, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		rows = append(rows, models.ProfileMealStyle{
			UserID:       userID,
			MealStyleKey: value,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func (r *ProfileRepository) replaceMealTypes(ctx context.Context, userID string, values []string) error {
	tx := r.db.WithContext(ctx)
	if err := tx.Where("user_id = ?", userID).Delete(&models.ProfileMealType{}).Error; err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}

	rows := make([]models.ProfileMealType, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		rows = append(rows, models.ProfileMealType{
			UserID:      userID,
			MealTypeKey: value,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func (r *ProfileRepository) replaceCuisines(ctx context.Context, userID, kind string, values []string) error {
	tx := r.db.WithContext(ctx)
	if err := tx.Where("user_id = ? AND kind = ?", userID, kind).Delete(&models.ProfileCuisine{}).Error; err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}

	rows := make([]models.ProfileCuisine, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		rows = append(rows, models.ProfileCuisine{
			UserID:     userID,
			CuisineKey: value,
			Kind:       kind,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func (r *ProfileRepository) replaceIntolerances(ctx context.Context, userID string, values []string) error {
	tx := r.db.WithContext(ctx)
	if err := tx.Where("user_id = ?", userID).Delete(&models.ProfileIntolerance{}).Error; err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}

	rows := make([]models.ProfileIntolerance, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		rows = append(rows, models.ProfileIntolerance{
			UserID:         userID,
			IntoleranceKey: value,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}

func (r *ProfileRepository) replaceConditions(ctx context.Context, userID, tableName string, values []string) error {
	if len(values) == 0 {
		switch tableName {
		case models.ProfileCondition{}.TableName():
			return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.ProfileCondition{}).Error
		case models.ProfileChronicCondition{}.TableName():
			return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.ProfileChronicCondition{}).Error
		default:
			return nil
		}
	}

	switch tableName {
	case models.ProfileCondition{}.TableName():
		if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.ProfileCondition{}).Error; err != nil {
			return err
		}
		rows := make([]models.ProfileCondition, 0, len(values))
		for _, value := range values {
			if value == "" {
				continue
			}
			rows = append(rows, models.ProfileCondition{UserID: userID, ConditionKey: value})
		}
		if len(rows) == 0 {
			return nil
		}
		return r.db.WithContext(ctx).Create(&rows).Error
	case models.ProfileChronicCondition{}.TableName():
		if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.ProfileChronicCondition{}).Error; err != nil {
			return err
		}
		rows := make([]models.ProfileChronicCondition, 0, len(values))
		for _, value := range values {
			if value == "" {
				continue
			}
			rows = append(rows, models.ProfileChronicCondition{UserID: userID, ConditionKey: value})
		}
		if len(rows) == 0 {
			return nil
		}
		return r.db.WithContext(ctx).Create(&rows).Error
	default:
		return nil
	}
}

func (r *ProfileRepository) listPreferenceIngredients(ctx context.Context, userID, kind string) ([]string, error) {
	var values []string
	err := r.db.WithContext(ctx).
		Table(models.ProfilePreferenceIngredient{}.TableName()).
		Where("user_id = ? AND kind = ?", userID, kind).
		Order("created_at ASC, ingredient_key ASC").
		Pluck("ingredient_key", &values).Error
	return values, err
}

func (r *ProfileRepository) listValues(ctx context.Context, tableName, column, userID string) ([]string, error) {
	var values []string
	err := r.db.WithContext(ctx).
		Table(tableName).
		Where("user_id = ?", userID).
		Order("created_at ASC, "+column+" ASC").
		Pluck(column, &values).Error
	return values, err
}

func (r *ProfileRepository) listValuesByKind(ctx context.Context, tableName, column, userID, kind string) ([]string, error) {
	var values []string
	err := r.db.WithContext(ctx).
		Table(tableName).
		Where("user_id = ? AND kind = ?", userID, kind).
		Order("created_at ASC, "+column+" ASC").
		Pluck(column, &values).Error
	return values, err
}

func (r *ProfileRepository) loadGroupedValues(ctx context.Context, tableName, valueColumn string, userIDs []string, filters map[string]any) (map[string][]string, error) {
	out := make(map[string][]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}

	query := r.db.WithContext(ctx).
		Table(tableName).
		Select("user_id, "+valueColumn).
		Where("user_id IN ?", userIDs).
		Order("created_at ASC, " + valueColumn + " ASC")

	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID string
		var value string
		if err := rows.Scan(&userID, &value); err != nil {
			return nil, err
		}
		out[userID] = append(out[userID], value)
	}

	return out, rows.Err()
}
