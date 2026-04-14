package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseAccessTokenRejectsWrongAudience(t *testing.T) {
	manager := &TokenManager{
		Secret:    []byte("0123456789abcdef0123456789abcdef"),
		Issuer:    "nutrimatch",
		Audience:  "nutrimatch_users",
		AccessTTL: 15 * time.Minute,
	}

	now := time.Now()
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			Issuer:    manager.Issuer,
			Audience:  []string{"another-audience"},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		SessionID: "session-1",
	}).SignedString(manager.Secret)
	if err != nil {
		t.Fatalf("unexpected sign error: %v", err)
	}

	if _, err := manager.ParseAccessToken(raw); err == nil {
		t.Fatalf("expected parse failure for wrong audience")
	}
}
