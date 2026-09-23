package config

import "testing"

func TestGetPrivateKey(t *testing.T) {
	original := ConfigFile.PrivateKey
	t.Cleanup(func() { ConfigFile.PrivateKey = original })

	t.Run("valid key decodes", func(t *testing.T) {
		key, err := GenerateSecureKey(32)
		if err != nil {
			t.Fatalf("GenerateSecureKey returned error: %v", err)
		}
		ConfigFile.PrivateKey = key

		got, err := GetPrivateKey()
		if err != nil {
			t.Fatalf("GetPrivateKey returned error: %v", err)
		}
		if len(got) != 32 {
			t.Errorf("decoded key length = %d, want 32", len(got))
		}
	})

	t.Run("empty key errors", func(t *testing.T) {
		ConfigFile.PrivateKey = ""

		if _, err := GetPrivateKey(); err == nil {
			t.Error("GetPrivateKey accepted an empty key, want error")
		}
	})

	t.Run("invalid base64 errors", func(t *testing.T) {
		ConfigFile.PrivateKey = "!!!not-valid-base64!!!"

		if _, err := GetPrivateKey(); err == nil {
			t.Error("GetPrivateKey accepted invalid base64, want error")
		}
	})

	t.Run("key decoding to nothing errors", func(t *testing.T) {
		// The base64 decoder skips newlines, so this is non-empty yet decodes
		// cleanly to zero bytes - an HS256 key nobody would notice was blank.
		ConfigFile.PrivateKey = "\r\n"

		got, err := GetPrivateKey()
		if err == nil || got != nil {
			t.Errorf("GetPrivateKey = (%v, %v), want nil key and an error", got, err)
		}
	})
}

func TestOIDCDisplayName(t *testing.T) {
	original := ConfigFile.OIDCProviderName
	t.Cleanup(func() { ConfigFile.OIDCProviderName = original })

	cases := []struct {
		configured string
		want       string
	}{
		{"Authelia", "Authelia"},
		{"", "Single sign-on"},
		{"   ", "Single sign-on"},
	}
	for _, c := range cases {
		ConfigFile.OIDCProviderName = c.configured
		if got := OIDCDisplayName(); got != c.want {
			t.Errorf("OIDCDisplayName() with %q = %q, want %q", c.configured, got, c.want)
		}
	}
}

func TestOIDCCallbackURL(t *testing.T) {
	origRedirect := ConfigFile.OIDCRedirectURL
	origURL := ConfigFile.PoenskelistenExternalURL
	origPort := ConfigFile.PoenskelistenPort
	t.Cleanup(func() {
		ConfigFile.OIDCRedirectURL = origRedirect
		ConfigFile.PoenskelistenExternalURL = origURL
		ConfigFile.PoenskelistenPort = origPort
	})

	cases := []struct {
		name        string
		redirect    string
		externalURL string
		want        string
	}{
		{"explicit redirect wins", "https://sso.example.com/cb", "https://wishlist.example.com", "https://sso.example.com/cb"},
		{"derived from external URL", "", "https://wishlist.example.com/", "https://wishlist.example.com/api/open/oidc/callback"},
		{"localhost fallback", "", "", "http://localhost:9090/api/open/oidc/callback"},
	}
	for _, c := range cases {
		ConfigFile.OIDCRedirectURL = c.redirect
		ConfigFile.PoenskelistenExternalURL = c.externalURL
		ConfigFile.PoenskelistenPort = 9090
		if got := OIDCCallbackURL(); got != c.want {
			t.Errorf("%s: OIDCCallbackURL() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestLocalLoginEnabled(t *testing.T) {
	original := ConfigFile
	t.Cleanup(func() { ConfigFile = original })

	cases := []struct {
		name         string
		disabled     bool
		oidcEnabled  bool
		issuer       string
		clientID     string
		wantEnabled  bool
		wantOIDCConf bool
	}{
		{"default", false, false, "", "", true, false},
		{"OIDC on, local still allowed", false, true, "https://auth.example.com", "app", true, true},
		{"OIDC-only", true, true, "https://auth.example.com", "app", false, true},
		{"disabled but OIDC off: stays on", true, false, "https://auth.example.com", "app", true, false},
		{"disabled but no issuer: stays on", true, true, "", "app", true, false},
		{"disabled but no client ID: stays on", true, true, "https://auth.example.com", " ", true, false},
	}
	for _, c := range cases {
		ConfigFile.LocalLoginDisabled = c.disabled
		ConfigFile.OIDCEnabled = c.oidcEnabled
		ConfigFile.OIDCIssuerURL = c.issuer
		ConfigFile.OIDCClientID = c.clientID
		if got := LocalLoginEnabled(); got != c.wantEnabled {
			t.Errorf("%s: LocalLoginEnabled() = %v, want %v", c.name, got, c.wantEnabled)
		}
		if got := OIDCConfigured(); got != c.wantOIDCConf {
			t.Errorf("%s: OIDCConfigured() = %v, want %v", c.name, got, c.wantOIDCConf)
		}
	}
}
