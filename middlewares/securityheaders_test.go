package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/wishlists/public/abc", nil)
	SecurityHeaders()(ctx)

	want := map[string]string{
		"Content-Security-Policy": ContentSecurityPolicy,
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "same-origin",
	}
	for name, value := range want {
		if got := w.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
	if ctx.IsAborted() {
		t.Error("SecurityHeaders aborted the request; it must only add headers")
	}
}

// The directives below are what keep an injected script from shipping data to
// another host or being framed; a later edit that drops one should fail here.
func TestContentSecurityPolicyDirectives(t *testing.T) {
	directives := map[string]string{}
	for _, part := range strings.Split(ContentSecurityPolicy, ";") {
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		directives[fields[0]] = strings.Join(fields[1:], " ")
	}

	cases := map[string]string{
		"default-src":     "'self'",
		"script-src":      "'self'",
		"connect-src":     "'self'",
		"img-src":         "'self' data: blob:",
		"object-src":      "'none'",
		"base-uri":        "'self'",
		"frame-ancestors": "'none'",
	}
	for name, value := range cases {
		if got, ok := directives[name]; !ok || got != value {
			t.Errorf("directive %s = %q (present=%v), want %q", name, got, ok, value)
		}
	}
	if _, ok := directives["form-action"]; ok {
		t.Error("form-action must stay unset: it would block the OAuth consent redirect to third-party clients")
	}
}
