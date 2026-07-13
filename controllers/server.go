package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func APIGetServerInfo(context *gin.Context) {
	serverInfo := models.ServerInfoReply{
		// Application
		AppName:                  config.ConfigFile.PoenskelistenName,
		PoenskelistenVersion:     config.ConfigFile.PoenskelistenVersion,
		PoenskelistenEnvironment: config.ConfigFile.PoenskelistenEnvironment,
		PoenskelistenExternalURL: config.ConfigFile.PoenskelistenExternalURL,
		PoenskelistenPort:        config.ConfigFile.PoenskelistenPort,
		Timezone:                 config.ConfigFile.Timezone,
		PoenskelistenLogLevel:    config.ConfigFile.PoenskelistenLogLevel,
		PoenskelistenTestEmail:   config.ConfigFile.PoenskelistenTestEmail,

		// Database (credentials intentionally omitted)
		DatabaseType:     config.ConfigFile.DBType,
		DatabaseName:     config.ConfigFile.DBName,
		DatabaseHost:     config.ConfigFile.DBIP,
		DatabasePort:     config.ConfigFile.DBPort,
		DatabaseSSL:      config.ConfigFile.DBSSL,
		DatabaseLocation: config.ConfigFile.DBLocation,

		// Email (password intentionally omitted)
		SMTPEnabled: config.ConfigFile.SMTPEnabled,
		SMTPHost:    config.ConfigFile.SMTPHost,
		SMTPPort:    config.ConfigFile.SMTPPort,
		SMTPFrom:    config.ConfigFile.SMTPFrom,

		// Single sign-on (client secret intentionally omitted)
		OIDCEnabled:         config.ConfigFile.OIDCEnabled,
		OIDCProviderName:    config.ConfigFile.OIDCProviderName,
		OIDCIssuerURL:       config.ConfigFile.OIDCIssuerURL,
		OIDCClientID:        config.ConfigFile.OIDCClientID,
		OIDCRedirectURL:     config.ConfigFile.OIDCRedirectURL,
		OIDCAutoCreateUsers: config.ConfigFile.OIDCAutoCreateUsers,

		// Security
		MFAEnforced:             config.ConfigFile.MFAEnforced,
		MFARecoveryCodesEnabled: config.ConfigFile.MFARecoveryCodesEnabled,
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Server info retrieved.", "server": serverInfo})

}
