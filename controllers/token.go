package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"log"
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
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// check if email exists and password is correct
	user, err = database.GetAllUserInformationByEmail(request.Email)
	if err != nil {
		log.Println("Failed to get user by e-mail. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	err = user.CheckPassword(request.Password)
	if err != nil {
		log.Println("Failed to verify password. Error: " + err.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, user.Email, user.ID, *user.Admin, *user.Verified)
	if err != nil {
		log.Println("Failed to generate token. Error: " + err.Error())
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
		log.Println("Failed to validate session. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	} else if claims.ExpiresAt.Time.Before(now) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	}

	token := ""

	// Refresh login if it is over 24 hours old
	if claims.IssuedAt != nil {

		// Get time difference between now and token issue time
		difference := now.Sub(claims.IssuedAt.Time)

		if float64(difference.Hours()/24/365) < 1.0 && claims.ExpiresAt.After(now) {

			// Change expiration to now + seve ndays
			claims.ExpiresAt.Time = now.Add(time.Hour * 24 * 7)

			// Get user object by ID and check and update admin status
			userObject, userErr := database.GetUserInformation(claims.UserID)
			if userErr != nil {
				log.Println("Failed to check admin status during token refresh.")
				return
			} else if *userObject.Admin != claims.Admin {
				claims.Admin = *userObject.Admin
			}

			// Re-generate token with updated claims
			token, err = auth.GenerateJWTFromClaims(claims)
			if err != nil {
				log.Println("Failed to re-sign JWT from claims. Error: " + err.Error())
				token = ""
			}
		}

	}

	context.JSON(http.StatusOK, gin.H{"message": "Valid session!", "data": claims, "token": token})

}
