package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/oauth"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type connectedScope struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type connectedApp struct {
	ClientID   string           `json:"client_id"`
	ClientName string           `json:"client_name"`
	Scopes     []connectedScope `json:"scopes"`
	GrantedAt  time.Time        `json:"granted_at"`
}

// APIListConnectedApps returns the third-party apps the authenticated user has
// authorized (their consents), with human-readable scope descriptions. The
// built-in web client auto-consents and so never appears here.
func APIListConnectedApps(ctx *gin.Context) {
	userID, err := middlewares.GetAuthUsername(ctx.GetHeader("Authorization"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session."})
		ctx.Abort()
		return
	}

	consents, err := database.GetUserConsents(userID)
	if err != nil {
		logger.Log.Error("Failed to list connected apps. Error: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list connected apps."})
		ctx.Abort()
		return
	}

	apps := make([]connectedApp, 0, len(consents))
	for _, consent := range consents {
		name := consent.ClientID
		if client, found, _ := database.GetOAuthClient(consent.ClientID); found && client.ClientName != "" {
			name = client.ClientName
		}

		scopes := make([]connectedScope, 0, len(consent.Scopes))
		for _, s := range consent.Scopes {
			desc := s
			if scope, ok := oauth.Lookup(s); ok {
				desc = scope.Description
			}
			scopes = append(scopes, connectedScope{Name: s, Description: desc})
		}

		apps = append(apps, connectedApp{
			ClientID:   consent.ClientID,
			ClientName: name,
			Scopes:     scopes,
			GrantedAt:  consent.UpdatedAt,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{"apps": apps})
}

// APIRevokeConnectedApp disconnects one app for the authenticated user: it removes
// the consent and revokes that app's refresh sessions for the user, so it can no
// longer act on their behalf.
func APIRevokeConnectedApp(ctx *gin.Context) {
	userID, err := middlewares.GetAuthUsername(ctx.GetHeader("Authorization"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session."})
		ctx.Abort()
		return
	}

	clientID := ctx.Param("client_id")

	if err := database.RevokeConsent(userID, clientID); err != nil {
		logger.Log.Error("Failed to revoke consent. Error: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect app."})
		ctx.Abort()
		return
	}
	if err := database.RevokeUserClientSessions(userID, clientID); err != nil {
		logger.Log.Error("Failed to revoke app sessions. Error: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect app."})
		ctx.Abort()
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "App disconnected."})
}
