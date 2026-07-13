// Package mcpserver exposes Pønskelisten as an authenticated MCP (Model Context
// Protocol) resource server. It is an OAuth 2.1 resource server: the MCP endpoint
// is guarded by the SDK's bearer-token middleware, which validates our ES256
// access tokens (audience = the MCP resource) and returns the RFC 9728
// WWW-Authenticate challenge so clients can discover the authorization server.
package mcpserver

import (
	pauth "aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	scopeWishlistsRead = "mcp:wishlists.read"
	scopeGroupsRead    = "mcp:groups.read"
)

// noArgs is the input schema for tools that take no arguments.
type noArgs struct{}

type listWishesArgs struct {
	WishlistID string `json:"wishlist_id" jsonschema:"the ID of the wishlist whose wishes to list"`
}

// Output shapes — deliberately trimmed so internal fields never reach the client.
type wishlistOut struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Date        string `json:"date,omitempty"`
}

type wishOut struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Note  string   `json:"note,omitempty"`
	URL   string   `json:"url,omitempty"`
	Price *float64 `json:"price,omitempty"`
}

type groupOut struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Handler builds the MCP server and returns a gin handler that guards it with
// OAuth bearer validation. Serves 404 when MCP is disabled.
func Handler() gin.HandlerFunc {
	server := buildServer()

	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)

	protected := mcpauth.RequireBearerToken(verifyToken, &mcpauth.RequireBearerTokenOptions{
		ResourceMetadataURL: config.OAuthIssuer() + "/.well-known/oauth-protected-resource",
	})(streamable)

	return func(ctx *gin.Context) {
		if !config.ConfigFile.MCPEnabled {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "MCP is not enabled."})
			return
		}
		protected.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

// verifyToken validates an OAuth access token for the MCP resource and returns
// its identity + scopes for the SDK to attach to the request.
func verifyToken(_ context.Context, token string, _ *http.Request) (*mcpauth.TokenInfo, error) {
	claims, err := pauth.ValidateOAuthAccessToken(token, config.MCPResource())
	if err != nil {
		return nil, mcpauth.ErrInvalidToken
	}
	info := &mcpauth.TokenInfo{
		UserID: claims.Subject,
		Scopes: strings.Fields(claims.Scope),
	}
	if claims.ExpiresAt != nil {
		info.Expiration = claims.ExpiresAt.Time
	}
	return info, nil
}

func buildServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "poenskelisten",
		Version: config.ConfigFile.PoenskelistenVersion,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_wishlists",
		Description: "List the wishlists you own, with their id, name, description and date.",
	}, listWishlists)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_wishes",
		Description: "List the wishes on one of your wishlists. Provide the wishlist_id (from list_wishlists).",
	}, listWishes)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_groups",
		Description: "List the groups you are a member of.",
	}, listGroups)

	return server
}

func listWishlists(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, any, error) {
	userID, err := requireScope(ctx, scopeWishlistsRead)
	if err != nil {
		return nil, nil, err
	}

	wishlists, err := database.GetOwnedWishlists(userID)
	if err != nil {
		return nil, nil, errors.New("failed to load wishlists")
	}

	out := make([]wishlistOut, 0, len(wishlists))
	for _, w := range wishlists {
		item := wishlistOut{ID: w.ID.String(), Name: w.Name, Description: w.Description}
		if w.Date != nil {
			item.Date = w.Date.Format("2006-01-02")
		}
		out = append(out, item)
	}
	return jsonResult(out), nil, nil
}

func listWishes(ctx context.Context, _ *mcp.CallToolRequest, args listWishesArgs) (*mcp.CallToolResult, any, error) {
	userID, err := requireScope(ctx, scopeWishlistsRead)
	if err != nil {
		return nil, nil, err
	}

	wishlistID, err := uuid.Parse(strings.TrimSpace(args.WishlistID))
	if err != nil {
		return nil, nil, errors.New("invalid wishlist_id")
	}

	owns, err := database.VerifyUserOwnershipToWishlist(userID, wishlistID)
	if err != nil || !owns {
		return nil, nil, errors.New("wishlist not found or not accessible")
	}

	_, wishes, err := database.GetWishesFromWishlist(wishlistID)
	if err != nil {
		return nil, nil, errors.New("failed to load wishes")
	}

	out := make([]wishOut, 0, len(wishes))
	for _, w := range wishes {
		out = append(out, wishOut{ID: w.ID.String(), Name: w.Name, Note: w.Note, URL: w.URL, Price: w.Price})
	}
	return jsonResult(out), nil, nil
}

func listGroups(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, any, error) {
	userID, err := requireScope(ctx, scopeGroupsRead)
	if err != nil {
		return nil, nil, err
	}

	groups, err := database.GetGroupsAUserIsAMemberOf(userID)
	if err != nil {
		return nil, nil, errors.New("failed to load groups")
	}

	out := make([]groupOut, 0, len(groups))
	for _, g := range groups {
		out = append(out, groupOut{ID: g.ID.String(), Name: g.Name, Description: g.Description})
	}
	return jsonResult(out), nil, nil
}

// requireScope resolves the authenticated user and enforces that the token
// carries the given scope.
func requireScope(ctx context.Context, scope string) (uuid.UUID, error) {
	info := mcpauth.TokenInfoFromContext(ctx)
	if info == nil {
		return uuid.Nil, errors.New("not authenticated")
	}
	if !containsScope(info.Scopes, scope) {
		return uuid.Nil, fmt.Errorf("missing required scope: %s", scope)
	}
	userID, err := uuid.Parse(info.UserID)
	if err != nil {
		return uuid.Nil, errors.New("invalid token subject")
	}
	return userID, nil
}

func containsScope(scopes []string, want string) bool {
	for _, s := range scopes {
		if s == want {
			return true
		}
	}
	return false
}

func jsonResult(v any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		b = []byte("[]")
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}
