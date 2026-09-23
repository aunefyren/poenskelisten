package main

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	textTemplate "text/template"

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
