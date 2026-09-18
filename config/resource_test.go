package config

import "testing"

func TestOAuthIssuer(t *testing.T) {
	origURL := ConfigFile.PoenskelistenExternalURL
	origPort := ConfigFile.PoenskelistenPort
	t.Cleanup(func() {
		ConfigFile.PoenskelistenExternalURL = origURL
		ConfigFile.PoenskelistenPort = origPort
	})

	t.Run("uses the external URL, trimming a trailing slash", func(t *testing.T) {
		ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com/"
		if got := OAuthIssuer(); got != "https://wishlist.example.com" {
			t.Errorf("OAuthIssuer() = %q, want no trailing slash", got)
		}
	})

	t.Run("falls back to localhost with the configured port", func(t *testing.T) {
		ConfigFile.PoenskelistenExternalURL = ""
		ConfigFile.PoenskelistenPort = 9090
		if got := OAuthIssuer(); got != "http://localhost:9090" {
			t.Errorf("OAuthIssuer() = %q, want http://localhost:9090", got)
		}
	})
}

func TestAPIResourceAndMCPResource(t *testing.T) {
	origURL := ConfigFile.PoenskelistenExternalURL
	t.Cleanup(func() { ConfigFile.PoenskelistenExternalURL = origURL })
	ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"

	if got := APIResource(); got != "https://wishlist.example.com/api" {
		t.Errorf("APIResource() = %q, want .../api", got)
	}
	if got := MCPResource(); got != "https://wishlist.example.com/mcp" {
		t.Errorf("MCPResource() = %q, want .../mcp", got)
	}
}
