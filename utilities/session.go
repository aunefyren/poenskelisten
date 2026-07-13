package utilities

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateOpaqueToken returns a high-entropy, URL-safe random token suitable for
// use as a refresh token. It carries 256 bits of entropy, so it needs no slow
// KDF — a plain SHA-256 is enough to store it safely (see HashOpaqueToken).
func GenerateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashOpaqueToken returns the hex-encoded SHA-256 of a token, for storage and
// constant-time-free lookup. Only the hash is persisted, so a database leak
// cannot be replayed as a valid token.
func HashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
