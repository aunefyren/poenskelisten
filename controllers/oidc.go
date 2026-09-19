package controllers

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/oidcprovider"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

const (
	oidcStateCookie = "oidc_state"
	oidcNonceCookie = "oidc_nonce"
	// oidcFlowCookieMaxAge bounds how long a login attempt (state/nonce) is valid.
	oidcFlowCookieMaxAge = 600 // 10 minutes
	sessionCookieMaxAge  = 7 * 24 * 3600
)

// APIGetOIDCConfig exposes just enough for the login page to render (or hide) the
// single sign-on button. It is intentionally public and reveals no secrets.
func APIGetOIDCConfig(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{
		"enabled":             config.ConfigFile.OIDCEnabled,
		"provider_name":       config.OIDCDisplayName(),
		"login_url":           "/api/open/oidc/login",
		"local_login_enabled": config.LocalLoginEnabled(),
	})
}

// OIDCLogin starts the authorization-code flow: it stores a random state and
// nonce in short-lived cookies and redirects the browser to the identity
// provider.
func OIDCLogin(ctx *gin.Context) {
	client, err := oidcprovider.Get()
	if err != nil {
		logger.Log.Error("OIDC login requested but not available. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on is not available.")
		return
	}

	state, err := randomToken()
	if err != nil {
		logger.Log.Error("Failed to generate OIDC state. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on failed to start.")
		return
	}
	nonce, err := randomToken()
	if err != nil {
		logger.Log.Error("Failed to generate OIDC nonce. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on failed to start.")
		return
	}

	setFlowCookie(ctx, oidcStateCookie, state)
	setFlowCookie(ctx, oidcNonceCookie, nonce)

	authURL := client.OAuth2Config.AuthCodeURL(state, oidc.Nonce(nonce))
	ctx.Redirect(http.StatusFound, authURL)
}

// OIDCCallback completes the flow: it validates state, exchanges the code,
// verifies the ID token (signature, audience, nonce), resolves the local account,
// and issues a session cookie before redirecting back into the app.
func OIDCCallback(ctx *gin.Context) {
	client, err := oidcprovider.Get()
	if err != nil {
		logger.Log.Error("OIDC callback but provider unavailable. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on is not available.")
		return
	}

	// Validate state (CSRF protection).
	stateParam := ctx.Query("state")
	stateCookie, _ := ctx.Cookie(oidcStateCookie)
	if stateParam == "" || stateCookie == "" || stateParam != stateCookie {
		logger.Log.Error("OIDC callback state mismatch.")
		redirectLoginError(ctx, "Single sign-on failed. Please try again.")
		return
	}

	code := ctx.Query("code")
	if code == "" {
		logger.Log.Error("OIDC callback missing code. Provider error: " + ctx.Query("error"))
		redirectLoginError(ctx, "Single sign-on was cancelled or failed.")
		return
	}

	oauth2Token, err := client.OAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		logger.Log.Error("OIDC code exchange failed. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on failed.")
		return
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		logger.Log.Error("OIDC token response missing id_token.")
		redirectLoginError(ctx, "Single sign-on failed.")
		return
	}

	idToken, err := client.Verifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		logger.Log.Error("OIDC ID token verification failed. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on failed.")
		return
	}

	// Bind the token to this browser's login attempt.
	nonceCookie, _ := ctx.Cookie(oidcNonceCookie)
	if idToken.Nonce == "" || idToken.Nonce != nonceCookie {
		logger.Log.Error("OIDC nonce mismatch.")
		redirectLoginError(ctx, "Single sign-on failed. Please try again.")
		return
	}

	var claims oidcClaims
	if err := idToken.Claims(&claims); err != nil {
		logger.Log.Error("Failed to parse OIDC claims. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on failed.")
		return
	}

	// Many IdPs (e.g. Authelia >= 4.39, by default) put the email/profile claims
	// only in the userinfo response, not the ID token. A userinfo failure isn't
	// fatal here: account resolution below still refuses a missing email.
	if claims.Email == "" || !claims.hasName() {
		if err := mergeUserInfoClaims(client.Provider, oauth2Token, idToken.Subject, &claims); err != nil {
			logger.Log.Warn("Failed to read OIDC userinfo claims. Error: " + err.Error())
		}
	}

	firstName, lastName := deriveNames(claims.GivenName, claims.FamilyName, claims.Name, claims.Email)

	user, err := database.ResolveOIDCUser(
		idToken.Issuer,
		idToken.Subject,
		claims.Email,
		firstName,
		lastName,
		claims.EmailVerified,
		config.ConfigFile.OIDCAutoCreateUsers,
	)
	if err != nil {
		logger.Log.Error("Failed to resolve OIDC user. Error: " + err.Error())
		redirectLoginError(ctx, oidcResolveErrorMessage(err))
		return
	}

	// OIDC login establishes the SSO session; the frontend then continues the OAuth
	// authorization flow (on load) to obtain tokens.
	if err := issueSSOSession(ctx, user); err != nil {
		logger.Log.Error("Failed to issue session after OIDC login. Error: " + err.Error())
		redirectLoginError(ctx, "Single sign-on failed.")
		return
	}

	// gin finalizes response headers at redirect time, so the one-time flow
	// cookies must be cleared before Redirect is called, not deferred after it.
	clearFlowCookie(ctx, oidcStateCookie)
	clearFlowCookie(ctx, oidcNonceCookie)

	ctx.Redirect(http.StatusFound, "/")
}

// oidcClaims is the subset of ID-token/userinfo claims used to resolve and
// provision the local account.
type oidcClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Name          string `json:"name"`
}

func (c oidcClaims) hasName() bool {
	return c.GivenName != "" || c.FamilyName != "" || c.Name != ""
}

// mergeUserInfoClaims fills claims missing from the ID token from the IdP's
// userinfo endpoint. Claims the ID token did carry are never overridden.
func mergeUserInfoClaims(provider *oidc.Provider, token *oauth2.Token, subject string, claims *oidcClaims) error {
	info, err := provider.UserInfo(context.Background(), oauth2.StaticTokenSource(token))
	if err != nil {
		return err
	}
	// OIDC Core 5.3.2: a userinfo response for a different subject than the ID
	// token must not be used (it could be a substituted token's).
	if info.Subject != subject {
		return errors.New("userinfo subject does not match the ID token subject")
	}

	var extra oidcClaims
	if err := info.Claims(&extra); err != nil {
		return err
	}

	// email and email_verified travel together: verification only counts for
	// the address it was asserted about.
	if claims.Email == "" {
		claims.Email = extra.Email
		claims.EmailVerified = extra.EmailVerified
	} else if strings.EqualFold(claims.Email, extra.Email) && extra.EmailVerified {
		claims.EmailVerified = true
	}
	if !claims.hasName() {
		claims.GivenName = extra.GivenName
		claims.FamilyName = extra.FamilyName
		claims.Name = extra.Name
	}
	return nil
}

// deriveNames picks a first/last name from the available claims, falling back to
// the display name and finally the email local-part so the account always has
// something readable.
func deriveNames(given string, family string, name string, email string) (firstName string, lastName string) {
	given = strings.TrimSpace(given)
	family = strings.TrimSpace(family)
	if given != "" || family != "" {
		return given, family
	}

	// strings.Fields collapses runs of whitespace, so "Ada  Lovelace" splits
	// cleanly and a middle name folds into the last name.
	if fields := strings.Fields(name); len(fields) > 0 {
		if len(fields) == 1 {
			return fields[0], ""
		}
		return fields[0], strings.Join(fields[1:], " ")
	}

	if at := strings.Index(email, "@"); at > 0 {
		return email[:at], ""
	}
	return "User", ""
}

// oidcResolveErrorMessage maps resolution errors to safe user-facing text.
func oidcResolveErrorMessage(err error) string {
	switch {
	case errors.Is(err, database.ErrOIDCEmailNotVerified):
		return "Your identity provider has not verified your email address."
	case errors.Is(err, database.ErrOIDCUserNotFound):
		return "No account is linked to this login. Please contact an administrator."
	case errors.Is(err, database.ErrOIDCNoEmail):
		return "Your identity provider did not share an email address."
	default:
		return "Single sign-on failed."
	}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func oidcCookieSecure() bool {
	return strings.HasPrefix(strings.ToLower(config.ConfigFile.PoenskelistenExternalURL), "https")
}

func setFlowCookie(ctx *gin.Context, name string, value string) {
	// Lax so the cookie survives the top-level redirect back from the IdP, while
	// still not being sent on cross-site subrequests. HttpOnly: JS never needs it.
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(name, value, oidcFlowCookieMaxAge, "/", "", oidcCookieSecure(), true)
}

func clearFlowCookie(ctx *gin.Context, name string) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(name, "", -1, "/", "", oidcCookieSecure(), true)
}

func redirectLoginError(ctx *gin.Context, message string) {
	// Cookies must be cleared before Redirect is called: gin finalizes response
	// headers at that point, so a deferred SetCookie afterwards is too late.
	// Clearing cookies that were never set (e.g. failures before the OIDC flow
	// started) is a harmless no-op.
	clearFlowCookie(ctx, oidcStateCookie)
	clearFlowCookie(ctx, oidcNonceCookie)
	ctx.Redirect(http.StatusFound, "/login?error="+url.QueryEscape(message))
}
