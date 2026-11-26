package middlewares

import (
	"aunefyren/poenskelisten/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// A page of another site, or of another origin of the same site, can't use
// a visitor's browser to sign in, complete MFA or give consent. Clients that
// aren't browsers send neither header and aren't concerned.
func TestSameOrigin(t *testing.T) {
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
	config.ConfigFile.PoenskelistenAdditionalURLs = "http://192.168.1.10:8080"

	cases := []struct {
		name, origin, fetchSite string
		allowed                 bool
	}{
		{"not a browser", "", "", true},
		{"external URL", "https://wishlist.example.com", "", true},
		{"external URL with its default port", "https://wishlist.example.com:443", "", true},
		{"additional URL", "http://192.168.1.10:8080", "", true},
		{"another site", "https://evil.example", "", false},
		{"another origin of the same site", "https://other.example.com", "", false},
		{"plain HTTP of the external URL", "http://wishlist.example.com", "", false},
		{"opaque origin", "null", "", false},
		{"same origin without Origin", "", "same-origin", true},
		{"typed address", "", "none", true},
		{"same site without Origin", "", "same-site", false},
		{"cross site without Origin", "", "cross-site", false},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/open/tokens/register", nil)
		if c.origin != "" {
			ctx.Request.Header.Set("Origin", c.origin)
		}
		if c.fetchSite != "" {
			ctx.Request.Header.Set("Sec-Fetch-Site", c.fetchSite)
		}
		SameOrigin()(ctx)
		if allowed := !ctx.IsAborted(); allowed != c.allowed {
			t.Errorf("%s: allowed = %v (status %d), want %v", c.name, allowed, w.Code, c.allowed)
		}
	}
}

// The external URL is configured by hand; the browser writes the origin its
// own way.
func TestSameOriginAcceptsTheExternalURLAsTyped(t *testing.T) {
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })

	for _, external := range []string{"https://Wishlist.Example.com:443/", "https://wishlist.example.com/app"} {
		config.ConfigFile.PoenskelistenExternalURL = external
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/open/tokens/register", nil)
		ctx.Request.Header.Set("Origin", "https://wishlist.example.com")
		SameOrigin()(ctx)
		if ctx.IsAborted() {
			t.Errorf("external URL %q: its own page refused with %d", external, w.Code)
		}
	}
}
