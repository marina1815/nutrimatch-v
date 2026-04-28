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
var ErrAuthTemporarilyBlocked = errors.New("authentication temporarily blocked")
var ErrSessionNotFound = errors.New("session not found")
var ErrPasswordConfirmationMismatch = errors.New("password confirmation mismatch")

type SessionSummary struct {
	ID            string
	AuthMethod    string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	CreatedAt     time.Time
	LastSeenAt    time.Time
	RevokedAt     *time.Time
	Current       bool
}

// AuthService handles users and sessions.
type AuthService struct {
	Users          repository.UserRepository
	Sessions       repository.SessionRepository
	Failures       repository.AuthFailureRepository
	TxManager      repository.TxManager
	Tokens         *security.TokenManager
	SessionIdleTTL time.Duration
	FailureWindow  time.Duration
	MaxFailures    int
	PasswordParams security.Argon2Params
	dummyHash      string
}

func (s *AuthService) Register(ctx context.Context, user *models.User, rawPassword string, userAgent, ip string) (string, time.Time, string, time.Time, error) {
	hash, err := security.HashPassword(rawPassword, s.PasswordParams)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	user.PasswordHash = hash

	if s.TxManager == nil {
		if err := s.Users.Create(ctx, user); err != nil {
			return "", time.Time{}, "", time.Time{}, err
		}
		return s.createSession(ctx, s.Sessions, user.ID, "local", userAgent, ip)
	}

	var (
		access     string
		accessExp  time.Time
		refresh    string
		refreshExp time.Time
	)

	err = s.TxManager.WithinTransaction(ctx, func(repos repository.Repositories) error {
		if err := repos.Users.Create(ctx, user); err != nil {
			return err
		}

		var createErr error
		access, accessExp, refresh, refreshExp, createErr = s.createSession(ctx, repos.Sessions, user.ID, "local", userAgent, ip)
		return createErr
	})
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return access, accessExp, refresh, refreshExp, nil
}

func (s *AuthService) Login(ctx context.Context, email, rawPassword, userAgent, ip string) (string, time.Time, string, time.Time, error) {
	user, err := s.AuthenticatePrimary(ctx, email, rawPassword, ip)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return s.createSession(ctx, s.Sessions, user.ID, "local", userAgent, ip)
}

func (s *AuthService) AuthenticatePrimary(ctx context.Context, email, rawPassword, ip string) (*models.User, error) {
	emailHash := security.HashFingerprint(email)
	ipHash := security.HashFingerprint(ip)
	blocked, err := s.isTemporarilyBlocked(ctx, emailHash, ipHash)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrAuthTemporarilyBlocked
	}

	user, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		s.consumeDummyVerification(rawPassword)
		_ = s.recordFailure(ctx, emailHash, ipHash, "invalid_credentials")
		return nil, ErrInvalidCredentials
	}

	valid, verifyErr := security.VerifyPassword(rawPassword, user.PasswordHash)
	if verifyErr != nil || !valid {
		_ = s.recordFailure(ctx, emailHash, ipHash, "invalid_credentials")
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, time.Time, string, time.Time, error) {
	refreshHash := s.Tokens.HashRefreshToken(refreshToken)
	session, err := s.Sessions.GetByRefreshHash(ctx, refreshHash)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, ErrInvalidCredentials
	}
	now := time.Now()
	if session.ExpiresAt.Before(now) || session.IdleExpiresAt.Before(now) || session.RevokedAt != nil {
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
	newCSRFBindingID := uuid.NewString()
	if err := s.Sessions.Rotate(ctx, session.ID, refreshHash, newHash, newCSRFBindingID, refreshExp, s.idleExpiry(refreshExp)); err != nil {
		return "", time.Time{}, "", time.Time{}, ErrInvalidCredentials
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

func (s *AuthService) SessionFromRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	refreshHash := s.Tokens.HashRefreshToken(refreshToken)
	session, err := s.Sessions.GetByRefreshHash(ctx, refreshHash)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return session, nil
}

func (s *AuthService) IssueSession(ctx context.Context, userID, authMethod, userAgent, ip string) (string, time.Time, string, time.Time, error) {
	return s.createSession(ctx, s.Sessions, userID, authMethod, userAgent, ip)
}

func (s *AuthService) ListSessions(ctx context.Context, userID, currentSessionID string) ([]SessionSummary, error) {
	if s == nil || s.Sessions == nil {
		return nil, ErrSessionNotFound
	}
	sessions, err := s.Sessions.ListByUser(ctx, userID, 50)
	if err != nil {
		return nil, err
	}
	out := make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, SessionSummary{
			ID:            session.ID,
			AuthMethod:    session.AuthMethod,
			ExpiresAt:     session.ExpiresAt,
			IdleExpiresAt: session.IdleExpiresAt,
			CreatedAt:     session.CreatedAt,
			LastSeenAt:    session.LastSeenAt,
			RevokedAt:     session.RevokedAt,
			Current:       session.ID == currentSessionID,
		})
	}
	return out, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	if s == nil || s.Sessions == nil {
		return ErrSessionNotFound
	}
	if err := s.Sessions.RevokeForUser(ctx, userID, sessionID); err != nil {
		return ErrSessionNotFound
	}
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, currentSessionID, currentPassword, newPassword, confirmation string) error {
	if newPassword != confirmation {
		return ErrPasswordConfirmationMismatch
	}

	user, err := s.Users.GetByID(ctx, userID)
	if err != nil {
		return ErrInvalidCredentials
	}
	valid, verifyErr := security.VerifyPassword(currentPassword, user.PasswordHash)
	if verifyErr != nil || !valid {
		return ErrInvalidCredentials
	}
	hash, err := security.HashPassword(newPassword, s.PasswordParams)
	if err != nil {
		return err
	}

	if s.TxManager == nil {
		if err := s.Users.UpdatePasswordHash(ctx, userID, hash); err != nil {
			return err
		}
		return s.Sessions.RevokeOthers(ctx, userID, currentSessionID)
	}

	return s.TxManager.WithinTransaction(ctx, func(repos repository.Repositories) error {
		if err := repos.Users.UpdatePasswordHash(ctx, userID, hash); err != nil {
			return err
		}
		return repos.Sessions.RevokeOthers(ctx, userID, currentSessionID)
	})
}

func (s *AuthService) createSession(ctx context.Context, sessions repository.SessionRepository, userID, authMethod, userAgent, ip string) (string, time.Time, string, time.Time, error) {
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
		AuthMethod:       authMethod,
		RefreshTokenHash: s.Tokens.HashRefreshToken(refresh),
		CSRFBindingID:    uuid.NewString(),
		ExpiresAt:        refreshExp,
		IdleExpiresAt:    s.idleExpiry(refreshExp),
		LastSeenAt:       time.Now(),
		UserAgentHash:    security.HashFingerprint(userAgent),
		IPHash:           security.HashFingerprint(ip),
	}
	if err := sessions.Create(ctx, session); err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	return access, accessExp, refresh, refreshExp, nil
}

func (s *AuthService) idleExpiry(refreshExp time.Time) time.Time {
	idleTTL := s.SessionIdleTTL
	if idleTTL <= 0 {
		idleTTL = 24 * time.Hour
	}

	idleExp := time.Now().Add(idleTTL)
	if idleExp.After(refreshExp) {
		return refreshExp
	}
	return idleExp
}

func (s *AuthService) recordFailure(ctx context.Context, emailHash, ipHash, reason string) error {
	if s == nil || s.Failures == nil {
		return nil
	}
	return s.Failures.Create(ctx, &models.AuthFailure{
		EmailHash:  emailHash,
		IPHash:     ipHash,
		Reason:     reason,
		OccurredAt: time.Now(),
	})
}

func (s *AuthService) isTemporarilyBlocked(ctx context.Context, emailHash, ipHash string) (bool, error) {
	if s == nil || s.Failures == nil || s.MaxFailures <= 0 {
		return false, nil
	}
	window := s.FailureWindow
	if window <= 0 {
		window = 15 * time.Minute
	}
	count, err := s.Failures.CountRecent(ctx, emailHash, ipHash, time.Now().Add(-window))
	if err != nil {
		return false, err
	}
	return int(count) >= s.MaxFailures, nil
}

func (s *AuthService) consumeDummyVerification(password string) {
	hash := s.ensureDummyHash()
	if hash == "" {
		return
	}
	_, _ = security.VerifyPassword(password, hash)
}

func (s *AuthService) ensureDummyHash() string {
	if s == nil {
		return ""
	}
	if s.dummyHash != "" {
		return s.dummyHash
	}
	hash, err := security.HashPassword("NutriMatch_dummy_password_123!", s.PasswordParams)
	if err != nil {
		return ""
	}
	s.dummyHash = hash
	return s.dummyHash
}
