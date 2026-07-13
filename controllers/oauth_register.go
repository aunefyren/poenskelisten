package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/oauth"
	"aunefyren/poenskelisten/utilities"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// APIOAuthRegister implements RFC 7591 dynamic client registration. Registration
// is open (rate-limited at the route): it always creates a **public PKCE** client
// (no secret), which still requires user consent at /oauth/authorize.
func APIOAuthRegister(ctx *gin.Context) {
	var request models.OAuthRegisterRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		registrationError(ctx, "invalid_client_metadata", "Failed to parse registration request.")
		return
	}

	// redirect_uris: required, at least one, each an absolute http(s) URL.
	if len(request.RedirectURIs) == 0 {
		registrationError(ctx, "invalid_redirect_uri", "At least one redirect_uri is required.")
		return
	}
	for _, uri := range request.RedirectURIs {
		if !isValidRedirectURI(uri) {
			registrationError(ctx, "invalid_redirect_uri", "redirect_uri must be an absolute http(s) URL.")
			return
		}
	}

	// grant_types: default to the supported set; reject anything else.
	grantTypes := request.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	for _, g := range grantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			registrationError(ctx, "invalid_client_metadata", "Unsupported grant_type: "+g)
			return
		}
	}

	// scope: filter to known scopes; default to all scopes when omitted (the user
	// still consents to the actual scopes at authorization time).
	scopes := oauth.FilterValid(oauth.Parse(request.Scope))
	if strings.TrimSpace(request.Scope) == "" {
		scopes = oauth.AllNames()
	}

	clientID, err := utilities.GenerateOpaqueToken()
	if err != nil {
		logger.Log.Error("Failed to generate client_id. Error: " + err.Error())
		registrationError(ctx, "server_error", "Failed to register client.")
		return
	}

	enabled := true
	client := models.OAuthClient{
		ClientID:                clientID,
		ClientName:              strings.TrimSpace(request.ClientName),
		RedirectURIs:            request.RedirectURIs,
		Scopes:                  scopes,
		GrantTypes:              grantTypes,
		TokenEndpointAuthMethod: models.TokenEndpointAuthNone,
		IsPublic:                true,
		IsFirstParty:            false,
		Registered:              true,
		Enabled:                 &enabled,
	}

	created, err := database.CreateOAuthClient(client)
	if err != nil {
		logger.Log.Error("Failed to create OAuth client. Error: " + err.Error())
		registrationError(ctx, "server_error", "Failed to register client.")
		return
	}

	// RFC 7591 registration response.
	ctx.JSON(http.StatusCreated, gin.H{
		"client_id":                  created.ClientID,
		"client_id_issued_at":        time.Now().Unix(),
		"redirect_uris":              created.RedirectURIs,
		"grant_types":                created.GrantTypes,
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": created.TokenEndpointAuthMethod,
		"scope":                      strings.Join(created.Scopes, " "),
		"client_name":                created.ClientName,
	})
}

func isValidRedirectURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

func registrationError(ctx *gin.Context, errCode string, desc string) {
	ctx.JSON(http.StatusBadRequest, gin.H{"error": errCode, "error_description": desc})
}

// APIAdminListOAuthClients lists all registered OAuth clients (secret stripped).
func APIAdminListOAuthClients(ctx *gin.Context) {
	clients, err := database.GetAllOAuthClients()
	if err != nil {
		logger.Log.Error("Failed to list OAuth clients. Error: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list clients."})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"clients": clients})
}

// APIAdminRevokeOAuthClient disables a registered client. The built-in first-party
// client cannot be revoked.
func APIAdminRevokeOAuthClient(ctx *gin.Context) {
	clientID := ctx.Param("client_id")
	if err := database.DisableOAuthClient(clientID); err != nil {
		logger.Log.Error("Failed to revoke OAuth client. Error: " + err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Client revoked."})
}
