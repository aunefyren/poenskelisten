package controllers

import (
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
