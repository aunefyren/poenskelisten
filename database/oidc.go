package database

import (
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// Sentinel errors returned by ResolveOIDCUser so the controller can map them to
// friendly, safe messages.
var (
	// ErrOIDCEmailNotVerified means the IdP did not assert a verified email, so we
	// refuse to link to or create a local account (guards against takeover).
	ErrOIDCEmailNotVerified = errors.New("oidc email is not verified")
	// ErrOIDCUserNotFound means no account matched and auto-provisioning is off.
	ErrOIDCUserNotFound = errors.New("no matching account and auto-create is disabled")
	// ErrOIDCNoEmail means the IdP returned no email, so we can't link or create.
	ErrOIDCNoEmail = errors.New("oidc token contained no email")
)

// GetUserByOIDCSubject looks up an enabled user previously linked to the given
// issuer + subject.
func GetUserByOIDCSubject(issuer string, subject string) (models.User, bool, error) {
	var user models.User
	record := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue, OIDCIssuer: &issuer, OIDCSubject: &subject}).
		Find(&user)
	if record.Error != nil {
		return models.User{}, false, record.Error
	}
	if record.RowsAffected == 0 {
		return models.User{}, false, nil
	}
	if record.RowsAffected != 1 {
		return models.User{}, false, errors.New("multiple users share the same OIDC subject")
	}
	return user, true, nil
}

// getEnabledUserByEmail returns an enabled user with the given email, reporting
// whether one was found without treating "not found" as an error.
func getEnabledUserByEmail(email string) (models.User, bool, error) {
	var user models.User
	record := Instance.
		Where(&models.User{Enabled: &utilities.DBTrue, Email: &email}).
		Find(&user)
	if record.Error != nil {
		return models.User{}, false, record.Error
	}
	if record.RowsAffected == 0 {
		return models.User{}, false, nil
	}
	if record.RowsAffected != 1 {
		return models.User{}, false, errors.New("multiple users share the same email")
	}
	return user, true, nil
}

// linkUserOIDC attaches an OIDC identity to an existing account. It deliberately
// leaves AuthSource untouched: a linked local account keeps its password and is
// still treated as a local account (e.g. for MFA enforcement).
func linkUserOIDC(userID uuid.UUID, issuer string, subject string) error {
	// Use a struct (not a raw map) so GORM resolves the actual column names for
	// the OIDC fields; only the non-zero pointer fields are updated.
	record := Instance.
		Model(&models.User{}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: userID}).
		Updates(models.User{OIDCIssuer: &issuer, OIDCSubject: &subject})
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("OIDC link not stored in database")
	}
	return nil
}

// createOIDCUser provisions a new account for an OIDC identity. The account is
// created pre-verified (the IdP vouched for the email) with no local password.
func createOIDCUser(issuer string, subject string, email string, firstName string, lastName string) (models.User, error) {
	trueVar := true
	authSource := models.AuthSourceOIDC

	user := models.User{
		FirstName:   firstName,
		LastName:    lastName,
		Email:       &email,
		Enabled:     &trueVar,
		Verified:    &trueVar,
		OIDCIssuer:  &issuer,
		OIDCSubject: &subject,
		AuthSource:  &authSource,
	}
	user.ID = uuid.New()

	// Mirror local registration: the very first account becomes an admin.
	count, err := GetAmountOfEnabledUsers()
	if err != nil {
		return models.User{}, err
	}
	if count == 0 {
		user.Admin = true
	}

	created, err := CreateUserInDB(user)
	if err != nil {
		return models.User{}, err
	}
	return created, nil
}

// ResolveOIDCUser maps a validated set of OIDC claims to a local account,
// applying the linking policy:
//
//  1. Match by issuer+subject (already linked) → log in.
//  2. Else match by email, but only link when the email is verified.
//  3. Else create a new account, but only when auto-create is enabled and the
//     email is verified.
func ResolveOIDCUser(issuer string, subject string, email string, firstName string, lastName string, emailVerified bool, autoCreate bool) (models.User, error) {
	// 1. Already linked by subject.
	if user, found, err := GetUserByOIDCSubject(issuer, subject); err != nil {
		return models.User{}, err
	} else if found {
		return user, nil
	}

	// Linking or creating both require an email.
	email = strings.TrimSpace(email)
	if email == "" {
		return models.User{}, ErrOIDCNoEmail
	}

	// 2. Link to an existing local account by verified email.
	existing, found, err := getEnabledUserByEmail(email)
	if err != nil {
		return models.User{}, err
	}
	if found {
		if !emailVerified {
			return models.User{}, ErrOIDCEmailNotVerified
		}
		if err := linkUserOIDC(existing.ID, issuer, subject); err != nil {
			return models.User{}, err
		}
		return GetAllUserInformation(existing.ID)
	}

	// 3. Auto-provision a new account.
	if !autoCreate {
		return models.User{}, ErrOIDCUserNotFound
	}
	if !emailVerified {
		return models.User{}, ErrOIDCEmailNotVerified
	}
	return createOIDCUser(issuer, subject, email, firstName, lastName)
}
