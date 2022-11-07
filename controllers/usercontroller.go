package controllers

import (
	"net/http"
	"poenskelisten/database"
	"poenskelisten/models"

	"github.com/gin-gonic/gin"
)

func RegisterUser(context *gin.Context) {
	var user models.User
	var usercreationrequest models.UserCreationRequest
	if err := context.ShouldBindJSON(&usercreationrequest); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if usercreationrequest.Password != usercreationrequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	user.Email = usercreationrequest.Email
	user.Password = usercreationrequest.Password
	user.FirstName = usercreationrequest.FirstName
	user.LastName = usercreationrequest.LastName

	// Hash the selected password
	if err := user.HashPassword(user.Password); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify unsued invite code exists
	unique_invitecode, err := database.VerifyUnusedUserInviteCode(usercreationrequest.InviteCode)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !unique_invitecode {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invitiation code is not valid."})
		context.Abort()
		return
	}

	// Verify e-mail is not in use
	unique_email, err := database.VerifyUniqueUserEmail(user.Email)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !unique_email {
		context.JSON(http.StatusBadRequest, gin.H{"error": "E-mail is already in use."})
		context.Abort()
		return
	}

	// Create user in DB
	record := database.Instance.Create(&user)
	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	// SET CODE TO USED

	// Return response
	context.JSON(http.StatusCreated, gin.H{"userId": user.ID, "email": user.Email})
}
