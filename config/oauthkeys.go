package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// Supported OAuth token signing algorithms. OAuth access/ID tokens are signed
// with an asymmetric key so external parties can verify them via the JWKS; the
// first-party web session cookie keeps using the symmetric PrivateKey.
const (
	OAuthAlgES256 = "ES256"
	OAuthAlgRS256 = "RS256"
)

// OAuthSigningAlgorithm is the fixed algorithm used to sign OAuth tokens.
const OAuthSigningAlgorithm = OAuthAlgES256

// OAuthIssuer returns the OAuth issuer URL. It is derived from the external URL
// (falling back to a localhost URL for local development), not persisted — the
// issuer, and the resource identifiers below, are always computed from config.
func OAuthIssuer() string {
	if strings.TrimSpace(ConfigFile.PoenskelistenExternalURL) != "" {
		return strings.TrimRight(ConfigFile.PoenskelistenExternalURL, "/")
	}
	return fmt.Sprintf("http://localhost:%d", ConfigFile.PoenskelistenPort)
}

// APIResource is the audience identifier for API access tokens.
func APIResource() string {
	return OAuthIssuer() + "/api"
}

// MCPResource is the audience identifier for MCP access tokens.
func MCPResource() string {
	return OAuthIssuer() + "/mcp"
}

// GenerateOAuthSigningKey creates a new signing keypair for the given algorithm
// and returns the PEM-encoded PKCS#8 private key plus a stable key ID (derived
// from the public key, so it survives restarts as long as the key does).
func GenerateOAuthSigningKey(alg string) (privateKeyPEM string, keyID string, err error) {
	var privDER, pubDER []byte

	switch strings.ToUpper(strings.TrimSpace(alg)) {
	case OAuthAlgRS256:
		key, genErr := rsa.GenerateKey(rand.Reader, 2048)
		if genErr != nil {
			return "", "", genErr
		}
		if privDER, err = x509.MarshalPKCS8PrivateKey(key); err != nil {
			return "", "", err
		}
		if pubDER, err = x509.MarshalPKIXPublicKey(&key.PublicKey); err != nil {
			return "", "", err
		}
	case OAuthAlgES256, "":
		key, genErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if genErr != nil {
			return "", "", genErr
		}
		if privDER, err = x509.MarshalPKCS8PrivateKey(key); err != nil {
			return "", "", err
		}
		if pubDER, err = x509.MarshalPKIXPublicKey(&key.PublicKey); err != nil {
			return "", "", err
		}
	default:
		return "", "", errors.New("unsupported OAuth signing algorithm: " + alg)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

	// Key ID: the first 16 bytes of SHA-256 over the public key, base64url. Stable
	// for a given key, distinct across keys, and safe to expose in the JWKS.
	sum := sha256.Sum256(pubDER)
	keyID = base64.RawURLEncoding.EncodeToString(sum[:16])

	return string(pemBytes), keyID, nil
}
