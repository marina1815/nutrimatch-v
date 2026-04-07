package gormrepo

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *SessionRepository) GetByRefreshHash(ctx context.Context, refreshHash string) (*models.Session, error) {
	var session models.Session
	if err := r.db.WithContext(ctx).Where("refresh_token_hash = ? AND revoked_at IS NULL", refreshHash).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) Rotate(ctx context.Context, sessionID, newRefreshHash string, expiresAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Session{}).Where("id = ?", sessionID).Updates(map[string]any{
		"refresh_token_hash": newRefreshHash,
		"expires_at":         expiresAt,
	}).Error
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Session{}).Where("id = ?", sessionID).Update("revoked_at", &now).Error
}

func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*models.Session, error) {
	var session models.Session
	if err := r.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}
