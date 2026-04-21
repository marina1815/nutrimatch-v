package gormrepo

import (
	"context"
	"errors"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RateLimitBucketRepository struct {
	db *gorm.DB
}

func NewRateLimitBucketRepository(db *gorm.DB) *RateLimitBucketRepository {
	return &RateLimitBucketRepository{db: db}
}

func (r *RateLimitBucketRepository) TakeToken(ctx context.Context, key, bucketType string, refillPerSecond float64, burst int, now time.Time) (bool, error) {
	if burst <= 0 || refillPerSecond <= 0 {
		return false, nil
	}

	var allowed bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		bucket, err := r.getOrCreateBucket(ctx, tx, key, bucketType, burst, now)
		if err != nil {
			return err
		}

		tokens := refillBucketTokens(bucket.Tokens, bucket.UpdatedAt, now, refillPerSecond, burst)
		if tokens < 1 {
			bucket.Tokens = tokens
			bucket.UpdatedAt = now
			allowed = false
			return tx.Model(bucket).Updates(map[string]any{
				"tokens":     bucket.Tokens,
				"updated_at": bucket.UpdatedAt,
			}).Error
		}

		bucket.Tokens = tokens - 1
		bucket.UpdatedAt = now
		allowed = true
		return tx.Model(bucket).Updates(map[string]any{
			"tokens":     bucket.Tokens,
			"updated_at": bucket.UpdatedAt,
		}).Error
	})
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func (r *RateLimitBucketRepository) getOrCreateBucket(ctx context.Context, tx *gorm.DB, key, bucketType string, burst int, now time.Time) (*models.RateLimitBucket, error) {
	var bucket models.RateLimitBucket
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("key = ?", key).
		First(&bucket).Error
	if err == nil {
		return &bucket, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	bucket = models.RateLimitBucket{
		Key:        key,
		BucketType: bucketType,
		Tokens:     float64(burst),
		UpdatedAt:  now,
	}
	if err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&bucket).Error; err != nil {
		return nil, err
	}

	err = tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("key = ?", key).
		First(&bucket).Error
	if err != nil {
		return nil, err
	}
	return &bucket, nil
}

func refillBucketTokens(current float64, updatedAt, now time.Time, refillPerSecond float64, burst int) float64 {
	if burst <= 0 {
		return 0
	}
	if current < 0 {
		current = 0
	}
	maxTokens := float64(burst)
	if now.After(updatedAt) {
		current += now.Sub(updatedAt).Seconds() * refillPerSecond
	}
	if current > maxTokens {
		current = maxTokens
	}
	return current
}
