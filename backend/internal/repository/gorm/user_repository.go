package gormrepo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if isUniqueViolation(err) {
		return repository.ErrDuplicate
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdateFullName(ctx context.Context, userID, fullName string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("full_name", fullName).Error
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"password_hash": passwordHash,
		"updated_at":    time.Now(),
	}).Error
}

func (r *UserRepository) UpdatePreferredMFAMethod(ctx context.Context, userID, method string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
		"preferred_mfa_method": method,
		"updated_at":           time.Now(),
	}).Error
}
