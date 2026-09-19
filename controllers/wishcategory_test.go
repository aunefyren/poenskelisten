package controllers

import (
	"aunefyren/poenskelisten/database"
	"aunefyren/poenskelisten/models"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestResolveWishCategoryForWishNoCategory(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)

	resolved, userErr, err := ResolveWishCategoryForWish(wishlist.ID, user.ID, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr != "" {
		t.Errorf("userErr = %q, want empty", userErr)
	}
	if resolved != nil {
		t.Errorf("resolved = %v, want nil", resolved)
	}
}

func TestResolveWishCategoryForWishCreatesByName(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)

	resolved, userErr, err := ResolveWishCategoryForWish(wishlist.ID, user.ID, nil, "Electronics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr != "" {
		t.Fatalf("userErr = %q, want empty", userErr)
	}
	if resolved == nil {
		t.Fatal("expected a resolved category ID")
	}

	categories, err := database.GetWishCategoriesFromWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("failed to list categories: %v", err)
	}
	if len(categories) != 1 || categories[0].Name != "Electronics" {
		t.Errorf("categories = %v, want exactly one named Electronics", categories)
	}
}

func TestResolveWishCategoryForWishReusesExistingName(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)
	existing := createTestWishCategory(t, wishlist.ID, user.ID)

	resolved, userErr, err := ResolveWishCategoryForWish(wishlist.ID, user.ID, nil, existing.Name)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr != "" {
		t.Fatalf("userErr = %q, want empty", userErr)
	}
	if resolved == nil || *resolved != existing.ID {
		t.Errorf("resolved = %v, want the existing category %v", resolved, existing.ID)
	}
}

func TestResolveWishCategoryForWishNameTooShort(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)

	_, userErr, err := ResolveWishCategoryForWish(wishlist.ID, user.ID, nil, "A")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr == "" {
		t.Error("expected a validation error for a one-letter category name")
	}
}

func TestResolveWishCategoryForWishInvalidCharacters(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)

	_, userErr, err := ResolveWishCategoryForWish(wishlist.ID, user.ID, nil, "<bad>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr == "" {
		t.Error("expected a validation error for disallowed characters")
	}
}

func TestResolveWishCategoryForWishByID(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)
	existing := createTestWishCategory(t, wishlist.ID, user.ID)

	resolved, userErr, err := ResolveWishCategoryForWish(wishlist.ID, user.ID, &existing.ID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr != "" {
		t.Fatalf("userErr = %q, want empty", userErr)
	}
	if resolved == nil || *resolved != existing.ID {
		t.Errorf("resolved = %v, want %v", resolved, existing.ID)
	}
}

func TestResolveWishCategoryForWishByIDWrongWishlist(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlistA := createTestWishlist(t, user.ID)
	wishlistB := createTestWishlist(t, user.ID)
	existing := createTestWishCategory(t, wishlistA.ID, user.ID)

	_, userErr, err := ResolveWishCategoryForWish(wishlistB.ID, user.ID, &existing.ID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr == "" {
		t.Error("expected a validation error when the category belongs to a different wishlist")
	}
}

func TestResolveWishCategoryForWishByIDNotFound(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)
	missing := uuid.New()

	_, userErr, err := ResolveWishCategoryForWish(wishlist.ID, user.ID, &missing, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userErr == "" {
		t.Error("expected a validation error for a nonexistent category ID")
	}
}

func TestCleanupWishCategoryIfEmptyRemoves(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)
	category := createTestWishCategory(t, wishlist.ID, user.ID)

	CleanupWishCategoryIfEmpty(category.ID)

	got, err := database.GetWishCategoryByID(category.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected the empty category to be removed, got %v", got)
	}
}

func TestCleanupWishCategoryIfEmptyKeepsNonEmpty(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)
	category := createTestWishCategory(t, wishlist.ID, user.ID)
	wish := createTestWish(t, user.ID, wishlist.ID)
	wish.CategoryID = &category.ID
	if result := database.Instance.Save(&wish); result.Error != nil {
		t.Fatalf("failed to assign category to wish: %v", result.Error)
	}

	CleanupWishCategoryIfEmpty(category.ID)

	got, err := database.GetWishCategoryByID(category.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Error("expected the non-empty category to remain")
	}
}

func TestGetWishlistCategoriesRequiresAuth(t *testing.T) {
	setupControllersDB(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/wishlists/x/categories", nil)
	ctx.Params = gin.Params{{Key: "wishlist_id", Value: uuid.NewString()}}
	APIGetWishlistCategories(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 without an Authorization header", w.Code)
	}
}

func TestGetWishlistCategoriesBadWishlistID(t *testing.T) {
	setupControllersDB(t)
	user := createTestUser(t)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/wishlists/x/categories", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, user.ID, false))
	ctx.Params = gin.Params{{Key: "wishlist_id", Value: "not-a-uuid"}}
	APIGetWishlistCategories(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a malformed wishlist_id", w.Code)
	}
}

func TestGetWishlistCategoriesForbidden(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	outsider := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/wishlists/"+wishlist.ID.String()+"/categories", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, outsider.ID, false))
	ctx.Params = gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}}
	APIGetWishlistCategories(ctx)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400 for a non-owner non-collaborator", w.Code)
	}
}

func TestGetWishlistCategoriesSuccess(t *testing.T) {
	setupControllersDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	createTestWishCategory(t, wishlist.ID, owner.ID)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("GET", "/api/auth/wishlists/"+wishlist.ID.String()+"/categories", nil)
	ctx.Request.Header.Set("Authorization", authHeader(t, owner.ID, false))
	ctx.Params = gin.Params{{Key: "wishlist_id", Value: wishlist.ID.String()}}
	APIGetWishlistCategories(ctx)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}

	var body struct {
		Categories []map[string]interface{} `json:"categories"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(body.Categories) != 1 {
		t.Errorf("categories = %v, want 1", body.Categories)
	}
}

func TestCleanupWishCategoryIfEmptyCountFailure(t *testing.T) {
	// Migrate without the Wish table so CountEnabledWishesInCategory fails.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishCategory{})
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)
	category := createTestWishCategory(t, wishlist.ID, user.ID)

	// Should not panic; the failure is logged and swallowed.
	CleanupWishCategoryIfEmpty(category.ID)

	got, err := database.GetWishCategoryByID(category.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Error("expected the category to remain untouched when the count query fails")
	}
}

func TestCleanupWishCategoryIfEmptyDeleteFailure(t *testing.T) {
	// Wish table present (so counting succeeds and returns 0) but drop the
	// WishCategory table afterward so the delete step fails.
	setupControllersDB(t, &models.User{}, &models.Wishlist{}, &models.WishCategory{}, &models.Wish{})
	user := createTestUser(t)
	wishlist := createTestWishlist(t, user.ID)
	category := createTestWishCategory(t, wishlist.ID, user.ID)

	if err := database.Instance.Migrator().DropTable(&models.WishCategory{}); err != nil {
		t.Fatalf("failed to drop wish_categories table: %v", err)
	}

	// Should not panic; the failure is logged and swallowed.
	CleanupWishCategoryIfEmpty(category.ID)
}

func TestWishCategoryHandlersDatabaseErrors(t *testing.T) {
	runDatabaseErrorCases(t, []dbErrorCase{
		{name: "APIGetWishlistCategories", handler: APIGetWishlistCategories, method: "GET", path: "/api/auth/wishlists/00000000-0000-0000-0000-00000000000a/categories", params: gin.Params{{Key: "wishlist_id", Value: "00000000-0000-0000-0000-00000000000a"}}},
	})
}
