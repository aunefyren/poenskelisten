package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

func RegisterUser(context *gin.Context) {

	// Initialize variables
	var user models.User
	var usercreationrequest models.UserCreationRequest

	// Parse creation request
	if err := context.ShouldBindJSON(&usercreationrequest); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Trim request input
	usercreationrequest.Email = strings.TrimSpace(usercreationrequest.Email)
	usercreationrequest.FirstName = strings.TrimSpace(usercreationrequest.FirstName)
	usercreationrequest.LastName = strings.TrimSpace(usercreationrequest.LastName)
	usercreationrequest.InviteCode = strings.TrimSpace(usercreationrequest.InviteCode)

	// Make sure password match
	if usercreationrequest.Password != usercreationrequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Make password is strong enough
	valid, requirements, err := utilities.ValidatePasswordFormat(usercreationrequest.Password)
	if err != nil {
		log.Println("Failed to verify password quality. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password quality."})
		context.Abort()
		return
	} else if !valid {
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Move values from request to object
	user.Email = usercreationrequest.Email
	user.Password = usercreationrequest.Password
	user.FirstName = usercreationrequest.FirstName

	stringMatch, requirements, err := utilities.ValidateTextCharacters(user.FirstName)
	if err != nil {
		log.Println("Failed to validate first name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("First name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	user.LastName = usercreationrequest.LastName

	stringMatch, requirements, err = utilities.ValidateTextCharacters(user.LastName)
	if err != nil {
		log.Println("Failed to validate last name text string. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate text string."})
		context.Abort()
		return
	} else if !stringMatch {
		log.Println("Last name text string failed validation.")
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	user.Enabled = true
	user.ID = uuid.New()

	user.ResetExpiration = time.Now()

	randomString := randstr.String(8)
	user.VerificationCode = strings.ToUpper(randomString)

	randomString = randstr.String(8)
	user.ResetCode = strings.ToUpper(randomString)

	// Check if any users exist, if not, make new user admin
	userAmount, err := database.GetAmountOfEnabledUsers()
	if err != nil {
		log.Println("Failed to verify user amount. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify user amount."})
		context.Abort()
		return
	} else if userAmount == 0 {
		var adminBool bool = true
		user.Admin = &adminBool
		log.Println("No other users found. New user is set to admin.")
	}

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config."})
		context.Abort()
		return
	}

	// If SMTP is disabled, create the user as verified
	if config.SMTPEnabled {
		var verifiedBool bool = false
		user.Verified = &verifiedBool
	} else {
		var verifiedBool bool = true
		user.Verified = &verifiedBool
	}

	// Hash the selected password
	if err := user.HashPassword(user.Password); err != nil {
		log.Println("Failed to hash password. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password."})
		context.Abort()
		return
	}

	// Verify unsued invite code exists
	unique_invitecode, err := database.VerifyUnusedUserInviteCode(usercreationrequest.InviteCode)
	if err != nil {
		log.Println("Failed to verify invite code. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify invite code."})
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
		log.Println("Failed to verify unique e-mail. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify unique e-mail."})
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
		log.Println("Failed to get create user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get create user."})
		context.Abort()
		return
	}

	// Set code to used
	err = database.SetUsedUserInviteCode(usercreationrequest.InviteCode, user.ID)
	if err != nil {
		log.Println("Failed to set invite code to used. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to set invite code to used."})
		context.Abort()
		return
	}

	// If user is not verified and SMTP is enabled, send verification e-mail
	if !*user.Verified && config.SMTPEnabled {

		log.Println("Sending verification e-mail to new user: " + user.FirstName + " " + user.LastName + ".")

		err = utilities.SendSMTPVerificationEmail(user)
		if err != nil {
			log.Println("Failed to send e-mail. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to send e-mail."})
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

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	requestingUserObject, err := database.GetUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	// Parse group id
	user_id_int, err := uuid.Parse(user)
	if err != nil {
		log.Println("Failed to parse group ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
		context.Abort()
		return
	}

	userObject := models.User{}
	if userID == user_id_int || (requestingUserObject.Admin != nil && *requestingUserObject.Admin == true) {
		userObject, err = database.GetAllUserInformationAnyState(user_id_int)
		if err != nil {
			log.Println("Failed to get user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user (all)."})
			context.Abort()
			return
		}
	} else {
		userObject, err = database.GetUserInformation(user_id_int)
		if err != nil {
			log.Println("Failed to get user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
			context.Abort()
			return
		}
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"user": userObject, "message": "User retrieved."})
}

func GetUsers(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	requestingUserObject, err := database.GetUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	users := []models.User{}

	includeDisabled, okay := context.GetQuery("includeDisabled")
	if okay && strings.ToLower(includeDisabled) == "true" && requestingUserObject.Admin != nil && *requestingUserObject.Admin == true {
		users, err = database.GetAllUsers()
		if err != nil {
			log.Println("Failed to get all users. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all users."})
			context.Abort()
			return
		}
	} else {
		users, err = database.GetEnabledUsers()
		if err != nil {
			log.Println("Failed to get enabled users. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get enabled users."})
			context.Abort()
			return
		}
	}

	notAMemberOfGroupIDString, notAMemberOfGroupIDOkay := context.GetQuery("notAMemberOfGroupID")
	if notAMemberOfGroupIDOkay {
		newUsers := []models.User{}
		notAMemberOfGroupID, err := uuid.Parse(notAMemberOfGroupIDString)
		if err != nil {
			log.Println("Failed to parse group ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse group ID."})
			context.Abort()
			return
		}

		for _, user := range users {
			member, err := database.VerifyUserMembershipToGroup(user.ID, notAMemberOfGroupID)
			if err != nil {
				log.Println("Failed to verify ownership to group. Error: " + err.Error())
				context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify ownership to group."})
				context.Abort()
				return
			}

			if !member {
				newUsers = append(newUsers, user)
			}
		}

		users = newUsers
	}

	notACollaboratorOfWishlistString, notACollaboratorOfWishlistOkay := context.GetQuery("notACollaboratorOfWishlistID")
	if notACollaboratorOfWishlistOkay {
		newUsers := []models.User{}
		notACollaboratorOfWishlistID, err := uuid.Parse(notACollaboratorOfWishlistString)
		if err != nil {
			log.Println("Failed to parse wishlist ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist ID."})
			context.Abort()
			return
		}

		wishlist, err := database.GetWishlist(notACollaboratorOfWishlistID)
		if err != nil {
			log.Println("Failed to get wishlist. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse wishlist."})
			context.Abort()
			return
		}

		wishlistObject, err := ConvertWishlistToWishlistObject(wishlist, nil)
		if err != nil {
			log.Println("Failed to convert wishlist to wishlist object. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wishlist to wishlist object.."})
			context.Abort()
			return
		}

		for _, user := range users {
			userFound := false

			for _, collaborator := range wishlistObject.Collaborators {
				if collaborator.ID == user.ID {
					userFound = true
				}
			}

			if !userFound {
				newUsers = append(newUsers, user)
			}
		}

		users = newUsers
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"users": users, "message": "Users retrieved."})
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
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Verify if code matches
	match, err := database.VerifyUserVerfificationCodeMatches(userID, code)
	if err != nil {
		log.Println("Failed to get verification code. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get verification code."})
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
		log.Println("Failed to set user verification. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set user verification."})
		context.Abort()
		return
	}

	// Get user object
	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user details. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
		context.Abort()
		return
	}

	// Generate new JWT token
	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, user.Email, user.ID, *user.Admin, *user.Verified)
	if err != nil {
		log.Println("Failed to generate JWT token. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token."})
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
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Create a new code
	_, err = database.GenrateRandomVerificationCodeForuser(userID)
	if err != nil {
		log.Println("Failed to generate verification code. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification code."})
		context.Abort()
		return
	}

	// Get user object
	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	// Send new e-mail
	err = utilities.SendSMTPVerificationEmail(user)
	if err != nil {
		log.Println("Failed to send e-mail. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send e-mail."})
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

	// Parse creation request
	if err := context.ShouldBindJSON(&userUpdateRequest); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Trim request input
	userUpdateRequest.Email = strings.TrimSpace(userUpdateRequest.Email)

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	credentialError := user.CheckPassword(userUpdateRequest.PasswordOriginal)
	if credentialError != nil {
		log.Println("Invalid credentials.")
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	// Make sure password match
	if userUpdateRequest.Password != "" && userUpdateRequest.Password != userUpdateRequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Make password is strong enough
	valid, requirements, err := utilities.ValidatePasswordFormat(userUpdateRequest.Password)
	if err != nil {
		log.Println("Failed to verify password quality. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password quality."})
		context.Abort()
		return
	} else if !valid && userUpdateRequest.Password != "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Get user object
	userOriginal, err := database.GetAllUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user details. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details."})
		context.Abort()
		return
	}

	if userOriginal.Email != userUpdateRequest.Email {

		// Verify e-mail is not in use
		unique_email, err := database.VerifyUniqueUserEmail(userUpdateRequest.Email)
		if err != nil {
			log.Println("Failed to verify e-mail. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify e-mail."})
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
			log.Println("Failed to change verification. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change verification."})
			context.Abort()
			return
		}

		userOriginal.Email = userUpdateRequest.Email

	}

	// Hash the selected password
	if userUpdateRequest.Password != "" {
		if err := userOriginal.HashPassword(userUpdateRequest.Password); err != nil {
			log.Println("Failed to hash password. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password."})
			context.Abort()
			return
		}
	}

	// Update profile image
	if userUpdateRequest.ProfileImage != "" {
		err = UpdateUserProfileImage(userOriginal.ID, userUpdateRequest.ProfileImage)
		if err != nil {
			log.Println("Failed to update profile image. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile image."})
			context.Abort()
			return
		}
	}

	// Update user in database
	userOriginal, err = database.UpdateUserInDB(userOriginal)
	if err != nil {
		log.Println("Failed to update user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user."})
		context.Abort()
		return
	}

	// Get updated user object
	user, err = database.GetAllUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	// Generate new JWT token
	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, user.Email, user.ID, *user.Admin, *user.Verified)
	if err != nil {
		log.Println("Failed to generate JWT token. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token."})
		context.Abort()
		return
	}

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config."})
		context.Abort()
		return
	}

	// If user is not verified and SMTP is enabled, send verification e-mail
	if config.SMTPEnabled && !*user.Verified {

		verificationCode, err := database.GenrateRandomVerificationCodeForuser(userID)
		if err != nil {
			log.Println("Failed to generate verification code. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification code."})
			context.Abort()
			return
		}

		user.VerificationCode = verificationCode

		log.Println("Sending verification e-mail to new user: " + user.FirstName + " " + user.LastName + ".")

		err = utilities.SendSMTPVerificationEmail(user)
		if err != nil {
			log.Println("Failed to send e-mail. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send e-mail."})
			context.Abort()
			return
		}
	}

	// Reply
	context.JSON(http.StatusOK, gin.H{"message": "Account updated.", "token": tokenString, "verified": user.Verified})

}

func APIResetPassword(context *gin.Context) {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config."})
		context.Abort()
		return
	}

	if !config.SMTPEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The website administrator has not enabled SMTP."})
		context.Abort()
		return
	}

	if config.PoenskelistenExternalURL == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The website administrator has not setup an external website URL."})
		context.Abort()
		return
	}

	type resetRequest struct {
		Email string `json:"email"`
	}

	var resetRequestVar resetRequest

	// Parse reset request
	if err := context.ShouldBindJSON(&resetRequestVar); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	user, err := database.GetUserInformationByEmail(resetRequestVar.Email)
	if err != nil {
		log.Println("Failed to find user using email during password reset. Replied with okay 200. Error: " + err.Error())
		context.JSON(http.StatusOK, gin.H{"message": "If the user exists, an email with a password reset has been sent."})
		context.Abort()
		return
	}

	_, err = database.GenerateRandomResetCodeForUser(user.ID)
	if err != nil {
		log.Println("Failed to generate reset code for user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	user, err = database.GetAllUserInformation(user.ID)
	if err != nil {
		log.Println("Failed to retrieve data for user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	err = utilities.SendSMTPResetEmail(user)
	if err != nil {
		log.Println("Failed to send email to user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "If the user exists, an email with a password reset has been sent."})

}

func APIChangePassword(context *gin.Context) {

	// Initialize variables
	var user models.User
	var userUpdatePasswordRequest models.UserUpdatePasswordRequest

	// Parse creation request
	if err := context.ShouldBindJSON(&userUpdatePasswordRequest); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Make sure password match
	if userUpdatePasswordRequest.Password != userUpdatePasswordRequest.PasswordRepeat {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Passwords must match."})
		context.Abort()
		return
	}

	// Make password is strong enough
	valid, requirements, err := utilities.ValidatePasswordFormat(userUpdatePasswordRequest.Password)
	if err != nil {
		log.Println("Failed to verify password quality. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password quality."})
		context.Abort()
		return
	} else if !valid {
		context.JSON(http.StatusBadRequest, gin.H{"error": requirements})
		context.Abort()
		return
	}

	// Get user object using reset code
	user, err = database.GetAllUserInformationByResetCode(userUpdatePasswordRequest.ResetCode)
	if err != nil {
		log.Println("Failed to retrieve user using reset code. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Reset code has expired."})
		context.Abort()
		return
	}

	now := time.Now()

	// Check if code has expired
	if user.ResetExpiration.Before(now) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Reset code has expired."})
		context.Abort()
		return
	}

	// Hash the selected password
	if err = user.HashPassword(userUpdatePasswordRequest.Password); err != nil {
		log.Println("Failed to hash password. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password."})
		context.Abort()
		return
	}

	// Save new password
	user, err = database.UpdateUserInDB(user)
	if err != nil {
		log.Println("Failed to update user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user."})
		context.Abort()
		return
	}

	// Change the reset code
	_, err = database.GenerateRandomResetCodeForUser(user.ID)
	if err != nil {
		log.Println("Failed to generate reset code for user during password reset. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"message": "Error."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Password reset. You can now log in."})

}
