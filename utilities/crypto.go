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
)

// deriveEncryptionKey turns the configured JWT private key into a fixed 32-byte
// AES-256 key. We hash rather than use the raw bytes so the key length is always
// correct regardless of how the private key was generated, and so the encryption
// key isn't byte-identical to the signing secret.
func deriveEncryptionKey() ([]byte, error) {
	privateKey, err := config.GetPrivateKey()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(privateKey)
	return sum[:], nil
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
