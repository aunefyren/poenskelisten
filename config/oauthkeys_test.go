package config

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func parsePrivateKey(t *testing.T, keyPEM string) interface{} {
	t.Helper()
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse PKCS#8 key: %v", err)
	}
	return key
}

func TestGenerateOAuthSigningKeyES256(t *testing.T) {
	keyPEM, kid, err := GenerateOAuthSigningKey(OAuthAlgES256)
	if err != nil {
		t.Fatalf("GenerateOAuthSigningKey error: %v", err)
	}
	if kid == "" {
		t.Error("expected a non-empty key ID")
	}
	if _, ok := parsePrivateKey(t, keyPEM).(*ecdsa.PrivateKey); !ok {
		t.Error("ES256 key is not an ECDSA private key")
	}
}

func TestGenerateOAuthSigningKeyRS256(t *testing.T) {
	keyPEM, kid, err := GenerateOAuthSigningKey(OAuthAlgRS256)
	if err != nil {
		t.Fatalf("GenerateOAuthSigningKey error: %v", err)
	}
	if kid == "" {
		t.Error("expected a non-empty key ID")
	}
	if _, ok := parsePrivateKey(t, keyPEM).(*rsa.PrivateKey); !ok {
		t.Error("RS256 key is not an RSA private key")
	}
}

func TestGenerateOAuthSigningKeyDefaultsToES256(t *testing.T) {
	keyPEM, _, err := GenerateOAuthSigningKey("")
	if err != nil {
		t.Fatalf("GenerateOAuthSigningKey error: %v", err)
	}
	if _, ok := parsePrivateKey(t, keyPEM).(*ecdsa.PrivateKey); !ok {
		t.Error("empty algorithm should default to ECDSA (ES256)")
	}
}

func TestGenerateOAuthSigningKeyUnsupported(t *testing.T) {
	if _, _, err := GenerateOAuthSigningKey("HS256"); err == nil {
		t.Error("expected an error for an unsupported algorithm")
	}
}

func TestGenerateOAuthSigningKeyDistinctKIDs(t *testing.T) {
	_, kid1, _ := GenerateOAuthSigningKey(OAuthAlgES256)
	_, kid2, _ := GenerateOAuthSigningKey(OAuthAlgES256)
	if kid1 == kid2 {
		t.Error("distinct keys should yield distinct key IDs")
	}
}
