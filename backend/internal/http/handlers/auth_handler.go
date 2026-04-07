package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marina1815/nutrimatch/internal/config"
	"github.com/marina1815/nutrimatch/internal/models"
	"github.com/marina1815/nutrimatch/internal/repository"
	"github.com/marina1815/nutrimatch/internal/services"
	"github.com/marina1815/nutrimatch/internal/validation"
)

type AuthHandler struct {
	Cfg   *config.Config
	Auth  *services.AuthService
	Users repository.UserRepository
}

type registerRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=120"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid payload")
		return
	}

	req.Email = validation.NormalizeEmail(req.Email)
	req.Name = validation.NormalizeString(req.Name)

	if err := validation.Validate.Struct(req); err != nil {
		respondError(c, http.StatusBadRequest, "validation failed")
		return
	}

	user := &models.User{
		Email:    req.Email,
		FullName: req.Name,
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.Register(c.Request.Context(), user, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		respondError(c, http.StatusBadRequest, "register failed")
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	c.JSON(http.StatusOK, tokenResponse(access, accessExp))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid payload")
		return
	}

	req.Email = validation.NormalizeEmail(req.Email)
	if err := validation.Validate.Struct(req); err != nil {
		respondError(c, http.StatusBadRequest, "validation failed")
		return
	}

	user, err := h.Users.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.Login(c.Request.Context(), user, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	c.JSON(http.StatusOK, tokenResponse(access, accessExp))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(h.Cfg.CookieNameRefresh)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "missing refresh token")
		return
	}

	access, accessExp, refresh, refreshExp, err := h.Auth.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	setRefreshCookie(c, h.Cfg, refresh, refreshExp)
	c.JSON(http.StatusOK, tokenResponse(access, accessExp))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie(h.Cfg.CookieNameRefresh)
	if err != nil {
		respondError(c, http.StatusUnauthorized, "missing refresh token")
		return
	}

	_ = h.Auth.Logout(c.Request.Context(), refreshToken)
	clearRefreshCookie(c, h.Cfg)
	c.Status(http.StatusNoContent)
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
		Path:     "/",
		Domain:   cfg.CookieDomain,
		Expires:  exp,
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
		Path:     "/",
		Domain:   cfg.CookieDomain,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   cfg.CookieSecure,
		SameSite: parseSameSite(cfg.CookieSameSite),
	}
	http.SetCookie(c.Writer, cookie)
}

func parseSameSite(input string) http.SameSite {
	switch input {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
