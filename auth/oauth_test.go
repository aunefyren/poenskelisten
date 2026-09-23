package auth

import (
	"aunefyren/poenskelisten/config"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

// setupOAuthKey generates and installs an OAuth signing key into the
// package-global config (always the fixed ES256 algorithm).
func setupOAuthKey(t *testing.T) {
	t.Helper()
	keyPEM, kid, err := config.GenerateOAuthSigningKey(config.OAuthSigningAlgorithm)
	if err != nil {
		t.Fatalf("failed to generate OAuth key: %v", err)
	}
	config.ConfigFile.OAuthSigningKey = keyPEM
	config.ConfigFile.OAuthSigningKeyID = kid
}

func TestOAuthPublicJWKS(t *testing.T) {
	setupOAuthKey(t)

	jwks, err := OAuthPublicJWKS()
	if err != nil {
		t.Fatalf("OAuthPublicJWKS error: %v", err)
	}
	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}
	key := jwks.Keys[0]
	if !key.IsPublic() {
		t.Error("JWKS must expose the public key only")
	}
	if key.KeyID != config.ConfigFile.OAuthSigningKeyID {
		t.Errorf("KeyID = %q, want %q", key.KeyID, config.ConfigFile.OAuthSigningKeyID)
	}
	if key.Algorithm != config.OAuthSigningAlgorithm {
		t.Errorf("Algorithm = %q, want %q", key.Algorithm, config.OAuthSigningAlgorithm)
	}
}

func TestOAuthPublicJWKSNoKey(t *testing.T) {
	config.ConfigFile.OAuthSigningKey = ""
	if _, err := OAuthPublicJWKS(); err == nil {
		t.Error("expected an error when no signing key is configured")
	}
}

// TestLoadOAuthSignerRejectsUnusableKeys covers each way a configured key can
// be unusable: not PEM at all, PEM that isn't PKCS#8, and a PKCS#8 key type
// that can't sign (X25519 is key-agreement only).
func TestLoadOAuthSignerRejectsUnusableKeys(t *testing.T) {
	origKey := config.ConfigFile.OAuthSigningKey
	t.Cleanup(func() { config.ConfigFile.OAuthSigningKey = origKey })

	x25519Key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate X25519 key: %v", err)
	}
	x25519DER, err := x509.MarshalPKCS8PrivateKey(x25519Key)
	if err != nil {
		t.Fatalf("failed to marshal X25519 key: %v", err)
	}

	cases := []struct {
		name    string
		keyPEM  string
		wantErr string
	}{
		{"not PEM", "definitely not a PEM block", "failed to decode"},
		{"not PKCS8", string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("garbage")})), ""},
		{"not a signer", string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x25519DER})), "not a usable signer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config.ConfigFile.OAuthSigningKey = c.keyPEM
			signer, kid, err := loadOAuthSigner()
			if err == nil || signer != nil || kid != "" {
				t.Fatalf("loadOAuthSigner = (%v, %q, %v), want nil signer and an error", signer, kid, err)
			}
			if c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, c.wantErr)
			}
		})
	}
}
