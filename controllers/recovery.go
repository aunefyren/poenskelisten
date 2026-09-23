package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
)

// Account recovery for operators with access to the server: these back the
// -resetpassword and -resetmfa startup flags. Being able to start the binary
// already means full control of the instance, so they add no new trust
// boundary. Neither depends on SMTP.

// IssuePasswordResetLink creates a reset code for the user with the given
// e-mail and returns the link to choose a new password. It goes through the
// same reset flow as the e-mailed link, so password rules and hashing apply
// as usual, and the code expires after a day or once used.
func IssuePasswordResetLink(email string) (models.User, string, error) {
	user, err := database.GetAllUserInformationByEmailCaseInsensitive(email)
	if err != nil {
		logger.Log.Error("Failed to find user for password reset. Error: " + err.Error())
		return models.User{}, "", errors.New("Failed to find user.")
	}

	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		logger.Log.Error("Failed to generate reset code. Error: " + err.Error())
		return models.User{}, "", errors.New("Failed to generate reset code.")
	}

	return user, utilities.PasswordResetLink(resetCode), nil
}

// ResetUserMFA removes MFA and recovery codes for the user with the given
// e-mail, the same as the admin "remove MFA" action.
func ResetUserMFA(email string) (models.User, error) {
	user, err := database.GetAllUserInformationByEmailCaseInsensitive(email)
	if err != nil {
		logger.Log.Error("Failed to find user for MFA reset. Error: " + err.Error())
		return models.User{}, errors.New("Failed to find user.")
	}

	err = database.DisableUserMFA(user.ID)
	if err != nil {
		logger.Log.Error("Failed to disable MFA. Error: " + err.Error())
		return models.User{}, errors.New("Failed to disable MFA.")
	}

	return user, nil
}
