package mcpserver

import (
	pauth "aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
)

func setupMCPConfig(t *testing.T) {
	t.Helper()
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })
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

// contextWithTokenInfo returns a context carrying info exactly the way the
// SDK's bearer middleware attaches it (the context key is unexported), so the
// tool functions can be called directly with any token shape.
func contextWithTokenInfo(t *testing.T, info *mcpauth.TokenInfo) context.Context {
	t.Helper()
	var captured context.Context
	verifier := func(context.Context, string, *http.Request) (*mcpauth.TokenInfo, error) { return info, nil }
	handler := mcpauth.RequireBearerToken(verifier, nil)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r.Context()
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer irrelevant")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if captured == nil {
		t.Fatal("bearer middleware did not reach the inner handler")
	}
	return captured
}

func TestRequireScope(t *testing.T) {
	userID := uuid.New()
	expiry := time.Now().Add(time.Hour)

	cases := []struct {
		name    string
		ctx     context.Context
		wantErr string
	}{
		{"no token info", context.Background(), "not authenticated"},
		{"missing scope", contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: userID.String(), Scopes: []string{scopeGroupsRead}, Expiration: expiry}), "missing required scope"},
		{"subject not a UUID", contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: "not-a-uuid", Scopes: []string{scopeWishlistsRead}, Expiration: expiry}), "invalid token subject"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := requireScope(c.ctx, scopeWishlistsRead)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("requireScope error = %v, want it to contain %q", err, c.wantErr)
			}
			if got != uuid.Nil {
				t.Errorf("requireScope userID = %v, want uuid.Nil on failure", got)
			}
		})
	}

	ctx := contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: userID.String(), Scopes: []string{scopeWishlistsRead}, Expiration: expiry})
	if got, err := requireScope(ctx, scopeWishlistsRead); err != nil || got != userID {
		t.Errorf("requireScope = (%v, %v), want (%v, nil)", got, err, userID)
	}
}

// TestToolsRejectMissingScope calls each tool directly with a token lacking
// the scope it needs; none of them may touch the database.
func TestToolsRejectMissingScope(t *testing.T) {
	ctx := contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: uuid.NewString(), Scopes: []string{"unrelated"}, Expiration: time.Now().Add(time.Hour)})

	if res, _, err := listWishlists(ctx, nil, noArgs{}); err == nil || res != nil {
		t.Errorf("listWishlists = (%v, %v), want nil result and a scope error", res, err)
	}
	if res, _, err := listWishes(ctx, nil, listWishesArgs{WishlistID: uuid.NewString()}); err == nil || res != nil {
		t.Errorf("listWishes = (%v, %v), want nil result and a scope error", res, err)
	}
	if res, _, err := listGroups(ctx, nil, noArgs{}); err == nil || res != nil {
		t.Errorf("listGroups = (%v, %v), want nil result and a scope error", res, err)
	}
}

// TestToolsSurfaceDatabaseFailures drops the table each tool reads so the
// query fails, and checks the tool returns its generic error rather than a
// partial or empty result.
func TestToolsSurfaceDatabaseFailures(t *testing.T) {
	allScopes := []string{scopeWishlistsRead, scopeGroupsRead}

	t.Run("list_wishlists", func(t *testing.T) {
		setupMCPDB(t)
		owner := createMCPTestUser(t, "Ada")
		if err := database.Instance.Migrator().DropTable(&models.Wishlist{}); err != nil {
			t.Fatalf("failed to drop wishlists: %v", err)
		}
		ctx := contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: owner.ID.String(), Scopes: allScopes, Expiration: time.Now().Add(time.Hour)})

		res, _, err := listWishlists(ctx, nil, noArgs{})
		if err == nil || err.Error() != "failed to load wishlists" || res != nil {
			t.Errorf("listWishlists = (%v, %v), want 'failed to load wishlists'", res, err)
		}
	})

	t.Run("list_wishes", func(t *testing.T) {
		setupMCPDB(t)
		owner := createMCPTestUser(t, "Ada")
		now := time.Now()
		wishlist := models.Wishlist{Name: "Birthday", Enabled: true, OwnerID: owner.ID, Date: &now}
		wishlist.ID = uuid.New()
		if r := database.Instance.Create(&wishlist); r.Error != nil {
			t.Fatalf("failed to create wishlist: %v", r.Error)
		}
		// Ownership still verifies against wishlists; only the wish query fails.
		if err := database.Instance.Migrator().DropTable(&models.Wish{}); err != nil {
			t.Fatalf("failed to drop wishes: %v", err)
		}
		ctx := contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: owner.ID.String(), Scopes: allScopes, Expiration: time.Now().Add(time.Hour)})

		res, _, err := listWishes(ctx, nil, listWishesArgs{WishlistID: wishlist.ID.String()})
		if err == nil || err.Error() != "failed to load wishes" || res != nil {
			t.Errorf("listWishes = (%v, %v), want 'failed to load wishes'", res, err)
		}
	})

	t.Run("list_groups", func(t *testing.T) {
		setupMCPDB(t)
		owner := createMCPTestUser(t, "Ada")
		if err := database.Instance.Migrator().DropTable(&models.Group{}); err != nil {
			t.Fatalf("failed to drop groups: %v", err)
		}
		ctx := contextWithTokenInfo(t, &mcpauth.TokenInfo{UserID: owner.ID.String(), Scopes: allScopes, Expiration: time.Now().Add(time.Hour)})

		res, _, err := listGroups(ctx, nil, noArgs{})
		if err == nil || err.Error() != "failed to load groups" || res != nil {
			t.Errorf("listGroups = (%v, %v), want 'failed to load groups'", res, err)
		}
	})
}
