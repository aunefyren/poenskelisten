package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	GormModel
	FirstName        string     `json:"first_name" gorm:"not null"`
	LastName         string     `json:"last_name" gorm:"not null"`
	Email            *string    `json:"email" gorm:"unique; not null"`
	Password         *string    `json:"password" gorm:"not null; type: varchar(256);"`
	Admin            bool       `json:"admin" gorm:"not null; default: false"`
	Enabled          *bool      `json:"enabled" gorm:"not null; default: false"`
	Verified         *bool      `json:"verified" gorm:"not null; default: false"`
	VerificationCode *string    `json:"verification_code"`
	ResetCode        *string    `json:"reset_code"`
	ResetExpiration  *time.Time `json:"reset_expiration"`
}

type UserMinimal struct {
	GormModel
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     *string `json:"email"`
}

type UserCreationRequest struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"password_repeat"`
	InviteCode     string `json:"invite_code"`
}

type UserUpdateRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	PasswordRepeat   string `json:"password_repeat"`
	ProfileImage     string `json:"profile_image"`
	PasswordOriginal string `json:"password_original"`
}

type UserUpdatePasswordRequest struct {
	ResetCode      string `json:"reset_code"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"password_repeat"`
}

func (user *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	*user.Password = string(bytes)
	return nil
}

func (user *User) CheckPassword(providedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(providedPassword))
	if err != nil {
		return err
	}
	return nil
}
