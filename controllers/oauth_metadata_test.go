package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func init() {
	gin.SetMode(gin.TestMode)
	if logger.Log == nil {
		logger.Log = logrus.New()
	}
}

// enableOAuth installs a working OAuth/MCP config (issuer + generated key).
func enableOAuth(t *testing.T) {
	t.Helper()
	keyPEM, kid, err := config.GenerateOAuthSigningKey(config.OAuthAlgES256)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
	config.ConfigFile.OAuthSigningKey = keyPEM
	config.ConfigFile.OAuthSigningKeyID = kid
	config.ConfigFile.MCPEnabled = true
}

func runHandler(handler gin.HandlerFunc) (int, map[string]interface{}) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	handler(ctx)

	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

func TestAuthorizationServerMetadata(t *testing.T) {
	enableOAuth(t)

	code, body := runHandler(APIOAuthAuthorizationServerMetadata)
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	if body["issuer"] != "https://wishlist.example.com" {
		t.Errorf("issuer = %v", body["issuer"])
	}
	if body["authorization_endpoint"] != "https://wishlist.example.com/oauth/authorize" {
		t.Errorf("authorization_endpoint = %v", body["authorization_endpoint"])
	}
	if body["token_endpoint"] != "https://wishlist.example.com/oauth/token" {
		t.Errorf("token_endpoint = %v", body["token_endpoint"])
	}
	if body["jwks_uri"] != "https://wishlist.example.com/.well-known/jwks.json" {
		t.Errorf("jwks_uri = %v", body["jwks_uri"])
	}
	methods, ok := body["code_challenge_methods_supported"].([]interface{})
	if !ok || len(methods) != 1 || methods[0] != "S256" {
		t.Errorf("code_challenge_methods_supported = %v, want [S256]", body["code_challenge_methods_supported"])
	}
}

func TestProtectedResourceMetadata(t *testing.T) {
	enableOAuth(t)

	code, body := runHandler(APIOAuthProtectedResourceMetadata)
	if code != 200 {
		t.Fatalf("status = %d, want 200", code)
	}
	if body["resource"] != "https://wishlist.example.com/mcp" {
		t.Errorf("resource = %v", body["resource"])
	}
	servers, ok := body["authorization_servers"].([]interface{})
	if !ok || len(servers) != 1 || servers[0] != "https://wishlist.example.com" {
		t.Errorf("authorization_servers = %v", body["authorization_servers"])
	}
}

func TestProtectedResourceMetadataDisabled(t *testing.T) {
	enableOAuth(t)
	config.ConfigFile.MCPEnabled = false
	if code, _ := runHandler(APIOAuthProtectedResourceMetadata); code != 404 {
		t.Errorf("status = %d, want 404 when MCP disabled", code)
	}
}

func TestJWKSEndpoint(t *testing.T) {
	enableOAuth(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	APIOAuthJWKS(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var jwks struct {
		Keys []map[string]interface{} `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jwks); err != nil {
		t.Fatalf("failed to parse JWKS: %v", err)
	}
	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(jwks.Keys))
	}
	// Public EC key must not leak the private component.
	if _, hasPrivate := jwks.Keys[0]["d"]; hasPrivate {
		t.Error("JWKS leaked the private key component 'd'")
	}
	if jwks.Keys[0]["kid"] != config.ConfigFile.OAuthSigningKeyID {
		t.Errorf("kid = %v, want %v", jwks.Keys[0]["kid"], config.ConfigFile.OAuthSigningKeyID)
	}
}
