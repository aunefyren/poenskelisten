package mcpserver

import (
	pauth "aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"context"
	"testing"

	"github.com/google/uuid"
)

func setupMCPConfig(t *testing.T) {
	t.Helper()
	keyPEM, kid, err := config.GenerateOAuthSigningKey(config.OAuthAlgES256)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	config.ConfigFile.OAuthSigningKey = keyPEM
	config.ConfigFile.OAuthSigningKeyID = kid
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
}

func TestContainsScope(t *testing.T) {
	scopes := []string{"mcp:wishlists.read", "mcp:groups.read"}
	if !containsScope(scopes, "mcp:groups.read") {
		t.Error("expected scope to be found")
	}
	if containsScope(scopes, "mcp:wishlists.write") {
		t.Error("did not expect scope to be found")
	}
}

func TestVerifyTokenValid(t *testing.T) {
	setupMCPConfig(t)
	userID := uuid.New()

	token, err := pauth.GenerateOAuthAccessToken(userID, config.MCPResource(), "mcp:wishlists.read mcp:groups.read", false, true)
	if err != nil {
		t.Fatalf("GenerateOAuthAccessToken error: %v", err)
	}

	info, err := verifyToken(context.Background(), token, nil)
	if err != nil {
		t.Fatalf("verifyToken error: %v", err)
	}
	if info.UserID != userID.String() {
		t.Errorf("UserID = %q, want %q", info.UserID, userID.String())
	}
	if !containsScope(info.Scopes, "mcp:wishlists.read") {
		t.Errorf("scopes = %v, want to include mcp:wishlists.read", info.Scopes)
	}
}

func TestVerifyTokenWrongAudience(t *testing.T) {
	setupMCPConfig(t)

	// A token for the API resource must not be accepted by the MCP resource server.
	token, err := pauth.GenerateOAuthAccessToken(uuid.New(), config.APIResource(), "openid", false, true)
	if err != nil {
		t.Fatalf("GenerateOAuthAccessToken error: %v", err)
	}
	if _, err := verifyToken(context.Background(), token, nil); err == nil {
		t.Error("verifyToken accepted a token for the wrong audience")
	}
}

func TestVerifyTokenGarbage(t *testing.T) {
	setupMCPConfig(t)
	if _, err := verifyToken(context.Background(), "not-a-token", nil); err == nil {
		t.Error("verifyToken accepted garbage")
	}
}
