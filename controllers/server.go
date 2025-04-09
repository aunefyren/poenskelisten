package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIGetServerInfo(context *gin.Context) {

	config, err := config.GetConfig()
	if err != nil {
		logger.Log.Error("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config."})
		context.Abort()
		return
	}

	serverInfo := models.ServerInfoReply{
		Timezone:                 config.Timezone,
		PoenskelistenVersion:     config.PoenskelistenVersion,
		PoenskelistenPort:        config.PoenskelistenPort,
		PoenskelistenExternalURL: config.PoenskelistenExternalURL,
		DatabaseType:             config.DBType,
		SMTPEnabled:              config.SMTPEnabled,
		PoenskelistenEnvironment: config.PoenskelistenEnvironment,
		PoenskelistenTestEmail:   config.PoenskelistenTestEmail,
		PoenskelistenLogLevel:    config.PoenskelistenLogLevel,
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Server info retrieved.", "server": serverInfo})

}
