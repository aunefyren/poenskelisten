package database

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
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

// keepInstance restores the package-global Instance (and the DBLocation that
// Connect's SQLite branch reads from config) after a test that calls Connect,
// which replaces both.
func keepInstance(t *testing.T) {
	t.Helper()
	origInstance, origLocation := Instance, config.ConfigFile.DBLocation
	t.Cleanup(func() {
		if Instance != nil && Instance != origInstance {
			if sqlDB, err := Instance.DB(); err == nil {
				sqlDB.Close()
			}
		}
		Instance, config.ConfigFile.DBLocation = origInstance, origLocation
	})
}

func TestConnectSQLiteCreatesMissingFile(t *testing.T) {
	keepInstance(t)
	location := filepath.Join(t.TempDir(), "new.db")
	config.ConfigFile.DBLocation = location

	if err := Connect("SQLite", "UTC", "", "", "", 0, "", false, location); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if _, err := os.Stat(location); err != nil {
		t.Errorf("expected SQLite file to be created at %s: %v", location, err)
	}
	if err := Instance.Exec("SELECT 1").Error; err != nil {
		t.Errorf("connected Instance can't run a query: %v", err)
	}
}

func TestConnectSQLiteOpensExistingFile(t *testing.T) {
	keepInstance(t)
	location := filepath.Join(t.TempDir(), "existing.db")
	if err := os.WriteFile(location, nil, 0o600); err != nil {
		t.Fatalf("failed to create SQLite file: %v", err)
	}
	config.ConfigFile.DBLocation = location

	if err := Connect("sqlite", "UTC", "", "", "", 0, "", false, location); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}
	if err := Instance.Exec("CREATE TABLE t (id INTEGER)").Error; err != nil {
		t.Errorf("connected Instance can't write: %v", err)
	}
}

func TestConnectSQLiteUncreatableFileFails(t *testing.T) {
	keepInstance(t)
	config.ConfigFile.DBLocation = filepath.Join(t.TempDir(), "missing-dir", "db.sqlite")

	if err := Connect("sqlite", "UTC", "", "", "", 0, "", false, ""); err == nil {
		t.Fatal("expected an error when the SQLite file's directory doesn't exist")
	}
}

func TestConnectUnknownTypeFails(t *testing.T) {
	keepInstance(t)

	if err := Connect("oracle", "UTC", "", "", "", 0, "", false, ""); err == nil {
		t.Fatal("expected an error for an unrecognized database type")
	}
}

// Port 1 on loopback has nothing listening, so the driver's connect-time ping
// is refused immediately instead of waiting on a timeout.
func TestConnectPostgresUnreachableFails(t *testing.T) {
	keepInstance(t)

	for _, ssl := range []bool{false, true} {
		if err := Connect("postgres", "UTC", "user", "pass", "127.0.0.1", 1, "db", ssl, ""); err == nil {
			t.Errorf("expected an error connecting to an unreachable postgres (ssl=%v)", ssl)
		}
	}
}

func TestConnectMySQLUnreachableFails(t *testing.T) {
	keepInstance(t)

	if err := Connect("mysql", "UTC", "user", "pass", "127.0.0.1", 1, "db", false, ""); err == nil {
		t.Fatal("expected an error connecting to an unreachable mysql")
	}
}

func TestCreateTableUnreachableFails(t *testing.T) {
	if err := CreateTable("user", "pass", "127.0.0.1", 1, "db"); err == nil {
		t.Fatal("expected an error creating a database on an unreachable mysql")
	}
}

func TestClientQueriesFailOnClosedDB(t *testing.T) {
	runClosedDBCases(t, map[string]func() error{
		"GenerateRandomInvite": func() error {
			_, err := GenerateRandomInvite()
			return err
		},
		"GenerateRandomVerificationCodeForUser": func() error {
			_, err := GenerateRandomVerificationCodeForUser(uuid.New())
			return err
		},
		"VerifyUniqueUserEmail": func() error {
			_, err := VerifyUniqueUserEmail("x")
			return err
		},
		"VerifyUserHasVerificationCode": func() error {
			_, err := VerifyUserHasVerificationCode(uuid.New())
			return err
		},
		"VerifyUserVerificationCodeMatches": func() error {
			_, err := VerifyUserVerificationCodeMatches(uuid.New(), "x")
			return err
		},
		"VerifyUserIsVerified": func() error {
			_, err := VerifyUserIsVerified(uuid.New())
			return err
		},
		"VerifyUnusedUserInviteCode": func() error {
			_, err := VerifyUnusedUserInviteCode("x")
			return err
		},
		"SetUserVerification": func() error {
			return SetUserVerification(uuid.New(), true)
		},
		"DeleteGroup": func() error {
			return DeleteGroup(uuid.New())
		},
		"DeleteGroupMembership": func() error {
			return DeleteGroupMembership(uuid.New())
		},
		"DeleteWishlist": func() error {
			return DeleteWishlist(uuid.New())
		},
		"DeleteWishlistMembership": func() error {
			return DeleteWishlistMembership(uuid.New())
		},
	})
}

// A path through a regular file fails os.Stat with ENOTDIR rather than
// ErrNotExist, so Connect must refuse instead of trying to create the file.
func TestConnectSQLiteUnstatableLocationFails(t *testing.T) {
	keepInstance(t)
	notADir := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(notADir, nil, 0o600); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	config.ConfigFile.DBLocation = filepath.Join(notADir, "db.sqlite")

	err := Connect("sqlite", "UTC", "", "", "", 0, "", false, "")
	if err == nil || err.Error() != "failed to verify SQLite file" {
		t.Fatalf("Connect error = %v, want \"failed to verify SQLite file\"", err)
	}
}

// A directory passes the existence check but can't be opened as a database,
// so the failure surfaces from gorm.Open's initial version query.
func TestConnectSQLiteDirectoryLocationFails(t *testing.T) {
	keepInstance(t)
	config.ConfigFile.DBLocation = t.TempDir()

	err := Connect("sqlite", "UTC", "", "", "", 0, "", false, "")
	if err == nil || err.Error() != "failed to connect to database" {
		t.Fatalf("Connect error = %v, want \"failed to connect to database\"", err)
	}
}

func TestMigratePanicsWhenSeedingFails(t *testing.T) {
	setupTestDB(t)
	injectFault(t, "create", "o_auth_clients", 0)

	defer func() {
		r := recover()
		if err, ok := r.(error); !ok || !errors.Is(err, errInjected) {
			t.Errorf("Migrate() recovered %v, want a panic carrying the seeding error", r)
		}
	}()
	Migrate()
}

func TestClientMissingRowsReturnErrors(t *testing.T) {
	setupTestDB(t)
	missing := uuid.New()

	if _, err := GenerateRandomVerificationCodeForUser(missing); err == nil {
		t.Error("GenerateRandomVerificationCodeForUser: expected error for an unknown user")
	}
	if _, err := VerifyUserHasVerificationCode(missing); err == nil {
		t.Error("VerifyUserHasVerificationCode: expected error for an unknown user")
	}
	if _, err := VerifyUserIsVerified(missing); err == nil {
		t.Error("VerifyUserIsVerified: expected error for an unknown user")
	}
	if err := SetUserVerification(missing, true); err == nil {
		t.Error("SetUserVerification: expected error for an unknown user")
	}
	if err := DeleteGroup(missing); err == nil {
		t.Error("DeleteGroup: expected error for an unknown group")
	}
	if err := DeleteWishlist(missing); err == nil {
		t.Error("DeleteWishlist: expected error for an unknown wishlist")
	}
	if err := DeleteWishlistMembership(missing); err == nil {
		t.Error("DeleteWishlistMembership: expected error for an unknown membership")
	}
}
