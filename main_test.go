package main

import (
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	textTemplate "text/template"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func init() {
	gin.SetMode(gin.TestMode)
	if logger.Log == nil {
		logger.Log = logrus.New()
	}
}

// withArgs swaps in a fresh flag.CommandLine and os.Args for one parseFlags
// call: parseFlags registers its flags on the global set, so calling it twice
// against the same set would panic on redefinition, and `go test` has already
// consumed os.Args for its own -test.* flags.
func withArgs(t *testing.T, args ...string) {
	t.Helper()
	origArgs, origCommandLine := os.Args, flag.CommandLine
	t.Cleanup(func() {
		os.Args, flag.CommandLine = origArgs, origCommandLine
	})
	os.Args = append([]string{"poenskelisten"}, args...)
	flag.CommandLine = flag.NewFlagSet("poenskelisten", flag.ContinueOnError)
}

func TestParseFlagsNoFlagsKeepsConfig(t *testing.T) {
	withArgs(t)
	logLevel := logger.Log.GetLevel()
	t.Cleanup(func() { logger.Log.SetLevel(logLevel) })

	in := models.ConfigStruct{
		PoenskelistenPort: 9000,
		PoenskelistenName: "Wishes",
		DBType:            "postgres",
		DBSSL:             true,
		SMTPEnabled:       true,
		MFAEnforced:       true,
	}

	out, generateInvite, flagsProvided, err := parseFlags(in)
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if flagsProvided {
		t.Error("flagsProvided = true with no flags on the command line")
	}
	if generateInvite {
		t.Error("generateInvite = true with no flags on the command line")
	}
	if out != in {
		t.Errorf("config changed with no flags provided:\n got %+v\nwant %+v", out, in)
	}
}

func TestParseFlagsZeroPortFallsBackTo8080(t *testing.T) {
	withArgs(t)

	out, _, _, err := parseFlags(models.ConfigStruct{})
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if out.PoenskelistenPort != 8080 {
		t.Errorf("PoenskelistenPort = %d, want 8080", out.PoenskelistenPort)
	}
}

func TestParseFlagsOverridesEveryField(t *testing.T) {
	withArgs(t,
		"-port", "9001",
		"-externalurl", "https://wish.example.com",
		"-timezone", "Europe/Oslo",
		"-environment", "test",
		"-testemail", "test@example.com",
		"-name", "Wishes",
		"-description", "A wishlist app",
		"-loglevel", "debug",
		"-dbport", "5432",
		"-dbtype", "postgres",
		"-dbusername", "dbuser",
		"-dbpassword", "dbpass",
		"-dbname", "wishes",
		"-dbip", "10.0.0.2",
		"-dbssl", "TRUE",
		"-dblocation", "/data/db.sqlite",
		"-disablesmtp", "false",
		"-smtphost", "smtp.example.com",
		"-smtpport", "587",
		"-smtpusername", "smtpuser",
		"-smtppassword", "smtppass",
		"-smtpfrom", "noreply@example.com",
		"-mfaenforced", "true",
		"-mfarecoverycodes", "true",
		"-oidcenabled", "true",
		"-oidcprovidername", "Authentik",
		"-oidcissuerurl", "https://idp.example.com",
		"-oidcclientid", "client",
		"-oidcclientsecret", "secret",
		"-oidcredirecturl", "https://wish.example.com/api/open/oidc/callback",
		"-oidcautocreateusers", "true",
		"-disablelocallogin", "true",
		"-mcpenabled", "true",
		"-generateinvite", "true",
	)
	logLevel := logger.Log.GetLevel()
	t.Cleanup(func() { logger.Log.SetLevel(logLevel) })

	out, generateInvite, flagsProvided, err := parseFlags(models.ConfigStruct{})
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if !flagsProvided {
		t.Error("flagsProvided = false with flags on the command line")
	}
	if !generateInvite {
		t.Error("generateInvite = false with -generateinvite true")
	}

	want := models.ConfigStruct{
		PoenskelistenPort:        9001,
		PoenskelistenExternalURL: "https://wish.example.com",
		Timezone:                 "Europe/Oslo",
		PoenskelistenEnvironment: "test",
		PoenskelistenTestEmail:   "test@example.com",
		PoenskelistenName:        "Wishes",
		PoenskelistenDescription: "A wishlist app",
		PoenskelistenLogLevel:    "debug",
		DBPort:                   5432,
		DBType:                   "postgres",
		DBUsername:               "dbuser",
		DBPassword:               "dbpass",
		DBName:                   "wishes",
		DBIP:                     "10.0.0.2",
		DBSSL:                    true,
		DBLocation:               "/data/db.sqlite",
		SMTPEnabled:              true,
		SMTPHost:                 "smtp.example.com",
		SMTPPort:                 587,
		SMTPUsername:             "smtpuser",
		SMTPPassword:             "smtppass",
		SMTPFrom:                 "noreply@example.com",
		MFAEnforced:              true,
		MFARecoveryCodesEnabled:  true,
		OIDCEnabled:              true,
		OIDCProviderName:         "Authentik",
		OIDCIssuerURL:            "https://idp.example.com",
		OIDCClientID:             "client",
		OIDCClientSecret:         "secret",
		OIDCRedirectURL:          "https://wish.example.com/api/open/oidc/callback",
		OIDCAutoCreateUsers:      true,
		LocalLoginDisabled:       true,
		MCPEnabled:               true,
	}
	if out != want {
		t.Errorf("parsed config mismatch:\n got %+v\nwant %+v", out, want)
	}
	if logger.Log.GetLevel() != logrus.DebugLevel {
		t.Errorf("logger level = %v, want debug", logger.Log.GetLevel())
	}
}

func TestParseFlagsDisableSMTPTrueDisablesSMTP(t *testing.T) {
	withArgs(t, "-disablesmtp", "true")

	out, _, _, err := parseFlags(models.ConfigStruct{SMTPEnabled: true})
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if out.SMTPEnabled {
		t.Error("SMTPEnabled = true after -disablesmtp true")
	}
}

func TestParseFlagsInvalidLogLevelKeepsConfigured(t *testing.T) {
	withArgs(t, "-loglevel", "not-a-level")
	logLevel := logger.Log.GetLevel()
	t.Cleanup(func() { logger.Log.SetLevel(logLevel) })

	out, _, _, err := parseFlags(models.ConfigStruct{PoenskelistenLogLevel: "info"})
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if out.PoenskelistenLogLevel != "info" {
		t.Errorf("PoenskelistenLogLevel = %q, want unchanged %q", out.PoenskelistenLogLevel, "info")
	}
	if logger.Log.GetLevel() != logLevel {
		t.Errorf("logger level changed to %v on an unparseable -loglevel", logger.Log.GetLevel())
	}
}

func serve(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

// TestInitRouterServesTemplatedFrontend renders every kind of templated static
// file through the real router, which also proves each template executes
// against the data initRouter hands it (a broken template would 500 here
// rather than only in a browser).
func TestInitRouterServesTemplatedFrontend(t *testing.T) {
	router := initRouter(models.ConfigStruct{
		PoenskelistenName:        "Wishes",
		PoenskelistenCurrency:    "NOK",
		PoenskelistenDescription: "A wishlist app",
	})

	cases := []struct {
		path        string
		contentType string
	}{
		{"/", "text/html"},
		{"/login", "text/html"},
		{"/login/mfa", "text/html"},
		{"/groups/abc", "text/html"},
		{"/wishlists/abc", "text/html"},
		{"/wishlists/public/abc", "text/html"},
		{"/oauth/callback", "text/html"},
		{"/enroll", "text/html"},
		{"/account", "text/html"},
		{"/js/login.js", "application/javascript"},
		{"/js/functions.js", "application/javascript"},
		{"/service-worker.js", "application/javascript"},
		{"/manifest.json", "application/json"},
		{"/robots.txt", "text/plain"},
		{"/css/main.css", "text/css"},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			w := serve(router, http.MethodGet, c.path)
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want 200; body: %s", c.path, w.Code, w.Body.String())
			}
			if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, c.contentType) {
				t.Errorf("GET %s Content-Type = %q, want prefix %q", c.path, got, c.contentType)
			}
		})
	}
}

func TestInitRouterInjectsTemplateData(t *testing.T) {
	router := initRouter(models.ConfigStruct{PoenskelistenName: "Distinctive App Name"})

	w := serve(router, http.MethodGet, "/manifest.json")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /manifest.json = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Distinctive App Name") {
		t.Errorf("manifest.json does not contain the configured app name; body: %s", w.Body.String())
	}
}

func TestInitRouterProtectsAuthAndAdminGroups(t *testing.T) {
	router := initRouter(models.ConfigStruct{})

	for _, path := range []string{"/api/auth/me", "/api/admin/invites"} {
		w := serve(router, http.MethodGet, path)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without a token = %d, want 401", path, w.Code)
		}
	}
}

func TestInitRouterUnknownPathIs404(t *testing.T) {
	router := initRouter(models.ConfigStruct{})

	if w := serve(router, http.MethodGet, "/no-such-page"); w.Code != http.StatusNotFound {
		t.Errorf("GET /no-such-page = %d, want 404", w.Code)
	}
}

func TestRegisterTemplatedStaticFilesForDirectoryMissingDirRegistersNothing(t *testing.T) {
	router := gin.New()

	got, err := registerTemplatedStaticFilesForDirectory(router, "/js", true, "./does-not-exist", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Routes()) != 0 {
		t.Errorf("registered %d routes for a missing directory, want 0", len(got.Routes()))
	}
}

func TestMustLoadTemplatesNoMatchReturnsNil(t *testing.T) {
	if tmpl := MustLoadTemplates("./does-not-exist/*.js"); tmpl != nil {
		t.Errorf("MustLoadTemplates on an empty glob = %v, want nil", tmpl)
	}
}

func TestRenderTemplateHandlers(t *testing.T) {
	tmpl := textTemplate.Must(textTemplate.New("ok").Parse("hello {{.}}"))

	cases := []struct {
		name        string
		handler     func(*textTemplate.Template, string, any) gin.HandlerFunc
		contentType string
	}{
		{"js", RenderJSTemplate, "application/javascript; charset=utf-8"},
		{"json", RenderJSONTemplate, "application/json; charset=utf-8"},
		{"text", RenderTextTemplate, "text/plain; charset=utf-8"},
	}
	for _, c := range cases {
		t.Run(c.name+"/ok", func(t *testing.T) {
			router := gin.New()
			router.GET("/", c.handler(tmpl, "ok", "world"))

			w := serve(router, http.MethodGet, "/")
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			if w.Body.String() != "hello world" {
				t.Errorf("body = %q, want %q", w.Body.String(), "hello world")
			}
			if got := w.Header().Get("Content-Type"); got != c.contentType {
				t.Errorf("Content-Type = %q, want %q", got, c.contentType)
			}
		})
		t.Run(c.name+"/missing template", func(t *testing.T) {
			router := gin.New()
			router.GET("/", c.handler(tmpl, "missing", nil))

			w := serve(router, http.MethodGet, "/")
			if w.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500", w.Code)
			}
			if !strings.Contains(w.Body.String(), "template error") {
				t.Errorf("body = %q, want a template error", w.Body.String())
			}
		})
	}
}
