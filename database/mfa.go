package database

import (
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"errors"
	"time"

	"github.com/google/uuid"
)

// SetUserPendingMFASecret stores an encrypted TOTP secret for the user and marks
// MFA as not-yet-active. This is the first step of enrollment; the user confirms
// with a valid code before MFA is switched on (see ActivateUserMFA).
//
// Callers are expected to have already confirmed the user exists, so the update
// is validated on error only. RowsAffected is not enforced because re-enrolling
// with an identical (never, since the secret is random) value is not a concern
// and a missing user is caught upstream.
func SetUserPendingMFASecret(userID uuid.UUID, encryptedSecret string) error {
	record := Instance.
		Model(&models.User{}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: userID}).
		Updates(map[string]interface{}{
			"mfa_secret":      encryptedSecret,
			"mfa_enabled":     false,
			"mfa_enrolled_at": nil,
		})

	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("MFA secret not stored in database")
	}
	return nil
}

// ActivateUserMFA switches MFA on for the user once they have confirmed a code.
func ActivateUserMFA(userID uuid.UUID) error {
	now := time.Now()
	record := Instance.
		Model(&models.User{}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: userID}).
		Updates(map[string]interface{}{
			"mfa_enabled":     true,
			"mfa_enrolled_at": now,
		})

	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("MFA activation not stored in database")
	}
	return nil
}

// DisableUserMFA clears all MFA state for the user and removes their recovery
// codes. It is idempotent: disabling MFA for a user that has none is a no-op and
// not treated as an error (used by the admin "delete MFA" recovery path).
func DisableUserMFA(userID uuid.UUID) error {
	record := Instance.
		Model(&models.User{}).
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: userID}).
		Updates(map[string]interface{}{
			"mfa_secret":      nil,
			"mfa_enabled":     false,
			"mfa_enrolled_at": nil,
		})
	if record.Error != nil {
		return record.Error
	}

	// Remove any recovery codes (soft-deleted via GormModel).
	deleteRecord := Instance.
		Where(&models.MFARecoveryCode{UserID: userID}).
		Delete(&models.MFARecoveryCode{})
	if deleteRecord.Error != nil {
		return deleteRecord.Error
	}

	return nil
}

// StoreRecoveryCodes persists a batch of hashed recovery codes for the user.
func StoreRecoveryCodes(userID uuid.UUID, hashes []string) error {
	if len(hashes) == 0 {
		return nil
	}

	codes := make([]models.MFARecoveryCode, 0, len(hashes))
	for _, hash := range hashes {
		code := models.MFARecoveryCode{
			UserID:   userID,
			CodeHash: hash,
		}
		code.ID = uuid.New()
		codes = append(codes, code)
	}

	record := Instance.Create(&codes)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != int64(len(hashes)) {
		return errors.New("not all recovery codes were stored")
	}
	return nil
}

// ClearUserRecoveryCodes removes all recovery codes for the user without
// touching other MFA state. Used when regenerating codes during (re)enrollment.
func ClearUserRecoveryCodes(userID uuid.UUID) error {
	record := Instance.
		Where(&models.MFARecoveryCode{UserID: userID}).
		Delete(&models.MFARecoveryCode{})
	return record.Error
}

// GetActiveRecoveryCodes returns the user's unused recovery codes.
func GetActiveRecoveryCodes(userID uuid.UUID) ([]models.MFARecoveryCode, error) {
	codes := []models.MFARecoveryCode{}
	record := Instance.
		Where(&models.MFARecoveryCode{UserID: userID}).
		Where("used_at IS NULL").
		Find(&codes)
	if record.Error != nil {
		return nil, record.Error
	}
	return codes, nil
}

// MarkRecoveryCodeUsed marks a single recovery code as consumed.
func MarkRecoveryCodeUsed(codeID uuid.UUID) error {
	now := time.Now()
	record := Instance.
		Model(&models.MFARecoveryCode{}).
		Where(&models.GormModel{ID: codeID}).
		Update("used_at", now)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("recovery code not marked as used")
	}
	return nil
}

// GetUserMFAEnrollmentState returns whether the user has MFA enabled and whether
// the account is a local (password) account. Used by the auth middleware to
// enforce enrollment without pulling the full user object.
func GetUserMFAEnrollmentState(userID uuid.UUID) (enabled bool, isLocal bool, err error) {
	var user models.User
	record := Instance.
		Select("mfa_enabled", "auth_source").
		Where(&models.User{Enabled: &utilities.DBTrue}).
		Where(&models.GormModel{ID: userID}).
		Find(&user)
	if record.Error != nil {
		return false, false, record.Error
	}
	if record.RowsAffected != 1 {
		return false, false, errors.New("failed to find correct user in DB")
	}
	return user.IsMFAEnabled(), user.IsLocalAuth(), nil
}
