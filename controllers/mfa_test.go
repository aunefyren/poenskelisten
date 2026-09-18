package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

// enableTOTPEncryption installs a private key so utilities.EncryptString/
// DecryptString (TOTP secret storage) and auth.GenerateMFAChallengeToken work.
func enableTOTPEncryption(t *testing.T) {
	t.Helper()
	key, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	config.ConfigFile.PrivateKey = key
}

func runJSONHandler(handler gin.HandlerFunc, method, path, body string, header map[string]string, params gin.Params) (int, map[string]interface{}) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
		ctx.Request = httptest.NewRequest(method, path, reader)
		ctx.Request.Header.Set("Content-Type", "application/json")
	} else {
		ctx.Request = httptest.NewRequest(method, path, nil)
	}
	for k, v := range header {
		ctx.Request.Header.Set(k, v)
	}
	if params != nil {
		ctx.Params = params
	}
	handler(ctx)

	var parsed map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &parsed)
	return w.Code, parsed
}

func TestEnrollMFARequiresAuth(t *testing.T) {
	setupControllersDB(t)

	code, _ := runJSONHandler(APIEnrollMFA, "POST", "/api/mfa/enroll", "", nil, nil)
	if code != 401 {
		t.Errorf("status = %d, want 401 without auth", code)
	}
}

func TestEnrollMFASuccess(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := createTestUser(t)

	code, body := runJSONHandler(APIEnrollMFA, "POST", "/api/mfa/enroll", "", map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, body)
	}
	if body["secret"] == nil || body["secret"] == "" {
		t.Error("expected a TOTP secret in the response")
	}
	if body["otpauth_url"] == nil || body["otpauth_url"] == "" {
		t.Error("expected an otpauth_url in the response")
	}
	if body["qr_code"] == nil || body["qr_code"] == "" {
		t.Error("expected a qr_code in the response")
	}
}

func TestEnrollMFANonLocalAuth(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := createTestUser(t)
	oidc := "oidc"
	user.AuthSource = &oidc
	user, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	code, _ := runJSONHandler(APIEnrollMFA, "POST", "/api/mfa/enroll", "", map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 for a non-local auth user", code)
	}
}

func TestEnrollMFAAlreadyEnabled(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := createTestUser(t)
	user.MFAEnabled = boolPtr(true)
	user, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	code, _ := runJSONHandler(APIEnrollMFA, "POST", "/api/mfa/enroll", "", map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 when MFA is already enabled", code)
	}
}

// enrollTestUser drives a real APIEnrollMFA call and returns the TOTP secret it
// generated, so activation/validation tests can produce live codes for it.
func enrollTestUser(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	code, body := runJSONHandler(APIEnrollMFA, "POST", "/api/mfa/enroll", "", map[string]string{
		"Authorization": authHeader(t, userID, false),
	}, nil)
	if code != 200 {
		t.Fatalf("enrollment failed: status = %d, body=%v", code, body)
	}
	secret, _ := body["secret"].(string)
	if secret == "" {
		t.Fatal("enrollment did not return a secret")
	}
	return secret
}

// mfaTestUserWithPassword inserts a user whose password is a real bcrypt hash of
// plaintext, so handlers that call user.CheckPassword (e.g. MFA disable) can be
// exercised on their success path.
func mfaTestUserWithPassword(t *testing.T, plaintext string) models.User {
	t.Helper()

	user := createTestUser(t)
	if err := user.HashPassword(plaintext); err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	updated, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to save hashed password: %v", err)
	}

	return updated
}

func totpCodeFor(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}
	return code
}

func TestActivateMFASuccessWithRecoveryCodes(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	config.ConfigFile.MFARecoveryCodesEnabled = true
	user := createTestUser(t)
	secret := enrollTestUser(t, user.ID)

	body := `{"code":"` + totpCodeFor(t, secret) + `"}`
	code, parsed := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", body, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, parsed)
	}
	codes, ok := parsed["recovery_codes"].([]interface{})
	if !ok || len(codes) == 0 {
		t.Errorf("expected recovery_codes in the response, got %v", parsed["recovery_codes"])
	}
}

func TestActivateMFASuccessWithoutRecoveryCodes(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	config.ConfigFile.MFARecoveryCodesEnabled = false
	user := createTestUser(t)
	secret := enrollTestUser(t, user.ID)

	body := `{"code":"` + totpCodeFor(t, secret) + `"}`
	code, parsed := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", body, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, parsed)
	}
}

func TestActivateMFAInvalidCode(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := createTestUser(t)
	enrollTestUser(t, user.ID)

	code, _ := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", `{"code":"000000"}`, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 for an invalid code", code)
	}
}

func TestActivateMFANotEnrolled(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := createTestUser(t)

	code, _ := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", `{"code":"123456"}`, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 when enrollment was never started", code)
	}
}

func TestActivateMFAAlreadyEnabled(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := createTestUser(t)
	user.MFAEnabled = boolPtr(true)
	user, err := database.UpdateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	code, _ := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", `{"code":"123456"}`, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 when MFA is already enabled", code)
	}
}

func TestActivateMFAMalformedJSON(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", `not-json`, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestActivateMFARequiresAuth(t *testing.T) {
	setupControllersDB(t)

	code, _ := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", `{"code":"123456"}`, nil, nil)
	if code != 401 {
		t.Errorf("status = %d, want 401 without auth", code)
	}
}

// activateTestUser enrolls and activates MFA for a user with a real password and
// recovery codes enabled, returning the TOTP secret and the plaintext recovery
// codes issued at activation.
func activateTestUser(t *testing.T, userID uuid.UUID) (secret string, recoveryCodes []string) {
	t.Helper()
	config.ConfigFile.MFARecoveryCodesEnabled = true
	secret = enrollTestUser(t, userID)

	body := `{"code":"` + totpCodeFor(t, secret) + `"}`
	code, parsed := runJSONHandler(APIActivateMFA, "POST", "/api/mfa/activate", body, map[string]string{
		"Authorization": authHeader(t, userID, false),
	}, nil)
	if code != 200 {
		t.Fatalf("activation failed: status = %d, body=%v", code, parsed)
	}
	rawCodes, _ := parsed["recovery_codes"].([]interface{})
	for _, c := range rawCodes {
		if s, ok := c.(string); ok {
			recoveryCodes = append(recoveryCodes, s)
		}
	}
	return secret, recoveryCodes
}

func TestDisableMFASuccess(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := mfaTestUserWithPassword(t, "correct horse battery staple")
	secret, _ := activateTestUser(t, user.ID)

	body := `{"password":"correct horse battery staple","code":"` + totpCodeFor(t, secret) + `"}`
	code, parsed := runJSONHandler(APIDisableMFA, "POST", "/api/mfa/disable", body, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, parsed)
	}
}

func TestDisableMFAWrongPassword(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := mfaTestUserWithPassword(t, "correct horse battery staple")
	secret, _ := activateTestUser(t, user.ID)

	body := `{"password":"wrong password","code":"` + totpCodeFor(t, secret) + `"}`
	code, _ := runJSONHandler(APIDisableMFA, "POST", "/api/mfa/disable", body, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 401 {
		t.Errorf("status = %d, want 401 for a wrong password", code)
	}
}

func TestDisableMFAInvalidCode(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := mfaTestUserWithPassword(t, "correct horse battery staple")
	activateTestUser(t, user.ID)

	body := `{"password":"correct horse battery staple","code":"000000"}`
	code, _ := runJSONHandler(APIDisableMFA, "POST", "/api/mfa/disable", body, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 for an invalid code", code)
	}
}

func TestDisableMFANotEnabled(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := mfaTestUserWithPassword(t, "correct horse battery staple")

	body := `{"password":"correct horse battery staple","code":"000000"}`
	code, _ := runJSONHandler(APIDisableMFA, "POST", "/api/mfa/disable", body, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 when MFA is not enabled", code)
	}
}

func TestDisableMFARequiresAuth(t *testing.T) {
	setupControllersDB(t)

	code, _ := runJSONHandler(APIDisableMFA, "POST", "/api/mfa/disable", `{"password":"x","code":"000000"}`, nil, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 without auth", code)
	}
}

func TestDisableMFAMalformedJSON(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := runJSONHandler(APIDisableMFA, "POST", "/api/mfa/disable", `not-json`, map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestValidateMFASuccess(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := mfaTestUserWithPassword(t, "correct horse battery staple")
	secret, _ := activateTestUser(t, user.ID)

	challenge, err := auth.GenerateMFAChallengeToken(user.ID)
	if err != nil {
		t.Fatalf("failed to generate challenge token: %v", err)
	}

	body := `{"mfa_token":"` + challenge + `","code":"` + totpCodeFor(t, secret) + `"}`
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/api/mfa/validate", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	APIValidateMFA(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	foundSSOCookie := false
	for _, c := range w.Result().Cookies() {
		if c.Name == ssoCookieName && c.Value != "" {
			foundSSOCookie = true
		}
	}
	if !foundSSOCookie {
		t.Error("expected an SSO cookie to be set on successful MFA validation")
	}
}

func TestValidateMFARecoveryCodeConsumed(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := mfaTestUserWithPassword(t, "correct horse battery staple")
	_, recoveryCodes := activateTestUser(t, user.ID)
	if len(recoveryCodes) == 0 {
		t.Fatal("expected at least one recovery code")
	}
	recoveryCode := recoveryCodes[0]

	challenge, err := auth.GenerateMFAChallengeToken(user.ID)
	if err != nil {
		t.Fatalf("failed to generate challenge token: %v", err)
	}
	body := `{"mfa_token":"` + challenge + `","code":"` + recoveryCode + `"}`
	code, parsed := runJSONHandler(APIValidateMFA, "POST", "/api/mfa/validate", body, nil, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200 for a fresh recovery code; body=%v", code, parsed)
	}

	// The same recovery code must not work twice.
	challenge2, err := auth.GenerateMFAChallengeToken(user.ID)
	if err != nil {
		t.Fatalf("failed to generate challenge token: %v", err)
	}
	body2 := `{"mfa_token":"` + challenge2 + `","code":"` + recoveryCode + `"}`
	code2, _ := runJSONHandler(APIValidateMFA, "POST", "/api/mfa/validate", body2, nil, nil)
	if code2 != 401 {
		t.Errorf("status = %d, want 401 for a reused recovery code", code2)
	}
}

func TestValidateMFAInvalidChallengeToken(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)

	code, _ := runJSONHandler(APIValidateMFA, "POST", "/api/mfa/validate", `{"mfa_token":"garbage","code":"123456"}`, nil, nil)
	if code != 401 {
		t.Errorf("status = %d, want 401 for an invalid challenge token", code)
	}
}

func TestValidateMFAWrongCode(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := mfaTestUserWithPassword(t, "correct horse battery staple")
	activateTestUser(t, user.ID)

	challenge, err := auth.GenerateMFAChallengeToken(user.ID)
	if err != nil {
		t.Fatalf("failed to generate challenge token: %v", err)
	}
	body := `{"mfa_token":"` + challenge + `","code":"000000"}`
	code, _ := runJSONHandler(APIValidateMFA, "POST", "/api/mfa/validate", body, nil, nil)
	if code != 401 {
		t.Errorf("status = %d, want 401 for a wrong code", code)
	}
}

func TestValidateMFANotEnabled(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	user := createTestUser(t)

	challenge, err := auth.GenerateMFAChallengeToken(user.ID)
	if err != nil {
		t.Fatalf("failed to generate challenge token: %v", err)
	}
	body := `{"mfa_token":"` + challenge + `","code":"123456"}`
	code, _ := runJSONHandler(APIValidateMFA, "POST", "/api/mfa/validate", body, nil, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 when MFA is not enabled", code)
	}
}

func TestValidateMFAMalformedJSON(t *testing.T) {
	setupControllersDB(t)

	code, _ := runJSONHandler(APIValidateMFA, "POST", "/api/mfa/validate", `not-json`, nil, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestAdminDeleteUserMFARequiresAuth(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	code, _ := runJSONHandler(APIAdminDeleteUserMFA, "DELETE", "/api/admin/users/"+user.ID.String()+"/mfa", "", nil, gin.Params{{Key: "user_id", Value: user.ID.String()}})
	if code != 400 {
		t.Errorf("status = %d, want 400 without auth", code)
	}
}

func TestAdminDeleteUserMFAInvalidUserID(t *testing.T) {
	setupControllersDB(t)
	admin := createTestUser(t)

	code, _ := runJSONHandler(APIAdminDeleteUserMFA, "DELETE", "/api/admin/users/not-a-uuid/mfa", "", map[string]string{
		"Authorization": authHeader(t, admin.ID, true),
	}, gin.Params{{Key: "user_id", Value: "not-a-uuid"}})
	if code != 400 {
		t.Errorf("status = %d, want 400 for a malformed user_id", code)
	}
}

func TestAdminDeleteUserMFASuccess(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	admin := createTestUser(t)
	target := createTestUser(t)
	target.MFAEnabled = boolPtr(true)
	target, err := database.UpdateUserInDB(target)
	if err != nil {
		t.Fatalf("failed to update target user: %v", err)
	}

	code, parsed := runJSONHandler(APIAdminDeleteUserMFA, "DELETE", "/api/admin/users/"+target.ID.String()+"/mfa", "", map[string]string{
		"Authorization": authHeader(t, admin.ID, true),
	}, gin.Params{{Key: "user_id", Value: target.ID.String()}})
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, parsed)
	}

	reloaded, err := database.GetAllUserInformation(target.ID)
	if err != nil {
		t.Fatalf("failed to reload target user: %v", err)
	}
	if reloaded.IsMFAEnabled() {
		t.Error("expected MFA to be disabled for the target user")
	}
}

func TestUpdateServerSettingsMalformedJSON(t *testing.T) {
	setupControllersDB(t)

	code, _ := runJSONHandler(APIUpdateServerSettings, "PUT", "/api/admin/settings", `not-json`, nil, nil)
	if code != 400 {
		t.Errorf("status = %d, want 400 for malformed JSON", code)
	}
}

func TestUpdateServerSettingsSuccess(t *testing.T) {
	setupControllersDB(t)

	// config.ConfigFile is a package-level global shared by every test in this
	// binary; restore the fields this test flips so later tests (e.g. the
	// OAuth authorize flow, which checks MFAEnforced) don't see them leaked.
	origEnforced := config.ConfigFile.MFAEnforced
	origRecovery := config.ConfigFile.MFARecoveryCodesEnabled
	t.Cleanup(func() {
		config.ConfigFile.MFAEnforced = origEnforced
		config.ConfigFile.MFARecoveryCodesEnabled = origRecovery
	})

	// config.SaveConfig writes to "./files/config.json" resolved to an absolute
	// path once, at package-init time (before any test runs) — go test's
	// working directory is always this package's directory, so that resolves
	// to controllers/files/config.json. Make sure that directory exists, and
	// clean up the file we cause it to write.
	if err := os.MkdirAll("files", 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}
	t.Cleanup(func() { os.Remove(filepath.Join("files", "config.json")) })

	code, parsed := runJSONHandler(APIUpdateServerSettings, "PUT", "/api/admin/settings", `{"mfa_enforced":true,"mfa_recovery_codes_enabled":true}`, nil, nil)
	if code != 200 {
		t.Fatalf("status = %d, want 200; body=%v", code, parsed)
	}
	if parsed["mfa_enforced"] != true {
		t.Errorf("mfa_enforced = %v, want true", parsed["mfa_enforced"])
	}
	if !config.ConfigFile.MFAEnforced {
		t.Error("expected config.ConfigFile.MFAEnforced to be updated")
	}
}

func TestEnrollMFAEncryptionFailure(t *testing.T) {
	setupControllersDB(t)
	orig := config.ConfigFile.PrivateKey
	config.ConfigFile.PrivateKey = ""
	t.Cleanup(func() { config.ConfigFile.PrivateKey = orig })
	user := createTestUser(t)

	code, _ := runJSONHandler(APIEnrollMFA, "POST", "/api/mfa/enroll", "", map[string]string{
		"Authorization": authHeader(t, user.ID, false),
	}, nil)
	if code != 500 {
		t.Errorf("status = %d, want 500 when the server has no private key configured to encrypt the TOTP secret", code)
	}
}

func TestDisableMFAUserNotFound(t *testing.T) {
	setupControllersDB(t)

	// A well-formed access token for a user ID that was never created in the DB.
	code, _ := runJSONHandler(APIDisableMFA, "POST", "/api/mfa/disable", `{"password":"x","code":"000000"}`, map[string]string{
		"Authorization": authHeader(t, uuid.New(), false),
	}, nil)
	if code != 500 {
		t.Errorf("status = %d, want 500 when the token's user doesn't exist", code)
	}
}

func TestValidateMFAUserNotFound(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)

	// A well-formed challenge token for a user ID that was never created in the DB.
	challenge, err := auth.GenerateMFAChallengeToken(uuid.New())
	if err != nil {
		t.Fatalf("failed to generate challenge token: %v", err)
	}
	body := `{"mfa_token":"` + challenge + `","code":"123456"}`
	code, _ := runJSONHandler(APIValidateMFA, "POST", "/api/mfa/validate", body, nil, nil)
	if code != 500 {
		t.Errorf("status = %d, want 500 when the challenge token's user doesn't exist", code)
	}
}

func TestUpdateServerSettingsSaveFailure(t *testing.T) {
	setupControllersDB(t)
	origEnforced := config.ConfigFile.MFAEnforced
	t.Cleanup(func() { config.ConfigFile.MFAEnforced = origEnforced })

	// No "files" directory present, so config.SaveConfig()'s WriteFile fails
	// and the handler takes its 500 path (see currency_test.go for the same
	// pattern: configFilePath is a fixed absolute path resolved once at
	// process start, so this is a plain directory-presence check, not a chdir).
	os.RemoveAll("files")

	code, _ := runJSONHandler(APIUpdateServerSettings, "PUT", "/api/admin/settings", `{"mfa_enforced":true}`, nil, nil)
	if code != 500 {
		t.Errorf("status = %d, want 500 when the config file can't be written", code)
	}
}

func TestVerifySecondFactorRecoveryDisabledByConfig(t *testing.T) {
	setupControllersDB(t)
	enableTOTPEncryption(t)
	config.ConfigFile.MFARecoveryCodesEnabled = false
	user := createTestUser(t)

	ok, err := verifySecondFactor(user, "NOTASIXDIGITCODE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected the recovery-code path to be rejected when recovery codes are disabled")
	}
}

func TestVerifySecondFactorNoTOTPSecret(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	_, err := verifySecondFactor(user, "123456")
	if err == nil {
		t.Error("expected an error when the user has no TOTP secret configured")
	}
}
