package repository

import (
	"context"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	UpdateFullName(ctx context.Context, userID, fullName string) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, sessionID string) (*models.Session, error)
	GetByRefreshHash(ctx context.Context, refreshHash string) (*models.Session, error)
	Rotate(ctx context.Context, sessionID, newRefreshHash string, expiresAt time.Time) error
	Revoke(ctx context.Context, sessionID string) error
}

type ProfileRepository interface {
	UpsertProfile(ctx context.Context, profile *models.Profile) error
	UpsertLifestyle(ctx context.Context, lifestyle *models.Lifestyle) error
	UpsertPreferences(ctx context.Context, preferences *models.Preferences) error
	UpsertConstraints(ctx context.Context, constraints *models.Constraints) error
	GetProfile(ctx context.Context, userID string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, error)
}
