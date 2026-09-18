package models

import "testing"

func TestOAuthClientIsEnabled(t *testing.T) {
	cases := []struct {
		name string
		c    OAuthClient
		want bool
	}{
		{"nil pointer", OAuthClient{}, false},
		{"false", OAuthClient{Enabled: boolPtr(false)}, false},
		{"true", OAuthClient{Enabled: boolPtr(true)}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.c.IsEnabled(); got != c.want {
				t.Errorf("IsEnabled() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestOAuthClientHasRedirectURI(t *testing.T) {
	client := OAuthClient{RedirectURIs: []string{"https://client.example/cb", "https://client.example/cb2"}}

	if !client.HasRedirectURI("https://client.example/cb") {
		t.Error("expected an exact match to be found")
	}
	if client.HasRedirectURI("https://client.example/cb?extra=1") {
		t.Error("HasRedirectURI must require an exact match, not a prefix match")
	}
	if client.HasRedirectURI("https://evil.example/cb") {
		t.Error("did not expect an unregistered URI to match")
	}
}

func TestOAuthClientAllowsScopes(t *testing.T) {
	client := OAuthClient{Scopes: []string{"openid", "profile", "mcp:wishlists.read"}}

	if !client.AllowsScopes([]string{"openid", "profile"}) {
		t.Error("expected a subset of allowed scopes to be permitted")
	}
	if !client.AllowsScopes(nil) {
		t.Error("expected an empty request to always be permitted")
	}
	if client.AllowsScopes([]string{"openid", "mcp:wishlists.write"}) {
		t.Error("did not expect a scope outside the allowed set to be permitted")
	}
}

func TestOAuthConsentCovers(t *testing.T) {
	consent := OAuthConsent{Scopes: []string{"openid", "profile"}}

	if !consent.Covers([]string{"openid"}) {
		t.Error("expected a granted scope to be covered")
	}
	if !consent.Covers(nil) {
		t.Error("expected an empty request to always be covered")
	}
	if consent.Covers([]string{"openid", "email"}) {
		t.Error("did not expect an ungranted scope to be covered")
	}
}
