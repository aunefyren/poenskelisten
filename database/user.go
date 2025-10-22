package database

import (
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

// Get redacted user information based on User ID for enabled users
func GetUserInformation(UserID uuid.UUID) (models.User, error) {
	var user models.User

	userRecord := Instance.
		Where(&models.GormModel{ID: UserID}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Find(&user)

	if userRecord.Error != nil {
		return models.User{}, userRecord.Error
	} else if userRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	// Redact user information
	user = RedactUserObject(user)

	return user, nil
}

// Get redacted user information based on User ID for all users
func GetUserInformationAnyState(UserID uuid.UUID) (models.User, error) {
	var user models.User

	userRecord := Instance.
		Where(&models.GormModel{ID: UserID}).
		Find(&user)

	if userRecord.Error != nil {
		return models.User{}, userRecord.Error
	} else if userRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	// Redact user information
	user = RedactUserObject(user)

	return user, nil
}

// Get ALL user information for enabled users (non-redacted)
func GetAllUserInformation(UserID uuid.UUID) (models.User, error) {
	var user models.User

	userRecord := Instance.
		Where(&models.GormModel{ID: UserID}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Find(&user)

	if userRecord.Error != nil {
		return models.User{}, userRecord.Error
	} else if userRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Get ALL user information for ALL users (non-redacted)
func GetAllUserInformationAnyState(UserID uuid.UUID) (models.User, error) {
	var user models.User

	userRecord := Instance.
		Where(&models.GormModel{ID: UserID}).
		Find(&user)

	if userRecord.Error != nil {
		return models.User{}, userRecord.Error
	} else if userRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Get redacted user information using email
func GetUserInformationByEmail(email string) (models.User, error) {
	var user models.User

	userRecord := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue, Email: &email}).
		Find(&user)

	if userRecord.Error != nil {
		return models.User{}, userRecord.Error
	} else if userRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	// Redact user information
	user = RedactUserObject(user)

	return user, nil
}

// Get ALL user information using email
func GetAllUserInformationByEmail(email string) (models.User, error) {
	var user models.User

	userRecord := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue, Email: &email}).
		Find(&user)

	if userRecord.Error != nil {
		return models.User{}, userRecord.Error
	} else if userRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Generate a random reset code and return it
func GenerateRandomResetCodeForUser(userID uuid.UUID) (string, error) {
	randomString := randstr.String(8)
	resetCode := strings.ToUpper(randomString)

	expirationDate := time.Now().AddDate(0, 0, 7)

	var user models.User
	userRecord := Instance.
		Model(user).Where(&models.GormModel{ID: userID}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Update("reset_code", resetCode)

	if userRecord.Error != nil {
		return "", userRecord.Error
	}
	if userRecord.RowsAffected != 1 {
		return "", errors.New("Reset code not changed in database.")
	}

	userRecord = Instance.Model(user).Where(&models.GormModel{ID: userID}).Where(&models.User{Enabled: &utilities.DBTrue}).Update("reset_expiration", expirationDate)
	if userRecord.Error != nil {
		return "", userRecord.Error
	}
	if userRecord.RowsAffected != 1 {
		return "", errors.New("Reset code expiration not changed in database.")
	}

	return resetCode, nil
}

// Retrieve ALL user information using the reset code on the user object
func GetAllUserInformationByResetCode(resetCode string) (models.User, error) {
	var user models.User

	userRecord := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue, ResetCode: &resetCode}).
		Find(&user)

	if userRecord.Error != nil {
		return models.User{}, userRecord.Error
	} else if userRecord.RowsAffected != 1 {
		return models.User{}, errors.New("Failed to find correct user in DB.")
	}

	return user, nil
}

// Retrieves the amount of enabled users in the user table
func GetAmountOfEnabledUsers() (int, error) {
	var users []models.User

	userRecords := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Find(&users)

	if userRecords.Error != nil {
		return 0, userRecords.Error
	}

	return int(userRecords.RowsAffected), nil
}

func RedactUserObject(user models.User) (userObject models.User) {
	userObject = user

	// Redact user information
	userObject.Password = nil
	userObject.VerificationCode = nil
	userObject.Verified = nil
	userObject.ResetCode = nil
	userObject.ResetExpiration = nil
	return
}

func GetEnabledUsers() (usersRedacted []models.User, err error) {
	users := []models.User{}
	usersRedacted = []models.User{}
	err = nil

	userRecord := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Find(&users)

	if userRecord.Error != nil {
		return usersRedacted, userRecord.Error
	}

	// Redact user information
	for _, user := range users {
		redactedUser := RedactUserObject(user)
		usersRedacted = append(usersRedacted, redactedUser)
	}

	return
}

// Gets enabled and disabled users
func GetAllUsers() (usersRedacted []models.User, err error) {
	users := []models.User{}
	usersRedacted = []models.User{}
	err = nil

	userRecord := Instance.Find(&users)

	if userRecord.Error != nil {
		return usersRedacted, userRecord.Error
	}

	// Redact user information
	for _, user := range users {
		redactedUser := RedactUserObject(user)
		usersRedacted = append(usersRedacted, redactedUser)
	}

	return
}

func UpdateUserInDB(userOriginal models.User) (user models.User, err error) {
	err = nil
	user = userOriginal

	record := Instance.Save(&user)

	if record.Error != nil {
		return user, record.Error
	}

	return
}

func CreateUserInDB(userRequest models.User) (user models.User, err error) {
	user = userRequest

	record := Instance.Create(&user)
	if record.Error != nil {
		logger.Log.Error("Failed to create user in DB. Error: " + record.Error.Error())
		return user, errors.New("Failed to create user in DB.")
	}

	return
}
