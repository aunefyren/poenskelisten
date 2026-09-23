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

func TestVerifyUniqueWishNameInWishlistExcludingWish(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	wish := createWishForOwner(t, wishlist.ID, owner.ID, "Headphones")

	t.Run("the wish's own name, excluding itself, is unique", func(t *testing.T) {
		ok, err := VerifyUniqueWishNameInWishlistExcludingWish("Headphones", wishlist.ID, wish.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected the wish's own name to be considered unique when excluding itself")
		}
	})

	t.Run("a genuinely new name is unique", func(t *testing.T) {
		ok, err := VerifyUniqueWishNameInWishlistExcludingWish("Something Else", wishlist.ID, wish.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected a new name to be unique")
		}
	})

	t.Run("a name used by a different wish is not unique", func(t *testing.T) {
		createWishForOwner(t, wishlist.ID, owner.ID, "Watch")
		ok, err := VerifyUniqueWishNameInWishlistExcludingWish("Watch", wishlist.ID, wish.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected the name of a different wish to not be unique")
		}
	})
}

func TestCreateWishlistInDBFailure(t *testing.T) {
	setupTestDB(t)
	owner := createTestUser(t)
	if err := Instance.Migrator().DropTable(&models.Wishlist{}); err != nil {
		t.Fatalf("failed to drop wishlists table: %v", err)
	}

	now := time.Now()
	wishlist := models.Wishlist{Name: "Test", Enabled: true, OwnerID: owner.ID, Date: &now}
	wishlist.ID = uuid.New()
	if _, err := CreateWishlistInDB(wishlist); err == nil {
		t.Error("expected an error when the wishlists table is unavailable")
	}
}

func TestCreateWishlistCollaboratorInDBFailure(t *testing.T) {
	setupTestDB(t)
	owner := createTestUser(t)
	collaborator := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	if err := Instance.Migrator().DropTable(&models.WishlistCollaborator{}); err != nil {
		t.Fatalf("failed to drop wishlist_collaborators table: %v", err)
	}

	collab := models.WishlistCollaborator{UserID: collaborator.ID, WishlistID: wishlist.ID, Enabled: true}
	collab.ID = uuid.New()
	if err := CreateWishlistCollaboratorInDB(collab); err == nil {
		t.Error("expected an error when the wishlist_collaborators table is unavailable")
	}
}

func TestCreateWishlistMembershipInDBFailure(t *testing.T) {
	setupTestDB(t)
	owner := createTestUser(t)
	wishlist := createTestWishlist(t, owner.ID)
	group := createTestGroup(t, owner.ID)
	if err := Instance.Migrator().DropTable(&models.WishlistMembership{}); err != nil {
		t.Fatalf("failed to drop wishlist_memberships table: %v", err)
	}

	membership := models.WishlistMembership{GroupID: group.ID, WishlistID: wishlist.ID, Enabled: true}
	membership.ID = uuid.New()
	if _, err := CreateWishlistMembershipInDB(membership); err == nil {
		t.Error("expected an error when the wishlist_memberships table is unavailable")
	}
}

func TestWishlistQueriesFailOnClosedDB(t *testing.T) {
	runClosedDBCases(t, map[string]func() error{
		"UpdateWishlistInDB": func() error {
			_, err := UpdateWishlistInDB(models.Wishlist{})
			return err
		},
		"CreateWishlistInDB": func() error {
			_, err := CreateWishlistInDB(models.Wishlist{})
			return err
		},
		"GetWishlistByWishlistID": func() error {
			_, _, err := GetWishlistByWishlistID(uuid.New())
			return err
		},
		"GetWishlistCollaboratorsFromWishlist": func() error {
			_, err := GetWishlistCollaboratorsFromWishlist(uuid.New())
			return err
		},
		"GetWishlistCollaboratorByUserIDAndWishlistID": func() error {
			_, err := GetWishlistCollaboratorByUserIDAndWishlistID(uuid.New(), uuid.New())
			return err
		},
		"CreateWishlistCollaboratorInDB": func() error {
			return CreateWishlistCollaboratorInDB(models.WishlistCollaborator{})
		},
		"VerifyWishlistCollaboratorToWishlist": func() error {
			_, err := VerifyWishlistCollaboratorToWishlist(uuid.New(), uuid.New())
			return err
		},
		"DeleteWishlistCollaboratorByWishlistCollaboratorID": func() error {
			return DeleteWishlistCollaboratorByWishlistCollaboratorID(uuid.New())
		},
		"GetWishlistsByUserIDThroughWishlistCollaborations": func() error {
			_, err := GetWishlistsByUserIDThroughWishlistCollaborations(uuid.New())
			return err
		},
		"GetWishlistsByUserIDThroughWishlistMemberships": func() error {
			_, err := GetWishlistsByUserIDThroughWishlistMemberships(uuid.New())
			return err
		},
		"GetWishlistsFromGroup": func() error {
			_, err := GetWishlistsFromGroup(uuid.New())
			return err
		},
		"GetOwnedWishlists": func() error {
			_, err := GetOwnedWishlists(uuid.New())
			return err
		},
		"GetWishlist": func() error {
			_, err := GetWishlist(uuid.New())
			return err
		},
		"VerifyUniqueWishNameInWishlist": func() error {
			_, err := VerifyUniqueWishNameInWishlist("x", uuid.New())
			return err
		},
		"VerifyUniqueWishNameInWishlistExcludingWish": func() error {
			_, err := VerifyUniqueWishNameInWishlistExcludingWish("x", uuid.New(), uuid.New())
			return err
		},
		"VerifyUniqueWishlistNameForUser": func() error {
			_, err := VerifyUniqueWishlistNameForUser("x", uuid.New())
			return err
		},
		"GetWishlistOwner": func() error {
			_, err := GetWishlistOwner(uuid.New())
			return err
		},
		"VerifyUserMembershipToGroupMembershipToWishlist": func() error {
			_, err := VerifyUserMembershipToGroupMembershipToWishlist(uuid.New(), uuid.New())
			return err
		},
		"VerifyUserOwnershipToWishlist": func() error {
			_, err := VerifyUserOwnershipToWishlist(uuid.New(), uuid.New())
			return err
		},
		"GetMembershipIDForGroupToWishlist": func() error {
			_, _, err := GetMembershipIDForGroupToWishlist(uuid.New(), uuid.New())
			return err
		},
		"GetPublicWishListByWishlistHash": func() error {
			_, _, err := GetPublicWishListByWishlistHash(uuid.New())
			return err
		},
		"CreateWishlistMembershipInDB": func() error {
			_, err := CreateWishlistMembershipInDB(models.WishlistMembership{})
			return err
		},
	})
}

func TestWishlistLookupsWithNoMatch(t *testing.T) {
	setupTestDB(t)
	missing := uuid.New()

	if err := DeleteWishlistCollaboratorByWishlistCollaboratorID(missing); err == nil {
		t.Error("DeleteWishlistCollaboratorByWishlistCollaboratorID: expected error for an unknown collaborator")
	}
	if wishlists, err := GetWishlistsFromGroup(missing); err != nil || wishlists == nil || len(wishlists) != 0 {
		t.Errorf("GetWishlistsFromGroup = (%v, %v), want an empty non-nil slice", wishlists, err)
	}
	if owner, err := GetWishlistOwner(missing); err == nil || owner != uuid.Nil {
		t.Errorf("GetWishlistOwner = (%v, %v), want an error and no owner", owner, err)
	}
	if found, _, err := GetMembershipIDForGroupToWishlist(missing, uuid.New()); err == nil || found {
		t.Errorf("GetMembershipIDForGroupToWishlist = (found=%v, %v), want an error", found, err)
	}
}

func TestWishlistWritesRejectWrongRowCount(t *testing.T) {
	setupTestDB(t)
	owner := createTestUser(t)
	existing := createTestWishlist(t, owner.ID)
	for _, table := range []string{"wishlists", "wishlist_collaborators", "wishlist_memberships"} {
		forceRowsAffected(t, "create", table, 0)
	}
	// Save falls back to an upsert when its UPDATE touches nothing, so both
	// paths have to report zero rows to reach the check.
	forceRowsAffected(t, "update", "wishlists", 0)

	if _, err := UpdateWishlistInDB(existing); err == nil || err.Error() != "Wishlist not changed in database." {
		t.Errorf("UpdateWishlistInDB error = %v, want \"Wishlist not changed in database.\"", err)
	}

	now := time.Now()
	wishlist := models.Wishlist{Name: "w", Enabled: true, OwnerID: owner.ID, Date: &now}
	wishlist.ID = uuid.New()
	if _, err := CreateWishlistInDB(wishlist); err == nil || err.Error() != "Wishlist not added to database." {
		t.Errorf("CreateWishlistInDB error = %v, want \"Wishlist not added to database.\"", err)
	}

	collaborator := models.WishlistCollaborator{WishlistID: existing.ID, UserID: owner.ID, Enabled: true}
	collaborator.ID = uuid.New()
	if err := CreateWishlistCollaboratorInDB(collaborator); err == nil || err.Error() != "Wishlist not added to database." {
		t.Errorf("CreateWishlistCollaboratorInDB error = %v, want \"Wishlist not added to database.\"", err)
	}

	membership := models.WishlistMembership{WishlistID: existing.ID, GroupID: uuid.New(), Enabled: true}
	membership.ID = uuid.New()
	if _, err := CreateWishlistMembershipInDB(membership); err == nil || err.Error() != "Wishlist not added to database." {
		t.Errorf("CreateWishlistMembershipInDB error = %v, want \"Wishlist not added to database.\"", err)
	}
}
