package gormrepo

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, event *models.AuditEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}
