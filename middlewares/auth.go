package middlewares

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// The API is an OAuth 2.1 resource server: it validates ES256 access tokens whose
// audience is the API resource. User-authentication gates (email verification,
// MFA enrollment) are enforced earlier, at /oauth/authorize, so a valid access
// token already implies a verified, enrolled user.

func Auth(admin bool) gin.HandlerFunc {
	return func(context *gin.Context) {
		success, errorString, httpStatus := AuthFunction(context, admin)

		if !success {
			context.JSON(httpStatus, gin.H{"error": errorString})
			context.Abort()
			return
		}

		context.Next()
	}
}

func AuthFunction(context *gin.Context, admin bool) (success bool, errorString string, httpStatus int) {
	tokenString := context.GetHeader("Authorization")
	if tokenString == "" {
		return false, "request does not contain an access token", http.StatusUnauthorized
	}

	claims, err := auth.ValidateOAuthAccessToken(tokenString, config.APIResource())
	if err != nil {
		logger.Log.Error("Failed to validate access token. Error: " + err.Error())
		return false, "failed to validate token.", http.StatusUnauthorized
	}

	if admin && !claims.Admin {
		return false, "insufficient permissions.", http.StatusForbidden
	}

	return true, "", http.StatusOK
}

// GetAuthUsername returns the user ID (token subject) from an OAuth access token.
func GetAuthUsername(tokenString string) (uuid.UUID, error) {
	if tokenString == "" {
		return uuid.UUID{}, errors.New("no Authorization header given")
	}
	claims, err := auth.ValidateOAuthAccessToken(tokenString, config.APIResource())
	if err != nil {
		return uuid.UUID{}, err
	}
	return uuid.Parse(claims.Subject)
}

// GetTokenClaims returns the validated OAuth claims from an access token.
func GetTokenClaims(tokenString string) (*auth.OAuthClaims, error) {
	if tokenString == "" {
		return nil, errors.New("no Authorization header given")
	}
	return auth.ValidateOAuthAccessToken(tokenString, config.APIResource())
}
