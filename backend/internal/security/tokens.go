package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	Secret       []byte
	Issuer       string
	Audience     string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	TokenPepper  []byte
}

type AccessClaims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
}

func (t *TokenManager) NewAccessToken(userID, sessionID string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(t.AccessTTL)

	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    t.Issuer,
			Audience:  []string{t.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		SessionID: sessionID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.Secret)
	return signed, exp, err
}

func (t *TokenManager) ParseAccessToken(raw string) (*AccessClaims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		return t.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := parsed.Claims.(*AccessClaims); ok && parsed.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

func (t *TokenManager) NewRefreshToken() (string, time.Time, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	exp := time.Now().Add(t.RefreshTTL)
	return token, exp, nil
}

func (t *TokenManager) HashRefreshToken(token string) string {
	mac := hmac.New(sha256.New, t.TokenPepper)
	mac.Write([]byte(token))
	sum := mac.Sum(nil)
	return base64.RawStdEncoding.EncodeToString(sum)
}
