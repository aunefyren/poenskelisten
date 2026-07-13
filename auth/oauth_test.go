package auth

import (
	"aunefyren/poenskelisten/config"
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
