package database

import (
	"testing"

	"aunefyren/poenskelisten/models"

	"github.com/google/uuid"
)

// createWishForOwner inserts an enabled wish owned by ownerID. Unlike the
// category-test helper, it takes a real owner so queries that join on an enabled
// owner (e.g. GetWishesFromWishlist) return the row.
func createWishForOwner(t *testing.T, wishlistID, ownerID uuid.UUID, name string) models.Wish {
	t.Helper()

	wish := models.Wish{
		Name:       name,
		Enabled:    true,
		OwnerID:    ownerID,
		WishlistID: wishlistID,
	}
	wish.ID = uuid.New()

	created, err := CreateWishInDB(wish)
	if err != nil {
		t.Fatalf("failed to create wish: %v", err)
	}
	// Regression guard: the create helper must return the persisted record.
	if created.ID != wish.ID {
		t.Fatalf("expected created wish to carry its ID, got %v", created.ID)
	}

	return created
}

func TestGetWishAndWishlistID(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createWishForOwner(t, wishlist.ID, owner.ID, "A wish")

	got, err := GetWishByWishID(wish.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != wish.ID {
		t.Fatalf("expected to find wish by ID")
	}

	wishlistID, err := GetWishlistIDFromWish(wish.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wishlistID == nil || *wishlistID != wishlist.ID {
		t.Fatalf("expected wishlist ID %v, got %v", wishlist.ID, wishlistID)
	}

	// Unknown wish resolves to (nil, nil), not an error.
	if got, err := GetWishByWishID(uuid.New()); err != nil || got != nil {
		t.Fatalf("expected nil,nil for unknown wish (got=%v err=%v)", got, err)
	}
}

func TestGetWishesFromWishlistRequiresEnabledOwner(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	createWishForOwner(t, wishlist.ID, owner.ID, "One")
	createWishForOwner(t, wishlist.ID, owner.ID, "Two")

	found, wishes, err := GetWishesFromWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || len(wishes) != 2 {
		t.Fatalf("expected 2 wishes, got %d (found=%v)", len(wishes), found)
	}

	// Disabling the owner hides their wishes from the listing.
	owner.Enabled = boolPtr(false)
	if _, err := UpdateUserInDB(owner); err != nil {
		t.Fatalf("failed to disable owner: %v", err)
	}
	found, wishes, err = GetWishesFromWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found || len(wishes) != 0 {
		t.Fatalf("expected no wishes once owner disabled, got %d (found=%v)", len(wishes), found)
	}
}

func TestVerifyUserOwnershipToWish(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	stranger := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createWishForOwner(t, wishlist.ID, owner.ID, "Mine")

	if ok, err := VerifyUserOwnershipToWish(owner.ID, wish.ID); err != nil || !ok {
		t.Fatalf("expected owner to own wish (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyUserOwnershipToWish(stranger.ID, wish.ID); err != nil || ok {
		t.Fatalf("expected stranger not to own wish (ok=%v err=%v)", ok, err)
	}
}

func TestWishClaimLifecycle(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	claimer := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createWishForOwner(t, wishlist.ID, owner.ID, "Claimable")

	// Not claimed to start with.
	if claimed, err := VerifyWishIsClaimed(wish.ID); err != nil || claimed {
		t.Fatalf("expected wish to be unclaimed (claimed=%v err=%v)", claimed, err)
	}

	claim := models.WishClaim{
		WishID:  wish.ID,
		UserID:  claimer.ID,
		Enabled: true,
	}
	claim.ID = uuid.New()
	created, err := CreateWishClaimInDB(claim)
	if err != nil {
		t.Fatalf("failed to create wish claim: %v", err)
	}
	if created.ID != claim.ID {
		t.Fatalf("expected created claim to carry its ID, got %v", created.ID)
	}

	if claimed, err := VerifyWishIsClaimed(wish.ID); err != nil || !claimed {
		t.Fatalf("expected wish to be claimed (claimed=%v err=%v)", claimed, err)
	}
	if ok, err := VerifyUserOwnershipToWishClaimByWish(claimer.ID, wish.ID); err != nil || !ok {
		t.Fatalf("expected claimer to own the claim (ok=%v err=%v)", ok, err)
	}

	claims, err := GetWishClaimFromWish(wish.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(claims) != 1 || claims[0].User.ID != claimer.ID {
		t.Fatalf("expected one claim by the claimer, got %d", len(claims))
	}

	if err := DeleteWishClaimByUserAndWish(wish.ID, claimer.ID); err != nil {
		t.Fatalf("failed to delete claim: %v", err)
	}
	if claimed, err := VerifyWishIsClaimed(wish.ID); err != nil || claimed {
		t.Fatalf("expected wish to be unclaimed after delete (claimed=%v err=%v)", claimed, err)
	}
}

func TestUpdateWishInDB(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createWishForOwner(t, wishlist.ID, owner.ID, "Original")

	wish.Name = "Updated name"
	wish.Note = "Updated note"
	if _, err := UpdateWishInDB(wish); err != nil {
		t.Fatalf("UpdateWishInDB returned error: %v", err)
	}

	got, err := GetWishByWishID(wish.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Name != "Updated name" || got.Note != "Updated note" {
		t.Fatalf("wish not updated, got %+v", got)
	}
}

func TestGetWishlistByWishID(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createWishForOwner(t, wishlist.ID, owner.ID, "Linked")

	found, got, err := GetWishlistByWishID(wish.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || got.ID != wishlist.ID {
		t.Fatalf("expected to resolve wishlist from wish")
	}
}
