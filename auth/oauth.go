package auth

import (
	"aunefyren/poenskelisten/config"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"sync"

	"github.com/go-jose/go-jose/v4"
)

// OAuth token signing uses an asymmetric key (see config.GenerateOAuthSigningKey)
// so external parties — MCP clients and resource servers — can verify tokens via
// the published JWKS without sharing a secret. This is separate from the
// first-party session cookie, which stays on HS256.

var (
	oauthKeyMutex sync.Mutex
	oauthSigner   crypto.Signer
	oauthKeyCache string // the PEM the cached signer was parsed from
	oauthKID      string
)

// loadOAuthSigner parses (and caches) the configured OAuth signing key. The cache
// is invalidated when the configured PEM changes.
func loadOAuthSigner() (crypto.Signer, string, error) {
	keyPEM := config.ConfigFile.OAuthSigningKey
	if keyPEM == "" {
		return nil, "", errors.New("OAuth signing key is not configured")
	}

	oauthKeyMutex.Lock()
	defer oauthKeyMutex.Unlock()

	if oauthSigner != nil && oauthKeyCache == keyPEM {
		return oauthSigner, oauthKID, nil
	}

	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, "", errors.New("failed to decode OAuth signing key PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, "", err
	}
	signer, ok := parsed.(crypto.Signer)
	if !ok {
		return nil, "", errors.New("OAuth signing key is not a usable signer")
	}

	oauthSigner = signer
	oauthKeyCache = keyPEM
	oauthKID = config.ConfigFile.OAuthSigningKeyID
	return signer, oauthKID, nil
}

// OAuthPublicJWKS returns the public JSON Web Key Set used to verify OAuth tokens.
func OAuthPublicJWKS() (jose.JSONWebKeySet, error) {
	signer, kid, err := loadOAuthSigner()
	if err != nil {
		return jose.JSONWebKeySet{}, err
	}

	jwk := jose.JSONWebKey{
		Key:       signer.Public(),
		KeyID:     kid,
		Algorithm: config.OAuthSigningAlgorithm,
		Use:       "sig",
	}

	return jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}, nil
}
