package controllers

import (
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
