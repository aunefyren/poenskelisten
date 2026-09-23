package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func postRegister(body string) (int, map[string]interface{}) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	APIOAuthRegister(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func TestOAuthRegisterSuccess(t *testing.T) {
	setupControllersDB(t)

	code, body := postRegister(`{"redirect_uris":["https://client.example/callback"],"client_name":"Test MCP","scope":"openid mcp:wishlists.read"}`)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, body)
	}
	if body["client_id"] == nil || body["client_id"] == "" {
		t.Error("expected a client_id")
	}
	if _, hasSecret := body["client_secret"]; hasSecret {
		t.Error("public client must not be issued a secret")
	}
	if body["token_endpoint_auth_method"] != "none" {
		t.Errorf("token_endpoint_auth_method = %v, want none", body["token_endpoint_auth_method"])
	}
	if !strings.Contains(body["scope"].(string), "mcp:wishlists.read") {
		t.Errorf("scope = %v, want the requested MCP scope", body["scope"])
	}
}

func TestOAuthRegisterRequiresRedirectURI(t *testing.T) {
	setupControllersDB(t)

	if code, _ := postRegister(`{"client_name":"No Redirect"}`); code != 400 {
		t.Errorf("status = %d, want 400 when redirect_uris missing", code)
	}
}

func TestOAuthRegisterRejectsBadRedirectURI(t *testing.T) {
	setupControllersDB(t)

	if code, _ := postRegister(`{"redirect_uris":["not-a-url"]}`); code != 400 {
		t.Errorf("status = %d, want 400 for a non-absolute redirect_uri", code)
	}
}

func TestOAuthRegisterRejectsUnsupportedGrantType(t *testing.T) {
	setupControllersDB(t)

	code, body := postRegister(`{"redirect_uris":["https://client.example/cb"],"grant_types":["client_credentials"]}`)
	if code != 400 {
		t.Fatalf("status = %d, want 400 for an unsupported grant_type; body=%v", code, body)
	}
	if body["error"] != "invalid_client_metadata" {
		t.Errorf("error = %v, want invalid_client_metadata", body["error"])
	}
}

func TestOAuthRegisterDefaultGrantTypes(t *testing.T) {
	setupControllersDB(t)

	code, body := postRegister(`{"redirect_uris":["https://client.example/cb"]}`)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, body)
	}
	grantTypes, ok := body["grant_types"].([]interface{})
	if !ok || len(grantTypes) != 2 {
		t.Fatalf("grant_types = %v, want the two default grant types", body["grant_types"])
	}
}

func TestOAuthRegisterEmptyScopeDefaultsToAll(t *testing.T) {
	setupControllersDB(t)

	code, body := postRegister(`{"redirect_uris":["https://client.example/cb"]}`)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, body)
	}
	scope, _ := body["scope"].(string)
	if !strings.Contains(scope, "openid") {
		t.Errorf("scope = %q, want it to default to every known scope (including openid)", scope)
	}
}

func TestOAuthRegisterFiltersUnknownScopes(t *testing.T) {
	setupControllersDB(t)

	code, body := postRegister(`{"redirect_uris":["https://client.example/cb"],"scope":"openid not-a-real-scope"}`)
	if code != 201 {
		t.Fatalf("status = %d, want 201; body=%v", code, body)
	}
	scope, _ := body["scope"].(string)
	if !strings.Contains(scope, "openid") {
		t.Errorf("scope = %q, want openid preserved", scope)
	}
	if strings.Contains(scope, "not-a-real-scope") {
		t.Errorf("scope = %q, want the unknown scope filtered out", scope)
	}
}

func TestOAuthRegisterDatabaseFailure(t *testing.T) {
	// Migrate without the OAuthClient table so database.CreateOAuthClient fails.
	setupControllersDB(t, &models.User{})

	code, body := postRegister(`{"redirect_uris":["https://client.example/cb"]}`)
	if code != 400 {
		t.Fatalf("status = %d, want 400 (registrationError always replies 400); body=%v", code, body)
	}
	if body["error"] != "server_error" {
		t.Errorf("error = %v, want server_error", body["error"])
	}
}

func TestAdminListOAuthClients(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/admin/oauth/clients", nil)
	APIAdminListOAuthClients(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Clients []map[string]interface{} `json:"clients"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body.Clients) != 0 {
		t.Errorf("clients = %v, want empty on a fresh DB", body.Clients)
	}

	if _, err := database.CreateOAuthClient(models.OAuthClient{
		ClientID:   "test-client",
		ClientName: "Test Client",
		Enabled:    boolPtr(true),
	}); err != nil {
		t.Fatalf("failed to create OAuth client: %v", err)
	}

	w2 := httptest.NewRecorder()
	ctx2, _ := gin.CreateTestContext(w2)
	ctx2.Request = httptest.NewRequest("GET", "/api/admin/oauth/clients", nil)
	APIAdminListOAuthClients(ctx2)

	if w2.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w2.Code, w2.Body.String())
	}
	var body2 struct {
		Clients []map[string]interface{} `json:"clients"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &body2); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body2.Clients) != 1 || body2.Clients[0]["client_name"] != "Test Client" {
		t.Errorf("clients = %v, want exactly the one Test Client", body2.Clients)
	}
}

func TestAdminRevokeOAuthClientSuccess(t *testing.T) {
	setupControllersDB(t)
	if _, err := database.CreateOAuthClient(models.OAuthClient{
		ClientID:   "test-client",
		ClientName: "Test Client",
		Enabled:    boolPtr(true),
	}); err != nil {
		t.Fatalf("failed to create OAuth client: %v", err)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/admin/oauth/clients/test-client", nil)
	ctx.Params = gin.Params{{Key: "client_id", Value: "test-client"}}
	APIAdminRevokeOAuthClient(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	client, found, err := database.GetOAuthClient("test-client")
	if err != nil || !found {
		t.Fatalf("expected the client to still exist (disabled, not deleted): found=%v err=%v", found, err)
	}
	if client.IsEnabled() {
		t.Error("expected the client to be disabled after revoke")
	}
}

func TestAdminRevokeOAuthClientNotFound(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/admin/oauth/clients/does-not-exist", nil)
	ctx.Params = gin.Params{{Key: "client_id", Value: "does-not-exist"}}
	APIAdminRevokeOAuthClient(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for an unknown client_id; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminRevokeOAuthClientRejectsFirstParty(t *testing.T) {
	setupControllersDB(t)
	if _, err := database.CreateOAuthClient(models.OAuthClient{
		ClientID:     models.FirstPartyClientID,
		ClientName:   "Built-in web client",
		IsFirstParty: true,
		Enabled:      boolPtr(true),
	}); err != nil {
		t.Fatalf("failed to create first-party OAuth client: %v", err)
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("DELETE", "/api/admin/oauth/clients/"+models.FirstPartyClientID, nil)
	ctx.Params = gin.Params{{Key: "client_id", Value: models.FirstPartyClientID}}
	APIAdminRevokeOAuthClient(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 when trying to revoke the built-in first-party client; body=%s", w.Code, w.Body.String())
	}
}

func TestIsValidRedirectURI(t *testing.T) {
	cases := map[string]bool{
		"https://client.example/cb": true,
		"http://localhost:1234/cb":  true,
		"not-a-url":                 false,
		"ftp://x/y":                 false,
		"":                          false,
		"/relative/path":            false,
		"http://[::1":               false, // unparseable
	}
	for uri, want := range cases {
		if got := isValidRedirectURI(uri); got != want {
			t.Errorf("isValidRedirectURI(%q) = %v, want %v", uri, got, want)
		}
	}
}

func TestOAuthRegisterMalformedJSON(t *testing.T) {
	setupControllersDB(t)

	code, body := postRegister(`not-json`)
	if code != 400 || body["error"] != "invalid_client_metadata" {
		t.Errorf("status = %d body=%v, want 400 invalid_client_metadata", code, body)
	}
}

func TestAdminListOAuthClientsDatabaseError(t *testing.T) {
	setupControllersDB(t)
	breakControllersDB(t)

	status, body, _ := doRequest(APIAdminListOAuthClients, "GET", "/api/admin/oauth/clients", "", nil, nil)
	if status != 500 || body["error"] != "Failed to list clients." {
		t.Errorf("status = %d body=%v, want 500 'Failed to list clients.'", status, body)
	}
}
