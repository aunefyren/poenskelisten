package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

// Get redacted user information based on User ID
func GetUserInformation(UserID uuid.UUID) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	// Redact user information
	user = RedactUserObject(user)

	return user, nil
}

// Get ALL user information
func GetAllUserInformation(UserID uuid.UUID) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("enabled = ?", true).Where("id = ?", UserID).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Get redacted user information using email
func GetUserInformationByEmail(email string) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.email = ?", email).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	// Redact user information
	user = RedactUserObject(user)

	return user, nil
}

// Get ALL user information using email
func GetAllUserInformationByEmail(email string) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.email = ?", email).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Genrate a random reset code and return it
func GenrateRandomResetCodeForuser(userID uuid.UUID) (string, error) {

	randomString := randstr.String(8)
	resetCode := strings.ToUpper(randomString)

	expirationDate := time.Now().AddDate(0, 0, 7)

	var user models.User
	userrecord := Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("reset_code", resetCode)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Reset code not changed in database.")
	}

	userrecord = Instance.Model(user).Where("`users`.enabled = ?", 1).Where("`users`.ID = ?", userID).Update("reset_expiration", expirationDate)
	if userrecord.Error != nil {
		return "", userrecord.Error
	}
	if userrecord.RowsAffected != 1 {
		return "", errors.New("Reset code expiration not changed in database.")
	}

	return resetCode, nil

}

// Retrieve ALL user information using the reset code on the user object
func GetAllUserInformationByResetCode(resetCode string) (models.User, error) {
	var user models.User
	userrecord := Instance.Where("`users`.enabled = ?", 1).Where("`users`.reset_code = ?", resetCode).Find(&user)
	if userrecord.Error != nil {
		return models.User{}, userrecord.Error
	} else if userrecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Retrieves the amount of enabled users in the user table
func GetAmountOfEnabledUsers() (int, error) {

	var users []models.User

	userRecords := Instance.Where("enabled = ?", true).Find(&users)
	if userRecords.Error != nil {
		return 0, userRecords.Error
	}

	return int(userRecords.RowsAffected), nil

}

func RedactUserObject(user models.User) (userObject models.User) {
	userObject = user

	// Redact user information
	userObject.Password = "REDACTED"
	userObject.Email = "REDACTED"
	userObject.VerificationCode = "REDACTED"
	userObject.ResetCode = "REDACTED"
	userObject.ResetExpiration = time.Now()
	return
}

func GetEnabledUsers() (usersRedacted []models.User, err error) {
	users := []models.User{}
	usersRedacted = []models.User{}
	err = nil

	userrecord := Instance.Where("`users`.enabled = ?", 1).Find(&users)
	if userrecord.Error != nil {
		return usersRedacted, userrecord.Error
	}

	// Redact user information
	for _, user := range users {
		redactedUser := RedactUserObject(user)
		usersRedacted = append(usersRedacted, redactedUser)
	}

	return
}
