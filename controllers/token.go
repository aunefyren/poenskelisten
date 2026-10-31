package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"errors"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// TokenRequest is the password login body. The e-mail is trimmed but not
// lowercased, for the same reason as resetRequest's.
type TokenRequest struct {
	Email    string `json:"email" sanitize:"trim"`
	Password string `json:"password" sanitize:"none"`
}

// dummyPasswordHash is checked against when there's no password to compare
// with, so that a login for an unknown account takes as long as one for a
// real account with a wrong password. It's built on first use, at the cost
// real passwords are hashed with.
var dummyPasswordHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("no account has this password"), models.BcryptCost)
	if err != nil {
		logger.Log.Error("Failed to build the placeholder password hash. Error: " + err.Error())
	}
	return hash
})

// passwordMatches reports whether password is the user's. When there's no
// account (found is false) or it has no local password (OIDC), it still does
// the bcrypt work before saying no, so the response time doesn't tell an
// attacker which e-mail addresses have an account.
func passwordMatches(user models.User, found bool, password string) bool {
	if found && user.HasPassword() {
		return user.CheckPassword(password) == nil
	}
	_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(password))
	return false
}

func GenerateToken(context *gin.Context) {
	var request TokenRequest

	if !bindJSON(context, &request) {
		return
	}

	// check if email exists and password is correct
	user, err := database.GetAllUserInformationByEmail(request.Email)
	found := err == nil
	if err != nil && !errors.Is(err, database.ErrUserNotFound) {
		logger.Log.Error("Failed to get user by e-mail. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log in."})
		context.Abort()
		return
	}

	// An unknown e-mail gets the same answer as a wrong password: the same
	// status, the same message and, through passwordMatches, the same delay.
	if !passwordMatches(user, found, request.Password) {
		logger.Log.Info("Failed login attempt: invalid credentials.")
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	// If MFA is enabled, the password alone is not enough: issue a short-lived
	// challenge token and require the second factor via /open/tokens/mfa instead
	// of handing out a session token here.
	if user.IsMFAEnabled() {
		challengeToken, challengeID, err := auth.GenerateMFAChallengeToken(user.ID)
		if err == nil {
			err = database.SetMFAChallenge(user.ID, challengeID)
		}
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
