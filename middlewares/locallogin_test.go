package middlewares

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func setLocalLogin(t *testing.T, disabled bool, oidcConfigured bool) {
	t.Helper()
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })

	config.ConfigFile.LocalLoginDisabled = disabled
	config.ConfigFile.OIDCEnabled = oidcConfigured
	config.ConfigFile.OIDCIssuerURL = "https://auth.example.com"
	config.ConfigFile.OIDCClientID = "poenskelisten"
	config.ConfigFile.OIDCProviderName = "Authelia"

	if logger.Log == nil {
		logger.Log = logrus.New()
		logger.Log.SetOutput(io.Discard)
	}
}

func runRequireLocalLogin() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/open/tokens/register", nil)
	RequireLocalLogin()(ctx)
	return ctx, w
}

func TestRequireLocalLoginAllowsByDefault(t *testing.T) {
	setLocalLogin(t, false, true)

	ctx, _ := runRequireLocalLogin()
	if ctx.IsAborted() {
		t.Error("password endpoint aborted although local login is enabled")
	}
}

func TestRequireLocalLoginBlocksWhenOIDCOnly(t *testing.T) {
	setLocalLogin(t, true, true)

	ctx, w := runRequireLocalLogin()
	if !ctx.IsAborted() {
		t.Fatal("password endpoint not aborted with local login disabled")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Log in with Authelia.") {
		t.Errorf("body = %s, want it to name the provider", w.Body.String())
	}
}

// Without a working OIDC setup there'd be no way in at all, so the flag is
// ignored rather than locking everyone out.
func TestRequireLocalLoginIgnoresFlagWithoutOIDC(t *testing.T) {
	setLocalLogin(t, true, false)

	ctx, _ := runRequireLocalLogin()
	if ctx.IsAborted() {
		t.Error("password endpoint aborted although OIDC isn't enabled")
	}
}
