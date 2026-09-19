package config

import (
	"aunefyren/poenskelisten/models"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	poenskelistenVersionParameter = "{{RELEASE_TAG}}"
	configFilePath, _             = filepath.Abs("./files/config.json")
	ConfigFile                    = models.ConfigStruct{}
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

	if ConfigFile.PoenskelistenName == "" {
		// Set new value
		ConfigFile.PoenskelistenName = "Pønskelisten"
		anythingChanged = true
	}

	if ConfigFile.PoenskelistenDescription == "" {
		// Set new value
		ConfigFile.PoenskelistenDescription = "Share wishlists in a meaningful way."
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

	if ConfigFile.PoenskelistenVersion == "" || ConfigFile.PoenskelistenVersion != poenskelistenVersionParameter {
		// Set new value
		ConfigFile.PoenskelistenVersion = poenskelistenVersionParameter
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

// Creates empty config.json
func CreateConfigFile() error {
	ConfigFile = models.ConfigStruct{}
	ConfigFile.PoenskelistenPort = 8080
	ConfigFile.PoenskelistenName = "Pønskelisten"
	ConfigFile.DBPort = 3306
	ConfigFile.DBType = "sqlite"
	ConfigFile.DBLocation = "files/data.db"
	ConfigFile.SMTPEnabled = false
	ConfigFile.PoenskelistenVersion = poenskelistenVersionParameter
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
func SaveConfig() error {
	file, err := json.MarshalIndent(ConfigFile, "", "	")
	if err != nil {
		fmt.Println("failed to marshal config file. error: " + err.Error())
		return errors.New("failed to marshal config file")
	}

	err = os.WriteFile(configFilePath, file, 0644)
	if err != nil {
		fmt.Println("failed to save config file to disk. error: " + err.Error())
		return errors.New("failed to save config file to disk")
	}

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
