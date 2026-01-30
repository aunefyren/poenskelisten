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

	err := context.ShouldBindJSON(&currency)
	if err != nil {
		logger.Log.Error("failed to parse request. error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		context.Abort()
		return
	}

	// Validate currency format
	stringMatch, requirements, err := utilities.ValidateTextCharacters(currency.PoenskelistenCurrency)
	if err != nil {
		logger.Log.Error("failed to validate currency text string. error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate text string"})
		context.Abort()
		return
	} else if !stringMatch {
		logger.Log.Error("currency string failed validation")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	config.ConfigFile.PoenskelistenCurrency = strings.TrimSpace(currency.PoenskelistenCurrency)
	config.ConfigFile.PoenskelistenCurrencyPad = currency.PoenskelistenCurrencyPad
	config.ConfigFile.PoenskelistenCurrencyLeft = currency.PoenskelistenCurrencyLeft

	err = config.SaveConfig()
	if err != nil {
		logger.Log.Error("failed to save config file. error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save config file"})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "currency updated", "currency": config.ConfigFile.PoenskelistenCurrency, "padding": config.ConfigFile.PoenskelistenCurrencyPad, "left": config.ConfigFile.PoenskelistenCurrencyLeft})
}
