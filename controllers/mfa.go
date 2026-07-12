package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/middlewares"
	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// verifySecondFactor checks a user-supplied code against the user's TOTP secret
// or their unused recovery codes. A matching recovery code is consumed. It
// returns true when the code is accepted.
//
// TOTP codes (6 digits) and recovery codes (16 base32 chars) are disjoint, so we
// branch strictly on the input shape. This avoids running a bcrypt comparison per
// recovery code on every TOTP attempt.
func verifySecondFactor(user models.User, code string) (bool, error) {
	if utilities.LooksLikeTOTPCode(code) {
		if user.MFASecret == nil {
			return false, utilities.ErrNoTOTPSecret
		}
		secret, err := utilities.DecryptString(*user.MFASecret)
		if err != nil {
			return false, err
		}
		return utilities.ValidateTOTPCode(secret, code), nil
	}

	// Recovery-code path: only available when the administrator has enabled
	// recovery codes. Turning the option off immediately removes the fallback,
	// even for users who already hold codes.
	if !config.ConfigFile.MFARecoveryCodesEnabled {
		return false, nil
	}

	// Match against stored hashes and consume on success.
	activeCodes, err := database.GetActiveRecoveryCodes(user.ID)
	if err != nil {
		return false, err
	}
	for _, recoveryCode := range activeCodes {
		if utilities.CheckRecoveryCode(recoveryCode.CodeHash, code) {
			if err := database.MarkRecoveryCodeUsed(recoveryCode.ID); err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

// APIEnrollMFA begins TOTP enrollment: it generates a secret, stores it encrypted
// as pending, and returns the secret plus otpauth URL for the client to render as
// a QR code. Enrollment is confirmed by APIActivateMFA.
func APIEnrollMFA(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Error("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	if !user.IsLocalAuth() {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Multi-factor authentication is managed by your identity provider."})
		context.Abort()
		return
	}

	if user.IsMFAEnabled() {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Multi-factor authentication is already enabled."})
		context.Abort()
		return
	}

	accountName := userID.String()
	if user.Email != nil && *user.Email != "" {
		accountName = *user.Email
	}

	secret, otpauthURL, qrCode, err := utilities.GenerateTOTPSecret(accountName)
	if err != nil {
		logger.Log.Error("Failed to generate TOTP secret. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate MFA secret."})
		context.Abort()
		return
	}

	encryptedSecret, err := utilities.EncryptString(secret)
	if err != nil {
		logger.Log.Error("Failed to encrypt TOTP secret. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store MFA secret."})
		context.Abort()
		return
	}

	if err := database.SetUserPendingMFASecret(userID, encryptedSecret); err != nil {
		logger.Log.Error("Failed to store pending MFA secret. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store MFA secret."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Enrollment started.", "secret": secret, "otpauth_url": otpauthURL, "qr_code": qrCode})
}

// APIActivateMFA confirms enrollment: it validates the first TOTP code, generates
// recovery codes, and switches MFA on. The recovery codes are returned once, in
// plaintext, for the user to store.
func APIActivateMFA(context *gin.Context) {
	var request models.MFAActivateRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Error("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	if user.IsMFAEnabled() {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Multi-factor authentication is already enabled."})
		context.Abort()
		return
	}

	if user.MFASecret == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Start enrollment before activating."})
		context.Abort()
		return
	}

	secret, err := utilities.DecryptString(*user.MFASecret)
	if err != nil {
		logger.Log.Error("Failed to decrypt TOTP secret. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify code."})
		context.Abort()
		return
	}

	if !utilities.ValidateTOTPCode(secret, request.Code) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code. Please try again."})
		context.Abort()
		return
	}

	// Always clear any leftover recovery codes from a previous partial enrollment.
	if err := database.ClearUserRecoveryCodes(userID); err != nil {
		logger.Log.Error("Failed to clear old recovery codes. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store recovery codes."})
		context.Abort()
		return
	}

	// Recovery codes are only issued when the administrator has enabled them. When
	// disabled, a locked-out user must have their MFA removed by an admin instead.
	var plainCodes []string
	if config.ConfigFile.MFARecoveryCodesEnabled {
		plainCodes, err = utilities.GenerateRecoveryCodes(utilities.RecoveryCodeCount)
		if err != nil {
			logger.Log.Error("Failed to generate recovery codes. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate recovery codes."})
			context.Abort()
			return
		}

		hashes := make([]string, 0, len(plainCodes))
		for _, plain := range plainCodes {
			hash, err := utilities.HashRecoveryCode(plain)
			if err != nil {
				logger.Log.Error("Failed to hash recovery code. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate recovery codes."})
				context.Abort()
				return
			}
			hashes = append(hashes, hash)
		}

		if err := database.StoreRecoveryCodes(userID, hashes); err != nil {
			logger.Log.Error("Failed to store recovery codes. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store recovery codes."})
			context.Abort()
			return
		}
	}

	if err := database.ActivateUserMFA(userID); err != nil {
		logger.Log.Error("Failed to activate MFA. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable MFA."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Multi-factor authentication enabled.", "recovery_codes": plainCodes})
}

// APIDisableMFA lets a user turn MFA off. It requires the current password and a
// valid second factor (TOTP or recovery code) to prevent an attacker with a
// hijacked session from stripping MFA.
func APIDisableMFA(context *gin.Context) {
	var request models.MFADisableRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Error("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	if !user.IsMFAEnabled() {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Multi-factor authentication is not enabled."})
		context.Abort()
		return
	}

	if err := user.CheckPassword(request.Password); err != nil {
		logger.Log.Error("Invalid credentials during MFA disable.")
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	verified, err := verifySecondFactor(user, request.Code)
	if err != nil {
		logger.Log.Error("Failed to verify second factor. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify code."})
		context.Abort()
		return
	}
	if !verified {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code. Please try again."})
		context.Abort()
		return
	}

	if err := database.DisableUserMFA(userID); err != nil {
		logger.Log.Error("Failed to disable MFA. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable MFA."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Multi-factor authentication disabled."})
}

// APIValidateMFA is the second login step. It accepts the challenge token issued
// after a correct password plus a second factor, and returns a full session token
// on success.
func APIValidateMFA(context *gin.Context) {
	var request models.MFAValidateRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	userID, err := auth.ValidateMFAChallengeToken(request.MFAToken)
	if err != nil {
		logger.Log.Error("Failed to validate MFA challenge token. Error: " + err.Error())
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Your login attempt expired. Please log in again."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Error("Failed to get user during MFA validation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	if !user.IsMFAEnabled() {
		// MFA was turned off between the two steps; nothing more to verify here.
		context.JSON(http.StatusBadRequest, gin.H{"error": "Multi-factor authentication is not enabled."})
		context.Abort()
		return
	}

	verified, err := verifySecondFactor(user, request.Code)
	if err != nil {
		logger.Log.Error("Failed to verify second factor. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify code."})
		context.Abort()
		return
	}
	if !verified {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code. Please try again."})
		context.Abort()
		return
	}

	tokenString, err := auth.GenerateJWT(user.FirstName, user.LastName, *user.Email, user.ID, user.Admin, *user.Verified)
	if err != nil {
		logger.Log.Error("Failed to generate token after MFA. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid credentials."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"token": tokenString, "message": "Logged in!"})
}

// APIAdminDeleteUserMFA lets an administrator strip MFA from a user, e.g. when the
// user has lost their authenticator and recovery codes.
func APIAdminDeleteUserMFA(context *gin.Context) {
	userIDString := context.Param("user_id")

	adminID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	targetID, err := uuid.Parse(userIDString)
	if err != nil {
		logger.Log.Error("Failed to parse user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse user ID."})
		context.Abort()
		return
	}

	if err := database.DisableUserMFA(targetID); err != nil {
		logger.Log.Error("Failed to delete user MFA. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user MFA."})
		context.Abort()
		return
	}

	logger.Log.Info("Admin " + adminID.String() + " removed MFA for user " + targetID.String() + ".")

	context.JSON(http.StatusOK, gin.H{"message": "Multi-factor authentication removed for user."})
}

// APIUpdateServerSettings updates admin-editable runtime settings (currently the
// MFA-enforcement flag) and persists them to the config file.
func APIUpdateServerSettings(context *gin.Context) {
	var request models.ServerSettingsRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	config.ConfigFile.MFAEnforced = request.MFAEnforced
	config.ConfigFile.MFARecoveryCodesEnabled = request.MFARecoveryCodesEnabled

	if err := config.SaveConfig(); err != nil {
		logger.Log.Error("Failed to save server settings. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save server settings."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Server settings updated.", "mfa_enforced": config.ConfigFile.MFAEnforced, "mfa_recovery_codes_enabled": config.ConfigFile.MFARecoveryCodesEnabled})
}
