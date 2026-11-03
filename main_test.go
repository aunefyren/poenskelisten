package main

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	textTemplate "text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
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

	out, actions, flagsProvided, err := parseFlags(in)
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if flagsProvided {
		t.Error("flagsProvided = true with no flags on the command line")
	}
	if actions != (startupActions{}) {
		t.Errorf("actions = %+v with no flags on the command line", actions)
	}
	if out != in {
		t.Errorf("config changed with no flags provided:\n got %+v\nwant %+v", out, in)
	}
}

func TestParseFlagsRecoveryActions(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		want        startupActions
		wantErr     bool
		wantPersist bool
	}{
		{
			name: "both flags, cleaned",
			args: []string{"-resetpassword", "  'Admin@Example.com' ", "-resetmfa", "\tadmin@example.com\n"},
			want: startupActions{resetPasswordEmail: "Admin@Example.com", resetMFAEmail: "admin@example.com"},
		},
		{
			name: "empty value means not requested",
			args: []string{"-resetpassword", "", "-resetmfa", "   "},
		},
		{
			name: "action flags alone don't persist config",
			args: []string{"-generateinvite", "true", "-resetmfa", "a@b.c"},
			want: startupActions{generateInvite: true, resetMFAEmail: "a@b.c"},
		},
		{
			name:        "config flag alongside still persists",
			args:        []string{"-resetmfa", "a@b.c", "-port", "9000"},
			want:        startupActions{resetMFAEmail: "a@b.c"},
			wantPersist: true,
		},
		{
			name:    "embedded newline rejected",
			args:    []string{"-resetpassword", "a@b.c\nfake log line"},
			wantErr: true,
		},
		{
			name:    "not an address rejected",
			args:    []string{"-resetmfa", "admin"},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withArgs(t, c.args...)

			_, actions, flagsProvided, err := parseFlags(models.ConfigStruct{})
			if c.wantErr {
				if err == nil {
					t.Errorf("expected an error, got actions %+v", actions)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFlags returned error: %v", err)
			}
			if actions != c.want {
				t.Errorf("actions = %+v, want %+v", actions, c.want)
			}
			if flagsProvided != c.wantPersist {
				t.Errorf("flagsProvided = %v, want %v", flagsProvided, c.wantPersist)
			}
		})
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
		"-additionalurls", "http://192.168.1.10:8080/, HTTP://Wish.LAN",
		"-trustedproxies", " 172.18.0.5/16, 10.0.0.1 ",
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
		"-accesstokenminutes", "30",
		"-refreshtokendays", "14",
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

	out, actions, flagsProvided, err := parseFlags(models.ConfigStruct{})
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if !flagsProvided {
		t.Error("flagsProvided = false with flags on the command line")
	}
	if !actions.generateInvite {
		t.Error("generateInvite = false with -generateinvite true")
	}

	want := models.ConfigStruct{
		PoenskelistenPort:           9001,
		PoenskelistenExternalURL:    "https://wish.example.com",
		PoenskelistenAdditionalURLs: "http://192.168.1.10:8080,http://wish.lan",
		PoenskelistenTrustedProxies: "172.18.0.0/16,10.0.0.1",
		Timezone:                    "Europe/Oslo",
		PoenskelistenEnvironment:    "test",
		PoenskelistenTestEmail:      "test@example.com",
		PoenskelistenName:           "Wishes",
		PoenskelistenDescription:    "A wishlist app",
		PoenskelistenLogLevel:       "debug",
		DBPort:                      5432,
		DBType:                      "postgres",
		DBUsername:                  "dbuser",
		DBPassword:                  "dbpass",
		DBName:                      "wishes",
		DBIP:                        "10.0.0.2",
		DBSSL:                       true,
		DBLocation:                  "/data/db.sqlite",
		SMTPEnabled:                 true,
		SMTPHost:                    "smtp.example.com",
		SMTPPort:                    587,
		SMTPUsername:                "smtpuser",
		SMTPPassword:                "smtppass",
		SMTPFrom:                    "noreply@example.com",
		MFAEnforced:                 true,
		MFARecoveryCodesEnabled:     true,
		AccessTokenMinutes:          30,
		RefreshTokenDays:            14,
		OIDCEnabled:                 true,
		OIDCProviderName:            "Authentik",
		OIDCIssuerURL:               "https://idp.example.com",
		OIDCClientID:                "client",
		OIDCClientSecret:            "secret",
		OIDCRedirectURL:             "https://wish.example.com/api/open/oidc/callback",
		OIDCAutoCreateUsers:         true,
		LocalLoginDisabled:          true,
		MCPEnabled:                  true,
	}
	if out != want {
		t.Errorf("parsed config mismatch:\n got %+v\nwant %+v", out, want)
	}
	if logger.Log.GetLevel() != logrus.DebugLevel {
		t.Errorf("logger level = %v, want debug", logger.Log.GetLevel())
	}
}

func TestParseFlagsInvalidAdditionalURLsFails(t *testing.T) {
	withArgs(t, "-additionalurls", "http://wish.lan,192.168.1.10:8080")

	_, _, _, err := parseFlags(models.ConfigStruct{PoenskelistenAdditionalURLs: "http://kept.lan"})
	if err == nil || !strings.Contains(err.Error(), "additionalurls") {
		t.Errorf("parseFlags error = %v, want an invalid -additionalurls error", err)
	}
}

func TestParseFlagsInvalidTrustedProxiesFails(t *testing.T) {
	withArgs(t, "-trustedproxies", "10.0.0.1,0.0.0.0/0")

	_, _, _, err := parseFlags(models.ConfigStruct{PoenskelistenTrustedProxies: "10.0.0.2"})
	if err == nil || !strings.Contains(err.Error(), "trustedproxies") {
		t.Errorf("parseFlags error = %v, want an invalid -trustedproxies error", err)
	}
}

func TestParseFlagsOutOfRangeTokenLifetimesFail(t *testing.T) {
	cases := []struct {
		flag  string
		value string
	}{
		{"accesstokenminutes", "61"},
		{"accesstokenminutes", "-1"},
		{"refreshtokendays", "366"},
		{"refreshtokendays", "-1"},
	}
	for _, c := range cases {
		withArgs(t, "-"+c.flag, c.value)

		_, _, _, err := parseFlags(models.ConfigStruct{})
		if err == nil || !strings.Contains(err.Error(), c.flag) {
			t.Errorf("parseFlags with -%s %s error = %v, want an invalid -%s error", c.flag, c.value, err, c.flag)
		}
	}
}

func TestParseFlagsZeroTokenLifetimeResetsToDefault(t *testing.T) {
	withArgs(t, "-accesstokenminutes", "0", "-refreshtokendays", "0")

	out, _, _, err := parseFlags(models.ConfigStruct{AccessTokenMinutes: 30, RefreshTokenDays: 14})
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if out.AccessTokenMinutes != 0 || out.RefreshTokenDays != 0 {
		t.Errorf("lifetimes = %d min / %d days, want 0 / 0 (the defaults)", out.AccessTokenMinutes, out.RefreshTokenDays)
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

// The SSO button has to start the flow on external_url even when the page was
// opened on an additional URL, since that's the only origin the IdP returns to.
func TestInitRouterRendersOIDCLoginURL(t *testing.T) {
	origConfig := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = origConfig })
	config.ConfigFile.PoenskelistenExternalURL = "https://wishes.example.com"
	config.ConfigFile.PoenskelistenAdditionalURLs = "http://192.168.1.10:8080"
	config.ConfigFile.OIDCRedirectURL = ""

	router := initRouter(models.ConfigStruct{})
	w := serve(router, http.MethodGet, "/js/functions.js")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /js/functions.js = %d, want 200", w.Code)
	}
	want := `var OIDC_LOGIN_URL = "https://wishes.example.com/api/open/oidc/login";`
	if !strings.Contains(w.Body.String(), want) {
		t.Errorf("functions.js does not contain %s", want)
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

// Every response, including API errors and 404s, carries the security headers,
// so the CSP also covers pages that render untrusted wish data.
func TestInitRouterSetsSecurityHeaders(t *testing.T) {
	router := initRouter(models.ConfigStruct{})

	for _, path := range []string{"/", "/wishlists/public/abc", "/api/auth/me", "/does-not-exist"} {
		w := serve(router, http.MethodGet, path)
		if got := w.Header().Get("Content-Security-Policy"); got != middlewares.ContentSecurityPolicy {
			t.Errorf("GET %s Content-Security-Policy = %q, want the shared policy", path, got)
		}
		if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET %s X-Content-Type-Options = %q, want nosniff", path, got)
		}
	}
}

// Verifies the body-size override table in initRouter matches real routes: a
// renamed image route would otherwise silently fall back to the 1 MB default.
func TestInitRouterBodySizeLimits(t *testing.T) {
	router := initRouter(models.ConfigStruct{})

	cases := []struct {
		path          string
		contentLength int64
		want          int
	}{
		// Under the image limit: passes the size check, stopped by auth instead.
		{"/api/auth/wishes", 5 << 20, http.StatusUnauthorized},
		{"/api/auth/wishes/00000000-0000-0000-0000-000000000001", 5 << 20, http.StatusUnauthorized},
		{"/api/auth/users/update", 5 << 20, http.StatusUnauthorized},
		{"/api/auth/users/update", 16 << 20, http.StatusRequestEntityTooLarge},
		// Routes without an override keep the default.
		{"/api/auth/groups", 2 << 20, http.StatusRequestEntityTooLarge},
		{"/api/open/users", 2 << 20, http.StatusRequestEntityTooLarge},
	}
	for _, c := range cases {
		request := httptest.NewRequest(http.MethodPost, c.path, strings.NewReader("{}"))
		request.ContentLength = c.contentLength
		w := httptest.NewRecorder()
		router.ServeHTTP(w, request)
		if w.Code != c.want {
			t.Errorf("POST %s with Content-Length %d = %d, want %d", c.path, c.contentLength, w.Code, c.want)
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

// setupRecoveryDB points database.Instance at a fresh in-memory SQLite DB
// holding the tables the account-recovery actions touch, restoring the
// previous instance afterwards.
func setupRecoveryDB(t *testing.T) {
	t.Helper()
	dbSQL, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { dbSQL.Close() })
	dbSQL.SetMaxOpenConns(1)

	instance, err := gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	if err := instance.AutoMigrate(&models.User{}, &models.MFARecoveryCode{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	origInstance := database.Instance
	t.Cleanup(func() { database.Instance = origInstance })
	database.Instance = instance
}

// createRecoveryUser inserts an enabled user with MFA switched on.
func createRecoveryUser(t *testing.T, email string) models.User {
	t.Helper()
	enabled, mfaEnabled, secret := true, true, "encrypted-secret"
	user := models.User{FirstName: "Ada", Email: &email, Enabled: &enabled, MFAEnabled: &mfaEnabled, MFASecret: &secret}
	user.ID = uuid.New()
	if r := database.Instance.Create(&user); r.Error != nil {
		t.Fatalf("failed to create user: %v", r.Error)
	}
	return user
}

// captureLogs attaches a test hook to logger.Log for the duration of the test.
func captureLogs(t *testing.T) *logrusTest.Hook {
	t.Helper()
	hook := logrusTest.NewLocal(logger.Log)
	t.Cleanup(func() { logger.Log.ReplaceHooks(make(logrus.LevelHooks)) })
	return hook
}

// logged reports whether an entry at level containing substr was captured.
func logged(hook *logrusTest.Hook, level logrus.Level, substr string) bool {
	for _, entry := range hook.AllEntries() {
		if entry.Level == level && strings.Contains(entry.Message, substr) {
			return true
		}
	}
	return false
}

func TestRunAccountRecoveryActionsNoneIsNoop(t *testing.T) {
	hook := captureLogs(t)
	// No DB is set up: with no actions requested, nothing may touch it.
	runAccountRecoveryActions(startupActions{generateInvite: true})
	if n := len(hook.AllEntries()); n != 0 {
		t.Errorf("logged %d entries with no recovery actions, want 0: %+v", n, hook.AllEntries())
	}
}

func TestRunAccountRecoveryActionsResetMFA(t *testing.T) {
	setupRecoveryDB(t)
	user := createRecoveryUser(t, "ada@example.com")
	hook := captureLogs(t)

	runAccountRecoveryActions(startupActions{resetMFAEmail: "ada@example.com"})

	var got models.User
	if r := database.Instance.First(&got, "id = ?", user.ID); r.Error != nil {
		t.Fatalf("failed to reload user: %v", r.Error)
	}
	if got.MFAEnabled == nil || *got.MFAEnabled || got.MFASecret != nil {
		t.Errorf("after resetmfa MFAEnabled = %v, MFASecret = %v; want false and nil", got.MFAEnabled, got.MFASecret)
	}
	if !logged(hook, logrus.WarnLevel, "removed MFA for user "+user.ID.String()) {
		t.Errorf("missing resetmfa success warning; entries: %+v", hook.AllEntries())
	}
}

func TestRunAccountRecoveryActionsUnknownEmailLogsErrors(t *testing.T) {
	setupRecoveryDB(t)
	hook := captureLogs(t)

	// Must log and carry on rather than exit: these flags often linger as env
	// vars after the user they named is gone.
	runAccountRecoveryActions(startupActions{resetMFAEmail: "ghost@example.com", resetPasswordEmail: "ghost@example.com"})

	if !logged(hook, logrus.ErrorLevel, "resetmfa: failed to remove MFA for 'ghost@example.com'") {
		t.Errorf("missing resetmfa error; entries: %+v", hook.AllEntries())
	}
	if !logged(hook, logrus.ErrorLevel, "resetpassword: failed to issue a reset link for 'ghost@example.com'") {
		t.Errorf("missing resetpassword error; entries: %+v", hook.AllEntries())
	}
}

func TestRunAccountRecoveryActionsResetPassword(t *testing.T) {
	cases := []struct {
		name          string
		oidcOnly      bool
		wantOIDCNotes bool
	}{
		{"local login enabled", false, false},
		{"OIDC-only instance warns link is refused", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			origConfig := config.ConfigFile
			t.Cleanup(func() { config.ConfigFile = origConfig })
			config.ConfigFile.PoenskelistenExternalURL = "https://wishes.example.com"
			config.ConfigFile.LocalLoginDisabled = c.oidcOnly
			config.ConfigFile.OIDCEnabled = c.oidcOnly
			config.ConfigFile.OIDCIssuerURL = "https://idp.example.com"
			config.ConfigFile.OIDCClientID = "client"

			setupRecoveryDB(t)
			user := createRecoveryUser(t, "ada@example.com")
			hook := captureLogs(t)

			runAccountRecoveryActions(startupActions{resetPasswordEmail: "ada@example.com"})

			var got models.User
			if r := database.Instance.First(&got, "id = ?", user.ID); r.Error != nil {
				t.Fatalf("failed to reload user: %v", r.Error)
			}
			if got.ResetCode == nil || *got.ResetCode == "" {
				t.Fatal("no reset code was stored for the user")
			}
			wantLink := "https://wishes.example.com/login?reset_code=" + *got.ResetCode
			if !logged(hook, logrus.WarnLevel, wantLink) {
				t.Errorf("missing warning containing %q; entries: %+v", wantLink, hook.AllEntries())
			}
			if gotNote := logged(hook, logrus.WarnLevel, "password login is disabled"); gotNote != c.wantOIDCNotes {
				t.Errorf("OIDC-only warning logged = %v, want %v", gotNote, c.wantOIDCNotes)
			}
		})
	}
}

func TestParseFlagsCleansAppText(t *testing.T) {
	withArgs(t, "-name", "   ", "-description", "  A wishlist app \n")

	out, _, _, err := parseFlags(models.ConfigStruct{PoenskelistenName: "Wishes"})
	if err != nil {
		t.Fatalf("parseFlags returned error: %v", err)
	}
	if out.PoenskelistenName != "Pønskelisten" {
		t.Errorf("PoenskelistenName = %q, want the default for a blank -name", out.PoenskelistenName)
	}
	if out.PoenskelistenDescription != "A wishlist app" {
		t.Errorf("PoenskelistenDescription = %q, want it trimmed", out.PoenskelistenDescription)
	}
}

// TestInitRouterEscapesAppText checks that an app name full of characters
// special to HTML, JS template literals and JSON can't break out of the
// text/template-rendered JS and JSON files.
func TestInitRouterEscapesAppText(t *testing.T) {
	name := "A`${alert(1)}\\ & <b>\"'"
	router := initRouter(models.ConfigStruct{PoenskelistenName: name, PoenskelistenDescription: name})

	w := serve(router, http.MethodGet, "/js/frontpage.js")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /js/frontpage.js = %d, want 200", w.Code)
	}
	escaped := "A&#96;&#36;{alert(1)}&#92; &amp; &lt;b&gt;&#34;&#39;"
	if !strings.Contains(w.Body.String(), escaped) {
		t.Errorf("frontpage.js does not contain the escaped app name %q", escaped)
	}
	if strings.Contains(w.Body.String(), name) || strings.Contains(w.Body.String(), "${alert") {
		t.Error("frontpage.js contains the raw app name")
	}

	w = serve(router, http.MethodGet, "/manifest.json")
	var manifest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("manifest.json is not valid JSON: %v\n%s", err, w.Body.String())
	}
	if manifest.Name != name || manifest.Description != name {
		t.Errorf("manifest.json name/description = %q, %q; want %q", manifest.Name, manifest.Description, name)
	}
}

// routerTestModels is the full schema, for tests that run requests through
// the real router.
var routerTestModels = []interface{}{
	&models.User{}, &models.Invite{}, &models.Group{}, &models.GroupMembership{},
	&models.Wishlist{}, &models.WishlistMembership{}, &models.WishlistCollaborator{},
	&models.WishCategory{}, &models.Wish{}, &models.WishClaim{}, &models.News{},
	&models.MFARecoveryCode{}, &models.Session{}, &models.OAuthClient{},
	&models.AuthorizationCode{}, &models.OAuthConsent{},
}

// setupRouterTest gives the real router a fresh database and an OAuth signing
// key, and returns a function minting first-party access tokens.
func setupRouterTest(t *testing.T) (*gin.Engine, func(userID uuid.UUID, admin bool) string) {
	t.Helper()
	dbSQL, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { dbSQL.Close() })
	dbSQL.SetMaxOpenConns(1)
	instance, err := gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	if err := instance.AutoMigrate(routerTestModels...); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	origInstance, origConfig := database.Instance, config.ConfigFile
	t.Cleanup(func() { database.Instance, config.ConfigFile = origInstance, origConfig })
	database.Instance = instance

	keyPEM, kid, err := config.GenerateOAuthSigningKey(config.OAuthAlgES256)
	if err != nil {
		t.Fatalf("failed to generate signing key: %v", err)
	}
	config.ConfigFile.PoenskelistenExternalURL = "https://wishlist.example.com"
	config.ConfigFile.OAuthSigningKey = keyPEM
	config.ConfigFile.OAuthSigningKeyID = kid

	token := func(userID uuid.UUID, admin bool) string {
		signed, err := auth.GenerateOAuthAccessToken(userID, models.FirstPartyClientID, config.APIResource(), "openid profile email", admin, true)
		if err != nil {
			t.Fatalf("failed to sign access token: %v", err)
		}
		return "Bearer " + signed
	}
	return initRouter(config.ConfigFile), token
}

// snapshotRouterTables dumps every row of every table, deleted ones included,
// so a test can check that nothing at all was written.
func snapshotRouterTables(t *testing.T) string {
	t.Helper()
	dump := map[string][]map[string]interface{}{}
	for _, model := range routerTestModels {
		statement := &gorm.Statement{DB: database.Instance}
		if err := statement.Parse(model); err != nil {
			t.Fatalf("failed to parse model: %v", err)
		}
		rows := []map[string]interface{}{}
		if err := database.Instance.Table(statement.Schema.Table).Order("id").Find(&rows).Error; err != nil {
			t.Fatalf("failed to read %s: %v", statement.Schema.Table, err)
		}
		dump[statement.Schema.Table] = rows
	}
	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatalf("failed to encode snapshot: %v", err)
	}
	return string(raw)
}

func createRouterRecord(t *testing.T, record interface{}) {
	t.Helper()
	if err := database.Instance.Create(record).Error; err != nil {
		t.Fatalf("failed to create %T: %v", record, err)
	}
}

func createRouterUser(t *testing.T, name string, admin bool) models.User {
	t.Helper()
	email := strings.ToLower(name) + "-" + uuid.NewString() + "@example.com"
	enabled, mfa := true, true
	user := models.User{FirstName: name, LastName: "Test", Email: &email, Enabled: &enabled, Verified: &enabled, MFAEnabled: &mfa, Admin: admin}
	user.ID = uuid.New()
	createRouterRecord(t, &user)
	return user
}

// Every route that takes an ID must refuse the nil UUID, wherever the request
// carries it, without answering with anyone's data and without writing
// anything.
func TestRoutesRefuseNilID(t *testing.T) {
	router, token := setupRouterTest(t)
	now := time.Now()
	truth := true

	admin := createRouterUser(t, "Admin", true)
	other := createRouterUser(t, "Other", false)
	claimer := createRouterUser(t, "Claimer", false)
	caller := createRouterUser(t, "Caller", false)

	otherGroup := models.Group{Name: "Other group", Description: "Other group", Enabled: true, OwnerID: other.ID}
	otherGroup.ID = uuid.New()
	createRouterRecord(t, &otherGroup)
	otherList := models.Wishlist{Name: "Other list", Enabled: true, OwnerID: other.ID, Date: &now, Claimable: &truth}
	otherList.ID = uuid.New()
	createRouterRecord(t, &otherList)
	publicList := models.Wishlist{Name: "Other public list", Enabled: true, OwnerID: other.ID, Date: &now, Public: &truth, PublicHash: uuid.New()}
	publicList.ID = uuid.New()
	createRouterRecord(t, &publicList)
	otherWish := models.Wish{Name: "Other wish", Enabled: true, OwnerID: other.ID, WishlistID: otherList.ID}
	otherWish.ID = uuid.New()
	createRouterRecord(t, &otherWish)
	otherCategory := models.WishCategory{Name: "Other category", Enabled: true, OwnerID: other.ID, WishlistID: otherList.ID}
	otherCategory.ID = uuid.New()
	createRouterRecord(t, &otherCategory)
	for _, memberID := range []uuid.UUID{other.ID, claimer.ID} {
		membership := models.GroupMembership{GroupID: otherGroup.ID, MemberID: memberID, Enabled: true}
		membership.ID = uuid.New()
		createRouterRecord(t, &membership)
	}
	share := models.WishlistMembership{GroupID: otherGroup.ID, WishlistID: otherList.ID, Enabled: true}
	share.ID = uuid.New()
	createRouterRecord(t, &share)
	claim := models.WishClaim{WishID: otherWish.ID, UserID: claimer.ID, Enabled: true}
	claim.ID = uuid.New()
	createRouterRecord(t, &claim)
	collaborator := models.WishlistCollaborator{WishlistID: otherList.ID, UserID: claimer.ID, Enabled: true}
	collaborator.ID = uuid.New()
	createRouterRecord(t, &collaborator)
	for _, user := range []models.User{other, claimer} {
		code := models.MFARecoveryCode{UserID: user.ID, CodeHash: "hash"}
		code.ID = uuid.New()
		createRouterRecord(t, &code)
		session := models.Session{UserID: user.ID, RefreshTokenHash: uuid.NewString(), Kind: models.SessionKindWebRefresh, IssuedAt: now, ExpiresAt: now.Add(time.Hour)}
		session.ID = uuid.New()
		createRouterRecord(t, &session)
	}
	invite := models.Invite{Code: "INVITECODE", Enabled: &truth}
	invite.ID = uuid.New()
	createRouterRecord(t, &invite)
	news := models.News{Title: "News", Body: "Body", Enabled: true, Date: now}
	news.ID = uuid.New()
	createRouterRecord(t, &news)

	// The caller owns a wishlist, a wish and a group, so ownership and
	// membership checks have records of theirs to match.
	callerGroup := models.Group{Name: "Caller group", Enabled: true, OwnerID: caller.ID}
	callerGroup.ID = uuid.New()
	createRouterRecord(t, &callerGroup)
	callerMembership := models.GroupMembership{GroupID: callerGroup.ID, MemberID: caller.ID, Enabled: true}
	callerMembership.ID = uuid.New()
	createRouterRecord(t, &callerMembership)
	callerList := models.Wishlist{Name: "Caller list", Enabled: true, OwnerID: caller.ID, Date: &now, Claimable: &truth}
	callerList.ID = uuid.New()
	createRouterRecord(t, &callerList)
	callerWish := models.Wish{Name: "Caller wish", Enabled: true, OwnerID: caller.ID, WishlistID: callerList.ID}
	callerWish.ID = uuid.New()
	createRouterRecord(t, &callerWish)

	nilID := uuid.Nil.String()
	future := now.AddDate(1, 0, 0).Format("2006-01-02")
	callerToken, adminToken := token(caller.ID, false), token(admin.ID, true)
	ownList, ownGroup, ownWish := callerList.ID.String(), callerGroup.ID.String(), callerWish.ID.String()

	cases := []struct {
		token, method, path, body string
	}{
		{callerToken, "GET", "/api/auth/wishes?wishlist=" + nilID, ""},
		{callerToken, "GET", "/api/auth/wishes?wishlist=00000000000000000000000000000000", ""},
		{callerToken, "GET", "/api/auth/wishes?wishlist=" + url.QueryEscape("{"+nilID+"}"), ""},
		{callerToken, "GET", "/api/auth/wishes?wishlist=" + url.QueryEscape("urn:uuid:"+nilID), ""},
		{callerToken, "POST", "/api/auth/wishes?wishlist=" + nilID, `{"name":"Nil wish name"}`},
		{callerToken, "POST", "/api/auth/wishes?wishlist=" + ownList, `{"name":"Nil category wish","category_id":"` + nilID + `"}`},
		{callerToken, "GET", "/api/auth/wishes/" + nilID, ""},
		{callerToken, "POST", "/api/auth/wishes/" + nilID, `{"name":"Renamed wish"}`},
		{callerToken, "POST", "/api/auth/wishes/" + ownWish, `{"name":"Caller wish","category_id":"` + nilID + `"}`},
		{callerToken, "DELETE", "/api/auth/wishes/" + nilID, ""},
		{callerToken, "POST", "/api/auth/wishes/" + nilID + "/claim", `{}`},
		{callerToken, "POST", "/api/auth/wishes/" + nilID + "/unclaim", `{}`},
		{callerToken, "POST", "/api/auth/wishes/" + otherWish.ID.String() + "/claim", `{"wishlist_id":"` + nilID + `"}`},
		{callerToken, "GET", "/api/both/wishes/" + nilID + "/image", ""},
		{"", "GET", "/api/both/wishes/" + nilID + "/image", ""},
		{"", "GET", "/api/open/wishlists/public/" + nilID, ""},
		{callerToken, "GET", "/api/auth/wishlists/" + nilID, ""},
		{callerToken, "POST", "/api/auth/wishlists/" + nilID, `{"name":"Renamed list","date":"` + future + `"}`},
		{callerToken, "DELETE", "/api/auth/wishlists/" + nilID, ""},
		{callerToken, "DELETE", "/api/auth/wishlists/" + ownList + "?group=" + nilID, ""},
		{callerToken, "POST", "/api/auth/wishlists/" + nilID + "/join", `{"groups":["` + ownGroup + `"]}`},
		{callerToken, "POST", "/api/auth/wishlists/" + ownList + "/join", `{"groups":["` + nilID + `"]}`},
		{callerToken, "POST", "/api/auth/wishlists/" + nilID + "/remove", `{"group_id":"` + otherGroup.ID.String() + `"}`},
		{callerToken, "POST", "/api/auth/wishlists/" + ownList + "/remove", `{"group_id":"` + nilID + `"}`},
		{callerToken, "POST", "/api/auth/wishlists/" + ownList + "/remove", `{}`},
		{callerToken, "POST", "/api/auth/wishlists/" + nilID + "/collaborate", `{"users":["` + admin.ID.String() + `"]}`},
		{callerToken, "POST", "/api/auth/wishlists/" + ownList + "/collaborate", `{"users":["` + nilID + `"]}`},
		{callerToken, "POST", "/api/auth/wishlists/" + nilID + "/un-collaborate", `{"user_id":"` + claimer.ID.String() + `"}`},
		{callerToken, "POST", "/api/auth/wishlists/" + ownList + "/un-collaborate", `{"user_id":"` + nilID + `"}`},
		{callerToken, "GET", "/api/auth/wishlists/" + nilID + "/categories", ""},
		{callerToken, "GET", "/api/auth/wishlists?group=" + nilID, ""},
		{callerToken, "GET", "/api/auth/wishlists?notAMemberOfGroupID=" + nilID, ""},
		{callerToken, "POST", "/api/auth/wishlists", `{"name":"Nil group list","date":"` + future + `","groups":["` + nilID + `"]}`},
		{callerToken, "POST", "/api/auth/wishlists?groupContextID=" + nilID, `{"name":"Nil context list","date":"` + future + `"}`},
		{callerToken, "GET", "/api/auth/groups/" + nilID, ""},
		{callerToken, "GET", "/api/auth/groups/" + nilID + "/members", ""},
		{callerToken, "POST", "/api/auth/groups/" + nilID, `{"name":"Renamed group","description":"Renamed"}`},
		{callerToken, "DELETE", "/api/auth/groups/" + nilID, ""},
		{callerToken, "POST", "/api/auth/groups/" + nilID + "/join", `{"members":["` + admin.ID.String() + `"]}`},
		{callerToken, "POST", "/api/auth/groups/" + ownGroup + "/join", `{"members":["` + nilID + `"]}`},
		{callerToken, "POST", "/api/auth/groups/" + nilID + "/leave", ""},
		{callerToken, "POST", "/api/auth/groups/" + nilID + "/remove", `{"member_id":"` + claimer.ID.String() + `"}`},
		{callerToken, "POST", "/api/auth/groups/" + ownGroup + "/remove", `{"member_id":"` + nilID + `"}`},
		{callerToken, "POST", "/api/auth/groups/" + nilID + "/add", `{"wishlists":["` + ownList + `"]}`},
		{callerToken, "POST", "/api/auth/groups/" + ownGroup + "/add", `{"wishlists":["` + nilID + `"]}`},
		{callerToken, "POST", "/api/auth/groups", `{"name":"Nil member group","members":["` + nilID + `"]}`},
		{callerToken, "GET", "/api/auth/groups?memberOfWishlistID=" + nilID, ""},
		{callerToken, "GET", "/api/auth/groups?notAMemberOfWishlistID=" + nilID, ""},
		{callerToken, "GET", "/api/auth/news/" + nilID, ""},
		{callerToken, "GET", "/api/auth/users/" + nilID, ""},
		{callerToken, "GET", "/api/auth/users/" + nilID + "/image", ""},
		{callerToken, "GET", "/api/auth/users?notAMemberOfGroupID=" + nilID, ""},
		{callerToken, "GET", "/api/auth/users?notACollaboratorOfWishlistID=" + nilID, ""},
		{adminToken, "DELETE", "/api/admin/users/" + nilID, ""},
		{adminToken, "DELETE", "/api/admin/users/" + nilID + "/mfa", ""},
		{adminToken, "DELETE", "/api/admin/users/" + nilID + "/sessions", ""},
		{adminToken, "DELETE", "/api/admin/invites/" + nilID, ""},
		{adminToken, "DELETE", "/api/admin/news/" + nilID, ""},
		{adminToken, "POST", "/api/admin/news/" + nilID, `{"title":"Renamed news","body":"Renamed"}`},
		// A token for the nil UUID can't be forged without the signing key,
		// but it must not stand for every account either.
		{token(uuid.Nil, false), "GET", "/api/auth/wishlists", ""},
		{token(uuid.Nil, false), "GET", "/api/auth/wishes?wishlist=" + otherList.ID.String(), ""},
		{token(uuid.Nil, false), "GET", "/api/auth/groups", ""},
	}

	send := func(token, method, path, body string) *httptest.ResponseRecorder {
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	// The same requests with real IDs work, so the refusals below come from
	// the nil UUID and not from a broken setup.
	for _, control := range []struct{ token, path, want string }{
		{callerToken, "/api/auth/wishes?wishlist=" + ownList, "Caller wish"},
		{callerToken, "/api/auth/groups/" + ownGroup, "Caller group"},
		{adminToken, "/api/admin/invites", "INVITECODE"},
		{"", "/api/open/wishlists/public/" + publicList.PublicHash.String(), "Other public list"},
	} {
		if w := send(control.token, "GET", control.path, ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), control.want) {
			t.Fatalf("GET %s = %d %s, want 200 with %q", control.path, w.Code, w.Body.String(), control.want)
		}
	}

	secrets := []string{*other.Email, *claimer.Email, "Other", otherList.ID.String(), publicList.PublicHash.String()}
	before := snapshotRouterTables(t)
	for _, c := range cases {
		w := send(c.token, c.method, c.path, c.body)
		switch w.Code {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		default:
			t.Errorf("%s %s = %d, want 400 or 404: %s", c.method, c.path, w.Code, w.Body.String())
		}
		for _, secret := range secrets {
			if strings.Contains(w.Body.String(), secret) {
				t.Errorf("%s %s leaked %q: %s", c.method, c.path, secret, w.Body.String())
			}
		}
	}
	if after := snapshotRouterTables(t); after != before {
		t.Errorf("requests naming the nil UUID changed the database:\nbefore %s\nafter  %s", before, after)
	}
}

// The Content-Security-Policy's script-src is 'self' only, so any inline script
// in the frontend is silently refused by the browser: the button simply does
// nothing. This keeps the markup in the shape the CSP needs (and that keeps
// API data out of code): no inline handlers, inline <script> blocks or
// javascript: URLs, and every action named in markup registered via
// registerActions (web/js/actions.js).
func TestFrontendHasNoInlineScript(t *testing.T) {
	files, err := filepath.Glob("./web/html/*.html")
	if err != nil {
		t.Fatal(err)
	}
	jsFiles, err := filepath.Glob("./web/js/*.js")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, jsFiles...)

	inlineHandler := regexp.MustCompile(`(?i)[\s"'](on[a-z]+)\s*=\s*["'\\]`)
	inlineScript := regexp.MustCompile(`(?i)<script(\s[^>]*)?>`)
	scriptURL := regexp.MustCompile(`(?i)(href|src|action|formaction)\s*=\s*["']\s*javascript:`)
	actionAttr := regexp.MustCompile(`data-on-[a-z]+="([A-Za-z_$][\w$]*)"`)
	actionHelper := regexp.MustCompile(`\bon(?:Click|Submit|Change)\(\s*['"]([A-Za-z_$][\w$]*)['"]`)
	registration := regexp.MustCompile(`(?s)registerActions\(\{(.*?)\}\);`)
	registeredName := regexp.MustCompile(`([A-Za-z_$][\w$]*)\s*[,:]`)

	used := map[string]string{}
	registered := map[string]bool{}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		src := string(content)

		for _, m := range inlineHandler.FindAllStringSubmatch(src, -1) {
			t.Errorf("%s: inline %s handler; use data-on-* / onClick() and registerActions instead", file, m[1])
		}
		for _, m := range scriptURL.FindAllString(src, -1) {
			t.Errorf("%s: javascript: URL %q", file, m)
		}
		if strings.HasSuffix(file, ".html") {
			for _, m := range inlineScript.FindAllStringSubmatch(src, -1) {
				if !strings.Contains(strings.ToLower(m[1]), "src=") {
					t.Errorf("%s: inline <script> block; move it to a web/js file", file)
				}
			}
		}

		for _, re := range []*regexp.Regexp{actionAttr, actionHelper} {
			for _, m := range re.FindAllStringSubmatch(src, -1) {
				used[m[1]] = file
			}
		}
		for _, block := range registration.FindAllStringSubmatch(src, -1) {
			for _, m := range registeredName.FindAllStringSubmatch(block[1]+",", -1) {
				registered[m[1]] = true
			}
		}
	}

	if len(used) == 0 {
		t.Fatal("found no actions in markup; the scan patterns no longer match the frontend")
	}
	for name, file := range used {
		if !registered[name] {
			t.Errorf("%s: action %q is used in markup but never passed to registerActions", file, name)
		}
	}
}

// The endpoints that check a secret are each rate-limited per client IP, and
// every MFA code check draws on one shared budget. httptest gives every
// request the same RemoteAddr, so they all count against one client.
func TestInitRouterRateLimitsSecretChecks(t *testing.T) {
	cases := []struct {
		method string
		path   string
		limit  int
	}{
		{http.MethodPost, "/api/open/tokens/register", 20},
		{http.MethodPost, "/api/open/tokens/mfa", 10},
		{http.MethodPost, "/api/open/users/mfa/activate", 10},
		{http.MethodPost, "/api/auth/users/mfa/disable", 10},
		{http.MethodGet, "/oauth/authorize", 60},
		{http.MethodPost, "/oauth/token", 120},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			// A fresh router per case, so every limiter starts empty.
			router, token := setupRouterTest(t)
			user := createRouterUser(t, "Limited", false)
			send := func() *httptest.ResponseRecorder {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(c.method, c.path, strings.NewReader("{}"))
				req.Header.Set("Authorization", token(user.ID, false))
				router.ServeHTTP(w, req)
				return w
			}

			for i := 0; i < c.limit; i++ {
				if w := send(); w.Code == http.StatusTooManyRequests {
					t.Fatalf("request %d of %d was rate-limited", i+1, c.limit)
				}
			}
			w := send()
			if w.Code != http.StatusTooManyRequests {
				t.Errorf("request over the limit = %d, want 429", w.Code)
			}
			if w.Header().Get("Retry-After") == "" {
				t.Error("429 response has no Retry-After header")
			}
		})
	}

	t.Run("MFA routes share a budget", func(t *testing.T) {
		router, token := setupRouterTest(t)
		user := createRouterUser(t, "Limited", false)
		send := func(path string) int {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
			req.Header.Set("Authorization", token(user.ID, false))
			router.ServeHTTP(w, req)
			return w.Code
		}

		for i := 0; i < 10; i++ {
			send("/api/open/tokens/mfa")
		}
		for _, path := range []string{"/api/open/users/mfa/activate", "/api/auth/users/mfa/disable"} {
			if code := send(path); code != http.StatusTooManyRequests {
				t.Errorf("POST %s after the MFA budget was spent = %d, want 429", path, code)
			}
		}
	})
}

// X-Forwarded-For only picks the client IP the rate limits count against when
// the peer is a configured proxy; from anyone else, a fresh header per request
// must not buy a fresh budget. httptest's peer address is 192.0.2.1.
func TestInitRouterTrustsForwardedForOnlyFromTrustedProxies(t *testing.T) {
	cases := []struct {
		name          string
		trusted       string
		wantLimitedAt int // 0: never limited within the requests sent
	}{
		{"no trusted proxies", "", 21},
		{"other proxy trusted", "10.0.0.0/8", 21},
		{"peer is a trusted proxy", "192.0.2.0/24", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupRouterTest(t)
			cfg := config.ConfigFile
			cfg.PoenskelistenTrustedProxies = c.trusted
			router := initRouter(cfg)

			limitedAt := 0
			for i := 1; i <= 25; i++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(http.MethodPost, "/api/open/tokens/register", strings.NewReader("{}"))
				req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
				router.ServeHTTP(w, req)
				if w.Code == http.StatusTooManyRequests {
					limitedAt = i
					break
				}
			}
			if limitedAt != c.wantLimitedAt {
				t.Errorf("first 429 at request %d, want %d", limitedAt, c.wantLimitedAt)
			}
		})
	}
}

// setupRunDir runs the test in a fresh working directory, so run's files/
// directory (config, log, SQLite database) lands there, with the repository's
// web/ linked in for initRouter. It restores every global run replaces.
func setupRunDir(t *testing.T) string {
	t.Helper()
	webDir, err := filepath.Abs("web")
	if err != nil {
		t.Fatalf("failed to resolve web directory: %v", err)
	}
	dir := t.TempDir()
	if err := os.Symlink(webDir, filepath.Join(dir, "web")); err != nil {
		t.Fatalf("failed to link web directory: %v", err)
	}
	t.Chdir(dir)

	origConfig, origInstance, origLog, origLocal := config.ConfigFile, database.Instance, logger.Log, time.Local
	t.Cleanup(func() {
		if database.Instance != nil && database.Instance != origInstance {
			if sqlDB, err := database.Instance.DB(); err == nil {
				sqlDB.Close()
			}
		}
		config.ConfigFile, database.Instance, logger.Log, time.Local = origConfig, origInstance, origLog, origLocal
	})
	return dir
}

// readSavedConfig decodes files/config.json in the working directory.
func readSavedConfig(t *testing.T) models.ConfigStruct {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("files", "config.json"))
	if err != nil {
		t.Fatalf("failed to read config.json: %v", err)
	}
	var saved models.ConfigStruct
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to parse config.json: %v", err)
	}
	return saved
}

// TestRunFirstStartServesFrontend is a first start in an empty directory: run
// creates the config, log and SQLite database, then hands serve a router that
// renders the frontend.
func TestRunFirstStartServesFrontend(t *testing.T) {
	setupRunDir(t)
	withArgs(t)

	var address string
	var status int
	err := run(func(router *gin.Engine, addr string) error {
		address = addr
		status = serve(router, http.MethodGet, "/").Code
		return nil
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if address != ":8080" {
		t.Errorf("serve address = %q, want :8080", address)
	}
	if status != http.StatusOK {
		t.Errorf("GET / = %d, want 200", status)
	}
	for _, name := range []string{"config.json", "poenskelisten.log", "data.db"} {
		if _, err := os.Stat(filepath.Join("files", name)); err != nil {
			t.Errorf("files/%s not created: %v", name, err)
		}
	}
	if !database.Instance.Migrator().HasTable(&models.User{}) {
		t.Error("database was not migrated")
	}
}

// TestRunPersistsFlagsAndGeneratesInvite checks that a configuration flag is
// saved to config.json and that -generateinvite adds an invite.
func TestRunPersistsFlagsAndGeneratesInvite(t *testing.T) {
	setupRunDir(t)
	withArgs(t, "-port", "9123", "-generateinvite", "true")

	var address string
	err := run(func(router *gin.Engine, addr string) error {
		address = addr
		return nil
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if address != ":9123" {
		t.Errorf("serve address = %q, want :9123", address)
	}
	if port := readSavedConfig(t).PoenskelistenPort; port != 9123 {
		t.Errorf("saved port = %d, want 9123", port)
	}
	var invites int64
	if err := database.Instance.Model(&models.Invite{}).Count(&invites).Error; err != nil {
		t.Fatalf("failed to count invites: %v", err)
	}
	if invites != 1 {
		t.Errorf("invites = %d, want 1", invites)
	}
}

// TestRunUnknownTimezoneFallsBackToParis checks that an unloadable time zone
// is replaced in the saved config rather than stopping startup.
func TestRunUnknownTimezoneFallsBackToParis(t *testing.T) {
	setupRunDir(t)
	withArgs(t, "-timezone", "Not/AZone")

	if err := run(func(*gin.Engine, string) error { return nil }); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if tz := readSavedConfig(t).Timezone; tz != "Europe/Paris" {
		t.Errorf("saved timezone = %q, want Europe/Paris", tz)
	}
}

// TestRunReturnsServeError checks that a server failure comes back from run
// (main exits non-zero on it) instead of being swallowed.
func TestRunReturnsServeError(t *testing.T) {
	setupRunDir(t)
	withArgs(t)

	serveErr := fmt.Errorf("address already in use")
	err := run(func(*gin.Engine, string) error { return serveErr })
	if !errors.Is(err, serveErr) {
		t.Errorf("run error = %v, want it to wrap %v", err, serveErr)
	}
}

// TestRunStartupFailures covers each step that stops startup.
func TestRunStartupFailures(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T)
		args    []string
		want    string
	}{
		{
			name: "files is not a directory",
			prepare: func(t *testing.T) {
				if err := os.WriteFile("files", nil, 0600); err != nil {
					t.Fatal(err)
				}
			},
			want: "failed to create 'files' directory",
		},
		{
			name: "config is not JSON",
			prepare: func(t *testing.T) {
				if err := os.Mkdir("files", 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join("files", "config.json"), []byte("{"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			want: "failed to load configuration file",
		},
		{
			name: "invalid flag",
			args: []string{"-additionalurls", "not a url"},
			want: "failed to parse input flags",
		},
		{
			name: "unknown database type",
			args: []string{"-dbtype", "oracle"},
			want: "failed to connect to database",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupRunDir(t)
			withArgs(t, tc.args...)
			if tc.prepare != nil {
				tc.prepare(t)
			}

			served := false
			err := run(func(*gin.Engine, string) error {
				served = true
				return nil
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("run error = %v, want it to contain %q", err, tc.want)
			}
			if served {
				t.Error("serve was called despite the startup failure")
			}
		})
	}
}
