package database

import (
	"aunefyren/poenskelisten/models"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

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

	err = instance.AutoMigrate(&models.User{}, &models.Wishlist{}, &models.WishCategory{}, &models.Wish{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	Instance = instance
}

func createTestCategory(t *testing.T, wishlistID uuid.UUID, name string, sortOrder int) models.WishCategory {
	t.Helper()

	category := models.WishCategory{
		Name:       name,
		WishlistID: wishlistID,
		OwnerID:    uuid.New(),
		SortOrder:  sortOrder,
		Enabled:    true,
	}
	category.ID = uuid.New()

	if err := CreateWishCategoryInDB(category); err != nil {
		t.Fatalf("failed to create category %q: %v", name, err)
	}

	return category
}

func createTestWish(t *testing.T, wishlistID uuid.UUID, categoryID *uuid.UUID) models.Wish {
	t.Helper()

	wish := models.Wish{
		Name:       "Test wish " + uuid.NewString(),
		Enabled:    true,
		OwnerID:    uuid.New(),
		WishlistID: wishlistID,
		CategoryID: categoryID,
	}
	wish.ID = uuid.New()

	if _, err := CreateWishInDB(wish); err != nil {
		t.Fatalf("failed to create wish: %v", err)
	}

	return wish
}

func TestCreateAndGetWishCategories(t *testing.T) {
	setupTestDB(t)

	wishlistID := uuid.New()
	otherWishlistID := uuid.New()

	catB := createTestCategory(t, wishlistID, "Books", 1)
	catA := createTestCategory(t, wishlistID, "Vinyl", 0)
	createTestCategory(t, otherWishlistID, "Elsewhere", 0)

	categories, err := GetWishCategoriesFromWishlist(wishlistID)
	if err != nil {
		t.Fatalf("GetWishCategoriesFromWishlist returned error: %v", err)
	}

	if len(categories) != 2 {
		t.Fatalf("expected 2 categories for wishlist, got %d", len(categories))
	}

	// Ordering is by sort_order ascending, so Vinyl (0) precedes Books (1).
	if categories[0].ID != catA.ID || categories[1].ID != catB.ID {
		t.Fatalf("categories not ordered by sort_order: got %q then %q", categories[0].Name, categories[1].Name)
	}
}

func TestGetWishCategoryByNameInWishlist(t *testing.T) {
	setupTestDB(t)

	wishlistID := uuid.New()
	created := createTestCategory(t, wishlistID, "Vinyl", 0)

	found, err := GetWishCategoryByNameInWishlist("Vinyl", wishlistID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found == nil || found.ID != created.ID {
		t.Fatalf("expected to find the created category by name")
	}

	// A name that exists in another wishlist must not match here.
	missing, err := GetWishCategoryByNameInWishlist("Vinyl", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for name in a different wishlist, got %v", missing)
	}
}

func TestGetNextWishCategorySortOrder(t *testing.T) {
	setupTestDB(t)

	wishlistID := uuid.New()

	// Empty wishlist starts at 0.
	next, err := GetNextWishCategorySortOrder(wishlistID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != 0 {
		t.Fatalf("expected next sort order 0 for empty wishlist, got %d", next)
	}

	createTestCategory(t, wishlistID, "Vinyl", 0)
	createTestCategory(t, wishlistID, "Books", 3)

	next, err = GetNextWishCategorySortOrder(wishlistID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != 4 {
		t.Fatalf("expected next sort order 4 (max 3 + 1), got %d", next)
	}
}

func TestDeleteWishCategoryDetachesWishes(t *testing.T) {
	setupTestDB(t)

	wishlistID := uuid.New()
	category := createTestCategory(t, wishlistID, "Vinyl", 0)

	wishA := createTestWish(t, wishlistID, &category.ID)
	createTestWish(t, wishlistID, &category.ID)

	count, err := CountEnabledWishesInCategory(category.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 wishes in category, got %d", count)
	}

	if err := DeleteWishCategory(category.ID); err != nil {
		t.Fatalf("DeleteWishCategory returned error: %v", err)
	}

	// The category should now be disabled and therefore not returned.
	if got, err := GetWishCategoryByID(category.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if got != nil {
		t.Fatalf("expected disabled category to be unreachable, got %v", got)
	}

	// Its wishes should have been detached (category_id nulled).
	refreshed, err := GetWishByWishID(wishA.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refreshed == nil {
		t.Fatalf("expected wish to still exist")
	}
	if refreshed.CategoryID != nil {
		t.Fatalf("expected wish category to be detached, still points at %v", refreshed.CategoryID)
	}
}
