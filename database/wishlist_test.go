package database

import (
	"testing"
	"time"

	"aunefyren/poenskelisten/models"
	"aunefyren/poenskelisten/utilities"

	"github.com/google/uuid"
)

func TestCreateAndGetWishlist(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	found, got, err := GetWishlistByWishlistID(wishlist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || got.ID != wishlist.ID {
		t.Fatalf("expected to find created wishlist")
	}

	// GetWishlist errors when not found.
	if _, err := GetWishlist(uuid.New()); err == nil {
		t.Fatalf("expected error for missing wishlist, got nil")
	}

	ownerID, err := GetWishlistOwner(wishlist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ownerID != owner.ID {
		t.Fatalf("expected owner %v, got %v", owner.ID, ownerID)
	}
}

func TestGetOwnedWishlistsAndOwnership(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	stranger := createTestUser(t)
	createTestWishlist(t, owner.ID)
	wl := createTestWishlist(t, owner.ID)

	owned, err := GetOwnedWishlists(owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(owned) != 2 {
		t.Fatalf("expected 2 owned wishlists, got %d", len(owned))
	}

	if ok, err := VerifyUserOwnershipToWishlist(owner.ID, wl.ID); err != nil || !ok {
		t.Fatalf("expected owner to own wishlist (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyUserOwnershipToWishlist(stranger.ID, wl.ID); err != nil || ok {
		t.Fatalf("expected stranger not to own wishlist (ok=%v err=%v)", ok, err)
	}
}

func TestVerifyUniqueWishlistNameForUser(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wl := createTestWishlist(t, owner.ID)

	if unique, err := VerifyUniqueWishlistNameForUser("Totally new", owner.ID); err != nil || !unique {
		t.Fatalf("expected new name to be unique (unique=%v err=%v)", unique, err)
	}
	if unique, err := VerifyUniqueWishlistNameForUser(wl.Name, owner.ID); err != nil || unique {
		t.Fatalf("expected existing name to be reported taken (unique=%v err=%v)", unique, err)
	}
}

func TestGetPublicWishlistByHash(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	hash := uuid.New()

	now := time.Now()
	wishlist := models.Wishlist{
		Name:       "Public list",
		Enabled:    true,
		OwnerID:    owner.ID,
		Date:       &now,
		Public:     &utilities.DBTrue,
		PublicHash: hash,
	}
	wishlist.ID = uuid.New()
	if _, err := CreateWishlistInDB(wishlist); err != nil {
		t.Fatalf("failed to create public wishlist: %v", err)
	}

	found, got, err := GetPublicWishListByWishlistHash(hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || got.ID != wishlist.ID {
		t.Fatalf("expected to find public wishlist by hash")
	}

	// An unknown hash must not resolve.
	if found, _, err := GetPublicWishListByWishlistHash(uuid.New()); err != nil || found {
		t.Fatalf("expected unknown hash to not resolve (found=%v err=%v)", found, err)
	}
}

func TestDeleteWishlistSoftDisables(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	if err := DeleteWishlist(wishlist.ID); err != nil {
		t.Fatalf("DeleteWishlist returned error: %v", err)
	}

	found, _, err := GetWishlistByWishlistID(wishlist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatalf("expected disabled wishlist to be unreachable")
	}
}

func TestUpdateWishlistInDB(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	wishlist.Name = "Renamed list"
	wishlist.Description = "Updated"
	if _, err := UpdateWishlistInDB(wishlist); err != nil {
		t.Fatalf("UpdateWishlistInDB returned error: %v", err)
	}

	got, err := GetWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Renamed list" || got.Description != "Updated" {
		t.Fatalf("wishlist not updated, got name=%q desc=%q", got.Name, got.Description)
	}
}

func TestVerifyUniqueWishNameInWishlist(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	createWishForOwner(t, wishlist.ID, owner.ID, "Existing wish")

	if unique, err := VerifyUniqueWishNameInWishlist("Brand new", wishlist.ID); err != nil || !unique {
		t.Fatalf("expected new wish name to be unique (unique=%v err=%v)", unique, err)
	}
	if unique, err := VerifyUniqueWishNameInWishlist("Existing wish", wishlist.ID); err != nil || unique {
		t.Fatalf("expected existing wish name to be taken (unique=%v err=%v)", unique, err)
	}
}

func TestWishlistMembershipChain(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	member := createTestUser(t)
	stranger := createTestUser(t)

	group := createTestGroup(t, owner.ID)
	addGroupMember(t, group.ID, member.ID)
	wishlist := createTestWishlist(t, owner.ID)
	membership := addWishlistMembership(t, wishlist.ID, group.ID)

	// Look up the membership row that links the group to the wishlist.
	found, gotMembership, err := GetMembershipIDForGroupToWishlist(wishlist.ID, group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found || gotMembership.ID != membership.ID {
		t.Fatalf("expected to find wishlist membership %v", membership.ID)
	}

	// Wishlists reachable by the member via the group.
	wishlists, err := GetWishlistsByUserIDThroughWishlistMemberships(member.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wishlists) != 1 || wishlists[0].ID != wishlist.ID {
		t.Fatalf("expected member to reach 1 wishlist, got %d", len(wishlists))
	}

	// A stranger reaches nothing.
	none, err := GetWishlistsByUserIDThroughWishlistMemberships(stranger.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected stranger to reach 0 wishlists, got %d", len(none))
	}

	// Wishlists attached to the group.
	fromGroup, err := GetWishlistsFromGroup(group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fromGroup) != 1 || fromGroup[0].ID != wishlist.ID {
		t.Fatalf("expected 1 wishlist from group, got %d", len(fromGroup))
	}

	// Membership-based access check.
	if ok, err := VerifyUserMembershipToGroupMembershipToWishlist(member.ID, wishlist.ID); err != nil || !ok {
		t.Fatalf("expected member to have access via group (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyUserMembershipToGroupMembershipToWishlist(stranger.ID, wishlist.ID); err != nil || ok {
		t.Fatalf("expected stranger to lack access (ok=%v err=%v)", ok, err)
	}

	// Deleting the wishlist membership severs the chain.
	if err := DeleteWishlistMembership(membership.ID); err != nil {
		t.Fatalf("DeleteWishlistMembership returned error: %v", err)
	}
	after, err := GetWishlistsByUserIDThroughWishlistMemberships(member.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected 0 wishlists after membership delete, got %d", len(after))
	}
}

func TestGetWishlistsByUserIDThroughCollaborations(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	collab := models.WishlistCollaborator{
		UserID:     collaborator.ID,
		WishlistID: wishlist.ID,
		Enabled:    true,
	}
	collab.ID = uuid.New()
	if err := CreateWishlistCollaboratorInDB(collab); err != nil {
		t.Fatalf("failed to create collaborator: %v", err)
	}

	wishlists, err := GetWishlistsByUserIDThroughWishlistCollaborations(collaborator.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wishlists) != 1 || wishlists[0].ID != wishlist.ID {
		t.Fatalf("expected collaborator to reach 1 wishlist, got %d", len(wishlists))
	}

	// Direct collaborator lookup by user + wishlist.
	got, err := GetWishlistCollaboratorByUserIDAndWishlistID(wishlist.ID, collaborator.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != collab.ID {
		t.Fatalf("expected collaborator %v, got %v", collab.ID, got.ID)
	}
}

func TestWishlistCollaborators(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)

	collab := models.WishlistCollaborator{
		UserID:     collaborator.ID,
		WishlistID: wishlist.ID,
		Enabled:    true,
	}
	collab.ID = uuid.New()
	if err := CreateWishlistCollaboratorInDB(collab); err != nil {
		t.Fatalf("failed to create collaborator: %v", err)
	}

	if ok, err := VerifyWishlistCollaboratorToWishlist(wishlist.ID, collaborator.ID); err != nil || !ok {
		t.Fatalf("expected collaborator to be verified (ok=%v err=%v)", ok, err)
	}

	collaborators, err := GetWishlistCollaboratorsFromWishlist(wishlist.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(collaborators) != 1 {
		t.Fatalf("expected 1 collaborator, got %d", len(collaborators))
	}

	if err := DeleteWishlistCollaboratorByWishlistCollaboratorID(collab.ID); err != nil {
		t.Fatalf("failed to delete collaborator: %v", err)
	}
	if ok, err := VerifyWishlistCollaboratorToWishlist(wishlist.ID, collaborator.ID); err != nil || ok {
		t.Fatalf("expected collaborator to be gone after delete (ok=%v err=%v)", ok, err)
	}
}
