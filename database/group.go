package database

import (
	"aunefyren/poenskelisten/models"
	"errors"
	"github.com/google/uuid"
)

func VerifyGroupExistsByNameForUser(groupName string, groupOwnerID uuid.UUID) (bool, models.Group, error) {
	var groupStruct models.Group

	groupRecords := Instance.Where(&models.Group{Enabled: true, Name: groupName, OwnerID: groupOwnerID}).Find(&groupStruct)

	if groupRecords.Error != nil {
		return false, models.Group{}, groupRecords.Error
	} else if groupRecords.RowsAffected != 1 {
		return false, groupStruct, nil
	}

	return true, groupStruct, nil
}

// Get group by Group ID
func GetGroupInformation(GroupID uuid.UUID) (models.Group, error) {
	var group models.Group

	grouprecord := Instance.Where(&models.Group{Enabled: true}).Where(&models.GormModel{ID: GroupID}).Find(&group)

	if grouprecord.Error != nil {
		return models.Group{}, grouprecord.Error
	} else if grouprecord.RowsAffected != 1 {
		return models.Group{}, errors.New("Failed to find correct group in DB.")
	}

	return group, nil
}

// Update values on group object in DB
func UpdateGroupValuesByID(groupID uuid.UUID, groupName string, groupDesc string) error {
	var group models.Group

	groupRecord := Instance.Model(group).Where(&models.Group{Enabled: true}).Where(&models.GormModel{ID: groupID}).Update("name", groupName)
	if groupRecord.Error != nil {
		return groupRecord.Error
	}
	if groupRecord.RowsAffected != 1 {
		return errors.New("Name not changed in database.")
	}

	groupRecord = Instance.Model(group).Where(&models.Group{Enabled: true}).Where(&models.GormModel{ID: groupID}).Update("description", groupDesc)
	if groupRecord.Error != nil {
		return groupRecord.Error
	}
	if groupRecord.RowsAffected != 1 {
		return errors.New("Description not changed in database.")
	}

	return nil

}

// Verify if a user ID is an owner of a group
func VerifyUserOwnershipToGroup(UserID uuid.UUID, GroupID uuid.UUID) (bool, error) {
	var group models.Group

	grouprecord := Instance.Where(&models.Group{Enabled: true, OwnerID: UserID}).Where(&models.GormModel{ID: GroupID}).Find(&group)

	if grouprecord.Error != nil {
		return false, grouprecord.Error
	} else if grouprecord.RowsAffected != 1 {
		return false, nil
	}

	return true, nil
}

// Get groups who are members of wishlist
func GetGroupMembersFromWishlist(wishlistID uuid.UUID, wishlistOwnerID uuid.UUID) ([]models.Group, error) {
	var groups []models.Group

	groupsRecords := Instance.
		Where(&models.Group{Enabled: true}).
		Joins("JOIN group_memberships ON groups.id = group_memberships.group_id").
		Where("group_memberships.enabled = ? AND group_memberships.member_id = ?", true, wishlistOwnerID).
		Joins("JOIN users ON group_memberships.member_id = users.id").
		Where("users.enabled = ?", true).
		Joins("JOIN wishlist_memberships ON groups.id = wishlist_memberships.group_id").
		Where("wishlist_memberships.enabled = ? AND wishlist_memberships.wishlist_id = ?", true, wishlistID).
		Group("groups.id").
		Find(&groups)

	if groupsRecords.Error != nil {
		return []models.Group{}, groupsRecords.Error
	}

	if len(groups) == 0 {
		groups = []models.Group{}
	}

	return groups, nil
}

func GetGroupsAUserIsAMemberOf(UserID uuid.UUID) ([]models.Group, error) {
	var groups []models.Group

	// Retrieve groups that the user is a member of
	groupRecords := Instance.
		Where(&models.Group{Enabled: true}).
		Joins("JOIN group_memberships ON group_memberships.group_id = groups.id").
		Where("group_memberships.enabled = ? AND group_memberships.member_id = ?", true, UserID).
		Find(&groups)

	if groupRecords.Error != nil {
		return []models.Group{}, groupRecords.Error
	} else if groupRecords.RowsAffected == 0 {
		return []models.Group{}, nil
	}

	if len(groups) == 0 {
		groups = []models.Group{}
	}

	return groups, nil
}

// Retrieve memberships from group using group ID. Check that users are enabled.
func GetGroupMembershipsFromGroup(GroupID uuid.UUID) ([]models.GroupMembership, error) {
	var groupMemberships []models.GroupMembership

	groupmembershipRecords := Instance.
		Where(&models.GroupMembership{Enabled: true, GroupID: GroupID}).
		Joins("JOIN users ON group_memberships.member_id = users.id").
		Where("users.enabled = ?", true).
		Find(&groupMemberships)

	if groupmembershipRecords.Error != nil {
		return []models.GroupMembership{}, groupmembershipRecords.Error
	} else if groupmembershipRecords.RowsAffected == 0 {
		return []models.GroupMembership{}, nil
	}

	if len(groupMemberships) == 0 {
		groupMemberships = []models.GroupMembership{}
	}

	return groupMemberships, nil

}

// Verify that a group with the same name and owner does not already exist
func VerifyIfGroupWithSameNameAndOwnerDoesNotExist(GroupName string, GroupOwnerID uuid.UUID) (bool, error) {
	var group = []models.Group{}

	groupRecords := Instance.Where(&models.Group{Enabled: true, Name: GroupName, OwnerID: GroupOwnerID}).Find(&group)

	if groupRecords.Error != nil {
		return true, groupRecords.Error
	} else if groupRecords.RowsAffected > 0 {
		return true, nil
	}
	return false, nil
}

// Verify if a user ID is a member of a group
func VerifyUserMembershipToGroup(UserID uuid.UUID, GroupID uuid.UUID) (bool, error) {
	var groupMembership models.GroupMembership

	groupMembershipRecord := Instance.Where(&models.GroupMembership{Enabled: true, GroupID: GroupID, MemberID: UserID}).Find(&groupMembership)

	if groupMembershipRecord.Error != nil {
		return false, groupMembershipRecord.Error
	} else if groupMembershipRecord.RowsAffected != 1 {
		return false, nil
	}

	return true, nil
}

// Verify if a group id is a member of a wishlist
func VerifyGroupMembershipToWishlist(WishlistID uuid.UUID, GroupID uuid.UUID) (bool, error) {
	var wishlistMembership models.WishlistMembership

	wishlistMembershipRecord := Instance.Where(&models.WishlistMembership{Enabled: true, WishlistID: WishlistID, GroupID: GroupID}).Find(&wishlistMembership)

	if wishlistMembershipRecord.Error != nil {
		return false, wishlistMembershipRecord.Error
	} else if wishlistMembershipRecord.RowsAffected != 1 {
		return false, nil
	}

	return true, nil
}

// Get group ID while checking for valid membership
func GetGroupUsingGroupIDAndMembershipUsingUserID(UserID uuid.UUID, GroupID uuid.UUID) (models.Group, error) {
	var group = models.Group{}

	groupRecord := Instance.
		Where(&models.Group{Enabled: true}).Where(&models.GormModel{ID: GroupID}).
		Joins("JOIN group_memberships ON group_memberships.group_id = groups.id").
		Where("group_memberships.enabled = ? AND group_memberships.member_id = ?", true, UserID).
		Find(&group)

	if groupRecord.Error != nil {
		return group, groupRecord.Error
	} else if groupRecord.RowsAffected != 1 {
		return group, errors.New("Failed to find group.")
	}
	return group, nil
}

// Verify the given user owns the given group by ID. Returns an error if not found.
func GetGroupUsingGroupIDAndUserIDAsOwner(UserID uuid.UUID, GroupID uuid.UUID) (models.Group, error) {
	var group = models.Group{}

	groupRecord := Instance.Where(&models.Group{Enabled: true, OwnerID: UserID}).Where(&models.GormModel{ID: GroupID}).Find(&group)
	if groupRecord.Error != nil {
		return group, groupRecord.Error
	} else if groupRecord.RowsAffected != 1 {
		return group, errors.New("Failed to find group in database.")
	}

	return group, nil
}

// Returns an error if not found
func GetGroupMembershipByGroupIDAndMemberID(GroupID uuid.UUID, MemberID uuid.UUID) (groupmembership models.GroupMembership, err error) {
	groupmembership = models.GroupMembership{}
	err = nil

	groupMembershipRecord := Instance.Where(&models.GroupMembership{Enabled: true, GroupID: GroupID, MemberID: MemberID}).Find(&groupmembership)

	if groupMembershipRecord.Error != nil {
		return groupmembership, groupMembershipRecord.Error
	} else if groupMembershipRecord.RowsAffected != 1 {
		return groupmembership, errors.New("Failed to find group membership.")
	}

	return groupmembership, err
}

func CreateGroupInDB(groupDB models.Group) (group models.Group, err error) {
	group = models.Group{}
	err = nil
	record := Instance.Create(&groupDB)

	if record.Error != nil {
		return group, record.Error
	}

	if record.RowsAffected != 1 {
		return group, errors.New("Group not added to database.")
	}

	return group, err
}

func CreateGroupMembershipInDB(groupMembershipDB models.GroupMembership) (groupMembership models.GroupMembership, err error) {
	groupMembership = models.GroupMembership{}
	err = nil
	record := Instance.Create(&groupMembershipDB)

	if record.Error != nil {
		return groupMembership, record.Error
	}

	if record.RowsAffected != 1 {
		return groupMembership, errors.New("Group membership not added to database.")
	}

	return groupMembership, err
}
