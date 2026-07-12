package middlewares

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupMiddlewareConfig points the global config at a valid signing key and, by
// default, disables SMTP (so the verification branch is skipped).
func setupMiddlewareConfig(t *testing.T) {
	t.Helper()

	key, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	config.ConfigFile.PrivateKey = key
	config.ConfigFile.PoenskelistenName = "TestIssuer"
	config.ConfigFile.SMTPEnabled = false
	// Reset MFA enforcement so it doesn't leak between tests (and so tests without
	// a DB don't reach the enrollment lookup).
	config.ConfigFile.MFAEnforced = false

	// AuthFunction logs validation failures; set a discard logger so those calls
	// don't dereference a nil pointer or touch the filesystem.
	if logger.Log == nil {
		logger.Log = logrus.New()
		logger.Log.SetOutput(io.Discard)
	}
}

// setupMiddlewareDB spins up an isolated in-memory SQLite database and points
// database.Instance at it, mirroring the pattern used by the database package
// tests.
func setupMiddlewareDB(t *testing.T) {
	t.Helper()

	dbSQL, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { dbSQL.Close() })
	dbSQL.SetMaxOpenConns(1)

	instance, err := gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	if err := instance.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	database.Instance = instance
}

// createUser inserts a user with the given verified state and returns its ID.
func createUser(t *testing.T, verified bool) uuid.UUID {
	t.Helper()

	email := uuid.NewString() + "@example.com"
	password := "hashed-password"
	enabled := true
	user := models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     &email,
		Password:  &password,
		Enabled:   &enabled,
		Verified:  &verified,
	}
	user.ID = uuid.New()

	created, err := database.CreateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return created.ID
}

// newContext builds a gin context whose request carries the given Authorization
// header value (omitted when empty).
func newContext(authHeader string) *gin.Context {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	if authHeader != "" {
		ctx.Request.Header.Set("Authorization", authHeader)
	}
	return ctx
}

func tokenForUser(t *testing.T, userID uuid.UUID, admin bool) string {
	t.Helper()
	token, err := auth.GenerateJWT("Test", "User", "test@example.com", userID, admin, true)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return token
}

func TestAuthFunctionNoToken(t *testing.T) {
	setupMiddlewareConfig(t)

	success, _, status := AuthFunction(newContext(""), false)
	if success {
		t.Error("AuthFunction succeeded without a token, want failure")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", status, http.StatusUnauthorized)
	}
}

func TestAuthFunctionInvalidToken(t *testing.T) {
	setupMiddlewareConfig(t)

	success, _, status := AuthFunction(newContext("not-a-real-token"), false)
	if success {
		t.Error("AuthFunction succeeded with an invalid token, want failure")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", status, http.StatusUnauthorized)
	}
}

func TestAuthFunctionValidNonAdmin(t *testing.T) {
	setupMiddlewareConfig(t)

	token := tokenForUser(t, uuid.New(), false)
	success, errStr, status := AuthFunction(newContext(token), false)
	if !success {
		t.Errorf("AuthFunction failed for a valid token: %q (status %d)", errStr, status)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
}

func TestAuthFunctionAdminRequiredButNotAdmin(t *testing.T) {
	setupMiddlewareConfig(t)

	token := tokenForUser(t, uuid.New(), false)
	success, _, status := AuthFunction(newContext(token), true)
	if success {
		t.Error("AuthFunction succeeded for non-admin on admin route, want failure")
	}
	if status != http.StatusForbidden {
		t.Errorf("status = %d, want %d", status, http.StatusForbidden)
	}
}

func TestAuthFunctionAdminSuccess(t *testing.T) {
	setupMiddlewareConfig(t)

	token := tokenForUser(t, uuid.New(), true)
	success, _, status := AuthFunction(newContext(token), true)
	if !success {
		t.Error("AuthFunction failed for a valid admin token, want success")
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
}

func TestAuthFunctionSMTPVerifiedUser(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)
	config.ConfigFile.SMTPEnabled = true

	userID := createUser(t, true)
	token := tokenForUser(t, userID, false)

	success, errStr, status := AuthFunction(newContext(token), false)
	if !success {
		t.Errorf("AuthFunction failed for verified user with SMTP on: %q (status %d)", errStr, status)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
}

func TestAuthFunctionSMTPUnverifiedUser(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)
	config.ConfigFile.SMTPEnabled = true

	userID := createUser(t, false)
	token := tokenForUser(t, userID, false)

	success, errStr, status := AuthFunction(newContext(token), false)
	if success {
		t.Error("AuthFunction succeeded for unverified user with SMTP on, want failure")
	}
	if status != http.StatusForbidden {
		t.Errorf("status = %d, want %d", status, http.StatusForbidden)
	}
	if !strings.Contains(strings.ToLower(errStr), "verify") {
		t.Errorf("error = %q, want it to mention verification", errStr)
	}
}

func TestAuthFunctionMFAEnforcedNotEnrolledBlocks(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)
	config.ConfigFile.MFAEnforced = true

	userID := createUser(t, true)
	token := tokenForUser(t, userID, false)

	success, errStr, status := AuthFunction(newContext(token), false)
	if success {
		t.Error("AuthFunction succeeded for enforced-but-unenrolled user, want failure")
	}
	if status != http.StatusForbidden {
		t.Errorf("status = %d, want %d", status, http.StatusForbidden)
	}
	if !strings.Contains(errStr, "mfa_enrollment_required") {
		t.Errorf("error = %q, want it to signal mfa_enrollment_required", errStr)
	}
}

func TestAuthFunctionMFAEnforcedEnrolledPasses(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)
	config.ConfigFile.MFAEnforced = true

	userID := createUser(t, true)
	if err := database.ActivateUserMFA(userID); err != nil {
		t.Fatalf("failed to activate MFA: %v", err)
	}
	token := tokenForUser(t, userID, false)

	success, errStr, status := AuthFunction(newContext(token), false)
	if !success {
		t.Errorf("AuthFunction failed for enrolled user under enforcement: %q (status %d)", errStr, status)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
}

func TestAuthFunctionMFANotEnforcedIgnoresEnrollment(t *testing.T) {
	setupMiddlewareConfig(t)
	setupMiddlewareDB(t)
	// MFAEnforced stays false: an unenrolled user must not be blocked.

	userID := createUser(t, true)
	token := tokenForUser(t, userID, false)

	success, _, status := AuthFunction(newContext(token), false)
	if !success {
		t.Error("AuthFunction blocked an unenrolled user while enforcement was off")
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
}

func TestIsMFAEnrollmentExemptPath(t *testing.T) {
	exempt := []string{
		"/api/auth/users/mfa/enroll",
		"/api/auth/users/mfa/activate",
		"/api/auth/tokens/validate",
	}
	for _, p := range exempt {
		if !isMFAEnrollmentExemptPath(p) {
			t.Errorf("isMFAEnrollmentExemptPath(%q) = false, want true", p)
		}
	}

	notExempt := []string{
		"/api/auth/wishlists",
		"/api/auth/users/mfa/disable",
		"/api/admin/users",
		"",
	}
	for _, p := range notExempt {
		if isMFAEnrollmentExemptPath(p) {
			t.Errorf("isMFAEnrollmentExemptPath(%q) = true, want false", p)
		}
	}
}

func TestGetAuthUsername(t *testing.T) {
	setupMiddlewareConfig(t)

	if _, err := GetAuthUsername(""); err == nil {
		t.Error("GetAuthUsername(\"\") returned no error, want error")
	}

	userID := uuid.New()
	token := tokenForUser(t, userID, false)
	got, err := GetAuthUsername(token)
	if err != nil {
		t.Fatalf("GetAuthUsername returned error: %v", err)
	}
	if got != userID {
		t.Errorf("GetAuthUsername = %v, want %v", got, userID)
	}

	if _, err := GetAuthUsername("garbage"); err == nil {
		t.Error("GetAuthUsername(garbage) returned no error, want error")
	}
}

func TestGetTokenClaims(t *testing.T) {
	setupMiddlewareConfig(t)

	if _, err := GetTokenClaims(""); err == nil {
		t.Error("GetTokenClaims(\"\") returned no error, want error")
	}

	userID := uuid.New()
	token := tokenForUser(t, userID, true)
	claims, err := GetTokenClaims(token)
	if err != nil {
		t.Fatalf("GetTokenClaims returned error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}
	if !claims.Admin {
		t.Error("claims.Admin = false, want true")
	}
}
