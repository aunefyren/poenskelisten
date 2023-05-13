package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIGetCurrency(context *gin.Context) {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config file."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Currency retrieved.", "currency": config.PoenskelistenCurrency})

}

func APIUpdateCurrency(context *gin.Context) {

	var currency models.UpdateCurrencyrequest

	if err := context.ShouldBindJSON(&currency); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Validate currency format
	stringMatch, requirements, err := utilities.ValidateTextCharacters(currency.PoenskelistenCurrency)
	if err != nil {
		log.Println("Failed to validate currency text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("Currencystring failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Get configuration
	configFile, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config file."})
		context.Abort()
		return
	}

	configFile.PoenskelistenCurrency = currency.PoenskelistenCurrency

	err = config.SaveConfig(configFile)
	if err != nil {
		log.Println("Failed to save config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config file."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Currency updated.", "currency": configFile.PoenskelistenCurrency})

}
