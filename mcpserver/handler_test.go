package mcpserver

import (
	pauth "aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupMCPDB points database.Instance at a fresh in-memory SQLite DB with the
// tables the mcpserver tools touch.
func setupMCPDB(t *testing.T) {
	t.Helper()

	dbSQL, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { dbSQL.Close() })
	dbSQL.SetMaxOpenConns(1)

	instance, err := gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	if err := instance.AutoMigrate(
		&models.User{}, &models.Wishlist{}, &models.Wish{},
		&models.Group{}, &models.GroupMembership{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	database.Instance = instance
}

// bearerRoundTripper injects a fixed Authorization header into every request,
// standing in for a real OAuth client since StreamableClientTransport has no
// simpler way to attach a bearer token.
type bearerRoundTripper struct {
	token string
}

func (rt bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+rt.token)
	return http.DefaultTransport.RoundTrip(req)
}

// startMCPTestServer wires Handler() into a real gin router served over HTTP,
// enables MCP, and returns the server URL plus a connected+authenticated
// client session ready to call tools.
func startMCPTestServer(t *testing.T, userID uuid.UUID, scope string) *sdkmcp.ClientSession {
	t.Helper()
	setupMCPConfig(t)
	config.ConfigFile.MCPEnabled = true

	router := gin.New()
	router.Any("/mcp", Handler())
	httpServer := httptest.NewServer(router)
	t.Cleanup(httpServer.Close)

	token, err := pauth.GenerateOAuthAccessToken(userID, "mcp-client", config.MCPResource(), scope, false, true)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	httpClient := &http.Client{Transport: bearerRoundTripper{token: token}}
	transport := &sdkmcp.StreamableClientTransport{Endpoint: httpServer.URL + "/mcp", HTTPClient: httpClient}
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "0.0.1"}, nil)

	session, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	t.Cleanup(func() { session.Close() })

	return session
}

func TestHandlerDisabledReturns404(t *testing.T) {
	setupMCPConfig(t)
	config.ConfigFile.MCPEnabled = false

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/mcp", nil)
	Handler()(ctx)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when MCP is disabled", w.Code)
	}
}

func TestHandlerRejectsMissingBearerToken(t *testing.T) {
	setupMCPConfig(t)
	config.ConfigFile.MCPEnabled = true

	router := gin.New()
	router.Any("/mcp", Handler())
	httpServer := httptest.NewServer(router)
	defer httpServer.Close()

	resp, err := http.Post(httpServer.URL+"/mcp", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 without a bearer token", resp.StatusCode)
	}
}

func TestListWishlistsTool(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	now := time.Now()
	wishlist := models.Wishlist{Name: "Birthday", Enabled: true, OwnerID: owner.ID, Date: &now}
	wishlist.ID = uuid.New()
	if r := database.Instance.Create(&wishlist); r.Error != nil {
		t.Fatalf("failed to create wishlist: %v", r.Error)
	}

	session := startMCPTestServer(t, owner.ID, "mcp:wishlists.read")

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "list_wishlists"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool call reported an error: %+v", result.Content)
	}

	text := firstText(t, result)
	var got []wishlistOut
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("failed to parse tool result: %v (raw: %s)", err, text)
	}
	if len(got) != 1 || got[0].Name != "Birthday" {
		t.Errorf("got %+v, want exactly the one 'Birthday' wishlist", got)
	}
}

func TestListWishlistsToolMissingScope(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")

	session := startMCPTestServer(t, owner.ID, "mcp:groups.read") // wrong scope

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "list_wishlists"})
	if err != nil {
		t.Fatalf("CallTool transport-level error: %v", err)
	}
	if !result.IsError {
		t.Error("expected an error result when the token lacks mcp:wishlists.read")
	}
}

func TestListWishesTool(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	now := time.Now()
	wishlist := models.Wishlist{Name: "Birthday", Enabled: true, OwnerID: owner.ID, Date: &now}
	wishlist.ID = uuid.New()
	if r := database.Instance.Create(&wishlist); r.Error != nil {
		t.Fatalf("failed to create wishlist: %v", r.Error)
	}
	wish := models.Wish{Name: "Headphones", Enabled: true, OwnerID: owner.ID, WishlistID: wishlist.ID}
	wish.ID = uuid.New()
	if r := database.Instance.Create(&wish); r.Error != nil {
		t.Fatalf("failed to create wish: %v", r.Error)
	}

	session := startMCPTestServer(t, owner.ID, "mcp:wishlists.read")

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "list_wishes",
		Arguments: map[string]any{"wishlist_id": wishlist.ID.String()},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool call reported an error: %+v", result.Content)
	}

	text := firstText(t, result)
	var got []wishOut
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("failed to parse tool result: %v (raw: %s)", err, text)
	}
	if len(got) != 1 || got[0].Name != "Headphones" {
		t.Errorf("got %+v, want exactly the one 'Headphones' wish", got)
	}
}

func TestListWishesToolInvalidWishlistID(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")

	session := startMCPTestServer(t, owner.ID, "mcp:wishlists.read")

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "list_wishes",
		Arguments: map[string]any{"wishlist_id": "not-a-uuid"},
	})
	if err != nil {
		t.Fatalf("CallTool transport-level error: %v", err)
	}
	if !result.IsError {
		t.Error("expected an error result for an invalid wishlist_id")
	}
}

func TestListWishesToolNotOwner(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	outsider := createMCPTestUser(t, "Bob")
	now := time.Now()
	wishlist := models.Wishlist{Name: "Birthday", Enabled: true, OwnerID: owner.ID, Date: &now}
	wishlist.ID = uuid.New()
	if r := database.Instance.Create(&wishlist); r.Error != nil {
		t.Fatalf("failed to create wishlist: %v", r.Error)
	}

	session := startMCPTestServer(t, outsider.ID, "mcp:wishlists.read")

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      "list_wishes",
		Arguments: map[string]any{"wishlist_id": wishlist.ID.String()},
	})
	if err != nil {
		t.Fatalf("CallTool transport-level error: %v", err)
	}
	if !result.IsError {
		t.Error("expected an error result when the caller doesn't own the wishlist")
	}
}

func TestListGroupsTool(t *testing.T) {
	setupMCPDB(t)
	owner := createMCPTestUser(t, "Ada")
	group := models.Group{Name: "Family", Enabled: true, OwnerID: owner.ID}
	group.ID = uuid.New()
	if r := database.Instance.Create(&group); r.Error != nil {
		t.Fatalf("failed to create group: %v", r.Error)
	}
	membership := models.GroupMembership{GroupID: group.ID, MemberID: owner.ID, Enabled: true}
	membership.ID = uuid.New()
	if r := database.Instance.Create(&membership); r.Error != nil {
		t.Fatalf("failed to create membership: %v", r.Error)
	}

	session := startMCPTestServer(t, owner.ID, "mcp:groups.read")

	result, err := session.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: "list_groups"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool call reported an error: %+v", result.Content)
	}

	text := firstText(t, result)
	var got []groupOut
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("failed to parse tool result: %v (raw: %s)", err, text)
	}
	if len(got) != 1 || got[0].Name != "Family" {
		t.Errorf("got %+v, want exactly the one 'Family' group", got)
	}
}

func firstText(t *testing.T, result *sdkmcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("tool result has no content")
	}
	text, ok := result.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("first content block is %T, want *mcp.TextContent", result.Content[0])
	}
	return text.Text
}

func boolPtr(b bool) *bool { return &b }

// createMCPTestUser inserts an enabled user with a unique e-mail (required,
// NOT NULL in the schema).
func createMCPTestUser(t *testing.T, firstName string) models.User {
	t.Helper()
	email := uuid.NewString() + "@example.com"
	user := models.User{FirstName: firstName, Email: &email, Enabled: boolPtr(true)}
	user.ID = uuid.New()
	if r := database.Instance.Create(&user); r.Error != nil {
		t.Fatalf("failed to create user: %v", r.Error)
	}
	return user
}

func TestBuildServerRegistersTools(t *testing.T) {
	server := buildServer()
	if server == nil {
		t.Fatal("buildServer returned nil")
	}
}

func TestJSONResult(t *testing.T) {
	result := jsonResult([]wishlistOut{{ID: "1", Name: "Test"}})
	text := firstText(t, result)
	var got []wishlistOut
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatalf("failed to parse jsonResult output: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Test" {
		t.Errorf("got %+v, want [{ID:1 Name:Test}]", got)
	}
}

func TestJSONResultUnmarshalableFallsBackToEmptyArray(t *testing.T) {
	// A Go channel cannot be marshaled to JSON; jsonResult should fall back to
	// "[]" rather than panicking or propagating the error.
	result := jsonResult(make(chan int))
	if firstText(t, result) != "[]" {
		t.Errorf("text = %q, want []", firstText(t, result))
	}
}
