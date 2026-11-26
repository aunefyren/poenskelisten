package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/oauth"
	"aunefyren/poenskelisten/utilities"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// APIOAuthRegister implements RFC 7591 dynamic client registration. Registration
// is open (rate-limited at the route): it always creates a **public PKCE** client
// (no secret), which still requires user consent at /oauth/authorize.
func APIOAuthRegister(ctx *gin.Context) {
	// Third-party clients can only ever get MCP tokens (see resolveResource), so
	// with MCP off there is nothing for a registered client to do.
	if !config.ConfigFile.MCPEnabled {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "MCP is not enabled."})
		return
	}

	var request models.OAuthRegisterRequest
	// Bound directly rather than through bindJSON because RFC 7591 dictates
	// this endpoint's error format.
	if err := ctx.ShouldBindJSON(&request); err != nil {
		registrationError(ctx, "invalid_client_metadata", "Failed to parse registration request.")
		return
	}
	// client_name is chosen by an unauthenticated third party and shown to
	// users on the consent screen, so it gets the "text" rule.
	if err := utilities.SanitizeRequest(&request); err != nil {
		registrationError(ctx, "invalid_client_metadata", "Invalid client metadata: "+err.Error()+".")
		return
	}

	// redirect_uris: required, at least one and at most a handful, each an
	// absolute http(s) URL of bounded length.
	if len(request.RedirectURIs) == 0 {
		registrationError(ctx, "invalid_redirect_uri", "At least one redirect_uri is required.")
		return
	}
	if len(request.RedirectURIs) > maxRegisteredRedirectURIs {
		registrationError(ctx, "invalid_redirect_uri", "At most "+strconv.Itoa(maxRegisteredRedirectURIs)+" redirect_uris are allowed.")
		return
	}
	for _, uri := range request.RedirectURIs {
		if !isValidRedirectURI(uri) {
			registrationError(ctx, "invalid_redirect_uri", "redirect_uri must be an absolute http(s) URL without a fragment or credentials, at most "+strconv.Itoa(maxRedirectURILength)+" characters long.")
			return
		}
	}

	// grant_types: default to the supported set; reject anything else, and
	// store each one once however often it was listed.
	grantTypes := []string{}
	for _, g := range request.GrantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			registrationError(ctx, "invalid_client_metadata", "Unsupported grant_type: "+g)
			return
		}
		if !slices.Contains(grantTypes, g) {
			grantTypes = append(grantTypes, g)
		}
	}
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}

	// scope: filter to known scopes; default to all scopes when omitted (the user
	// still consents to the actual scopes at authorization time).
	scopes := oauth.FilterValid(oauth.Parse(request.Scope))
	if request.Scope == "" {
		scopes = oauth.AllNames()
	}

	// client_name is shown on the consent page and in the admin's connected
	// apps, so it is kept to a short, single line. Besides control characters
	// that rules out line and paragraph separators, and invisible format
	// characters such as bidi overrides, which can make a name read as
	// something it isn't.
	if utf8.RuneCountInString(request.ClientName) > maxClientNameLength || strings.ContainsFunc(request.ClientName, isDisallowedClientNameRune) {
		registrationError(ctx, "invalid_client_metadata", "client_name must be a single line of at most "+strconv.Itoa(maxClientNameLength)+" characters.")
		return
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
		ClientName:              request.ClientName,
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

// Bounds on what a dynamically registered client may store. Registration is
// open, so without them one request could store a megabyte of metadata, and
// show it on consent pages and in the admin's connected apps.
const (
	maxRegisteredRedirectURIs = 10
	maxRedirectURILength      = 2000
	maxClientNameLength       = 100
)

// isDisallowedClientNameRune reports whether r may not appear in a
// client_name: control characters, line and paragraph separators, and
// invisible format characters (bidi overrides, zero-width spaces and joiners).
func isDisallowedClientNameRune(r rune) bool {
	return unicode.IsControl(r) || unicode.In(r, unicode.Zl, unicode.Zp, unicode.Cf)
}

// isValidRedirectURI accepts an absolute http(s) URL of bounded length. A
// fragment is refused (RFC 6749 §3.1.2: the code is appended as a query, and
// a fragment would swallow it), and so are credentials in the URL, which have
// no business there and can disguise the real host on the consent page.
func isValidRedirectURI(uri string) bool {
	if len(uri) > maxRedirectURILength {
		return false
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Fragment != "" || strings.Contains(uri, "#") || parsed.User != nil {
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
