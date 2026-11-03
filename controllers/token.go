package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TokenRequest is the password login body. The e-mail is trimmed but not
// lowercased, for the same reason as resetRequest's.
type TokenRequest struct {
	Email    string `json:"email" sanitize:"trim"`
	Password string `json:"password" sanitize:"none"`
}

func GenerateToken(context *gin.Context) {
	var request TokenRequest
	var user models.User

	if !bindJSON(context, &request) {
		return
	}

	// check if email exists and password is correct
	user, err := database.GetAllUserInformationByEmail(request.Email)
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

	if err := issueSSOSession(context, user); err != nil {
		logger.Log.Error("Failed to issue session. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log in."})
		context.Abort()
		return
	}

	// The browser is now logged in at the AS; the frontend continues the OAuth
	// authorization flow to obtain tokens. No token is returned here.
	context.JSON(http.StatusOK, gin.H{"message": "Logged in!"})
}
