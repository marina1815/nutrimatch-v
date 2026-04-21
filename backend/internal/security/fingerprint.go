package security

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

func HashFingerprint(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(normalized))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}
