package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/oauth"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIOAuthAuthorizationServerMetadata serves RFC 8414 authorization-server
// metadata so OAuth clients (and MCP clients) can discover the endpoints. The
// authorize/token endpoints are advertised here; they are implemented in a later
// sub-phase. Served only when the OAuth server is enabled.
func APIOAuthAuthorizationServerMetadata(ctx *gin.Context) {
	issuer := config.OAuthIssuer()
	ctx.JSON(http.StatusOK, gin.H{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oauth/authorize",
		"token_endpoint":                        issuer + "/oauth/token",
		"registration_endpoint":                 issuer + "/oauth/register",
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"scopes_supported":                      oauth.AllNames(),
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
	})
}

// APIOAuthProtectedResourceMetadata serves RFC 9728 protected-resource metadata,
// pointing MCP clients at the authorization server that guards this resource.
// Served only when the MCP resource server is enabled.
func APIOAuthProtectedResourceMetadata(ctx *gin.Context) {
	if !config.ConfigFile.MCPEnabled {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "MCP is not enabled."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"resource":                 config.MCPResource(),
		"authorization_servers":    []string{config.OAuthIssuer()},
		"scopes_supported":         oauth.MCPNames(),
		"bearer_methods_supported": []string{"header"},
	})
}

// APIOAuthJWKS publishes the public keys used to verify OAuth tokens.
func APIOAuthJWKS(ctx *gin.Context) {
	jwks, err := auth.OAuthPublicJWKS()
	if err != nil {
		logger.Log.Error("Failed to build JWKS. Error: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build JWKS."})
		return
	}

	ctx.JSON(http.StatusOK, jwks)
}
