package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/security"
	"github.com/marina1815/nutrimatch/internal/services"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type AuthHandler struct {
	Cfg      *config.Config
	Auth     *services.AuthService
	Users    repository.UserRepository
	Profiles interface {
		Get(ctx context.Context, userID string) (*models.Profile, *models.Lifestyle, *models.Preferences, *models.Constraints, string, error)
	}
	CSRF  *security.CSRFManager
	OIDC  *services.OIDCService
	MFA   *services.MFAService
	Audit *services.AuditService
}

type registerRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=120"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,max=128"`
}

type mfaLoginTOTPRequest struct {
	ChallengeID string `json:"challengeId" validate:"required,uuid4"`
	Code        string `json:"code" validate:"required,len=6,numeric"`
}

type mfaPreferenceRequest struct {
	PreferredMethod string `json:"preferredMethod" validate:"omitempty,oneof=totp passkey"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required,max=128"`
	NewPassword     string `json:"newPassword" validate:"required,min=12,max=128"`
	ConfirmPassword string `json:"confirmPassword" validate:"required,min=12,max=128"`
}

type totpCodeRequest struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := bindStrictJSON(c, &req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.register",
			ResourceType: "identity.user",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "invalid_payload"},
		})
		respondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	req.Email = validation.NormalizeEmail(req.Email)
	req.Name = validation.NormalizeString(req.Name)

	if err := validation.Validate.Struct(req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.register",
			ResourceType: "identity.user",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "validation_failed"},
		})
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed")
		return
	}

	user := &models.User{
		Email:    req.Email,
		FullName: req.Name,
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.Register(c.Request.Context(), user, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		status := http.StatusBadRequest
		code := "REGISTER_FAILED"
		message := "register failed"
		reason := "registration_failed"
		if errors.Is(err, repository.ErrDuplicate) {
			status = http.StatusConflict
			code = "USER_ALREADY_EXISTS"
			message = "user already exists"
			reason = "user_already_exists"
		}
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.register",
			ResourceType: "identity.user",
			Outcome:      "failed",
			Details:      map[string]any{"reason": reason},
		})
		respondError(c, status, code, message)
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF, h.Auth, access)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.register",
		ResourceType: "identity.user",
		ResourceID:   user.ID,
	}))
	respondOK(c, http.StatusOK, tokenResponse(access, accessExp))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := bindStrictJSON(c, &req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.login",
			ResourceType: "identity.session",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "invalid_payload"},
		})
		respondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	req.Email = validation.NormalizeEmail(req.Email)
	if err := validation.Validate.Struct(req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.login",
			ResourceType: "identity.session",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "validation_failed"},
		})
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed")
		return
	}

	user, err := h.Auth.AuthenticatePrimary(c.Request.Context(), req.Email, req.Password, c.ClientIP())
	if err != nil {
		status := http.StatusUnauthorized
		errorMessage := "invalid credentials"
		reason := "invalid_credentials"
		if err == services.ErrAuthTemporarilyBlocked {
			status = http.StatusTooManyRequests
			errorMessage = "authentication temporarily blocked"
			reason = "temporarily_blocked"
		}
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.login",
			ResourceType: "identity.session",
			Outcome:      "denied",
			Details:      map[string]any{"reason": reason},
		})
		code := "INVALID_CREDENTIALS"
		if err == services.ErrAuthTemporarilyBlocked {
			code = "AUTH_TEMPORARILY_BLOCKED"
		}
		respondError(c, status, code, errorMessage)
		return
	}

	if h.MFA != nil {
		status, statusErr := h.MFA.Status(c.Request.Context(), user.ID)
		if statusErr != nil {
			recordAudit(c, h.Audit, services.AuditRecord{
				UserID:       user.ID,
				EventType:    "auth.login",
				ResourceType: "identity.session",
				Outcome:      "failed",
				Details:      map[string]any{"reason": "mfa_status_failed"},
			})
			respondError(c, http.StatusInternalServerError, "MFA_STATUS_FAILED", "mfa status failed")
			return
		}
		if status.StepUpAvailable {
			challenge, challengeErr := h.MFA.IssueLoginChallenge(c.Request.Context(), user, c.Request.UserAgent(), c.ClientIP())
			if challengeErr != nil {
				recordAudit(c, h.Audit, services.AuditRecord{
					UserID:       user.ID,
					EventType:    "auth.login",
					ResourceType: "identity.mfa_challenge",
					Outcome:      "failed",
				})
				respondError(c, http.StatusInternalServerError, "MFA_CHALLENGE_FAILED", "mfa challenge failed")
				return
			}
			recordAudit(c, h.Audit, services.AuditRecord{
				UserID:       user.ID,
				EventType:    "auth.login.mfa_required",
				ResourceType: "identity.mfa_challenge",
				ResourceID:   challenge.ID,
				Details: map[string]any{
					"preferredMethod": challenge.PreferredMethod,
					"allowedMethods":  challenge.AllowedMethods,
				},
			})
			respondOK(c, http.StatusOK, gin.H{
				"mfa_required":     true,
				"challenge_id":     challenge.ID,
				"preferred_method": challenge.PreferredMethod,
				"allowed_methods":  challenge.AllowedMethods,
				"expires_at":       challenge.ExpiresAt.Format(time.RFC3339),
			})
			return
		}
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.IssueSession(c.Request.Context(), user.ID, "local", c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       user.ID,
			EventType:    "auth.login",
			ResourceType: "identity.session",
			Outcome:      "failed",
			Details:      map[string]any{"reason": "session_issue_failed"},
		})
		respondError(c, http.StatusInternalServerError, "LOGIN_FAILED", "login failed")
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF, h.Auth, access)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.login",
		ResourceType: "identity.session",
	}))
	respondOK(c, http.StatusOK, tokenResponse(access, accessExp))
}

func (h *AuthHandler) CompleteMFALoginTOTP(c *gin.Context) {
	var req mfaLoginTOTPRequest
	if err := bindStrictJSON(c, &req); err != nil || validation.Validate.Struct(req) != nil {
		respondError(c, http.StatusBadRequest, "INVALID_MFA_CHALLENGE", "invalid mfa challenge")
		return
	}
	if h.MFA == nil {
		respondError(c, http.StatusServiceUnavailable, "MFA_UNAVAILABLE", "mfa unavailable")
		return
	}

	challenge, err := h.MFA.ConsumeLoginChallenge(c.Request.Context(), req.ChallengeID, "totp")
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.login.mfa.totp",
			ResourceType: "identity.mfa_challenge",
			ResourceID:   req.ChallengeID,
			Outcome:      "denied",
			Details:      map[string]any{"reason": "challenge_invalid"},
		})
		respondError(c, http.StatusUnauthorized, "MFA_CHALLENGE_INVALID", "mfa challenge invalid")
		return
	}
	if err := h.MFA.VerifyTOTP(c.Request.Context(), challenge.UserID, req.Code); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       challenge.UserID,
			EventType:    "auth.login.mfa.totp",
			ResourceType: "identity.mfa_challenge",
			ResourceID:   req.ChallengeID,
			Outcome:      "denied",
			Details:      map[string]any{"reason": "totp_invalid"},
		})
		respondError(c, http.StatusUnauthorized, "MFA_VERIFICATION_FAILED", "mfa verification failed")
		return
	}
	h.issueMFALoginSession(c, challenge.UserID, "local+mfa:totp", "auth.login.mfa.totp")
}

func (h *AuthHandler) BeginMFALoginPasskey(c *gin.Context) {
	var req struct {
		ChallengeID string `json:"challengeId" validate:"required,uuid4"`
	}
	if err := bindStrictJSON(c, &req); err != nil || validation.Validate.Struct(req) != nil {
		respondError(c, http.StatusBadRequest, "INVALID_MFA_CHALLENGE", "invalid mfa challenge")
		return
	}
	if h.MFA == nil {
		respondError(c, http.StatusServiceUnavailable, "MFA_UNAVAILABLE", "mfa unavailable")
		return
	}
	challenge, err := h.MFA.GetLoginChallenge(c.Request.Context(), req.ChallengeID)
	if err != nil || !containsString(challenge.AllowedMethods, "passkey") {
		respondError(c, http.StatusUnauthorized, "MFA_CHALLENGE_INVALID", "mfa challenge invalid")
		return
	}
	options, passkeyChallengeID, err := h.MFA.BeginPasskeyAuthentication(c.Request.Context(), challenge.UserID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PASSKEY_BEGIN_FAILED", "passkey begin failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       challenge.UserID,
		EventType:    "auth.login.mfa.passkey.begin",
		ResourceType: "identity.mfa_challenge",
		ResourceID:   req.ChallengeID,
	})
	respondOK(c, http.StatusOK, gin.H{"challengeId": passkeyChallengeID, "options": options})
}

func (h *AuthHandler) CompleteMFALoginPasskey(c *gin.Context) {
	challengeID := strings.TrimSpace(c.Query("challengeId"))
	passkeyChallengeID := strings.TrimSpace(c.Query("passkeyChallengeId"))
	if challengeID == "" || passkeyChallengeID == "" {
		respondError(c, http.StatusBadRequest, "INVALID_MFA_CHALLENGE", "invalid mfa challenge")
		return
	}
	if h.MFA == nil {
		respondError(c, http.StatusServiceUnavailable, "MFA_UNAVAILABLE", "mfa unavailable")
		return
	}

	challenge, err := h.MFA.GetLoginChallenge(c.Request.Context(), challengeID)
	if err != nil || !containsString(challenge.AllowedMethods, "passkey") {
		respondError(c, http.StatusUnauthorized, "MFA_CHALLENGE_INVALID", "mfa challenge invalid")
		return
	}
	if err := h.MFA.FinishPasskeyAuthentication(c.Request.Context(), challenge.UserID, passkeyChallengeID, c.Request); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       challenge.UserID,
			EventType:    "auth.login.mfa.passkey.finish",
			ResourceType: "identity.mfa_challenge",
			ResourceID:   challengeID,
			Outcome:      "denied",
			Details:      map[string]any{"reason": "passkey_invalid"},
		})
		respondError(c, http.StatusUnauthorized, "MFA_VERIFICATION_FAILED", "mfa verification failed")
		return
	}
	if _, err := h.MFA.ConsumeLoginChallenge(c.Request.Context(), challengeID, "passkey"); err != nil {
		respondError(c, http.StatusUnauthorized, "MFA_CHALLENGE_INVALID", "mfa challenge invalid")
		return
	}
	h.issueMFALoginSession(c, challenge.UserID, "local+mfa:passkey", "auth.login.mfa.passkey.finish")
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(h.Cfg.CookieNameRefresh)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.refresh",
			ResourceType: "identity.session",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "missing_refresh_cookie"},
		})
		respondError(c, http.StatusUnauthorized, "MISSING_REFRESH_TOKEN", "missing refresh token")
		return
	}
	if !h.validateRefreshCSRF(c, refreshToken) {
		return
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.refresh",
			ResourceType: "identity.session",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "invalid_refresh_token"},
		})
		respondError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "invalid refresh token")
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF, h.Auth, access)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.refresh",
		ResourceType: "identity.session",
	}))
	respondOK(c, http.StatusOK, tokenResponse(access, accessExp))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(h.Cfg.CookieNameRefresh)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.logout",
			ResourceType: "identity.session",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "missing_refresh_cookie"},
		})
		respondError(c, http.StatusUnauthorized, "MISSING_REFRESH_TOKEN", "missing refresh token")
		return
	}
	if !h.validateRefreshCSRF(c, refreshToken) {
		return
	}

	if err := h.Auth.Logout(c.Request.Context(), refreshToken); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.logout",
			ResourceType: "identity.session",
			Outcome:      "failed",
			Details:      map[string]any{"reason": "logout_failed"},
		})
		respondError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "invalid refresh token")
		return
	}
	clearRefreshCookie(c, h.Cfg)
	clearCSRFCookie(c, h.Cfg)
	c.Writer.Header().Set("Clear-Site-Data", `"cache", "storage"`)
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.logout",
		ResourceType: "identity.session",
	})
	respondNoContent(c)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := bindStrictJSON(c, &req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.password.change",
			ResourceType: "identity.user",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "invalid_payload"},
		})
		respondError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}
	if err := validation.Validate.Struct(req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.password.change",
			ResourceType: "identity.user",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "validation_failed"},
		})
		respondError(c, http.StatusBadRequest, "VALIDATION_FAILED", "validation failed")
		return
	}

	userID := c.GetString("user_id")
	sessionID := c.GetString("session_id")
	err := h.Auth.ChangePassword(c.Request.Context(), userID, sessionID, req.CurrentPassword, req.NewPassword, req.ConfirmPassword)
	if err != nil {
		status := http.StatusUnauthorized
		code := "INVALID_CURRENT_PASSWORD"
		message := "invalid current password"
		if err == services.ErrPasswordConfirmationMismatch {
			status = http.StatusBadRequest
			code = "PASSWORD_CONFIRMATION_MISMATCH"
			message = "password confirmation mismatch"
		}
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			SessionID:    sessionID,
			EventType:    "auth.password.change",
			ResourceType: "identity.user",
			ResourceID:   userID,
			Outcome:      "denied",
			Details:      map[string]any{"reason": code},
		})
		respondError(c, status, code, message)
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		SessionID:    sessionID,
		EventType:    "auth.password.change",
		ResourceType: "identity.user",
		ResourceID:   userID,
		Details:      map[string]any{"otherSessionsRevoked": true},
	})
	respondNoContent(c)
}

func (h *AuthHandler) CSRFToken(c *gin.Context) {
	if h.CSRF == nil {
		respondError(c, http.StatusServiceUnavailable, "CSRF_UNAVAILABLE", "csrf unavailable")
		return
	}

	sessionID := ""
	csrfBindingID := ""
	if refreshToken, cookieErr := c.Cookie(h.Cfg.CookieNameRefresh); cookieErr == nil && h.Auth != nil {
		if session, sessionErr := h.Auth.SessionFromRefreshToken(c.Request.Context(), refreshToken); sessionErr == nil {
			sessionID = session.ID
			csrfBindingID = session.CSRFBindingID
			c.Set("user_id", session.UserID)
			c.Set("session_id", session.ID)
			c.Set("csrf_binding_id", session.CSRFBindingID)
			c.Set("auth_method", session.AuthMethod)
		}
	}

	token, err := h.CSRF.IssueTokenForSession(sessionID, csrfBindingID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.csrf.issue",
			ResourceType: "identity.csrf",
			Outcome:      "failed",
		})
		respondError(c, http.StatusInternalServerError, "CSRF_ISSUE_FAILED", "csrf issue failed")
		return
	}
	setCSRFCookie(c, h.Cfg, token, time.Now().Add(h.Cfg.CSRFTTL))
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.csrf.issue",
		ResourceType: "identity.csrf",
	})
	respondOK(c, http.StatusOK, gin.H{"csrf_token": token, "header_name": h.Cfg.CSRFHeaderName})
}

func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	if h.OIDC == nil || !h.OIDC.Enabled() {
		respondError(c, http.StatusServiceUnavailable, "OIDC_UNAVAILABLE", "oidc unavailable")
		return
	}

	authURL, signedState, err := h.OIDC.BeginAuth(c.Request.Context(), c.Query("redirect"))
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.oidc.begin",
			ResourceType: "identity.external",
			Outcome:      "failed",
			Details:      map[string]any{"provider": h.Cfg.OIDCProviderName},
		})
		respondError(c, http.StatusInternalServerError, "OIDC_INIT_FAILED", "oidc init failed")
		return
	}
	setOIDCCookie(c, h.Cfg, signedState, time.Now().Add(h.Cfg.CSRFTTL))
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.oidc.begin",
		ResourceType: "identity.external",
		Details:      map[string]any{"provider": h.Cfg.OIDCProviderName},
	})
	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	if h.OIDC == nil || !h.OIDC.Enabled() {
		respondError(c, http.StatusServiceUnavailable, "OIDC_UNAVAILABLE", "oidc unavailable")
		return
	}

	stateCookie, err := c.Cookie(h.Cfg.CookieNameOIDC)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "MISSING_OIDC_STATE", "missing oidc state")
		return
	}

	access, _, refresh, refreshExp, redirectPath, err := h.OIDC.CompleteAuth(
		c.Request.Context(),
		stateCookie,
		c.Query("state"),
		c.Query("code"),
		c.Request.UserAgent(),
		c.ClientIP(),
	)
	if err != nil {
		clearOIDCCookie(c, h.Cfg)
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.oidc.callback",
			ResourceType: "identity.external",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "callback_failed", "provider": h.Cfg.OIDCProviderName},
		})
		respondError(c, http.StatusUnauthorized, "OIDC_CALLBACK_FAILED", "oidc callback failed")
		return
	}

	clearOIDCCookie(c, h.Cfg)
	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF, h.Auth, access)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.oidc.callback",
		ResourceType: "identity.external",
		Details:      map[string]any{"provider": h.Cfg.OIDCProviderName},
	}))

	target := h.Cfg.OIDCFrontendSuccessURL + "?next=" + url.QueryEscape(redirectPath)
	c.Redirect(http.StatusFound, target)
}

func (h *AuthHandler) WhoAmI(c *gin.Context) {
	userID := c.GetString("user_id")
	user, err := h.Users.GetByID(c.Request.Context(), userID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "auth.whoami",
			ResourceType: "identity.user",
			Outcome:      "failed",
		})
		respondError(c, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}

	hasProfile := false
	profileID := ""
	if h.Profiles != nil {
		profile, _, _, _, _, profileErr := h.Profiles.Get(c.Request.Context(), userID)
		if profileErr == nil && profile != nil {
			hasProfile = true
			profileID = profile.ID
		}
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "auth.whoami",
		ResourceType: "identity.user",
		ResourceID:   user.ID,
	})
	respondOK(c, http.StatusOK, gin.H{
		"userId":     user.ID,
		"email":      user.Email,
		"fullName":   user.FullName,
		"sessionId":  c.GetString("session_id"),
		"authMethod": c.GetString("auth_method"),
		"hasProfile": hasProfile,
		"profileId":  profileID,
	})
}

func (h *AuthHandler) MFAStatus(c *gin.Context) {
	if h.MFA == nil {
		respondOK(c, http.StatusOK, services.MFAStatus{})
		return
	}
	status, err := h.MFA.Status(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "MFA_STATUS_FAILED", "mfa status failed")
		return
	}
	respondOK(c, http.StatusOK, status)
}

func (h *AuthHandler) SetMFAPreference(c *gin.Context) {
	var req mfaPreferenceRequest
	if err := bindStrictJSON(c, &req); err != nil || validation.Validate.Struct(req) != nil {
		respondError(c, http.StatusBadRequest, "INVALID_MFA_PREFERENCE", "invalid mfa preference")
		return
	}
	if h.MFA == nil {
		respondError(c, http.StatusServiceUnavailable, "MFA_UNAVAILABLE", "mfa unavailable")
		return
	}
	userID := c.GetString("user_id")
	if err := h.MFA.SetPreferredMethod(c.Request.Context(), userID, req.PreferredMethod); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "auth.mfa.preference",
			ResourceType: "identity.mfa",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "method_not_available"},
		})
		respondError(c, http.StatusBadRequest, "MFA_METHOD_NOT_AVAILABLE", "mfa method not available")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		EventType:    "auth.mfa.preference",
		ResourceType: "identity.mfa",
		Details:      map[string]any{"preferredMethod": req.PreferredMethod},
	})
	respondNoContent(c)
}

func (h *AuthHandler) BeginTOTPSetup(c *gin.Context) {
	setup, err := h.MFA.BeginTOTPSetup(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.mfa.totp.setup",
			ResourceType: "identity.mfa",
			Outcome:      "failed",
		})
		respondError(c, http.StatusInternalServerError, "TOTP_SETUP_FAILED", "totp setup failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.mfa.totp.setup",
		ResourceType: "identity.mfa",
	})
	respondOK(c, http.StatusOK, setup)
}

func (h *AuthHandler) ConfirmTOTP(c *gin.Context) {
	var req totpCodeRequest
	if err := bindStrictJSON(c, &req); err != nil || validation.Validate.Struct(req) != nil {
		respondError(c, http.StatusBadRequest, "INVALID_TOTP_CODE", "invalid totp code")
		return
	}
	if err := h.MFA.ConfirmTOTP(c.Request.Context(), c.GetString("user_id"), req.Code); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.mfa.totp.confirm",
			ResourceType: "identity.mfa",
			Outcome:      "denied",
		})
		respondError(c, http.StatusUnauthorized, "TOTP_VERIFICATION_FAILED", "totp verification failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.mfa.totp.confirm",
		ResourceType: "identity.mfa",
	})
	respondNoContent(c)
}

func (h *AuthHandler) DisableTOTP(c *gin.Context) {
	var req totpCodeRequest
	if err := bindStrictJSON(c, &req); err != nil || validation.Validate.Struct(req) != nil {
		respondError(c, http.StatusBadRequest, "INVALID_TOTP_CODE", "invalid totp code")
		return
	}
	if err := h.MFA.DisableTOTP(c.Request.Context(), c.GetString("user_id"), req.Code); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.mfa.totp.disable",
			ResourceType: "identity.mfa",
			Outcome:      "denied",
		})
		respondError(c, http.StatusUnauthorized, "TOTP_VERIFICATION_FAILED", "totp verification failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.mfa.totp.disable",
		ResourceType: "identity.mfa",
	})
	respondNoContent(c)
}

func (h *AuthHandler) BeginPasskeyRegistration(c *gin.Context) {
	options, challengeID, err := h.MFA.BeginPasskeyRegistration(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PASSKEY_BEGIN_FAILED", "passkey begin failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.mfa.passkey.registration.begin",
		ResourceType: "identity.mfa",
	})
	respondOK(c, http.StatusOK, gin.H{"challengeId": challengeID, "options": options})
}

func (h *AuthHandler) FinishPasskeyRegistration(c *gin.Context) {
	challengeID := strings.TrimSpace(c.Query("challengeId"))
	displayName := validation.NormalizeString(c.Query("displayName"))
	if challengeID == "" {
		respondError(c, http.StatusBadRequest, "MISSING_CHALLENGE_ID", "missing challenge id")
		return
	}
	if err := h.MFA.FinishPasskeyRegistration(c.Request.Context(), c.GetString("user_id"), challengeID, displayName, c.Request); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.mfa.passkey.registration.finish",
			ResourceType: "identity.mfa",
			Outcome:      "denied",
		})
		respondError(c, http.StatusUnauthorized, "PASSKEY_REGISTRATION_FAILED", "passkey registration failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.mfa.passkey.registration.finish",
		ResourceType: "identity.mfa",
	})
	respondNoContent(c)
}

func (h *AuthHandler) BeginPasskeyAuthentication(c *gin.Context) {
	options, challengeID, err := h.MFA.BeginPasskeyAuthentication(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PASSKEY_BEGIN_FAILED", "passkey begin failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.mfa.passkey.authentication.begin",
		ResourceType: "identity.mfa",
	})
	respondOK(c, http.StatusOK, gin.H{"challengeId": challengeID, "options": options})
}

func (h *AuthHandler) FinishPasskeyAuthentication(c *gin.Context) {
	challengeID := strings.TrimSpace(c.Query("challengeId"))
	if challengeID == "" {
		respondError(c, http.StatusBadRequest, "MISSING_CHALLENGE_ID", "missing challenge id")
		return
	}
	if err := h.MFA.FinishPasskeyAuthentication(c.Request.Context(), c.GetString("user_id"), challengeID, c.Request); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.mfa.passkey.authentication.finish",
			ResourceType: "identity.mfa",
			Outcome:      "denied",
		})
		respondError(c, http.StatusUnauthorized, "PASSKEY_AUTHENTICATION_FAILED", "passkey authentication failed")
		return
	}
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.mfa.passkey.authentication.finish",
		ResourceType: "identity.mfa",
	})
	respondNoContent(c)
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID := c.GetString("user_id")
	currentSessionID := c.GetString("session_id")
	sessions, err := h.Auth.ListSessions(c.Request.Context(), userID, currentSessionID)
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    "auth.sessions.list",
			ResourceType: "identity.session",
			Outcome:      "failed",
		})
		respondError(c, http.StatusInternalServerError, "SESSIONS_FAILED", "sessions failed")
		return
	}

	items := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, gin.H{
			"id":            session.ID,
			"authMethod":    session.AuthMethod,
			"expiresAt":     session.ExpiresAt.Format(time.RFC3339),
			"idleExpiresAt": session.IdleExpiresAt.Format(time.RFC3339),
			"createdAt":     session.CreatedAt.Format(time.RFC3339),
			"lastSeenAt":    session.LastSeenAt.Format(time.RFC3339),
			"revoked":       session.RevokedAt != nil,
			"current":       session.Current,
		})
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		SessionID:    currentSessionID,
		EventType:    "auth.sessions.list",
		ResourceType: "identity.session",
		Details:      map[string]any{"count": len(items)},
	})
	respondOK(c, http.StatusOK, gin.H{"sessions": items})
}

func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userID := c.GetString("user_id")
	currentSessionID := c.GetString("session_id")
	sessionID := strings.TrimSpace(c.Param("sessionId"))
	if sessionID == "" {
		respondError(c, http.StatusBadRequest, "INVALID_SESSION_ID", "invalid session id")
		return
	}

	if err := h.Auth.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			SessionID:    currentSessionID,
			EventType:    "auth.sessions.revoke",
			ResourceType: "identity.session",
			ResourceID:   sessionID,
			Outcome:      "denied",
		})
		respondError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
		return
	}

	recordAudit(c, h.Audit, services.AuditRecord{
		UserID:       userID,
		SessionID:    currentSessionID,
		EventType:    "auth.sessions.revoke",
		ResourceType: "identity.session",
		ResourceID:   sessionID,
	})
	respondNoContent(c)
}

func (h *AuthHandler) issueMFALoginSession(c *gin.Context, userID, authMethod, eventType string) {
	access, accessExp, refresh, refreshExp, err := h.Auth.IssueSession(c.Request.Context(), userID, authMethod, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       userID,
			EventType:    eventType,
			ResourceType: "identity.session",
			Outcome:      "failed",
		})
		respondError(c, http.StatusInternalServerError, "LOGIN_FAILED", "login failed")
		return
	}
	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF, h.Auth, access)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		UserID:       userID,
		EventType:    eventType,
		ResourceType: "identity.session",
	}))
	respondOK(c, http.StatusOK, tokenResponse(access, accessExp))
}

func (h *AuthHandler) tokenAuditRecord(access string, record services.AuditRecord) services.AuditRecord {
	if h == nil || h.Auth == nil || h.Auth.Tokens == nil {
		return record
	}

	claims, err := h.Auth.Tokens.ParseAccessToken(access)
	if err != nil {
		return record
	}
	if record.UserID == "" {
		record.UserID = claims.Subject
	}
	if record.SessionID == "" {
		record.SessionID = claims.SessionID
	}
	if record.ResourceID == "" {
		record.ResourceID = claims.SessionID
	}
	return record
}

func containsString(values models.StringSlice, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (h *AuthHandler) validateRefreshCSRF(c *gin.Context, refreshToken string) bool {
	if h == nil || h.CSRF == nil || h.Auth == nil {
		return true
	}
	csrfToken, err := c.Cookie(h.Cfg.CookieNameCSRF)
	if err != nil {
		respondError(c, http.StatusForbidden, "MISSING_CSRF_COOKIE", "missing csrf cookie")
		return false
	}
	session, err := h.Auth.SessionFromRefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "invalid refresh token")
		return false
	}
	if err := h.CSRF.ValidateTokenForSession(csrfToken, session.ID, session.CSRFBindingID); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			UserID:       session.UserID,
			SessionID:    session.ID,
			EventType:    "auth.csrf.validate",
			ResourceType: "identity.csrf",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "session_mismatch"},
		})
		respondError(c, http.StatusForbidden, "INVALID_CSRF_TOKEN", "invalid csrf token")
		return false
	}
	c.Set("user_id", session.UserID)
	c.Set("session_id", session.ID)
	c.Set("csrf_binding_id", session.CSRFBindingID)
	c.Set("auth_method", session.AuthMethod)
	return true
}

func tokenResponse(access string, exp time.Time) gin.H {
	return gin.H{
		"access_token": access,
		"expires_at":   exp.Format(time.RFC3339),
	}
}

func setRefreshCookie(c *gin.Context, cfg *config.Config, token string, exp time.Time) {
	cookie := &http.Cookie{
		Name:     cfg.CookieNameRefresh,
		Value:    token,
		Path:     cfg.CookiePathRefresh,
		Domain:   cfg.CookieDomain,
		Expires:  exp,
		MaxAge:   int(time.Until(exp).Seconds()),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: parseSameSite(cfg.CookieSameSite),
	}
	http.SetCookie(c.Writer, cookie)
}

func clearRefreshCookie(c *gin.Context, cfg *config.Config) {
	cookie := &http.Cookie{
		Name:     cfg.CookieNameRefresh,
		Value:    "",
		Path:     cfg.CookiePathRefresh,
		Domain:   cfg.CookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: parseSameSite(cfg.CookieSameSite),
	}
	http.SetCookie(c.Writer, cookie)
}

func parseSameSite(input string) http.SameSite {
	switch {
	case strings.EqualFold(input, "Strict"):
		return http.SameSiteStrictMode
	case strings.EqualFold(input, "None"):
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func setCSRFCookie(c *gin.Context, cfg *config.Config, token string, exp time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.CookieNameCSRF,
		Value:    token,
		Path:     cfg.CookiePathCSRF,
		Domain:   cfg.CookieDomain,
		Expires:  exp,
		MaxAge:   int(time.Until(exp).Seconds()),
		HttpOnly: false,
		Secure:   cfg.CookieSecure,
		SameSite: parseSameSite(cfg.CookieSameSite),
	})
}

func ensureCSRFCookie(c *gin.Context, cfg *config.Config, manager *security.CSRFManager, auth *services.AuthService, access string) {
	if manager == nil {
		return
	}
	sessionID := ""
	csrfBindingID := ""
	if auth != nil && auth.Tokens != nil && access != "" {
		if claims, err := auth.Tokens.ParseAccessToken(access); err == nil && auth.Sessions != nil {
			if session, sessionErr := auth.Sessions.GetByID(c.Request.Context(), claims.SessionID); sessionErr == nil {
				sessionID = session.ID
				csrfBindingID = session.CSRFBindingID
			}
		}
	}
	token, err := manager.IssueTokenForSession(sessionID, csrfBindingID)
	if err != nil {
		return
	}
	setCSRFCookie(c, cfg, token, time.Now().Add(cfg.CSRFTTL))
}

func clearCSRFCookie(c *gin.Context, cfg *config.Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.CookieNameCSRF,
		Value:    "",
		Path:     cfg.CookiePathCSRF,
		Domain:   cfg.CookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   cfg.CookieSecure,
		SameSite: parseSameSite(cfg.CookieSameSite),
	})
}

func setOIDCCookie(c *gin.Context, cfg *config.Config, value string, exp time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.CookieNameOIDC,
		Value:    value,
		Path:     cfg.CookiePathRefresh,
		Domain:   cfg.CookieDomain,
		Expires:  exp,
		MaxAge:   int(time.Until(exp).Seconds()),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: parseSameSite(cfg.CookieSameSite),
	})
}

func clearOIDCCookie(c *gin.Context, cfg *config.Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.CookieNameOIDC,
		Value:    "",
		Path:     cfg.CookiePathRefresh,
		Domain:   cfg.CookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: parseSameSite(cfg.CookieSameSite),
	})
}
