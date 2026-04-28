package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

type BlindIndexer struct {
	key []byte
}

func NewBlindIndexer(secret string) *BlindIndexer {
	if strings.TrimSpace(secret) == "" {
		return nil
	}
	return &BlindIndexer{key: []byte(secret)}
}

func (i *BlindIndexer) Index(scope, value string) string {
	if i == nil || len(i.key) == 0 {
		return ""
	}
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	mac := hmac.New(sha256.New, i.key)
	mac.Write([]byte(scope))
	mac.Write([]byte{0})
	mac.Write([]byte(normalized))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
