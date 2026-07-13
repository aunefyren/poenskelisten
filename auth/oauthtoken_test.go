package auth

import (
	"aunefyren/poenskelisten/config"
	"crypto/sha256"
	"encoding/base64"
	"testing"

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
