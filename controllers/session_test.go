package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestLogoutAllRequiresAuth(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Session{})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/auth/logout/all", nil)
	APILogoutAll(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without an Authorization header", w.Code)
	}
}

func TestLogoutAllRevokesSessions(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Session{})
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/auth/logout/all", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	APILogoutAll(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["message"] != "Signed out of all sessions." {
		t.Errorf("message = %v", body["message"])
	}

	cookies := w.Result().Cookies()
	foundCleared := false
	for _, c := range cookies {
		if c.Name == refreshCookieName && c.MaxAge < 0 {
			foundCleared = true
		}
	}
	if !foundCleared {
		t.Error("expected the refresh cookie to be cleared")
	}
}

func TestAdminRevokeUserSessionsInvalidUserID(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Session{})

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/users/not-a-uuid/sessions/revoke", nil)
	ctx.Params = gin.Params{{Key: "user_id", Value: "not-a-uuid"}}
	APIAdminRevokeUserSessions(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed user_id", w.Code)
	}
}

func TestAdminRevokeUserSessionsSuccess(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Session{})
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/users/"+user.ID.String()+"/sessions/revoke", nil)
	ctx.Params = gin.Params{{Key: "user_id", Value: user.ID.String()}}
	APIAdminRevokeUserSessions(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminRevokeUserSessionsUnknownUser(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Session{})

	unknown := uuid.New()
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/users/"+unknown.String()+"/sessions/revoke", nil)
	ctx.Params = gin.Params{{Key: "user_id", Value: unknown.String()}}
	APIAdminRevokeUserSessions(ctx)

	// Revoking sessions for a user with none is not an error.
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestLogoutAllSessionRevokeFailure(t *testing.T) {
	// User table present (so auth/user lookup works) but no Session table, so
	// the second step of revokeAllForUser (RevokeAllUserSessions) fails.
	setupControllersDB(t, &models.User{})
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/auth/logout/all", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	APILogoutAll(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when sessions can't be revoked; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminRevokeUserSessionsFailure(t *testing.T) {
	setupControllersDB(t, &models.User{})
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/admin/users/"+user.ID.String()+"/sessions/revoke", nil)
	ctx.Params = gin.Params{{Key: "user_id", Value: user.ID.String()}}
	APIAdminRevokeUserSessions(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when sessions can't be revoked; body=%s", w.Code, w.Body.String())
	}
}

func TestIssueSSOSessionFailsWithoutPrivateKey(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Session{})
	// Deliberately leave config.ConfigFile.PrivateKey unset.
	config.ConfigFile.PrivateKey = ""
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/", nil)

	if err := issueSSOSession(ctx, user); err == nil {
		t.Error("expected an error when no private key is configured")
	}
}

func TestIssueSSOSessionSuccess(t *testing.T) {
	setupControllersDB(t, &models.User{}, &models.Session{})
	enablePrivateKey(t)
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/", nil)

	if err := issueSSOSession(ctx, user); err != nil {
		t.Fatalf("issueSSOSession returned error: %v", err)
	}

	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == ssoCookieName && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected the SSO cookie to be set")
	}
}

func TestSessionHandlersDatabaseErrors(t *testing.T) {
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "APILogoutAll", handler: APILogoutAll, method: "POST", path: "/api/auth/tokens/logout-all"},
		{name: "APIAdminRevokeUserSessions", handler: APIAdminRevokeUserSessions, method: "DELETE", path: "/api/admin/users/00000000-0000-0000-0000-00000000000a/sessions", params: gin.Params{{Key: "user_id", Value: "00000000-0000-0000-0000-00000000000a"}}, admin: true},
	})
}
