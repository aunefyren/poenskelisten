package utilities

import (
	"aunefyren/poenskelisten/config"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// deriveEncryptionKey derives a fixed 32-byte AES-256 key from the configured
// server private key.
//
// The input is NOT a user password: config.PrivateKey is a 512-bit
// cryptographically-random key (config.GenerateSecureKey). Deriving a key from
// such a high-entropy secret is exactly what HKDF (RFC 5869) is for — a slow
// password hash (bcrypt/scrypt/argon2) is unnecessary and inappropriate for
// random key material. HKDF also gives us domain separation (the info label), so
// this encryption key is distinct from the JWT signing key.
func deriveEncryptionKey() ([]byte, error) {
	masterKey, err := config.GetPrivateKey()
	if err != nil {
		return nil, err
	}

	key := make([]byte, 32)
	reader := hkdf.New(sha256.New, masterKey, nil, []byte("poenskelisten/totp-secret-encryption"))
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptString encrypts plaintext with AES-256-GCM using a key derived from the
// server private key, returning a base64 string of nonce||ciphertext. Used for
// data that must be recoverable at rest, such as TOTP secrets.
func EncryptString(plaintext string) (string, error) {
	key, err := deriveEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal appends the ciphertext to nonce, so the returned blob is
	// nonce || ciphertext || tag.
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptString reverses EncryptString.
func DecryptString(encoded string) (string, error) {
	key, err := deriveEncryptionKey()
	if err != nil {
		return "", err
	}

	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
