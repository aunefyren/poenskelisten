package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func APIGetCurrency(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"message": "Currency retrieved.", "currency": config.ConfigFile.PoenskelistenCurrency, "padding": config.ConfigFile.PoenskelistenCurrencyPad, "left": config.ConfigFile.PoenskelistenCurrencyLeft})
}

func APIUpdateCurrency(context *gin.Context) {
	var currency models.UpdateCurrencyRequest

	if err := context.ShouldBindJSON(&currency); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Validate currency format
	stringMatch, requirements, err := utilities.ValidateTextCharacters(currency.PoenskelistenCurrency)
	if err != nil {
		logger.Log.Error("Failed to validate currency text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("Currency string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	config.ConfigFile.PoenskelistenCurrency = strings.TrimSpace(currency.PoenskelistenCurrency)
	config.ConfigFile.PoenskelistenCurrencyPad = currency.PoenskelistenCurrencyPad
	config.ConfigFile.PoenskelistenCurrencyLeft = currency.PoenskelistenCurrencyLeft

	err = config.SaveConfig()
	if err != nil {
		logger.Log.Error("Failed to save config file. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config file."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Currency updated.", "currency": config.ConfigFile.PoenskelistenCurrency, "padding": config.ConfigFile.PoenskelistenCurrencyPad, "left": config.ConfigFile.PoenskelistenCurrencyLeft})
}
