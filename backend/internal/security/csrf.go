package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

type CSRFManager struct {
	Secret []byte
	TTL    time.Duration
}

type csrfPayload struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
}

func (m *CSRFManager) IssueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	payload := csrfPayload{
		Token:     base64.RawURLEncoding.EncodeToString(buf),
		ExpiresAt: time.Now().Add(m.TTL).Unix(),
	}
	return m.signPayload(payload)
}

func (m *CSRFManager) ValidateToken(raw string) error {
	payload, err := m.parsePayload(raw)
	if err != nil {
		return err
	}
	if time.Now().Unix() > payload.ExpiresAt {
		return errors.New("csrf token expired")
	}
	return nil
}

func (m *CSRFManager) signPayload(payload csrfPayload) (string, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(buf)
	mac := hmac.New(sha256.New, m.Secret)
	mac.Write([]byte(encoded))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + signature, nil
}

func (m *CSRFManager) parsePayload(raw string) (*csrfPayload, error) {
	parts := splitToken(raw)
	if len(parts) != 2 {
		return nil, errors.New("invalid csrf token")
	}

	mac := hmac.New(sha256.New, m.Secret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("invalid csrf signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	var payload csrfPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func splitToken(raw string) []string {
	for i := 0; i < len(raw); i++ {
		if raw[i] == '.' {
			return []string{raw[:i], raw[i+1:]}
		}
	}
	return nil
}
