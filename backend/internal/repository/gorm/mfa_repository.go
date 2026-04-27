package gormrepo

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MFARepository struct {
	db *gorm.DB
}

func NewMFARepository(db *gorm.DB) *MFARepository {
	return &MFARepository{db: db}
}

func (r *MFARepository) GetTOTPSecret(ctx context.Context, userID string) (*models.TOTPSecret, error) {
	var secret models.TOTPSecret
	if err := r.db.WithContext(ctx).First(&secret, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &secret, nil
}

func (r *MFARepository) UpsertTOTPSecret(ctx context.Context, secret *models.TOTPSecret) error {
	secret.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"secret_ciphertext": secret.SecretCiphertext,
			"enabled":           secret.Enabled,
			"confirmed_at":      secret.ConfirmedAt,
			"updated_at":        secret.UpdatedAt,
		}),
	}).Create(secret).Error
}

func (r *MFARepository) EnableTOTP(ctx context.Context, userID string, confirmedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.TOTPSecret{}).Where("user_id = ?", userID).Updates(map[string]any{
		"enabled":      true,
		"confirmed_at": confirmedAt,
		"updated_at":   time.Now(),
	}).Error
}

func (r *MFARepository) DisableTOTP(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&models.TOTPSecret{}).Where("user_id = ?", userID).Updates(map[string]any{
		"enabled":    false,
		"updated_at": time.Now(),
	}).Error
}

func (r *MFARepository) CreateWebAuthnCredential(ctx context.Context, credential *models.WebAuthnCredential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *MFARepository) ListWebAuthnCredentials(ctx context.Context, userID string) ([]models.WebAuthnCredential, error) {
	var credentials []models.WebAuthnCredential
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&credentials).Error
	return credentials, err
}

func (r *MFARepository) UpdateWebAuthnCredentialUsed(ctx context.Context, credentialID string, usedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.WebAuthnCredential{}).
		Where("credential_id = ?", credentialID).
		Update("last_used_at", usedAt).Error
}

func (r *MFARepository) CreateWebAuthnChallenge(ctx context.Context, challenge *models.WebAuthnChallenge) error {
	return r.db.WithContext(ctx).Create(challenge).Error
}

func (r *MFARepository) ConsumeWebAuthnChallenge(ctx context.Context, userID, challengeID, kind string, now time.Time) (*models.WebAuthnChallenge, error) {
	var challenge models.WebAuthnChallenge
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ? AND kind = ? AND consumed_at IS NULL AND expires_at > ?", challengeID, userID, kind, now).
			First(&challenge).Error; err != nil {
			return err
		}
		return tx.Model(&models.WebAuthnChallenge{}).Where("id = ?", challenge.ID).Update("consumed_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}
