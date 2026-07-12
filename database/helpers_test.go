package database

import (
	"aunefyren/poenskelisten/models"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// allModels lists every entity the schema is built from, kept in one place so
// each test gets the full set of tables and the list can't drift from Migrate().
func allModels() []interface{} {
	return []interface{}{
		&models.User{},
		&models.Invite{},
		&models.Group{},
		&models.GroupMembership{},
		&models.Wishlist{},
		&models.WishlistMembership{},
		&models.WishlistCollaborator{},
		&models.WishCategory{},
		&models.Wish{},
		&models.WishClaim{},
		&models.News{},
		&models.MFARecoveryCode{},
	}
}

// setupTestDB spins up an isolated in-memory SQLite database (CGO-free modernc
// driver) and points the package-global Instance at it. Each call gets a fresh
// schema so tests don't leak state into one another.
func setupTestDB(t *testing.T) {
	t.Helper()

	dbSQL, err := sql.Open("sqlite", "file:"+uuid.NewString()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { dbSQL.Close() })

	// A shared-cache in-memory DB lives only while a connection is held open, so
	// pin the pool to a single connection for the duration of the test.
	dbSQL.SetMaxOpenConns(1)

	instance, err := gorm.Open(sqlite.Dialector{Conn: dbSQL}, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	if err := instance.AutoMigrate(allModels()...); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	Instance = instance
}

func boolPtr(b bool) *bool { return &b }

func strPtr(s string) *string { return &s }

// createTestUser inserts an enabled, verified user with a unique e-mail.
func createTestUser(t *testing.T) models.User {
	t.Helper()

	email := uuid.NewString() + "@example.com"
	password := "hashed-password"

	user := models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     &email,
		Password:  &password,
		Enabled:   boolPtr(true),
		Verified:  boolPtr(true),
	}
	user.ID = uuid.New()

	created, err := CreateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return created
}

// createTestWishlist inserts an enabled wishlist owned by ownerID.
func createTestWishlist(t *testing.T, ownerID uuid.UUID) models.Wishlist {
	t.Helper()

	now := time.Now()
	wishlist := models.Wishlist{
		Name:    "Wishlist " + uuid.NewString(),
		Enabled: true,
		OwnerID: ownerID,
		Date:    &now,
	}
	wishlist.ID = uuid.New()

	created, err := CreateWishlistInDB(wishlist)
	if err != nil {
		t.Fatalf("failed to create wishlist: %v", err)
	}

	return created
}

// addWishlistMembership links a group to a wishlist and returns the membership.
func addWishlistMembership(t *testing.T, wishlistID, groupID uuid.UUID) models.WishlistMembership {
	t.Helper()

	membership := models.WishlistMembership{
		GroupID:    groupID,
		WishlistID: wishlistID,
		Enabled:    true,
	}
	membership.ID = uuid.New()

	created, err := CreateWishlistMembershipInDB(membership)
	if err != nil {
		t.Fatalf("failed to create wishlist membership: %v", err)
	}

	return created
}

// createTestGroup inserts an enabled group owned by ownerID.
func createTestGroup(t *testing.T, ownerID uuid.UUID) models.Group {
	t.Helper()

	group := models.Group{
		Name:    "Group " + uuid.NewString(),
		Enabled: true,
		OwnerID: ownerID,
	}
	group.ID = uuid.New()

	created, err := CreateGroupInDB(group)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	return created
}
