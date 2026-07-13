package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	GormModel
	FirstName string  `json:"first_name" gorm:"not null"`
	LastName  string  `json:"last_name" gorm:"not null"`
	Email     *string `json:"email" gorm:"unique; not null"`
	// Password is nullable: OIDC-only users (introduced in a later phase) have no
	// local password. Local accounts always set one during registration.
	Password         *string    `json:"password" gorm:"type: varchar(256);"`
	Admin            bool       `json:"admin" gorm:"not null; default: false"`
	Enabled          *bool      `json:"enabled" gorm:"not null; default: false"`
	Verified         *bool      `json:"verified" gorm:"not null; default: false"`
	VerificationCode *string    `json:"verification_code"`
	ResetCode        *string    `json:"reset_code"`
	ResetExpiration  *time.Time `json:"reset_expiration"`

	// MFA (TOTP) fields.
	// MFASecret holds the TOTP shared secret encrypted at rest (see
	// utilities.EncryptString); it is never returned to clients.
	MFAEnabled    *bool      `json:"mfa_enabled" gorm:"not null; default: false"`
	MFASecret     *string    `json:"-"`
	MFAEnrolledAt *time.Time `json:"mfa_enrolled_at"`

	// OIDC linkage fields (reserved for the OpenID Connect phase; added now so the
	// schema is migrated once). OIDCSubject stores the IdP 'sub' claim.
	OIDCSubject *string `json:"-" gorm:"index"`
	OIDCIssuer  *string `json:"-"`
	AuthSource  *string `json:"auth_source"`

	// SessionsInvalidatedAt is a global logout marker: SSO login-state tokens and
	// access tokens issued before this instant are rejected. Set by "sign out
	// everywhere" and admin session revocation.
	SessionsInvalidatedAt *time.Time `json:"-"`
}

// MFARecoveryCode is a single-use backup code that lets a user complete MFA when
// they can't produce a TOTP code. Codes are stored hashed (bcrypt); UsedAt marks
// a code as consumed.
type MFARecoveryCode struct {
	GormModel
	UserID   uuid.UUID  `json:"user_id" gorm:"not null; index"`
	CodeHash string     `json:"-" gorm:"not null"`
	UsedAt   *time.Time `json:"used_at"`
}

// AuthSourceLocal / AuthSourceOIDC are the recognized values for User.AuthSource.
const (
	AuthSourceLocal = "local"
	AuthSourceOIDC  = "oidc"
)

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

// MFAActivateRequest carries the TOTP code a user enters to confirm enrollment.
type MFAActivateRequest struct {
	Code string `json:"code"`
}

// MFADisableRequest carries the credentials required to turn MFA off: the current
// password plus a valid TOTP or recovery code.
type MFADisableRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// MFAValidateRequest is the second login step: the challenge token issued after a
// correct password, plus a TOTP or recovery code.
type MFAValidateRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

// ServerSettingsRequest is the admin-editable runtime settings payload.
type ServerSettingsRequest struct {
	MFAEnforced             bool `json:"mfa_enforced"`
	MFARecoveryCodesEnabled bool `json:"mfa_recovery_codes_enabled"`
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
	if !user.HasPassword() {
		return errors.New("user has no local password")
	}
	err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(providedPassword))
	if err != nil {
		return err
	}
	return nil
}

// HasPassword reports whether the user has a usable local password. OIDC-only
// users (later phase) do not.
func (user *User) HasPassword() bool {
	return user.Password != nil && *user.Password != ""
}

// IsMFAEnabled reports whether TOTP is active for the user, tolerating a nil
// pointer (treated as disabled).
func (user *User) IsMFAEnabled() bool {
	return user.MFAEnabled != nil && *user.MFAEnabled
}

// IsLocalAuth reports whether the account authenticates with a local password
// rather than an external identity provider. A nil/empty AuthSource is treated
// as local, since every account today is local.
func (user *User) IsLocalAuth() bool {
	return user.AuthSource == nil || *user.AuthSource == "" || *user.AuthSource == AuthSourceLocal
}
