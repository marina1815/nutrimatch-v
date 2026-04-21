package gormrepo

import (
	"context"

	"github.com/marina1815/nutrimatch/internal/repository"
	"gorm.io/gorm"
)

type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) WithinTransaction(ctx context.Context, fn func(repository.Repositories) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repos := repository.Repositories{
			Users:              NewUserRepository(tx),
			Profiles:           NewProfileRepository(tx),
			Sessions:           NewSessionRepository(tx),
			MedicalRules:       NewMedicalRuleRepository(tx),
			RecommendationRuns: NewRecommendationTraceRepository(tx),
			Audit:              NewAuditRepository(tx),
			AuthFailures:       NewAuthFailureRepository(tx),
			RateLimitBuckets:   NewRateLimitBucketRepository(tx),
			ExternalIdentities: NewExternalIdentityRepository(tx),
		}
		return fn(repos)
	})
}
