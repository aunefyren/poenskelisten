package config

import (
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var wrapperr_version_parameter = "v0.0.1"
var config_path, _ = filepath.Abs("./files/config.json")

func GetConfig() (*models.ConfigStruct, error) {
	// Create config.json if it doesn't exist
	if _, err := os.Stat(config_path); errors.Is(err, os.ErrNotExist) {
		log.Println("Config file does not exist. Creating...")
		fmt.Println("Config file does not exist. Creating...")

		err := CreateConfigFile()
		if err != nil {
			return nil, err
		}
	}

	file, err := os.Open(config_path)
	if err != nil {
		log.Println("Get config file threw error trying to open the file.")
		fmt.Println("Get config file threw error trying to open the file.")
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	config := models.ConfigStruct{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Println("Get config file threw error trying to parse the file.")
		fmt.Println("Get config file threw error trying to parse the file.")
		return nil, err
	}

	anythingChanged := false

	if config.PoenskelistenName == "" {
		// Set new value
		config.PoenskelistenName = "Pønskelisten"
		anythingChanged = true
	}

	if config.PoenskelistenPort == 0 {
		// Set new value
		config.PoenskelistenPort = 8080
		anythingChanged = true
	}

	if config.DBPort == 0 {
		// Set new value
		config.DBPort = 3306
		anythingChanged = true
	}

	if anythingChanged {
		// Save new version of config json
		err = SaveConfig(&config)
		if err != nil {
			return nil, err
		}
	}

	// Return config object
	return &config, nil

}

// Creates empty config.json
func CreateConfigFile() error {

	var config models.ConfigStruct

	config.PoenskelistenPort = 8080
	config.PoenskelistenName = "Pønskelisten"
	config.DBPort = 3306
	config.SMTPEnabled = true

	err := SaveConfig(&config)
	if err != nil {
		log.Println("Create config file threw error trying to save the file.")
		fmt.Println("Create config file threw error trying to save the file.")
		return err
	}

	return nil

}

// Saves the given config struct as config.json
func SaveConfig(config *models.ConfigStruct) error {

	file, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}

	err = os.WriteFile(config_path, file, 0644)
	if err != nil {
		return err
	}

	return nil
}
