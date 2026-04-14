package config

import (
	"testing"
	"time"
)

func TestValidateRejectsWeakSecrets(t *testing.T) {
	cfg := &Config{
		AppEnv:             "production",
		DBURL:              "postgres://user:password@localhost:5432/nutrimatch?sslmode=disable",
		BodyLimitBytes:     1024,
		JWTSecret:          "short",
		JWTIssuer:          "nutrimatch",
		JWTAudience:        "users",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		RefreshTokenPepper: "short",
		Argon2Time:         1,
		Argon2Memory:       1024,
		Argon2Threads:      0,
		Argon2KeyLength:    16,
		Argon2SaltLength:   8,
		CookieNameRefresh:  "nm_refresh",
		CookiePathRefresh:  "/api/v1/auth",
		CookieSameSite:     "None",
		CookieSecure:       false,
		TrustedOrigins:     []string{"http://localhost:3000"},
		SpoonacularBaseURL: "http://api.spoonacular.com",
		GoogleAIBaseURL:    "http://generativelanguage.googleapis.com",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation to fail for weak configuration")
	}
}
