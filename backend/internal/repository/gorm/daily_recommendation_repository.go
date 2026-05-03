package gormrepo

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type DailyRecommendationRepository struct {
	db *gorm.DB
}

func NewDailyRecommendationRepository(db *gorm.DB) *DailyRecommendationRepository {
	return &DailyRecommendationRepository{db: db}
}

func (r *DailyRecommendationRepository) GetActiveSet(ctx context.Context, userID, profileID string, now time.Time) (*models.DailyRecommendationSet, []*models.DailyRecommendationMeal, error) {
	var set models.DailyRecommendationSet
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND profile_id = ? AND valid_from <= ? AND valid_until > ?", userID, profileID, now, now).
		Order("valid_from DESC").
		First(&set).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var meals []*models.DailyRecommendationMeal
	if err := r.db.WithContext(ctx).
		Where("set_id = ?", set.ID).
		Order("final_rank ASC, created_at ASC").
		Find(&meals).Error; err != nil {
		return nil, nil, err
	}
	return &set, meals, nil
}

func (r *DailyRecommendationRepository) GetPreviousShownRecipeIDs(ctx context.Context, userID, profileID string, now time.Time) ([]string, error) {
	var set models.DailyRecommendationSet
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND profile_id = ? AND valid_until <= ?", userID, profileID, now).
		Order("valid_until DESC").
		First(&set).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []string{}, nil
		}
		return nil, err
	}

	var rows []struct {
		RecipeID string `gorm:"column:recipe_id"`
	}
	if err := r.db.WithContext(ctx).
		Model(&models.DailyRecommendationMeal{}).
		Select("recipe_id").
		Where("set_id = ?", set.ID).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.RecipeID != "" {
			out = append(out, row.RecipeID)
		}
	}
	return out, nil
}

func (r *DailyRecommendationRepository) GetSuppressedRecipeIDs(ctx context.Context, userID, profileID string, now time.Time) ([]string, error) {
	var rows []struct {
		RecipeID string `gorm:"column:recipe_id"`
	}
	if err := r.db.WithContext(ctx).
		Model(&models.RecipeChoice{}).
		Select("recipe_id").
		Where("user_id = ? AND profile_id = ? AND expires_at > ?", userID, profileID, now).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.RecipeID == "" {
			continue
		}
		if _, ok := seen[row.RecipeID]; ok {
			continue
		}
		seen[row.RecipeID] = struct{}{}
		out = append(out, row.RecipeID)
	}
	return out, nil
}

func (r *DailyRecommendationRepository) CreateSet(ctx context.Context, set *models.DailyRecommendationSet, meals []*models.DailyRecommendationMeal) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(set).Error; err != nil {
			return err
		}
		for _, meal := range meals {
			meal.SetID = set.ID
			if err := tx.Create(meal).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DailyRecommendationRepository) ReplaceSetMeals(ctx context.Context, setID string, meals []*models.DailyRecommendationMeal, decisionSummary models.JSONMap, selectionMode, status string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("set_id = ?", setID).Delete(&models.DailyRecommendationMeal{}).Error; err != nil {
			return err
		}
		for _, meal := range meals {
			if meal == nil {
				continue
			}
			meal.SetID = setID
			if err := tx.Create(meal).Error; err != nil {
				return err
			}
		}
		updates := map[string]any{
			"decision_summary": decisionSummary,
		}
		if selectionMode != "" {
			updates["selection_mode"] = selectionMode
		}
		if status != "" {
			updates["status"] = status
		}
		return tx.Model(&models.DailyRecommendationSet{}).
			Where("id = ?", setID).
			Updates(updates).Error
	})
}

func (r *DailyRecommendationRepository) UpdateSetExplanations(ctx context.Context, setID string, explanations map[string]string, decisionSummary models.JSONMap, selectionMode string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.DailyRecommendationMeal{}).
			Where("set_id = ?", setID).
			Update("ai_explanation", "").Error; err != nil {
			return err
		}
		for recipeID, explanation := range explanations {
			if recipeID == "" || explanation == "" {
				continue
			}
			if err := tx.Model(&models.DailyRecommendationMeal{}).
				Where("set_id = ? AND recipe_id = ?", setID, recipeID).
				Update("ai_explanation", explanation).Error; err != nil {
				return err
			}
		}
		updates := map[string]any{
			"decision_summary": decisionSummary,
		}
		if selectionMode != "" {
			updates["selection_mode"] = selectionMode
		}
		return tx.Model(&models.DailyRecommendationSet{}).
			Where("id = ?", setID).
			Updates(updates).Error
	})
}

func (r *DailyRecommendationRepository) GetMealInActiveSet(ctx context.Context, userID, profileID, recipeID string, now time.Time) (*models.DailyRecommendationSet, *models.DailyRecommendationMeal, error) {
	set, _, err := r.GetActiveSet(ctx, userID, profileID, now)
	if err != nil || set == nil {
		return set, nil, err
	}

	var meal models.DailyRecommendationMeal
	err = r.db.WithContext(ctx).
		Where("set_id = ? AND recipe_id = ?", set.ID, recipeID).
		First(&meal).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return set, nil, nil
		}
		return nil, nil, err
	}
	return set, &meal, nil
}

func (r *DailyRecommendationRepository) GetChoiceForSet(ctx context.Context, setID, userID, profileID string) (*models.RecipeChoice, error) {
	var choice models.RecipeChoice
	err := r.db.WithContext(ctx).
		Where("set_id = ? AND user_id = ? AND profile_id = ?", setID, userID, profileID).
		Order("chosen_at DESC").
		First(&choice).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &choice, nil
}

func (r *DailyRecommendationRepository) CreateChoice(ctx context.Context, choice *models.RecipeChoice) error {
	return r.db.WithContext(ctx).Create(choice).Error
}
