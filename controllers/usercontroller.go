package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

func RegisterUser(context *gin.Context) {

	// Initialize variables
	var user models.User
	var usercreationrequest models.UserCreationRequest

	// Parse creation request
	if err := context.ShouldBindJSON(&usercreationrequest); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Make sure password match
	if usercreationrequest.Password != usercreationrequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Move values from request to object
	user.Email = usercreationrequest.Email
	user.Password = usercreationrequest.Password
	user.FirstName = usercreationrequest.FirstName
	user.LastName = usercreationrequest.LastName
	user.Enabled = true

	randomString := randstr.String(8)
	user.VerificationCode = strings.ToUpper(randomString)

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// If SMTP is disabled, create the user as verified
	if config.SMTPEnabled {
		user.Verified = false
	} else {
		user.Verified = true
	}

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

	// Set code to used
	err = database.SetUsedUserInviteCode(usercreationrequest.InviteCode, int(user.ID))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// If user is not verified and SMTP is enabled, send verification e-mail
	if !user.Verified && config.SMTPEnabled {

		log.Println("Sending verification e-mail to new user: " + user.FirstName + " " + user.LastName + ".")

		err = utilities.SendSMTPVerificationEmail(user)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
	}

	// Return response
	context.JSON(http.StatusCreated, gin.H{"message": "User created!"})

}

func GetUser(context *gin.Context) {

	// Create user request
	var user = context.Param("user_id")

	// Parse group id
	user_id_int, err := strconv.Atoi(user)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	user_object, err := database.GetUserInformation(user_id_int)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"user": user_object, "message": "User retrieved."})
}

func GetUsers(context *gin.Context) {

	// Create user request
	var user_struct []models.User

	userrecord := database.Instance.Where("`users`.enabled = ?", 1).Find(&user_struct)
	if userrecord.Error != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": userrecord.Error})
		context.Abort()
		return
	}

	for index, _ := range user_struct {
		// Redact user information
		user_struct[index].Email = "REDACTED"
		user_struct[index].Password = "REDACTED"
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"users": user_struct, "message": "Users retrieved."})
}

func VerifyUser(context *gin.Context) {

	// Get code from URL
	var code = context.Param("code")

	// Check if the string is empty
	if code == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "No code found."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify if code matches
	match, err := database.VerifyUserVerfificationCodeMatches(userID, code)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Check if code matches
	if !match {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Verificaton code invalid."})
		context.Abort()
		return
	}

	// Set account to verified
	err = database.SetUserVerification(userID, true)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user object
	var user models.User
	record := database.Instance.Where("ID = ?", userID).First(&user)
	if record.Error != nil {
		fmt.Println("Invalid credentials. Error: " + record.Error.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
		context.Abort()
		return
	}

	// Generate new JWT token
	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, user.Email, int(user.ID), *user.Admin, user.Verified)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "User verified.", "token": tokenString})

}

func SendUserVerificationCode(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Create a new code
	_, err = database.GenrateRandomVerificationCodeForuser(userID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user object
	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Send new e-mail
	err = utilities.SendSMTPVerificationEmail(user)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "New verification code sent."})

}

func UpdateUser(context *gin.Context) {

	// Initialize variables
	var userUpdateRequest models.UserUpdateRequest
	var err error
	emailChanged := false

	// Parse creation request
	if err := context.ShouldBindJSON(&userUpdateRequest); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Make sure password match
	if userUpdateRequest.Password != "" && userUpdateRequest.Password != userUpdateRequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user object
	var userOriginal models.User
	record := database.Instance.Where("ID = ?", userID).First(&userOriginal)
	if record.Error != nil {
		fmt.Println("Invalid credentials. Error: " + record.Error.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
		context.Abort()
		return
	}

	if userOriginal.Email != userUpdateRequest.Email {

		// Verify e-mail is not in use
		unique_email, err := database.VerifyUniqueUserEmail(userUpdateRequest.Email)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		} else if !unique_email {
			context.JSON(http.StatusBadRequest, gin.H{"error": "E-mail is already in use."})
			context.Abort()
			return
		}

		// Set account to not verified
		err = database.SetUserVerification(userID, false)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		userOriginal.Email = userUpdateRequest.Email
		emailChanged = true

	}

	// Hash the selected password
	if userUpdateRequest.Password != "" {
		if err := userOriginal.HashPassword(userUpdateRequest.Password); err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
	}

	// Update user in database
	err = database.UpdateUserValuesByUserID(int(userOriginal.ID), userOriginal.Email, userOriginal.Password)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get updated user object
	var user models.User
	record = database.Instance.Where("ID = ?", userID).First(&user)
	if record.Error != nil {
		fmt.Println("Invalid credentials. Error: " + record.Error.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
		context.Abort()
		return
	}

	// Generate new JWT token
	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, user.Email, int(user.ID), *user.Admin, user.Verified)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	// If SMTP is disabled, create the user as verified
	if config.SMTPEnabled {
		user.Verified = false
	} else {
		user.Verified = true
	}

	// If user is not verified and SMTP is enabled, send verification e-mail
	if !user.Verified && config.SMTPEnabled && emailChanged {

		verificationCode, err := database.GenrateRandomVerificationCodeForuser(userID)
		if err != nil {
			context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			context.Abort()
			return
		}

		user.VerificationCode = verificationCode

		log.Println("Sending verification e-mail to new user: " + user.FirstName + " " + user.LastName + ".")

		err = utilities.SendSMTPVerificationEmail(user)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Account updated.", "token": tokenString, "verified": user.Verified})

}
