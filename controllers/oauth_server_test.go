package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/bcrypt"
)

// --- fixtures scoped to this file (avoid colliding with parallel test files) ---

// oauthServerTestSetup wires up both the OAuth signing key (enableOAuth) and the
// HS256 secret GenerateSSOToken needs, so both access tokens and SSO cookies can
// be minted.
func oauthServerTestSetup(t *testing.T) {
	t.Helper()
	restoreConfig(t)
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

// A first-party redirect mismatch is almost always an operator opening the app
// on an origin that isn't configured, so it's logged with a hint; a third-party
// mismatch is the client's problem and isn't.
func TestOAuthAuthorizeInvalidRedirectURIWarnsForFirstParty(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	hook := logrusTest.NewLocal(logger.Log)
	t.Cleanup(func() { logger.Log.ReplaceHooks(make(logrus.LevelHooks)) })

	firstParty := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://wish.example.com/oauth/callback")
	thirdParty := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")

	warned := func() bool {
		for _, entry := range hook.AllEntries() {
			if entry.Level == logrus.WarnLevel && strings.Contains(entry.Message, "additionalurls") {
				return true
			}
		}
		return false
	}

	getAuthorize(url.Values{"client_id": {thirdParty.ClientID}, "redirect_uri": {"https://evil.example/cb"}})
	if warned() {
		t.Error("third-party redirect mismatch logged the additionalurls hint")
	}

	w := getAuthorize(url.Values{"client_id": {firstParty.ClientID}, "redirect_uri": {"http://192.168.1.10:8080/oauth/callback"}})
	if w.Code != http.StatusBadRequest || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("status = %d content-type=%q, want a 400 HTML page", w.Code, w.Header().Get("Content-Type"))
	}
	for _, want := range []string{"http://192.168.1.10:8080", config.OAuthIssuer(), "additionalurls"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("error page doesn't mention %q:\n%s", want, w.Body.String())
		}
	}
	if !warned() {
		t.Error("first-party redirect mismatch didn't log the additionalurls hint")
	}
}

// The redirect URI on the error page comes straight from the query string.
func TestOAuthAuthorizeOriginNotAllowedPageEscapes(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://wish.example.com/oauth/callback")

	for _, redirectURI := range []string{`http://x"><script>alert(1)</script>/oauth/callback`, `"><script>alert(1)</script>`} {
		w := getAuthorize(url.Values{"client_id": {client.ClientID}, "redirect_uri": {redirectURI}})
		if strings.Contains(w.Body.String(), "<script>") {
			t.Errorf("redirect URI %q rendered unescaped:\n%s", redirectURI, w.Body.String())
		}
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

// authorizeFirstPartyQuery builds a valid first-party authorize request.
func authorizeFirstPartyQuery(client models.OAuthClient) url.Values {
	_, challenge := oauthServerTestPKCE()
	q := url.Values{}
	q.Set("client_id", client.ClientID)
	q.Set("redirect_uri", client.RedirectURIs[0])
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(client.Scopes, " "))
	q.Set("state", "xyz")
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return q
}

func TestOAuthAuthorizeEnforcedMFARedirectsToEnroll(t *testing.T) {
	restoreConfig(t)
	setupControllersDB(t)
	oauthServerTestSetup(t)
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })
	config.ConfigFile.MFAEnforced = true
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	w := getAuthorize(authorizeFirstPartyQuery(client), oauthServerTestSSOCookie(t, user.ID))
	if loc := w.Header().Get("Location"); w.Code != http.StatusFound || loc != "/enroll" {
		t.Errorf("status = %d, Location = %q; want 302 to /enroll", w.Code, loc)
	}
}

// With password login disabled, local MFA protects nothing (a linked local
// account signs in via the IdP), so enforcement mustn't force enrollment.
func TestOAuthAuthorizeEnforcedMFASkippedWhenLocalLoginDisabled(t *testing.T) {
	restoreConfig(t)
	setupControllersDB(t)
	oauthServerTestSetup(t)
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })
	config.ConfigFile.MFAEnforced = true
	config.ConfigFile.LocalLoginDisabled = true
	config.ConfigFile.OIDCEnabled = true
	config.ConfigFile.OIDCIssuerURL = "https://auth.example.com"
	config.ConfigFile.OIDCClientID = "poenskelisten"
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	w := getAuthorize(authorizeFirstPartyQuery(client), oauthServerTestSSOCookie(t, user.ID))
	loc, _ := url.Parse(w.Header().Get("Location"))
	if w.Code != http.StatusFound || loc == nil || loc.Query().Get("code") == "" {
		t.Errorf("status = %d, Location = %q; want a code issued without enrollment", w.Code, w.Header().Get("Location"))
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

func TestIssueAuthorizationCodeDatabaseFailure(t *testing.T) {
	// Migrate without the AuthorizationCode table so database.CreateAuthorizationCode fails.
	setupControllersDB(t, &models.User{}, &models.OAuthClient{})
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	issueAuthorizationCode(ctx, client, user, client.RedirectURIs[0], []string{"openid"}, "", "challenge", "state123")

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (error redirect); body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "error=server_error") {
		t.Errorf("Location = %q, want a server_error redirect", loc)
	}
	if !strings.Contains(loc, "state=state123") {
		t.Errorf("Location = %q, want the state param preserved", loc)
	}
}

func TestHandleRefreshTokenGrantUserNotFound(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)

	// Exchange the code for a real refresh token.
	exchangeForm := url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	}
	exchangeCode, exchangeBody, _ := postForm(APIOAuthToken, "/oauth/token", exchangeForm)
	if exchangeCode != http.StatusOK {
		t.Fatalf("token exchange status = %d, want 200; body=%v", exchangeCode, exchangeBody)
	}
	refreshToken, _ := exchangeBody["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatal("expected a refresh_token from the exchange")
	}

	// The session now exists, but the user behind it is gone.
	if result := database.Instance.Unscoped().Delete(&models.User{}, "id = ?", user.ID); result.Error != nil {
		t.Fatalf("failed to hard-delete user: %v", result.Error)
	}

	refreshForm := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {client.ClientID},
		"refresh_token": {refreshToken},
	}
	code, body, _ := postForm(APIOAuthToken, "/oauth/token", refreshForm)
	if code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 when the session's user no longer exists; body=%v", code, body)
	}
	if body["error"] != "invalid_grant" {
		t.Errorf("error = %v, want invalid_grant", body["error"])
	}
}

// --- further branches ---

// oauthServerTestExchange runs the authorization_code grant for a freshly
// issued code and returns the response, failing the test on a non-200.
func oauthServerTestExchange(t *testing.T, client models.OAuthClient, user models.User) (map[string]interface{}, *httptest.ResponseRecorder) {
	t.Helper()
	authCode, verifier := oauthServerTestIssueCode(t, client, user)
	code, body, w := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	})
	if code != http.StatusOK {
		t.Fatalf("token exchange status = %d, want 200; body=%v", code, body)
	}
	return body, w
}

// oauthServerTestCookie returns the named cookie set on the response, or nil.
func oauthServerTestCookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// oauthServerTestClearSigningKey removes the OAuth signing key so access-token
// minting fails, restoring it afterwards.
func oauthServerTestClearSigningKey(t *testing.T) {
	t.Helper()
	restoreConfig(t)
	orig := config.ConfigFile.OAuthSigningKey
	config.ConfigFile.OAuthSigningKey = ""
	t.Cleanup(func() { config.ConfigFile.OAuthSigningKey = orig })
}

func TestResolveSSOUserRejections(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	invalidated := createTestUser(t)
	cookieBeforeLogout := oauthServerTestSSOCookie(t, invalidated.ID)
	if err := database.SetUserSessionsInvalidatedAt(invalidated.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("failed to stamp logout marker: %v", err)
	}

	cases := []struct {
		name   string
		cookie *http.Cookie
	}{
		{"no cookie", nil},
		{"garbage token", &http.Cookie{Name: ssoCookieName, Value: "garbage"}},
		{"unknown user", oauthServerTestSSOCookie(t, uuid.New())},
		{"issued before global logout", cookieBeforeLogout},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/oauth/authorize", nil)
			if c.cookie != nil {
				ctx.Request.AddCookie(c.cookie)
			}
			if user, ok := resolveSSOUser(ctx); ok {
				t.Errorf("resolveSSOUser accepted the session for %v", user.ID)
			}
		})
	}
}

func TestOAuthAuthorizeClientLookupFails(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")
	breakControllersDB(t)

	w := getAuthorize(authorizeFirstPartyQuery(client))
	if w.Code != http.StatusInternalServerError || w.Body.String() != "Authorization error." {
		t.Errorf("status = %d body=%q, want 500 'Authorization error.'", w.Code, w.Body.String())
	}
}

func TestOAuthAuthorizeUnverifiedUserRedirectsToVerify(t *testing.T) {
	restoreConfig(t)
	setupControllersDB(t)
	oauthServerTestSetup(t)
	origSMTP := config.ConfigFile.SMTPEnabled
	config.ConfigFile.SMTPEnabled = true
	t.Cleanup(func() { config.ConfigFile.SMTPEnabled = origSMTP })
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	user.Verified = boolPtr(false)
	if _, err := database.UpdateUserInDB(user); err != nil {
		t.Fatalf("failed to unverify user: %v", err)
	}

	w := getAuthorize(authorizeFirstPartyQuery(client), oauthServerTestSSOCookie(t, user.ID))
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/verify" {
		t.Errorf("status = %d Location=%q, want 302 to /verify", w.Code, w.Header().Get("Location"))
	}
}

func TestOAuthConsentAllowInvalidScope(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	// "email" is a real scope, but not one this client may request.
	form := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"scope": {"email"}, "state": {"s1"}, "action": {"allow"},
	}
	code, _, w := postForm(APIOAuthConsent, "/oauth/consent", form, oauthServerTestSSOCookie(t, user.ID))
	loc, _ := url.Parse(w.Header().Get("Location"))
	if code != http.StatusFound || loc.Query().Get("error") != "invalid_scope" || loc.Query().Get("state") != "s1" {
		t.Errorf("status = %d Location=%q, want 302 with error=invalid_scope&state=s1", code, w.Header().Get("Location"))
	}
	if _, found, _ := database.GetConsent(user.ID, client.ClientID); found {
		t.Error("no consent should be stored for a rejected scope")
	}
}

func TestOAuthConsentStoreFails(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	failDBOperation(t, "create", "o_auth_consents", 0)

	form := url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]},
		"scope": {"openid"}, "state": {"s1"}, "action": {"allow"},
	}
	code, _, w := postForm(APIOAuthConsent, "/oauth/consent", form, oauthServerTestSSOCookie(t, user.ID))
	if code != http.StatusInternalServerError || w.Body.String() != "Failed to store consent." {
		t.Errorf("status = %d body=%q, want 500 'Failed to store consent.'", code, w.Body.String())
	}
}

func TestIssueAuthorizationCodeUnparseableRedirect(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	issueAuthorizationCode(ctx, client, user, "http://[::1", []string{"openid"}, "", "challenge", "")

	if w.Code != http.StatusBadRequest || w.Body.String() != "Invalid redirect URI." {
		t.Errorf("status = %d body=%q, want 400 'Invalid redirect URI.'", w.Code, w.Body.String())
	}
}

func TestRedirectAuthErrorUnparseableRedirect(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	redirectAuthError(ctx, "http://[::1", "state", "access_denied")

	if w.Code != http.StatusBadRequest || w.Header().Get("Location") != "" {
		t.Errorf("status = %d Location=%q, want a plain 400 and no redirect", w.Code, w.Header().Get("Location"))
	}
}

func TestOAuthTokenAuthorizationCodeUserGone(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	authCode, verifier := oauthServerTestIssueCode(t, client, user)
	if result := database.Instance.Unscoped().Delete(&models.User{}, "id = ?", user.ID); result.Error != nil {
		t.Fatalf("failed to hard-delete user: %v", result.Error)
	}

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {authCode},
		"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
		"code_verifier": {verifier},
	})
	if code != http.StatusBadRequest || body["error_description"] != "user not found" {
		t.Errorf("status=%d body=%v, want 400 invalid_grant 'user not found'", code, body)
	}
}

func TestOAuthTokenAuthorizationCodeServerErrors(t *testing.T) {
	cases := []struct {
		name   string
		inject func(t *testing.T)
	}{
		{"refresh session not stored", func(t *testing.T) { failDBOperation(t, "create", "sessions", 0) }},
		{"access token not minted", oauthServerTestClearSigningKey},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			oauthServerTestSetup(t)
			client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
			user := createTestUser(t)
			authCode, verifier := oauthServerTestIssueCode(t, client, user)
			c.inject(t)

			code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
				"grant_type": {"authorization_code"}, "code": {authCode},
				"redirect_uri": {client.RedirectURIs[0]}, "client_id": {client.ClientID},
				"code_verifier": {verifier},
			})
			if code != http.StatusInternalServerError || body["error"] != "server_error" {
				t.Errorf("status=%d body=%v, want 500 server_error", code, body)
			}
		})
	}
}

func TestOAuthTokenRefreshUnknownClient(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {"x"}, "client_id": {"does-not-exist"},
	})
	if code != http.StatusUnauthorized || body["error"] != "invalid_client" {
		t.Errorf("status=%d body=%v, want 401 invalid_client", code, body)
	}
}

func TestOAuthTokenRefreshFirstPartyCookieRotates(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	_, exchange := oauthServerTestExchange(t, client, user)
	refreshCookie := oauthServerTestCookie(exchange, refreshCookieName)
	if refreshCookie == nil || refreshCookie.Value == "" {
		t.Fatal("expected a refresh cookie from the first-party exchange")
	}

	// The cookie wins over the (bogus) form value for the first-party client.
	code, body, w := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {"ignored"}, "client_id": {client.ClientID},
	}, &http.Cookie{Name: refreshCookieName, Value: refreshCookie.Value})
	if code != http.StatusOK || body["access_token"] == nil || body["id_token"] == nil {
		t.Fatalf("status=%d body=%v, want 200 with access and id tokens", code, body)
	}
	if body["refresh_token"] != nil {
		t.Error("first-party client should not receive the refresh token in the body")
	}
	rotated := oauthServerTestCookie(w, refreshCookieName)
	if rotated == nil || rotated.Value == "" || rotated.Value == refreshCookie.Value {
		t.Errorf("rotated refresh cookie = %v, want a new non-empty value", rotated)
	}
}

func TestOAuthTokenRefreshFirstPartyInvalidClearsCookie(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")

	code, body, w := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "client_id": {client.ClientID},
	}, &http.Cookie{Name: refreshCookieName, Value: "garbage"})
	if code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want 400 invalid_grant", code, body)
	}
	if cleared := oauthServerTestCookie(w, refreshCookieName); cleared == nil || cleared.MaxAge >= 0 {
		t.Errorf("refresh cookie = %v, want it cleared", cleared)
	}
}

func TestOAuthTokenRefreshWithinGraceReturnsSameToken(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	tokens, _ := oauthServerTestExchange(t, client, user)
	refreshToken, _ := tokens["refresh_token"].(string)

	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}, "client_id": {client.ClientID}}
	if code, body, _ := postForm(APIOAuthToken, "/oauth/token", form); code != http.StatusOK || body["refresh_token"] == refreshToken {
		t.Fatalf("first refresh: status=%d body=%v, want 200 with a rotated token", code, body)
	}

	// A second use of the just-rotated token inside the grace window (e.g. a
	// concurrent tab) succeeds without rotating again and echoes it back.
	code, body, _ := postForm(APIOAuthToken, "/oauth/token", form)
	if code != http.StatusOK || body["refresh_token"] != refreshToken {
		t.Errorf("second refresh: status=%d body=%v, want 200 echoing the presented token", code, body)
	}
}

func TestOAuthTokenRefreshAccessTokenMintFails(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	tokens, _ := oauthServerTestExchange(t, client, user)
	oauthServerTestClearSigningKey(t)

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tokens["refresh_token"].(string)}, "client_id": {client.ClientID},
	})
	if code != http.StatusInternalServerError || body["error"] != "server_error" {
		t.Errorf("status=%d body=%v, want 500 server_error", code, body)
	}
}

func TestOAuthRevokeWithCookie(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, true, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	_, exchange := oauthServerTestExchange(t, client, user)
	refreshCookie := oauthServerTestCookie(exchange, refreshCookieName)

	code, body, _ := postForm(APIOAuthRevoke, "/oauth/revoke", url.Values{}, &http.Cookie{Name: refreshCookieName, Value: refreshCookie.Value})
	if code != http.StatusOK || body["message"] != "Revoked." {
		t.Fatalf("status=%d body=%v, want 200 Revoked.", code, body)
	}

	code, body, _ = postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "client_id": {client.ClientID},
	}, &http.Cookie{Name: refreshCookieName, Value: refreshCookie.Value})
	if code != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Errorf("status=%d body=%v, want the cookie's session to be revoked", code, body)
	}
}

func TestOAuthRevokeDatabaseFailureStillLogsOut(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	failDBOperation(t, "update", "sessions", 0)

	code, body, w := postForm(APIOAuthRevoke, "/oauth/revoke", url.Values{"token": {"some-token"}})
	if code != http.StatusOK || body["message"] != "Revoked." {
		t.Errorf("status=%d body=%v, want 200 Revoked. even when the revoke write fails", code, body)
	}
	if cleared := oauthServerTestCookie(w, ssoCookieName); cleared == nil || cleared.MaxAge >= 0 {
		t.Errorf("SSO cookie = %v, want it cleared", cleared)
	}
}

// --- Audience rule (resolveResource): the API is first-party only ---

func TestResolveResource(t *testing.T) {
	oauthServerTestSetup(t)
	firstParty := models.OAuthClient{IsFirstParty: true}
	thirdParty := models.OAuthClient{}
	api, mcp := config.APIResource(), config.MCPResource()

	cases := []struct {
		name      string
		client    models.OAuthClient
		requested string
		mcpOn     bool
		want      string
		wantOK    bool
	}{
		{"first-party defaults to the API", firstParty, "", true, api, true},
		{"first-party may ask for the API", firstParty, api, false, api, true},
		{"first-party may ask for MCP while it's on", firstParty, mcp, true, mcp, true},
		{"third-party defaults to MCP", thirdParty, "", true, mcp, true},
		{"third-party may not ask for the API", thirdParty, api, true, "", false},
		{"third-party gets nothing with MCP off", thirdParty, "", false, "", false},
		{"unknown resources are refused", firstParty, "https://elsewhere.example/api", true, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config.ConfigFile.MCPEnabled = c.mcpOn
			got, ok := resolveResource(c.client, c.requested)
			if ok != c.wantOK || (ok && got != c.want) {
				t.Errorf("resolveResource = %q, %v; want %q, %v", got, ok, c.want, c.wantOK)
			}
		})
	}
}

// The original bypass: a self-registered client asking only for "openid
// profile" (no resource) must end up with an MCP token, not an API one.
func TestOAuthThirdPartyWithoutResourceGetsMCPToken(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid", "profile"}, "https://client.example/cb")
	user := createTestUser(t)

	body, _ := oauthServerTestExchange(t, client, user)
	accessToken, _ := body["access_token"].(string)
	if _, err := auth.ValidateOAuthAccessToken(accessToken, config.APIResource()); err == nil {
		t.Fatal("third-party token was accepted for the API audience")
	}
	claims, err := auth.ValidateOAuthAccessToken(accessToken, config.MCPResource())
	if err != nil {
		t.Fatalf("third-party token isn't a valid MCP token: %v", err)
	}
	if claims.ClientID != client.ClientID {
		t.Errorf("client_id claim = %q, want %q", claims.ClientID, client.ClientID)
	}
}

func TestOAuthAuthorizeThirdPartyAPIResourceRefused(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	_, challenge := oauthServerTestPKCE()

	w := getAuthorize(url.Values{
		"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]}, "response_type": {"code"},
		"scope": {"openid"}, "state": {"s"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"},
		"resource": {config.APIResource()},
	})
	if w.Code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "error=invalid_target") {
		t.Errorf("status = %d location=%q, want a redirect with error=invalid_target", w.Code, w.Header().Get("Location"))
	}
}

// The consent form round-trips through the browser, so a tampered resource
// field must not widen the audience.
func TestOAuthConsentRevalidatesResource(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)

	for _, resource := range []string{config.APIResource(), "https://elsewhere.example/api"} {
		form := url.Values{
			"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURIs[0]}, "scope": {"openid"},
			"state": {"s"}, "code_challenge": {"c"}, "resource": {resource}, "action": {"allow"},
		}
		code, _, w := postForm(APIOAuthConsent, "/oauth/consent", form, oauthServerTestSSOCookie(t, user.ID))
		if code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "error=invalid_target") {
			t.Errorf("resource %q: status = %d location=%q, want error=invalid_target", resource, code, w.Header().Get("Location"))
		}
	}
	if consents, err := database.GetUserConsents(user.ID); err == nil && len(consents) != 0 {
		t.Errorf("a refused consent was still stored: %v", consents)
	}
}

func TestOAuthTokenCodeGrantRefusesDisallowedResource(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	user := createTestUser(t)
	verifier, challenge := oauthServerTestPKCE()

	// A code minted before the audience rule existed, bound to the API.
	if _, err := database.CreateAuthorizationCode(models.AuthorizationCode{
		CodeHash: utilities.HashOpaqueToken("legacy-code"), ClientID: client.ClientID, UserID: user.ID,
		RedirectURI: client.RedirectURIs[0], Scope: "openid", Resource: config.APIResource(),
		CodeChallenge: challenge, CodeChallengeMethod: "S256", ExpiresAt: time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("failed to create code: %v", err)
	}

	code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {"legacy-code"}, "redirect_uri": {client.RedirectURIs[0]},
		"client_id": {client.ClientID}, "code_verifier": {verifier},
	})
	if code != http.StatusBadRequest || body["error"] != "invalid_target" {
		t.Errorf("status = %d body=%v, want 400 invalid_target", code, body)
	}
}

func TestOAuthTokenRefreshRefusesDisallowedSessions(t *testing.T) {
	setupControllersDB(t)
	oauthServerTestSetup(t)
	client := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://client.example/cb")
	other := oauthServerTestNewClient(t, false, true, []string{"openid"}, "https://other.example/cb")
	user := createTestUser(t)

	cases := []struct {
		name, sessionClient, resource string
	}{
		{"API session held by a third-party client", client.ClientID, config.APIResource()},
		{"another client's session", other.ClientID, config.MCPResource()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plain := "refresh-" + uuid.NewString()
			if _, err := database.CreateOAuthRefreshSession(user.ID, utilities.HashOpaqueToken(plain), c.sessionClient, "openid", c.resource, "ua", "127.0.0.1"); err != nil {
				t.Fatalf("failed to create session: %v", err)
			}

			code, body, _ := postForm(APIOAuthToken, "/oauth/token", url.Values{
				"grant_type": {"refresh_token"}, "refresh_token": {plain}, "client_id": {client.ClientID},
			})
			if code != http.StatusBadRequest || body["error"] != "invalid_grant" || body["access_token"] != nil {
				t.Errorf("status = %d body=%v, want 400 invalid_grant and no token", code, body)
			}
		})
	}
}
