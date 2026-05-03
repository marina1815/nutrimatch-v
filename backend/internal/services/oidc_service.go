package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
)

type OIDCService struct {
	Config       *config.Config
	StateManager *security.StateManager
	Users        repository.UserRepository
	External     repository.ExternalIdentityRepository
	TxManager    repository.TxManager
	Auth         *AuthService
}

type oidcClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func (s *OIDCService) Enabled() bool {
	return s != nil && s.Config != nil && s.Config.OIDCIssuerURL != "" && s.Config.OIDCClientID != "" && s.Config.OIDCClientSecret != "" && s.Config.OIDCRedirectURL != ""
}

func (s *OIDCService) BeginAuth(ctx context.Context, redirectPath string) (string, string, error) {
	if !s.Enabled() {
		return "", "", errors.New("oidc is disabled")
	}
	provider, err := oidc.NewProvider(ctx, s.Config.OIDCIssuerURL)
	if err != nil {
		return "", "", err
	}

	state, signedState, err := s.StateManager.Issue(sanitizeRedirectPath(redirectPath))
	if err != nil {
		return "", "", err
	}

	oauthConfig := s.oauthConfig(provider.Endpoint())
	authURL := oauthConfig.AuthCodeURL(
		state.State,
		oidc.Nonce(state.Nonce),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(state.Verifier)),
	)
	return authURL, signedState, nil
}

func sanitizeRedirectPath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "/results"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/results"
	}
	if strings.HasPrefix(trimmed, "//") || strings.Contains(trimmed, "://") {
		return "/results"
	}
	return trimmed
}

func (s *OIDCService) CompleteAuth(ctx context.Context, rawStateCookie, returnedState, code, userAgent, ip string) (string, time.Time, string, time.Time, string, error) {
	if !s.Enabled() {
		return "", time.Time{}, "", time.Time{}, "", errors.New("oidc is disabled")
	}

	state, err := s.StateManager.Parse(rawStateCookie)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	if state.State != returnedState {
		return "", time.Time{}, "", time.Time{}, "", errors.New("oidc state mismatch")
	}

	provider, err := oidc.NewProvider(ctx, s.Config.OIDCIssuerURL)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	oauthConfig := s.oauthConfig(provider.Endpoint())
	token, err := oauthConfig.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", state.Verifier))
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return "", time.Time{}, "", time.Time{}, "", errors.New("missing id_token")
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: s.Config.OIDCClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	if idToken.Nonce != state.Nonce {
		return "", time.Time{}, "", time.Time{}, "", errors.New("oidc nonce mismatch")
	}

	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	if claims.Email == "" {
		return "", time.Time{}, "", time.Time{}, "", errors.New("oidc email claim missing")
	}

	user, err := s.resolveUser(ctx, claims)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	access, accessExp, refresh, refreshExp, err := s.Auth.IssueSession(ctx, user.ID, "oidc", userAgent, ip)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	return access, accessExp, refresh, refreshExp, state.RedirectPath, nil
}

func (s *OIDCService) oauthConfig(endpoint oauth2.Endpoint) oauth2.Config {
	scopes := s.Config.OIDCScopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	return oauth2.Config{
		ClientID:     s.Config.OIDCClientID,
		ClientSecret: s.Config.OIDCClientSecret,
		RedirectURL:  s.Config.OIDCRedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}
}

func (s *OIDCService) resolveUser(ctx context.Context, claims oidcClaims) (*models.User, error) {
	if identity, err := s.External.GetByProviderSubject(ctx, s.Config.OIDCProviderName, s.Config.OIDCIssuerURL, claims.Subject); err == nil {
		_ = s.External.UpdateLogin(ctx, identity.ID, claims.Email, claims.EmailVerified, time.Now())
		return s.Users.GetByID(ctx, identity.UserID)
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(claims.Email))
	displayName := strings.TrimSpace(claims.Name)
	if displayName == "" {
		displayName = normalizedEmail
	}

	var user *models.User
	existing, err := s.Users.GetByEmail(ctx, normalizedEmail)
	if err == nil {
		user = existing
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if s.TxManager == nil {
		return nil, errors.New("transaction manager is required for oidc")
	}

	err = s.TxManager.WithinTransaction(ctx, func(repos repository.Repositories) error {
		if user == nil {
			placeholderPassword, hashErr := security.HashPassword(uuid.NewString(), s.Auth.PasswordParams)
			if hashErr != nil {
				return hashErr
			}
			user = &models.User{
				Email:        normalizedEmail,
				FullName:     displayName,
				PasswordHash: placeholderPassword,
			}
			if err := repos.Users.Create(ctx, user); err != nil {
				return err
			}
		}

		identity := &models.ExternalIdentity{
			UserID:        user.ID,
			Provider:      s.Config.OIDCProviderName,
			Issuer:        s.Config.OIDCIssuerURL,
			Subject:       claims.Subject,
			Email:         normalizedEmail,
			EmailVerified: claims.EmailVerified,
			LastLoginAt:   time.Now(),
		}
		return repos.ExternalIdentities.Create(ctx, identity)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return s.resolveUser(ctx, claims)
		}
		return nil, err
	}

	return user, nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
