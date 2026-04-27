package gormrepo

import (
	"context"
	"errors"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func (r *AuditRepository) ListSince(ctx context.Context, since time.Time, limit int) ([]models.AuditEvent, error) {
	if limit <= 0 || limit > 10000 {
		limit = 1000
	}
	events := []models.AuditEvent{}
	err := r.db.WithContext(ctx).
		Where("occurred_at >= ?", since).
		Order("occurred_at ASC, created_at ASC, id ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, event *models.AuditEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *AuditRepository) LatestHash(ctx context.Context) (string, error) {
	var event models.AuditEvent
	err := r.db.WithContext(ctx).
		Select("event_hash").
		Order("created_at DESC, id DESC").
		First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return event.EventHash, nil
}
