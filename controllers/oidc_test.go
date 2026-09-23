package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDeriveNames(t *testing.T) {
	cases := []struct {
		given, family, name, email string
		wantFirst, wantLast        string
	}{
		{"Ada", "Lovelace", "ignored", "ada@example.com", "Ada", "Lovelace"},
		{"", "", "Grace Hopper", "grace@example.com", "Grace", "Hopper"},
		{"", "", "Cher", "cher@example.com", "Cher", ""},
		{"", "", "", "alan@example.com", "alan", ""},
		{"", "", "", "", "User", ""},
		{"  ", "  ", "  Ada  Lovelace ", "x@y.com", "Ada", "Lovelace"},
		{"", "", "Mary Jane Watson", "mj@example.com", "Mary", "Jane Watson"},
	}

	for _, c := range cases {
		gotFirst, gotLast := deriveNames(c.given, c.family, c.name, c.email)
		if gotFirst != c.wantFirst || gotLast != c.wantLast {
			t.Errorf("deriveNames(%q,%q,%q,%q) = (%q,%q), want (%q,%q)",
				c.given, c.family, c.name, c.email, gotFirst, gotLast, c.wantFirst, c.wantLast)
		}
	}
}

func TestOIDCResolveErrorMessage(t *testing.T) {
	// Sentinel errors map to specific messages; anything else is generic.
	if msg := oidcResolveErrorMessage(database.ErrOIDCEmailNotVerified); msg == "Single sign-on failed." {
		t.Error("expected a specific message for ErrOIDCEmailNotVerified")
	}
	if msg := oidcResolveErrorMessage(database.ErrOIDCUserNotFound); msg == "Single sign-on failed." {
		t.Error("expected a specific message for ErrOIDCUserNotFound")
	}
	if msg := oidcResolveErrorMessage(database.ErrOIDCNoEmail); msg == "Single sign-on failed." {
		t.Error("expected a specific message for ErrOIDCNoEmail")
	}
	if msg := oidcResolveErrorMessage(errors.New("boom")); msg != "Single sign-on failed." {
		t.Errorf("unknown error mapped to %q, want generic message", msg)
	}
}

// TestOIDCCallbackClearsFlowCookiesBeforeRedirect covers a bug where the
// one-time state/nonce cookies were cleared via `defer` after ctx.Redirect
// had already run - too late, since gin finalizes response headers at
// redirect time, so the Set-Cookie header clearing them never reached the
// client. redirectLoginError must clear them beforehand instead.
func TestOIDCCallbackClearsFlowCookiesBeforeRedirect(t *testing.T) {
	restoreConfig(t)
	setupControllersDB(t)
	config.ConfigFile.OIDCEnabled = false
	t.Cleanup(func() { config.ConfigFile.OIDCEnabled = false })

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/oidc/callback?state=abc&code=xyz", nil)
	ctx.Request.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: "abc"})
	ctx.Request.AddCookie(&http.Cookie{Name: oidcNonceCookie, Value: "def"})

	OIDCCallback(ctx)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}

	cleared := map[string]bool{}
	for _, c := range w.Result().Cookies() {
		if c.MaxAge < 0 {
			cleared[c.Name] = true
		}
	}
	if !cleared[oidcStateCookie] || !cleared[oidcNonceCookie] {
		t.Errorf("expected both flow cookies cleared in the redirect response, got: %v", w.Result().Cookies())
	}
}
