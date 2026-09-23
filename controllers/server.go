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
		// AllowedOrigins always starts with the issuer.
		PoenskelistenAdditionalURLs: config.AllowedOrigins()[1:],
		OAuthIssuer:                 config.OAuthIssuer(),
		PoenskelistenPort:           config.ConfigFile.PoenskelistenPort,
		Timezone:                    config.ConfigFile.Timezone,
		PoenskelistenLogLevel:       config.ConfigFile.PoenskelistenLogLevel,
		PoenskelistenTestEmail:      config.ConfigFile.PoenskelistenTestEmail,

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
		OIDCProviderName:    config.OIDCDisplayName(),
		OIDCIssuerURL:       config.ConfigFile.OIDCIssuerURL,
		OIDCClientID:        config.ConfigFile.OIDCClientID,
		OIDCRedirectURL:     config.OIDCCallbackURL(),
		OIDCAutoCreateUsers: config.ConfigFile.OIDCAutoCreateUsers,
		LocalLoginEnabled:   config.LocalLoginEnabled(),

		// Security
		MFAEnforced:             config.ConfigFile.MFAEnforced,
		MFARecoveryCodesEnabled: config.ConfigFile.MFARecoveryCodesEnabled,

		// AI assistants
		MCPEnabled:  config.ConfigFile.MCPEnabled,
		MCPEndpoint: config.MCPResource(),
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Server info retrieved.", "server": serverInfo})

}
