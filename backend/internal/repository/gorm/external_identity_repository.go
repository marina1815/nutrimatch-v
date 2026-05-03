package gormrepo

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"gorm.io/gorm"
)

type ExternalIdentityRepository struct {
	db *gorm.DB
}

func NewExternalIdentityRepository(db *gorm.DB) *ExternalIdentityRepository {
	return &ExternalIdentityRepository{db: db}
}

func (r *ExternalIdentityRepository) GetByProviderSubject(ctx context.Context, provider, issuer, subject string) (*models.ExternalIdentity, error) {
	var identity models.ExternalIdentity
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND issuer = ? AND subject = ?", provider, issuer, subject).
		First(&identity).Error; err != nil {
		return nil, err
	}
	return &identity, nil
}

func (r *ExternalIdentityRepository) Create(ctx context.Context, identity *models.ExternalIdentity) error {
	return r.db.WithContext(ctx).Create(identity).Error
}

func (r *ExternalIdentityRepository) UpdateLogin(ctx context.Context, id string, email string, emailVerified bool, loginAt time.Time) error {
	return r.db.WithContext(ctx).Model(&models.ExternalIdentity{}).Where("id = ?", id).Updates(map[string]any{
		"email":          email,
		"email_verified": emailVerified,
		"last_login_at":  loginAt,
		"updated_at":     loginAt,
	}).Error
}
