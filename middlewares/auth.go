package middlewares

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
		return false, "Request does not contain an access token", http.StatusBadRequest
	}

	err := auth.ValidateToken(tokenString, admin)
	if err != nil {
		logger.Log.Error("Failed to validate token. Error: " + err.Error())
		return false, "Failed to validate token.", http.StatusBadRequest
	}

	// If SMTP is enabled, verify if user is enabled
	if config.ConfigFile.SMTPEnabled {

		// Get userID from header
		userID, err := GetAuthUsername(context.GetHeader("Authorization"))
		if err != nil {
			logger.Log.Error("Failed to get user ID from token. Error: " + err.Error())
			return false, "Failed to get user ID from token.", http.StatusInternalServerError
		}

		// Check if the user is verified
		verified, err := database.VerifyUserIsVerified(userID)
		if err != nil {
			panic(err)
		}
		if !verified {

			// Verify user has verification code
			hasVerificationCode, err := database.VerifyUserHasVerificationCode(userID)
			if err != nil {
				logger.Log.Error("Failed to get verification code. Error: " + err.Error())
				return false, "Failed to get verification code.", http.StatusInternalServerError
			}

			// If the user doesn't have a code, set one
			if !hasVerificationCode {
				_, err := database.GenerateRandomVerificationCodeForUser(userID)
				if err != nil {
					logger.Log.Error("Failed to generate verification code. Error: " + err.Error())
					return false, "Failed to generate verification code.", http.StatusInternalServerError
				}
			}

			// Return error
			return false, "You must verify your account.", http.StatusForbidden
		}
	}

	return true, "", http.StatusOK
}

func GetAuthUsername(tokenString string) (uuid.UUID, error) {

	if tokenString == "" {
		return uuid.UUID{}, errors.New("No Authorization header given.")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return uuid.UUID{}, err
	}
	return claims.UserID, nil
}

func GetTokenClaims(tokenString string) (*auth.JWTClaim, error) {

	if tokenString == "" {
		return &auth.JWTClaim{}, errors.New("No Authorization header given.")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return &auth.JWTClaim{}, err
	}
	return claims, nil
}
