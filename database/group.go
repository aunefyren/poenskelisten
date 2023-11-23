package database

import (
	"aunefyren/poenskelisten/models"
	"errors"

	"github.com/google/uuid"
)

func VerifyGroupExistsByNameForUser(groupName string, groupOwnerID uuid.UUID) (bool, models.Group, error) {

	var groupStruct models.Group

	groupRecords := Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.name = ?", groupName).Where("`groups`.Owner = ?", groupOwnerID).Find(&groupStruct)

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
	grouprecord := Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", GroupID).Find(&group)
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

	groupRecord := Instance.Model(group).Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", groupID).Update("name", groupName)
	if groupRecord.Error != nil {
		return groupRecord.Error
	}
	if groupRecord.RowsAffected != 1 {
		return errors.New("Name not changed in database.")
	}

	groupRecord = Instance.Model(group).Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", groupID).Update("description", groupDesc)
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

	grouprecord := Instance.Where("`groups`.enabled = ?", 1).
		Where("`groups`.id = ?", GroupID).
		Where("`groups`.owner_id = ?", UserID).
		Find(&group)

	if grouprecord.Error != nil {
		return false, grouprecord.Error
	} else if grouprecord.RowsAffected != 1 {
		return false, nil
	}

	return true, nil
}

// Get groups who are members of wishlist
func GetGroupMembersFromWishlist(WishlistID uuid.UUID) ([]models.Group, error) {

	var groups []models.Group

	groupsrecords := Instance.
		Where("`groups`.enabled = ?", 1).
		Joins("JOIN `group_memberships` on `groups`.id = `group_memberships`.group_id").
		Where("`group_memberships`.enabled = ?", 1).
		Joins("JOIN `users` on `group_memberships`.member_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Joins("JOIN `wishlist_memberships` on `groups`.id = `wishlist_memberships`.group_id").
		Where("`wishlist_memberships`.enabled = ?", 1).
		Where("`wishlist_memberships`.wishlist_id = ?", WishlistID).
		Group("groups.ID").
		Find(&groups)

	if groupsrecords.Error != nil {
		return []models.Group{}, groupsrecords.Error
	}

	if len(groups) == 0 {
		groups = []models.Group{}
	}

	return groups, nil
}

func GetGroupsAUserIsAMemberOf(UserID uuid.UUID) ([]models.Group, error) {
	var groups []models.Group

	// Retrieve groups that the user is a member of
	groupRecords := Instance.Where("`groups`.enabled = ?", 1).
		Joins("JOIN group_memberships on group_memberships.group_id = groups.id").
		Where("`group_memberships`.enabled = ?", 1).
		Where("`group_memberships`.member_id = ?", UserID).
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
		Where("`group_memberships`.enabled = ?", 1).
		Where("`group_memberships`.group_id = ?", GroupID).
		Joins("JOIN `users` on `group_memberships`.member_id = `users`.id").
		Where("`users`.enabled = ?", 1).Find(&groupMemberships)

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

	groupRecords := Instance.Where("`groups`.enabled = ?", 1).
		Where("`groups`.name = ?", GroupName).
		Where("`groups`.owner_id = ?", GroupOwnerID).
		Find(&group)

	if groupRecords.Error != nil {
		return true, groupRecords.Error
	} else if groupRecords.RowsAffected > 0 {
		return true, nil
	}
	return false, nil
}

// Verify if a user ID is a member of a group
func VerifyUserMembershipToGroup(UserID uuid.UUID, GroupID uuid.UUID) (bool, error) {
	var groupmembership models.GroupMembership
	groupmembershiprecord := Instance.Where("`group_memberships`.enabled = ?", 1).Where("`group_memberships`.group_id = ?", GroupID).Where("`group_memberships`.member_id = ?", UserID).Find(&groupmembership)
	if groupmembershiprecord.Error != nil {
		return false, groupmembershiprecord.Error
	} else if groupmembershiprecord.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Verify if a group id is a member of a wishlist
func VerifyGroupMembershipToWishlist(WishlistID uuid.UUID, GroupID uuid.UUID) (bool, error) {
	var wishlistmembership models.WishlistMembership

	wishlistmembershipprecord := Instance.Where("`wishlist_memberships`.enabled = ?", 1).Where("`wishlist_memberships`.wishlist_id = ?", WishlistID).Where("`wishlist_memberships`.group_id = ?", GroupID).Find(&wishlistmembership)
	if wishlistmembershipprecord.Error != nil {
		return false, wishlistmembershipprecord.Error
	} else if wishlistmembershipprecord.RowsAffected != 1 {
		return false, nil
	}

	return true, nil
}

// Get group ID while checking for valid membership
func GetGroupUsingGroupIDAndMembershipUsingUserID(UserID uuid.UUID, GroupID uuid.UUID) (models.Group, error) {
	var group = models.Group{}

	groupRecord := Instance.
		Where("`groups`.enabled = ?", 1).
		Joins("JOIN group_memberships on group_memberships.group_id = groups.id").
		Where("`group_memberships`.enabled = ?", 1).
		Where("`group_memberships`.member_id = ?", UserID).
		Where("`group_memberships`.group_id = ?", GroupID).
		Find(&group)

	if groupRecord.Error != nil {
		return group, groupRecord.Error
	} else if groupRecord.RowsAffected != 1 {
		return group, errors.New("Failed to find group.")
	}
	return group, nil
}

// Verify the given user owns the given group by ID
func GetGroupUsingGroupIDAndUserIDAsOwner(UserID uuid.UUID, GroupID uuid.UUID) (models.Group, error) {
	var group = models.Group{}

	groupRecord := Instance.Where("`groups`.enabled = ?", 1).Where("`groups`.id = ?", GroupID).Where("`groups`.owner_id = ?", UserID).Find(&group)
	if groupRecord.Error != nil {
		return group, groupRecord.Error
	} else if groupRecord.RowsAffected != 1 {
		return group, errors.New("Failed to find group in database.")
	}

	return group, nil
}
