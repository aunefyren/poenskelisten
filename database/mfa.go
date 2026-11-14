package database

import (
	"aunefyren/poenskelisten/models"
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
		Where("id = ? AND enabled = ?", userID, true).
		Updates(map[string]interface{}{
			"mfa_secret":      encryptedSecret,
			"mfa_enabled":     false,
			"mfa_enrolled_at": nil,
			"mfa_last_step":   nil,
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
		Where("id = ? AND enabled = ?", userID, true).
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
		Where("id = ? AND enabled = ?", userID, true).
		Updates(map[string]interface{}{
			"mfa_secret":       nil,
			"mfa_enabled":      false,
			"mfa_enrolled_at":  nil,
			"mfa_last_step":    nil,
			"mfa_challenge_id": nil,
		})
	if record.Error != nil {
		return record.Error
	}

	// Remove any recovery codes (soft-deleted via GormModel).
	deleteRecord := Instance.
		Where("user_id = ?", userID).
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
		Where("user_id = ?", userID).
		Delete(&models.MFARecoveryCode{})
	return record.Error
}

// GetActiveRecoveryCodes returns the user's unused recovery codes.
func GetActiveRecoveryCodes(userID uuid.UUID) ([]models.MFARecoveryCode, error) {
	codes := []models.MFARecoveryCode{}
	record := Instance.
		Where("user_id = ? AND used_at IS NULL", userID).
		Find(&codes)
	if record.Error != nil {
		return nil, record.Error
	}
	return codes, nil
}

// ErrRecoveryCodeUsed means the recovery code was used, by another sign-in,
// after it was looked up.
var ErrRecoveryCodeUsed = errors.New("recovery code already used")

// MarkRecoveryCodeUsed consumes an unused recovery code. Only one of two
// sign-ins sending the same code at once gets it; the other gets
// ErrRecoveryCodeUsed.
func MarkRecoveryCodeUsed(codeID uuid.UUID) error {
	now := time.Now()
	record := Instance.
		Model(&models.MFARecoveryCode{}).
		Where("id = ? AND used_at IS NULL", codeID).
		Update("used_at", now)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return ErrRecoveryCodeUsed
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
		Where("id = ? AND enabled = ?", userID, true).
		Find(&user)
	if record.Error != nil {
		return false, false, record.Error
	}
	if record.RowsAffected != 1 {
		return false, false, errors.New("failed to find correct user in DB")
	}
	return user.IsMFAEnabled(), user.IsLocalAuth(), nil
}

// UseTOTPStep records that the user's TOTP code for step was accepted, and
// reports false when that step, or a later one, was already used: each code
// works once, even when sent twice at the same time.
func UseTOTPStep(userID uuid.UUID, step int64) (bool, error) {
	record := Instance.
		Model(&models.User{}).
		Where("id = ? AND (mfa_last_step IS NULL OR mfa_last_step < ?)", userID, step).
		Update("mfa_last_step", step)
	if record.Error != nil {
		return false, record.Error
	}
	return record.RowsAffected == 1, nil
}

// SetMFAChallenge records the challenge a password sign-in was given. It
// replaces any earlier one: only the latest sign-in can be completed.
func SetMFAChallenge(userID uuid.UUID, challengeID uuid.UUID) error {
	record := Instance.
		Model(&models.User{}).
		Where("id = ? AND enabled = ?", userID, true).
		Update("mfa_challenge_id", challengeID)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("MFA challenge not stored in database")
	}
	return nil
}

// ConsumeMFAChallenge completes the user's pending challenge if it is the
// given one, and reports false when it was already completed or replaced.
func ConsumeMFAChallenge(userID uuid.UUID, challengeID uuid.UUID) (bool, error) {
	record := Instance.
		Model(&models.User{}).
		Where("id = ? AND mfa_challenge_id = ?", userID, challengeID).
		Update("mfa_challenge_id", nil)
	if record.Error != nil {
		return false, record.Error
	}
	return record.RowsAffected == 1, nil
}

// CompleteMFAChallengeWithTOTP completes the user's pending challenge and
// uses up the TOTP step in one update, and reports false, changing neither,
// unless the challenge is still the given one and the step is unused. Done
// as two updates, a sign-in that loses the race for the challenge would
// still use up the user's code.
func CompleteMFAChallengeWithTOTP(userID uuid.UUID, challengeID uuid.UUID, step int64) (bool, error) {
	record := Instance.
		Model(&models.User{}).
		Where("id = ? AND mfa_challenge_id = ? AND (mfa_last_step IS NULL OR mfa_last_step < ?)", userID, challengeID, step).
		Updates(map[string]interface{}{
			"mfa_last_step":    step,
			"mfa_challenge_id": nil,
		})
	if record.Error != nil {
		return false, record.Error
	}
	return record.RowsAffected == 1, nil
}

// ClearMFAChallenge drops the user's pending MFA challenge, if any, so a
// sign-in begun with the old password can't be finished after a change.
func ClearMFAChallenge(userID uuid.UUID) error {
	return Instance.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("mfa_challenge_id", nil).Error
}
