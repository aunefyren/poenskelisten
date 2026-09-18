package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestListConnectedAppsRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/connected-apps", nil)
	APIListConnectedApps(ctx)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401 without an Authorization header", w.Code)
	}
}

func TestListConnectedAppsEmpty(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/connected-apps", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	APIListConnectedApps(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Apps []map[string]interface{} `json:"apps"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body.Apps) != 0 {
		t.Errorf("apps = %v, want empty", body.Apps)
	}
}

func TestListConnectedAppsWithConsent(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	client := models.OAuthClient{
		ClientID:   "test-client",
		ClientName: "Test Client",
	}
	if result := database.Instance.Create(&client); result.Error != nil {
		t.Fatalf("failed to create OAuth client: %v", result.Error)
	}

	consent := models.OAuthConsent{
		UserID:   user.ID,
		ClientID: "test-client",
		Scopes:   []string{"openid", "profile"},
	}
	if result := database.Instance.Create(&consent); result.Error != nil {
		t.Fatalf("failed to create consent: %v", result.Error)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/connected-apps", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	APIListConnectedApps(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Apps []map[string]interface{} `json:"apps"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body.Apps) != 1 {
		t.Fatalf("apps = %v, want 1 entry", body.Apps)
	}
	if body.Apps[0]["client_name"] != "Test Client" {
		t.Errorf("client_name = %v, want 'Test Client'", body.Apps[0]["client_name"])
	}
}

func TestRevokeConnectedAppRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/auth/connected-apps/test-client", nil)
	ctx.Params = gin.Params{{Key: "client_id", Value: "test-client"}}
	APIRevokeConnectedApp(ctx)

	if w.Code != 401 {
		t.Fatalf("status = %d, want 401 without an Authorization header", w.Code)
	}
}

func TestRevokeConnectedAppSuccess(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	client := models.OAuthClient{ClientID: "test-client", ClientName: "Test Client"}
	if result := database.Instance.Create(&client); result.Error != nil {
		t.Fatalf("failed to create OAuth client: %v", result.Error)
	}
	consent := models.OAuthConsent{UserID: user.ID, ClientID: "test-client", Scopes: []string{"openid"}}
	if result := database.Instance.Create(&consent); result.Error != nil {
		t.Fatalf("failed to create consent: %v", result.Error)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/auth/connected-apps/test-client", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	ctx.Params = gin.Params{{Key: "client_id", Value: "test-client"}}
	APIRevokeConnectedApp(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	remaining, err := database.GetUserConsents(user.ID)
	if err != nil {
		t.Fatalf("failed to list consents: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected the consent to be revoked, got %v", remaining)
	}
}

func TestRevokeConnectedAppUnknownClient(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/auth/connected-apps/does-not-exist", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	ctx.Params = gin.Params{{Key: "client_id", Value: "does-not-exist"}}
	APIRevokeConnectedApp(ctx)

	// Revoking a consent that doesn't exist is not an error.
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestRevokeConnectedAppConsentDeleteFailure(t *testing.T) {
	// Migrate without the oauth_consents table so RevokeConsent's DELETE fails.
	setupControllersDB(t, &models.User{}, &models.Session{})
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/auth/connected-apps/test-client", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	ctx.Params = gin.Params{{Key: "client_id", Value: "test-client"}}
	APIRevokeConnectedApp(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the consent table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestRevokeConnectedAppSessionRevokeFailure(t *testing.T) {
	// Migrate without the sessions table so RevokeUserClientSessions fails,
	// even though the preceding RevokeConsent succeeds.
	setupControllersDB(t, &models.User{}, &models.OAuthConsent{})
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/auth/connected-apps/test-client", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	ctx.Params = gin.Params{{Key: "client_id", Value: "test-client"}}
	APIRevokeConnectedApp(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the sessions table is unavailable; body=%s", w.Code, w.Body.String())
	}
}

func TestListConnectedAppsDatabaseFailure(t *testing.T) {
	// Migrate without the oauth_consents table so GetUserConsents fails.
	setupControllersDB(t, &models.User{})
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/connected-apps", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	APIListConnectedApps(ctx)

	if w.Code != 500 {
		t.Fatalf("status = %d, want 500 when the consent table is unavailable; body=%s", w.Code, w.Body.String())
	}
}
