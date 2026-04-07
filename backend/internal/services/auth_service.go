package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

// AuthService handles users and sessions.
type AuthService struct {
	Users    repository.UserRepository
	Sessions repository.SessionRepository
	Tokens   *security.TokenManager
	PasswordParams security.Argon2Params
}

func (s *AuthService) Register(ctx context.Context, user *models.User, rawPassword string, userAgent, ip string) (string, time.Time, string, time.Time, error) {
	hash, err := security.HashPassword(rawPassword, s.PasswordParams)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	user.PasswordHash = hash
	if err := s.Users.Create(ctx, user); err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return s.createSession(ctx, user.ID, userAgent, ip)
}

func (s *AuthService) Login(ctx context.Context, user *models.User, rawPassword string, userAgent, ip string) (string, time.Time, string, time.Time, error) {
	valid, err := security.VerifyPassword(rawPassword, user.PasswordHash)
	if err != nil || !valid {
		return "", time.Time{}, "", time.Time{}, ErrInvalidCredentials
	}

	return s.createSession(ctx, user.ID, userAgent, ip)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, time.Time, string, time.Time, error) {
	refreshHash := s.Tokens.HashRefreshToken(refreshToken)
	session, err := s.Sessions.GetByRefreshHash(ctx, refreshHash)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, ErrInvalidCredentials
	}
	if session.ExpiresAt.Before(time.Now()) || session.RevokedAt != nil {
		return "", time.Time{}, "", time.Time{}, ErrInvalidCredentials
	}

	access, accessExp, err := s.Tokens.NewAccessToken(session.UserID, session.ID)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	newRefresh, refreshExp, err := s.Tokens.NewRefreshToken()
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	newHash := s.Tokens.HashRefreshToken(newRefresh)
	if err := s.Sessions.Rotate(ctx, session.ID, newHash, refreshExp); err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return access, accessExp, newRefresh, refreshExp, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	refreshHash := s.Tokens.HashRefreshToken(refreshToken)
	session, err := s.Sessions.GetByRefreshHash(ctx, refreshHash)
	if err != nil {
		return ErrInvalidCredentials
	}
	return s.Sessions.Revoke(ctx, session.ID)
}

func (s *AuthService) createSession(ctx context.Context, userID, userAgent, ip string) (string, time.Time, string, time.Time, error) {
	sessionID := uuid.NewString()
	access, accessExp, err := s.Tokens.NewAccessToken(userID, sessionID)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	refresh, refreshExp, err := s.Tokens.NewRefreshToken()
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	session := &models.Session{
		ID:               sessionID,
		UserID:           userID,
		RefreshTokenHash: s.Tokens.HashRefreshToken(refresh),
		ExpiresAt:        refreshExp,
		UserAgent:        userAgent,
		IP:               ip,
	}
	if err := s.Sessions.Create(ctx, session); err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return access, accessExp, refresh, refreshExp, nil
}
