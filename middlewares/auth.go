package middlewares

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"fmt"
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

	claims, err := validateAPIToken(tokenString)
	if err != nil {
		logger.Log.Error("Failed to validate access token. Error: " + err.Error())
		return false, "failed to validate token.", http.StatusUnauthorized
	}

	if err := CheckNotSignedOut(claims); err != nil {
		if errors.Is(err, ErrTokenRevoked) {
			logger.Log.Info("Refused access token. Error: " + err.Error())
			return false, "failed to validate token.", http.StatusUnauthorized
		}
		logger.Log.Error("Failed to check whether the access token was revoked. Error: " + err.Error())
		return false, "failed to validate token.", http.StatusInternalServerError
	}

	if admin && !claims.Admin {
		return false, "insufficient permissions.", http.StatusForbidden
	}

	return true, "", http.StatusOK
}

// ErrTokenRevoked is returned by CheckNotSignedOut for a token that must be
// refused: it predates the user's last sign-out everywhere (a password reset or
// change, "sign out everywhere", an admin revoking the user's sessions), or its
// account no longer exists or is disabled.
var ErrTokenRevoked = errors.New("access token has been revoked")

// CheckNotSignedOut checks an already validated access token against its
// user's global logout marker. Access tokens are stateless, so without this
// check a revocation would leave every token already issued working until it
// expires. A refused token gives ErrTokenRevoked; any other error is a failed
// database lookup, which callers answer as a server error rather than a 401.
func CheckNotSignedOut(claims *auth.OAuthClaims) error {
	userID, err := utilities.ParseID(claims.Subject)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTokenRevoked, err)
	}
	marker, err := database.GetUserSessionsInvalidatedAt(userID)
	if errors.Is(err, database.ErrUserNotFound) {
		return fmt.Errorf("%w: %v", ErrTokenRevoked, err)
	} else if err != nil {
		return err
	}
	if auth.IssuedBeforeSignOut(claims.IssuedAt, marker) {
		return fmt.Errorf("%w: it was issued before the user was signed out everywhere", ErrTokenRevoked)
	}
	return nil
}

// GetAuthUsername returns the user ID (token subject) from an OAuth access token.
func GetAuthUsername(tokenString string) (uuid.UUID, error) {
	if tokenString == "" {
		return uuid.UUID{}, errors.New("no Authorization header given")
	}
	claims, err := validateAPIToken(tokenString)
	if err != nil {
		return uuid.UUID{}, err
	}
	return utilities.ParseID(claims.Subject)
}

// GetTokenClaims returns the validated OAuth claims from an access token.
func GetTokenClaims(tokenString string) (*auth.OAuthClaims, error) {
	if tokenString == "" {
		return nil, errors.New("no Authorization header given")
	}
	return validateAPIToken(tokenString)
}

// validateAPIToken validates an access token for the JSON API. The API has no
// per-route scope checks, so any token it accepts is full account access; only
// the built-in web app may hold one. Third-party clients are confined to the MCP
// resource, whose tools enforce their own mcp:* scopes.
func validateAPIToken(tokenString string) (*auth.OAuthClaims, error) {
	claims, err := auth.ValidateOAuthAccessToken(tokenString, config.APIResource())
	if err != nil {
		return nil, err
	}
	if claims.ClientID != models.FirstPartyClientID {
		return nil, errors.New("API access tokens must be issued to the first-party client")
	}
	return claims, nil
}
