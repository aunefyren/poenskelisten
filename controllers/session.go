package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// The OAuth refresh cookie (first-party web client). Scoped to /oauth so it is
	// only sent to the token/revoke endpoints.
	refreshCookieName   = "poenskelisten_refresh"
	refreshCookieMaxAge = 7 * 24 * 3600
	refreshCookiePath   = "/oauth"

	// The SSO login-state cookie, read only by /oauth/authorize.
	ssoCookieName   = "poenskelisten_sso"
	ssoCookieMaxAge = 12 * 3600
	ssoCookiePath   = "/"
)

func setRefreshCookie(ctx *gin.Context, token string) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(refreshCookieName, token, refreshCookieMaxAge, refreshCookiePath, "", oidcCookieSecure(), true)
}

func clearRefreshCookie(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", oidcCookieSecure(), true)
}

func setSSOCookie(ctx *gin.Context, token string) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(ssoCookieName, token, ssoCookieMaxAge, ssoCookiePath, "", oidcCookieSecure(), true)
}

func clearSSOCookie(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(ssoCookieName, "", -1, ssoCookiePath, "", oidcCookieSecure(), true)
}

// issueSSOSession records "this browser is logged in as this user" by setting the
// HS256 SSO cookie. This is the single place every browser login funnels through
// (password, MFA, OIDC, email verification). It does NOT issue OAuth tokens —
// those come from the /oauth/authorize + /oauth/token flow.
func issueSSOSession(ctx *gin.Context, user models.User) error {
	token, err := auth.GenerateSSOToken(user.ID)
	if err != nil {
		return err
	}
	setSSOCookie(ctx, token)
	return nil
}

// APILogoutAll signs the authenticated user out everywhere: it stamps the global
// invalidation marker (killing all SSO + access tokens issued before now) and
// revokes every refresh session.
func APILogoutAll(ctx *gin.Context) {
	userID, err := middlewares.GetAuthUsername(ctx.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		return
	}

	if err := revokeAllForUser(userID); err != nil {
		logger.Log.Error("Failed to sign out of all sessions. Error: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign out of all sessions."})
		return
	}

	clearRefreshCookie(ctx)
	clearSSOCookie(ctx)
	ctx.JSON(http.StatusOK, gin.H{"message": "Signed out of all sessions."})
}

// APIAdminRevokeUserSessions lets an admin force-log-out a user everywhere.
func APIAdminRevokeUserSessions(ctx *gin.Context) {
	targetID, err := uuid.Parse(ctx.Param("user_id"))
	if err != nil {
		logger.Log.Error("Failed to parse user ID. Error: " + err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		return
	}

	if err := revokeAllForUser(targetID); err != nil {
		logger.Log.Error("Failed to revoke user sessions. Error: " + err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke user sessions."})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Sessions revoked for user."})
}

// revokeAllForUser stamps the global invalidation marker and revokes all refresh
// sessions for a user.
func revokeAllForUser(userID uuid.UUID) error {
	if err := database.SetUserSessionsInvalidatedAt(userID, time.Now()); err != nil {
		return err
	}
	return database.RevokeAllUserSessions(userID)
}
