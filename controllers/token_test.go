package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func postGenerateToken(body string) (int, map[string]interface{}) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/open/tokens/register", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	GenerateToken(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func TestGenerateTokenBadJSON(t *testing.T) {
	setupControllersDB(t)
	if code, _ := postGenerateToken("not json"); code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestGenerateTokenUnknownEmail(t *testing.T) {
	setupControllersDB(t)

	code, body := postGenerateToken(`{"email":"nobody@example.com","password":"whatever"}`)
	if code != 500 {
		t.Fatalf("status = %d, want 500 for an unknown e-mail; body=%v", code, body)
	}
	if body["error"] != "Invalid credentials." {
		t.Errorf("error = %v, want a generic invalid-credentials message", body["error"])
	}
}

func TestGenerateTokenWrongPassword(t *testing.T) {
	setupControllersDB(t)
	user := createTestUserWithPassword(t, "correct-horse-battery-staple")

	code, body := postGenerateToken(`{"email":"` + *user.Email + `","password":"wrong-password"}`)
	if code != 401 {
		t.Fatalf("status = %d, want 401 for a wrong password; body=%v", code, body)
	}
}

func TestGenerateTokenSuccess(t *testing.T) {
	setupControllersDB(t)
	enablePrivateKey(t)
	user := createTestUserWithPassword(t, "correct-horse-battery-staple")

	code, body := postGenerateToken(`{"email":"` + *user.Email + `","password":"correct-horse-battery-staple"}`)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["message"] != "Logged in!" {
		t.Errorf("message = %v", body["message"])
	}
	if body["mfa_required"] != nil {
		t.Errorf("mfa_required = %v, want absent for a user without MFA", body["mfa_required"])
	}
}

func TestGenerateTokenMFARequired(t *testing.T) {
	setupControllersDB(t)
	// The MFA challenge token is HS256-signed; don't rely on an earlier test
	// having left a key configured.
	origKey := config.ConfigFile.PrivateKey
	t.Cleanup(func() { config.ConfigFile.PrivateKey = origKey })
	enablePrivateKey(t)
	user := createTestUserWithPassword(t, "correct-horse-battery-staple")

	user.MFAEnabled = boolPtr(true)
	user.MFASecret = strPtr("JBSWY3DPEHPK3PXP")
	updated, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to enable MFA on test user: %v", err)
	}

	code, body := postGenerateToken(`{"email":"` + *updated.Email + `","password":"correct-horse-battery-staple"}`)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["mfa_required"] != true {
		t.Errorf("mfa_required = %v, want true", body["mfa_required"])
	}
	if body["mfa_token"] == nil || body["mfa_token"] == "" {
		t.Error("expected a non-empty mfa_token")
	}
}

func strPtr(s string) *string { return &s }

// tokenTestClearPrivateKey unsets the HS256 signing key for the rest of the
// test, so minting the MFA challenge token or SSO cookie fails.
func tokenTestClearPrivateKey(t *testing.T) {
	t.Helper()
	orig := config.ConfigFile.PrivateKey
	t.Cleanup(func() { config.ConfigFile.PrivateKey = orig })
	config.ConfigFile.PrivateKey = ""
}

func TestGenerateTokenSessionFailure(t *testing.T) {
	setupControllersDB(t)
	user := createTestUserWithPassword(t, "correct-horse-battery-staple")
	tokenTestClearPrivateKey(t)

	code, body := postGenerateToken(`{"email":"` + *user.Email + `","password":"correct-horse-battery-staple"}`)
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the session cookie can't be signed; body=%v", code, body)
	}
	if body["error"] != "Failed to log in." {
		t.Errorf("error = %v", body["error"])
	}
}

func TestGenerateTokenMFAChallengeFailure(t *testing.T) {
	setupControllersDB(t)
	user := createTestUserWithPassword(t, "correct-horse-battery-staple")
	user.MFAEnabled = boolPtr(true)
	user.MFASecret = strPtr("JBSWY3DPEHPK3PXP")
	if _, err := database.UpdateUserInDB(user); err != nil {
		t.Fatalf("failed to enable MFA on test user: %v", err)
	}
	tokenTestClearPrivateKey(t)

	code, body := postGenerateToken(`{"email":"` + *user.Email + `","password":"correct-horse-battery-staple"}`)
	if code != 500 {
		t.Fatalf("status = %d, want 500 when the MFA challenge token can't be signed; body=%v", code, body)
	}
	if body["mfa_token"] != nil {
		t.Errorf("mfa_token = %v, want absent", body["mfa_token"])
	}
}
