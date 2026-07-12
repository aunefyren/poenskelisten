package database

import (
	"testing"

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

	if _, err := GetGroupInformation(group.ID); err == nil {
		t.Fatalf("expected disabled group to be unreachable, got nil error")
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
	if _, err := GetGroupUsingGroupIDAndUserIDAsOwner(member.ID, group.ID); err == nil {
		t.Fatalf("expected error for non-owner, got nil")
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

	if _, err := GetGroupMembershipByGroupIDAndMemberID(group.ID, uuid.New()); err == nil {
		t.Fatalf("expected error for unknown member, got nil")
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
