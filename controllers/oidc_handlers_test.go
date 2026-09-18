package controllers

import (
	"aunefyren/poenskelisten/config"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
