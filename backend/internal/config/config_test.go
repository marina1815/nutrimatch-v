package config

import (
	"testing"
	"time"
)

func validConfig() *Config {
	return &Config{
		AppEnv:                    "development",
		DBURL:                     "postgres://user:password@localhost:5432/nutrimatch?sslmode=disable",
		BodyLimitBytes:            1024 * 1024,
		JWTSecret:                 "abcdefghijklmnopqrstuvwxyz123456",
		JWTIssuer:                 "nutrimatch",
		JWTAudience:               "users",
		AccessTokenTTL:            15 * time.Minute,
		RefreshTokenTTL:           24 * time.Hour,
		SessionIdleTTL:            12 * time.Hour,
		AuthFailureWindow:         15 * time.Minute,
		AuthMaxFailures:           5,
		RefreshTokenPepper:        "1234567890abcdefghijklmnopqrstuvwxyz",
		HealthDataKey:             "12345678901234567890123456789012",
		Argon2Time:                2,
		Argon2Memory:              65536,
		Argon2Threads:             1,
		Argon2KeyLength:           32,
		Argon2SaltLength:          16,
		CookieNameRefresh:         "nm_refresh",
		CookieNameCSRF:            "nm_csrf",
		CookiePathRefresh:         "/api/v1/auth",
		CookiePathCSRF:            "/api/v1",
		CookieSameSite:            "Lax",
		CookieSecure:              false,
		CSRFHeaderName:            "X-CSRF-Token",
		CSRFTTL:                   30 * time.Minute,
		TrustedOrigins:            []string{"http://localhost:3000"},
		FrontendBaseURL:           "http://localhost:3000",
		SpoonacularBaseURL:        "https://api.spoonacular.com",
		SpoonacularSearchCacheTTL: 15 * time.Minute,
		GoogleAIBaseURL:           "https://generativelanguage.googleapis.com",
	}
}

func TestValidateRejectsWeakSecrets(t *testing.T) {
	cfg := validConfig()
	cfg.AppEnv = "production"
	cfg.JWTSecret = "short"
	cfg.RefreshTokenPepper = "short"
	cfg.HealthDataKey = "too-short"
	cfg.Argon2Time = 1
	cfg.Argon2Memory = 1024
	cfg.Argon2Threads = 0
	cfg.Argon2KeyLength = 16
	cfg.Argon2SaltLength = 8
	cfg.CookieSameSite = "None"
	cfg.CookieSecure = false
	cfg.SpoonacularBaseURL = "http://api.spoonacular.com"
	cfg.GoogleAIBaseURL = "http://generativelanguage.googleapis.com"

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation to fail for weak configuration")
	}
}

func TestValidateRejectsHealthDataKeyLengthMismatch(t *testing.T) {
	cfg := validConfig()
	cfg.HealthDataKey = "1234567890123456789012345678901"

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation to fail when health key is shorter than 32 bytes")
	}

	cfg = validConfig()
	cfg.HealthDataKey = "123456789012345678901234567890123"

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation to fail when health key is longer than 32 bytes")
	}
}
