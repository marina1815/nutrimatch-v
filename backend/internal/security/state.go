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

type StateManager struct {
	Secret []byte
	TTL    time.Duration
}

type OIDCState struct {
	State        string `json:"state"`
	Nonce        string `json:"nonce"`
	Verifier     string `json:"verifier"`
	RedirectPath string `json:"redirectPath"`
	ExpiresAt    int64  `json:"expiresAt"`
}

func (m *StateManager) Issue(redirectPath string) (*OIDCState, string, error) {
	state, err := randomString(24)
	if err != nil {
		return nil, "", err
	}
	nonce, err := randomString(24)
	if err != nil {
		return nil, "", err
	}
	verifier, err := randomString(48)
	if err != nil {
		return nil, "", err
	}

	payload := &OIDCState{
		State:        state,
		Nonce:        nonce,
		Verifier:     verifier,
		RedirectPath: redirectPath,
		ExpiresAt:    time.Now().Add(m.TTL).Unix(),
	}
	raw, err := m.sign(payload)
	if err != nil {
		return nil, "", err
	}
	return payload, raw, nil
}

func (m *StateManager) Parse(raw string) (*OIDCState, error) {
	parts := splitToken(raw)
	if len(parts) != 2 {
		return nil, errors.New("invalid state token")
	}

	mac := hmac.New(sha256.New, m.Secret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("invalid state signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	var payload OIDCState
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if time.Now().Unix() > payload.ExpiresAt {
		return nil, errors.New("state expired")
	}
	return &payload, nil
}

func (m *StateManager) sign(payload *OIDCState) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, m.Secret)
	mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func randomString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
