package database

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

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

// snapshotTables dumps every row of every table, deleted ones included, so a
// test can check that nothing at all was written.
func snapshotTables(t *testing.T) string {
	t.Helper()
	dump := map[string][]map[string]interface{}{}
	for _, model := range allModels() {
		statement := &gorm.Statement{DB: Instance}
		if err := statement.Parse(model); err != nil {
			t.Fatalf("failed to parse model: %v", err)
		}
		rows := []map[string]interface{}{}
		if err := Instance.Table(statement.Schema.Table).Order("id").Find(&rows).Error; err != nil {
			t.Fatalf("failed to read %s: %v", statement.Schema.Table, err)
		}
		dump[statement.Schema.Table] = rows
	}
	raw, err := json.Marshal(dump)
	if err != nil {
		t.Fatalf("failed to encode snapshot: %v", err)
	}
	return string(raw)
}

// GORM leaves zero values out of struct conditions. With the nil UUID or an
// empty string, every lookup must find nothing and every write change nothing.
func TestNilIDMatchesNothing(t *testing.T) {
	setupTestDB(t)
	nilID := uuid.Nil

	// The caller owns a wishlist and a group, so ownership and membership
	// checks have records of theirs to match.
	caller := createTestUser(t)
	callerWishlist := createTestWishlist(t, caller.ID)
	callerGroup := createTestGroup(t, caller.ID)
	callerWish := models.Wish{Name: "Caller wish", Enabled: true, OwnerID: caller.ID, WishlistID: callerWishlist.ID}
	callerWish.ID = uuid.New()
	if _, err := CreateWishInDB(callerWish); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateGroupMembershipInDB(models.GroupMembership{GormModel: models.GormModel{ID: uuid.New()}, GroupID: callerGroup.ID, MemberID: caller.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	other := createTestUser(t)
	claimer := createTestUser(t)
	for _, user := range []models.User{other, claimer} {
		if err := SetUserPendingMFASecret(user.ID, "secret"); err != nil {
			t.Fatal(err)
		}
		if err := ActivateUserMFA(user.ID); err != nil {
			t.Fatal(err)
		}
		if err := StoreRecoveryCodes(user.ID, []string{"hash"}); err != nil {
			t.Fatal(err)
		}
		if _, err := CreateSession(user.ID, "refresh-"+user.ID.String(), "agent", "127.0.0.1"); err != nil {
			t.Fatal(err)
		}
		if err := UpsertConsent(user.ID, "client-"+user.ID.String(), []string{"openid"}); err != nil {
			t.Fatal(err)
		}
	}
	otherGroup := createTestGroup(t, other.ID)
	if _, err := CreateGroupMembershipInDB(models.GroupMembership{GormModel: models.GormModel{ID: uuid.New()}, GroupID: otherGroup.ID, MemberID: claimer.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	otherWishlist := createTestWishlist(t, other.ID)
	addWishlistMembership(t, otherWishlist.ID, otherGroup.ID)
	publicWishlist := createTestWishlist(t, other.ID)
	publicWishlist.Public = boolPtr(true)
	publicWishlist.PublicHash = uuid.New()
	if _, err := UpdateWishlistInDB(publicWishlist); err != nil {
		t.Fatal(err)
	}
	otherWish := models.Wish{Name: "Other wish", Enabled: true, OwnerID: other.ID, WishlistID: otherWishlist.ID}
	otherWish.ID = uuid.New()
	if _, err := CreateWishInDB(otherWish); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateWishClaimInDB(models.WishClaim{GormModel: models.GormModel{ID: uuid.New()}, WishID: otherWish.ID, UserID: claimer.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := CreateWishCategoryInDB(models.WishCategory{GormModel: models.GormModel{ID: uuid.New()}, Name: "Other category", WishlistID: otherWishlist.ID, OwnerID: other.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := CreateWishlistCollaboratorInDB(models.WishlistCollaborator{GormModel: models.GormModel{ID: uuid.New()}, WishlistID: otherWishlist.ID, UserID: claimer.ID, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateRandomInvite(); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateNewsPostInDB(models.News{GormModel: models.GormModel{ID: uuid.New()}, Title: "News", Enabled: true, Date: time.Now()}); err != nil {
		t.Fatal(err)
	}

	before := snapshotTables(t)

	t.Run("lookups", func(t *testing.T) {
		mustNot := func(name string, found bool, err error) {
			t.Helper()
			if err != nil || found {
				t.Errorf("%s: found=%v err=%v, want nothing found and no error", name, found, err)
			}
		}
		mustFail := func(name string, err error) {
			t.Helper()
			if err == nil {
				t.Errorf("%s: no error, want not found", name)
			}
		}

		found, _, err := GetWishlistByWishlistID(nilID)
		mustNot("GetWishlistByWishlistID", found, err)
		_, err = GetWishlist(nilID)
		mustFail("GetWishlist", err)
		_, err = GetWishlistOwner(nilID)
		mustFail("GetWishlistOwner", err)
		owns, err := VerifyUserOwnershipToWishlist(caller.ID, nilID)
		mustNot("VerifyUserOwnershipToWishlist(caller, nil)", owns, err)
		owns, err = VerifyUserOwnershipToWishlist(nilID, otherWishlist.ID)
		mustNot("VerifyUserOwnershipToWishlist(nil, wishlist)", owns, err)
		member, err := VerifyUserMembershipToGroupMembershipToWishlist(caller.ID, nilID)
		mustNot("VerifyUserMembershipToGroupMembershipToWishlist", member, err)
		collaborates, err := VerifyWishlistCollaboratorToWishlist(nilID, claimer.ID)
		mustNot("VerifyWishlistCollaboratorToWishlist", collaborates, err)
		found, wishes, err := GetWishesFromWishlist(nilID)
		mustNot("GetWishesFromWishlist", found || len(wishes) > 0, err)
		wishIDs, err := GetWishIDsFromWishlist(nilID)
		mustNot("GetWishIDsFromWishlist", len(wishIDs) > 0, err)
		wish, err := GetWishByWishID(nilID)
		mustNot("GetWishByWishID", wish != nil, err)
		wishlistID, err := GetWishlistIDFromWish(nilID)
		mustNot("GetWishlistIDFromWish", wishlistID != nil, err)
		owns, err = VerifyUserOwnershipToWish(caller.ID, nilID)
		mustNot("VerifyUserOwnershipToWish", owns, err)
		claims, err := GetWishClaimFromWish(nilID)
		mustNot("GetWishClaimFromWish", len(claims) > 0, err)
		claimed, err := VerifyWishIsClaimed(nilID)
		mustNot("VerifyWishIsClaimed", claimed, err)
		found, _, err = GetPublicWishListByWishlistHash(nilID)
		mustNot("GetPublicWishListByWishlistHash", found, err)
		wishlists, err := GetOwnedWishlists(nilID)
		mustNot("GetOwnedWishlists", len(wishlists) > 0, err)
		wishlists, err = GetWishlistsByUserIDThroughWishlistMemberships(nilID)
		mustNot("GetWishlistsByUserIDThroughWishlistMemberships", len(wishlists) > 0, err)
		wishlists, err = GetWishlistsByUserIDThroughWishlistCollaborations(nilID)
		mustNot("GetWishlistsByUserIDThroughWishlistCollaborations", len(wishlists) > 0, err)
		collaborators, err := GetWishlistCollaboratorsFromWishlist(nilID)
		mustNot("GetWishlistCollaboratorsFromWishlist", len(collaborators) > 0, err)
		found, _, err = GetMembershipIDForGroupToWishlist(nilID, otherGroup.ID)
		mustNot("GetMembershipIDForGroupToWishlist", found, err)
		shared, err := VerifyGroupMembershipToWishlist(nilID, otherGroup.ID)
		mustNot("VerifyGroupMembershipToWishlist", shared, err)
		unique, err := VerifyUniqueWishNameInWishlist("", nilID)
		mustNot("VerifyUniqueWishNameInWishlist (taken)", !unique, err)

		_, err = GetGroupInformation(nilID)
		mustFail("GetGroupInformation", err)
		owns, err = VerifyUserOwnershipToGroup(caller.ID, nilID)
		mustNot("VerifyUserOwnershipToGroup", owns, err)
		member, err = VerifyUserMembershipToGroup(caller.ID, nilID)
		mustNot("VerifyUserMembershipToGroup", member, err)
		memberships, err := GetGroupMembershipsFromGroup(nilID)
		mustNot("GetGroupMembershipsFromGroup", len(memberships) > 0, err)
		_, err = GetGroupUsingGroupIDAndUserIDAsOwner(caller.ID, nilID)
		mustFail("GetGroupUsingGroupIDAndUserIDAsOwner", err)
		_, err = GetGroupUsingGroupIDAndMembershipUsingUserID(caller.ID, nilID)
		mustFail("GetGroupUsingGroupIDAndMembershipUsingUserID", err)
		_, err = GetGroupMembershipByGroupIDAndMemberID(nilID, claimer.ID)
		mustFail("GetGroupMembershipByGroupIDAndMemberID", err)
		groups, err := GetGroupsAUserIsAMemberOf(nilID)
		mustNot("GetGroupsAUserIsAMemberOf", len(groups) > 0, err)
		wishlists, err = GetWishlistsFromGroup(nilID)
		mustNot("GetWishlistsFromGroup", len(wishlists) > 0, err)

		_, err = GetUserInformation(nilID)
		mustFail("GetUserInformation", err)
		_, err = GetAllUserInformationAnyState(nilID)
		mustFail("GetAllUserInformationAnyState", err)
		_, _, err = GetUserMFAEnrollmentState(nilID)
		mustFail("GetUserMFAEnrollmentState", err)
		codes, err := GetActiveRecoveryCodes(nilID)
		mustNot("GetActiveRecoveryCodes", len(codes) > 0, err)
		consents, err := GetUserConsents(nilID)
		mustNot("GetUserConsents", len(consents) > 0, err)
		category, err := GetWishCategoryByID(nilID)
		mustNot("GetWishCategoryByID", category != nil, err)
		categories, err := GetWishCategoriesFromWishlist(nilID)
		mustNot("GetWishCategoriesFromWishlist", len(categories) > 0, err)
		_, err = GetInviteByID(nilID)
		mustFail("GetInviteByID", err)
		_, err = GetNewsPostByNewsID(nilID)
		mustFail("GetNewsPostByNewsID", err)
		valid, err := VerifyUnusedUserInviteCode("")
		mustNot("VerifyUnusedUserInviteCode", valid, err)
		_, found, err = GetOAuthClient("")
		mustNot("GetOAuthClient", found, err)
	})

	t.Run("writes", func(t *testing.T) {
		// Refusing with an error or quietly doing nothing are both fine;
		// what matters is the snapshot below.
		_ = UpdateGroupValuesByID(nilID, "Renamed group", "Renamed")
		_ = DisableUserMFA(nilID)
		_ = ClearUserRecoveryCodes(nilID)
		_ = SetUserPendingMFASecret(nilID, "secret")
		_ = ActivateUserMFA(nilID)
		_ = MarkRecoveryCodeUsed(nilID)
		_ = RevokeAllUserSessions(nilID)
		_ = RevokeSessionByRefreshHash("")
		_ = RevokeConsent(other.ID, "")
		_ = SetUserSessionsInvalidatedAt(nilID, time.Now())
		_ = SetUserVerification(nilID, false)
		_, _ = GenerateRandomVerificationCodeForUser(nilID)
		_, _ = GenerateRandomResetCodeForUser(nilID, true)
		_ = DeleteGroup(nilID)
		_ = DeleteGroupMembership(nilID)
		_ = DeleteWishlist(nilID)
		_ = DeleteWishlistMembership(nilID)
		_ = DeleteWishlistCollaboratorByWishlistCollaboratorID(nilID)
		_ = DeleteWishClaimByUserAndWish(nilID, claimer.ID)
		_ = DeleteWishClaimByUserAndWish(otherWish.ID, nilID)
		_ = DeleteWishCategory(nilID)
		_ = DeleteInviteByID(nilID)
		_ = DeleteNewsPost(nilID)
		_ = DisableOAuthClient("")

		if after := snapshotTables(t); after != before {
			t.Errorf("writes for the nil UUID changed the database:\nbefore %s\nafter  %s", before, after)
		}
	})
}

// Struct conditions drop zero-valued fields, so queries name their columns.
func TestQueriesNameTheirColumns(t *testing.T) {
	structCondition := regexp.MustCompile(`\b(Where|Not|Or|First|Find|Take|Last|Delete)\(\s*&?models\.[A-Za-z]+\{[^}]`)
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no database sources found: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range structCondition.FindAllString(string(raw), -1) {
			t.Errorf("%s uses a struct condition: %s", file, match)
		}
	}
}
