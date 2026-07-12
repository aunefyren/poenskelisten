package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type TokenRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func GenerateToken(context *gin.Context) {
	var request TokenRequest
	var user models.User

	err := context.ShouldBindJSON(&request)
	if err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// check if email exists and password is correct
	user, err = database.GetAllUserInformationByEmail(request.Email)
	if err != nil {
		logger.Log.Error("Failed to get user by e-mail. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	err = user.CheckPassword(request.Password)
	if err != nil {
		logger.Log.Error("Failed to verify password. Error: " + err.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	// If MFA is enabled, the password alone is not enough: issue a short-lived
	// challenge token and require the second factor via /open/tokens/mfa instead
	// of handing out a session token here.
	if user.IsMFAEnabled() {
		challengeToken, err := auth.GenerateMFAChallengeToken(user.ID)
		if err != nil {
			logger.Log.Error("Failed to generate MFA challenge token. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid credentials."})
			context.Abort()
			return
		}

		context.JSON(http.StatusOK, gin.H{"mfa_required": true, "mfa_token": challengeToken, "message": "Multi-factor authentication required."})
		return
	}

	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, *user.Email, user.ID, user.Admin, *user.Verified)
	if err != nil {
		logger.Log.Error("Failed to generate token. Error: " + err.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"token": tokenString, "message": "Logged in!"})
}

func ValidateToken(context *gin.Context) {
	now := time.Now()

	claims, err := middlewares.GetTokenClaims(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to validate session. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	} else if claims.ExpiresAt.Time.Before(now) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	}

	token := ""

	// Refresh login if it was issued over 24 hours ago (and is still valid), so
	// active sessions keep sliding forward instead of expiring abruptly.
	if claims.IssuedAt != nil {

		// Get time difference between now and token issue time
		difference := now.Sub(claims.IssuedAt.Time)

		if difference > 24*time.Hour && claims.ExpiresAt.After(now) {

			// Slide the issue/expiry window forward by the standard token lifetime
			claims.IssuedAt.Time = now
			claims.ExpiresAt.Time = now.Add(auth.TokenValidDuration)

			// Get user object by ID and check and update admin status
			userObject, err := database.GetUserInformation(claims.UserID)
			if err != nil {
				logger.Log.Error("Failed to check admin status during token refresh. Error: " + err.Error())
				context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to validate session. Please log in again."})
				context.Abort()
				return
			} else if userObject.Admin != claims.Admin {
				claims.Admin = userObject.Admin
			}

			// Re-generate token with updated claims
			token, err = auth.GenerateJWTFromClaims(claims)
			if err != nil {
				logger.Log.Error("Failed to re-sign JWT from claims. Error: " + err.Error())
				context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to validate session. Please log in again."})
				context.Abort()
				return
			}
		}

	}

	// Signal the frontend to route the user to MFA enrollment when enforcement is
	// on and this local account hasn't enrolled yet.
	mfaEnrollmentRequired := false
	if config.ConfigFile.MFAEnforced {
		enabled, isLocal, err := database.GetUserMFAEnrollmentState(claims.UserID)
		if err != nil {
			logger.Log.Error("Failed to check MFA enrollment state. Error: " + err.Error())
		} else if isLocal && !enabled {
			mfaEnrollmentRequired = true
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Valid session!", "data": claims, "token": token, "mfa_enrollment_required": mfaEnrollmentRequired})

}
