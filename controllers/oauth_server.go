package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/oauth"
	"aunefyren/poenskelisten/utilities"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const authorizationCodeTTL = 60 * time.Second

// resolveSSOUser validates the SSO cookie and returns the logged-in user. It
// enforces the global logout marker (SessionsInvalidatedAt).
func resolveSSOUser(ctx *gin.Context) (models.User, bool) {
	ssoToken, err := ctx.Cookie(ssoCookieName)
	if err != nil || ssoToken == "" {
		return models.User{}, false
	}
	claims, err := auth.ValidateSSOToken(ssoToken)
	if err != nil {
		return models.User{}, false
	}
	user, err := database.GetAllUserInformation(claims.UserID)
	if err != nil {
		return models.User{}, false
	}
	if user.SessionsInvalidatedAt != nil && claims.IssuedAt != nil && claims.IssuedAt.Time.Before(*user.SessionsInvalidatedAt) {
		return models.User{}, false
	}
	return user, true
}

// resolveGateUser identifies the user for endpoints used during the login gate
// (email verification, forced MFA enrollment). Such a user is authenticated at
// the SSO layer but may not yet hold an OAuth access token, so we accept either:
// the access token (fully logged-in, e.g. voluntary MFA setup from the account
// page) or, failing that, the SSO cookie (logged in but gated before tokens).
func resolveGateUser(ctx *gin.Context) (models.User, bool) {
	if userID, err := middlewares.GetAuthUsername(ctx.GetHeader("Authorization")); err == nil {
		if user, err := database.GetAllUserInformation(userID); err == nil {
			return user, true
		}
	}
	return resolveSSOUser(ctx)
}

// APIOAuthAuthorize is the authorization endpoint (RFC 6749 §4.1 + PKCE).
func APIOAuthAuthorize(ctx *gin.Context) {
	q := ctx.Request.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")
	scopeParam := q.Get("scope")
	state := q.Get("state")
	challenge := q.Get("code_challenge")
	challengeMethod := q.Get("code_challenge_method")
	resource := q.Get("resource")

	client, found, err := database.GetOAuthClient(clientID)
	if err != nil {
		logger.Log.Error("Failed to load OAuth client. Error: " + err.Error())
		ctx.String(http.StatusInternalServerError, "Authorization error.")
		return
	}
	// Before redirect_uri is validated we must not redirect anywhere.
	if !found || !client.IsEnabled() {
		ctx.String(http.StatusBadRequest, "Unknown or disabled client.")
		return
	}
	if !client.HasRedirectURI(redirectURI) {
		ctx.String(http.StatusBadRequest, "Invalid redirect URI.")
		return
	}

	// From here, protocol errors are reported by redirecting back to the client.
	if responseType != "code" {
		redirectAuthError(ctx, redirectURI, state, "unsupported_response_type")
		return
	}
	if challenge == "" || challengeMethod != "S256" {
		redirectAuthError(ctx, redirectURI, state, "invalid_request")
		return
	}

	requested := oauth.FilterValid(oauth.Parse(scopeParam))
	if len(requested) == 0 || !client.AllowsScopes(requested) {
		redirectAuthError(ctx, redirectURI, state, "invalid_scope")
		return
	}

	if resource == "" {
		resource = config.APIResource()
	}
	if resource != config.APIResource() && resource != config.MCPResource() {
		redirectAuthError(ctx, redirectURI, state, "invalid_target")
		return
	}

	// Authenticate the resource owner via the SSO session.
	user, loggedIn := resolveSSOUser(ctx)
	if !loggedIn {
		next := ctx.Request.URL.RequestURI()
		ctx.Redirect(http.StatusFound, "/login?next="+url.QueryEscape(next))
		return
	}

	// Login-time gates (moved here from the API middleware).
	if config.ConfigFile.SMTPEnabled && (user.Verified == nil || !*user.Verified) {
		ctx.Redirect(http.StatusFound, "/verify")
		return
	}
	if config.ConfigFile.MFAEnforced && user.IsLocalAuth() && !user.IsMFAEnabled() {
		ctx.Redirect(http.StatusFound, "/enroll")
		return
	}

	// Consent (auto-approved for the first-party client).
	if !client.IsFirstParty {
		consent, haveConsent, _ := database.GetConsent(user.ID, clientID)
		if !haveConsent || !consent.Covers(requested) {
			renderConsentPage(ctx, client, requested, redirectURI, state, challenge, resource)
			return
		}
	}

	issueAuthorizationCode(ctx, client, user, redirectURI, requested, resource, challenge, state)
}

// APIOAuthConsent handles the consent form submission for non-first-party clients.
func APIOAuthConsent(ctx *gin.Context) {
	user, loggedIn := resolveSSOUser(ctx)
	if !loggedIn {
		ctx.Redirect(http.StatusFound, "/login")
		return
	}

	clientID := ctx.PostForm("client_id")
	redirectURI := ctx.PostForm("redirect_uri")
	state := ctx.PostForm("state")
	challenge := ctx.PostForm("code_challenge")
	resource := ctx.PostForm("resource")
	requested := oauth.FilterValid(oauth.Parse(ctx.PostForm("scope")))

	client, found, err := database.GetOAuthClient(clientID)
	if err != nil || !found || !client.IsEnabled() || !client.HasRedirectURI(redirectURI) {
		ctx.String(http.StatusBadRequest, "Invalid consent request.")
		return
	}

	if ctx.PostForm("action") != "allow" {
		redirectAuthError(ctx, redirectURI, state, "access_denied")
		return
	}
	if len(requested) == 0 || !client.AllowsScopes(requested) {
		redirectAuthError(ctx, redirectURI, state, "invalid_scope")
		return
	}

	if err := database.UpsertConsent(user.ID, clientID, requested); err != nil {
		logger.Log.Error("Failed to store consent. Error: " + err.Error())
		ctx.String(http.StatusInternalServerError, "Failed to store consent.")
		return
	}

	issueAuthorizationCode(ctx, client, user, redirectURI, requested, resource, challenge, state)
}

// issueAuthorizationCode mints a single-use code and redirects back to the client.
func issueAuthorizationCode(ctx *gin.Context, client models.OAuthClient, user models.User, redirectURI string, scopes []string, resource string, challenge string, state string) {
	codePlain, err := utilities.GenerateOpaqueToken()
	if err != nil {
		logger.Log.Error("Failed to generate authorization code. Error: " + err.Error())
		redirectAuthError(ctx, redirectURI, state, "server_error")
		return
	}

	code := models.AuthorizationCode{
		CodeHash:            utilities.HashOpaqueToken(codePlain),
		ClientID:            client.ClientID,
		UserID:              user.ID,
		RedirectURI:         redirectURI,
		Scope:               strings.Join(scopes, " "),
		Resource:            resource,
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(authorizationCodeTTL),
	}
	if _, err := database.CreateAuthorizationCode(code); err != nil {
		logger.Log.Error("Failed to store authorization code. Error: " + err.Error())
		redirectAuthError(ctx, redirectURI, state, "server_error")
		return
	}

	target, err := url.Parse(redirectURI)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid redirect URI.")
		return
	}
	rq := target.Query()
	rq.Set("code", codePlain)
	if state != "" {
		rq.Set("state", state)
	}
	target.RawQuery = rq.Encode()
	ctx.Redirect(http.StatusFound, target.String())
}

// APIOAuthToken is the token endpoint.
func APIOAuthToken(ctx *gin.Context) {
	switch ctx.PostForm("grant_type") {
	case "authorization_code":
		handleAuthorizationCodeGrant(ctx)
	case "refresh_token":
		handleRefreshTokenGrant(ctx)
	default:
		tokenError(ctx, http.StatusBadRequest, "unsupported_grant_type", "")
	}
}

func handleAuthorizationCodeGrant(ctx *gin.Context) {
	clientID := ctx.PostForm("client_id")
	client, ok := authenticateClient(ctx, clientID)
	if !ok {
		return
	}

	code, err := database.ConsumeAuthorizationCode(utilities.HashOpaqueToken(ctx.PostForm("code")))
	if err != nil {
		tokenError(ctx, http.StatusBadRequest, "invalid_grant", "authorization code invalid, expired, or already used")
		return
	}
	if code.ClientID != client.ClientID || code.RedirectURI != ctx.PostForm("redirect_uri") {
		tokenError(ctx, http.StatusBadRequest, "invalid_grant", "code binding mismatch")
		return
	}
	if !auth.VerifyPKCE(ctx.PostForm("code_verifier"), code.CodeChallenge) {
		tokenError(ctx, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}

	user, err := database.GetAllUserInformation(code.UserID)
	if err != nil {
		tokenError(ctx, http.StatusBadRequest, "invalid_grant", "user not found")
		return
	}

	issueTokenResponse(ctx, client, user, code.Scope, code.Resource)
}

func handleRefreshTokenGrant(ctx *gin.Context) {
	clientID := ctx.PostForm("client_id")
	client, ok := authenticateClient(ctx, clientID)
	if !ok {
		return
	}

	oldPlain := ctx.PostForm("refresh_token")
	if client.IsFirstParty {
		if cookie, err := ctx.Cookie(refreshCookieName); err == nil {
			oldPlain = cookie
		}
	}
	if oldPlain == "" {
		tokenError(ctx, http.StatusBadRequest, "invalid_grant", "missing refresh token")
		return
	}

	newPlain, err := utilities.GenerateOpaqueToken()
	if err != nil {
		tokenError(ctx, http.StatusInternalServerError, "server_error", "")
		return
	}
	result, err := database.RotateSession(utilities.HashOpaqueToken(oldPlain), utilities.HashOpaqueToken(newPlain), ctx.GetHeader("User-Agent"), ctx.ClientIP())
	if err != nil {
		if client.IsFirstParty {
			clearRefreshCookie(ctx)
		}
		tokenError(ctx, http.StatusBadRequest, "invalid_grant", "refresh token invalid or expired")
		return
	}

	user, err := database.GetAllUserInformation(result.UserID)
	if err != nil {
		tokenError(ctx, http.StatusBadRequest, "invalid_grant", "user not found")
		return
	}

	accessToken, err := auth.GenerateOAuthAccessToken(user.ID, result.Resource, result.Scope, user.Admin, *user.Verified)
	if err != nil {
		tokenError(ctx, http.StatusInternalServerError, "server_error", "")
		return
	}

	resp := gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(auth.OAuthAccessTokenDuration.Seconds()),
		"scope":        result.Scope,
	}
	if scopeContains(result.Scope, "openid") {
		if idToken, err := auth.GenerateIDToken(user.ID, client.ClientID, derefString(user.Email), displayName(user)); err == nil {
			resp["id_token"] = idToken
		}
	}

	if client.IsFirstParty {
		if result.Rotated {
			setRefreshCookie(ctx, newPlain)
		}
	} else if result.Rotated {
		resp["refresh_token"] = newPlain
	} else {
		resp["refresh_token"] = oldPlain
	}

	ctx.JSON(http.StatusOK, resp)
}

// issueTokenResponse mints and delivers access + refresh (+ id) tokens.
func issueTokenResponse(ctx *gin.Context, client models.OAuthClient, user models.User, scope string, resource string) {
	refreshPlain, err := utilities.GenerateOpaqueToken()
	if err != nil {
		tokenError(ctx, http.StatusInternalServerError, "server_error", "")
		return
	}
	if _, err := database.CreateOAuthRefreshSession(user.ID, utilities.HashOpaqueToken(refreshPlain), client.ClientID, scope, resource, ctx.GetHeader("User-Agent"), ctx.ClientIP()); err != nil {
		tokenError(ctx, http.StatusInternalServerError, "server_error", "")
		return
	}

	accessToken, err := auth.GenerateOAuthAccessToken(user.ID, resource, scope, user.Admin, *user.Verified)
	if err != nil {
		tokenError(ctx, http.StatusInternalServerError, "server_error", "")
		return
	}

	resp := gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(auth.OAuthAccessTokenDuration.Seconds()),
		"scope":        scope,
	}
	if scopeContains(scope, "openid") {
		if idToken, err := auth.GenerateIDToken(user.ID, client.ClientID, derefString(user.Email), displayName(user)); err == nil {
			resp["id_token"] = idToken
		}
	}

	// First-party browser client keeps the refresh token in an HttpOnly cookie
	// (XSS-safe); other clients receive it in the response body.
	if client.IsFirstParty {
		setRefreshCookie(ctx, refreshPlain)
	} else {
		resp["refresh_token"] = refreshPlain
	}

	ctx.JSON(http.StatusOK, resp)
}

// APIOAuthRevoke revokes a refresh token (RFC 7009) and clears the browser's
// refresh + SSO cookies — it doubles as the web app's logout.
func APIOAuthRevoke(ctx *gin.Context) {
	token := ctx.PostForm("token")
	if token == "" {
		if cookie, err := ctx.Cookie(refreshCookieName); err == nil {
			token = cookie
		}
	}
	if token != "" {
		if err := database.RevokeSessionByRefreshHash(utilities.HashOpaqueToken(token)); err != nil {
			logger.Log.Error("Failed to revoke session. Error: " + err.Error())
		}
	}
	clearRefreshCookie(ctx)
	clearSSOCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "Revoked."})
}

// authenticateClient loads and authenticates the client at the token endpoint.
// Public clients need no secret (PKCE proves possession); confidential clients
// present a secret.
func authenticateClient(ctx *gin.Context, clientID string) (models.OAuthClient, bool) {
	client, found, err := database.GetOAuthClient(clientID)
	if err != nil || !found || !client.IsEnabled() {
		tokenError(ctx, http.StatusUnauthorized, "invalid_client", "")
		return models.OAuthClient{}, false
	}
	if !client.IsPublic {
		secret := ctx.PostForm("client_secret")
		if client.ClientSecretHash == nil || bcrypt.CompareHashAndPassword([]byte(*client.ClientSecretHash), []byte(secret)) != nil {
			tokenError(ctx, http.StatusUnauthorized, "invalid_client", "")
			return models.OAuthClient{}, false
		}
	}
	return client, true
}

func renderConsentPage(ctx *gin.Context, client models.OAuthClient, scopes []string, redirectURI string, state string, challenge string, resource string) {
	var items strings.Builder
	for _, name := range scopes {
		desc := name
		if s, ok := oauth.Lookup(name); ok {
			desc = s.Description
		}
		items.WriteString("<li>" + htmlEscape(desc) + "</li>")
	}

	page := `<!doctype html><html><head><meta charset="utf-8"><title>Authorize</title></head><body>` +
		`<h2>` + htmlEscape(client.ClientName) + ` wants to access your account</h2>` +
		`<ul>` + items.String() + `</ul>` +
		`<form method="post" action="/oauth/consent">` +
		hiddenField("client_id", client.ClientID) +
		hiddenField("redirect_uri", redirectURI) +
		hiddenField("scope", strings.Join(scopes, " ")) +
		hiddenField("state", state) +
		hiddenField("code_challenge", challenge) +
		hiddenField("resource", resource) +
		`<button type="submit" name="action" value="allow">Allow</button> ` +
		`<button type="submit" name="action" value="deny">Deny</button>` +
		`</form></body></html>`

	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

func redirectAuthError(ctx *gin.Context, redirectURI string, state string, errCode string) {
	target, err := url.Parse(redirectURI)
	if err != nil {
		ctx.String(http.StatusBadRequest, "Invalid redirect URI.")
		return
	}
	rq := target.Query()
	rq.Set("error", errCode)
	if state != "" {
		rq.Set("state", state)
	}
	target.RawQuery = rq.Encode()
	ctx.Redirect(http.StatusFound, target.String())
}

func tokenError(ctx *gin.Context, status int, errCode string, desc string) {
	body := gin.H{"error": errCode}
	if desc != "" {
		body["error_description"] = desc
	}
	ctx.JSON(status, body)
}

func scopeContains(scope string, name string) bool {
	for _, s := range strings.Fields(scope) {
		if s == name {
			return true
		}
	}
	return false
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func displayName(user models.User) string {
	return strings.TrimSpace(user.FirstName + " " + user.LastName)
}

func hiddenField(name string, value string) string {
	return `<input type="hidden" name="` + htmlEscape(name) + `" value="` + htmlEscape(value) + `">`
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return replacer.Replace(s)
}
