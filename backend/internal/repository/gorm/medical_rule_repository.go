package gormrepo

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type MedicalRuleRepository struct {
	db *gorm.DB
}

func NewMedicalRuleRepository(db *gorm.DB) *MedicalRuleRepository {
	return &MedicalRuleRepository{db: db}
}

func (r *MedicalRuleRepository) ListActive(ctx context.Context) ([]models.MedicalRule, error) {
	var rules []models.MedicalRule
	if err := r.db.WithContext(ctx).Where("active = ?", true).Order("condition_key ASC, code ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}
