package gormrepo

import (
	"context"
	"errors"
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
	if err := r.db.WithContext(ctx).
		Where("refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > now() AND idle_expires_at > now()", refreshHash).
		First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) Rotate(ctx context.Context, sessionID, oldRefreshHash, newRefreshHash, csrfBindingID string, expiresAt, idleExpiresAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.Session{}).
		Where("id = ? AND refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > now() AND idle_expires_at > now()", sessionID, oldRefreshHash).
		Updates(map[string]any{
			"refresh_token_hash": newRefreshHash,
			"csrf_binding_id":    csrfBindingID,
			"expires_at":         expiresAt,
			"idle_expires_at":    idleExpiresAt,
			"last_seen_at":       time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *SessionRepository) Touch(ctx context.Context, sessionID string, idleExpiresAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Session{}).Where("id = ? AND revoked_at IS NULL", sessionID).Updates(map[string]any{
		"idle_expires_at": idleExpiresAt,
		"last_seen_at":    time.Now(),
	}).Error
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Session{}).Where("id = ?", sessionID).Update("revoked_at", &now).Error
}

func (r *SessionRepository) RevokeForUser(ctx context.Context, userID, sessionID string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&models.Session{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).
		Update("revoked_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *SessionRepository) RevokeOthers(ctx context.Context, userID, keepSessionID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Session{}).
		Where("user_id = ? AND id <> ? AND revoked_at IS NULL", userID, keepSessionID).
		Update("revoked_at", &now).Error
}

func (r *SessionRepository) ListByUser(ctx context.Context, userID string, limit int) ([]models.Session, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var sessions []models.Session
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_seen_at DESC, created_at DESC").
		Limit(limit).
		Find(&sessions).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []models.Session{}, nil
	}
	return sessions, err
}

func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*models.Session, error) {
	var session models.Session
	if err := r.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}
