package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIGetCurrency(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"message": "Currency retrieved.", "currency": config.ConfigFile.PoenskelistenCurrency, "padding": config.ConfigFile.PoenskelistenCurrencyPad, "left": config.ConfigFile.PoenskelistenCurrencyLeft})
}

func APIUpdateCurrency(context *gin.Context) {
	var currency models.UpdateCurrencyRequest

	if !bindJSON(context, &currency) {
		return
	}

	config.ConfigFile.PoenskelistenCurrency = currency.PoenskelistenCurrency
	config.ConfigFile.PoenskelistenCurrencyPad = currency.PoenskelistenCurrencyPad
	config.ConfigFile.PoenskelistenCurrencyLeft = currency.PoenskelistenCurrencyLeft

	err := config.SaveConfig()
	if err != nil {
		logger.Log.Error("failed to save config file. error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save config file"})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "currency updated", "currency": config.ConfigFile.PoenskelistenCurrency, "padding": config.ConfigFile.PoenskelistenCurrencyPad, "left": config.ConfigFile.PoenskelistenCurrencyLeft})
}
