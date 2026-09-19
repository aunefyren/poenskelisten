package config

import (
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// withTempConfigFile points the package-global configFilePath at a fresh path
// inside a temp dir for the duration of the test, and restores both it and
// ConfigFile afterward. Same package as config.go, so configFilePath (an
// unexported var) is directly assignable here.
func withTempConfigFile(t *testing.T) string {
	t.Helper()

	origPath := configFilePath
	origConfig := ConfigFile
	t.Cleanup(func() {
		configFilePath = origPath
		ConfigFile = origConfig
	})

	configFilePath = filepath.Join(t.TempDir(), "config.json")
	ConfigFile = models.ConfigStruct{}
	return configFilePath
}

func TestCreateConfigFile(t *testing.T) {
	path := withTempConfigFile(t)

	if err := CreateConfigFile(); err != nil {
		t.Fatalf("CreateConfigFile returned error: %v", err)
	}

	if ConfigFile.PoenskelistenPort != 8080 {
		t.Errorf("PoenskelistenPort = %d, want 8080", ConfigFile.PoenskelistenPort)
	}
	if ConfigFile.DBType != "sqlite" {
		t.Errorf("DBType = %q, want sqlite", ConfigFile.DBType)
	}
	if ConfigFile.PrivateKey == "" {
		t.Error("expected a generated PrivateKey")
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected config.json to be written: %v", err)
	}
}

func TestSaveConfig(t *testing.T) {
	path := withTempConfigFile(t)
	ConfigFile.PoenskelistenName = "Saved Name"

	if err := SaveConfig(); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}
	var saved map[string]interface{}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("saved config is not valid JSON: %v", err)
	}
	if saved["poenskelisten_name"] != "Saved Name" {
		t.Errorf("poenskelisten_name = %v, want 'Saved Name'", saved["poenskelisten_name"])
	}
}

func TestSaveConfigWriteFailure(t *testing.T) {
	withTempConfigFile(t)
	// Point at a path whose parent directory doesn't exist, so the write fails.
	configFilePath = filepath.Join(t.TempDir(), "does-not-exist", "config.json")

	if err := SaveConfig(); err == nil {
		t.Error("expected an error when the parent directory doesn't exist")
	}
}

func TestLoadConfigCreatesFileWhenMissing(t *testing.T) {
	withTempConfigFile(t)

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if ConfigFile.PoenskelistenName != "Pønskelisten" {
		t.Errorf("PoenskelistenName = %q, want the default", ConfigFile.PoenskelistenName)
	}
	if ConfigFile.OAuthSigningKey == "" {
		t.Error("expected LoadConfig to have generated an OAuth signing key")
	}
}

func TestLoadConfigFillsDefaultsAndPersists(t *testing.T) {
	path := withTempConfigFile(t)

	// Write a mostly-empty config file: LoadConfig should fill in every default
	// and, because something changed, save the result back to disk.
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if ConfigFile.PoenskelistenPort != 8080 {
		t.Errorf("PoenskelistenPort = %d, want 8080", ConfigFile.PoenskelistenPort)
	}
	if ConfigFile.Timezone != "Europe/Paris" {
		t.Errorf("Timezone = %q, want Europe/Paris", ConfigFile.Timezone)
	}
	if ConfigFile.DBType != "mysql" {
		t.Errorf("DBType = %q, want mysql (the fallback default)", ConfigFile.DBType)
	}
	if ConfigFile.PoenskelistenLogLevel != logrus.InfoLevel.String() {
		t.Errorf("PoenskelistenLogLevel = %q, want %q", ConfigFile.PoenskelistenLogLevel, logrus.InfoLevel.String())
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read persisted config: %v", err)
	}
	var saved map[string]interface{}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("persisted config is not valid JSON: %v", err)
	}
	if saved["poenskelisten_port"] != float64(8080) {
		t.Errorf("persisted poenskelisten_port = %v, want 8080", saved["poenskelisten_port"])
	}
}

func TestLoadConfigSQLiteDefaultLocation(t *testing.T) {
	path := withTempConfigFile(t)
	if err := os.WriteFile(path, []byte(`{"db_type":"sqlite"}`), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if ConfigFile.DBLocation != "files/data.db" {
		t.Errorf("DBLocation = %q, want files/data.db", ConfigFile.DBLocation)
	}
}

func TestLoadConfigTestEnvironmentRequiresTestEmail(t *testing.T) {
	path := withTempConfigFile(t)
	if err := os.WriteFile(path, []byte(`{"poenskelisten_environment":"test"}`), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err == nil {
		t.Error("expected an error when environment is 'test' with no test e-mail configured")
	}
}

// readSavedConfig decodes the config file LoadConfig saved, to check what was
// persisted as opposed to what's only in memory.
func readSavedConfig(t *testing.T, path string) models.ConfigStruct {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}
	var saved models.ConfigStruct
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}
	return saved
}

// OIDC defaults must not be derived/persisted at load time: LoadConfig runs
// before flags/env vars apply, so an oidcenabled/externalurl given that way on
// first start would otherwise leave the redirect URL empty.
func TestLoadConfigDoesNotPersistOIDCDefaults(t *testing.T) {
	path := withTempConfigFile(t)
	body := `{"oidc_enabled":true,"poenskelisten_external_url":"https://wishlist.example.com"}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	saved := readSavedConfig(t, path)
	if saved.OIDCProviderName != "" || saved.OIDCRedirectURL != "" {
		t.Errorf("persisted OIDC defaults (name %q, redirect %q), want both empty", saved.OIDCProviderName, saved.OIDCRedirectURL)
	}
	if got := OIDCCallbackURL(); got != "https://wishlist.example.com/api/open/oidc/callback" {
		t.Errorf("OIDCCallbackURL() = %q, want it derived from the external URL", got)
	}
}

// Older versions persisted the defaults; values identical to them are cleared so
// the redirect URL follows later external-URL changes.
func TestLoadConfigClearsLegacyPersistedOIDCDefaults(t *testing.T) {
	path := withTempConfigFile(t)
	body := `{"oidc_enabled":true,"poenskelisten_external_url":"https://wishlist.example.com/",` +
		`"oidc_provider_name":"Single sign-on","oidc_redirect_url":"https://wishlist.example.com/api/open/oidc/callback"}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	saved := readSavedConfig(t, path)
	if saved.OIDCProviderName != "" || saved.OIDCRedirectURL != "" {
		t.Errorf("legacy defaults kept (name %q, redirect %q), want both cleared", saved.OIDCProviderName, saved.OIDCRedirectURL)
	}
}

func TestLoadConfigKeepsCustomOIDCSettings(t *testing.T) {
	path := withTempConfigFile(t)
	body := `{"oidc_enabled":true,"poenskelisten_external_url":"https://wishlist.example.com",` +
		`"oidc_provider_name":"Authelia","oidc_redirect_url":"https://sso.example.com/cb"}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if ConfigFile.OIDCProviderName != "Authelia" || ConfigFile.OIDCRedirectURL != "https://sso.example.com/cb" {
		t.Errorf("custom OIDC settings changed: name %q, redirect %q", ConfigFile.OIDCProviderName, ConfigFile.OIDCRedirectURL)
	}
}

func TestLoadConfigInvalidLogLevelFallsBackToInfo(t *testing.T) {
	path := withTempConfigFile(t)
	body := `{"poenskelisten_log_level":"not-a-real-level"}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if ConfigFile.PoenskelistenLogLevel != logrus.InfoLevel.String() {
		t.Errorf("PoenskelistenLogLevel = %q, want the info fallback", ConfigFile.PoenskelistenLogLevel)
	}
}

func TestLoadConfigValidLogLevelIsApplied(t *testing.T) {
	path := withTempConfigFile(t)
	body := `{"poenskelisten_log_level":"debug"}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}
	origLevel := logrus.GetLevel()
	t.Cleanup(func() { logrus.SetLevel(origLevel) })

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if logrus.GetLevel() != logrus.DebugLevel {
		t.Errorf("logrus level = %v, want debug", logrus.GetLevel())
	}
}

func TestLoadConfigMalformedJSON(t *testing.T) {
	path := withTempConfigFile(t)
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to seed config file: %v", err)
	}

	if err := LoadConfig(); err == nil {
		t.Error("expected an error for malformed config JSON")
	}
}

func TestLoadConfigNoChangesNeeded(t *testing.T) {
	// A config that already has every default filled in should not need to be
	// re-saved. We can't easily observe "did SaveConfig run" directly, so
	// instead assert the round-trip is idempotent and error-free.
	path := withTempConfigFile(t)
	if err := CreateConfigFile(); err != nil {
		t.Fatalf("failed to seed via CreateConfigFile: %v", err)
	}
	// CreateConfigFile already wrote a fully-defaulted file at path; reset
	// ConfigFile so LoadConfig has to read it back from disk.
	ConfigFile = models.ConfigStruct{}

	if err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if ConfigFile.PoenskelistenPort != 8080 {
		t.Errorf("PoenskelistenPort = %d, want 8080", ConfigFile.PoenskelistenPort)
	}
	_ = path
}
