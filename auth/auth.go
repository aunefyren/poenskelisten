package auth

import (
	"aunefyren/poenskelisten/config"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenValidDuration is how long a freshly issued (or refreshed) session token
// remains valid.
const TokenValidDuration = 7 * 24 * time.Hour

// MFAChallengeValidDuration is how long the short-lived token issued between a
// correct password and TOTP entry remains valid.
const MFAChallengeValidDuration = 5 * time.Minute

// AccessTokenValidDuration is how long a short-lived access token is valid. It is
// deliberately short because access tokens are validated statelessly (no DB hit),
// so revocation only takes effect once the current access token expires.
const AccessTokenValidDuration = 15 * time.Minute

// Token purposes. An empty Purpose denotes a legacy session token (issued before
// access/refresh tokens existed); it is still accepted as an access token so
// upgrades don't force everyone to log in again. PurposeAccess marks a modern
// short-lived access token. Other purposes are single-use and rejected by the
// session-validation path.
const (
	PurposeMFAChallenge = "mfa_challenge"
	PurposeAccess       = "access"
)

// ErrNotAdmin is returned by ValidateToken when an admin session is required but
// the token does not carry the admin claim. Callers use errors.Is to map this to
// an authorization (403) rather than an authentication (401) failure.
var ErrNotAdmin = errors.New("token not an admin session")

type JWTClaim struct {
	Firstname string    `json:"first_name"`
	Lastname  string    `json:"last_name"`
	Email     string    `json:"email"`
	Admin     bool      `json:"admin"`
	Verified  bool      `json:"verified"`
	UserID    uuid.UUID `json:"id"`
	// Purpose distinguishes single-purpose tokens (e.g. an MFA challenge) from a
	// normal session token. Empty means a normal session token.
	Purpose string `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

func GenerateJWT(firstname string, lastname string, email string, userid uuid.UUID, admin bool, verified bool) (tokenString string, err error) {
	now := time.Now()
	claims := &JWTClaim{
		Firstname: firstname,
		Lastname:  lastname,
		Email:     email,
		Admin:     admin,
		UserID:    userid,
		Verified:  verified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenValidDuration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    config.ConfigFile.PoenskelistenName,
		},
	}
	return GenerateJWTFromClaims(claims)
}

// GenerateAccessJWT issues a short-lived access token (Purpose "access"). Access
// tokens are what the API middleware validates on every request.
func GenerateAccessJWT(firstname string, lastname string, email string, userid uuid.UUID, admin bool, verified bool) (tokenString string, err error) {
	now := time.Now()
	claims := &JWTClaim{
		Firstname: firstname,
		Lastname:  lastname,
		Email:     email,
		Admin:     admin,
		UserID:    userid,
		Verified:  verified,
		Purpose:   PurposeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenValidDuration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    config.ConfigFile.PoenskelistenName,
		},
	}
	return GenerateJWTFromClaims(claims)
}

func GenerateJWTFromClaims(claims *JWTClaim) (tokenString string, err error) {
	jwtKey, err := config.GetPrivateKey()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(jwtKey)
	return
}

func ValidateToken(signedToken string, admin bool) (err error) {
	_, err = ValidateTokenGetClaims(signedToken, admin)
	return err
}

// ValidateTokenGetClaims validates the token and returns its claims, so callers
// that also need the claims don't have to parse the token a second time.
func ValidateTokenGetClaims(signedToken string, admin bool) (*JWTClaim, error) {
	claims, err := ParseToken(signedToken)
	if err != nil {
		return nil, err
	}
	if claims.ExpiresAt == nil || claims.NotBefore == nil {
		return nil, errors.New("claims not present")
	}
	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		return nil, errors.New("token has expired")
	}
	if claims.NotBefore.Time.After(now) {
		return nil, errors.New("token has not begun")
	}
	// Only access tokens (or legacy empty-purpose session tokens) may authenticate
	// API requests. Single-purpose tokens (e.g. an MFA challenge) are rejected.
	if claims.Purpose != "" && claims.Purpose != PurposeAccess {
		return nil, errors.New("token is not a session token")
	}
	if admin && !claims.Admin {
		return nil, ErrNotAdmin
	}
	return claims, nil
}

// GenerateMFAChallengeToken mints a short-lived token that stands in for a
// successful password check while the user completes the TOTP step. It carries
// only the user ID and a purpose marker; it cannot be used as a session token.
func GenerateMFAChallengeToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := &JWTClaim{
		UserID:  userID,
		Purpose: PurposeMFAChallenge,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(MFAChallengeValidDuration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    config.ConfigFile.PoenskelistenName,
		},
	}
	return GenerateJWTFromClaims(claims)
}

// ValidateMFAChallengeToken parses and validates an MFA challenge token, returning
// the user ID it was issued for. It rejects expired, not-yet-valid, and
// wrong-purpose tokens.
func ValidateMFAChallengeToken(signedToken string) (uuid.UUID, error) {
	claims, err := ParseToken(signedToken)
	if err != nil {
		return uuid.UUID{}, err
	}
	if claims.ExpiresAt == nil || claims.NotBefore == nil {
		return uuid.UUID{}, errors.New("claims not present")
	}
	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		return uuid.UUID{}, errors.New("challenge token has expired")
	}
	if claims.NotBefore.Time.After(now) {
		return uuid.UUID{}, errors.New("challenge token has not begun")
	}
	if claims.Purpose != PurposeMFAChallenge {
		return uuid.UUID{}, errors.New("token is not an MFA challenge token")
	}
	return claims.UserID, nil
}

func ParseToken(signedToken string) (*JWTClaim, error) {
	jwtKey, err := config.GetPrivateKey()
	if err != nil {
		return nil, err
	}

	// Tolerate an optional "Bearer " scheme prefix so callers can use either the
	// raw token or the standard Authorization header form.
	if len(signedToken) >= 7 && strings.EqualFold(signedToken[:7], "bearer ") {
		signedToken = signedToken[7:]
	}

	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			// Only HMAC signing is expected; reject any other algorithm to
			// guard against algorithm-confusion attacks.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtKey, nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		return nil, errors.New("couldn't parse claims")
	}
	return claims, nil
}
