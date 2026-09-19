package middlewares

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireLocalLogin guards the password-based endpoints (login, its MFA step,
// self-registration, password reset). When local login is disabled the only way
// in is OIDC, so these are refused server-side, not just hidden in the UI.
func RequireLocalLogin() gin.HandlerFunc {
	return func(context *gin.Context) {
		if config.LocalLoginEnabled() {
			context.Next()
			return
		}

		logger.Log.Warn("Refused " + context.Request.URL.Path + ": local login is disabled.")
		context.JSON(http.StatusForbidden, gin.H{"error": "Password login is disabled. Log in with " + config.OIDCDisplayName() + "."})
		context.Abort()
	}
}
