package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIGetServerInfo(context *gin.Context) {
	serverInfo := models.ServerInfoReply{
		Timezone:                 config.ConfigFile.Timezone,
		PoenskelistenVersion:     config.ConfigFile.PoenskelistenVersion,
		PoenskelistenPort:        config.ConfigFile.PoenskelistenPort,
		PoenskelistenExternalURL: config.ConfigFile.PoenskelistenExternalURL,
		DatabaseType:             config.ConfigFile.DBType,
		SMTPEnabled:              config.ConfigFile.SMTPEnabled,
		PoenskelistenEnvironment: config.ConfigFile.PoenskelistenEnvironment,
		PoenskelistenTestEmail:   config.ConfigFile.PoenskelistenTestEmail,
		PoenskelistenLogLevel:    config.ConfigFile.PoenskelistenLogLevel,
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Server info retrieved.", "server": serverInfo})

}
