package handlers

import (
	"fmt"
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
	Cfg   *config.Config
	Auth  *services.AuthService
	Users repository.UserRepository
	CSRF  *security.CSRFManager
	OIDC  *services.OIDCService
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

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := bindStrictJSON(c, &req); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.register",
			ResourceType: "identity.user",
			Outcome:      "denied",
			Details:      map[string]any{"reason": "invalid_payload"},
		})
		respondError(c, http.StatusBadRequest, "invalid payload")
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
		respondError(c, http.StatusBadRequest, "validation failed")
		return
	}

	user := &models.User{
		Email:    req.Email,
		FullName: req.Name,
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.Register(c.Request.Context(), user, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.register",
			ResourceType: "identity.user",
			Outcome:      "failed",
			Details:      map[string]any{"reason": "registration_failed"},
		})
		respondError(c, http.StatusBadRequest, "register failed")
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.register",
		ResourceType: "identity.user",
		ResourceID:   user.ID,
	}))
	c.JSON(http.StatusOK, tokenResponse(access, accessExp))
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
		respondError(c, http.StatusBadRequest, "invalid payload")
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
		respondError(c, http.StatusBadRequest, "validation failed")
		return
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
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
		respondError(c, status, errorMessage)
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.login",
		ResourceType: "identity.session",
	}))
	c.JSON(http.StatusOK, tokenResponse(access, accessExp))
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
		respondError(c, http.StatusUnauthorized, "missing refresh token")
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
		respondError(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.refresh",
		ResourceType: "identity.session",
	}))
	c.JSON(http.StatusOK, tokenResponse(access, accessExp))
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
		respondError(c, http.StatusUnauthorized, "missing refresh token")
		return
	}

	if err := h.Auth.Logout(c.Request.Context(), refreshToken); err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.logout",
			ResourceType: "identity.session",
			Outcome:      "failed",
			Details:      map[string]any{"reason": "logout_failed"},
		})
		respondError(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	clearRefreshCookie(c, h.Cfg)
	clearCSRFCookie(c, h.Cfg)
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.logout",
		ResourceType: "identity.session",
	})
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) CSRFToken(c *gin.Context) {
	if h.CSRF == nil {
		respondError(c, http.StatusServiceUnavailable, "csrf unavailable")
		return
	}

	token, err := h.CSRF.IssueToken()
	if err != nil {
		recordAudit(c, h.Audit, services.AuditRecord{
			EventType:    "auth.csrf.issue",
			ResourceType: "identity.csrf",
			Outcome:      "failed",
		})
		respondError(c, http.StatusInternalServerError, "csrf issue failed")
		return
	}
	setCSRFCookie(c, h.Cfg, token, time.Now().Add(h.Cfg.CSRFTTL))
	recordAudit(c, h.Audit, services.AuditRecord{
		EventType:    "auth.csrf.issue",
		ResourceType: "identity.csrf",
	})
	c.JSON(http.StatusOK, gin.H{"csrf_token": token, "header_name": h.Cfg.CSRFHeaderName})
}

func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	if h.OIDC == nil || !h.OIDC.Enabled() {
		respondError(c, http.StatusServiceUnavailable, "oidc unavailable")
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
		respondError(c, http.StatusInternalServerError, "oidc init failed")
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
		respondError(c, http.StatusServiceUnavailable, "oidc unavailable")
		return
	}

	stateCookie, err := c.Cookie(h.Cfg.CookieNameOIDC)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "missing oidc state")
		return
	}

	access, accessExp, refresh, refreshExp, redirectPath, err := h.OIDC.CompleteAuth(
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
		respondError(c, http.StatusUnauthorized, "oidc callback failed")
		return
	}

	clearOIDCCookie(c, h.Cfg)
	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	ensureCSRFCookie(c, h.Cfg, h.CSRF)
	recordAudit(c, h.Audit, h.tokenAuditRecord(access, services.AuditRecord{
		EventType:    "auth.oidc.callback",
		ResourceType: "identity.external",
		Details:      map[string]any{"provider": h.Cfg.OIDCProviderName},
	}))

	target := fmt.Sprintf(
		"%s?next=%s#access_token=%s&expires_at=%s",
		h.Cfg.OIDCFrontendSuccessURL,
		url.QueryEscape(redirectPath),
		access,
		accessExp.Format(time.RFC3339),
	)
	c.Redirect(http.StatusFound, target)
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

func ensureCSRFCookie(c *gin.Context, cfg *config.Config, manager *security.CSRFManager) {
	if manager == nil {
		return
	}
	token, err := manager.IssueToken()
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
