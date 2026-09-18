package oidcprovider

import (
	"aunefyren/poenskelisten/config"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// resetCache clears the package-level provider cache before/after a test, since
// it's shared global state across the whole test binary.
func resetCache(t *testing.T) {
	t.Helper()
	mutex.Lock()
	cached = nil
	cacheKey = ""
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		cached = nil
		cacheKey = ""
		mutex.Unlock()
	})
}

// startFakeOIDCProvider serves just enough of the OIDC discovery document for
// go-oidc's provider discovery to succeed.
func startFakeOIDCProvider(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/auth",
			"token_endpoint":         srv.URL + "/token",
			"jwks_uri":               srv.URL + "/jwks",
		})
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func setOIDCConfig(t *testing.T, issuerURL string) {
	t.Helper()
	orig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = orig })

	config.ConfigFile.OIDCEnabled = true
	config.ConfigFile.OIDCIssuerURL = issuerURL
	config.ConfigFile.OIDCClientID = "test-client"
	config.ConfigFile.OIDCClientSecret = "test-secret"
	config.ConfigFile.OIDCRedirectURL = "https://app.example.com/api/open/oidc/callback"
}

func TestGetDisabled(t *testing.T) {
	resetCache(t)
	orig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = orig })
	config.ConfigFile.OIDCEnabled = false

	if _, err := Get(); err != ErrNotConfigured {
		t.Errorf("Get() error = %v, want ErrNotConfigured", err)
	}
}

func TestGetMissingRequiredFields(t *testing.T) {
	resetCache(t)
	orig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = orig })
	config.ConfigFile.OIDCEnabled = true
	config.ConfigFile.OIDCIssuerURL = ""
	config.ConfigFile.OIDCClientID = "test-client"
	config.ConfigFile.OIDCRedirectURL = "https://app.example.com/callback"

	if _, err := Get(); err != ErrNotConfigured {
		t.Errorf("Get() error = %v, want ErrNotConfigured for a missing issuer URL", err)
	}
}

func TestGetSuccess(t *testing.T) {
	resetCache(t)
	srv := startFakeOIDCProvider(t)
	setOIDCConfig(t, srv.URL)

	client, err := Get()
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if client.OAuth2Config.ClientID != "test-client" {
		t.Errorf("ClientID = %q, want test-client", client.OAuth2Config.ClientID)
	}
	if client.OAuth2Config.RedirectURL != "https://app.example.com/api/open/oidc/callback" {
		t.Errorf("RedirectURL = %q", client.OAuth2Config.RedirectURL)
	}
	if client.Verifier == nil {
		t.Error("expected a non-nil Verifier")
	}
}

func TestGetCachesClientForSameConfig(t *testing.T) {
	resetCache(t)
	var discoveryHits int
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		discoveryHits++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/auth",
			"token_endpoint":         srv.URL + "/token",
			"jwks_uri":               srv.URL + "/jwks",
		})
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	setOIDCConfig(t, srv.URL)

	first, err := Get()
	if err != nil {
		t.Fatalf("first Get() returned error: %v", err)
	}
	second, err := Get()
	if err != nil {
		t.Fatalf("second Get() returned error: %v", err)
	}
	if first != second {
		t.Error("expected the same cached *Client to be returned for an unchanged config")
	}
	if discoveryHits != 1 {
		t.Errorf("discovery endpoint was hit %d times, want exactly 1 (cached on the second call)", discoveryHits)
	}
}

func TestGetRebuildsWhenConfigChanges(t *testing.T) {
	resetCache(t)
	srv := startFakeOIDCProvider(t)
	setOIDCConfig(t, srv.URL)

	first, err := Get()
	if err != nil {
		t.Fatalf("first Get() returned error: %v", err)
	}

	config.ConfigFile.OIDCClientID = "a-different-client"
	second, err := Get()
	if err != nil {
		t.Fatalf("second Get() returned error: %v", err)
	}
	if first == second {
		t.Error("expected a new *Client after the OIDC client ID changed")
	}
	if second.OAuth2Config.ClientID != "a-different-client" {
		t.Errorf("ClientID = %q, want a-different-client", second.OAuth2Config.ClientID)
	}
}

func TestGetProviderUnreachable(t *testing.T) {
	resetCache(t)
	// Nothing is listening here, so discovery must fail.
	setOIDCConfig(t, "http://127.0.0.1:1")

	if _, err := Get(); err == nil {
		t.Error("expected an error when the OIDC issuer is unreachable")
	}
}
