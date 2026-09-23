package database

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"aunefyren/poenskelisten/models"

	"github.com/google/uuid"
)

// addGroupMember enrolls memberID into groupID and returns the membership.
func addGroupMember(t *testing.T, groupID, memberID uuid.UUID) models.GroupMembership {
	t.Helper()

	membership := models.GroupMembership{
		GroupID:  groupID,
		MemberID: memberID,
		Enabled:  true,
	}
	membership.ID = uuid.New()

	created, err := CreateGroupMembershipInDB(membership)
	if err != nil {
		t.Fatalf("failed to create group membership: %v", err)
	}
	// Regression guard: the create helper must return the persisted row, not a
	// zero value.
	if created.ID != membership.ID {
		t.Fatalf("expected created membership to carry its ID, got %v", created.ID)
	}

	return created
}

func TestCreateGroupReturnsPersistedRecord(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	// Regression guard for the create-helper returning an empty struct.
	if group.ID == uuid.Nil || group.OwnerID != owner.ID {
		t.Fatalf("expected created group to be populated, got %+v", group)
	}
}

func TestVerifyGroupExistsAndUniqueness(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	exists, found, err := VerifyGroupExistsByNameForUser(group.Name, owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists || found.ID != group.ID {
		t.Fatalf("expected to find group by name for owner")
	}

	// A different owner with the same name should not match.
	other := createTestUser(t)
	exists, _, err = VerifyGroupExistsByNameForUser(group.Name, other.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatalf("did not expect a match for a different owner")
	}

	taken, err := VerifyIfGroupWithSameNameAndOwnerDoesNotExist(group.Name, owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !taken {
		t.Fatalf("expected duplicate name+owner to be reported as existing")
	}
}

func TestUpdateGroupValuesByID(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	if err := UpdateGroupValuesByID(group.ID, "Renamed", "New description"); err != nil {
		t.Fatalf("UpdateGroupValuesByID returned error: %v", err)
	}

	got, err := GetGroupInformation(group.ID)
	if err != nil {
		t.Fatalf("GetGroupInformation returned error: %v", err)
	}
	if got.Name != "Renamed" || got.Description != "New description" {
		t.Fatalf("group not updated, got name=%q desc=%q", got.Name, got.Description)
	}
}

func TestVerifyUserOwnershipAndMembership(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	member := createTestUser(t)
	stranger := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMember(t, group.ID, member.ID)

	if ok, err := VerifyUserOwnershipToGroup(owner.ID, group.ID); err != nil || !ok {
		t.Fatalf("expected owner to own group (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyUserOwnershipToGroup(member.ID, group.ID); err != nil || ok {
		t.Fatalf("expected member not to be owner (ok=%v err=%v)", ok, err)
	}

	if ok, err := VerifyUserMembershipToGroup(member.ID, group.ID); err != nil || !ok {
		t.Fatalf("expected member to be a member (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyUserMembershipToGroup(stranger.ID, group.ID); err != nil || ok {
		t.Fatalf("expected stranger not to be a member (ok=%v err=%v)", ok, err)
	}
}

func TestGetGroupsAUserIsAMemberOf(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	member := createTestUser(t)

	groupA := createTestGroup(t, owner.ID)
	groupB := createTestGroup(t, owner.ID)
	createTestGroup(t, owner.ID) // member is not in this one

	addGroupMember(t, groupA.ID, member.ID)
	addGroupMember(t, groupB.ID, member.ID)

	groups, err := GetGroupsAUserIsAMemberOf(member.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected member to be in 2 groups, got %d", len(groups))
	}

	memberships, err := GetGroupMembershipsFromGroup(groupA.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memberships) != 1 || memberships[0].MemberID != member.ID {
		t.Fatalf("expected one membership for member in groupA, got %d", len(memberships))
	}
}

func TestDeleteGroupSoftDisables(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)

	if err := DeleteGroup(group.ID); err != nil {
		t.Fatalf("DeleteGroup returned error: %v", err)
	}

	if _, err := GetGroupInformation(group.ID); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("err = %v, want ErrGroupNotFound for a disabled group", err)
	}
}

func TestGetGroupUsingGroupIDAndMembership(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	member := createTestUser(t)
	stranger := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMember(t, group.ID, member.ID)

	got, err := GetGroupUsingGroupIDAndMembershipUsingUserID(member.ID, group.ID)
	if err != nil {
		t.Fatalf("expected member to resolve group: %v", err)
	}
	if got.ID != group.ID {
		t.Fatalf("expected group %v, got %v", group.ID, got.ID)
	}

	if _, err := GetGroupUsingGroupIDAndMembershipUsingUserID(stranger.ID, group.ID); err == nil {
		t.Fatalf("expected error for non-member, got nil")
	}
}

func TestGetGroupUsingGroupIDAndUserIDAsOwner(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	member := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	addGroupMember(t, group.ID, member.ID)

	got, err := GetGroupUsingGroupIDAndUserIDAsOwner(owner.ID, group.ID)
	if err != nil {
		t.Fatalf("expected owner to resolve group: %v", err)
	}
	if got.ID != group.ID {
		t.Fatalf("expected group %v, got %v", group.ID, got.ID)
	}

	// A member who is not the owner must not resolve as owner.
	if _, err := GetGroupUsingGroupIDAndUserIDAsOwner(member.ID, group.ID); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("err = %v, want ErrGroupNotFound for non-owner", err)
	}
}

func TestGetGroupMembershipByGroupIDAndMemberID(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	member := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	membership := addGroupMember(t, group.ID, member.ID)

	got, err := GetGroupMembershipByGroupIDAndMemberID(group.ID, member.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != membership.ID {
		t.Fatalf("expected membership %v, got %v", membership.ID, got.ID)
	}

	if _, err := GetGroupMembershipByGroupIDAndMemberID(group.ID, uuid.New()); !errors.Is(err, ErrGroupMembershipNotFound) {
		t.Fatalf("err = %v, want ErrGroupMembershipNotFound for unknown member", err)
	}
}

func TestDeleteGroupMembership(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	member := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	membership := addGroupMember(t, group.ID, member.ID)

	if err := DeleteGroupMembership(membership.ID); err != nil {
		t.Fatalf("DeleteGroupMembership returned error: %v", err)
	}

	if ok, err := VerifyUserMembershipToGroup(member.ID, group.ID); err != nil || ok {
		t.Fatalf("expected membership gone after delete (ok=%v err=%v)", ok, err)
	}

	// Deleting an unknown membership fails (RowsAffected != 1).
	if err := DeleteGroupMembership(uuid.New()); err == nil {
		t.Fatalf("expected error deleting unknown membership, got nil")
	}
}

func TestGroupToWishlistMembership(t *testing.T) {
	setupTestDB(t)

	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	// GetGroupMembersFromWishlist matches groups the given user is a member of,
	// so the owner must also be enrolled as a member.
	addGroupMember(t, group.ID, owner.ID)
	wishlist := createTestWishlist(t, owner.ID)
	addWishlistMembership(t, wishlist.ID, group.ID)

	if ok, err := VerifyGroupMembershipToWishlist(wishlist.ID, group.ID); err != nil || !ok {
		t.Fatalf("expected group to be linked to wishlist (ok=%v err=%v)", ok, err)
	}
	if ok, err := VerifyGroupMembershipToWishlist(wishlist.ID, uuid.New()); err != nil || ok {
		t.Fatalf("expected unknown group not linked (ok=%v err=%v)", ok, err)
	}

	// Groups the owner belongs to that are attached to the wishlist.
	groups, err := GetGroupMembersFromWishlist(wishlist.ID, owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != group.ID {
		t.Fatalf("expected 1 group linked to wishlist, got %d", len(groups))
	}
}

func TestCreateGroupInDBFailure(t *testing.T) {
	setupTestDB(t)
	owner := createTestUser(t)
	if err := Instance.Migrator().DropTable(&models.Group{}); err != nil {
		t.Fatalf("failed to drop groups table: %v", err)
	}

	group := models.Group{Name: "Test", Enabled: true, OwnerID: owner.ID}
	group.ID = uuid.New()
	if _, err := CreateGroupInDB(group); err == nil {
		t.Error("expected an error when the groups table is unavailable")
	}
}

func TestCreateGroupMembershipInDBFailure(t *testing.T) {
	setupTestDB(t)
	owner := createTestUser(t)
	group := createTestGroup(t, owner.ID)
	if err := Instance.Migrator().DropTable(&models.GroupMembership{}); err != nil {
		t.Fatalf("failed to drop group_memberships table: %v", err)
	}

	membership := models.GroupMembership{GroupID: group.ID, MemberID: owner.ID, Enabled: true}
	membership.ID = uuid.New()
	if _, err := CreateGroupMembershipInDB(membership); err == nil {
		t.Error("expected an error when the group_memberships table is unavailable")
	}
}

func TestGroupQueriesFailOnClosedDB(t *testing.T) {
	runClosedDBCases(t, map[string]func() error{
		"VerifyGroupExistsByNameForUser": func() error {
			_, _, err := VerifyGroupExistsByNameForUser("x", uuid.New())
			return err
		},
		"GetGroupInformation": func() error {
			_, err := GetGroupInformation(uuid.New())
			return err
		},
		"UpdateGroupValuesByID": func() error {
			return UpdateGroupValuesByID(uuid.New(), "x", "x")
		},
		"VerifyUserOwnershipToGroup": func() error {
			_, err := VerifyUserOwnershipToGroup(uuid.New(), uuid.New())
			return err
		},
		"GetGroupMembersFromWishlist": func() error {
			_, err := GetGroupMembersFromWishlist(uuid.New(), uuid.New())
			return err
		},
		"GetGroupsAUserIsAMemberOf": func() error {
			_, err := GetGroupsAUserIsAMemberOf(uuid.New())
			return err
		},
		"GetGroupMembershipsFromGroup": func() error {
			_, err := GetGroupMembershipsFromGroup(uuid.New())
			return err
		},
		"VerifyIfGroupWithSameNameAndOwnerDoesNotExist": func() error {
			_, err := VerifyIfGroupWithSameNameAndOwnerDoesNotExist("x", uuid.New())
			return err
		},
		"VerifyUserMembershipToGroup": func() error {
			_, err := VerifyUserMembershipToGroup(uuid.New(), uuid.New())
			return err
		},
		"VerifyGroupMembershipToWishlist": func() error {
			_, err := VerifyGroupMembershipToWishlist(uuid.New(), uuid.New())
			return err
		},
		"GetGroupUsingGroupIDAndMembershipUsingUserID": func() error {
			_, err := GetGroupUsingGroupIDAndMembershipUsingUserID(uuid.New(), uuid.New())
			return err
		},
		"GetGroupUsingGroupIDAndUserIDAsOwner": func() error {
			_, err := GetGroupUsingGroupIDAndUserIDAsOwner(uuid.New(), uuid.New())
			return err
		},
		"GetGroupMembershipByGroupIDAndMemberID": func() error {
			_, err := GetGroupMembershipByGroupIDAndMemberID(uuid.New(), uuid.New())
			return err
		},
		"CreateGroupInDB": func() error {
			_, err := CreateGroupInDB(models.Group{})
			return err
		},
		"CreateGroupMembershipInDB": func() error {
			_, err := CreateGroupMembershipInDB(models.GroupMembership{})
			return err
		},
	})
}

func TestUpdateGroupValuesByIDFailures(t *testing.T) {
	t.Run("unknown group", func(t *testing.T) {
		setupTestDB(t)
		err := UpdateGroupValuesByID(uuid.New(), "name", "desc")
		if err == nil || err.Error() != "Name not changed in database." {
			t.Fatalf("error = %v, want \"Name not changed in database.\"", err)
		}
	})

	t.Run("description write fails", func(t *testing.T) {
		setupTestDB(t)
		group := createTestGroup(t, createTestUser(t).ID)
		injectFault(t, "update", "groups", 1)

		if err := UpdateGroupValuesByID(group.ID, "name", "desc"); !errors.Is(err, errInjected) {
			t.Fatalf("error = %v, want the injected fault", err)
		}
	})

	t.Run("description not written", func(t *testing.T) {
		setupTestDB(t)
		group := createTestGroup(t, createTestUser(t).ID)
		if err := registerCallback("update", true, func(db *gorm.DB) {
			if dest, ok := db.Statement.Dest.(map[string]interface{}); ok {
				if _, isDesc := dest["description"]; isDesc {
					db.RowsAffected = 0
				}
			}
		}); err != nil {
			t.Fatalf("failed to register callback: %v", err)
		}

		err := UpdateGroupValuesByID(group.ID, "name", "desc")
		if err == nil || err.Error() != "Description not changed in database." {
			t.Fatalf("error = %v, want \"Description not changed in database.\"", err)
		}
	})
}

// The list getters promise a non-nil empty slice (it serializes as [] rather
// than null), both when nothing matches and when the driver reports rows but
// scans none.
func TestGroupListGettersReturnEmptyNonNilSlices(t *testing.T) {
	setupTestDB(t)
	user := createTestUser(t)

	groups, err := GetGroupMembersFromWishlist(uuid.New(), user.ID)
	if err != nil || groups == nil || len(groups) != 0 {
		t.Errorf("GetGroupMembersFromWishlist = %v, %v; want empty non-nil slice", groups, err)
	}
	groups, err = GetGroupsAUserIsAMemberOf(user.ID)
	if err != nil || groups == nil || len(groups) != 0 {
		t.Errorf("GetGroupsAUserIsAMemberOf = %v, %v; want empty non-nil slice", groups, err)
	}
	memberships, err := GetGroupMembershipsFromGroup(uuid.New())
	if err != nil || memberships == nil || len(memberships) != 0 {
		t.Errorf("GetGroupMembershipsFromGroup = %v, %v; want empty non-nil slice", memberships, err)
	}

	forceRowsAffected(t, "query", "groups", 1)
	forceRowsAffected(t, "query", "group_memberships", 1)

	groups, err = GetGroupsAUserIsAMemberOf(user.ID)
	if err != nil || groups == nil || len(groups) != 0 {
		t.Errorf("GetGroupsAUserIsAMemberOf (rows reported, none scanned) = %v, %v; want empty non-nil slice", groups, err)
	}
	memberships, err = GetGroupMembershipsFromGroup(uuid.New())
	if err != nil || memberships == nil || len(memberships) != 0 {
		t.Errorf("GetGroupMembershipsFromGroup (rows reported, none scanned) = %v, %v; want empty non-nil slice", memberships, err)
	}
}

func TestVerifyIfGroupWithSameNameAndOwnerDoesNotExistNoMatch(t *testing.T) {
	setupTestDB(t)
	exists, err := VerifyIfGroupWithSameNameAndOwnerDoesNotExist("no such group", uuid.New())
	if err != nil || exists {
		t.Fatalf("got (%v, %v), want (false, nil) when no group matches", exists, err)
	}
}

func TestGroupCreatesRejectWrongRowCount(t *testing.T) {
	setupTestDB(t)
	forceRowsAffected(t, "create", "groups", 0)
	forceRowsAffected(t, "create", "group_memberships", 0)

	group := models.Group{Name: "g", Enabled: true, OwnerID: uuid.New()}
	group.ID = uuid.New()
	if _, err := CreateGroupInDB(group); err == nil || err.Error() != "Group not added to database." {
		t.Errorf("CreateGroupInDB error = %v, want \"Group not added to database.\"", err)
	}
	membership := models.GroupMembership{GroupID: uuid.New(), MemberID: uuid.New(), Enabled: true}
	membership.ID = uuid.New()
	if _, err := CreateGroupMembershipInDB(membership); err == nil || err.Error() != "Group membership not added to database." {
		t.Errorf("CreateGroupMembershipInDB error = %v, want \"Group membership not added to database.\"", err)
	}
}
