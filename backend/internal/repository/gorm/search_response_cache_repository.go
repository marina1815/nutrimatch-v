package gormrepo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/marina1815/nutrimatch/internal/clients/spoonacular"
	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SearchResponseCacheRepository struct {
	db *gorm.DB
}

func NewSearchResponseCacheRepository(db *gorm.DB) *SearchResponseCacheRepository {
	return &SearchResponseCacheRepository{db: db}
}

func (r *SearchResponseCacheRepository) Get(ctx context.Context, cacheKey string) (*spoonacular.SearchResponse, bool, error) {
	var entry models.SearchResponseCache
	err := r.db.WithContext(ctx).
		Where("cache_key = ? AND expires_at > now()", cacheKey).
		First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	payload, err := json.Marshal(entry.Payload)
	if err != nil {
		return nil, false, err
	}

	var response spoonacular.SearchResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, false, err
	}
	response.CacheHit = true
	return &response, true, nil
}

func (r *SearchResponseCacheRepository) Set(ctx context.Context, cacheKey string, response *spoonacular.SearchResponse, ttl time.Duration) error {
	if response == nil || ttl <= 0 {
		return nil
	}

	payload, err := marshalSearchResponse(response)
	if err != nil {
		return err
	}

	now := time.Now()
	entry := &models.SearchResponseCache{
		CacheKey:  cacheKey,
		Source:    "spoonacular",
		Payload:   payload,
		FetchedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "cache_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"source":     entry.Source,
			"payload":    entry.Payload,
			"fetched_at": entry.FetchedAt,
			"expires_at": entry.ExpiresAt,
		}),
	}).Create(entry).Error
}

func marshalSearchResponse(response *spoonacular.SearchResponse) (models.JSONMap, error) {
	copied := *response
	copied.CacheHit = false

	raw, err := json.Marshal(copied)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return models.JSONMap(payload), nil
}
