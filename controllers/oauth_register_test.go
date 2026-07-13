package controllers

import (
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

func TestIsValidRedirectURI(t *testing.T) {
	cases := map[string]bool{
		"https://client.example/cb": true,
		"http://localhost:1234/cb":  true,
		"not-a-url":                 false,
		"ftp://x/y":                 false,
		"":                          false,
		"/relative/path":            false,
	}
	for uri, want := range cases {
		if got := isValidRedirectURI(uri); got != want {
			t.Errorf("isValidRedirectURI(%q) = %v, want %v", uri, got, want)
		}
	}
}
