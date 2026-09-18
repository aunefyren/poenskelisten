package auth

import (
	"aunefyren/poenskelisten/config"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func setupOAuthTokenTest(t *testing.T) {
	t.Helper()
	setupOAuthKey(t)
	config.ConfigFile.PoenskelistenExternalURL = "https://iss.example.com"
}

func TestOAuthAccessTokenRoundTrip(t *testing.T) {
	setupOAuthTokenTest(t)
	userID := uuid.New()

	token, err := GenerateOAuthAccessToken(userID, "https://iss.example.com/api", "openid email", true, true)
	if err != nil {
		t.Fatalf("GenerateOAuthAccessToken error: %v", err)
	}

	claims, err := ValidateOAuthAccessToken(token, "https://iss.example.com/api")
	if err != nil {
		t.Fatalf("ValidateOAuthAccessToken error: %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("sub = %q, want %q", claims.Subject, userID.String())
	}
	if !claims.Admin {
		t.Error("admin claim lost")
	}
	if !claims.HasScope("email") || claims.HasScope("profile") {
		t.Errorf("scope handling wrong: %q", claims.Scope)
	}
}

func TestOAuthAccessTokenWrongAudience(t *testing.T) {
	setupOAuthTokenTest(t)

	token, err := GenerateOAuthAccessToken(uuid.New(), "https://iss.example.com/api", "openid", false, true)
	if err != nil {
		t.Fatalf("GenerateOAuthAccessToken error: %v", err)
	}

	// A token minted for the API must not validate for the MCP resource.
	if _, err := ValidateOAuthAccessToken(token, "https://iss.example.com/mcp"); err == nil {
		t.Error("access token validated for the wrong audience, want error")
	}
}

func TestOAuthAccessTokenBearerPrefix(t *testing.T) {
	setupOAuthTokenTest(t)

	token, err := GenerateOAuthAccessToken(uuid.New(), "https://iss.example.com/api", "openid", false, true)
	if err != nil {
		t.Fatalf("GenerateOAuthAccessToken error: %v", err)
	}
	if _, err := ValidateOAuthAccessToken("Bearer "+token, "https://iss.example.com/api"); err != nil {
		t.Errorf("ValidateOAuthAccessToken rejected a Bearer-prefixed token: %v", err)
	}
}

func TestSSOTokenRoundTripAndRejectedAsSession(t *testing.T) {
	setupAuthTestConfig(t)
	userID := uuid.New()

	token, err := GenerateSSOToken(userID)
	if err != nil {
		t.Fatalf("GenerateSSOToken error: %v", err)
	}

	claims, err := ValidateSSOToken(token)
	if err != nil {
		t.Fatalf("ValidateSSOToken error: %v", err)
	}
	if claims.UserID != userID || claims.Purpose != PurposeSSO {
		t.Errorf("sso claims wrong: %+v", claims)
	}

	// An SSO token must never authenticate an API request.
	if err := ValidateToken(token, false); err == nil {
		t.Error("ValidateToken accepted an SSO token as a session, want error")
	}
	// A normal session token is not a valid SSO token.
	sessionToken, _ := GenerateJWT("A", "B", "a@b.c", uuid.New(), false, true)
	if _, err := ValidateSSOToken(sessionToken); err == nil {
		t.Error("ValidateSSOToken accepted a session token, want error")
	}
}

func TestVerifyPKCE(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	if !VerifyPKCE(verifier, challenge) {
		t.Error("VerifyPKCE rejected a valid verifier/challenge pair")
	}
	if VerifyPKCE("wrong-verifier", challenge) {
		t.Error("VerifyPKCE accepted a wrong verifier")
	}
	if VerifyPKCE("", challenge) || VerifyPKCE(verifier, "") {
		t.Error("VerifyPKCE accepted an empty input")
	}
}

func TestGenerateIDToken(t *testing.T) {
	setupOAuthTokenTest(t)
	userID := uuid.New()

	tokenString, err := GenerateIDToken(userID, "client-abc", "ada@example.com", "Ada Lovelace")
	if err != nil {
		t.Fatalf("GenerateIDToken error: %v", err)
	}
	if tokenString == "" {
		t.Fatal("expected a non-empty token string")
	}

	signer, kid, err := loadOAuthSigner()
	if err != nil {
		t.Fatalf("loadOAuthSigner error: %v", err)
	}

	claims := &IDTokenClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Header["kid"] != kid {
			t.Errorf("kid header = %v, want %v", token.Header["kid"], kid)
		}
		return signer.Public(), nil
	}, jwt.WithValidMethods([]string{oauthSigningMethod().Alg()}))
	if err != nil {
		t.Fatalf("failed to parse/verify the ID token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("ID token reported invalid")
	}

	if claims.Subject != userID.String() {
		t.Errorf("sub = %q, want %q", claims.Subject, userID.String())
	}
	if claims.Email != "ada@example.com" {
		t.Errorf("email = %q, want ada@example.com", claims.Email)
	}
	if claims.Name != "Ada Lovelace" {
		t.Errorf("name = %q, want 'Ada Lovelace'", claims.Name)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != "client-abc" {
		t.Errorf("audience = %v, want [client-abc]", claims.Audience)
	}
}

func TestValidateSSOTokenExpired(t *testing.T) {
	setupAuthTestConfig(t)
	now := time.Now()
	claims := &JWTClaim{
		UserID:  uuid.New(),
		Purpose: PurposeSSO,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}
	token, err := GenerateJWTFromClaims(claims)
	if err != nil {
		t.Fatalf("GenerateJWTFromClaims error: %v", err)
	}

	if _, err := ValidateSSOToken(token); err == nil {
		t.Error("ValidateSSOToken accepted an expired token, want error")
	}
}

func TestValidateSSOTokenNotYetValid(t *testing.T) {
	setupAuthTestConfig(t)
	now := time.Now()
	claims := &JWTClaim{
		UserID:  uuid.New(),
		Purpose: PurposeSSO,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token, err := GenerateJWTFromClaims(claims)
	if err != nil {
		t.Fatalf("GenerateJWTFromClaims error: %v", err)
	}

	if _, err := ValidateSSOToken(token); err == nil {
		t.Error("ValidateSSOToken accepted a not-yet-valid token, want error")
	}
}

func TestValidateSSOTokenMissingClaims(t *testing.T) {
	setupAuthTestConfig(t)
	claims := &JWTClaim{
		UserID:           uuid.New(),
		Purpose:          PurposeSSO,
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	token, err := GenerateJWTFromClaims(claims)
	if err != nil {
		t.Fatalf("GenerateJWTFromClaims error: %v", err)
	}

	if _, err := ValidateSSOToken(token); err == nil {
		t.Error("ValidateSSOToken accepted a token with no exp/nbf claims, want error")
	}
}
