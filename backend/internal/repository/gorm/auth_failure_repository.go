package gormrepo

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type AuthFailureRepository struct {
	db *gorm.DB
}

func NewAuthFailureRepository(db *gorm.DB) *AuthFailureRepository {
	return &AuthFailureRepository{db: db}
}

func (r *AuthFailureRepository) Create(ctx context.Context, failure *models.AuthFailure) error {
	return r.db.WithContext(ctx).Create(failure).Error
}

func (r *AuthFailureRepository) CountRecent(ctx context.Context, emailHash, ipHash string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AuthFailure{}).
		Where("occurred_at >= ?", since).
		Where("(email_hash <> '' AND email_hash = ?) OR (ip_hash <> '' AND ip_hash = ?)", emailHash, ipHash).
		Count(&count).Error
	return count, err
}
