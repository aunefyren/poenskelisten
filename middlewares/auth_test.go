package middlewares

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testAPIResource = "https://wishlist.example.com/api"
const testMCPResource = "https://wishlist.example.com/mcp"

// setupMiddlewareConfig installs an OAuth signing key + issuer + API resource so
// the resource-server validation works.
func setupMiddlewareConfig(t *testing.T) {
	t.Helper()

	keyPEM, kid, err := config.GenerateOAuthSigningKey(config.OAuthAlgES256)
	if err != nil {
		t.Fatalf("failed to generate OAuth key: %v", err)
	}
	config.ConfigFile.OAuthSigningKey = keyPEM
	config.ConfigFile.OAuthSigningKeyID = kid
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"

	if logger.Log == nil {
		logger.Log = logrus.New()
		logger.Log.SetOutput(io.Discard)
	}
}

// setupMiddlewareDB points database.Instance at a fresh in-memory SQLite DB
// with the users table, which AuthFunction reads the logout marker from.
func setupMiddlewareDB(t *testing.T) {
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
	if err := instance.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	original := database.Instance
	database.Instance = instance
	t.Cleanup(func() { database.Instance = original })
}

// createMiddlewareUser inserts an enabled user, so tokens minted for it pass
// the revocation check.
func createMiddlewareUser(t *testing.T) uuid.UUID {
	t.Helper()
	email := uuid.NewString() + "@example.com"
	enabled := true
	user := models.User{FirstName: "Test", Email: &email, Enabled: &enabled}
	user.ID = uuid.New()
	if err := database.Instance.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user.ID
}

func newContext(authHeader string) *gin.Context {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if authHeader != "" {
		ctx.Request.Header.Set("Authorization", authHeader)
	}
	return ctx
}

func apiTokenForUser(t *testing.T, userID uuid.UUID, admin bool) string {
	t.Helper()
	token, err := auth.GenerateOAuthAccessToken(userID, models.FirstPartyClientID, testAPIResource, "openid", admin, true)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	return token
}

func TestAuthFunctionNoToken(t *testing.T) {
	setupMiddlewareConfig(t)

	success, _, status := AuthFunction(newContext(""), false)
	if success || status != http.StatusUnauthorized {
		t.Errorf("no-token: success=%v status=%d, want false/401", success, status)
	}
}

func TestAuthFunctionInvalidToken(t *testing.T) {
	setupMiddlewareConfig(t)

	success, _, status := AuthFunction(newContext("not-a-real-token"), false)
	if success || status != http.StatusUnauthorized {
		t.Errorf("invalid: success=%v status=%d, want false/401", success, status)
	}
}

func TestAuthFunctionValidNonAdmin(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)

	token := apiTokenForUser(t, createMiddlewareUser(t), false)
	success, errStr, status := AuthFunction(newContext(token), false)
	if !success || status != http.StatusOK {
		t.Errorf("valid token rejected: %q (status %d)", errStr, status)
	}
}

func TestAuthFunctionAdminRequiredButNotAdmin(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)

	token := apiTokenForUser(t, createMiddlewareUser(t), false)
	success, _, status := AuthFunction(newContext(token), true)
	if success || status != http.StatusForbidden {
		t.Errorf("non-admin on admin route: success=%v status=%d, want false/403", success, status)
	}
}

func TestAuthFunctionAdminSuccess(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)

	token := apiTokenForUser(t, createMiddlewareUser(t), true)
	success, _, status := AuthFunction(newContext(token), true)
	if !success || status != http.StatusOK {
		t.Errorf("admin token rejected on admin route (status %d)", status)
	}
}

func TestAuthFunctionWrongAudience(t *testing.T) {
	setupMiddlewareConfig(t)

	// A token minted for the MCP resource must not authenticate the API.
	token, err := auth.GenerateOAuthAccessToken(uuid.New(), "mcp-client", testMCPResource, "mcp:wishlists.read", false, true)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	success, _, status := AuthFunction(newContext(token), false)
	if success || status != http.StatusUnauthorized {
		t.Errorf("wrong-audience token accepted: success=%v status=%d", success, status)
	}
}

// An API-audience token issued to any client other than the built-in web app
// must be refused by every API entry point, even though its signature and
// audience are valid.
func TestAPITokenFromThirdPartyClientRejected(t *testing.T) {
	setupMiddlewareConfig(t)

	for _, clientID := range []string{"third-party-client", ""} {
		token, err := auth.GenerateOAuthAccessToken(uuid.New(), clientID, testAPIResource, "openid profile", true, true)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		if success, _, status := AuthFunction(newContext(token), true); success || status != http.StatusUnauthorized {
			t.Errorf("client %q: AuthFunction success=%v status=%d, want false/401", clientID, success, status)
		}
		if _, err := GetAuthUsername(token); err == nil {
			t.Errorf("client %q: GetAuthUsername accepted the token", clientID)
		}
		if _, err := GetTokenClaims(token); err == nil {
			t.Errorf("client %q: GetTokenClaims accepted the token", clientID)
		}
	}
}

func TestGetAuthUsername(t *testing.T) {
	setupMiddlewareConfig(t)

	if _, err := GetAuthUsername(""); err == nil {
		t.Error("GetAuthUsername(\"\") returned no error, want error")
	}

	userID := uuid.New()
	token := apiTokenForUser(t, userID, false)
	got, err := GetAuthUsername(token)
	if err != nil {
		t.Fatalf("GetAuthUsername error: %v", err)
	}
	if got != userID {
		t.Errorf("GetAuthUsername = %v, want %v", got, userID)
	}

	if _, err := GetAuthUsername("garbage"); err == nil {
		t.Error("GetAuthUsername(garbage) returned no error, want error")
	}
}

func TestGetTokenClaims(t *testing.T) {
	setupMiddlewareConfig(t)

	userID := uuid.New()
	token := apiTokenForUser(t, userID, true)
	claims, err := GetTokenClaims(token)
	if err != nil {
		t.Fatalf("GetTokenClaims error: %v", err)
	}
	if claims.Subject != userID.String() || !claims.Admin {
		t.Errorf("claims mismatch: %+v", claims)
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	setupMiddlewareConfig(t)

	ctx := newContext("")
	Auth(false)(ctx)

	if !ctx.IsAborted() {
		t.Error("expected the request to be aborted without a token")
	}
	if ctx.Writer.Status() != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", ctx.Writer.Status())
	}
}

func TestAuthMiddlewareAllowsValidToken(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)

	ctx := newContext("Bearer " + apiTokenForUser(t, createMiddlewareUser(t), false))
	Auth(false)(ctx)

	if ctx.IsAborted() {
		t.Error("expected a valid token to not abort the request")
	}
}

func TestAuthMiddlewareRejectsNonAdminForAdminRoute(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)

	ctx := newContext("Bearer " + apiTokenForUser(t, createMiddlewareUser(t), false))
	Auth(true)(ctx)

	if !ctx.IsAborted() {
		t.Error("expected a non-admin token to be rejected on an admin-only route")
	}
	if ctx.Writer.Status() != http.StatusForbidden {
		t.Errorf("status = %d, want 403", ctx.Writer.Status())
	}
}

func TestAuthMiddlewareAllowsAdminForAdminRoute(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)

	ctx := newContext("Bearer " + apiTokenForUser(t, createMiddlewareUser(t), true))
	Auth(true)(ctx)

	if ctx.IsAborted() {
		t.Error("expected an admin token to be allowed on an admin-only route")
	}
}

func TestGetTokenClaimsEmptyToken(t *testing.T) {
	claims, err := GetTokenClaims("")
	if err == nil || err.Error() != "no Authorization header given" || claims != nil {
		t.Errorf("GetTokenClaims(\"\") = (%v, %v), want nil claims and 'no Authorization header given'", claims, err)
	}
}

// A token issued before the user was signed out everywhere (a password reset
// or change, "sign out everywhere") is refused, while one issued after is not.
func TestAuthFunctionRefusesTokenIssuedBeforeSignOut(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)
	userID := createMiddlewareUser(t)
	oldToken := apiTokenForUser(t, userID, false)

	if err := database.SetUserSessionsInvalidatedAt(userID, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("failed to stamp logout marker: %v", err)
	}
	if success, _, status := AuthFunction(newContext(oldToken), false); success || status != http.StatusUnauthorized {
		t.Errorf("token from before the sign-out: success=%v status=%d, want false/401", success, status)
	}

	if err := database.SetUserSessionsInvalidatedAt(userID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("failed to stamp logout marker: %v", err)
	}
	if success, errStr, status := AuthFunction(newContext(apiTokenForUser(t, userID, false)), false); !success {
		t.Errorf("token from after the sign-out rejected: %q (status %d)", errStr, status)
	}
}

// A token whose account is gone or disabled no longer works either.
func TestAuthFunctionRefusesTokenForMissingUser(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)

	token := apiTokenForUser(t, uuid.New(), false)
	if success, _, status := AuthFunction(newContext(token), false); success || status != http.StatusUnauthorized {
		t.Errorf("token for an unknown user: success=%v status=%d, want false/401", success, status)
	}
}

// A failed lookup is a server error, not a reason to tell the client its
// token is bad.
func TestAuthFunctionDatabaseFailure(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)
	token := apiTokenForUser(t, createMiddlewareUser(t), false)
	if err := database.Instance.Migrator().DropTable(&models.User{}); err != nil {
		t.Fatalf("failed to drop users table: %v", err)
	}

	if success, _, status := AuthFunction(newContext(token), false); success || status != http.StatusInternalServerError {
		t.Errorf("success=%v status=%d, want false/500", success, status)
	}
}

func TestCheckNotSignedOutBadSubject(t *testing.T) {
	claims := &auth.OAuthClaims{RegisteredClaims: jwt.RegisteredClaims{Subject: "not-a-uuid"}}
	if err := CheckNotSignedOut(claims); !errors.Is(err, ErrTokenRevoked) {
		t.Errorf("err = %v, want ErrTokenRevoked", err)
	}
}
