package utilities

import (
	"aunefyren/poenskelisten/config"
	"testing"
)

// setupCryptoTestKey points the package-global config at a valid base64 signing
// key so the encryption-key derivation works without a config.json on disk.
func setupCryptoTestKey(t *testing.T) {
	t.Helper()

	key, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	config.ConfigFile.PrivateKey = key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	setupCryptoTestKey(t)

	for _, plaintext := range []string{"", "JBSWY3DPEHPK3PXP", "a longer secret with spaces & symbols !@#"} {
		encrypted, err := EncryptString(plaintext)
		if err != nil {
			t.Fatalf("EncryptString(%q) error: %v", plaintext, err)
		}
		if encrypted == plaintext && plaintext != "" {
			t.Errorf("EncryptString(%q) returned plaintext unchanged", plaintext)
		}

		decrypted, err := DecryptString(encrypted)
		if err != nil {
			t.Fatalf("DecryptString error: %v", err)
		}
		if decrypted != plaintext {
			t.Errorf("round trip = %q, want %q", decrypted, plaintext)
		}
	}
}

func TestEncryptProducesDistinctCiphertexts(t *testing.T) {
	setupCryptoTestKey(t)

	// A random nonce means encrypting the same plaintext twice yields different
	// ciphertexts, both of which decrypt correctly.
	a, err := EncryptString("same-secret")
	if err != nil {
		t.Fatalf("EncryptString error: %v", err)
	}
	b, err := EncryptString("same-secret")
	if err != nil {
		t.Fatalf("EncryptString error: %v", err)
	}
	if a == b {
		t.Error("expected distinct ciphertexts for repeated encryption, got identical output")
	}
}

func TestDecryptWithRotatedKeyFails(t *testing.T) {
	setupCryptoTestKey(t)

	encrypted, err := EncryptString("secret")
	if err != nil {
		t.Fatalf("EncryptString error: %v", err)
	}

	// Rotate the private key: the previously encrypted value must no longer decrypt.
	newKey, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate replacement key: %v", err)
	}
	config.ConfigFile.PrivateKey = newKey

	if _, err := DecryptString(encrypted); err == nil {
		t.Error("DecryptString succeeded after key rotation, want error")
	}
}

func TestDecryptGarbageFails(t *testing.T) {
	setupCryptoTestKey(t)

	for _, garbage := range []string{"", "not-base64!!!", "YWJj"} {
		if _, err := DecryptString(garbage); err == nil {
			t.Errorf("DecryptString(%q) succeeded, want error", garbage)
		}
	}
}
