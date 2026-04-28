package gormrepo

import (
	"context"
	"errors"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

const auditChainAdvisoryLockID int64 = 71881294736001

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

func (r *AuditRepository) AppendChained(ctx context.Context, event *models.AuditEvent, hash func(previousHash string, occurredAt time.Time) string) error {
	if r.db.Dialector.Name() != "postgres" {
		previousHash, err := r.LatestHash(ctx)
		if err != nil {
			return err
		}
		event.PreviousHash = previousHash
		if event.OccurredAt.IsZero() {
			event.OccurredAt = time.Now().UTC()
		}
		event.EventHash = hash(previousHash, event.OccurredAt.UTC())
		return r.Create(ctx, event)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", auditChainAdvisoryLockID).Error; err != nil {
			return err
		}

		var latest models.AuditEvent
		err := tx.
			Select("event_hash").
			Order("created_at DESC, id DESC").
			First(&latest).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			event.PreviousHash = ""
		} else if err != nil {
			return err
		} else {
			event.PreviousHash = latest.EventHash
		}

		if event.OccurredAt.IsZero() {
			event.OccurredAt = time.Now().UTC()
		}
		event.EventHash = hash(event.PreviousHash, event.OccurredAt.UTC())
		return tx.Create(event).Error
	})
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
