package middlewares

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Auth(admin bool) gin.HandlerFunc {
	return func(context *gin.Context) {
		tokenString := context.GetHeader("Authorization")
		if tokenString == "" {
			context.JSON(401, gin.H{"error": "Request does not contain an access token"})
			context.Abort()
			return
		}

		err := auth.ValidateToken(tokenString, admin)
		if err != nil {
			context.JSON(401, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		// Get configuration
		config, err := config.GetConfig()
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		// If SMTP is enabled, verify if user is enabled
		if config.SMTPEnabled {

			// Get userID from header
			userID, err := GetAuthUsername(context.GetHeader("Authorization"))
			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				context.Abort()
				return
			}

			// Check if the user is verified
			verified, err := database.VerifyUserIsVerified(userID)
			if !verified {

				// Verify user has verification code
				hasVerficationCode, err := database.VerifyUserHasVerfificationCode(userID)
				if err != nil {
					context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					context.Abort()
					return
				}

				// If the user doesn't have a code, set one
				if !hasVerficationCode {
					_, err := database.GenrateRandomVerificationCodeForuser(userID)
					if err != nil {
						context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
						context.Abort()
						return
					}
				}

				// Return error
				context.JSON(http.StatusForbidden, gin.H{"error": "You must verify your account."})
				context.Abort()
				return
			}

		}

		context.Next()
	}
}

func GetAuthUsername(tokenString string) (int, error) {

	if tokenString == "" {
		return 0, errors.New("No Auhtorization header given.")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

func GetTokenClaims(tokenString string) (*auth.JWTClaim, error) {

	if tokenString == "" {
		return &auth.JWTClaim{}, errors.New("No Auhtorization header given.")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return &auth.JWTClaim{}, err
	}
	return claims, nil
}
