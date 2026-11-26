package config

import (
	"aunefyren/poenskelisten/models"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	// Relative to the working directory at use, like the log file and the
	// default SQLite location, so a test that changes directory moves all three.
	configFilePath = filepath.Join("files", "config.json")
	ConfigFile     = models.ConfigStruct{}
)

func LoadConfig() (err error) {
	// Create config.json if it doesn't exist
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		fmt.Println("config file does not exist. creating...")

		err := CreateConfigFile()
		if err != nil {
			return err
		}
	}

	restrictConfigFilePermissions()

	file, err := os.Open(configFilePath)
	if err != nil {
		fmt.Println("load config file threw error trying to open the file")
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ConfigFile)
	if err != nil {
		fmt.Println("load config file threw error trying to parse the file")
		return err
	}

	anythingChanged := false

	if ConfigFile.PrivateKey == "" {
		// Set new value
		newKey, err := GenerateSecureKey(64)
		if err != nil {
			return errors.New("failed to generate secure key. error: " + err.Error())
		}
		ConfigFile.PrivateKey = newKey
		anythingChanged = true
		fmt.Println("new private key set.")
	}

	if CleanAppText(&ConfigFile) {
		anythingChanged = true
	}

	if ConfigFile.PoenskelistenEnvironment == "" {
		// Set new value
		ConfigFile.PoenskelistenEnvironment = "production"
		anythingChanged = true
	} else if ConfigFile.PoenskelistenEnvironment == "test" && ConfigFile.PoenskelistenTestEmail == "" {
		return errors.New("Pønskelisten environment is set to 'test', but no test e-mail is configured")
	}

	if ConfigFile.Timezone == "" {
		// Set new value
		ConfigFile.Timezone = "Europe/Paris"
		anythingChanged = true
	}

	if ConfigFile.PoenskelistenPort == 0 {
		// Set new value
		ConfigFile.PoenskelistenPort = 8080
		anythingChanged = true
	}

	if ConfigFile.DBPort == 0 {
		// Set new value
		ConfigFile.DBPort = 3306
		anythingChanged = true
	}

	if resolved := resolveVersion(); ConfigFile.PoenskelistenVersion != resolved {
		ConfigFile.PoenskelistenVersion = resolved
		anythingChanged = true
	}

	if ConfigFile.PoenskelistenCurrency == "" {
		// Set new value
		ConfigFile.PoenskelistenCurrency = "$"
		anythingChanged = true
	}

	if ConfigFile.DBType == "" || (strings.ToLower(ConfigFile.DBType) != "mysql" && strings.ToLower(ConfigFile.DBType) != "postgres" && strings.ToLower(ConfigFile.DBType) != "sqlite") {
		// Set new value
		ConfigFile.DBType = "mysql"
		anythingChanged = true
	}

	if (strings.ToLower(ConfigFile.DBType) == "sqlite") && ConfigFile.DBLocation == "" {
		// Set new value
		ConfigFile.DBLocation = "files/data.db"
		anythingChanged = true
	}

	// OIDC defaults are computed at runtime (see OIDCDisplayName/OIDCCallbackURL)
	// rather than persisted here: LoadConfig runs before flags/env vars are
	// applied, so a default derived now would miss an oidcenabled/externalurl set
	// that way. Older versions did persist them; clear values identical to the
	// default so they follow later changes to the external URL.
	if ConfigFile.OIDCProviderName == defaultOIDCProviderName {
		ConfigFile.OIDCProviderName = ""
		anythingChanged = true
	}
	if ConfigFile.OIDCRedirectURL != "" && ConfigFile.OIDCRedirectURL == derivedOIDCCallbackURL() {
		ConfigFile.OIDCRedirectURL = ""
		anythingChanged = true
	}

	// The OAuth signing key is generated once and persisted: it must survive
	// restarts, otherwise all issued tokens and the published JWKS would break. The
	// issuer, algorithm, and resource identifiers are not persisted — they are
	// computed from config at runtime (see OAuthIssuer / APIResource / MCPResource).
	if ConfigFile.OAuthSigningKey == "" {
		keyPEM, keyID, err := GenerateOAuthSigningKey(OAuthSigningAlgorithm)
		if err != nil {
			return errors.New("failed to generate OAuth signing key. error: " + err.Error())
		}
		ConfigFile.OAuthSigningKey = keyPEM
		ConfigFile.OAuthSigningKeyID = keyID
		anythingChanged = true
		fmt.Println("new OAuth signing key generated.")
	}

	if ConfigFile.PoenskelistenLogLevel == "" {
		level := logrus.InfoLevel
		ConfigFile.PoenskelistenLogLevel = level.String()
		anythingChanged = true
	} else {
		parsedLogLevel, err := logrus.ParseLevel(ConfigFile.PoenskelistenLogLevel)
		if err != nil {
			fmt.Println("failed to load log level: " + err.Error())
			level := logrus.InfoLevel
			ConfigFile.PoenskelistenLogLevel = level.String()
			anythingChanged = true
		} else {
			logrus.SetLevel(parsedLogLevel)
		}
	}

	if anythingChanged {
		// Save new version of config json
		err = SaveConfig()
		if err != nil {
			return err
		}
	}

	// Return nil
	return nil
}

const (
	defaultAppName        = "Pønskelisten"
	defaultAppDescription = "Share wishlists in a meaningful way."
)

// CleanAppText trims the app name and description and falls back to the
// defaults when nothing but whitespace is left. It runs on load and again after
// flags/env vars are applied, since those can set either one to blank. Reports
// whether anything changed.
func CleanAppText(configFile *models.ConfigStruct) bool {
	changed := false

	name := strings.TrimSpace(configFile.PoenskelistenName)
	if name == "" {
		name = defaultAppName
	}
	if name != configFile.PoenskelistenName {
		configFile.PoenskelistenName = name
		changed = true
	}

	description := strings.TrimSpace(configFile.PoenskelistenDescription)
	if description == "" {
		description = defaultAppDescription
	}
	if description != configFile.PoenskelistenDescription {
		configFile.PoenskelistenDescription = description
		changed = true
	}

	return changed
}

// Creates empty config.json
func CreateConfigFile() error {
	ConfigFile = models.ConfigStruct{}
	ConfigFile.PoenskelistenPort = 8080
	ConfigFile.PoenskelistenName = defaultAppName
	ConfigFile.DBPort = 3306
	ConfigFile.DBType = "sqlite"
	ConfigFile.DBLocation = "files/data.db"
	ConfigFile.SMTPEnabled = false
	ConfigFile.PoenskelistenVersion = resolveVersion()
	ConfigFile.PoenskelistenCurrencyLeft = true

	privateKey, err := GenerateSecureKey(64)
	if err != nil {
		fmt.Println("failed to generate private key. error: " + err.Error())
		return errors.New("failed to generate private key")
	}
	ConfigFile.PrivateKey = privateKey

	err = SaveConfig()
	if err != nil {
		fmt.Println("create config file threw error trying to save the file. error: " + err.Error())
		return errors.New("create config file threw error trying to save the file")
	}

	return nil
}

// Saves the given config struct as config.json
// configFileMode is owner read/write only. os.WriteFile applies it to new
// files only, so restrictConfigFilePermissions handles files written as 0644
// by versions before 2.5.0.
const configFileMode os.FileMode = 0600

// OwnerOnlyDirMode is for the files/ directory, which holds config.json, the
// log and, with SQLite, the database with every password hash.
const OwnerOnlyDirMode os.FileMode = 0700

// restrictConfigFilePermissions removes from an existing config file every
// permission bit configFileMode doesn't grant. It never adds any, so an
// operator's read-only file stays read-only. A failure is only reported: the
// file is still usable, and refusing to start over its permissions would be
// worse.
func restrictConfigFilePermissions() {
	changed, err := RestrictPermissions(configFilePath, configFileMode)
	if errors.Is(err, os.ErrNotExist) {
		return
	} else if err != nil {
		fmt.Println("failed to restrict config file permissions. error: " + err.Error())
	} else if changed {
		fmt.Println("config.json was readable by other users; restricted it to its owner.")
	}
}

// RestrictPermissions removes the permission bits of path that mode doesn't
// grant (it never adds any), and reports whether it had to change anything.
// On Windows it only checks that path exists: Go reports every directory there
// as 0777 and every writable file as 0666, and os.Chmod can only toggle the
// read-only flag, so it would "change" the same path on every call.
func RestrictPermissions(path string, mode os.FileMode) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if runtime.GOOS == "windows" {
		return false, nil
	}
	current := info.Mode().Perm()
	if current&^mode == 0 {
		return false, nil
	}
	if err := os.Chmod(path, current&mode); err != nil {
		return false, err
	}
	return true, nil
}

func SaveConfig() error {
	file, err := json.MarshalIndent(ConfigFile, "", "	")
	if err != nil {
		fmt.Println("failed to marshal config file. error: " + err.Error())
		return errors.New("failed to marshal config file")
	}

	// 0600: the file holds the token signing keys and any DB/SMTP/OIDC secrets.
	err = os.WriteFile(configFilePath, file, configFileMode)
	if err != nil {
		fmt.Println("failed to save config file to disk. error: " + err.Error())
		return errors.New("failed to save config file to disk")
	}

	// os.WriteFile only applies the mode when it creates the file.
	restrictConfigFilePermissions()

	return nil
}

// GetPrivateKey returns the decoded JWT signing secret. It fails loudly rather
// than silently regenerating the key on error: rotating the secret would
// invalidate every live session, and an empty secret would sign tokens
// insecurely without anyone noticing.
func GetPrivateKey() ([]byte, error) {
	if ConfigFile.PrivateKey == "" {
		return nil, errors.New("private key is not configured")
	}

	secretKey, err := base64.StdEncoding.DecodeString(ConfigFile.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(secretKey) == 0 {
		return nil, errors.New("private key decoded to an empty value")
	}

	return secretKey, nil
}

// GenerateSecureKey creates a cryptographically secure random key of the given length (in bytes).
func GenerateSecureKey(length int) (string, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	// Encode to Base64 to make it easy to store
	return base64.StdEncoding.EncodeToString(key), nil
}

const (
	// defaultOIDCProviderName labels the SSO login button when no provider name is set.
	defaultOIDCProviderName = "Single sign-on"
	// oidcCallbackPath is where the IdP sends the browser back to (controllers.OIDCCallback).
	oidcCallbackPath = "/api/open/oidc/callback"
	// oidcLoginPath starts the flow (controllers.OIDCLogin).
	oidcLoginPath = "/api/open/oidc/login"
)

// OIDCDisplayName is the SSO login button label: the configured provider name,
// or a generic default.
func OIDCDisplayName() string {
	if strings.TrimSpace(ConfigFile.OIDCProviderName) != "" {
		return ConfigFile.OIDCProviderName
	}
	return defaultOIDCProviderName
}

// OIDCCallbackURL is the redirect URL registered with the IdP: the explicitly
// configured one, or else derived from the external URL. Like OAuthIssuer it is
// computed on every call rather than persisted, so it always reflects the
// final configuration after flags/env vars are applied.
func OIDCCallbackURL() string {
	if strings.TrimSpace(ConfigFile.OIDCRedirectURL) != "" {
		return ConfigFile.OIDCRedirectURL
	}
	return derivedOIDCCallbackURL()
}

func derivedOIDCCallbackURL() string {
	return OAuthIssuer() + oidcCallbackPath
}

// OIDCLoginURL is where the SSO button sends the browser: the login endpoint
// beside the callback, not on whatever origin the page was opened on. The login
// endpoint sets the state and nonce cookies and the callback reads them back,
// so starting from an additional URL would set them on an origin the callback
// never sees. A path prefix on an explicit redirect URL (a reverse proxy
// serving the app under a subpath) is kept.
func OIDCLoginURL() string {
	callback, err := url.Parse(OIDCCallbackURL())
	if err != nil || callback.Scheme == "" || callback.Host == "" {
		return oidcLoginPath
	}
	prefix := ""
	if strings.HasSuffix(callback.Path, oidcCallbackPath) {
		prefix = strings.TrimSuffix(callback.Path, oidcCallbackPath)
	}
	return callback.Scheme + "://" + callback.Host + prefix + oidcLoginPath
}

// LocalLoginEnabled reports whether password login, self-registration and
// password reset are available. Disabling them only takes effect while OIDC is
// enabled and configured, so a broken or missing SSO setup can never leave the
// instance with no way to log in at all.
func LocalLoginEnabled() bool {
	return !ConfigFile.LocalLoginDisabled || !OIDCConfigured()
}

// OIDCConfigured reports whether OIDC is enabled with the settings needed to
// start a login (the client secret is optional for public clients).
func OIDCConfigured() bool {
	return ConfigFile.OIDCEnabled &&
		strings.TrimSpace(ConfigFile.OIDCIssuerURL) != "" &&
		strings.TrimSpace(ConfigFile.OIDCClientID) != ""
}
