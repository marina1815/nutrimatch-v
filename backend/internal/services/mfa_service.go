package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var ErrMFAUnavailable = errors.New("mfa unavailable")
var ErrMFAVerificationFailed = errors.New("mfa verification failed")

type MFAService struct {
	Repo     repository.MFARepository
	Users    repository.UserRepository
	Cipher   *security.Cipher
	WebAuthn *webauthn.WebAuthn
	Issuer   string
}

type MFAStatus struct {
	TOTPEnabled     bool `json:"totpEnabled"`
	PasskeyEnabled  bool `json:"passkeyEnabled"`
	PasskeyCount    int  `json:"passkeyCount"`
	StepUpAvailable bool `json:"stepUpAvailable"`
}

type TOTPSetup struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauthUrl"`
}

func (s *MFAService) Status(ctx context.Context, userID string) (*MFAStatus, error) {
	if s == nil || s.Repo == nil {
		return &MFAStatus{}, nil
	}
	status := &MFAStatus{}
	if secret, err := s.Repo.GetTOTPSecret(ctx, userID); err == nil {
		status.TOTPEnabled = secret.Enabled
	}
	credentials, err := s.Repo.ListWebAuthnCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	status.PasskeyCount = len(credentials)
	status.PasskeyEnabled = len(credentials) > 0
	status.StepUpAvailable = status.TOTPEnabled || status.PasskeyEnabled
	return status, nil
}

func (s *MFAService) BeginTOTPSetup(ctx context.Context, userID string) (*TOTPSetup, error) {
	if s == nil || s.Repo == nil || s.Users == nil || s.Cipher == nil {
		return nil, ErrMFAUnavailable
	}
	user, err := s.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	issuer := s.Issuer
	if issuer == "" {
		issuer = "NutriMatch"
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user.Email,
		Period:      30,
		SecretSize:  32,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, err
	}
	ciphertext, err := s.Cipher.Encrypt(key.Secret())
	if err != nil {
		return nil, err
	}
	if err := s.Repo.UpsertTOTPSecret(ctx, &models.TOTPSecret{
		UserID:           userID,
		SecretCiphertext: ciphertext,
		Enabled:          false,
	}); err != nil {
		return nil, err
	}
	return &TOTPSetup{Secret: key.Secret(), OTPAuthURL: key.URL()}, nil
}

func (s *MFAService) ConfirmTOTP(ctx context.Context, userID, code string) error {
	valid, err := s.verifyTOTP(ctx, userID, code, false)
	if err != nil {
		return err
	}
	if !valid {
		return ErrMFAVerificationFailed
	}
	return s.Repo.EnableTOTP(ctx, userID, time.Now().UTC())
}

func (s *MFAService) VerifyTOTP(ctx context.Context, userID, code string) error {
	valid, err := s.verifyTOTP(ctx, userID, code, true)
	if err != nil {
		return err
	}
	if !valid {
		return ErrMFAVerificationFailed
	}
	return nil
}

func (s *MFAService) DisableTOTP(ctx context.Context, userID, code string) error {
	if err := s.VerifyTOTP(ctx, userID, code); err != nil {
		return err
	}
	return s.Repo.DisableTOTP(ctx, userID)
}

func (s *MFAService) verifyTOTP(ctx context.Context, userID, code string, requireEnabled bool) (bool, error) {
	if s == nil || s.Repo == nil || s.Cipher == nil {
		return false, ErrMFAUnavailable
	}
	secretRecord, err := s.Repo.GetTOTPSecret(ctx, userID)
	if err != nil {
		return false, err
	}
	if requireEnabled && !secretRecord.Enabled {
		return false, ErrMFAVerificationFailed
	}
	secret, err := s.Cipher.Decrypt(secretRecord.SecretCiphertext)
	if err != nil {
		return false, err
	}
	return totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
}

func (s *MFAService) BeginPasskeyRegistration(ctx context.Context, userID string) (any, string, error) {
	user, err := s.webAuthnUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	options, sessionData, err := s.WebAuthn.BeginRegistration(user)
	if err != nil {
		return nil, "", err
	}
	challengeID, err := s.storeChallenge(ctx, userID, "registration", sessionData)
	return options, challengeID, err
}

func (s *MFAService) FinishPasskeyRegistration(ctx context.Context, userID, challengeID, displayName string, r *http.Request) error {
	user, err := s.webAuthnUser(ctx, userID)
	if err != nil {
		return err
	}
	sessionData, err := s.consumeChallenge(ctx, userID, challengeID, "registration")
	if err != nil {
		return err
	}
	credential, err := s.WebAuthn.FinishRegistration(user, *sessionData, r)
	if err != nil {
		return err
	}
	payload := map[string]any{}
	raw, _ := json.Marshal(credential)
	_ = json.Unmarshal(raw, &payload)
	return s.Repo.CreateWebAuthnCredential(ctx, &models.WebAuthnCredential{
		UserID:         userID,
		CredentialID:   base64.RawURLEncoding.EncodeToString(credential.ID),
		CredentialJSON: models.JSONMap(payload),
		DisplayName:    displayName,
	})
}

func (s *MFAService) BeginPasskeyAuthentication(ctx context.Context, userID string) (any, string, error) {
	user, err := s.webAuthnUser(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	options, sessionData, err := s.WebAuthn.BeginLogin(user)
	if err != nil {
		return nil, "", err
	}
	challengeID, err := s.storeChallenge(ctx, userID, "authentication", sessionData)
	return options, challengeID, err
}

func (s *MFAService) FinishPasskeyAuthentication(ctx context.Context, userID, challengeID string, r *http.Request) error {
	user, err := s.webAuthnUser(ctx, userID)
	if err != nil {
		return err
	}
	sessionData, err := s.consumeChallenge(ctx, userID, challengeID, "authentication")
	if err != nil {
		return err
	}
	credential, err := s.WebAuthn.FinishLogin(user, *sessionData, r)
	if err != nil {
		return err
	}
	return s.Repo.UpdateWebAuthnCredentialUsed(ctx, base64.RawURLEncoding.EncodeToString(credential.ID), time.Now().UTC())
}

func (s *MFAService) storeChallenge(ctx context.Context, userID, kind string, sessionData *webauthn.SessionData) (string, error) {
	payload := map[string]any{}
	raw, _ := json.Marshal(sessionData)
	_ = json.Unmarshal(raw, &payload)
	challenge := &models.WebAuthnChallenge{
		ID:          uuid.NewString(),
		UserID:      userID,
		Kind:        kind,
		SessionData: models.JSONMap(payload),
		ExpiresAt:   sessionData.Expires,
	}
	if err := s.Repo.CreateWebAuthnChallenge(ctx, challenge); err != nil {
		return "", err
	}
	return challenge.ID, nil
}

func (s *MFAService) consumeChallenge(ctx context.Context, userID, challengeID, kind string) (*webauthn.SessionData, error) {
	challenge, err := s.Repo.ConsumeWebAuthnChallenge(ctx, userID, challengeID, kind, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(challenge.SessionData)
	var sessionData webauthn.SessionData
	if err := json.Unmarshal(raw, &sessionData); err != nil {
		return nil, err
	}
	return &sessionData, nil
}

func (s *MFAService) webAuthnUser(ctx context.Context, userID string) (*mfaWebAuthnUser, error) {
	if s == nil || s.Repo == nil || s.Users == nil || s.WebAuthn == nil {
		return nil, ErrMFAUnavailable
	}
	user, err := s.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	records, err := s.Repo.ListWebAuthnCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	credentials := make([]webauthn.Credential, 0, len(records))
	for _, record := range records {
		raw, _ := json.Marshal(record.CredentialJSON)
		var credential webauthn.Credential
		if err := json.Unmarshal(raw, &credential); err == nil {
			credentials = append(credentials, credential)
		}
	}
	return &mfaWebAuthnUser{user: user, credentials: credentials}, nil
}

type mfaWebAuthnUser struct {
	user        *models.User
	credentials []webauthn.Credential
}

func (u *mfaWebAuthnUser) WebAuthnID() []byte {
	parsed, err := uuid.Parse(u.user.ID)
	if err != nil {
		return []byte(u.user.ID)
	}
	return parsed[:]
}

func (u *mfaWebAuthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *mfaWebAuthnUser) WebAuthnDisplayName() string {
	return u.user.FullName
}

func (u *mfaWebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}
