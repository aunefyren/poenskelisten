package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// --- fixtures scoped to this file (avoid colliding with parallel test files) ---

// oauthServerTestSetup wires up both the OAuth signing key (enableOAuth) and the
// HS256 secret GenerateSSOToken needs, so both access tokens and SSO cookies can
// be minted.
func oauthServerTestSetup(t *testing.T) {
	t.Helper()
	enableOAuth(t)
	key, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	config.ConfigFile.PrivateKey = key
	config.ConfigFile.PoenskelistenName = "TestApp"
}

func oauthServerTestNewClient(t *testing.T, firstParty, public bool, scopes []string, redirectURI string) models.OAuthClient {
	t.Helper()
	enabled := true
	client := models.OAuthClient{
		ClientID:                uuid.NewString(),
		ClientName:              "Test Client",
		RedirectURIs:            []string{redirectURI},
		Scopes:                  scopes,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: "none",
		IsPublic:                public,
		IsFirstParty:            firstParty,
		Registered:              true,
		Enabled:                 &enabled,
	}
	created, err := database.CreateOAuthClient(client)
	if err != nil {
		t.Fatalf("failed to create oauth client: %v", err)
	}
	return created
}

// oauthServerTestNewConfidentialClient creates a non-public client with a known
// plaintext secret (hashed for storage), for authenticateClient tests.
func oauthServerTestNewConfidentialClient(t *testing.T, secret, redirectURI string) models.OAuthClient {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash secret: %v", err)
	}
	hashStr := string(hash)
	enabled := true
	client := models.OAuthClient{
		ClientID:                uuid.NewString(),
		ClientName:              "Confidential Client",
		ClientSecretHash:        &hashStr,
		RedirectURIs:            []string{redirectURI},
		Scopes:                  []string{"openid", "profile", "email"},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: "client_secret_post",
		IsPublic:                false,
		IsFirstParty:            false,
		Registered:              true,
		Enabled:                 &enabled,
	}
	created, err := database.CreateOAuthClient(client)
	if err != nil {
		t.Fatalf("failed to create confidential oauth client: %v", err)
	}
	return created
}

func oauthServerTestSSOCookie(t *testing.T, userID uuid.UUID) *http.Cookie {
	t.Helper()
	token, err := auth.GenerateSSOToken(userID)
	if err != nil {
		t.Fatalf("failed to generate sso token: %v", err)
	}
	return &http.Cookie{Name: ssoCookieName, Value: token}
}

// oauthServerTestPKCE returns a verifier/S256-challenge pair.
func oauthServerTestPKCE() (verifier string, challenge string) {
	verifier = "test-verifier-" + uuid.NewString()
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return
}

// oauthServerTestIssueCode drives a full APIOAuthAuthorize request for a
// first-party (auto-approved) client and returns the issued authorization code
// plus the PKCE verifier needed to redeem it.
func oauthServerTestIssueCode(t *testing.T, client models.OAuthClient, user models.User) (code string, verifier string) {
	t.Helper()
	verifier, challenge := oauthServerTestPKCE()

	q := url.Values{}
	q.Set("client_id", client.ClientID)
	q.Set("redirect_uri", client.RedirectURIs[0])
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(client.Scopes, " "))
	q.Set("state", "xyz")
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")

	cookie := oauthServerTestSSOCookie(t, user.ID)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/oauth/authorize?"+q.Encode(), nil)
	ctx.Request.AddCookie(cookie)
	APIOAuthAuthorize(ctx)
	ctx.Writer.WriteHeaderNow()

	var loc *url.URL
	var err error
	if w.Code == http.StatusOK {
		// Non-first-party client: the authorize step only rendered the
		// consent page. Submit it (allow) to actually get a code.
		form := url.Values{
			"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
			"scope": {strings.Join(client.Scopes, " ")}, "state": {"xyz"},
			"code_challenge": {challenge}, "resource": {q.Get("resource")}, "action": {"allow"},
		}
		_, _, consentW := postForm(APIOAuthConsent, "/oauth/consent", form, cookie)
		w = consentW
	}
	if w.Code != http.StatusFound {
		t.Fatalf("authorize/consent status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	loc, err = url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatalf("failed to parse redirect location: %v", err)
	}
	code = loc.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in redirect location: %s", loc.String())
	}
	return code, verifier
}

// postForm calls handler directly (bypassing the router). Bypassing gin's
// Engine also bypasses the WriteHeaderNow() flush it normally does at the end
// of the request cycle, so a redirect on a POST (which - unlike GET - writes
// no body) would otherwise never reach the ResponseRecorder's status code;
// force the flush here so w.Code reflects what the handler actually set.
func postForm(handler gin.HandlerFunc, path string, form url.Values, cookies ...*http.Cookie) (int, map[string]interface{}, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	ctx.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		ctx.Request.AddCookie(c)
	}
	handler(ctx)
	ctx.Writer.WriteHeaderNow()

	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body, w
}

func getAuthorize(query url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/oauth/authorize?"+query.Encode(), nil)
	for _, c := range cookies {
		ctx.Request.AddCookie(c)
	}
	APIOAuthAuthorize(ctx)
	ctx.Writer.WriteHeaderNow()
	return w
}

// --- APIOAuthAuthorize ---

func TestOAuthAuthorizeUnknownClient(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)

	q := url.Values{"client_id": {"does-not-exist"}, "redirect_uri": {"https://client.example/cb"}}
	w := getAuthorize(q)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestOAuthAuthorizeInvalidRedirectURI(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")

	q := url.Values{"client_id": {client.ClientID}, "redirect_uri": {"https://evil.example/cb"}}
	w := getAuthorize(q)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestOAuthAuthorizeNotLoggedIn(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	_, challenge := oauthServerTestPKCE()

	q := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"response_type": {"code"}, "scope": {"openid"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	w := getAuthorize(q)
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/login?next=") {
		t.Errorf("Location = %q, want a /login?next= redirect", loc)
	}
}

func TestOAuthAuthorizeUnsupportedResponseType(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	_, challenge := oauthServerTestPKCE()

	q := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"response_type": {"token"}, "scope": {"openid"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "state": {"s1"},
	}
	w := getAuthorize(q, oauthServerTestSSOCookie(t, user.ID))
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("error") != "unsupported_response_type" {
		t.Errorf("error = %q, want unsupported_response_type", loc.Query().Get("error"))
	}
	if loc.Query().Get("state") != "s1" {
		t.Errorf("state = %q, want s1 echoed back", loc.Query().Get("state"))
	}
}

func TestOAuthAuthorizeMissingPKCEChallenge(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	q := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"response_type": {"code"}, "scope": {"openid"},
	}
	w := getAuthorize(q, oauthServerTestSSOCookie(t, user.ID))
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("error") != "invalid_request" {
		t.Errorf("error = %q, want invalid_request; status=%d", loc.Query().Get("error"), w.Code)
	}
}

func TestOAuthAuthorizeInvalidScope(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	_, challenge := oauthServerTestPKCE()

	q := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"response_type": {"code"}, "scope": {"mcp:wishlists.write"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	w := getAuthorize(q, oauthServerTestSSOCookie(t, user.ID))
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("error") != "invalid_scope" {
		t.Errorf("error = %q, want invalid_scope", loc.Query().Get("error"))
	}
}

func TestOAuthAuthorizeInvalidTargetResource(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	_, challenge := oauthServerTestPKCE()

	q := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"response_type": {"code"}, "scope": {"openid"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
		"resource": {"https://not-a-real-resource.example"},
	}
	w := getAuthorize(q, oauthServerTestSSOCookie(t, user.ID))
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("error") != "invalid_target" {
		t.Errorf("error = %q, want invalid_target", loc.Query().Get("error"))
	}
}

func TestOAuthAuthorizeFirstPartyAutoApprove(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, true, true, []string{"openid", "profile"}, "https://client.example/cb")
	user := createTestUser(t)

	code, _ := oauthServerTestIssueCode(t, client, user)
	if code == "" {
		t.Fatal("expected a non-empty authorization code")
	}
}

func TestOAuthAuthorizeThirdPartyShowsConsent(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid", "profile"}, "https://client.example/cb")
	user := createTestUser(t)
	_, challenge := oauthServerTestPKCE()

	q := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"response_type": {"code"}, "scope": {"openid profile"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	w := getAuthorize(q, oauthServerTestSSOCookie(t, user.ID))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (consent page); body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "wants to access your account") {
		t.Error("expected the consent page body to explain what's being requested")
	}
}

// --- APIOAuthConsent ---

func TestOAuthConsentNotLoggedIn(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/oauth/consent", strings.NewReader(""))
	ctx.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	APIOAuthConsent(ctx)
	ctx.Writer.WriteHeaderNow()

	if w.Code != http.StatusFound || w.Header().Get("Location") != "/login" {
		t.Fatalf("status=%d location=%q, want 302 to /login", w.Code, w.Header().Get("Location"))
	}
}

func TestOAuthConsentDeny(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	form := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"scope": {"openid"}, "state": {"s1"}, "action": {"deny"},
	}
	code, _, w := postForm(APIOAuthConsent, "/oauth/consent", form, oauthServerTestSSOCookie(t, user.ID))
	if code != http.StatusFound {
		t.Fatalf("status = %d, want 302", code)
	}
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("error") != "access_denied" {
		t.Errorf("error = %q, want access_denied", loc.Query().Get("error"))
	}
}

func TestOAuthConsentAllowIssuesCode(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid", "profile"}, "https://client.example/cb")
	user := createTestUser(t)
	_, challenge := oauthServerTestPKCE()

	form := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"scope": {"openid profile"}, "state": {"s1"}, "action": {"allow"},
		"code_challenge": {challenge},
	}
	code, _, w := postForm(APIOAuthConsent, "/oauth/consent", form, oauthServerTestSSOCookie(t, user.ID))
	if code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", code, w.Body.String())
	}
	loc, _ := url.Parse(w.Header().Get("Location"))
	if loc.Query().Get("code") == "" {
		t.Error("expected an authorization code in the redirect")
	}

	consent, found, err := database.GetConsent(user.ID, client.ClientID)
	if err != nil || !found {
		t.Fatalf("expected consent to be stored, found=%v err=%v", found, err)
	}
	if !consent.Covers([]string{"openid", "profile"}) {
		t.Errorf("stored consent %v does not cover the requested scopes", consent.Scopes)
	}
}

func TestOAuthConsentInvalidClient(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	user := createTestUser(t)

	form := url.Values{"client_id": {"nope"}, "redirect_uri": {"https://x.example/cb"}, "action": {"allow"}}
	code, _, _ := postForm(APIOAuthConsent, "/oauth/consent", form, oauthServerTestSSOCookie(t, user.ID))
	if code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}

// --- APIOAuthToken ---

func TestOAuthTokenUnsupportedGrantType(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{"grant_type": {"password"}})
	if code != http.StatusBadRequest || body["error"] != "unsupported_grant_type" {
		t.Errorf("status=%d body=%v, want 400 unsupported_grant_type", code, body)
	}
}

func TestOAuthTokenAuthorizationCodeSuccessPublicClient(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid", "profile"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	}
	code, body, w := postForm(APIOAuthToken, "/oauth/token", form)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["access_token"] == nil || body["access_token"] == "" {
		t.Error("expected an access_token")
	}
	if body["id_token"] == nil {
		t.Error("expected an id_token for the openid scope")
	}
	// Non-first-party client: refresh token travels in the body, not a cookie.
	if body["refresh_token"] == nil || body["refresh_token"] == "" {
		t.Error("expected a refresh_token in the body for a non-first-party client")
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == refreshCookieName {
			t.Error("did not expect a refresh cookie for a non-first-party client")
		}
	}
}

func TestOAuthTokenAuthorizationCodeSuccessFirstPartyClient(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	}
	code, body, w := postForm(APIOAuthToken, "/oauth/token", form)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["refresh_token"] != nil {
		t.Error("first-party client should get the refresh token via cookie, not the body")
	}
	found := false
	for _, c := range w.Result().Cookies() {
		if c.Name == refreshCookieName && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected the refresh cookie to be set for a first-party client")
	}
}

func TestOAuthTokenAuthorizationCodeInvalidCode(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {"not-a-real-code"},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {"whatever"},
	}
	code, body, _ := postForm(APIOAuthToken, "/oauth/token", form)
	if code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want 400 invalid_grant", code, body)
	}
}

func TestOAuthTokenAuthorizationCodeBadPKCE(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, _ := oauthServerTestIssueCode(t, client, user)

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {"wrong-verifier"},
	}
	code, body, _ := postForm(APIOAuthToken, "/oauth/token", form)
	if code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want 400 invalid_grant for bad PKCE", code, body)
	}
}

func TestOAuthTokenAuthorizationCodeBindingMismatch(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {"https://different.example/cb"}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	}
	code, body, _ := postForm(APIOAuthToken, "/oauth/token", form)
	if code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want 400 invalid_grant for redirect_uri mismatch", code, body)
	}
}

func TestOAuthTokenRefreshSuccess(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)

	_, tokenBody, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	})
	refreshToken, _ := tokenBody["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatal("expected a refresh token from the authorization_code exchange")
	}

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}, "client_id": {client.ClientID},
	})
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["access_token"] == nil || body["access_token"] == "" {
		t.Error("expected a new access_token")
	}
}

func TestOAuthTokenRefreshMissingToken(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "client_id": {client.ClientID},
	})
	if code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want 400 invalid_grant", code, body)
	}
}

func TestOAuthTokenRefreshInvalidToken(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {"garbage"}, "client_id": {client.ClientID},
	})
	if code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want 400 invalid_grant", code, body)
	}
}

// --- authenticateClient (via the token endpoint) ---

func TestOAuthTokenUnknownClient(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {"x"}, "client_id": {"does-not-exist"}, "code_verifier": {"y"},
	})
	if code != http.StatusUnauthorized || body["error"] != "invalid_client" {
		t.Errorf("status=%d body=%v, want 401 invalid_client", code, body)
	}
}

func TestOAuthTokenConfidentialClientBadSecret(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewConfidentialClient(t, "correct-secret", "https://client.example/cb")

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {"x"}, "client_id": {client.ClientID},
		"client_secret": {"wrong-secret"}, "code_verifier": {"y"},
	})
	if code != http.StatusUnauthorized || body["error"] != "invalid_client" {
		t.Errorf("status=%d body=%v, want 401 invalid_client", code, body)
	}
}

func TestOAuthTokenConfidentialClientGoodSecret(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewConfidentialClient(t, "correct-secret", "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"client_secret": {"correct-secret"}, "code_verifier": {verifier},
	})
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with the correct secret; body=%v", code, body)
	}
}

// --- APIOAuthRevoke ---

func TestOAuthRevokeWithFormToken(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)
	_, tokenBody, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	})
	refreshToken := tokenBody["refresh_token"].(string)

	code, body, w := postForm(APIOAuthRevoke, "/oauth/revoke", url.Values{"token": {refreshToken}})
	if code != http.StatusOK || body["message"] != "Revoked." {
		t.Fatalf("status=%d body=%v, want 200 Revoked.", code, body)
	}
	sawCleared := false
	for _, c := range w.Result().Cookies() {
		if c.Name == refreshCookieName && c.MaxAge < 0 {
			sawCleared = true
		}
	}
	if !sawCleared {
		t.Error("expected the refresh cookie to be cleared")
	}

	// The revoked refresh token must no longer work.
	code2, body2, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}, "client_id": {client.ClientID},
	})
	if code2 != http.StatusBadRequest || body2["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want the revoked refresh token to be rejected", code2, body2)
	}
}

func TestOAuthRevokeNoToken(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)

	code, body, _ := postForm(APIOAuthRevoke, "/oauth/revoke", url.Values{})
	if code != http.StatusOK || body["message"] != "Revoked." {
		t.Errorf("status=%d body=%v, want 200 Revoked. even with nothing to revoke", code, body)
	}
}

// --- small pure-function coverage ---

func TestScopeContains(t *testing.T) {
	if !scopeContains("openid profile", "openid") {
		t.Error("expected scopeContains to find openid")
	}
	if scopeContains("openid profile", "email") {
		t.Error("did not expect scopeContains to find email")
	}
}

func TestDerefString(t *testing.T) {
	if derefString(nil) != "" {
		t.Error("derefString(nil) should be empty")
	}
	s := "hello"
	if derefString(&s) != "hello" {
		t.Error("derefString should dereference a non-nil pointer")
	}
}

func TestDisplayName(t *testing.T) {
	u := models.User{FirstName: "Ada", LastName: "Lovelace"}
	if displayName(u) != "Ada Lovelace" {
		t.Errorf("displayName = %q", displayName(u))
	}
}

func TestHTMLEscape(t *testing.T) {
	if htmlEscape(`<b>"a" & 'b'</b>`) != "&lt;b&gt;&quot;a&quot; &amp; &#39;b&#39;&lt;/b&gt;" {
		t.Errorf("htmlEscape produced unexpected output: %q", htmlEscape(`<b>"a" & 'b'</b>`))
	}
}
