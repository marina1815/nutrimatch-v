package gormrepo

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type RecommendationTraceRepository struct {
	db *gorm.DB
}

func NewRecommendationTraceRepository(db *gorm.DB) *RecommendationTraceRepository {
	return &RecommendationTraceRepository{db: db}
}

func (r *RecommendationTraceRepository) CreateRun(ctx context.Context, run *models.RecommendationRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *RecommendationTraceRepository) ReplaceCandidates(ctx context.Context, runID string, candidates []*models.RecommendationCandidate) error {
	tx := r.db.WithContext(ctx)
	if err := tx.Where("run_id = ?", runID).Delete(&models.RecommendationCandidate{}).Error; err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}
	return tx.Create(&candidates).Error
}

func (r *RecommendationTraceRepository) GetLatestRunByProfile(ctx context.Context, userID, profileID string) (*models.RecommendationRun, []*models.RecommendationCandidate, error) {
	var run models.RecommendationRun
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND profile_id = ?", userID, profileID).
		Order("created_at DESC").
		First(&run).Error; err != nil {
		return nil, nil, err
	}

	var candidates []*models.RecommendationCandidate
	if err := r.db.WithContext(ctx).
		Where("run_id = ?", run.ID).
		Order("final_rank ASC, created_at ASC").
		Find(&candidates).Error; err != nil {
		return nil, nil, err
	}

	return &run, candidates, nil
}

func (r *RecommendationTraceRepository) GetCandidateByRecipeID(ctx context.Context, userID, profileID, recipeID string) (*models.RecommendationCandidate, error) {
	var candidate models.RecommendationCandidate
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND profile_id = ? AND external_recipe_id = ?", userID, profileID, recipeID).
		Order("created_at DESC").
		First(&candidate).Error; err != nil {
		return nil, err
	}
	return &candidate, nil
}
