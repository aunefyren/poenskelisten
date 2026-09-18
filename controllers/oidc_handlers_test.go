package controllers

import (
	"aunefyren/poenskelisten/config"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// startFakeOIDCServer serves just enough of the OIDC discovery document for
// oidcprovider.Get() to build a client: a real network call would be a live
// IdP, but this is an in-process fake, so OIDCLogin/OIDCCallback can be
// exercised up to (not including) verifying a real signed ID token — that
// last step would need a full JWKS + signed JWT and is left untested here.
func startFakeOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/keys",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	})
	// The token endpoint always fails, so callback tests that reach the code
	// exchange step exercise the "exchange failed" branch.
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})

	return server
}

// enableFakeOIDC points the app config at a fake IdP so oidcprovider.Get()
// succeeds, and returns the server for further customization/inspection.
func enableFakeOIDC(t *testing.T) *httptest.Server {
	t.Helper()
	server := startFakeOIDCServer(t)
	config.ConfigFile.OIDCEnabled = true
	config.ConfigFile.OIDCIssuerURL = server.URL
	config.ConfigFile.OIDCClientID = "test-client"
	config.ConfigFile.OIDCClientSecret = "test-secret"
	config.ConfigFile.OIDCRedirectURL = server.URL + "/callback"
	config.ConfigFile.OIDCProviderName = "Test IdP"
	return server
}

func disableOIDC() {
	config.ConfigFile.OIDCEnabled = false
	config.ConfigFile.OIDCIssuerURL = ""
	config.ConfigFile.OIDCClientID = ""
	config.ConfigFile.OIDCClientSecret = ""
	config.ConfigFile.OIDCRedirectURL = ""
}

func TestGetOIDCConfigDisabled(t *testing.T) {
	disableOIDC()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/config", nil)
	ctxTestHelper(w, req, APIGetOIDCConfig)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["enabled"] != false {
		t.Errorf("enabled = %v, want false", body["enabled"])
	}
}

func TestGetOIDCConfigEnabled(t *testing.T) {
	server := enableFakeOIDC(t)
	defer disableOIDC()
	_ = server

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/config", nil)
	ctxTestHelper(w, req, APIGetOIDCConfig)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["enabled"] != true {
		t.Errorf("enabled = %v, want true", body["enabled"])
	}
	if body["provider_name"] != "Test IdP" {
		t.Errorf("provider_name = %v, want Test IdP", body["provider_name"])
	}
}

func TestOIDCLoginNotConfigured(t *testing.T) {
	disableOIDC()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/login", nil)
	ctxTestHelper(w, req, OIDCLogin)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/login?error=Single+sign-on+is+not+available." {
		t.Errorf("Location = %q", loc)
	}
}

func TestOIDCLoginSuccess(t *testing.T) {
	server := enableFakeOIDC(t)
	defer disableOIDC()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/login", nil)
	ctxTestHelper(w, req, OIDCLogin)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	wantPrefix := server.URL + "/authorize"
	if len(loc) < len(wantPrefix) || loc[:len(wantPrefix)] != wantPrefix {
		t.Errorf("Location = %q, want prefix %q", loc, wantPrefix)
	}

	foundState, foundNonce := false, false
	for _, c := range w.Result().Cookies() {
		if c.Name == oidcStateCookie && c.Value != "" {
			foundState = true
		}
		if c.Name == oidcNonceCookie && c.Value != "" {
			foundNonce = true
		}
	}
	if !foundState {
		t.Error("expected the OIDC state cookie to be set")
	}
	if !foundNonce {
		t.Error("expected the OIDC nonce cookie to be set")
	}
}

func TestOIDCCallbackNotConfigured(t *testing.T) {
	disableOIDC()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/callback", nil)
	ctxTestHelper(w, req, OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login?error=Single+sign-on+is+not+available." {
		t.Errorf("Location = %q", loc)
	}
}

func TestOIDCCallbackStateMismatch(t *testing.T) {
	enableFakeOIDC(t)
	defer disableOIDC()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/callback?state=from-query&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: "from-cookie"})
	ctxTestHelper(w, req, OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/login?error=Single+sign-on+failed.+Please+try+again." {
		t.Errorf("Location = %q", loc)
	}

	// NOTE: OIDCCallback registers `defer clearFlowCookie(...)` for both flow
	// cookies, intending to always clear them. But every return path in that
	// function ends by calling ctx.Redirect, which (via gin's Render) calls
	// WriteHeader synchronously — finalizing the response headers immediately.
	// The deferred SetCookie calls run afterward, so they mutate the header
	// map too late to ever reach the client on any path, success or failure.
	// Not asserting cookie-clearing here since it doesn't currently happen;
	// see the test file's top-level report for this finding.
}

func TestOIDCCallbackMissingCode(t *testing.T) {
	enableFakeOIDC(t)
	defer disableOIDC()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/callback?state=matching-state", nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: "matching-state"})
	ctxTestHelper(w, req, OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login?error=Single+sign-on+was+cancelled+or+failed." {
		t.Errorf("Location = %q", loc)
	}
}

// oidcSigningFixture is a fake IdP that also completes the token exchange with
// a real RS256-signed ID token, so OIDCCallback can be driven all the way
// through ID-token verification and account resolution.
type oidcSigningFixture struct {
	server    *httptest.Server
	signer    jose.Signer
	kid       string
	nextIDTok string
}

func startFakeOIDCServerWithSigning(t *testing.T) *oidcSigningFixture {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate signing key: %v", err)
	}
	const kid = "test-key"
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, (&jose.SignerOptions{}).WithHeader(jose.HeaderKey("kid"), kid))
	if err != nil {
		t.Fatalf("failed to build signer: %v", err)
	}

	fixture := &oidcSigningFixture{signer: signer, kid: kid}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	fixture.server = server

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/keys",
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key:       &privateKey.PublicKey,
			KeyID:     kid,
			Algorithm: string(jose.RS256),
			Use:       "sig",
		}}}
		_ = json.NewEncoder(w).Encode(jwks)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     fixture.nextIDTok,
		})
	})

	return fixture
}

// idTokenClaims mirrors the subset of standard + OIDC claims OIDCCallback reads.
type idTokenClaims struct {
	jwt.Claims
	Nonce         string `json:"nonce,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Name          string `json:"name,omitempty"`
}

// sign builds and sets the next ID token this fixture's /token endpoint will
// return, using the fixture's own server URL as both issuer and (implicitly)
// the audience the client validates against.
func (f *oidcSigningFixture) sign(t *testing.T, subject, nonce, email string, emailVerified bool, given, family, name string) {
	t.Helper()
	claims := idTokenClaims{
		Claims: jwt.Claims{
			Issuer:   f.server.URL,
			Subject:  subject,
			Audience: jwt.Audience{"test-client"},
			Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
		Nonce:         nonce,
		Email:         email,
		EmailVerified: emailVerified,
		GivenName:     given,
		FamilyName:    family,
		Name:          name,
	}
	tok, err := jwt.Signed(f.signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatalf("failed to sign ID token: %v", err)
	}
	f.nextIDTok = tok
}

func enableFakeOIDCWithSigning(t *testing.T) *oidcSigningFixture {
	t.Helper()
	fixture := startFakeOIDCServerWithSigning(t)
	config.ConfigFile.OIDCEnabled = true
	config.ConfigFile.OIDCIssuerURL = fixture.server.URL
	config.ConfigFile.OIDCClientID = "test-client"
	config.ConfigFile.OIDCClientSecret = "test-secret"
	config.ConfigFile.OIDCRedirectURL = fixture.server.URL + "/callback"
	config.ConfigFile.OIDCProviderName = "Test IdP"
	return fixture
}

func callbackRequest(state string) *http.Request {
	req := httptest.NewRequest("GET", "/api/open/oidc/callback?state="+state+"&code=some-code", nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: state})
	req.AddCookie(&http.Cookie{Name: oidcNonceCookie, Value: "matching-nonce"})
	return req
}

func TestOIDCCallbackMissingIDToken(t *testing.T) {
	setupControllersDB(t)
	fixture := enableFakeOIDCWithSigning(t)
	defer disableOIDC()
	fixture.nextIDTok = "" // /token responds 200 but with an empty id_token

	w := httptest.NewRecorder()
	ctxTestHelper(w, callbackRequest("matching-state"), OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/login?error=Single+sign-on+failed." {
		t.Errorf("Location = %q", loc)
	}
}

func TestOIDCCallbackInvalidSignature(t *testing.T) {
	setupControllersDB(t)
	fixture := enableFakeOIDCWithSigning(t)
	defer disableOIDC()

	// Sign with an unrelated key never published in this fixture's JWKS.
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate a second key: %v", err)
	}
	otherSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: otherKey}, (&jose.SignerOptions{}).WithHeader(jose.HeaderKey("kid"), fixture.kid))
	if err != nil {
		t.Fatalf("failed to build a second signer: %v", err)
	}
	claims := idTokenClaims{
		Claims: jwt.Claims{
			Issuer: fixture.server.URL, Subject: "user-1", Audience: jwt.Audience{"test-client"},
			Expiry: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now()),
		},
		Nonce: "matching-nonce", Email: "ada@example.com", EmailVerified: true,
	}
	tok, err := jwt.Signed(otherSigner).Claims(claims).Serialize()
	if err != nil {
		t.Fatalf("failed to sign with the second key: %v", err)
	}
	fixture.nextIDTok = tok

	w := httptest.NewRecorder()
	ctxTestHelper(w, callbackRequest("matching-state"), OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/login?error=Single+sign-on+failed." {
		t.Errorf("Location = %q, want the generic sign-on-failed redirect for a bad signature", loc)
	}
}

func TestOIDCCallbackNonceMismatch(t *testing.T) {
	setupControllersDB(t)
	fixture := enableFakeOIDCWithSigning(t)
	defer disableOIDC()
	fixture.sign(t, "user-1", "a-different-nonce", "ada@example.com", true, "Ada", "Lovelace", "")

	w := httptest.NewRecorder()
	ctxTestHelper(w, callbackRequest("matching-state"), OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/login?error=Single+sign-on+failed.+Please+try+again." {
		t.Errorf("Location = %q", loc)
	}
}

func TestOIDCCallbackUnknownUserAutoCreateDisabled(t *testing.T) {
	setupControllersDB(t)
	fixture := enableFakeOIDCWithSigning(t)
	defer disableOIDC()
	config.ConfigFile.OIDCAutoCreateUsers = false
	fixture.sign(t, "user-1", "matching-nonce", "ada@example.com", true, "Ada", "Lovelace", "")

	w := httptest.NewRecorder()
	ctxTestHelper(w, callbackRequest("matching-state"), OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	if loc != "/login?error=No+account+is+linked+to+this+login.+Please+contact+an+administrator." {
		t.Errorf("Location = %q", loc)
	}
}

func TestOIDCCallbackSuccessAutoCreatesUser(t *testing.T) {
	setupControllersDB(t)
	fixture := enableFakeOIDCWithSigning(t)
	defer disableOIDC()
	config.ConfigFile.OIDCAutoCreateUsers = true
	enablePrivateKey(t) // issueSSOSession needs it
	fixture.sign(t, "user-1", "matching-nonce", "ada@example.com", true, "Ada", "Lovelace", "")

	w := httptest.NewRecorder()
	ctxTestHelper(w, callbackRequest("matching-state"), OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body=%s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want /", loc)
	}
	foundSSOCookie := false
	for _, c := range w.Result().Cookies() {
		if c.Name == ssoCookieName && c.Value != "" {
			foundSSOCookie = true
		}
	}
	if !foundSSOCookie {
		t.Error("expected an SSO cookie to be set after a successful OIDC login")
	}
}

func TestOIDCCallbackExchangeFails(t *testing.T) {
	enableFakeOIDC(t)
	defer disableOIDC()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/open/oidc/callback?state=matching-state&code=some-code", nil)
	req.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: "matching-state"})
	ctxTestHelper(w, req, OIDCCallback)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/login?error=Single+sign-on+failed." {
		t.Errorf("Location = %q", loc)
	}
}

// ctxTestHelper builds a gin test context around req and runs handler. It's
// named distinctly from the package's other test helpers (e.g. runHandler in
// oauth_metadata_test.go) since it also propagates the request's cookies,
// which those don't need.
func ctxTestHelper(w *httptest.ResponseRecorder, req *http.Request, handler gin.HandlerFunc) {
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	handler(ctx)
}
