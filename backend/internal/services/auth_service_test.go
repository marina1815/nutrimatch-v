package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/security"
)

type authTestUserRepository struct {
	byEmail map[string]*models.User
}

func (r *authTestUserRepository) Create(_ context.Context, user *models.User) error {
	if r.byEmail == nil {
		r.byEmail = map[string]*models.User{}
	}
	r.byEmail[user.Email] = user
	return nil
}

func (r *authTestUserRepository) GetByEmail(_ context.Context, email string) (*models.User, error) {
	user, ok := r.byEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return user, nil
}

func (r *authTestUserRepository) GetByID(_ context.Context, _ string) (*models.User, error) {
	return nil, errors.New("not found")
}

func (r *authTestUserRepository) UpdateFullName(_ context.Context, _, _ string) error {
	return nil
}

type authTestFailureRepository struct {
	failures []models.AuthFailure
}

func (r *authTestFailureRepository) Create(_ context.Context, failure *models.AuthFailure) error {
	copied := *failure
	r.failures = append(r.failures, copied)
	return nil
}

func (r *authTestFailureRepository) CountRecent(_ context.Context, emailHash, ipHash string, since time.Time) (int64, error) {
	var count int64
	for _, failure := range r.failures {
		if failure.OccurredAt.Before(since) {
			continue
		}
		if failure.EmailHash == emailHash || failure.IPHash == ipHash {
			count++
		}
	}
	return count, nil
}

func authPasswordParams() security.Argon2Params {
	return security.Argon2Params{
		Time:       2,
		Memory:     64 * 1024,
		Threads:    1,
		KeyLength:  32,
		SaltLength: 16,
	}
}

func authTokenManager() *security.TokenManager {
	return &security.TokenManager{
		Secret:      []byte("abcdefghijklmnopqrstuvwxyz123456"),
		Issuer:      "nutrimatch-test",
		Audience:    "nutrimatch-users",
		AccessTTL:   15 * time.Minute,
		RefreshTTL:  24 * time.Hour,
		TokenPepper: []byte("1234567890abcdefghijklmnopqrstuvwxyz"),
	}
}

func TestAuthServiceLoginUnknownAndWrongPasswordShareSameError(t *testing.T) {
	params := authPasswordParams()
	hash, err := security.HashPassword("VeryStrongPass123!", params)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	failures := &authTestFailureRepository{}
	service := &AuthService{
		Users: &authTestUserRepository{
			byEmail: map[string]*models.User{
				"user@example.com": {
					ID:           "user-1",
					Email:        "user@example.com",
					PasswordHash: hash,
				},
			},
		},
		Sessions:       &fakeSessionRepository{},
		Failures:       failures,
		Tokens:         authTokenManager(),
		SessionIdleTTL: 12 * time.Hour,
		FailureWindow:  15 * time.Minute,
		MaxFailures:    5,
		PasswordParams: params,
	}

	_, _, _, _, errUnknown := service.Login(context.Background(), "unknown@example.com", "WrongPassword123!", "ua", "127.0.0.1")
	_, _, _, _, errWrong := service.Login(context.Background(), "user@example.com", "WrongPassword123!", "ua", "127.0.0.2")

	if !errors.Is(errUnknown, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for unknown user, got %v", errUnknown)
	}
	if !errors.Is(errWrong, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for wrong password, got %v", errWrong)
	}
	if len(failures.failures) != 2 {
		t.Fatalf("expected two recorded auth failures, got %d", len(failures.failures))
	}
}

func TestAuthServiceLoginBlocksAfterTooManyRecentFailures(t *testing.T) {
	params := authPasswordParams()
	hash, err := security.HashPassword("VeryStrongPass123!", params)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	email := "blocked@example.com"
	emailHash := security.HashFingerprint(email)
	ipHash := security.HashFingerprint("127.0.0.1")
	failures := &authTestFailureRepository{
		failures: []models.AuthFailure{
			{EmailHash: emailHash, IPHash: ipHash, Reason: "invalid_credentials", OccurredAt: time.Now().Add(-2 * time.Minute)},
			{EmailHash: emailHash, IPHash: ipHash, Reason: "invalid_credentials", OccurredAt: time.Now().Add(-90 * time.Second)},
			{EmailHash: emailHash, IPHash: ipHash, Reason: "invalid_credentials", OccurredAt: time.Now().Add(-30 * time.Second)},
		},
	}
	service := &AuthService{
		Users: &authTestUserRepository{
			byEmail: map[string]*models.User{
				email: {
					ID:           "user-1",
					Email:        email,
					PasswordHash: hash,
				},
			},
		},
		Sessions:       &fakeSessionRepository{},
		Failures:       failures,
		Tokens:         authTokenManager(),
		SessionIdleTTL: 12 * time.Hour,
		FailureWindow:  15 * time.Minute,
		MaxFailures:    3,
		PasswordParams: params,
	}

	_, _, _, _, err = service.Login(context.Background(), email, "VeryStrongPass123!", "ua", "127.0.0.1")
	if !errors.Is(err, ErrAuthTemporarilyBlocked) {
		t.Fatalf("expected ErrAuthTemporarilyBlocked, got %v", err)
	}
}
