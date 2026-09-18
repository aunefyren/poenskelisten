package database

import (
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	if logger.Log == nil {
		logger.Log = logrus.New()
	}
}

func TestUserVerificationFlow(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	// A freshly created user has no verification code.
	if has, err := VerifyUserHasVerificationCode(user.ID); err != nil || has {
		t.Fatalf("expected no verification code initially (has=%v err=%v)", has, err)
	}

	code, err := GenerateRandomVerificationCodeForUser(user.ID)
	if err != nil {
		t.Fatalf("GenerateRandomVerificationCodeForUser returned error: %v", err)
	}
	if code == "" {
		t.Fatalf("expected a non-empty verification code")
	}

	if has, err := VerifyUserHasVerificationCode(user.ID); err != nil || !has {
		t.Fatalf("expected a verification code to be set (has=%v err=%v)", has, err)
	}

	// The right code matches; a wrong one does not.
	if ok, err := VerifyUserVerificationCodeMatches(user.ID, code); err != nil || !ok {
		t.Fatalf("expected matching code to verify (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyUserVerificationCodeMatches(user.ID, "WRONG"); err != nil || ok {
		t.Fatalf("expected wrong code to fail (ok=%v err=%v)", ok, err)
	}
}

func TestSetUserVerification(t *testing.T) {
	setupTestDB(t)

	user := createTestUser(t)

	// createTestUser marks the user verified; flip it off and confirm.
	if err := SetUserVerification(user.ID, false); err != nil {
		t.Fatalf("SetUserVerification returned error: %v", err)
	}
	if verified, err := VerifyUserIsVerified(user.ID); err != nil || verified {
		t.Fatalf("expected user to be unverified (verified=%v err=%v)", verified, err)
	}

	if err := SetUserVerification(user.ID, true); err != nil {
		t.Fatalf("SetUserVerification returned error: %v", err)
	}
	if verified, err := VerifyUserIsVerified(user.ID); err != nil || !verified {
		t.Fatalf("expected user to be verified (verified=%v err=%v)", verified, err)
	}
}

func TestMigrate(t *testing.T) {
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
	Instance = instance

	// Migrate() panics on failure; a clean unmigrated DB should succeed.
	Migrate()

	if !Instance.Migrator().HasTable(&models.User{}) {
		t.Error("expected the users table to exist after Migrate()")
	}
	if !Instance.Migrator().HasTable(&models.OAuthClient{}) {
		t.Error("expected the o_auth_clients table to exist after Migrate()")
	}

	clients, err := GetAllOAuthClients()
	if err != nil {
		t.Fatalf("GetAllOAuthClients error: %v", err)
	}
	found := false
	for _, c := range clients {
		if c.IsFirstParty {
			found = true
		}
	}
	if !found {
		t.Error("expected Migrate() to have seeded the first-party client")
	}
}

func TestMigratePanicsOnFailure(t *testing.T) {
	// A closed connection makes every AutoMigrate call fail.
	dbSQL, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	instance, err := gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	dbSQL.Close()
	Instance = instance

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected Migrate() to panic when the underlying connection is closed")
		}
	}()
	Migrate()
}
