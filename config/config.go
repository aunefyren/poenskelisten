package config

import (
	"aunefyren/poenskelisten/logger"
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
		fmt.Println("Config file does not exist. Creating...")

		err := CreateConfigFile()
		if err != nil {
			return err
		}
	}

	file, err := os.Open(configFilePath)
	if err != nil {
		fmt.Println("Get config file threw error trying to open the file.")
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ConfigFile)
	if err != nil {
		fmt.Println("Get config file threw error trying to parse the file.")
		return err
	}

	anythingChanged := false

	if ConfigFile.PrivateKey == "" {
		// Set new value
		newKey, err := GenerateSecureKey(64)
		if err != nil {
			return errors.New("Failed to generate secure key. Error: " + err.Error())
		}
		ConfigFile.PrivateKey = newKey
		anythingChanged = true
		fmt.Println("New private key set.")
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
		return errors.New("Pønskelisten environment is set to 'test', but no test e-mail is configured.")
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

	if ConfigFile.PoenskelistenLogLevel == "" {
		level := logrus.InfoLevel
		ConfigFile.PoenskelistenLogLevel = level.String()
		anythingChanged = true
	} else {
		parsedLogLevel, err := logrus.ParseLevel(ConfigFile.PoenskelistenLogLevel)
		if err != nil {
			fmt.Println("Failed to load log level: %v", err)
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
	ConfigFile.DBType = "mysql"
	ConfigFile.SMTPEnabled = false
	ConfigFile.PoenskelistenVersion = poenskelistenVersionParameter
	ConfigFile.PoenskelistenCurrencyLeft = true

	privateKey, err := GenerateSecureKey(64)
	if err != nil {
		logger.Log.Error("Failed to generate private key. Error: " + err.Error())
		fmt.Println("Failed to generate private key. Error: " + err.Error())
		return err
	}
	ConfigFile.PrivateKey = privateKey

	err = SaveConfig()
	if err != nil {
		logger.Log.Error("Create config file threw error trying to save the file.")
		fmt.Println("Create config file threw error trying to save the file.")
		return err
	}

	return nil
}

// Saves the given config struct as config.json
func SaveConfig() error {
	file, err := json.MarshalIndent(ConfigFile, "", "	")
	if err != nil {
		return err
	}

	err = os.WriteFile(configFilePath, file, 0644)
	if err != nil {
		return err
	}

	return nil
}

func GetPrivateKey(epoch int) []byte {
	if epoch > 5 {
		fmt.Println("Failed to load private key. Exiting...")
		os.Exit(1)
	}

	secretKey, err := base64.StdEncoding.DecodeString(ConfigFile.PrivateKey)
	if err != nil {
		ResetSecureKey()
		return GetPrivateKey(epoch + 1)
	}

	return secretKey
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

func ResetSecureKey() {
	privateKey, err := GenerateSecureKey(64)
	if err != nil {
		fmt.Println("Failed to generate new secret key. Exiting...")
		os.Exit(1)
	}
	ConfigFile.PrivateKey = privateKey
	err = SaveConfig()
	if err != nil {
		fmt.Println("Failed to save new config. Exiting...")
		os.Exit(1)
	}
	logger.Log.Info("New private key set.")
}
