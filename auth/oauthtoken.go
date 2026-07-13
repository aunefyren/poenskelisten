package auth

import (
	"aunefyren/poenskelisten/config"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// OAuthAccessTokenDuration is the lifetime of an OAuth access token. Short,
// because access tokens are validated statelessly (revocation bites at refresh).
const OAuthAccessTokenDuration = 15 * time.Minute

// IDTokenDuration is the lifetime of an OIDC ID token.
const IDTokenDuration = 15 * time.Minute

// PurposeSSO marks the first-party login-state (SSO) token — an HS256 cookie read
// only by /oauth/authorize.
const PurposeSSO = "sso"

// SSOTokenValidDuration is how long a browser stays "logged in" at the AS before
// re-authenticating at /oauth/authorize.
const SSOTokenValidDuration = 12 * time.Hour

// OAuthClaims are the claims carried by an ES256 OAuth access token.
type OAuthClaims struct {
	Scope    string `json:"scope"`
	Admin    bool   `json:"admin"`
	Verified bool   `json:"verified"`
	jwt.RegisteredClaims
}

// IDTokenClaims are the OIDC ID-token claims.
type IDTokenClaims struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
	jwt.RegisteredClaims
}

// oauthSigningMethod maps the configured algorithm to a JWT signing method.
func oauthSigningMethod() jwt.SigningMethod {
	if strings.EqualFold(config.OAuthSigningAlgorithm, config.OAuthAlgRS256) {
		return jwt.SigningMethodRS256
	}
	return jwt.SigningMethodES256
}

// GenerateOAuthAccessToken mints an ES256 (or RS256) access token bound to an
// audience (the target resource identifier) and scope.
func GenerateOAuthAccessToken(userID uuid.UUID, audience string, scope string, admin bool, verified bool) (string, error) {
	signer, kid, err := loadOAuthSigner()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := &OAuthClaims{
		Scope:    scope,
		Admin:    admin,
		Verified: verified,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{audience},
			Issuer:    config.OAuthIssuer(),
			ExpiresAt: jwt.NewNumericDate(now.Add(OAuthAccessTokenDuration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(oauthSigningMethod(), claims)
	token.Header["kid"] = kid
	return token.SignedString(signer)
}

// GenerateIDToken mints an OIDC ID token (audience = client_id).
func GenerateIDToken(userID uuid.UUID, clientID string, email string, name string) (string, error) {
	signer, kid, err := loadOAuthSigner()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := &IDTokenClaims{
		Email: email,
		Name:  name,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{clientID},
			Issuer:    config.OAuthIssuer(),
			ExpiresAt: jwt.NewNumericDate(now.Add(IDTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(oauthSigningMethod(), claims)
	token.Header["kid"] = kid
	return token.SignedString(signer)
}

// ValidateOAuthAccessToken verifies an access token's signature (via the local
// public key), issuer, audience, and expiry, returning its claims. This is the
// resource-server check used by both the API and the MCP endpoint (differing only
// in expectedAudience).
func ValidateOAuthAccessToken(signedToken string, expectedAudience string) (*OAuthClaims, error) {
	signer, _, err := loadOAuthSigner()
	if err != nil {
		return nil, err
	}
	publicKey := signer.Public()

	// Tolerate an optional "Bearer " scheme prefix.
	if len(signedToken) >= 7 && strings.EqualFold(signedToken[:7], "bearer ") {
		signedToken = signedToken[7:]
	}

	token, err := jwt.ParseWithClaims(
		signedToken,
		&OAuthClaims{},
		func(*jwt.Token) (interface{}, error) { return publicKey, nil },
		jwt.WithValidMethods([]string{oauthSigningMethod().Alg()}),
		jwt.WithIssuer(config.OAuthIssuer()),
		jwt.WithAudience(expectedAudience),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*OAuthClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid access token")
	}
	return claims, nil
}

// HasScope reports whether the token's space-delimited scope contains name.
func (c *OAuthClaims) HasScope(name string) bool {
	for _, s := range strings.Fields(c.Scope) {
		if s == name {
			return true
		}
	}
	return false
}

// GenerateSSOToken mints the HS256 login-state token set after a successful
// browser login and read by /oauth/authorize.
func GenerateSSOToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := &JWTClaim{
		UserID:  userID,
		Purpose: PurposeSSO,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(SSOTokenValidDuration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    config.ConfigFile.PoenskelistenName,
		},
	}
	return GenerateJWTFromClaims(claims)
}

// ValidateSSOToken validates an SSO token's signature, expiry, and purpose. The
// caller additionally checks the user's SessionsInvalidatedAt against IssuedAt for
// global logout (that needs a DB lookup, so it lives in the controller).
func ValidateSSOToken(signedToken string) (*JWTClaim, error) {
	claims, err := ParseToken(signedToken)
	if err != nil {
		return nil, err
	}
	if claims.ExpiresAt == nil || claims.NotBefore == nil {
		return nil, errors.New("claims not present")
	}
	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		return nil, errors.New("sso token has expired")
	}
	if claims.NotBefore.Time.After(now) {
		return nil, errors.New("sso token has not begun")
	}
	if claims.Purpose != PurposeSSO {
		return nil, errors.New("token is not an sso token")
	}
	return claims, nil
}

// VerifyPKCE checks a PKCE code_verifier against a stored S256 challenge.
func VerifyPKCE(verifier string, challenge string) bool {
	if verifier == "" || challenge == "" {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}
