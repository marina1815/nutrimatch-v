package gormrepo

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/repository"
	"gorm.io/gorm"
)

type RetentionRepository struct {
	db *gorm.DB
}

func NewRetentionRepository(db *gorm.DB) *RetentionRepository {
	return &RetentionRepository{db: db}
}

func (r *RetentionRepository) ApplyRetention(ctx context.Context, policy repository.RetentionPolicy, now time.Time) error {
	sessionCutoff := now.AddDate(0, 0, -policy.SessionRetentionDays)
	authFailureCutoff := now.AddDate(0, 0, -policy.AuthFailureRetentionDays)
	rateLimitCutoff := now.Add(-time.Duration(policy.RateLimitBucketRetentionHours) * time.Hour)
	traceCutoff := now.AddDate(0, 0, -policy.RecommendationTraceRetentionDays)

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			DELETE FROM identity.mfa_login_challenges
			WHERE consumed_at IS NOT NULL OR expires_at < ?
		`, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			DELETE FROM identity.webauthn_challenges
			WHERE consumed_at IS NOT NULL OR expires_at < ?
		`, now).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			DELETE FROM identity.sessions
			WHERE (revoked_at IS NOT NULL OR expires_at < ?) AND created_at < ?
		`, now, sessionCutoff).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			DELETE FROM security.auth_failures
			WHERE occurred_at < ?
		`, authFailureCutoff).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			DELETE FROM security.rate_limit_buckets
			WHERE updated_at < ?
		`, rateLimitCutoff).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			DELETE FROM recommendation.runs
			WHERE created_at < ?
		`, traceCutoff).Error; err != nil {
			return err
		}
		return nil
	})
}
