package controllers

import (
	"aunefyren/poenskelisten/auth"
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// TestMain drops models.BcryptCost to bcrypt's minimum for the whole
// package's test run. Several tests (newUserWithPassword,
// createTestUserWithPassword, and the handlers they exercise via
// CheckPassword) hash or verify a real password; at the production cost
// that's deliberately slow, and under `go test -race` it's roughly another
// order of magnitude slower still (measured: ~1s per op at the production
// cost without -race, ~12s with it) - these tests only need the round-trip
// to work, not production-strength hardness.
func TestMain(m *testing.M) {
	models.BcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}

// allControllersTestModels is the full schema, kept in one place so every
// controller test can migrate everything it might touch (a handler under test
// often reaches through several related tables) without each file having to
// enumerate them.
func allControllersTestModels() []interface{} {
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
		&models.Session{},
		&models.OAuthClient{},
		&models.AuthorizationCode{},
		&models.OAuthConsent{},
	}
}

// setupControllersDB points database.Instance at a fresh in-memory SQLite DB.
// With no arguments it migrates the full schema (allControllersTestModels);
// pass specific models only if a test wants a deliberately incomplete schema
// (e.g. to exercise a "table missing" error path).
func setupControllersDB(t *testing.T, tables ...interface{}) {
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
	if len(tables) == 0 {
		tables = allControllersTestModels()
	}
	if err := instance.AutoMigrate(tables...); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	database.Instance = instance
}

// createTestUser inserts an enabled, verified user with a unique e-mail.
// Callers must have migrated &models.User{} via setupControllersDB.
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

	created, err := database.CreateUserInDB(user)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
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

	created, err := database.CreateGroupInDB(group)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	return created
}

// addGroupMembership links a user to a group and returns the membership.
func addGroupMembership(t *testing.T, groupID, memberID uuid.UUID) models.GroupMembership {
	t.Helper()

	membership := models.GroupMembership{
		GroupID:  groupID,
		MemberID: memberID,
		Enabled:  true,
	}
	membership.ID = uuid.New()

	created, err := database.CreateGroupMembershipInDB(membership)
	if err != nil {
		t.Fatalf("failed to create group membership: %v", err)
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

	created, err := database.CreateWishlistInDB(wishlist)
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

	created, err := database.CreateWishlistMembershipInDB(membership)
	if err != nil {
		t.Fatalf("failed to create wishlist membership: %v", err)
	}

	return created
}

// createTestWishCategory inserts an enabled category on a wishlist.
func createTestWishCategory(t *testing.T, wishlistID, ownerID uuid.UUID) models.WishCategory {
	t.Helper()

	category := models.WishCategory{
		Name:       "Category " + uuid.NewString(),
		WishlistID: wishlistID,
		OwnerID:    ownerID,
		Enabled:    true,
	}
	category.ID = uuid.New()

	if err := database.CreateWishCategoryInDB(category); err != nil {
		t.Fatalf("failed to create wish category: %v", err)
	}

	return category
}

// createTestWish inserts an enabled wish owned by ownerID on wishlistID.
func createTestWish(t *testing.T, ownerID, wishlistID uuid.UUID) models.Wish {
	t.Helper()

	wish := models.Wish{
		Name:       "Wish " + uuid.NewString(),
		Enabled:    true,
		OwnerID:    ownerID,
		WishlistID: wishlistID,
	}
	wish.ID = uuid.New()

	created, err := database.CreateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to create wish: %v", err)
	}

	return created
}

// createTestInvite inserts an enabled, unused invite code.
func createTestInvite(t *testing.T) models.Invite {
	t.Helper()

	code, err := database.GenerateRandomInvite()
	if err != nil {
		t.Fatalf("failed to create invite: %v", err)
	}

	invites, err := database.GetAllEnabledInvites()
	if err != nil {
		t.Fatalf("failed to list invites: %v", err)
	}
	for _, inv := range invites {
		if inv.Code == code {
			return inv
		}
	}

	t.Fatalf("created invite %q not found among enabled invites", code)
	return models.Invite{}
}

// createTestNews inserts an enabled news post.
func createTestNews(t *testing.T) models.News {
	t.Helper()

	news := models.News{
		Title:   "News " + uuid.NewString(),
		Body:    "Body",
		Enabled: true,
		Date:    time.Now(),
	}
	news.ID = uuid.New()

	created, err := database.CreateNewsPostInDB(news)
	if err != nil {
		t.Fatalf("failed to create news post: %v", err)
	}

	return created
}

// createTestUserWithPassword inserts an enabled, verified user whose password
// is a real bcrypt hash of plaintext, so handlers that call user.CheckPassword
// (e.g. the login endpoint) can be exercised on their success path.
func createTestUserWithPassword(t *testing.T, plaintext string) models.User {
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

// enablePrivateKey installs a valid base64 signing key for the first-party
// HS256 tokens (SSO session cookie, MFA challenge token, email verification,
// password reset) so handlers that mint or validate them don't fail with
// "private key is not configured". The previous key is restored on cleanup.
func enablePrivateKey(t *testing.T) {
	t.Helper()
	key, err := config.GenerateSecureKey(64)
	if err != nil {
		t.Fatalf("failed to generate test private key: %v", err)
	}
	restoreConfig(t)
	config.ConfigFile.PrivateKey = key
}

// restoreConfig snapshots config.ConfigFile and puts it back when the test
// ends. config.ConfigFile is a package-level global shared by every test in
// the binary, so any test or fixture that writes to it (directly, or through a
// handler like APIUpdateServerSettings) calls this first. Cleanups run LIFO,
// so nested snapshots within one test still unwind to the original.
func restoreConfig(t *testing.T) {
	t.Helper()
	original := config.ConfigFile
	t.Cleanup(func() { config.ConfigFile = original })
}

func boolPtr(b bool) *bool { return &b }

// authHeader mints a valid OAuth access token for userID and returns the
// "Authorization" header value an authenticated request needs. Handlers pull
// the user ID out of this via middlewares.GetAuthUsername, so tests that call
// a handler directly (bypassing the router's Auth middleware) still need it.
func authHeader(t *testing.T, userID uuid.UUID, admin bool) string {
	t.Helper()
	enableOAuth(t)
	token, err := auth.GenerateOAuthAccessToken(userID, models.FirstPartyClientID, config.APIResource(), "openid profile email", admin, true)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	return "Bearer " + token
}

// breakControllersDB closes the connection behind database.Instance so every
// query after this point fails. Handlers' "unexpected database error" 500
// branches can't otherwise be reached with a healthy in-memory DB; the next
// test's setupControllersDB installs a fresh instance, so nothing leaks.
func breakControllersDB(t *testing.T) {
	t.Helper()
	sqlDB, err := database.Instance.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("failed to close sql.DB: %v", err)
	}
}

// failDBOperation makes GORM operations of kind op ("query", "create",
// "update", "delete" or "row") against table fail once skip matching calls
// have gone through. breakControllersDB can only reach a handler's first
// query; this reaches the 500 branches further in, after earlier lookups
// have succeeded. It hooks the current database.Instance, which the next
// test's setupControllersDB replaces, so nothing leaks between tests.
func failDBOperation(t *testing.T, op, table string, skip int) {
	t.Helper()
	calls := 0
	registerDBCallback(t, op, func(db *gorm.DB) {
		if db.Statement.Table != table {
			return
		}
		calls++
		if calls > skip {
			db.AddError(errors.New("injected " + op + " failure on " + table))
		}
	})
}

// onDBOperation runs fn once, just before the (skip+1)th GORM operation of
// kind op against table. It reaches the branches for a record that vanishes
// or changes between two of a handler's checks. fn gets a fresh session on the
// statement's own connection (and transaction, if any), since the test DB has
// a single connection and a write through database.Instance would deadlock
// against an open transaction.
func onDBOperation(t *testing.T, op, table string, skip int, fn func(tx *gorm.DB)) {
	t.Helper()
	calls := 0
	registerDBCallback(t, op, func(db *gorm.DB) {
		if db.Statement.Table != table {
			return
		}
		calls++
		if calls == skip+1 {
			fn(db.Session(&gorm.Session{NewDB: true}))
		}
	})
}

// registerDBCallback hooks cb in just before GORM's own callback for op on
// the current database.Instance.
func registerDBCallback(t *testing.T, op string, cb func(*gorm.DB)) {
	t.Helper()
	name := "test:hook:" + uuid.NewString()
	callbacks := database.Instance.Callback()
	var err error
	switch op {
	case "query":
		err = callbacks.Query().Before("gorm:query").Register(name, cb)
	case "create":
		err = callbacks.Create().Before("gorm:create").Register(name, cb)
	case "update":
		err = callbacks.Update().Before("gorm:update").Register(name, cb)
	case "delete":
		err = callbacks.Delete().Before("gorm:delete").Register(name, cb)
	case "row":
		err = callbacks.Row().Before("gorm:row").Register(name, cb)
	default:
		t.Fatalf("unknown GORM operation %q", op)
	}
	if err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}
}

// dbErrorCase is one handler call expected to fail with a 500 once the
// database is unreachable.
type dbErrorCase struct {
	name    string
	handler gin.HandlerFunc
	method  string
	path    string
	body    string
	params  gin.Params
	admin   bool
}

// runDatabaseErrorCases calls each case's handler as an authenticated user
// against a broken database and asserts the handler reports a 500 rather than
// panicking or treating the failure as "not found"/"forbidden".
func runDatabaseErrorCases(t *testing.T, cases []dbErrorCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)
			header := map[string]string{"Authorization": authHeader(t, uuid.New(), c.admin)}
			breakControllersDB(t)

			status, body, _ := doRequest(c.handler, c.method, c.path, c.body, header, c.params)
			if status != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500; body=%v", status, body)
			}
		})
	}
}

// runUnauthenticatedCases calls each handler with no Authorization header and
// asserts it rejects the request with a 400 before doing any work. Handlers
// are normally behind middlewares.Auth, but each re-reads the caller from the
// header itself, so that branch still has to hold on its own.
func runUnauthenticatedCases(t *testing.T, cases []dbErrorCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupControllersDB(t)

			status, body, _ := doRequest(c.handler, c.method, c.path, c.body, nil, c.params)
			if status != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body=%v", status, body)
			}
		})
	}
}
